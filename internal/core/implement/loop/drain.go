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
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/intentdriven/abcd/internal/core/capture"
	"github.com/intentdriven/abcd/internal/core/implement"
	"github.com/intentdriven/abcd/internal/core/jsonstrict"
	"github.com/intentdriven/abcd/internal/fsutil"
)

// DrainStateRel is the drain's state file, beside the runs. Its name is not a
// run id, so the run listing never reads it as one.
const DrainStateRel = RunRelDir + "/drain.json"

// drainLockRel is the lock one drain invocation holds from its first read to
// its last write, so two drains never open two lanes. It is not the runs'
// lock, which Start takes inside it.
const drainLockRel = RunRelDir + "/drain.lock"

// DrainSchemaVersion is the drain state's shape.
const DrainSchemaVersion = 1

// maxDrainStateBytes caps the drain state read.
const maxDrainStateBytes = 4 << 20

// StageDrain is the refusal stage of the drain run.
const StageDrain = "drain"

// Why a drain ended.
const (
	// DrainStoppedCap: the drain opened --max lanes.
	DrainStoppedCap = "cap"
	// DrainStoppedEmpty: no eligible issue is left that this drain has not
	// taken.
	DrainStoppedEmpty = "empty"
	// DrainStoppedOutage (outage.go): the run gave up on a lost connection.
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
	// Stopped says why the drain ended, and EndedAt when; empty while it runs.
	Stopped string     `json:"stopped,omitempty"`
	EndedAt *time.Time `json:"ended_at,omitempty"`
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
	// judgement's before any lane opened, "field" for the rule's.
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
}

// DrainOptions are the drain's own flags.
type DrainOptions struct {
	// Max is --max as typed; 0 when not given, which is all.
	Max int
	// Judgement is --judgement: the path of the host's answer to the
	// judgement the drain awaits; empty when not given.
	Judgement string
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
	// call read it, each naming its rule, written nowhere.
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
	Stopped        string                 `json:"stopped,omitempty"`
	Complete       bool                   `json:"complete"`
	Next           string                 `json:"next"`
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
	// The rule first: a repository without its own record is refused before
	// anything else is read or written (criterion 11).
	plan, err := capture.PlanDrain(capture.DrainPlanRequest{RepoRoot: repoRoot})
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
		res, err = drainMove(repoRoot, o, d, flags, plan)
		return err
	})
	if errors.Is(err, fsutil.ErrLockContention) {
		return DrainResult{}, contend(StageDrain, "", "", "another drain is moving in this checkout", "back off and retry")
	}
	return res, err
}

// drainMove is Drain under the drain's lock.
func drainMove(repoRoot string, o Options, d DrainOptions, flags paceFlags, plan capture.DrainPlan) (DrainResult, error) {
	now := o.now()
	st, live, err := readDrain(repoRoot)
	if err != nil {
		return DrainResult{}, err
	}
	started := false
	if !live {
		pace, err := resolvePace(o.roots(repoRoot), flags)
		if err != nil {
			return DrainResult{}, err
		}
		st = DrainState{SchemaVersion: DrainSchemaVersion, StartedAt: now, UpdatedAt: now, Rule: plan.Record,
			Max: d.Max, Pace: &pace, WindowStartedAt: &now, Lanes: []DrainLane{}, HandBacks: []DrainRoute{}}
		started = true
	} else {
		if d.Max != 0 && d.Max != st.Max {
			return DrainResult{}, refuse(StageDrain, "", "", fmt.Sprintf("the drain started %s is in progress with %s; --max %d names another, and a cap is set when a drain starts",
				st.StartedAt.Format(time.RFC3339), capPhrase(st.Max), d.Max),
				"drain again without --max; the drain keeps the cap it started with")
		}
		if flags.set {
			if err := resumeWithFlags(StartResult{RunID: "the drain started " + st.StartedAt.Format(time.RFC3339), Pace: st.Pace}, flags); err != nil {
				return DrainResult{}, err
			}
		}
	}
	res := DrainResult{State: DrainStateRel, Started: started, Rule: plan.Record, Loosened: plan.Loosened,
		Order: plan.Order, Dispositions: plan.Dispositions, Routed: []DrainRoute{}, Passed: []Excluded{},
		Flags: fieldFlags(plan)}
	finish := func(write bool) (DrainResult, error) {
		if write {
			st.UpdatedAt = now
			if err := writeDrain(repoRoot, st); err != nil {
				return DrainResult{}, err
			}
		}
		res.Max, res.Pace, res.Lanes, res.HandBacks = st.Max, st.Pace, st.Lanes, st.HandBacks
		res.Judging, res.Judgements = st.Judging, st.Judgements
		if res.Judgements == nil {
			res.Judgements = []DrainJudgement{}
		}
		res.NextEligibleAt, res.Stopped, res.Complete = st.NextEligibleAt, st.Stopped, st.Stopped != ""
		if i := slices.IndexFunc(st.Lanes, func(l DrainLane) bool { return l.Outcome == DrainLaneInProgress }); i >= 0 {
			l := st.Lanes[i]
			res.Lane = &l
		}
		return res, nil
	}

	// The host's answer to the judgement the drain awaits (scope 3): taken
	// before anything else moves, so a refusal writes nothing. A yes routes the
	// issue as a lane's hand-back of its kind is routed; a no lets the lane
	// open below.
	if d.Judgement != "" {
		j, err := takeJudgement(repoRoot, d.Judgement, st, live, plan, now)
		if err != nil {
			return DrainResult{}, err
		}
		if j.Applied && j.Answer == JudgementYes {
			r, err := routeHandBack(repoRoot, j.Issue, DrainFromJudgement, HandBack{Kind: j.Kind, Reason: j.Reason}, now)
			if err != nil {
				return DrainResult{}, err
			}
			st.HandBacks = append(st.HandBacks, r)
			res.Routed = append(res.Routed, r)
		}
		st.Judgements = append(st.Judgements, j)
		st.Judging = nil
		res.Judged = &j
	}

	// A run that gave up on a lost connection has stopped (iss-2610080620372731),
	// and the drain stops with it: it opens no lane while the outage stands.
	if cur, err := CurrentOutage(repoRoot); err != nil {
		return DrainResult{}, outageUnreadable(err)
	} else if cur != nil && cur.Status == implement.OutageGaveUp {
		st.Stopped, st.EndedAt = DrainStoppedOutage, &now
		res.Next = fmt.Sprintf("the run gave up on the %s outage open since %s after %d probe(s), so the drain stops here with %d lane(s) opened; once the connection is back, close the outage with `abcd implement outage clear --session <id> --reason <why>`, then `abcd drain` begins a new drain",
			strings.Join(cur.Services, " and "), cur.StartedAt.UTC().Format(time.RFC3339), len(cur.Probes), len(st.Lanes))
		return finish(true)
	}

	// The window clock, as a run keeps it: before next_eligible_at nothing
	// moves; a pause that has ended opens the next window; a window that has
	// elapsed closes here, and the call opens nothing.
	if st.NextEligibleAt != nil {
		if now.Before(*st.NextEligibleAt) {
			at := st.NextEligibleAt.UTC().Format(time.RFC3339)
			res.Next = "nothing before " + at + ": the drain's window has elapsed; run `abcd drain` again at or after " + at
			// A judgement taken this move was applied above (a yes already
			// routed), so it is recorded even though nothing else moves.
			return finish(res.Judged != nil)
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
	for i := range st.Lanes {
		l := &st.Lanes[i]
		if l.Outcome != DrainLaneInProgress {
			continue
		}
		run, err := ReadState(repoRoot, l.RunID)
		if err != nil {
			return DrainResult{}, err
		}
		outcome, pr, hb := laneOutcome(run)
		switch outcome {
		case DrainLaneInProgress:
			res.Next = fmt.Sprintf("drive %s's lane, run %s: `abcd implement step --run %s` (one lane at a time); run `abcd drain` again once it is handed back or its pull request is open",
				l.Issue, l.RunID, l.RunID)
			return finish(true)
		case DrainLaneHandedBack:
			r, err := routeHandBack(repoRoot, l.Issue, "lane", *hb, now)
			if err != nil {
				return DrainResult{}, err
			}
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

	// The next eligible issue in the drain order, less any this drain has
	// taken or handed back and any this checkout already has a run for. Its
	// lane opens only once the host judgement over its remedy has answered no
	// for the remedy as it stands; until then the drain asks and opens nothing.
	runs, err := Runs(repoRoot)
	if err != nil {
		return DrainResult{}, err
	}
	var issues map[string]capture.Issue
	for _, v := range plan.Dispositions {
		if v.Outcome != capture.DrainEligible {
			continue
		}
		if slices.ContainsFunc(st.Lanes, func(l DrainLane) bool { return l.Issue == v.ID }) ||
			slices.ContainsFunc(st.HandBacks, func(r DrainRoute) bool { return r.Issue == v.ID }) {
			continue
		}
		if i := slices.IndexFunc(runs, func(r State) bool { return r.Key == v.ID }); i >= 0 {
			res.Passed = append(res.Passed, Excluded{ID: v.ID, Check: CheckRun,
				Reason: "this checkout already has run " + runs[i].RunID + " for it; drive or finish that run"})
			continue
		}
		if issues == nil {
			if issues, err = openIssues(repoRoot); err != nil {
				return DrainResult{}, err
			}
		}
		iss, ok := issues[v.ID]
		if !ok {
			continue
		}
		if j := judgementFor(st, v.ID, remedyDigest(iss.Remedy)); j == nil || j.Answer != JudgementNo {
			judging, err := requestJudgement(repoRoot, iss, v, st.Judging, now)
			if err != nil {
				return DrainResult{}, err
			}
			st.Judging = &judging
			res.Next = judgementMove(judging)
			return finish(true)
		}
		st.Judging = nil
		started, err := Start(repoRoot, v.ID, o)
		if err != nil {
			r, ok := AsRefusal(err)
			if !ok {
				return DrainResult{}, err
			}
			res.Passed = append(res.Passed, Excluded{ID: v.ID, Check: r.Check, Reason: r.Reason})
			continue
		}
		st.Lanes = append(st.Lanes, DrainLane{Issue: v.ID, RunID: started.RunID, OpenedAt: now, Outcome: DrainLaneInProgress})
		res.Start = &started
		res.Next = fmt.Sprintf("drive %s's lane, run %s: %s; run `abcd drain` again once it is handed back or its pull request is open",
			v.ID, started.RunID, started.Next)
		return finish(true)
	}
	st.Judging = nil
	st.Stopped, st.EndedAt = DrainStoppedEmpty, &now
	res.Next = fmt.Sprintf("nothing eligible is left: the drain opened %d lane(s) and ends here", len(st.Lanes))
	return finish(true)
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
	if run.Complete() {
		return DrainLaneDone, 0, nil
	}
	i := run.current()
	if i < 0 {
		return DrainLaneDone, 0, nil
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

// fieldFlags are the rule's hand-backs over the ledger: each open issue the
// rule hands back, flagged naming the rule that did. They are written nowhere
// (decision 8: the classification is re-derived every run).
func fieldFlags(plan capture.DrainPlan) []DrainRoute {
	out := []DrainRoute{}
	for _, v := range plan.Dispositions {
		if v.Outcome != capture.DrainHandBack {
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
func routeHandBack(repoRoot, issue, from string, hb HandBack, now time.Time) (DrainRoute, error) {
	r := DrainRoute{Issue: issue, From: from, Kind: hb.Kind, Reason: hb.Reason, Wrote: "nothing: a flag in this summary", At: &now}
	switch hb.Kind {
	case HandBackUserVisible:
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
	data, err := fsutil.ReadGuardedInRoot(root, DrainStateRel, maxDrainStateBytes)
	if errors.Is(err, os.ErrNotExist) {
		return DrainState{}, false, nil
	}
	if err != nil {
		return DrainState{}, false, refuse(StageDrain, "", "", fmt.Sprintf("%s cannot be read as the drain's state: %v", DrainStateRel, err),
			"the drain writes a regular file in a real directory; restore that, or remove "+DrainStateRel)
	}
	var st DrainState
	if err := jsonstrict.Decode(data, &st); err != nil || st.SchemaVersion != DrainSchemaVersion {
		why := fmt.Sprintf("schema_version %d (this abcd reads %d)", st.SchemaVersion, DrainSchemaVersion)
		if err != nil {
			why = err.Error()
		}
		return DrainState{}, false, refuse(StageDrain, "", "", DrainStateRel+" does not parse as the drain's state: "+why,
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
