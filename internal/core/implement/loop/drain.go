package loop

// drain.go is the drain run (itd-82; spc-2609212015054359 scope 4, 5 and 7): a
// loop over the ordered eligible set that hands each issue to the implement
// loop's issue key through Start, one lane at a time, reads each lane's
// outcome, routes every hand-back by its kind, and is bounded by the pace
// rule's window and by --max.
//
// The loop is driven by the host session (decision 5 on the parent), so a
// drain is too: each `abcd drain` performs one move and exits. It routes what
// the lane it opened last has come to, then opens the next issue's lane, or
// says why it opens none: the lane is still in progress (drive it with `abcd
// implement step`), the window has elapsed (next_eligible_at is written), the
// cap is reached, or nothing eligible is left. The drain's own state is one
// file beside the runs, `.abcd/.work.local/run/drain.json`; each lane is an
// ordinary run the loop's own verbs drive.
//
// The classification is re-derived every invocation from the ledger
// (decision 8), so a field hand-back is flagged afresh each time and written
// nowhere; what the drain records is what it did: the lanes it opened and the
// lanes' hand-backs it routed, with the record change each made.

import (
	"cmp"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/intentdriven/abcd/internal/core/capture"
	"github.com/intentdriven/abcd/internal/core/claim"
	"github.com/intentdriven/abcd/internal/core/drainrule"
	"github.com/intentdriven/abcd/internal/core/implement"
	"github.com/intentdriven/abcd/internal/core/jsonstrict"
	"github.com/intentdriven/abcd/internal/core/runner"
	"github.com/intentdriven/abcd/internal/fsutil"
	"github.com/intentdriven/abcd/internal/gitutil"
	"github.com/intentdriven/abcd/internal/localtier"
	"github.com/intentdriven/abcd/internal/termsafe"
)

// DrainStateRel is the drain's state file, beside the runs. Its name is not a
// run id, so the run listing never reads it as one.
const DrainStateRel = RunRelDir + "/drain.json"

// drainLockRel is the lock one drain invocation holds from its first read to
// its last write, so two drains never open two lanes. It is not the runs'
// lock, which Start takes inside it.
const drainLockRel = RunRelDir + "/drain.lock"

// DrainSchemaVersion is the drain state's shape.
//
// Version 2 added a route's pins and override to a routed judge's stored
// route: a judgement's route carries the pin and sealed configuration's
// SHA-256 of a pinned opencode run (`provider_pin`, `pin_config_sha256`), a
// route to a model on this machine its local pin, what its server reported
// and that it is local (`local_pin`, `local_served`, `local`), and the
// route record's `override`, as the run state's version 16 does. The field
// comes with the route record the two share; no drain flag sets it. Version
// 1 is its strict subset, read as a drain whose judge was pointed at no pin
// and carries no override, and written back at version 2; a file of version
// 1 carrying any of them is not one its version wrote, and is refused naming
// the key. v0.15.1 reads no version first, so it refuses a version-2 file as
// one that does not parse; releases after it that write version 1 refuse it
// as a newer abcd's before any strict decode.
const DrainSchemaVersion = 2

// drainSchemaVersionUnpinned is the drain state's version before a judge
// route's pins: read, never written.
const drainSchemaVersionUnpinned = 1

// maxDrainStateBytes caps the drain state read.
const maxDrainStateBytes = 4 << 20

// StageDrain is the refusal stage of the drain run.
const StageDrain = "drain"

// Why a drain ended.
const (
	// DrainStoppedCap: the drain opened --max lanes.
	DrainStoppedCap = "cap"
	// DrainStoppedEmpty: no eligible issue is left that this drain has not
	// taken, held by another or not; with held work left the drain waits
	// (drain_wait.go).
	DrainStoppedEmpty = "empty"
	// DrainStoppedOutage (outage.go): the run gave up on a lost connection.
	// DrainStoppedOrigin (drain_origin.go): the drain gave up waiting on
	// origin, which no move could read through the outage schedule.
)

// The outcomes of a lane the drain opened, as the drain reads its run.
const (
	DrainLaneInProgress = "in-progress"
	// DrainLanePullRequest: the landing opened the lane's pull request and
	// armed it or left it open; the merge is a person's gate.
	DrainLanePullRequest = "pull-request"
	DrainLaneHandedBack  = "handed-back"
	// DrainLaneDone: the run is complete.
	DrainLaneDone = "done"
)

// The routes a hand-back takes, by kind (scope 5).
const (
	// RoutePromoted: a user moment, promoted to an intent draft with
	// `capture promote`; the issue gains the draft in related_intents and
	// nothing else is written.
	RoutePromoted = "promoted"
	// RouteDecisionRecord: a rule about trust or safety, flagged as needing a
	// decision record with the question stated; nothing is minted.
	RouteDecisionRecord = "decision-record"
	// RouteRule: above the rule's severity, outside its categories, or held
	// by another of its rules; flagged naming the rule.
	RouteRule = "rule"
	// RouteHome: anything else, flagged with the home the decision belongs in.
	RouteHome = "home"
)

// DrainState is one drain, between invocations.
type DrainState struct {
	SchemaVersion int       `json:"schema_version"`
	StartedAt     time.Time `json:"started_at"`
	UpdatedAt     time.Time `json:"updated_at"`
	// Rule is the decision record the rule was read from when the drain began.
	Rule string `json:"rule"`
	// Max is the cap on lanes the drain opens; 0 is all (the default).
	Max int `json:"max"`
	// Pace is the pace the drain started on; its window and pause bound the
	// drain as they bound a run.
	Pace            *Pace        `json:"pace"`
	WindowStartedAt *time.Time   `json:"window_started_at,omitempty"`
	NextEligibleAt  *time.Time   `json:"next_eligible_at,omitempty"`
	Lanes           []DrainLane  `json:"lanes"`
	HandBacks       []DrainRoute `json:"hand_backs"`
	// Judging is the host judgement the drain awaits before it opens the next
	// issue's lane (scope 3), and Judgements every answer it has taken.
	Judging    *DrainJudging    `json:"judging,omitempty"`
	Judgements []DrainJudgement `json:"judgements,omitempty"`
	// JudgeFallbacks are the receipts of every routed judgement a runner did
	// not answer, each left to the host's request (drain_judge.go).
	JudgeFallbacks []runner.FallbackReceipt `json:"judge_fallbacks,omitempty"`
	// Lapsed are the judgement requests dropped unanswered once the hold the
	// drain took for them lapsed, the latest last; their issues are asked
	// again only once a cool-down has passed, and then only after the
	// drain's other eligible work (drain_hold.go).
	Lapsed []DrainLapse `json:"lapsed,omitempty"`
	// PendingHandBacks are the hand-backs a yes routed whose tombstone the
	// move could not publish (drain_pending.go); each is published over the
	// issue's next release on origin, and carried into the next drain.
	PendingHandBacks []DrainPendingHandBack `json:"pending_hand_backs,omitempty"`
	// Stopped says why the drain ended, and EndedAt when; empty while it runs.
	Stopped string     `json:"stopped,omitempty"`
	EndedAt *time.Time `json:"ended_at,omitempty"`
	// OriginWait is the wait on origin a move that could not read it began
	// (drain_origin.go); nil while origin answers.
	OriginWait *DrainOriginWait `json:"origin_wait,omitempty"`
	// WaitingUntil is when a drain waiting at its holder's cap, or on work
	// others hold alone, looks again, and Waiting which of the two it waits
	// on (DrainWaitCap, DrainWaitHeld); both empty otherwise. It is kept
	// apart from NextEligibleAt, so a wait never resets the pace window
	// (drain_wait.go).
	WaitingUntil *time.Time `json:"waiting_until,omitempty"`
	Waiting      string     `json:"waiting,omitempty"`
	// Skips are the issues and paths a person holds back for now, each with
	// its reason; the list clears when the drain ends (drain_skip.go).
	Skips []DrainSkip `json:"skips,omitempty"`
}

// DrainLane is one issue the drain handed to a lane.
type DrainLane struct {
	Issue    string    `json:"issue"`
	RunID    string    `json:"run_id"`
	OpenedAt time.Time `json:"opened_at"`
	Outcome  string    `json:"outcome"`
	PR       int       `json:"pr,omitempty"`
}

// DrainRoute is one hand-back and where it went.
type DrainRoute struct {
	Issue string `json:"issue"`
	// From is "lane" for a lane's hand-back, "judgement" for the host
	// judgement's before any lane opened, "reservation" for an issue the
	// drain could not reserve (its found_at names no path), "field" for the
	// rule's.
	From  string `json:"from"`
	Kind  string `json:"kind"`
	Route string `json:"route"`
	// Draft is the intent draft a promotion made; Question the question a
	// decision-record flag states; Rule the rule a field hand-back names; Home
	// the home a flag names.
	Draft    string `json:"draft,omitempty"`
	Question string `json:"question,omitempty"`
	Rule     string `json:"rule,omitempty"`
	Home     string `json:"home,omitempty"`
	Reason   string `json:"reason"`
	// Wrote is the record change the route made, in words; "nothing" for a
	// flag.
	Wrote string     `json:"wrote"`
	At    *time.Time `json:"at,omitempty"`
	// Blob is the blob id of the issue's record on origin's default branch
	// when the hand-back was routed, "" without origin: once that record
	// changes, the drain takes the issue up again (drain_readmit.go).
	Blob string `json:"blob,omitempty"`
}

// DrainOptions are the drain's own flags.
type DrainOptions struct {
	// Max is --max as typed; 0 when not given, which is all.
	Max int
	// Judgement is --judgement: the path of the host's answer to the
	// judgement the drain awaits; empty when not given.
	Judgement string
	// Transcripts is where a routed judge's transcript lands
	// (drain_judge.go); nil leaves the judgement on the host's request,
	// routed or not.
	Transcripts runner.TranscriptStore
	// JudgeTimeout bounds one routed judgement; drainJudgeTimeout when zero.
	JudgeTimeout time.Duration
	// Context ends the move's routed judge: the front door ends it on an
	// interrupt or a termination. context.Background when nil.
	Context context.Context
	// Skip are --skip's targets, issue ids or paths, held back for now with
	// SkipReason, --reason; Unskip are --unskip's, taken off the list
	// (drain_skip.go).
	Skip       []string
	SkipReason string
	Unskip     []string
}

// DrainResult is one drain invocation's summary.
type DrainResult struct {
	State string `json:"state"`
	// Started is true when this invocation began a new drain.
	Started  bool     `json:"started"`
	Rule     string   `json:"rule"`
	Loosened []string `json:"loosened"`
	Order    string   `json:"order"`
	Max      int      `json:"max"`
	Pace     *Pace    `json:"pace"`
	// Lanes are every lane the drain opened, with its outcome as last read.
	Lanes []DrainLane `json:"lanes"`
	// Lane is the lane in progress after this call, if any.
	Lane *DrainLane `json:"lane"`
	// Start is the run this call started, when it opened a lane.
	Start *StartResult `json:"start"`
	// Routed are the hand-backs this call routed; HandBacks every one the
	// drain has routed; Flags the rule's hand-backs over the ledger as this
	// call read it, each naming its rule, written nowhere, less the issues
	// this drain promoted itself (fieldFlags).
	Routed    []DrainRoute `json:"routed"`
	HandBacks []DrainRoute `json:"hand_backs"`
	Flags     []DrainRoute `json:"flags"`
	// Judging is the host judgement the drain awaits after this call, if any;
	// Judged the answer this call took; Judgements every answer the drain
	// has taken, each with whether it decided the issue's disposition.
	Judging    *DrainJudging    `json:"judging"`
	Judged     *DrainJudgement  `json:"judged,omitempty"`
	Judgements []DrainJudgement `json:"judgements"`
	// Passed are eligible issues this call did not take, each with why.
	Passed         []Excluded             `json:"passed"`
	Dispositions   []capture.DrainVerdict `json:"dispositions"`
	NextEligibleAt *time.Time             `json:"next_eligible_at"`
	// WaitingUntil is when a drain waiting at the cap, or on held work alone,
	// looks again, and Waiting which it waits on (drain_wait.go).
	WaitingUntil *time.Time `json:"waiting_until"`
	Waiting      string     `json:"waiting,omitempty"`
	Stopped      string     `json:"stopped,omitempty"`
	Complete     bool       `json:"complete"`
	// Finished are the merged lanes this call took the landing's last step
	// for, outside the window clock (drain_finish.go).
	Finished []DrainFinished `json:"finished"`
	// Parked are the lanes parked at land whose head is not on the default
	// branch, each as its pull request read (drain_finish.go).
	Parked []DrainParked `json:"parked"`
	// Counts are the drain's totals as this call leaves it (scope 8).
	Counts DrainCounts `json:"counts"`
	Next   string      `json:"next"`
	// Origin is the view of origin's default branch this move planned from,
	// and OriginWait the wait a move that could not read it is in
	// (drain_origin.go); both nil in a checkout with no remote named origin.
	Origin     *DrainOrigin     `json:"origin,omitempty"`
	OriginWait *DrainOriginWait `json:"origin_wait,omitempty"`
	// PendingHandBacks are the hand-backs routed and not yet published on
	// origin, as this call leaves them (drain_pending.go).
	PendingHandBacks []DrainPendingHandBack `json:"pending_hand_backs,omitempty"`
	// Skips is the drain's skip list as this call leaves it: empty once the
	// drain has ended, which clears it (drain_skip.go).
	Skips []DrainSkip `json:"skips"`
}

// DrainCounts are the drain's totals (itd-82 scope 8): its lanes by outcome,
// the pull requests they opened, its hand-backs by route, the rule's flags,
// the host judgements, the issues this call passed over, and the spend.
type DrainCounts struct {
	LanesOpened int `json:"lanes_opened"`
	InProgress  int `json:"in_progress"`
	// PullRequests are the lanes whose landing opened a pull request, done
	// or not; whether one merged is a person's gate, and not counted.
	PullRequests    int `json:"pull_requests"`
	LanesHandedBack int `json:"lanes_handed_back"`
	// Promotions, DecisionRecords and Homes are the hand-backs the drain
	// routed, from its lanes and from the host judgement, by route.
	Promotions      int `json:"promotions"`
	DecisionRecords int `json:"decision_records"`
	Homes           int `json:"homes"`
	// RuleFlags are the rule's hand-backs over the ledger as this call read it,
	// less the issues this drain promoted itself, which Promotions counts.
	RuleFlags  int        `json:"rule_flags"`
	Judgements int        `json:"judgements"`
	Passed     int        `json:"passed"`
	Spend      DrainSpend `json:"spend"`
}

// DrainSpend is what the drain's lanes have spent, in agent runs, the unit the
// budget check estimates a run in: each implementer's receipt the loop
// verified (the implement stage's, each fix round's, a hand-back's) and each
// validator's return it parsed a verdict from. No runner reports tokens or
// money, so neither is counted. Unread names a lane's run whose state could
// not be read, which makes the counts a floor.
type DrainSpend struct {
	AgentRuns    int      `json:"agent_runs"`
	Implementers int      `json:"implementers"`
	Validators   int      `json:"validators"`
	Unread       []string `json:"unread"`
}

// Drain performs one move of the drain run: it begins a drain when none is in
// progress, honours the window clock, routes the last lane's outcome, and
// opens the next eligible issue's lane, or says why it opens none. A drain
// without the repository's rule is refused before anything is written.
func Drain(repoRoot string, o Options, d DrainOptions) (DrainResult, error) {
	if d.Max < 0 {
		return DrainResult{}, refuse(StageDrain, "", "", fmt.Sprintf("--max %d is not a number of lanes", d.Max),
			"give --max a whole number of lanes, 1 or more, or leave it out to drain all")
	}
	flags, err := parsePaceFlags(o.Pace, o.SubAgents, o.FixRounds)
	if err != nil {
		return DrainResult{}, err
	}
	if err := checkSkips(d); err != nil {
		return DrainResult{}, err
	}
	// The rule first: a repository without its own record is refused before
	// anything else is read or written (criterion 11). The ledger is read
	// under the lock, from origin's default branch (drain_origin.go).
	rule, err := drainrule.Load(repoRoot)
	if err != nil {
		return DrainResult{}, err
	}
	if err := tierPresent(repoRoot); err != nil {
		return DrainResult{}, err
	}
	if err := fsutil.EnsureRealDirAll(repoRoot, RunRelDir, dirPerm); err != nil {
		return DrainResult{}, fmt.Errorf("creating %s: %w", RunRelDir, err)
	}
	var res DrainResult
	err = fsutil.WithFileLock(filepath.Join(repoRoot, filepath.FromSlash(drainLockRel)), lockTimeout, func() error {
		var err error
		res, err = drainMove(repoRoot, o, d, flags, rule)
		return err
	})
	if errors.Is(err, fsutil.ErrLockContention) {
		return DrainResult{}, contend(StageDrain, "", "", "another drain is moving in this checkout", "back off and retry")
	}
	return res, err
}

// drainView and startLane are claim.View and Start, seams a test replaces to
// make the move's view unreadable or its lane's start fail.
var (
	drainView = claim.View
	startLane = Start
)

// drainMove is Drain under the drain's lock.
func drainMove(repoRoot string, o Options, d DrainOptions, flags paceFlags, rule drainrule.Rule) (_ DrainResult, err error) {
	now := o.now()
	st, live, err := readDrain(repoRoot)
	if err != nil {
		return DrainResult{}, err
	}
	// The drain's judgement holds this move knows (drain_hold.go): each it
	// leaves unused is released on the way out, on every path the process
	// lives through, a panic included. persisted is the judgement awaited as
	// the state on disk records it: a move that fails, or panics, before its
	// write leaves the one read, which keeps its hold, and every other hold
	// the move took on which no lane opened is released. Releasing is
	// idempotent, so the settle here after finish's own pushes nothing more.
	// A process killed outright runs no defer: its hold lapses after one
	// lease.
	holds := &judgeHolds{repoRoot: repoRoot, now: now}
	if j := st.Judging; j != nil {
		holds.add(j.Issue, j.Hold, false)
	}
	persisted := st.Judging
	defer func() {
		r := recover()
		holds.settle(persisted)
		if r != nil {
			panic(r)
		}
	}()
	started := false
	// repaced is what this move's new pace came to, "" when it took none;
	// repace is the new pace the lanes in progress take once the move's
	// state is written, and stillPace the pace the drain ran on before it,
	// which a refused move puts back.
	repaced := ""
	var repace *drainRepace
	stillPace := st.Pace
	if !live {
		pace, err := resolvePace(o.roots(repoRoot), flags)
		if err != nil {
			return DrainResult{}, err
		}
		// A hand-back an ended drain could not publish is still a person's,
		// so the new drain keeps it pending (drain_pending.go).
		pending := st.PendingHandBacks
		st = DrainState{SchemaVersion: DrainSchemaVersion, StartedAt: now, UpdatedAt: now, Rule: rule.Record,
			Max: d.Max, Pace: &pace, WindowStartedAt: &now, Lanes: []DrainLane{}, HandBacks: []DrainRoute{},
			PendingHandBacks: pending}
		started = true
	} else {
		if d.Max != 0 && d.Max != st.Max {
			return DrainResult{}, refuse(StageDrain, "", "", fmt.Sprintf("the drain started %s is in progress with %s; --max %d names another, and a cap is set when a drain starts",
				st.StartedAt.Format(time.RFC3339), capPhrase(st.Max), d.Max),
				"drain again without --max; the drain keeps the cap it started with")
		}
		// A running drain takes a new pace rather than refusing it
		// (drain_pace.go).
		if flags.set {
			if repace, repaced, err = repaceDrain(repoRoot, &st, o, flags); err != nil {
				return DrainResult{}, err
			}
		}
	}
	// The skip list takes this move's --skip and --unskip before the plan
	// is read, so the move passes over what it holds back (drain_skip.go).
	skipped, err := applySkips(&st, d, now)
	if err != nil {
		return DrainResult{}, err
	}
	// The ledger as origin's default branch holds it, fetched now, before any
	// judgement or reservation: a move that cannot read it plans nothing and
	// waits (drain_origin.go).
	plan, origin, wait, recovered, probed, err := movePlan(repoRoot, &st, rule, now)
	if err != nil {
		return DrainResult{}, err
	}
	// blobs are the issues' record blobs on origin's default branch as this
	// move planned from it, read once when a hand-back is routed or a
	// handed-back issue is considered again (drain_readmit.go).
	blobs := &originBlobs{repoRoot: repoRoot, origin: origin}
	res := DrainResult{State: DrainStateRel, Started: started, Rule: plan.Record, Loosened: plan.Loosened,
		Order: plan.Order, Dispositions: plan.Dispositions, Routed: []DrainRoute{}, Passed: []Excluded{},
		Flags: fieldFlags(plan, st.HandBacks), Finished: []DrainFinished{}, Parked: []DrainParked{}, Origin: origin}
	// viewUnread is why the reservations this move fetched could not be read,
	// and lapse what became of a judgement request whose hold lapsed;
	// leaseNotes are the renewals of the lane in progress that failed or
	// found the hold another's (drain_lease.go).
	viewUnread, lapse := "", ""
	var leaseNotes []string
	// published is what became of the pending hand-backs this move ended.
	var published []string
	finish := func(write bool) (DrainResult, error) {
		// A drain that ends awaits nothing, and every judgement hold the move
		// leaves unused is released before the state is written.
		if st.Stopped != "" && st.Judging != nil {
			st.Judging, write = nil, true
		}
		if st.Stopped != "" && st.WaitingUntil != nil {
			st.WaitingUntil, st.Waiting, write = nil, "", true
		}
		if skipped != "" {
			res.Next = joinNext(skipped, res.Next)
			write = true
		}
		// A drain that ends clears its skip list: a hold for now never
		// outlives the drain it was given to.
		if st.Stopped != "" && len(st.Skips) > 0 {
			res.Next = joinNext(res.Next, fmt.Sprintf("the skip list's %d entr%s cleared with the drain's end", len(st.Skips), plural(len(st.Skips), "y is", "ies are")))
			st.Skips, write = nil, true
		}
		if repaced != "" {
			res.Next = joinNext(repaced, res.Next)
			write = true
		}
		if clauses := holds.settle(st.Judging); len(clauses) > 0 {
			res.Next = joinNext(append([]string{res.Next}, clauses...)...)
			write = true
		}
		if lapse != "" {
			res.Next = joinNext(res.Next, lapse)
			write = true
		}
		if len(published) > 0 {
			res.Next = joinNext(append([]string{res.Next}, published...)...)
			write = true
		}
		if viewUnread != "" {
			res.Next = joinNext(res.Next, viewUnread)
		}
		if len(leaseNotes) > 0 {
			res.Next = joinNext(append([]string{res.Next}, leaseNotes...)...)
		}
		if diags := drainRouterDiagnostics(repoRoot, o); len(diags) > 0 {
			res.Next = joinNext(append([]string{res.Next}, diags...)...)
		}
		if write {
			st.UpdatedAt = now
			if err := writeDrain(repoRoot, st); err != nil {
				return DrainResult{}, err
			}
			persisted = st.Judging
		}
		// The lanes in progress take a new pace only once the move has
		// written it, so a move refused before here takes none.
		if repace != nil {
			if err := repace.lanes(repoRoot, st, now); err != nil {
				return DrainResult{}, fmt.Errorf("the drain took its new pace, but a lane in progress did not: %w; give the pace again", err)
			}
		}
		res.Max, res.Pace, res.Lanes, res.HandBacks = st.Max, st.Pace, st.Lanes, st.HandBacks
		res.Judging, res.Judgements = st.Judging, st.Judgements
		if res.Judgements == nil {
			res.Judgements = []DrainJudgement{}
		}
		res.NextEligibleAt, res.Stopped, res.Complete = st.NextEligibleAt, st.Stopped, st.Stopped != ""
		res.WaitingUntil, res.Waiting = st.WaitingUntil, st.Waiting
		if note := parkedNote(res.Parked); note != "" {
			res.Next = joinNext(note, res.Next)
		}
		if note := finishedNote(res.Finished); note != "" {
			res.Next = joinNext(note, res.Next)
		}
		if recovered != "" {
			res.Next = joinNext(recovered, res.Next)
		}
		res.OriginWait = st.OriginWait
		res.PendingHandBacks = st.PendingHandBacks
		res.Skips = st.Skips
		if res.Skips == nil {
			res.Skips = []DrainSkip{}
		}
		res.Counts = drainCounts(repoRoot, st, res.Flags, res.Passed)
		// The normal ending is claimed only once every lane has ended: an
		// outage stop is taken before the lane in progress is read, so a
		// drain can stop with a lane still mid-flight.
		if res.Complete && res.Counts.PullRequests == 0 && res.Counts.LanesOpened > 0 && res.Counts.InProgress == 0 {
			res.Next = joinNext(res.Next, "no lane ended in a pull request, which is a normal ending: every lane's outcome and hand-back is above")
		}
		if i := slices.IndexFunc(st.Lanes, func(l DrainLane) bool { return l.Outcome == DrainLaneInProgress }); i >= 0 {
			l := st.Lanes[i]
			res.Lane = &l
		}
		return res, nil
	}
	// refuseMoved is a refusal once the move has done anything: what it
	// already did, a judgement it took, a lane it finished, a hand-back it
	// routed, is written to a live drain first, so the next move does not do
	// it again, and the refusal says what was finished and whether the answer
	// given was taken. The judgement awaited as written is the one the
	// move's clean-up keeps the hold of (persisted), so a hold the written
	// state no longer records is released. A drain this move began writes
	// nothing.
	refuseMoved := func(err error) (DrainResult, error) {
		// A refused move takes no new pace (drain_pace.go).
		tookPace := repaced != ""
		st.Pace = stillPace
		if live {
			st.UpdatedAt = now
			if werr := writeDrain(repoRoot, st); werr != nil {
				return DrainResult{}, werr
			}
			persisted = st.Judging
		}
		var notes []string
		if note := finishedNote(res.Finished); note != "" {
			notes = append(notes, "before refusing, this move "+note)
		}
		if res.Judged != nil {
			notes = append(notes, "the judgement answer given was taken and recorded before the refusal, so it is not to be given again")
		}
		if tookPace {
			notes = append(notes, "the new pace given was not taken: give it again once this is settled")
		}
		if len(notes) == 0 {
			return DrainResult{}, err
		}
		if r, ok := AsRefusal(err); ok {
			r.Reason += "; " + strings.Join(notes, "; ")
			return DrainResult{}, err
		}
		return DrainResult{}, fmt.Errorf("%w; %s", err, strings.Join(notes, "; "))
	}

	// A move that could not read origin plans nothing: no judgement is taken,
	// no lane is read or opened, and the answer given waits for the next move.
	if wait != "" {
		res.Next = wait
		if d.Judgement != "" {
			res.Next += "; the judgement answer given was not taken: give it again with --judgement once origin answers"
		}
		return finish(started || probed)
	}

	// The reservations as this move's fetch brought them (drain_origin.go):
	// every hold below is judged on this one view, so passing over held work
	// fetches nothing more. Nil without origin, and when the view cannot be
	// read, where each reservation act's own fetch still judges the hold.
	var view *claim.Snapshot
	if origin != nil {
		if snap, err := drainView(repoRoot, now); err == nil {
			view, holds.view = &snap, &snap
		} else {
			viewUnread = "the reservations this move fetched could not be read (" + termsafe.Sanitize(fsutil.RedactHome(err.Error())) +
				"), so held work was judged by each reservation act alone, and the drain's own judgement hold by the commit it records"
		}
	}

	// The host's answer to the judgement the drain awaits (scope 3): taken
	// before anything else moves, so a refusal writes nothing. A yes routes the
	// issue as a lane's hand-back of its kind is routed, and releases the
	// reservation the drain took before it asked, and no other; a no lets the
	// lane open below, on that reservation.
	//
	// answered is the hold the answer was asked on: the lane a no opens on it
	// names the fork check its reservation could not make or waived, as a
	// lane that reserved for itself does (step 10).
	//
	// take applies a judgement over the request st awaits, the host's or a
	// routed judge's (drain_judge.go), and records it.
	take := func(j DrainJudgement) error {
		if j.Applied && j.Answer == JudgementYes {
			r, err := routeHandBack(repoRoot, j.Issue, DrainFromJudgement, HandBack{Kind: j.Kind, Reason: j.Reason}, origin, now)
			if err != nil {
				// Nothing is taken: the answer waits on disk, with its hold.
				if rf, ok := AsRefusal(err); ok {
					rf.Reason += "; the judgement answer given was not taken: give it again with --judgement once this is settled"
				}
				return err
			}
			// The release publishes the hand-back on origin, so no other
			// account's drain judges the issue again while its record stands
			// (spc-2610161044461448 step 3). One the move cannot publish, its
			// hold having moved on or never been the drain's, is kept pending
			// and published over the issue's next release (drain_pending.go).
			clause, pending, known := holds.release(j.Issue, j.Kind)
			if clause != "" {
				r.Wrote += "; " + clause
			}
			if !known && origin != nil {
				pending = "the drain took no reservation of its own to judge " + j.Issue + ": this person held it live from this copy"
			}
			if pending != "" {
				r.Wrote += "; " + pend(&st, j.Issue, j.Kind, judgedBlob(repoRoot, j.Issue, origin), pending, now)
			}
			r.Blob = blobs.of(j.Issue)
			st.HandBacks = append(st.HandBacks, r)
			res.Routed = append(res.Routed, r)
		} else if !j.Applied {
			// The issue left the eligible set since the request, so no lane
			// opens on the reservation taken for it.
			if clause, _, _ := holds.release(j.Issue, ""); clause != "" {
				j.Note += "; " + clause
			}
		}
		st.Judgements = append(st.Judgements, j)
		st.Judging = nil
		res.Judged = &j
		pruneLapse(&st, j.Issue)
		// With eligibility-residual in shadow, the decision adapter is asked
		// beside the answer just taken and the pair is recorded in the newest
		// lab home; it changes nothing the move does (drain_decide.go).
		shadowDrainJudgement(d.Context, repoRoot, o, j, plan)
		return nil
	}
	if d.Judgement != "" {
		j, err := takeJudgement(repoRoot, d.Judgement, st, live, plan, now)
		if err != nil {
			return DrainResult{}, err
		}
		// The judgement keeps the hold it was asked on, so the lane a no
		// opens on it names the fork check its reservation could not make or
		// waived, in this move or a later one (step 10).
		j.Hold = st.Judging.Hold
		if err := take(j); err != nil {
			return DrainResult{}, err
		}
	}

	// The drain takes its judgement hold once per request (drain_hold.go): a
	// request whose hold has lapsed unanswered is dropped and the hold
	// released at once, before anything else in the move can reach the
	// issue, and its issue is asked again neither in this move nor before a
	// cool-down of the take-over margin, and then only after the drain's
	// other eligible work, so a host that polls without answering never
	// holds the issue past one lease, and another account has a window to
	// take it (dcn-2610161315385888, refined by dcn-2610161420314313).
	if j := st.Judging; j != nil && j.Hold != nil && !now.Before(j.Hold.ExpiresAt) {
		l := DrainLapse{Issue: j.Issue, RequestedAt: j.RequestedAt, LapsedAt: j.Hold.ExpiresAt, DroppedAt: now}
		pruneLapse(&st, j.Issue)
		st.Lapsed = append(st.Lapsed, l)
		lapse = fmt.Sprintf("the judgement over %s was not answered before the reservation the drain took for it lapsed at %s, so the request is dropped; %s is asked again no sooner than %s, and only after the drain's other eligible work",
			j.Issue, stamp(j.Hold.ExpiresAt), j.Issue, stamp(l.coolDownEnds()))
		if clause, _, _ := holds.release(j.Issue, ""); clause != "" {
			lapse += "; " + clause
		}
		st.Judging = nil
	}

	// A hand-back a yes could not publish is published over the issue's next
	// release on origin, as this move's view shows it (drain_pending.go).
	if published = publishPending(repoRoot, &st, view, now); len(published) > 0 {
		syncRouted(&res, st)
	}

	// A run that gave up on a lost connection has stopped (iss-2610080620372731),
	// and the drain stops with it: it opens no lane while the outage stands.
	if cur, err := CurrentOutage(repoRoot); err != nil {
		return refuseMoved(outageUnreadable(err))
	} else if cur != nil && cur.Status == implement.OutageGaveUp {
		st.Stopped, st.EndedAt = DrainStoppedOutage, &now
		res.Next = fmt.Sprintf("the run gave up on the %s outage open since %s after %d probe(s), so the drain stops here with %d lane(s) opened; once the connection is back, close the outage with `abcd implement outage clear --session <id> --reason <why>`, then `abcd drain` begins a new drain",
			strings.Join(cur.Services, " and "), cur.StartedAt.UTC().Format(time.RFC3339), len(cur.Probes), len(st.Lanes))
		return finish(true)
	}

	// Merged lanes are finished before the window clock is read: the cleanup
	// does no new work, so the pace does not hold it (iss-2610090936460789).
	if res.Finished, res.Parked, err = finishMerged(repoRoot, now); err != nil {
		return refuseMoved(err)
	}
	// The lane in progress is renewed at half its lease on every move,
	// waiting or not, before the window gate (drain_lease.go).
	leaseNotes = renewInProgress(repoRoot, st, view, now)
	finished := false
	for _, f := range res.Finished {
		for i := range st.Lanes {
			if f.Done && st.Lanes[i].RunID == f.RunID {
				st.Lanes[i].Outcome, st.Lanes[i].PR, finished = DrainLaneDone, f.PR, true
			}
		}
	}
	// A parked lane whose pull request was closed unmerged was handed back as
	// closed, its reservation released, by finishMerged; a lane of this
	// drain's is routed here as a lane's hand-back is (step 8).
	for _, p := range res.Parked {
		// A parked lane that went back through its sync is driven again, as
		// the lane in progress is (step 13).
		if p.State == ParkedSyncing {
			for i := range st.Lanes {
				if st.Lanes[i].RunID == p.RunID && st.Lanes[i].Outcome == DrainLanePullRequest {
					st.Lanes[i].Outcome, finished = DrainLaneInProgress, true
				}
			}
			continue
		}
		if p.State != ParkedClosed {
			continue
		}
		for i := range st.Lanes {
			if st.Lanes[i].RunID != p.RunID || st.Lanes[i].Outcome == DrainLaneHandedBack {
				continue
			}
			run, err := ReadState(repoRoot, p.RunID)
			if err != nil {
				return refuseMoved(err)
			}
			outcome, _, hb := laneOutcome(run)
			if outcome != DrainLaneHandedBack {
				continue
			}
			r, err := routeHandBack(repoRoot, p.Issue, "lane", *hb, origin, now)
			if err != nil {
				return refuseMoved(err)
			}
			if rel := releaseHandedBack(repoRoot, run, *hb, now); rel != "" {
				r.Wrote += "; " + rel
			}
			// The record it judged, so a change to it on origin is a new
			// question this drain takes up again, as every route keeps.
			r.Blob = blobs.of(p.Issue)
			st.HandBacks = append(st.HandBacks, r)
			res.Routed = append(res.Routed, r)
			st.Lanes[i].Outcome, finished = DrainLaneHandedBack, true
		}
	}

	// The window clock, as a run keeps it: before next_eligible_at nothing
	// moves; a pause that has ended opens the next window; a window that has
	// elapsed closes here, and the call opens nothing.
	if st.NextEligibleAt != nil {
		if now.Before(*st.NextEligibleAt) {
			at := st.NextEligibleAt.UTC().Format(time.RFC3339)
			res.Next = "nothing before " + at + ": the drain's window has elapsed; run `abcd drain` again at or after " + at
			// A judgement taken this move was applied above (a yes already
			// routed), so it is recorded even though nothing else moves, and
			// so is a wait on origin this move cleared.
			return finish(res.Judged != nil || finished || probed)
		}
		st.NextEligibleAt, st.WindowStartedAt = nil, &now
	}
	if st.Pace != nil && st.WindowStartedAt != nil {
		if end := st.WindowStartedAt.Add(time.Duration(st.Pace.WorkMinutes.Value) * time.Minute); !now.Before(end) {
			until := now.Add(time.Duration(st.Pace.PauseMinutes.Value) * time.Minute)
			st.NextEligibleAt = &until
			at := until.UTC().Format(time.RFC3339)
			res.Next = fmt.Sprintf("nothing before %s: the drain's %d-minute window has elapsed (a %d-minute pause); a lane in progress may still be driven with `abcd implement step`; run `abcd drain` again at or after %s",
				at, st.Pace.WorkMinutes.Value, st.Pace.PauseMinutes.Value, at)
			return finish(true)
		}
	}

	// The lane the drain opened last: route what it has come to, or wait on it.
	// A lane in progress that is stuck, refusing past the bounds the pace's
	// layers set or in a way that cannot change, is handed back as stuck here,
	// its reservation released with the tombstone, and the move takes up the
	// next issue (stuck.go).
	var bounds *stuckBounds
	for i := range st.Lanes {
		l := &st.Lanes[i]
		if l.Outcome != DrainLaneInProgress {
			continue
		}
		run, err := ReadState(repoRoot, l.RunID)
		if err != nil {
			return refuseMoved(err)
		}
		outcome, pr, hb := laneOutcome(run)
		if outcome == DrainLaneInProgress {
			if bounds == nil {
				b, err := resolveStuckBounds(o.roots(repoRoot))
				if err != nil {
					return refuseMoved(err)
				}
				bounds = &b
			}
			if stuckReason(run, *bounds) != "" {
				if run, err = handBackStuck(repoRoot, l.RunID, *bounds, now); err != nil {
					return refuseMoved(err)
				}
				outcome, pr, hb = laneOutcome(run)
			}
		}
		switch outcome {
		case DrainLaneInProgress:
			res.Next = fmt.Sprintf("drive %s's lane, run %s: `abcd implement step --run %s` (one lane at a time); run `abcd drain` again once it is handed back or its pull request is open",
				l.Issue, l.RunID, l.RunID)
			if note := refusingNote(run, *bounds); note != "" {
				res.Next += "; " + note
			}
			return finish(true)
		case DrainLaneHandedBack:
			r, err := routeHandBack(repoRoot, l.Issue, "lane", *hb, origin, now)
			if err != nil {
				return refuseMoved(err)
			}
			// The issue is a person's again: its reservation is released
			// rather than left to hold others off until it lapses.
			if rel := releaseHandedBack(repoRoot, run, *hb, now); rel != "" {
				r.Wrote += "; " + rel
			}
			r.Blob = blobs.of(l.Issue)
			st.HandBacks = append(st.HandBacks, r)
			res.Routed = append(res.Routed, r)
		}
		l.Outcome, l.PR = outcome, pr
	}

	if st.Max > 0 && len(st.Lanes) >= st.Max {
		st.Stopped, st.EndedAt = DrainStoppedCap, &now
		res.Next = fmt.Sprintf("the cap is reached: --max %d, and the drain opened %d lane(s); it stops here. `abcd drain` again begins a new drain", st.Max, len(st.Lanes))
		return finish(true)
	}

	// A peer this checkout cannot read refuses the move once, here, where
	// per-issue contention begins, rather than holding every issue (step 9):
	// the housekeeping above, which no peer's holding bears on, is done first.
	if err := unreadablePeers(repoRoot); err != nil {
		return refuseMoved(err)
	}

	// The next eligible issue in the drain order, less any this drain has
	// taken or handed back and any this checkout already has a run for. Its
	// lane opens only once the host judgement over its remedy has answered no
	// for the remedy as it stands; until then the drain asks and opens nothing.
	// An issue, or a footprint, another copy holds in the move's view is
	// passed over before anything is pushed or asked; any other is reserved
	// before its judgement is asked, so the push is the compare-and-swap
	// between accounts, and the lane a no opens starts on that reservation
	// (spc-2610161044461448 step 2; dcn-2610161040368890).
	runs, err := Runs(repoRoot)
	if err != nil {
		return refuseMoved(err)
	}
	// In a checkout with a remote named origin, each issue is reserved there
	// as its lane starts, with its found_at path as the footprint (C37);
	// without one there is nowhere to reserve, and nothing is.
	reserving, err := claim.HasOrigin(repoRoot)
	if err != nil {
		return refuseMoved(err)
	}
	// cooling is the earliest a lapsed issue's cool-down this move waited
	// out ends, zero when none: a drain left with only such issues waits
	// for it rather than ending.
	var cooling time.Time
	// The wait this move may end in (drain_wait.go): the cap, counted in the
	// move's view before any reservation or judgement for the holder this
	// copy reserves as, read with no network act (claim.ReaderSelf: unknown
	// when neither a holder id nor a login can be read here, where each
	// reservation act counts it instead), and the
	// issues passed over because another holds them. A wait the drain was
	// in is taken off here, and written again if the move still waits.
	prevWait := waitFrom(&st)
	var capHolder claim.Self
	// The cap is the person's own (reservation.max_live); a setting that
	// cannot be used leaves the count to the reservation act, which refuses
	// naming the file.
	capLimit, capErr := claim.MaxLive, error(nil)
	if view != nil && reserving {
		capHolder = claim.ReaderSelf(repoRoot, claimOptions(now))
		capLimit, capErr = claim.MaxLiveFor(claimOptions(now))
	}
	var capped, held []string
	for _, v := range lapsedLast(plan.Dispositions, st.Lapsed) {
		if v.Outcome != capture.DrainEligible {
			continue
		}
		// An issue this drain has in hand is left; one it handed back is
		// taken up again once its record has changed on origin
		// (drain_readmit.go).
		if st.taken(v.ID, blobs) {
			continue
		}
		// An issue the skip list holds back is passed over before anything
		// is pushed or asked over it (drain_skip.go).
		if iss, ok := plan.OpenIssue(v.ID); ok {
			if s, ok := skipOf(st, iss); ok {
				res.Passed = append(res.Passed, skipPassed(v.ID, s))
				continue
			}
		}
		if l, ok := lapseOf(st, v.ID); ok && now.Before(l.coolDownEnds()) {
			ends := l.coolDownEnds()
			if cooling.IsZero() || ends.Before(cooling) {
				cooling = ends
			}
			res.Passed = append(res.Passed, Excluded{ID: v.ID, Check: CheckLapsed,
				Reason: fmt.Sprintf("its judgement request lapsed unanswered at %s, and it is asked again no sooner than %s, so another account has a window to take it",
					stamp(l.LapsedAt), stamp(ends))})
			continue
		}
		// A run of the issue here that was not handed back passes it over,
		// in progress or complete: a complete one landed its pull request,
		// which may still be open, and a second run would open a second.
		// One handed back does too without origin, where nothing says
		// whether its record has changed since; on origin the hand-back's
		// tombstone and this drain's routes say so (drain_readmit.go).
		if i := slices.IndexFunc(runs, func(r State) bool { return r.Key == v.ID && !r.issueHandedBack() }); i >= 0 {
			reason := "this checkout already has run " + runs[i].RunID + " for it; drive or finish that run"
			if runs[i].Complete() {
				reason = "this checkout's run " + runs[i].RunID + " for it is complete, and what it landed settles the issue, so no second run opens for it"
			}
			res.Passed = append(res.Passed, Excluded{ID: v.ID, Check: CheckRun, Reason: reason})
			continue
		}
		if i := slices.IndexFunc(runs, func(r State) bool { return r.Key == v.ID }); i >= 0 && origin == nil {
			res.Passed = append(res.Passed, Excluded{ID: v.ID, Check: CheckRun,
				Reason: "this checkout's run " + runs[i].RunID + " for it was handed back, and with no remote named " + claim.Remote +
					" the drain cannot tell whether its record has changed since; it is a person's"})
			continue
		}
		iss, ok := plan.OpenIssue(v.ID)
		if !ok {
			continue
		}
		// An issue with no found_at path cannot be reserved: it is handed
		// back before any judgement is asked over it, and the drain moves on.
		var footprint []string
		if reserving {
			if footprint, ok = issueFootprint(iss); !ok {
				r := noFoundAtHandBack(v.ID, now)
				r.Blob = blobs.of(v.ID)
				st.HandBacks = append(st.HandBacks, r)
				res.Routed = append(res.Routed, r)
				continue
			}
		}
		if view != nil {
			// A hand-back published on origin, by any copy, stands until a
			// person changes the issue's record there: the drain passes it
			// over before anything is pushed or asked (spc-2610161044461448
			// step 3).
			if why := view.HandBack(v.ID, now); why != "" {
				res.Passed = append(res.Passed, Excluded{ID: v.ID, Check: CheckReservation,
					Reason: termsafe.Sanitize(why) + "; passed over with nothing pushed and no judgement asked: it is a person's until its record changes"})
				continue
			}
			// Under --take-over, this person's own reservation from another
			// copy holds nothing off: the reservation act below takes it over
			// (spc-2610161044461448 step 15). A hosted session that restarted
			// lost its local tier, so its earlier copy is another copy; it is
			// still this person's only when its holder id is (claim.Self): kept
			// across sessions by ABCD_HOLDER_ID, or declared in holder-ids. A
			// restart that minted a fresh id reads its earlier holds as another
			// person's, which no take-over moves.
			var takeOver claim.Self
			if o.TakeOver {
				takeOver = capHolder
			}
			if why := view.Contention(v.ID, footprint, now, takeOver); why != "" {
				// The reason quotes what another writer chose (a reserved
				// footprint), and the agent driving the drain reads it.
				why = termsafe.Sanitize(why) + "; passed over with nothing pushed and no judgement asked"
				if r, ok := view.Lookup(v.ID); ok && r.State == claim.StateLive && r.Copy != claim.CopyThis && capHolder.Owns(r.Holder) {
					why += "; it is your own reservation from another copy: to move it to this one, run `abcd drain --take-over`"
				}
				res.Passed = append(res.Passed, Excluded{ID: v.ID, Check: CheckReservation, Reason: why})
				held = append(held, termsafe.Sanitize(why))
				continue
			}
		}
		hold, owned := ownHold(view, v.ID, footprint)
		judged := judgementFor(st, v.ID, remedyDigest(iss.Remedy))
		// routedHold is the hold a routed judge's no was asked on in this
		// move, which the move's view, fetched before it was taken, cannot
		// show: the lane opens on it as on one the view shows.
		var routedHold *DrainHold
		if j := judged; j == nil || j.Answer != JudgementNo {
			// The drain's hold is taken once per request: a request awaited
			// over this issue keeps the one it took, and a poll never renews
			// it (dcn-2610161315385888). It is kept only while the move's view
			// still shows its commit at the tip: one released or renewed by
			// hand since, or released by a move whose state write then
			// failed, is no hold any longer, so the issue is reserved again
			// (or, renewed by hand, left on the person's own reservation)
			// before the judgement is asked.
			var judgeHold *DrainHold
			if p := st.Judging; p != nil && p.Issue == v.ID {
				judgeHold = p.Hold
				if judgeHold != nil && view != nil {
					if r, ok := view.Lookup(v.ID); !ok || r.Commit != judgeHold.Commit {
						holds.forget(v.ID, judgeHold.Commit)
						judgeHold = nil
					}
				}
			}
			if reserving && !owned && judgeHold == nil {
				// At the cap no reservation is made and no judgement asked:
				// the drain waits for a pull request to merge.
				if capHolder.Known() && capErr == nil {
					if mine := view.LiveHeldBy(capHolder, v.ID); len(mine) >= capLimit {
						capped = mine
						break
					}
				}
				h, err := reserveForJudgement(repoRoot, v.ID, footprint, o.TakeOver, now)
				if err != nil {
					if r, ok := AsRefusal(err); ok && r.capped != nil {
						capped = r.capped
						break
					}
					if r, ok := AsRefusal(err); ok && r.Contention {
						res.Passed = append(res.Passed, Excluded{ID: v.ID, Check: r.Check, Reason: termsafe.Sanitize(r.Reason)})
						held = append(held, termsafe.Sanitize(r.Reason))
						continue
					}
					return refuseMoved(err)
				}
				holds.add(v.ID, h, true)
				judgeHold = h
			}
			judging, err := requestJudgement(repoRoot, iss, v, st.Judging, now)
			if err != nil {
				return refuseMoved(err)
			}
			judging.Hold = judgeHold
			st.Judging = &judging
			pruneLapse(&st, v.ID)
			// The decision adapter, when eligibility-residual is turned on,
			// and then a judge routed to a runner answer in this move, taken
			// as the host's answer is; one that does not answer leaves the
			// host's request standing, and is not asked again for it
			// (drain_decide.go, drain_judge.go). Each is marked asked alone,
			// and only once it was: a runner that never started is started
			// by a later move, whatever the adapter did.
			var taken *DrainJudgement
			clause := ""
			if judging.AdapterAsked {
				clause = adapterAskedClause(v.ID)
			} else {
				var adapterAsked bool
				if taken, clause, adapterAsked, err = adapterJudgement(d.Context, repoRoot, o, judging, plan, &st, now); err != nil {
					return refuseMoved(err)
				}
				if taken == nil && adapterAsked {
					st.Judging.AdapterAsked = true
				}
			}
			if taken == nil {
				if judging.RoutedAsked {
					clause = joinNext(clause, judgeAskedClause(v.ID))
				} else {
					var routed string
					var started bool
					if taken, routed, started, err = routedJudgement(repoRoot, iss, v, judging, plan, &st, o, d, now); err != nil {
						return refuseMoved(err)
					}
					clause = joinNext(clause, routed)
					if taken == nil && started {
						st.Judging.RoutedAsked = true
					}
				}
			}
			if taken == nil {
				res.Next = joinNext(judgementMove(*st.Judging), clause)
				return finish(true)
			}
			taken.Hold = judging.Hold
			if err := take(*taken); err != nil {
				return refuseMoved(err)
			}
			if taken.Answer != JudgementNo || !taken.Applied {
				continue
			}
			judged, routedHold = &st.Judgements[len(st.Judgements)-1], judging.Hold
		}
		st.Judging = nil
		// Every lane runs on the drain's pace, not on what the configuration
		// resolves when it opens (drain_pace.go).
		lane := o
		lane.Pace, lane.SubAgents, lane.FixRounds, lane.pace = nil, nil, nil, st.Pace
		lane.reserve = footprint
		lane.plan = &plan
		lane.drain = true
		if owned {
			lane.held = &heldReservation{at: hold.ClaimedAt, note: fmt.Sprintf("reserved %s on %s for this copy until %s before its judgement, footprint: %s",
				v.ID, claim.Remote, stamp(hold.EffectiveExpiry), strings.Join(hold.Footprint, ", "))}
			// The fork check the hold's reservation could not make or waived
			// is the recorded judgement's, whichever move opens the lane.
			if h := judged.Hold; h != nil && h.Commit == hold.Commit {
				lane.held.forkUnchecked, lane.held.forkAllowed = h.ForkUnchecked, h.ForkAllowed
				if h.ForkUnchecked != "" {
					lane.held.note += "; " + claim.ForkUncheckedLine(h.ForkUnchecked)
				}
				if h.ForkAllowed != "" {
					lane.held.note += "; " + claim.ForkAllowedLine(h.ForkAllowed)
				}
			}
		} else if h := routedHold; h != nil {
			lane.held = &heldReservation{at: h.ClaimedAt, note: fmt.Sprintf("reserved %s on %s for this copy until %s before its judgement, footprint: %s",
				v.ID, claim.Remote, stamp(h.ExpiresAt), strings.Join(footprint, ", ")),
				forkUnchecked: h.ForkUnchecked, forkAllowed: h.ForkAllowed}
			if h.ForkUnchecked != "" {
				lane.held.note += "; " + claim.ForkUncheckedLine(h.ForkUnchecked)
			}
			if h.ForkAllowed != "" {
				lane.held.note += "; " + claim.ForkAllowedLine(h.ForkAllowed)
			}
		}
		holds.starting(v.ID)
		if drainBeforeStart != nil {
			drainBeforeStart(v.ID)
		}
		started, err := startLane(repoRoot, v.ID, lane)
		if err != nil {
			r, ok := AsRefusal(err)
			// At the cap the lane's own reservation is not made: the drain
			// waits for a pull request to merge, the judgement kept.
			if ok && r.capped != nil {
				if clause, _, _ := holds.release(v.ID, ""); clause != "" {
					res.Next = joinNext(res.Next, clause)
				}
				capped = r.capped
				break
			}
			// A reservation refused for a reason of this checkout's own (no
			// username, origin not answering, the reservations unreadable
			// here) stops the drain's move; another's hold passes over the
			// issue. Either way the judgement taken above is kept.
			if !ok || (r.Stage == StageReservation && !r.Contention) {
				return refuseMoved(err)
			}
			// A peer the start's own check could not read, there since the
			// move's check above, stops the move with the move's refusal:
			// it would hold every issue after this one as well.
			if err := unreadableRefusal(repoRoot, r.unjudged); err != nil {
				return refuseMoved(err)
			}
			// The reason may quote what another writer chose (a reserved
			// footprint), and the agent driving the drain reads it.
			reason := termsafe.Sanitize(r.Reason)
			// A reservation the drain took for the judgement is no lane's now.
			if clause, _, _ := holds.release(v.ID, ""); clause != "" {
				reason += "; " + clause
			}
			res.Passed = append(res.Passed, Excluded{ID: v.ID, Check: r.Check, Reason: reason})
			if r.Contention {
				held = append(held, reason)
			}
			continue
		}
		holds.keep(v.ID)
		st.Lanes = append(st.Lanes, DrainLane{Issue: v.ID, RunID: started.RunID, OpenedAt: now, Outcome: DrainLaneInProgress})
		res.Start = &started
		res.Next = fmt.Sprintf("drive %s's lane, run %s: %s; run `abcd drain` again once it is handed back or its pull request is open",
			v.ID, started.RunID, started.Next)
		return finish(true)
	}
	st.Judging = nil
	// At the cap, or with only work others hold left, the drain waits and
	// keeps its judgements; it stops as empty only when nothing eligible is
	// left, held or not (drain_wait.go).
	if capped != nil {
		return finish(waitMove(&st, &res, prevWait, DrainWaitCap, capWaitText(capped, runs), now, time.Time{}))
	}
	if len(held) > 0 {
		return finish(waitMove(&st, &res, prevWait, DrainWaitHeld, heldWaitText(held, len(st.Judgements)), now, cooling))
	}
	if !cooling.IsZero() {
		at := stamp(cooling)
		res.Next = fmt.Sprintf("nothing to ask before %s: the only eligible work left is a lapsed judgement request's issue, asked again once its cool-down has passed; run `abcd drain` again at or after %s", at, at)
		return finish(true)
	}
	st.Stopped, st.EndedAt = DrainStoppedEmpty, &now
	res.Next = fmt.Sprintf("nothing eligible is left: the drain opened %d lane(s) and ends here", len(st.Lanes))
	return finish(true)
}

// joinNext joins the clauses of a move's next step that are not empty with
// "; ", so a step that was empty never starts or ends with the separator.
func joinNext(parts ...string) string {
	kept := make([]string, 0, len(parts))
	for _, p := range parts {
		if p != "" {
			kept = append(kept, p)
		}
	}
	return strings.Join(kept, "; ")
}

// ownHold is this copy's live reservation of id in view, when its footprint
// covers footprint: the one the drain took before it asked the judgement
// over id. False with no view.
func ownHold(view *claim.Snapshot, id string, footprint []string) (claim.Reservation, bool) {
	if view == nil {
		return claim.Reservation{}, false
	}
	r, ok := view.Lookup(id)
	if !ok || r.Copy != claim.CopyThis || r.State != claim.StateLive || !covers(r.Footprint, footprint) {
		return claim.Reservation{}, false
	}
	return r, true
}

// capPhrase names a drain's cap.
func capPhrase(max int) string {
	if max == 0 {
		return "no cap (all)"
	}
	return fmt.Sprintf("--max %d", max)
}

// laneOutcome reads what an issue run has come to: handed back (with the
// hand-back), a pull request opened (armed or left open), done, or in
// progress.
func laneOutcome(run State) (string, int, *HandBack) {
	i := run.current()
	if run.Complete() || i < 0 {
		// A lane done before the drain read it keeps its pull request.
		pr := 0
		for _, l := range run.Lanes {
			if l.PR > 0 {
				pr = l.PR
			}
		}
		return DrainLaneDone, pr, nil
	}
	l := run.Lanes[i]
	switch {
	case l.HandBack != nil:
		return DrainLaneHandedBack, 0, l.HandBack
	case l.Landing != nil && l.Landing.Merge != "":
		return DrainLanePullRequest, l.PR, nil
	}
	return DrainLaneInProgress, 0, nil
}

// drainCounts totals the drain as st holds it (scope 8), with the rule's flags
// and the issues passed over as this call read them. The spend reads each
// lane's run; a run whose state cannot be read is named, not refused, since a
// summary that will not render is worse than one that says it is a floor.
func drainCounts(repoRoot string, st DrainState, flags []DrainRoute, passed []Excluded) DrainCounts {
	c := DrainCounts{LanesOpened: len(st.Lanes), RuleFlags: len(flags), Judgements: len(st.Judgements), Passed: len(passed),
		Spend: DrainSpend{Unread: []string{}}}
	for _, l := range st.Lanes {
		switch l.Outcome {
		case DrainLaneInProgress:
			c.InProgress++
		case DrainLaneHandedBack:
			c.LanesHandedBack++
		}
		if l.PR > 0 {
			c.PullRequests++
		}
		run, err := ReadState(repoRoot, l.RunID)
		if err != nil {
			c.Spend.Unread = append(c.Spend.Unread, l.RunID)
			continue
		}
		for _, rl := range run.Lanes {
			c.Spend.Implementers += len(rl.Receipts)
			for _, round := range rl.Validation {
				for _, v := range round.Validators {
					if v.Verdict != "" {
						c.Spend.Validators++
					}
				}
			}
		}
	}
	c.Spend.AgentRuns = c.Spend.Implementers + c.Spend.Validators
	for _, r := range st.HandBacks {
		switch r.Route {
		case RoutePromoted:
			c.Promotions++
		case RouteDecisionRecord:
			c.DecisionRecords++
		case RouteHome:
			c.Homes++
		}
	}
	return c
}

// fieldFlags are the rule's hand-backs over the ledger: each open issue the
// rule hands back, flagged naming the rule that did. They are written nowhere
// (decision 8: the classification is re-derived every run). An issue the
// drain itself promoted, among routed, reads as promoted on every move after,
// and that is the drain's own hand-back, counted once among routed's
// promotions; it is not flagged as the rule's as well. A flag says which it
// is by its fields: `from` is "field" and `route` is "rule", with the rule in
// `rule`, where a promotion the drain routed has `route` "promoted" and its
// draft.
func fieldFlags(plan capture.DrainPlan, routed []DrainRoute) []DrainRoute {
	out := []DrainRoute{}
	for _, v := range plan.Dispositions {
		if v.Outcome != capture.DrainHandBack {
			continue
		}
		if v.Rule == capture.RulePromoted && slices.ContainsFunc(routed, func(r DrainRoute) bool {
			return r.Route == RoutePromoted && r.Issue == v.ID
		}) {
			continue
		}
		out = append(out, DrainRoute{Issue: v.ID, From: "field", Kind: string(v.Rule), Route: RouteRule, Rule: string(v.Rule),
			Reason: v.Reason, Wrote: "nothing: a flag in this summary"})
	}
	return out
}

// routeHandBack routes a hand-back by its kind (scope 5), from a lane or from
// the host judgement before one opened, and makes the one record change a
// route makes: a user moment is promoted to an intent draft (`capture
// promote`, which stamps the issue's related_intents and nothing else); every
// other kind is a flag and writes nothing. A promotion a killed call already
// made is found, not made twice.
//
// A move planned from origin (origin not nil) promotes only an issue this
// checkout holds as origin's default branch does: one absent here, or held
// here as another record, is not promoted here, and the route says why. The
// hand-back is still published on origin by the release that follows
// (spc-2610161044461448 step 3), so a person whose checkout holds the record
// promotes it.
func routeHandBack(repoRoot, issue, from string, hb HandBack, origin *DrainOrigin, now time.Time) (DrainRoute, error) {
	r := DrainRoute{Issue: issue, From: from, Kind: hb.Kind, Reason: hb.Reason, Wrote: "nothing: a flag in this summary", At: &now}
	switch hb.Kind {
	case HandBackUserVisible:
		// A promotion a killed call already made is found first: it has
		// changed this checkout's record, which origin's no longer matches.
		if into, err := promotedDraft(repoRoot, issue); err == nil && into != "" {
			r.Route, r.Draft = RoutePromoted, into
			r.Wrote = "intent draft " + r.Draft + ", and " + issue + "'s related_intents names it"
			return r, nil
		}
		if why, err := notHeldAsOrigin(repoRoot, issue, origin); err != nil {
			return DrainRoute{}, err
		} else if why != "" {
			r.Route, r.Home = RouteHome, "a checkout that holds "+issue+" as "+claim.Remote+"'s default branch does: `abcd capture promote "+issue+"` there"
			r.Wrote = "nothing here: " + why + ", so it is not promoted here"
			return r, nil
		}
		r.Route = RoutePromoted
		pr, err := capture.Promote(capture.PromoteRequest{RepoRoot: repoRoot, ID: issue})
		switch {
		case err == nil:
			r.Draft = pr.IntentID
		case errors.Is(err, capture.ErrAlreadyPromoted):
			into, ferr := promotedDraft(repoRoot, issue)
			if ferr != nil {
				return DrainRoute{}, ferr
			}
			r.Draft = into
		default:
			return DrainRoute{}, refuse(StageDrain, "", "", "`capture promote "+issue+"` refused routing its "+from+"'s hand-back: "+fsutil.RedactHome(err.Error()),
				"settle what the capture store names, then run `abcd drain` again; the hand-back is routed then")
		}
		r.Wrote = "intent draft " + r.Draft + ", and " + issue + "'s related_intents names it"
	case HandBackTrustRule:
		r.Route, r.Question = RouteDecisionRecord, hb.Reason
	case HandBackDesignFinding, HandBackSecondPackage:
		r.Route, r.Home = RouteHome, hb.Home
	case HandBackStuck, HandBackClosed:
		// The lane kept refusing (stuck.go), or its pull request was closed
		// unmerged (drain_finish.go): the issue stays open for a person, and
		// the hand-back names the discard of its run.
		r.Route, r.Home = RouteHome, hb.Home
	default:
		// The lane stopped after its fix rounds (itd-50): the issue stays open
		// with the last findings, a person's to judge.
		r.Route, r.Kind, r.Home = RouteHome, "unachievable", "the issue stays open for a person, with the lane's last findings"
		r.Reason = handBackSummary(issue, hb)
	}
	return r, nil
}

// promotedDraft is the draft an already-promoted issue names.
func promotedDraft(repoRoot, issue string) (string, error) {
	lr, err := capture.List(capture.ListRequest{RepoRoot: repoRoot, State: capture.StateOpen})
	if err != nil {
		return "", err
	}
	for _, iss := range lr.Issues {
		if iss.ID == issue {
			return capture.PromotedInto(repoRoot, iss)
		}
	}
	return "", fmt.Errorf("%s is promoted and no longer open, so the draft it names cannot be read", issue)
}

// readDrain reads the drain's state; live is false when there is none, or the
// one there has ended.
func readDrain(repoRoot string) (DrainState, bool, error) {
	root, err := os.OpenRoot(repoRoot)
	if err != nil {
		return DrainState{}, false, fmt.Errorf("opening the checkout: %w", err)
	}
	defer root.Close()
	data, err := localtier.ReadInRoot(root, DrainStateRel, maxDrainStateBytes)
	if errors.Is(err, os.ErrNotExist) {
		return DrainState{}, false, nil
	}
	if rf := tierRefusal(StageDrain, "", err); rf != nil {
		return DrainState{}, false, rf
	}
	if err != nil {
		return DrainState{}, false, refuse(StageDrain, "", "", fmt.Sprintf("%s cannot be read as the drain's state: %v", DrainStateRel, err),
			"the drain writes a regular file in a real directory; restore that, or remove "+DrainStateRel)
	}
	// A newer abcd's drain state is refused as that before any strict decode,
	// as the run state's is (decodeState): a field the newer version added is
	// never read as a file that does not parse, whose remedy would throw the
	// newer run's drain away (iss-2610230223231843).
	var peek struct {
		SchemaVersion int `json:"schema_version"`
	}
	if json.Unmarshal(data, &peek) == nil && peek.SchemaVersion > DrainSchemaVersion {
		return DrainState{}, false, refuse(StageDrain, "", "", fmt.Sprintf("%s is schema version %d, which a newer abcd wrote; this abcd reads version %d",
			DrainStateRel, peek.SchemaVersion, DrainSchemaVersion),
			"run the newer abcd that wrote it (`abcd update` brings this one up to date); do not remove the drain state, which holds the drain")
	}
	var st DrainState
	if err := jsonstrict.Decode(data, &st); err != nil || (st.SchemaVersion != DrainSchemaVersion && st.SchemaVersion != drainSchemaVersionUnpinned) {
		why := fmt.Sprintf("schema_version %d (this abcd reads %d to %d)", st.SchemaVersion, drainSchemaVersionUnpinned, DrainSchemaVersion)
		if err != nil {
			why = err.Error()
		}
		return DrainState{}, false, refuse(StageDrain, "", "", DrainStateRel+" does not parse as the drain's state: "+why,
			"the drain is the file's only writer; restore it or remove it")
	}
	if st.SchemaVersion == drainSchemaVersionUnpinned {
		if p := cmp.Or(st.pinned(), st.overridden()); p != "" {
			return DrainState{}, false, refuse(StageDrain, "", "", fmt.Sprintf("%s is schema version %d but carries %s, which version %d never wrote",
				DrainStateRel, st.SchemaVersion, p, st.SchemaVersion),
				"the drain is the file's only writer; restore it or remove it")
		}
		// Read as the current version; the move's write carries it.
		st.SchemaVersion = DrainSchemaVersion
	}
	// The stored pace names the layers and origins the resolver writes, so no
	// other text reaches the drain's --json payload or the refusal of a move
	// typed with another pace (iss-2610121206163414).
	if st.Pace != nil && !st.Pace.written() {
		return DrainState{}, false, refuse(StageDrain, "", "", DrainStateRel+" names a pace the drain never writes",
			"the drain is the file's only writer; restore it or remove it")
	}
	// The stored lane, hand-back and judgement text carries no control, bidi
	// or zero-width rune, so none reaches the drain's --json payload; the
	// refusal does not echo it (iss-2610121505153157).
	if !st.OriginWait.written() {
		return DrainState{}, false, refuse(StageDrain, "", "", DrainStateRel+" names a wait on origin the drain never writes",
			"the drain is the file's only writer; restore it or remove it")
	}
	if !st.textWritten() {
		return DrainState{}, false, refuse(StageDrain, "", "", DrainStateRel+" carries text the drain never writes: a control, bidi or zero-width character in its rule, a lane, a hand-back or a judgement",
			"the drain is the file's only writer; restore it or remove it")
	}
	if !st.waitWritten() {
		return DrainState{}, false, refuse(StageDrain, "", "", DrainStateRel+" names a wait the drain never writes",
			"the drain is the file's only writer; restore it or remove it")
	}
	if !st.skipsWritten() {
		return DrainState{}, false, refuse(StageDrain, "", "", DrainStateRel+" names a skip-list entry the drain never writes",
			"the drain is the file's only writer; restore it or remove it")
	}
	if !st.pendingWritten() {
		return DrainState{}, false, refuse(StageDrain, "", "", DrainStateRel+" names a pending hand-back the drain never writes",
			"the drain is the file's only writer; restore it or remove it")
	}
	if slices.ContainsFunc(st.Judgements, func(j DrainJudgement) bool { return j.Hold != nil && !gitutil.IsFullSHA(j.Hold.Commit) }) {
		return DrainState{}, false, refuse(StageDrain, "", "", DrainStateRel+" names a judgement hold that is not a reservation commit",
			"the drain is the file's only writer; restore it or remove it")
	}
	if slices.ContainsFunc(st.Lapsed, func(l DrainLapse) bool { return !validIssueKey(l.Issue) }) {
		return DrainState{}, false, refuse(StageDrain, "", "", DrainStateRel+" names a lapsed judgement request that is not an issue",
			"the drain is the file's only writer; restore it or remove it")
	}
	if j := st.Judging; j != nil && j.Hold != nil && !gitutil.IsFullSHA(j.Hold.Commit) {
		return DrainState{}, false, refuse(StageDrain, "", "", DrainStateRel+" names a judgement hold that is not a reservation commit",
			"the drain is the file's only writer; restore it or remove it")
	}
	for _, l := range st.Lanes {
		if !validIssueKey(l.Issue) || !ValidRunID(l.RunID) {
			return DrainState{}, false, refuse(StageDrain, "", "", DrainStateRel+" names a lane that is not an issue and a run",
				"the drain is the file's only writer; restore it or remove it")
		}
	}
	return st, st.Stopped == "", nil
}

// pinned reports the first judgement, in a drain state, whose route carries a
// pin or names its model as local, which only version 2 and later write, or
// "". A judgement's route is the one route record the drain state stores.
func (s DrainState) pinned() string {
	for i, j := range s.Judgements {
		if k := pinKey(j.Route); k != "" {
			return fmt.Sprintf("`%s` on judgement %d's route (%s)", k, i+1, j.Issue)
		}
	}
	return ""
}

// overridden reports the first judgement, in a drain state, whose route
// carries `override`, which only version 2 and later write, or "".
func (s DrainState) overridden() string {
	for i, j := range s.Judgements {
		if j.Route != nil && j.Route.Override != "" {
			return fmt.Sprintf("`override` on judgement %d's route (%s)", i+1, j.Issue)
		}
	}
	return ""
}

// writeDrain writes the drain's state atomically.
func writeDrain(repoRoot string, st DrainState) error {
	root, err := os.OpenRoot(repoRoot)
	if err != nil {
		return fmt.Errorf("opening the checkout: %w", err)
	}
	defer root.Close()
	data, err := json.MarshalIndent(st, "", "  ")
	if err != nil {
		return fmt.Errorf("encoding the drain's state: %w", err)
	}
	data = append(data, '\n')
	return fsutil.WriteFileAtomicInRoot(root, DrainStateRel, data, filePerm)
}

// DrainSummaryLine is a route in one line, for the text surfaces.
func DrainSummaryLine(r DrainRoute) string {
	var b strings.Builder
	fmt.Fprintf(&b, "%s (%s, %s): ", r.Issue, r.From, r.Kind)
	switch r.Route {
	case RoutePromoted:
		fmt.Fprintf(&b, "promoted to %s", r.Draft)
	case RouteDecisionRecord:
		fmt.Fprintf(&b, "needs a decision record; the question: %s", r.Question)
	case RouteRule:
		fmt.Fprintf(&b, "a person's by the rule %s: %s", r.Rule, r.Reason)
	case RouteHome:
		fmt.Fprintf(&b, "a person's, its home: %s", r.Home)
	}
	if r.Route != RouteRule && r.Reason != "" && r.Route != RouteDecisionRecord {
		fmt.Fprintf(&b, " (%s)", r.Reason)
	}
	fmt.Fprintf(&b, "; wrote %s", r.Wrote)
	return b.String()
}

// syncRouted brings each route this call routed up to date with the drain's
// record of it, which a pending hand-back published later in the move adds
// to.
func syncRouted(res *DrainResult, st DrainState) {
	for i, r := range res.Routed {
		for _, h := range st.HandBacks {
			if h.Issue == r.Issue && h.From == r.From && h.At != nil && r.At != nil && h.At.Equal(*r.At) {
				res.Routed[i] = h
			}
		}
	}
}
