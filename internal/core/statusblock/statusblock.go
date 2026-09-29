// Package statusblock computes the Now / Next / Later block the bare `abcd`
// board and the site's Status page render (itd-2609212103568351,
// spc-2609212138241908). The block is rendered status, never stored
// (adr-2609212115255771 decision 2): it is read from the lifecycle shelves, the
// readiness gate and the build's state file each time, so nothing on it can be
// edited into a lie.
//
//   - Now is every intent the build's state file shows in a lane, with its lane
//     state, then the head marked "next up": the first READY intent in pick
//     order that the build's record-only pre-start checks
//     (intent.StartChecksIn) let start and that is in no lane. Now is empty
//     only when no READY intent passes them.
//   - Next is every planned intent the readiness gate reports READY, in pick
//     order: `abcd build next`'s one order (intent.PickLess), each intent
//     scored by the read the pick scores through (intent.ReadinessIn), the
//     readiest first and the oldest among equals.
//   - Later is every planned intent the gate reports not READY, each with the
//     gating checks it fails, then every draft.
//
// Next and Later are read from the record alone. The state file adds the lane
// rows to Now, and the head passes over an intent it shows in a lane, as the
// pick does over an intent with a run in progress; removing it empties Now's
// lane rows and leaves a head (criterion 3).
//
// The package reads the state file through a LaneReader its caller supplies
// rather than importing the implement loop: the loop's own imports reach the
// site renderer, which renders this block, so the reader is handed in by the
// front door (the loop's StatusLanes) and both surfaces call the one Read.
//
// Core never writes to stdout; the front doors format the Block.
package statusblock

import (
	"fmt"
	"sort"

	"github.com/intentdriven/abcd/internal/core/intent"
	"github.com/intentdriven/abcd/internal/core/spec"
)

// OrderPick names the order Next and the head are read in: the pick order of
// `abcd build next` (itd-2609211116005482), the readiest first by the pick's
// score and the oldest among equals.
const OrderPick = "pick"

// Block is the three lists. Each is non-nil, so --json carries [] rather than
// null for an empty one.
type Block struct {
	Now   []Row `json:"now"`
	Next  []Row `json:"next"`
	Later []Row `json:"later"`
	// Order names the order Next and the head were read in.
	Order string `json:"order"`
}

// Row is one intent on the block: its id and title, the shelf it sits on, and
// what places it where it is. A field another placement needs (a target
// release, a score) joins here, omitted when empty.
type Row struct {
	ID     string `json:"id"`
	Title  string `json:"title"`
	Bucket string `json:"bucket"`
	// NextUp marks the pick order's head on Now.
	NextUp bool `json:"next_up,omitempty"`
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
	// Step is the lane's next step (worktree, brief, implement, validate,
	// land), or "pending" while the run waits to open its next lane.
	Step string `json:"step"`
	// Awaiting is the agent role the lane waits on, when it waits on one.
	Awaiting string `json:"awaiting,omitempty"`
}

// Started is one intent the state file shows in a lane.
type Started struct {
	Intent string
	Lane   Lane
}

// LaneReader reads the build's state file for the checkout at repoRoot. An
// absent state file reads as no lanes, never as an error.
type LaneReader func(repoRoot string) ([]Started, error)

// Read computes the block for the checkout at repoRoot. lanes may be nil, which
// reads as an absent state file. It writes nothing.
func Read(repoRoot string, lanes LaneReader) (Block, error) {
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
		return Row{ID: it.ID, Title: l.Title, Bucket: it.Bucket}, nil
	}

	var planned, drafts []intent.Intent
	for _, it := range corpus.Intents {
		switch it.Bucket {
		case intent.BucketPlanned:
			planned = append(planned, it)
		case intent.BucketDrafts:
			drafts = append(drafts, it)
		}
	}
	sortByID(planned)
	sortByID(drafts)

	// The state file is read first: the head passes over an intent it shows in
	// a lane, as the pick passes over an intent with a run in progress.
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

	// ready pairs a READY intent with its row, the pick's view of it, and
	// whether the build's record-only pre-start checks let it start.
	type ready struct {
		it        intent.Intent
		row       Row
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
		if res.Ready {
			score, err := intent.ReadinessIn(repoRoot, store, it, res.SpecID)
			if err != nil {
				return Block{}, fmt.Errorf("scoring %s for the pick order: %w", it.ID, err)
			}
			chk, err := intent.StartChecksIn(repoRoot, corpus, store, res)
			if err != nil {
				return Block{}, fmt.Errorf("reading the pre-start checks of %s: %w", it.ID, err)
			}
			readies = append(readies, ready{it: it, row: r, cand: intent.PickCandidate{ID: it.ID, Score: score}, startable: chk.OK()})
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
	for _, rd := range readies {
		b.Next = append(b.Next, rd.row)
		// The head is the first READY intent in pick order the build would
		// start: not one its record-only pre-start checks refuse (an open
		// question, an unanswered claim section, a hold, an unshipped blocker,
		// no step left to build), and not one already in a lane. The build's
		// peers check is not run: the block does not consult other checkouts.
		if head == nil && rd.startable && !inLane[rd.it.ID] {
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
		}
		lane := s.Lane
		r.Lane = &lane
		b.Now = append(b.Now, r)
	}
	if head != nil {
		b.Now = append(b.Now, *head)
	}
	return b, nil
}

// sortByID orders intents oldest first by record id (intent.IDOlder, the
// pick's tie-break): the order Later is listed in.
func sortByID(its []intent.Intent) {
	sort.SliceStable(its, func(i, j int) bool { return intent.IDOlder(its[i].ID, its[j].ID) })
}
