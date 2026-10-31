package loop

// autoplan.go is `abcd build <itd-N> --auto-plan` (spc-2609202134338445 step
// 11; itd-2609201916151817 decision 6): the planning path run by the loop,
// with no human, on a draft whose decisions are all recorded. The substitute
// for the product thinker's sign-off is adr-2610310235000654's four parts:
// recorded decisions, grounds present and criteria provenance (the record-only
// rows of intent.AutoPlanChecks), and two review receipts, one per lens, each
// verdict parsed by the loop from the reviewer's own return. A start whose
// rows do not all pass is refused and writes nothing.
//
// The run has one lane, which takes two stages instead of the build's five:
//
//	plan-review  two fresh adversarial reviewers, one per lens, each handed
//	             the draft and writing a return; the loop records each
//	             verdict, and a review that does not pass hands the lane back
//	plan         the substitute checked again on the record as it stands and
//	             against the recorded verdicts, then `intent plan`
//
// Both stages are bound to the draft's bytes as the start checked them: the
// digest is recorded on the lane, and a draft that changed or left drafts/
// before the plan is handed back unplanned. The plan writes what a person's
// `abcd intent plan` writes, in the checkout the run lives in, and the run
// ends there: writing the spec's steps, committing the plan and building it
// are separate acts.
//
// The files of the lane live in its directory:
//
//	plan-review/<lens>/brief.md    the brief the loop renders for the reviewer
//	plan-review/<lens>/return.md   the reviewer's return

import (
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"time"

	"github.com/intentdriven/abcd/internal/core/intent"
	"github.com/intentdriven/abcd/internal/core/recordid"
	"github.com/intentdriven/abcd/internal/fsutil"
)

// The auto-plan lane's stages.
const (
	StagePlanReview Stage = "plan-review"
	StagePlan       Stage = "plan"
)

// The two lenses of the plan review, one fresh reviewer each (the
// adversarial-review principle's proven pair).
const (
	LensDesign = "design"
	LensRecord = "record"
)

// PlanReviewDirName is the lane's directory of plan reviews.
const PlanReviewDirName = "plan-review"

// HandBackNotPlanned is the verdict an auto-plan lane is handed back with: the
// substitute did not hold, so the draft is the person's to plan.
const HandBackNotPlanned = "not-planned"

// AutoPlan is an auto-plan lane's record: the draft as the start checked it,
// each review with the verdict the loop parsed from its return, and what the
// plan made.
type AutoPlan struct {
	// Draft is the draft's path when the run started, relative to the
	// checkout root, and Digest the "sha256:<hex>" of its bytes then.
	Draft  string `json:"draft"`
	Digest string `json:"digest"`
	// Reviews are the two reviews, the design lens first.
	Reviews []PlanReview `json:"reviews"`
	// Planned and Spec are the planned record's path and the spec plan
	// minted; empty until the plan stage ran.
	Planned string `json:"planned,omitempty"`
	Spec    string `json:"spec,omitempty"`
}

// PlanReview is one review of the draft: the lens it judges by, the agent it
// is started as, its brief and return, and the verdict the loop parsed.
type PlanReview struct {
	Lens    string `json:"lens"`
	Role    string `json:"role"`
	Brief   string `json:"brief"`
	Return  string `json:"return"`
	Verdict string `json:"verdict,omitempty"`
	Pass    bool   `json:"pass"`
}

// autoPlanning reports whether the run is an auto-plan run.
func (s State) autoPlanning() bool {
	for _, l := range s.Lanes {
		if l.AutoPlan != nil {
			return true
		}
	}
	return false
}

// autoPlanned reports the first lane, in a state, that carries `auto_plan`,
// which only version 17 and later write, or "".
func (s State) autoPlanned() string {
	for _, l := range s.Lanes {
		if l.AutoPlan != nil {
			return "`auto_plan` on " + l.ID
		}
	}
	return ""
}

// autoPlanCheck is the start's checks for an auto-plan of the draft key: the
// key, intent.AutoPlanChecks's rows and the peers. The result carries the
// lane's record when every row passes.
func autoPlanCheck(repoRoot string, res CheckResult, session string, snap *peerSnapshot) (CheckResult, error) {
	ap, err := intent.AutoPlanChecks(repoRoot, res.Key)
	if err != nil {
		res.Checks = append(res.Checks, CheckRow{Name: intent.AutoPlanCheckDraft, Detail: err.Error(),
			Remedy: "name a draft intent: `abcd intent` lists them"})
		return res, nil
	}
	res.Intent = ap.IntentID
	for _, r := range ap.Rows {
		res.Checks = append(res.Checks, CheckRow{Name: r.Name, OK: r.OK, Detail: r.Detail, Remedy: r.Remedy})
	}
	res.Checks = append(res.Checks, peersCheck(ap.IntentID, ap.Bucket, session, snap, true))
	res.OK = true
	for _, c := range res.Checks {
		if !c.OK {
			res.OK = false
		}
	}
	if res.OK {
		res.autoPlan = &AutoPlan{Draft: ap.Path, Digest: ap.Digest}
	}
	return res, nil
}

// digestRe is the shape of a stored draft digest.
var digestRe = regexp.MustCompile(`^sha256:[0-9a-f]{64}$`)

// autoPlansWritten reports whether every lane's stored auto-plan record is one
// the loop writes: the draft and the planned record under the intents
// folders, a digest of its shape, a spec id, the two lenses with the reviewer
// role, each brief and return in the lane's own directory, and each verdict
// one the role states. It reaches `implement status --json` and `implement
// record --json` as stored, so a hand edit is refused, never echoed.
func (st State) autoPlansWritten() bool {
	for _, l := range st.Lanes {
		ap := l.AutoPlan
		if ap == nil {
			continue
		}
		draftDir := intent.IntentsRelDir + "/" + intent.BucketDrafts + "/"
		if !fsutil.ValidRelPath(ap.Draft) || !strings.HasPrefix(ap.Draft, draftDir) || !digestRe.MatchString(ap.Digest) {
			return false
		}
		if ap.Planned != "" && ap.Planned != intent.PlannedPathOf(ap.Draft) {
			return false
		}
		if ap.Spec != "" && !recordid.ValidSpecID(ap.Spec) {
			return false
		}
		dir, err := laneRel(st.RunID, l.ID, StagePlanReview)
		if err != nil || len(ap.Reviews) != 2 {
			return false
		}
		for i, lens := range []string{LensDesign, LensRecord} {
			r := ap.Reviews[i]
			d := dir + "/" + PlanReviewDirName + "/" + lens
			if r.Lens != lens || r.Role != RoleRuthless || r.Brief != d+"/"+BriefFileName || r.Return != d+"/"+ReturnFileName {
				return false
			}
			if r.Verdict == "" {
				if r.Pass {
					return false
				}
				continue
			}
			k := slices.IndexFunc(reviewerVerdicts[r.Role], func(w verdictWord) bool { return w.word == r.Verdict })
			if k < 0 || reviewerVerdicts[r.Role][k].pass != r.Pass {
				return false
			}
		}
	}
	return true
}

// autoPlans reports whether a start plans its key: --auto-plan was given
// and the key is an intent that is not planned (an unknown one too, so the
// auto-plan's own row names it).
func autoPlans(repoRoot, key string, o Options) (bool, error) {
	if !o.AutoPlan || !recordid.ValidIntentID(key) {
		return false, nil
	}
	corpus, err := intent.Load(repoRoot)
	if err != nil {
		return false, err
	}
	it, ok := corpus.Lookup(key)
	return !ok || it.Bucket != intent.BucketPlanned, nil
}

// autoPlanFootprint is what an auto-plan run reserves its key with: the
// draft and the path plan moves it to, the two files of the record it writes.
func autoPlanFootprint(ap AutoPlan) []string {
	return []string{ap.Draft, intent.PlannedPathOf(ap.Draft)}
}

// openAutoPlanLane opens the auto-plan run's one lane, state-only, with its two
// reviews laid out.
func openAutoPlanLane(st *State, ap AutoPlan) error {
	id := "lane-1"
	dir, err := laneRel(st.RunID, id, StagePlanReview)
	if err != nil {
		return err
	}
	for _, lens := range []string{LensDesign, LensRecord} {
		d := dir + "/" + PlanReviewDirName + "/" + lens
		ap.Reviews = append(ap.Reviews, PlanReview{Lens: lens, Role: RoleRuthless, Brief: d + "/" + BriefFileName, Return: d + "/" + ReturnFileName})
	}
	st.Lanes = append(st.Lanes, Lane{ID: id, Key: st.Key, StepTitle: "plan " + st.Intent + " (--auto-plan)", Stage: StagePlanReview, AutoPlan: &ap})
	return nil
}

// draftHandBack hands an auto-plan lane back when the draft is not the one the
// start checked; nil when it is.
func draftHandBack(c Context, ap AutoPlan) (*HandBack, error) {
	data, err := fsutil.ReadGuarded(filepath.Join(c.RepoRoot, filepath.FromSlash(ap.Draft)), maxIntentBytes)
	switch {
	case errors.Is(err, fs.ErrNotExist):
		return &HandBack{Verdict: HandBackNotPlanned, Reason: "the draft " + ap.Draft + " is no longer in drafts/, so what the reviews read is not what would be planned"}, nil
	case err != nil:
		return nil, fmt.Errorf("reading the draft %s: %w", ap.Draft, err)
	}
	if got := intent.DraftDigest(data); got != ap.Digest {
		return &HandBack{Verdict: HandBackNotPlanned, Reason: "the draft " + ap.Draft + " changed after the run checked it (" + shortDigest(ap.Digest) + ", now " + shortDigest(got) + "), so what the reviews read is not what would be planned"}, nil
	}
	return nil, nil
}

// shortDigest is a digest's first twelve hex digits, for a sentence.
func shortDigest(d string) string {
	h := strings.TrimPrefix(d, "sha256:")
	if len(h) > 12 {
		h = h[:12]
	}
	return h
}

// planReviewStage is the plan-review stage's body: it hands the draft to the
// first review with no verdict that is not already out, and, once both have
// returned, completes the stage when both passed or hands the lane back with
// the findings when one did not.
func planReviewStage(c Context, lane *Lane) (Outcome, error) {
	ap := lane.AutoPlan
	if ap == nil {
		return Outcome{}, refuse(string(StagePlanReview), "", lane.ID, "the lane carries no auto-plan record", "the loop writes it at the start; restore the run's state file")
	}
	if hb, err := draftHandBack(c, *ap); err != nil || hb != nil {
		return Outcome{HandBack: hb}, err
	}
	for i := range ap.Reviews {
		r := ap.Reviews[i]
		if r.Verdict != "" || lane.awaited(r.Return) {
			continue
		}
		if err := writePlanReviewBrief(c, *lane, r); err != nil {
			return Outcome{}, err
		}
		return Outcome{Await: &Await{Role: r.Role, Brief: r.Brief, Receipt: r.Return},
			Note: fmt.Sprintf("%s handed to a fresh %s for the %s review; awaiting its return at %s", ap.Draft, r.Role, lensName(r.Lens), r.Return)}, nil
	}
	var failing []string
	for _, r := range ap.Reviews {
		if r.Verdict == "" {
			return Outcome{}, contend(string(StagePlanReview), "", lane.ID, "the plan reviews are out",
				"hand back their returns with `abcd implement receipt <path>`")
		}
		if !r.Pass {
			failing = append(failing, r.Return)
		}
	}
	if len(failing) > 0 {
		return Outcome{HandBack: &HandBack{Verdict: HandBackNotPlanned, Verdicts: planVerdicts(*ap), Findings: failing,
			Reason: "a plan review did not pass, and an unattended plan has nobody to answer its findings"}}, nil
	}
	return Outcome{Note: "both plan reviews passed (" + planVerdicts(*ap) + "); the plan checks the substitute and plans"}, nil
}

// lensName is a lens for a sentence.
func lensName(lens string) string {
	if lens == LensDesign {
		return "design and feasibility"
	}
	return "record-discipline"
}

// planVerdicts renders the reviews' verdicts as the loop recorded them.
func planVerdicts(ap AutoPlan) string {
	parts := make([]string, 0, len(ap.Reviews))
	for _, r := range ap.Reviews {
		v := r.Verdict
		if v == "" {
			v = "none yet"
		}
		parts = append(parts, r.Lens+" "+v)
	}
	return strings.Join(parts, ", ")
}

// writePlanReviewBrief renders one reviewer's brief.
func writePlanReviewBrief(c Context, lane Lane, r PlanReview) error {
	var b bytes.Buffer
	p := func(format string, a ...any) { fmt.Fprintf(&b, format, a...) }
	ap := lane.AutoPlan
	st := c.State
	p("# Plan review brief: the %s lens, %s of %s\n\n", lensName(r.Lens), lane.ID, st.RunID)
	p("You are a fresh %s agent (agents/%s.md), reviewing a draft intent, not a diff. The run\n", r.Role, r.Role)
	p("`abcd build %s --auto-plan` plans this draft with no person present, so your review and one other,\n", st.Intent)
	p("by the other lens, stand in for the adversarial reviews before a planning interview\n")
	p("(adr-2610310235000654). The precondition about a gated build does not apply: there is no code.\n")
	p("No one will answer a question about this brief; what it does not settle, say so in your return.\n\n")
	p("- the draft: `%s`\n", abs(c.RepoRoot, ap.Draft))
	p("- its bytes as the run checked them: %s\n\n", ap.Digest)
	p("Read only; change nothing. A change to the draft hands the run back unplanned.\n\n")
	p("## The lens\n\n")
	if r.Lens == LensDesign {
		p("Design and feasibility: can what the press release promises be built as the record says, within\n")
		p("its scope conditions? Does each acceptance criterion state something a test can decide? Does a\n")
		p("decision contradict another, the brief's invariants or a principle under .abcd/development/principles/?\n")
	} else {
		p("Record discipline: is every decision recorded with its grounds, and none left open in prose? Are the\n")
		p("criteria the record's own, each Given-When-Then, and does every scope condition bound the claim?\n")
		p("Do its typed links, its `blocked_by` and its cited records exist and say what the draft says they do?\n")
	}
	p("\nA finding that would change what is built, or that a person must decide, is a FIX FIRST: an\n")
	p("unattended plan has nobody to answer it, so it hands the draft back to the person.\n\n")
	p("## What you hand back\n\n")
	p("Write your whole return to:\n\n`%s`\n\n", abs(c.RepoRoot, r.Return))
	words := make([]string, 0, len(reviewerVerdicts[r.Role]))
	for _, w := range reviewerVerdicts[r.Role] {
		words = append(words, w.word)
	}
	p("It ends with one `### Verdict` section stating exactly one of: %s. The loop reads that\n", strings.Join(words, ", "))
	p("section and records the verdict itself; a verdict written anywhere else is nobody's.\n")
	return writeRoundFile(c.RepoRoot, r.Brief, b.Bytes())
}

// verifyPlanReview takes back one review's return: the loop parses the verdict
// from it and records it on the lane. A return it cannot read as a verdict
// records nothing and is refused.
func verifyPlanReview(c Context, lane *Lane, receiptRel string) error {
	if lane.AutoPlan == nil {
		return refuse("receipt", "", lane.ID, "the lane carries no auto-plan record", "restore the run's state file")
	}
	ap := *lane.AutoPlan
	ap.Reviews = slices.Clone(ap.Reviews)
	k := slices.IndexFunc(ap.Reviews, func(r PlanReview) bool { return samePath(c.RepoRoot, r.Return, receiptRel) })
	if k < 0 {
		return refuse("receipt", "", lane.ID, receiptRel+" is not a plan review's return", "hand back the return a plan review's brief names")
	}
	raw, err := readReturn(c.RepoRoot, ap.Reviews[k].Return, lane.ID)
	if err != nil {
		return err
	}
	got, err := parseVerdict(string(raw), reviewerVerdicts[ap.Reviews[k].Role])
	if err != nil {
		return refuse("receipt", "", lane.ID, "the return at "+ap.Reviews[k].Return+" states no verdict the loop can read: "+err.Error(),
			"end the return with one `### Verdict` section whose first line states the verdict, then hand it back")
	}
	ap.Reviews[k].Verdict, ap.Reviews[k].Pass = got.word, got.pass
	lane.AutoPlan = &ap
	return nil
}

// planStage is the plan stage's body: the substitute checked again, on the
// record as it stands and against the verdicts the loop recorded, then the
// plan. A record a killed call already planned is recorded as planned.
func planStage(c Context, lane *Lane) (Outcome, error) {
	if lane.AutoPlan == nil {
		return Outcome{}, refuse(string(StagePlan), "", lane.ID, "the lane carries no auto-plan record", "the loop writes it at the start; restore the run's state file")
	}
	ap := *lane.AutoPlan
	id := c.State.Intent
	if done, ok, err := plannedAlready(c.RepoRoot, id, ap); err != nil || ok {
		if ok {
			lane.AutoPlan = &done
			return Outcome{Note: planNote(c, done, "was found planned, as an earlier call left it")}, nil
		}
		return Outcome{}, err
	}
	if hb, err := draftHandBack(c, ap); err != nil || hb != nil {
		return Outcome{HandBack: hb}, err
	}
	for _, r := range ap.Reviews {
		if r.Verdict == "" || !r.Pass {
			return Outcome{HandBack: &HandBack{Verdict: HandBackNotPlanned, Verdicts: planVerdicts(ap),
				Reason: "the plan needs both reviews' passing verdicts, as the loop recorded them"}}, nil
		}
	}
	chk, err := intent.AutoPlanChecks(c.RepoRoot, id)
	if err != nil {
		return Outcome{}, err
	}
	if chk.Digest != ap.Digest || chk.Path != ap.Draft {
		return Outcome{HandBack: &HandBack{Verdict: HandBackNotPlanned, Reason: "the draft changed after the run checked it, so what the reviews read is not what would be planned"}}, nil
	}
	if !chk.OK() {
		var why []string
		for _, r := range chk.Rows {
			if !r.OK {
				why = append(why, r.Name+": "+r.Detail)
			}
		}
		return Outcome{HandBack: &HandBack{Verdict: HandBackNotPlanned, Reason: "the substitute no longer holds: " + strings.Join(why, "; ")}}, nil
	}
	res, err := intent.Plan(c.RepoRoot, id, intent.PlanOptions{})
	if err != nil {
		return Outcome{}, refuse(string(StagePlan), "", lane.ID, "plan refused "+id+": "+fsutil.RedactHome(err.Error()),
			"settle what it names; the draft is unplanned and the lane resumes here")
	}
	ap.Planned, ap.Spec = res.Intent.Path, res.Spec.ID
	lane.AutoPlan = &ap
	return Outcome{Note: planNote(c, ap, "is planned")}, nil
}

// plannedAlready reports the auto-plan record completed when the intent is
// already in planned/ at the path plan moves the draft to, linked to a spec:
// the plan a call killed before its state write made.
func plannedAlready(repoRoot, id string, ap AutoPlan) (AutoPlan, bool, error) {
	corpus, err := intent.Load(repoRoot)
	if err != nil {
		return ap, false, err
	}
	it, ok := corpus.Lookup(id)
	if !ok || it.Bucket != intent.BucketPlanned || it.Path != intent.PlannedPathOf(ap.Draft) || it.SpecID == "" || it.SpecID == "null" {
		return ap, false, nil
	}
	ap.Planned, ap.Spec = it.Path, it.SpecID
	return ap, true, nil
}

// planNote is the run record's line for the plan.
func planNote(c Context, ap AutoPlan, what string) string {
	return fmt.Sprintf("%s %s at %s, with %s minted, on the substitute adr-2610310235000654 records (%s)",
		c.State.Intent, what, ap.Planned, ap.Spec, planVerdicts(ap))
}

// PlanStageNext reports whether the run's next move may be an auto-plan
// lane's plan, which writes the intent and spec records in this checkout as
// `abcd intent plan` does: the front door then holds the step to the record
// writers' shared-checkout rule first. A run it cannot read reports false,
// and the step names why.
func PlanStageNext(repoRoot, runID string) bool {
	st, err := ReadState(repoRoot, runID)
	if err != nil {
		return false
	}
	for _, l := range st.Lanes {
		if l.AutoPlan != nil && l.Stage == StagePlan {
			return true
		}
	}
	return false
}

// autoPlanDoneMove is what a caller is told once an auto-plan run is complete.
func autoPlanDoneMove(st State) string {
	for _, l := range st.Lanes {
		if l.AutoPlan != nil && l.AutoPlan.Spec != "" {
			return fmt.Sprintf("nothing more in %s: %s is planned with %s; write the spec's steps and footprint, commit the plan on a branch for review, then `abcd build %s` once it is on the default branch",
				st.RunID, st.Intent, l.AutoPlan.Spec, st.Intent)
		}
	}
	return "nothing: every lane of " + st.RunID + " is done"
}

// releaseAutoPlan releases an auto-plan run's reservation once its lane is
// done or handed back: the run builds nothing, so nothing else will.
func releaseAutoPlan(repoRoot string, st *State, now time.Time) {
	if st.ReservedAt == nil || !st.autoPlanning() {
		return
	}
	clause, failed := releaseReservation(repoRoot, st.Key, "", now)
	stage := StageReservationReleased
	if failed {
		clause += ", so it lapses with its lease"
		stage = StageReservationReleaseFailed
	}
	st.Record = append(st.Record, Entry{At: now, Stage: stage, Note: clause})
	st.UpdatedAt = now
}
