// Package statusblock computes the Now / Next / Later block the bare `abcd`
// board and the site's Status page render (itd-2609212103568351,
// spc-2609212138241908). The block is rendered status, never stored
// (adr-2609292012006845 decision 2): it is read from the lifecycle shelves, the
// readiness gate and the build's state file each time, so nothing on it can be
// edited into a lie.
//
//   - Now is every intent the build's state file shows in a lane, with its lane
//     state, then the head marked "next up": the first READY intent in pick
//     order that the build's record-only pre-start checks
//     (intent.StartChecksIn) let start, that is in no lane, and that no peer
//     holds when the caller hands in the build's peers check (ruling CC1 of
//     2026-09-29: the bare board pays that read, so its "next up" is the
//     pick). Now is empty only when no READY intent passes them.
//   - Next is every planned intent the readiness gate reports READY and the
//     state file shows in no lane, in pick order: `abcd build next`'s one
//     order (intent.PickLess), each intent scored by the read the pick scores
//     through (intent.ReadinessIn), the readiest first and the oldest among
//     equals.
//   - Later is every planned intent the gate reports not READY, each with the
//     gating checks it fails, then every draft, again leaving out an intent
//     in a lane.
//
// An intent the state file shows in a lane is listed under Now alone, never
// also under Next or Later (ruling BV2 of 2026-09-29). Next and Later are
// otherwise read from the record alone. The state file adds the lane rows to
// Now, and the head passes over an intent it shows in a lane, as the pick does
// over an intent with a run in progress; removing it empties Now's lane rows,
// returns each such intent to the list the gate places it in, and leaves a
// head (criterion 3).
//
// A run started for an issue (`abcd build <iss-N>`, every drain lane) is looked
// up in the issue ledger rather than the intent corpus: its rows carry the
// issue's one-line summary marked as an issue (KindIssue). A lane leaves Now
// once its pull request has merged or its branch is gone, though the state file
// still shows it at its land stage until the loop's next step sees the merge
// (iss-2610090824041378); the head still passes over its intent, as the pick
// does over a run in progress.
//
// The package reads the state file through a LaneReader, and the peers through
// a PeerReader, its caller supplies rather than importing the implement loop:
// the loop's own imports reach the site renderer, which renders this block, so
// each reader is handed in by the front door (the loop's StatusLanes and
// StatusPeers) and both surfaces call the one Read. The bare board hands in
// both; the site's Status page hands in no PeerReader, because another
// checkout's holdings are this machine's local state and never a published
// page's.
//
// Core never writes to stdout; the front doors format the Block.
package statusblock

import (
	"errors"
	"fmt"
	"os/exec"
	"path"
	"path/filepath"
	"slices"
	"sort"
	"strings"

	"github.com/intentdriven/abcd/internal/core/intent"
	"github.com/intentdriven/abcd/internal/core/issuerecord"
	"github.com/intentdriven/abcd/internal/core/recordid"
	"github.com/intentdriven/abcd/internal/core/spec"
	"github.com/intentdriven/abcd/internal/gitutil"
)

// OrderPick names the order Next and the head are read in: the pick order of
// `abcd build next` (itd-2609211116005482), the readiest first by the pick's
// score and the oldest among equals.
const OrderPick = "pick"

// KindIssue marks a Now row whose run was started for an issue rather than an
// intent (`abcd build <iss-N>`, every drain lane).
const KindIssue = "issue"

// Block is the three lists. Each is non-nil, so --json carries [] rather than
// null for an empty one.
type Block struct {
	Now   []Row `json:"now"`
	Next  []Row `json:"next"`
	Later []Row `json:"later"`
	// Order names the order Next and the head were read in.
	Order string `json:"order"`
}

// Row is one intent on the block: its id and title, the shelf it sits on, the
// release it targets, and what places it where it is. A field another
// placement needs (a score) joins here, omitted when empty.
type Row struct {
	ID     string `json:"id"`
	Title  string `json:"title"`
	Bucket string `json:"bucket"`
	// Kind is KindIssue on the row of a run started for an issue, whose Bucket
	// is then the ledger folder the issue sits in; empty on an intent's row.
	Kind string `json:"kind,omitempty"`
	// Target is the release a planned intent names as the one it must land by
	// (`target_release`: `next` or vX.Y.Z, itd-2609212103572513 criterion 4),
	// empty when it names none. A draft shows none: a target is a promise about
	// planned work, and the cut reads it off planned intents alone.
	Target string `json:"target_release,omitempty"`
	// NextUp marks the pick order's head on Now.
	NextUp bool `json:"next_up,omitempty"`
	// SpecID is the spec the readiness gate judged for a planned intent, its
	// own spec_id or the spec that names it; empty for a draft and for a
	// planned intent with no spec (spc-2610031844142274, A5).
	SpecID string `json:"spec_id,omitempty"`
	// Lane is the lane state of a Now row the state file shows.
	Lane *Lane `json:"lane,omitempty"`
	// Failing names the gating readiness checks a Later planned row fails, in
	// the gate's own order.
	Failing []string `json:"failing_checks,omitempty"`
}

// Lane is one lane's state as the build's state file records it.
type Lane struct {
	// Run is the run the lane belongs to.
	Run string `json:"run"`
	// Lane is the lane's id inside the run (lane-1, …); empty while the run
	// waits to open its next lane.
	Lane string `json:"lane,omitempty"`
	// Stage is the lane's next stage (worktree, brief, implement, validate,
	// land), or "pending" while the run waits to open its next lane.
	Stage string `json:"stage"`
	// Awaiting is the agent roles the lane waits on, when it waits on any:
	// one while its implementer works, one per validator out (ruling DR6).
	Awaiting string `json:"awaiting,omitempty"`
	// Waiting is what the lane waits for when it is no agent: its full check
	// before its push, with the time the wait began (ruling DR6d-2).
	Waiting string `json:"waiting,omitempty"`
	// Branch is the lane's branch as the state file records it
	// (build/<run>-<lane>), empty before the lane's worktree stage cuts it.
	Branch string `json:"branch,omitempty"`
	// InFlight is set when the lane's branch exists and its intent's spec is
	// open: work is on a branch and the design it builds is still live
	// (spc-2610031844142274, A5 and open question 3).
	InFlight bool `json:"in_flight,omitempty"`
}

// Started is one intent the state file shows in lanes: every lane of its run
// alive, each reported as its own Now row (ruling DR6, a run works several
// lanes at once).
type Started struct {
	Intent string
	Lanes  []Lane
}

// LaneReader reads the build's state file for the checkout at repoRoot. An
// absent state file reads as no lanes, never as an error.
type LaneReader func(repoRoot string) ([]Started, error)

// HeldBy reports whether a peer holds the intent a readiness result judges,
// and why: the build's peers check, read once for the whole block. A non-empty
// reason is a holding.
type HeldBy func(r intent.ReadyResult) string

// PeerReader reads the peers of the checkout at repoRoot once and returns the
// check the head is judged by. It is the build's own peers check (the loop's
// StatusPeers), so the head passes over exactly the intents the pick does.
type PeerReader func(repoRoot string) (HeldBy, error)

// Read computes the block for the checkout at repoRoot. lanes may be nil, which
// reads as an absent state file; peers may be nil, which leaves the head
// unjudged against other checkouts. It writes nothing.
func Read(repoRoot string, lanes LaneReader, peers PeerReader) (Block, error) {
	b := Block{Now: []Row{}, Next: []Row{}, Later: []Row{}, Order: OrderPick}

	corpus, err := intent.Load(repoRoot)
	if err != nil {
		return Block{}, err
	}
	// Every planned intent is judged against the one corpus and the one spec
	// store loaded here, rather than reloading both once per intent.
	store, err := spec.Load(repoRoot)
	if err != nil {
		return Block{}, err
	}
	// Each row's title is the status listing's, read for the intents the block
	// names rather than through intent.Status, which also judges every shipped
	// intent's review.
	row := func(it intent.Intent) (Row, error) {
		l, err := intent.Listing(repoRoot, it)
		if err != nil {
			return Row{}, err
		}
		r := Row{ID: it.ID, Title: l.Title, Bucket: it.Bucket}
		if it.Bucket == intent.BucketPlanned {
			r.Target = it.TargetRelease
		}
		return r, nil
	}

	// The state file is read first: an intent it shows in a lane is listed
	// under Now alone, so Next and Later leave it out and the head passes over
	// it, as the pick passes over an intent with a run in progress.
	var started []Started
	if lanes != nil {
		if started, err = lanes(repoRoot); err != nil {
			return Block{}, err
		}
	}
	inLane := map[string]bool{}
	for _, s := range started {
		inLane[s.Intent] = true
	}

	var planned, drafts []intent.Intent
	for _, it := range corpus.Intents {
		if inLane[it.ID] {
			continue
		}
		switch it.Bucket {
		case intent.BucketPlanned:
			planned = append(planned, it)
		case intent.BucketDrafts:
			drafts = append(drafts, it)
		}
	}
	sortByID(planned)
	sortByID(drafts)

	// ready pairs a READY intent with its row, the pick's view of it, and
	// whether the build's record-only pre-start checks let it start.
	type ready struct {
		it        intent.Intent
		row       Row
		res       intent.ReadyResult
		cand      intent.PickCandidate
		startable bool
	}
	var readies []ready
	var notReady []Row
	for _, it := range planned {
		res, err := intent.ReadyIn(repoRoot, store, it)
		if err != nil {
			return Block{}, fmt.Errorf("reading the readiness of %s: %w", it.ID, err)
		}
		r, err := row(it)
		if err != nil {
			return Block{}, err
		}
		r.SpecID = judgedSpec(store, res)
		if res.Ready {
			score, err := intent.ReadinessIn(repoRoot, store, it, res.SpecID)
			if err != nil {
				return Block{}, fmt.Errorf("scoring %s for the pick order: %w", it.ID, err)
			}
			chk, err := intent.StartChecksIn(repoRoot, corpus, store, res)
			if err != nil {
				return Block{}, fmt.Errorf("reading the pre-start checks of %s: %w", it.ID, err)
			}
			readies = append(readies, ready{it: it, row: r, res: res, cand: intent.PickCandidate{ID: it.ID, Score: score}, startable: chk.OK()})
			continue
		}
		for _, c := range res.Checks {
			if !c.OK && !c.Advisory {
				r.Failing = append(r.Failing, c.Name)
			}
		}
		notReady = append(notReady, r)
	}
	sort.SliceStable(readies, func(i, j int) bool { return intent.PickLess(readies[i].cand, readies[j].cand) })

	var head *Row
	var heldBy HeldBy
	peersRead := false
	for _, rd := range readies {
		b.Next = append(b.Next, rd.row)
		// The head is the first READY intent in pick order the build would
		// start: not one its record-only pre-start checks refuse (an open
		// question, an unanswered claim section, a hold, an unshipped blocker,
		// no step left to build), nor one the build's peers check finds another
		// checkout holding when the caller handed that check in; one already
		// in a lane is not in readies at all.
		if head == nil && rd.startable {
			// The peers are read once, and only when a head is in reach: a
			// board with nothing to start pays no read of other checkouts.
			if peers != nil && !peersRead {
				peersRead = true
				if heldBy, err = peers(repoRoot); err != nil {
					return Block{}, fmt.Errorf("reading the peers for the next-up head: %w", err)
				}
			}
			if heldBy != nil && heldBy(rd.res) != "" {
				continue
			}
			h := rd.row
			h.NextUp = true
			head = &h
		}
	}
	b.Later = append(b.Later, notReady...)
	for _, it := range drafts {
		r, err := row(it)
		if err != nil {
			return Block{}, err
		}
		b.Later = append(b.Later, r)
	}

	for _, s := range started {
		r := Row{ID: s.Intent}
		if it, ok := corpus.Lookup(s.Intent); ok {
			if r, err = row(it); err != nil {
				return Block{}, err
			}
			if it.Bucket == intent.BucketPlanned {
				res, err := intent.ReadyIn(repoRoot, store, it)
				if err != nil {
					return Block{}, fmt.Errorf("reading the readiness of %s: %w", it.ID, err)
				}
				r.SpecID = judgedSpec(store, res)
			}
		} else if issueID(s.Intent) {
			r = issueRow(repoRoot, s.Intent)
		}
		for _, lane := range s.Lanes {
			// A lane whose pull request merged, or whose branch is gone, is no
			// longer building, though the state file still shows it at its land
			// stage until the loop's next step sees the merge.
			if landed(repoRoot, lane) {
				continue
			}
			lr := r
			l := lane
			l.InFlight = inFlight(repoRoot, store, r.SpecID, l.Branch)
			lr.Lane = &l
			b.Now = append(b.Now, lr)
		}
	}
	if head != nil {
		b.Now = append(b.Now, *head)
		// The head is listed once, under Now: Next is the READY intents
		// after it (the product thinker's ruling of 2026-10-03,
		// iss-2610031207397996).
		b.Next = slices.DeleteFunc(b.Next, func(r Row) bool { return r.ID == head.ID })
	}
	return b, nil
}

// judgedSpec is the spec the readiness gate judged, when the store holds it:
// an intent that names no spec carries its spec_id's raw "null", and one that
// names a spec the store lacks names nothing a reader can open, so neither
// gives the row a spec.
func judgedSpec(store spec.Store, res intent.ReadyResult) string {
	if _, ok := store.Lookup(res.SpecID); !ok {
		return ""
	}
	return res.SpecID
}

// inFlight reports whether a lane's work is on a branch building a live
// design: its recorded branch resolves in the checkout and its intent's spec is
// open (spc-2610031844142274 open question 3). A lane at its worktree stage,
// before its branch is cut, records none and is not in flight.
func inFlight(repoRoot string, store spec.Store, specID, branch string) bool {
	if branch == "" || specID == "" {
		return false
	}
	sp, ok := store.Lookup(specID)
	if !ok || sp.Status != spec.StatusOpen {
		return false
	}
	_, err := gitutil.ResolveCommit(repoRoot, "refs/heads/"+branch)
	return err == nil
}

// stageLand is the lane stage that pushes the lane's branch and opens its
// pull request: the one stage at which a branch on the default branch means
// the pull request merged, rather than a branch freshly cut at the default tip.
const stageLand = "land"

// landed reports whether a lane has left the build though the state file still
// shows it: its recorded branch is gone, or the lane is at its land stage and
// its branch's tip is on the default branch as last fetched (the pull request
// merged). It reads only local refs, never the forge. A lane with no branch
// yet, and a lane git cannot answer for, have not landed.
func landed(repoRoot string, l Lane) bool {
	if l.Branch == "" || !gitutil.RefIsSafe(l.Branch) {
		return false
	}
	tip, err := gitutil.Run(repoRoot, "rev-parse", "--verify", "--quiet", "refs/heads/"+l.Branch+"^{commit}", "--")
	if err != nil {
		// rev-parse --verify --quiet exits 1, saying nothing, for a ref that
		// names no commit; any other failure is git unable to answer.
		var ee *exec.ExitError
		return errors.As(err, &ee) && ee.ExitCode() == 1
	}
	if l.Stage != stageLand || !gitutil.IsFullSHA(tip) {
		return false
	}
	def := gitutil.DefaultRef(repoRoot)
	if def == "" {
		return false
	}
	on, err := gitutil.IsAncestor(repoRoot, tip, def)
	return err == nil && on
}

// issueID reports whether id names an issue record (iss-N).
func issueID(id string) bool {
	return strings.HasPrefix(id, "iss-")
}

// maxIssueTitle caps an issue row's summary, the cap the build gives an
// issue lane's step title: an issue's first body line is often a paragraph.
const maxIssueTitle = 120

// issueRow is the row of a run started for an issue: the issue's one-line
// summary (its first non-blank body line) marked as an issue, and the ledger
// folder it sits in. An issue the ledger does not hold, or holds in a record
// its reader refuses, is shown by its id: the board names the lane either way,
// and the ledger's own gates report the record.
func issueRow(repoRoot, id string) Row {
	r := Row{ID: id, Kind: KindIssue, Title: KindIssue + ": " + id}
	rel, ok, err := recordid.LookupOne(repoRoot, id)
	if err != nil || !ok {
		return r
	}
	status := path.Base(path.Dir(rel))
	_, body, refusal, claims := issuerecord.Judge(filepath.Join(repoRoot, filepath.FromSlash(rel)), status)
	if refusal != nil || !claims {
		return r
	}
	r.Bucket = status
	for _, line := range strings.Split(body, "\n") {
		t := strings.Join(strings.Fields(line), " ")
		if t == "" {
			continue
		}
		if rs := []rune(t); len(rs) > maxIssueTitle {
			t = string(rs[:maxIssueTitle]) + "…"
		}
		r.Title = KindIssue + ": " + t
		break
	}
	return r
}

// sortByID orders intents oldest first by record id (intent.IDOlder, the
// pick's tie-break): the order Later is listed in.
func sortByID(its []intent.Intent) {
	sort.SliceStable(its, func(i, j int) bool { return intent.IDOlder(its[i].ID, its[j].ID) })
}
