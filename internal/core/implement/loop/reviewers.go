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

import (
	"fmt"
	"slices"
	"strings"

	"github.com/intentdriven/abcd/internal/core/oracle"
	"github.com/intentdriven/abcd/internal/core/runner"
)

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
func reviewerPhrase(index int, class, model string) string {
	if class == "" {
		return ""
	}
	return fmt.Sprintf("reviewer %d at %s", index, classPhrase(class, model))
}

// phrase is reviewerPhrase for the validator.
func (v ValidatorRun) phrase() string { return reviewerPhrase(v.Reviewer, v.Class, v.Model) }

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
	for _, l := range s.Lanes {
		for _, a := range l.Awaits {
			if a.Reviewer == nil {
				if a.Class != "" || a.Model != "" || a.Effort != "" {
					return false
				}
				continue
			}
			if a.Class == "" || !reviewer(a.Role, *a.Reviewer, a.Class, a.Model, a.Effort) {
				return false
			}
		}
		for _, r := range l.Validation {
			for _, v := range r.Validators {
				if !reviewer(v.Role, v.Reviewer, v.Class, v.Model, v.Effort) {
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
