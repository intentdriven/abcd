// Package statusblock computes the Now / Next / Later block the bare `abcd`
// board and the site's Status page render (itd-2609212103568351,
// spc-2609212138241908). The block is rendered status, never stored
// (adr-2609212115255771 decision 2): it is read from the lifecycle shelves, the
// readiness gate and the build's state file each time, so nothing on it can be
// edited into a lie.
//
//   - Now is every intent the build's state file shows in a lane, with its lane
//     state, then the head of the pick order marked "next up", so Now is never
//     empty while anything is READY.
//   - Next is every planned intent the readiness gate reports READY, in pick
//     order.
//   - Later is every planned intent the gate reports not READY, each with the
//     gating checks it fails, then every draft.
//
// Next, Later and the head are read from the record alone; the state file adds
// the lane rows to Now and changes nothing else, so removing it empties Now's
// lane rows and leaves the head (criterion 3).
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
	"strings"

	"github.com/intentdriven/abcd/internal/core/intent"
	"github.com/intentdriven/abcd/internal/core/spec"
)

// OrderRecordID is the order Next and the head are read in until `abcd build
// next`'s pick order exists (itd-2609211116005482): oldest first by record id,
// the tie-break that pick uses. It is carried on the Block so a reader, and a
// later build, can tell the interim order from the pick's.
const OrderRecordID = "record-id"

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
	b := Block{Now: []Row{}, Next: []Row{}, Later: []Row{}, Order: OrderRecordID}

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

	var head *Row
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
			b.Next = append(b.Next, r)
			// The head is the first READY intent the build would not refuse
			// for its hold: a held intent is not one the pick may take.
			if head == nil && it.Held == "" && !it.HeldMalformed {
				h := r
				h.NextUp = true
				head = &h
			}
			continue
		}
		for _, c := range res.Checks {
			if !c.OK && !c.Advisory {
				r.Failing = append(r.Failing, c.Name)
			}
		}
		notReady = append(notReady, r)
	}
	b.Later = append(b.Later, notReady...)
	for _, it := range drafts {
		r, err := row(it)
		if err != nil {
			return Block{}, err
		}
		b.Later = append(b.Later, r)
	}

	if lanes != nil {
		started, err := lanes(repoRoot)
		if err != nil {
			return Block{}, err
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
	}
	if head != nil {
		b.Now = append(b.Now, *head)
	}
	return b, nil
}

// sortByID orders intents oldest first by record id: an ordinal id predates
// every timestamp id (adr-45), and ids of one kind order by their number.
func sortByID(its []intent.Intent) {
	sort.SliceStable(its, func(i, j int) bool { return idLess(its[i].ID, its[j].ID) })
}

// idLess orders two intent ids by their number: the shorter number is the
// smaller, and numbers of one length compare as text.
func idLess(a, b string) bool {
	na, nb := strings.TrimPrefix(a, "itd-"), strings.TrimPrefix(b, "itd-")
	na, nb = strings.TrimLeft(na, "0"), strings.TrimLeft(nb, "0")
	if len(na) != len(nb) {
		return len(na) < len(nb)
	}
	return na < nb
}
