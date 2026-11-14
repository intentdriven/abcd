package loop

// reviewers.go keys a validation round by reviewer (itd-2610240922461281,
// spc-2610240951109420 step 3). The run's review set (reviewset.go) names,
// for each of the two code-review roles, one or two reviewers at model
// classes; a round's unit is then (role, reviewer index), so two reviewers of
// one role never collide:
//
//   - each reviewer has its own directory in the round: index 0 keeps the
//     role's directory (validate/round-<n>/<role>/), the one a round written
//     before reviewers were keyed used for its single reviewer, and index i
//     takes validate/round-<n>/<role>.<i>/; its brief, return and receipt
//     match live there, and the verifier matches a return to its reviewer by
//     that path;
//   - the two briefs differ only in their return path: the class and the
//     model a reviewer is asked for travel in its await, the `awaiting`
//     payload a host reads, and as the request's model a runner is handed;
//   - the reviewers are handed out in consecutive steps, each taking a slot
//     of the run's sub-agent ceiling, on the round's one head.
//
// Once every reviewer of a role has returned, the round records how they
// compared: they disagree exactly when one verdict passes and another fails,
// by the pass bit of each verdict word; two failing verdicts (BLOCK and
// NEEDS-INPUT) agree. The round's verdict does not hang on the comparison:
// it fails when any reviewer fails it, as any failing validator always has,
// which is the stricter verdict. A role that ran one reviewer records that no
// comparison was made.
//
// A reviewer is asked for its class's effort as it is for the model: a runner
// that can pass an effort is handed it, and its route records it as asked; a
// host-run reviewer is asked for it and never confirms it.
//
// A host-run reviewer is asked for its class and model and never confirms it,
// so its record says the model was asked (ModelAsked); one a runner ran and
// that reported its model says so (ModelReported), with the reported model on
// its route beside the one asked for. One the reviewer route (oracle.review's
// RP route) answered was never asked for its class's model at all, since that
// route's request carries none, and its record says so (ModelUnsent).
//
// A model other than the writer's reviews every change (itd-2610301750162657,
// spc-2610301925429250 step 4). Before a reviewer of the two code-review roles
// is handed out, on the host or to a runner, the loop compares the model the
// lane's implementer reported on the receipt the round judges with the
// model the reviewer's class resolves to, by matrix.OtherModel, which counts
// every doubtful pair (an empty or unreadable name, a vendor prefix, a version
// or date stamp) as the same model. Where the class holds the writer's model
// the reviewer takes the next class, in the order Classes gives and wrapping
// round, whose model in the class map the run started on differs; a run
// started before that map was stored chooses among the classes its set
// recorded. When no class differs, the lane is handed back as stuck with the
// reason noOtherModel. The reviewer records both models, and the choice is
// made once: a resumed run reads it back and never decides it again. A
// reviewer's return names the model it ran on (reviewerModelLine) unless its
// runner reported one, and a return naming none, or naming the writer's, is
// refused as the writer's own review.

import (
	"fmt"
	"slices"
	"strings"
	"unicode"

	"github.com/intentdriven/abcd/internal/core/layered"
	"github.com/intentdriven/abcd/internal/core/matrix"
	"github.com/intentdriven/abcd/internal/core/oracle"
	"github.com/intentdriven/abcd/internal/core/runner"
	"github.com/intentdriven/abcd/internal/termsafe"
)

// noOtherModel is the reason a lane is handed back with when no class the run
// may choose resolves to a model other than the writer's.
const noOtherModel = "no reviewer model other than the writer's"

// ReviewerModelPrefix opens the line a reviewer's return names the model it
// ran on with.
const ReviewerModelPrefix = "Reviewer model:"

// reviewerModelLine is what a reviewer's brief says of the model line.
const reviewerModelLine = "Name the model you run on in a line of its own, `" + ReviewerModelPrefix + " <model>`, straight after that first line. " +
	"A return that names none, or names the model that wrote the lane, is refused as the writer's own review: a model other than the writer's reviews every change."

// maxStoredModel bounds a writer's or a reviewer's model the state keeps.
const maxStoredModel = 200

// modelWritten reports whether s is a model the state keeps as a writer's or
// a reviewer's: one plain token, bounded, with no space, control, bidi or
// zero-width rune.
func modelWritten(s string) bool {
	if s == "" || len(s) > maxStoredModel || !plainWritten(s) {
		return false
	}
	return !strings.ContainsFunc(s, func(r rune) bool { return unicode.IsSpace(r) || unicode.IsControl(r) })
}

// shownModel is a writer's or a reviewer's model as a receipt line, a pull
// request's body or a terminal shows it: the name itself when it is one plain
// token of a model's runes, so nothing in it can start a heading, a trailer
// or markup; else a description that does not repeat it.
func shownModel(s string) string {
	if modelWritten(s) && len(s) <= maxNotedModel && !strings.ContainsFunc(s, func(r rune) bool {
		return !(r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || strings.ContainsRune("._:/@+-", r))
	}) {
		return s
	}
	if s == "" {
		return "none named"
	}
	return termsafe.DescribeRefused(s)
}

// laneWriterModel is the model the lane's implementer reported on the latest
// receipt the loop verified for it, the one whose head a round judges: the
// record's model, else its route's; "" when it reported none.
func laneWriterModel(l Lane) string {
	for i := len(l.Receipts) - 1; i >= 0; i-- {
		r := l.Receipts[i]
		if r.Role != RoleImplementer {
			continue
		}
		if strings.TrimSpace(r.Model) != "" {
			return r.Model
		}
		if r.Route != nil {
			return r.Route.Model
		}
		return ""
	}
	return ""
}

// choosable are the classes a reviewer may be moved to, in the order Classes
// gives: the class map the run started on, else, for a run started before it
// was stored, each class the set recorded (its reviewers' and its judge's).
func (rs ReviewSet) choosable() []ReviewClass {
	if len(rs.Classes) > 0 {
		return rs.Classes
	}
	seen := map[string]ReviewClass{}
	for _, r := range rs.Roles {
		for _, c := range r.Reviewers {
			if _, ok := seen[c.Class]; !ok {
				seen[c.Class] = c
			}
		}
	}
	if j := rs.Judge; j != nil {
		if _, ok := seen[j.Class]; !ok {
			seen[j.Class] = ReviewClass{Class: j.Class, Model: j.Model, MapLayer: j.MapLayer, MapOrigin: j.MapOrigin,
				Effort: j.Effort, EffortLayer: j.EffortLayer, EffortOrigin: j.EffortOrigin}
		}
	}
	var out []ReviewClass
	for _, c := range Classes {
		if rc, ok := seen[c]; ok {
			out = append(out, rc)
		}
	}
	return out
}

// otherClass is the class a reviewer the set asked for at entry is
// dispatched on when writer wrote the lane: entry itself when its model is
// other than the writer's, else the next class after it, in Classes order and
// wrapping round, among those rs may choose, whose model is; false when none
// is.
func otherClass(rs ReviewSet, entry ReviewClass, writer string) (ReviewClass, bool) {
	if matrix.OtherModel(writer, entry.Model) {
		return entry, true
	}
	from := slices.Index(Classes, entry.Class)
	choosable := rs.choosable()
	for k := 1; k < len(Classes); k++ {
		next := Classes[(from+k+len(Classes))%len(Classes)]
		i := slices.IndexFunc(choosable, func(c ReviewClass) bool { return c.Class == next })
		if i >= 0 && matrix.OtherModel(writer, choosable[i].Model) {
			return choosable[i], true
		}
	}
	return ReviewClass{}, false
}

// chooseReviewerModel settles, once, the model a reviewer of the code-review
// roles is dispatched on, before it is handed out: it records the writer's
// model and, for a reviewer the set asked a class of, moves it to another
// class when its own holds the writer's model, recording the model it is
// dispatched on. It returns the hand-back when no class the run may choose
// resolves to another model, or the writer's model cannot be read. A reviewer
// that already records the writer's model was settled before and is left as it
// is, so a resumed run never decides it again.
func chooseReviewerModel(st State, l Lane, v *ValidatorRun) *HandBack {
	if !slices.Contains(reviewRoles, v.Role) || v.WriterModel != "" {
		return nil
	}
	writer := laneWriterModel(l)
	if !modelWritten(writer) {
		return noOtherModelHandBack(st, l, v.Role, fmt.Sprintf("the lane's implementer reported %s, which cannot be told from any reviewer's", shownModel(writer)))
	}
	if v.Class == "" || st.Review == nil {
		v.WriterModel = writer
		return nil
	}
	got, ok := otherClass(*st.Review, ReviewClass{Class: v.Class, Model: v.Model, Effort: v.Effort}, writer)
	if !ok {
		return noOtherModelHandBack(st, l, v.Role, fmt.Sprintf("the writer's model is %s, and the %s's class %s and every other class the run may choose resolve to it",
			shownModel(writer), v.Role, classPhrase(v.Class, shownModel(v.Model), v.Effort)))
	}
	v.Class, v.Model, v.Effort = got.Class, got.Model, got.Effort
	v.WriterModel, v.ReviewerModel = writer, got.Model
	return nil
}

// noOtherModelHandBack is the stuck hand-back of a lane no reviewer other
// than the writer's model can judge.
func noOtherModelHandBack(st State, l Lane, role, why string) *HandBack {
	return &HandBack{Kind: HandBackStuck, Reason: noOtherModel + ": " + why,
		Home: fmt.Sprintf("a person: give a class of models.classes in %s a model other than the writer's, discard the lane with "+
			"`abcd implement step --run %s --discard %s`, and build %s again; the run keeps the class map it started on, so the %s is never "+
			"moved to a model the run did not record", layered.Config.MachineOrigin(), st.RunID, l.ID, keyOf(st), role)}
}

// namedReviewerModel is the model a reviewer's return names on its model
// line, the first line opening with ReviewerModelPrefix among the return's
// first lines; "" when it names none.
func namedReviewerModel(text string) string {
	lines := strings.SplitN(text, "\n", 8)
	for _, l := range lines[:min(len(lines), 7)] {
		if rest, ok := strings.CutPrefix(strings.TrimRight(l, "\r"), ReviewerModelPrefix); ok {
			return strings.TrimSpace(rest)
		}
	}
	return ""
}

// modelled reports the first thing, in a state, that only version 27 and
// later write: a reviewer's writer's or reviewer's model, or the review set's
// class map; "" when there is none.
func (s State) modelled() string {
	if s.Review != nil && len(s.Review.Classes) > 0 {
		return "the review set's class map"
	}
	for _, l := range s.Lanes {
		for _, r := range l.Validation {
			for i, v := range r.Validators {
				if v.WriterModel != "" || v.ReviewerModel != "" {
					return fmt.Sprintf("a writer's or a reviewer's model on %s's round %d validator %d", l.ID, r.Round, i+1)
				}
			}
		}
	}
	return ""
}

// roundReviewers are the validators a round hands role to, in reviewer
// order: one per class the run's review set names for a reviewer role, and
// one with no class for the intent-auditor and for a run started before
// review sets, which runs one reviewer per role as it always has.
func roundReviewers(st State, role string) []ValidatorRun {
	if st.Review != nil && slices.Contains(reviewRoles, role) {
		if r, ok := st.Review.Role(role); ok && len(r.Reviewers) > 0 {
			out := make([]ValidatorRun, 0, len(r.Reviewers))
			for i, c := range r.Reviewers {
				out = append(out, ValidatorRun{Role: role, Reviewer: i, Class: c.Class, Model: c.Model, Effort: c.Effort})
			}
			return out
		}
	}
	return []ValidatorRun{{Role: role}}
}

// reviewerDir is the directory of reviewer index of role in round n,
// relative to the checkout root: the role's own directory for index 0, and
// <role>.<index> beside it for every later one.
func reviewerDir(runID, laneID string, n int, role string, index int) (string, error) {
	agent := role
	if index > 0 {
		agent = fmt.Sprintf("%s.%d", role, index)
	}
	return roundDir(runID, laneID, n, agent)
}

// await is what the round hands the validator out as: its role, brief and
// return, and, for a review set's reviewer, its index, class, model and the
// class's effort.
func (v ValidatorRun) await() Await {
	a := Await{Role: v.Role, Brief: v.Brief, Receipt: v.Return}
	if v.Class != "" {
		i := v.Reviewer
		a.Reviewer, a.Class, a.Model, a.Effort = &i, v.Class, v.Model, v.Effort
	}
	return a
}

// reviewerPhrase names a review set's reviewer as the record and the text
// surfaces show it, its class with the model the class holds (decision 2);
// "" for a validator no review set asked for a class.
func reviewerPhrase(index int, class, model, effort string) string {
	if class == "" {
		return ""
	}
	return fmt.Sprintf("reviewer %d at %s", index, classPhrase(class, model, effort))
}

// phrase is reviewerPhrase for the validator.
func (v ValidatorRun) phrase() string { return reviewerPhrase(v.Reviewer, v.Class, v.Model, v.Effort) }

// recordModelStatus records, on a review set's reviewer whose verdict the
// loop has just recorded, how far its model is known: unsent when the
// reviewer route answered it (its route record names that route as the one
// that ran), since that route's request carries no model and the class's
// model was never asked of it; reported when the runner that ran it reported
// a model; else asked, as for every host-run reviewer.
func (v *ValidatorRun) recordModelStatus(reported string) {
	if v.Class == "" {
		return
	}
	switch {
	case answeredByReviewRoute(v.Route):
		v.ModelStatus = ModelUnsent
	case reported != "":
		v.ModelStatus = ModelReported
	default:
		v.ModelStatus = ModelAsked
	}
}

// answeredByReviewRoute reports whether a validator's route record names the
// reviewer route (oracle.review's RP route) as the one that ran it.
func answeredByReviewRoute(r *runner.RouteRecord) bool {
	return r != nil && r.Ran == oracle.ReviewRP
}

// modelStatusPhrase is the record's words for a reviewer's model status.
func modelStatusPhrase(status string) string {
	switch status {
	case ModelReported:
		return "the model its runner reported recorded beside it"
	case ModelUnsent:
		return "the reviewer route answered it, and that route is sent no model, so the class's model was never asked for"
	}
	return "the model asked for, not confirmed"
}

// compareRole records, on round r, how role's reviewers compared, once every
// one of them has a verdict; it does nothing for a role whose reviewers no
// review set asked for a class, or while one is still out. It returns the
// comparison it recorded.
func compareRole(r *ValidationRound, role string) (Comparison, bool) {
	var vs []ValidatorRun
	for _, v := range r.Validators {
		if v.Role == role {
			vs = append(vs, v)
		}
	}
	if len(vs) == 0 || vs[0].Class == "" {
		return Comparison{}, false
	}
	cmp := Comparison{Role: role, Reviewers: len(vs), Outcome: ComparisonNone}
	passed, failed := false, false
	for _, v := range vs {
		if v.Verdict == "" {
			return Comparison{}, false
		}
		cmp.Verdicts = append(cmp.Verdicts, v.Verdict)
		if v.Pass {
			passed = true
		} else {
			failed = true
		}
	}
	if len(vs) > 1 {
		cmp.Outcome = ComparisonAgree
		if passed && failed {
			cmp.Outcome = ComparisonDisagree
		}
	}
	cmps := slices.DeleteFunc(slices.Clone(r.Comparisons), func(c Comparison) bool { return c.Role == role })
	r.Comparisons = append(cmps, cmp)
	return cmp, true
}

// comparisonNote is the record's words for a comparison.
func comparisonNote(c Comparison) string {
	switch c.Outcome {
	case ComparisonNone:
		return fmt.Sprintf("the %s ran one reviewer, so no comparison was made", c.Role)
	case ComparisonDisagree:
		return fmt.Sprintf("the %s's %d reviewers disagree (%s): one passes and another fails, and the stricter verdict decides, so the round does not pass",
			c.Role, c.Reviewers, strings.Join(c.Verdicts, " against "))
	}
	return fmt.Sprintf("the %s's %d reviewers agree (%s)", c.Role, c.Reviewers, strings.Join(c.Verdicts, " and "))
}

// keyed reports the first thing, in a state, that only version 20 and later
// write: a reviewer's index, class, model or model status, an await's
// reviewer, or a round's comparisons; "" when there is none.
func (s State) keyed() string {
	for _, l := range s.Lanes {
		for _, a := range l.Awaits {
			if a.Reviewer != nil || a.Class != "" || a.Model != "" {
				return "a reviewer on " + l.ID + "'s await of the " + a.Role
			}
		}
		for _, r := range l.Validation {
			if len(r.Comparisons) > 0 {
				return fmt.Sprintf("`comparisons` on %s's round %d", l.ID, r.Round)
			}
			for i, v := range r.Validators {
				if v.Reviewer != 0 || v.Class != "" || v.Model != "" || v.ModelStatus != "" {
					return fmt.Sprintf("a reviewer on %s's round %d validator %d", l.ID, r.Round, i+1)
				}
			}
		}
	}
	return ""
}

// reviewersWritten reports whether every reviewer a state carries is one the
// loop writes: on one of the review set's roles, at an index among the
// reviewers the run's stored review set names for that role, with exactly
// the class and model the set names at that index (the set is fixed when the
// run starts and every round's reviewers are cut from it, roundReviewers); a
// model status the verifier writes, unsent only where the reviewer route
// answered; an await's reviewer likewise; and each comparison one of the
// outcomes over one or two reviewers. A validator with no class carries no
// index and no model.
func (s State) reviewersWritten() bool {
	reviewer := func(role string, index int, class, model, effort string) bool {
		if class == "" {
			return index == 0 && model == "" && effort == ""
		}
		if s.Review == nil || !slices.Contains(reviewRoles, role) || index < 0 || index >= maxReviewers {
			return false
		}
		set, ok := s.Review.Role(role)
		if !ok || index >= len(set.Reviewers) {
			return false
		}
		want := set.Reviewers[index]
		return class == want.Class && model == want.Model && effort == want.Effort &&
			slices.Contains(Classes, class) && validRoute(model)
	}
	// A reviewer that records the writer's model was settled by
	// chooseReviewerModel: it holds the set's class, or the one otherClass
	// takes instead, recomputed here from the stored map and the recorded
	// writer's model; anything else is refused, the comparison never loosened.
	validator := func(v ValidatorRun) bool {
		if v.WriterModel == "" {
			return v.ReviewerModel == "" && reviewer(v.Role, v.Reviewer, v.Class, v.Model, v.Effort)
		}
		if !slices.Contains(reviewRoles, v.Role) || !modelWritten(v.WriterModel) || (v.ReviewerModel != "" && !modelWritten(v.ReviewerModel)) {
			return false
		}
		if v.Class == "" {
			// A reviewer no class was asked of records the model its return
			// named once its verdict is recorded, and only one other than the
			// writer's.
			return reviewer(v.Role, v.Reviewer, "", v.Model, v.Effort) &&
				(v.ReviewerModel == "" && v.Verdict == "" || v.ReviewerModel != "" && matrix.OtherModel(v.WriterModel, v.ReviewerModel))
		}
		if s.Review == nil || v.ReviewerModel != v.Model || v.Reviewer < 0 || v.Reviewer >= maxReviewers {
			return false
		}
		set, ok := s.Review.Role(v.Role)
		if !ok || v.Reviewer >= len(set.Reviewers) {
			return false
		}
		want, ok := otherClass(*s.Review, set.Reviewers[v.Reviewer], v.WriterModel)
		return ok && v.Class == want.Class && v.Model == want.Model && v.Effort == want.Effort
	}
	for _, l := range s.Lanes {
		for _, a := range l.Awaits {
			if a.Reviewer == nil {
				if a.Class != "" || a.Model != "" || a.Effort != "" {
					return false
				}
				continue
			}
			if a.Class == "" {
				return false
			}
			// An await of a moved reviewer carries the class its validator
			// was dispatched on, which that validator's own check holds.
			moved := false
			for _, r := range l.Validation {
				for _, v := range r.Validators {
					if v.Return == a.Receipt && v.WriterModel != "" && v.Role == a.Role && v.Reviewer == *a.Reviewer &&
						v.Class == a.Class && v.Model == a.Model && v.Effort == a.Effort {
						moved = true
					}
				}
			}
			if !moved && !reviewer(a.Role, *a.Reviewer, a.Class, a.Model, a.Effort) {
				return false
			}
		}
		for _, r := range l.Validation {
			for _, v := range r.Validators {
				if !validator(v) {
					return false
				}
				switch v.ModelStatus {
				case "":
				case ModelAsked, ModelReported:
					if v.Class == "" || answeredByReviewRoute(v.Route) {
						return false
					}
				case ModelUnsent:
					if v.Class == "" || !answeredByReviewRoute(v.Route) {
						return false
					}
				default:
					return false
				}
			}
			for _, c := range r.Comparisons {
				if !slices.Contains(reviewRoles, c.Role) || c.Reviewers < 1 || c.Reviewers > maxReviewers || len(c.Verdicts) != c.Reviewers ||
					!slices.Contains([]string{ComparisonAgree, ComparisonDisagree, ComparisonNone}, c.Outcome) {
					return false
				}
			}
		}
	}
	return true
}

// recordReviewerModel checks, as a code-review role's verdict is about to be
// recorded, that the model that ran the reviewer is other than the writer's:
// the model its runner reported, else the one its return names on its model
// line. A return naming none counts as the writer's, and is refused as one
// naming the writer's model is, so the verdict is not recorded. A reviewer no
// class was asked of keeps the model it ran on as its ReviewerModel; one the
// set asked a class of keeps the model it was dispatched on. A reviewer handed
// out before version 27 records neither, and is still checked against the
// lane's writer.
func (v *ValidatorRun) recordReviewerModel(l Lane, rel, reported, text string) error {
	if !slices.Contains(reviewRoles, v.Role) {
		return nil
	}
	writer := v.WriterModel
	if writer == "" {
		writer = laneWriterModel(l)
	}
	ran, from := reported, "its runner reported"
	if ran == "" {
		ran, from = namedReviewerModel(text), "it names"
	}
	if !matrix.OtherModel(writer, ran) {
		what := "names no model it ran on"
		if ran != "" {
			what = fmt.Sprintf("was reviewed by %s, the model %s, which counts as the writer's, %s", shownModel(ran), from, shownModel(writer))
		}
		return refuse("receipt", "", l.ID, fmt.Sprintf("%s %s: a model other than the writer's reviews every change", rel, what),
			fmt.Sprintf("start a fresh %s with the brief %s on a model other than %s; its return names that model on a line `%s <model>` straight after its first line, then hand it back",
				v.Role, v.Brief, shownModel(writer), ReviewerModelPrefix))
	}
	if v.WriterModel != "" && v.Class == "" && modelWritten(ran) {
		v.ReviewerModel = ran
	}
	return nil
}

// modelsPhrase names a writer's and a reviewer's models, each as shownModel
// shows it; "" when neither is recorded.
func modelsPhrase(writer, reviewer string) string {
	if writer == "" && reviewer == "" {
		return ""
	}
	return "written by " + shownModel(writer) + ", reviewed by " + shownModel(reviewer)
}
