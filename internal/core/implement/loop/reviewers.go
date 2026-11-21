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
// A model other than every writer's reviews every change (itd-2610301750162657,
// spc-2610301925429250 step 4). The writers of a lane are every model that
// wrote into it up to the round (laneWriters): each model a runner was started
// on for a writing role, whether or not a receipt came back (Lane.Launched),
// and for each implementer's receipt, the first, a fix round's and a conflict
// resolver's alike, the model its runner reported and the models the receipt
// named, where each is known. A round with no runner's report and no known
// name leaves its writer unknown, and the lane is handed back as stuck. A
// model learned after reviewers were handed out is a writer from then on
// (resettleWriters), and every save holds the stored models to the read's
// checks (modelsSavable). Before a reviewer of the two code-review roles is handed out, on the
// host or to a runner, the loop compares each writer's model with the model the
// reviewer's class resolves to, by matrix.OtherModel, under which two names are
// other models only when that is shown. Where the class holds a writer's model
// the reviewer takes the next class, in the order Classes gives and wrapping
// round, whose model differs from every writer's, that the run's runners can
// launch, and, while one remains, that no other reviewer of the role holds;
// a run started before the class map was stored chooses among the classes its
// set recorded. When no class differs, the lane is handed back as stuck with
// the reason noOtherModel. The choice is made once: a resumed run reads it
// back and never decides it again. At its return, every model the loop holds
// for the reviewer (the one it was asked for, the one its runner reported, and
// the one its return names on its model line, which every return must carry)
// must differ from every writer's, or the return is refused and its verdict
// not recorded.

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
	"A return that names none, or names a model that wrote the lane, is refused as a writer's own review: a model other than every writer's reviews every change."

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
// token of ASCII letters, digits, `.`, `_` and `-`, with a `/` or a `:` only
// between two of those, so nothing in it can start a heading, a trailer, a
// mention, a link or markup; else a description that does not repeat it.
func shownModel(s string) string {
	if s == "" {
		return "none named"
	}
	plain := func(c byte) bool {
		return c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9' || c == '.' || c == '_' || c == '-'
	}
	ok := len(s) <= maxNotedModel && modelWritten(s)
	for i := 0; ok && i < len(s); i++ {
		switch c := s[i]; {
		case plain(c):
		case c == '/' || c == ':':
			ok = i > 0 && i < len(s)-1 && plain(s[i-1]) && plain(s[i+1])
		default:
			ok = false
		}
	}
	if ok {
		return s
	}
	return termsafe.DescribeRefused(s)
}

// spannedModel is shownModel in a code span where the name itself is shown,
// as the pull request's body and the run record print it.
func spannedModel(s string) string {
	if shown := shownModel(s); shown == s {
		return "`" + s + "`"
	} else {
		return shown
	}
}

// writerKnown reports whether s is a model the loop can keep as a writer's
// and compare: one modelWritten keeps, which matrix.OtherModel can read.
func writerKnown(s string) bool { return modelWritten(s) && matrix.KnownModel(s) }

// laneWriters are the models that wrote into lane l up to its kth receipt, in
// the order they wrote, each named once: before each receipt, every model a
// runner was started on for a writing role while that many receipts stood
// (Launched), whether or not a receipt came back from it; then, for the
// receipt, the model its runner reported, the model the receipt named before
// the landing replaced it (Named), and its model, each where it is a known
// name. A receipt whose runner reported a model no comparison can read, or
// whose runner reported none while neither of its own names is known, leaves
// its writer unknown: why says which, and no writers are returned. Where no
// runner reported, the receipt's own word is believed.
func laneWriters(l Lane, k int) (writers []string, why string) {
	add := func(m string) {
		if !slices.Contains(writers, m) {
			writers = append(writers, m)
		}
	}
	for i, r := range l.Receipts[:min(k, len(l.Receipts))] {
		for _, w := range l.Launched {
			if w.After == i && writerKnown(w.Model) {
				add(w.Model)
			}
		}
		if r.Role != RoleImplementer {
			continue
		}
		reported := ""
		if r.Route != nil {
			reported = r.Route.Model
		}
		if reported != "" {
			if !writerKnown(reported) {
				return nil, fmt.Sprintf("the runner of receipt %d (%s) reported %s, which %s", i+1, r.Receipt, spannedModel(reported), noFamily)
			}
			add(reported)
		}
		named := false
		for _, m := range []string{r.Named, r.Model} {
			if writerKnown(m) {
				add(m)
				named = true
			}
		}
		if !named && reported == "" {
			what := "names no model"
			if r.Model != "" {
				what = "names " + spannedModel(r.Model) + ", which " + noFamily + ","
			}
			return nil, fmt.Sprintf("receipt %d (%s) %s and no runner reported the model that wrote it", i+1, r.Receipt, what)
		}
	}
	if len(writers) == 0 {
		return nil, "the lane has no implementer's receipt"
	}
	return writers, ""
}

// noFamily is what a reason says of a model name matrix.KnownModel does not
// read: it may hold a family the comparison does not list, which reads as the
// writer's own model.
const noFamily = "holds no model family the comparison recognises (internal/core/matrix/samemodel.go lists them)"

// differsFromAll reports whether model is another model than every writer's.
func differsFromAll(writers []string, model string) bool {
	if len(writers) == 0 {
		return false
	}
	for _, w := range writers {
		if !matrix.OtherModel(w, model) {
			return false
		}
	}
	return true
}

// writersPhrase names writers as the record shows them, each spanned.
func writersPhrase(writers []string) string {
	shown := make([]string, 0, len(writers))
	for _, w := range writers {
		shown = append(shown, spannedModel(w))
	}
	return strings.Join(shown, " and ")
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

// otherClass is the class a reviewer of role the set asked for at entry is
// dispatched on when writers wrote the lane: entry itself when its model is
// another than every writer's; else the next class after it, in Classes order
// and wrapping round, among those rs may choose, whose model is, and which the
// run's runners can launch for role (rs.Unlaunchable), passing over one whose
// model is the same as a sibling's (another reviewer of the role) while a
// further class qualifies; false when none does.
func otherClass(rs ReviewSet, role string, entry ReviewClass, writers, siblings []string) (ReviewClass, bool) {
	if differsFromAll(writers, entry.Model) {
		return entry, true
	}
	from := slices.Index(Classes, entry.Class)
	choosable := rs.choosable()
	var shared *ReviewClass
	for k := 1; k < len(Classes); k++ {
		next := Classes[(from+k+len(Classes))%len(Classes)]
		i := slices.IndexFunc(choosable, func(c ReviewClass) bool { return c.Class == next })
		if i < 0 || !differsFromAll(writers, choosable[i].Model) || slices.Contains(rs.unlaunchableFor(role), next) {
			continue
		}
		if !slices.ContainsFunc(siblings, func(s string) bool { return !matrix.OtherModel(s, choosable[i].Model) }) {
			return choosable[i], true
		}
		if shared == nil {
			c := choosable[i]
			shared = &c
		}
	}
	if shared != nil {
		return *shared, true
	}
	return ReviewClass{}, false
}

// siblingModels are the models v's siblings hold, the other reviewers of its
// role in round r, as they stood when v was handed out: a sibling handed out
// before it (a lower index) at the model it was dispatched on, and one after it
// at the model the set names for it.
func siblingModels(rs ReviewSet, r ValidationRound, v ValidatorRun) []string {
	set, _ := rs.Role(v.Role)
	var out []string
	for _, u := range r.Validators {
		switch {
		case u.Role != v.Role || u.Reviewer == v.Reviewer:
		case u.Reviewer < v.Reviewer:
			out = append(out, u.Model)
		case u.Reviewer < len(set.Reviewers):
			out = append(out, set.Reviewers[u.Reviewer].Model)
		}
	}
	return out
}

// chooseReviewerModel settles, once, the model a reviewer of the code-review
// roles in round r is dispatched on, before it is handed out: it records the
// lane's writers and, for a reviewer the set asked a class of, moves it to
// another class when its own holds a writer's model (otherClass), recording
// the model it is dispatched on. It returns the hand-back when a writer's
// model is unknown or no class the run may choose resolves to another model.
// A reviewer that already records its writers, or that an earlier version
// handed out, was settled before and is left as it is, so a resumed run never
// decides it again.
func chooseReviewerModel(st State, l Lane, r ValidationRound, v *ValidatorRun) *HandBack {
	if !slices.Contains(reviewRoles, v.Role) || len(v.WriterModels) > 0 || v.ModelsUnrecorded {
		return nil
	}
	writers, why := laneWriters(l, len(l.Receipts))
	if why != "" {
		return unknownWriterHandBack(st, l, why)
	}
	if v.Class == "" || st.Review == nil {
		v.WriterModels = writers
		return nil
	}
	// The move starts from the class the set names for the reviewer, as the
	// stored-state check recomputes it.
	entry := ReviewClass{Class: v.Class, Model: v.Model, Effort: v.Effort}
	if set, ok := st.Review.Role(v.Role); ok && v.Reviewer >= 0 && v.Reviewer < len(set.Reviewers) {
		entry = set.Reviewers[v.Reviewer]
	}
	got, ok := otherClass(*st.Review, v.Role, entry, writers, siblingModels(*st.Review, r, *v))
	if !ok {
		why := fmt.Sprintf("the lane was written by %s, and the %s's class %s and every other class the run may choose and launch resolve to one of them",
			writersPhrase(writers), v.Role, classPhrase(v.Class, spannedModel(v.Model), v.Effort))
		var unread []string
		for _, c := range st.Review.choosable() {
			if !matrix.KnownModel(c.Model) {
				unread = append(unread, c.Class+"'s "+spannedModel(c.Model))
			}
		}
		if len(unread) > 0 {
			why += "; " + strings.Join(unread, " and ") + " " + noFamily + ", so it counts as a writer's"
		}
		return noOtherModelHandBack(st, l, v.Role, why)
	}
	v.Class, v.Model, v.Effort = got.Class, got.Model, got.Effort
	v.WriterModels, v.ReviewerModel = writers, got.Model
	return nil
}

// noOtherModelHandBack is the stuck hand-back of a lane no reviewer other
// than its writers' models can judge.
func noOtherModelHandBack(st State, l Lane, role, why string) *HandBack {
	return &HandBack{Kind: HandBackStuck, Reason: noOtherModel + ": " + why,
		Home: fmt.Sprintf("a person: give a class of models.classes in %s a model other than every writer's (one a drive's runners can launch, for a drive), discard the lane with "+
			"`abcd implement step --run %s --discard %s`, and build %s again; the run keeps the class map it started on, so the %s is never "+
			"moved to a model the run did not record", layered.Config.MachineOrigin(), st.RunID, l.ID, keyOf(st), role)}
}

// unknownWriterHandBack is the stuck hand-back of a lane one of whose writers'
// models is unknown: no reviewer can be shown to differ from it, and the
// lane's receipts are fixed once verified, so only a new lane cures it.
func unknownWriterHandBack(st State, l Lane, why string) *HandBack {
	return &HandBack{Kind: HandBackStuck, Reason: noOtherModel + ": " + why + ", so no reviewer can be shown to be another model",
		Home: fmt.Sprintf("a person: discard the lane with `abcd implement step --run %s --discard %s` and build %s again with an implementer whose "+
			"model holds a family the comparison recognises: on a runner that reports such a model, or, with no runner's report, with a receipt naming it as a plain id "+
			"(a bare claude-* id, or <vendor>:<model-id>); a family internal/core/matrix/samemodel.go does not list reads as the writer's own, "+
			"and a verified receipt is never read again for its model before the landing", st.RunID, l.ID, keyOf(st))}
}

// namedReviewerModel is the model a reviewer's return names on its model
// line, the first line opening with ReviewerModelPrefix among the return's
// first seven lines; "" when it names none.
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
// later write: a reviewer's models or its mark, the review set's class map or
// its unlaunchable classes, a receipt's named model, or a lane's launched
// writers; "" when there is none.
func (s State) modelled() string {
	if s.Review != nil && len(s.Review.Classes) > 0 {
		return "the review set's class map"
	}
	if s.Review != nil && len(s.Review.Unlaunchable) > 0 {
		return "the review set's unlaunchable classes"
	}
	for _, l := range s.Lanes {
		if len(l.Launched) > 0 {
			return "the launched writers of " + l.ID
		}
		for i, r := range l.Receipts {
			if r.Named != "" {
				return fmt.Sprintf("a named model on %s's receipt %d", l.ID, i+1)
			}
		}
		for _, r := range l.Validation {
			for i, v := range r.Validators {
				if len(v.WriterModels) > 0 || v.ReviewerModel != "" || v.ReviewerNamed != "" || v.ModelsUnrecorded {
					return fmt.Sprintf("a writer's or a reviewer's model on %s's round %d validator %d", l.ID, r.Round, i+1)
				}
			}
		}
	}
	return ""
}

// markUnrecorded marks each code-review reviewer a state read from an earlier
// version had handed out (awaited, or with its verdict) and that records no
// model: it was dispatched before writers were compared, and the mark lets a
// version-27 file keep it so. It is called only on such a read.
func (s *State) markUnrecorded() {
	for li := range s.Lanes {
		l := &s.Lanes[li]
		for ri := range l.Validation {
			for vi := range l.Validation[ri].Validators {
				v := &l.Validation[ri].Validators[vi]
				if slices.Contains(reviewRoles, v.Role) && len(v.WriterModels) == 0 && v.ReviewerModel == "" &&
					(v.Verdict != "" || l.awaited(v.Return)) {
					v.ModelsUnrecorded = true
				}
			}
		}
	}
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
// loop writes (reviewersUnwritten).
func (s State) reviewersWritten() bool { return s.reviewersUnwritten() == "" }

// maxLaunchedWriters bounds the launched writers a lane keeps.
const maxLaunchedWriters = 64

// setReviewer reports whether a reviewer of role at index, with class, model
// and effort, is one the run's stored review set names: on one of the review
// set's roles, at an index among the reviewers the set names for that role,
// with exactly the class and model the set names at that index (the set is
// fixed when the run starts and every round's reviewers are cut from it,
// roundReviewers). A validator with no class carries no index and no model.
func (s State) setReviewer(role string, index int, class, model, effort string) bool {
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

// reviewersUnwritten names the first reviewer a state carries that the loop
// does not write, "" when every one is: each at the set's class (setReviewer)
// or the class the move takes instead (modelsWritten); a model status the
// verifier writes, unsent only where the reviewer route answered; an await's
// reviewer likewise; each comparison one of the outcomes over one or two
// reviewers; a receipt's named model only where the landing replaced its model
// and the comparison reads it; and each launched writer a known name, counted
// against the lane's receipts. The read refuses a file it names, and so does
// every save (storeState), so the loop never writes a state its own read
// refuses.
func (s State) reviewersUnwritten() string {
	for _, l := range s.Lanes {
		// A receipt keeps the name it named only where the landing replaced
		// its model with one the trailer can name (rereadModels), and only a
		// name the comparison reads (keptName).
		for i, r := range l.Receipts {
			if r.Named != "" && (r.Named == r.Model || !nameable(r.Model) || !writerKnown(r.Named)) {
				return fmt.Sprintf("%s's receipt %d keeps a named model the landing never keeps", l.ID, i+1)
			}
		}
		if len(l.Launched) > maxLaunchedWriters {
			return fmt.Sprintf("%s keeps more launched writers than %d", l.ID, maxLaunchedWriters)
		}
		for i, w := range l.Launched {
			if !writerKnown(w.Model) || w.After < 0 || w.After > len(l.Receipts) {
				return fmt.Sprintf("%s's launched writer %d is no known model started before one of its receipts", l.ID, i+1)
			}
		}
		for _, a := range l.Awaits {
			if a.Role == RoleJudge {
				if !judgeAwaitWritten(l, a) {
					return fmt.Sprintf("%s's await of the %s names no judge its round records", l.ID, a.Role)
				}
				continue
			}
			if a.Reviewer == nil {
				if a.Class != "" || a.Model != "" || a.Effort != "" {
					return fmt.Sprintf("%s's await of the %s carries a class with no reviewer", l.ID, a.Role)
				}
				continue
			}
			if a.Class == "" {
				return fmt.Sprintf("%s's await of the %s names a reviewer with no class", l.ID, a.Role)
			}
			// An await of a moved reviewer carries the class its validator
			// was dispatched on, which that validator's own check holds.
			moved := false
			for _, r := range l.Validation {
				for _, v := range r.Validators {
					if v.Return == a.Receipt && len(v.WriterModels) > 0 && v.Role == a.Role && v.Reviewer == *a.Reviewer &&
						v.Class == a.Class && v.Model == a.Model && v.Effort == a.Effort {
						moved = true
					}
				}
			}
			if !moved && !s.setReviewer(a.Role, *a.Reviewer, a.Class, a.Model, a.Effort) {
				return fmt.Sprintf("%s's await of the %s names a class or model its review set and its validator do not", l.ID, a.Role)
			}
		}
		for ri, r := range l.Validation {
			for vi, v := range r.Validators {
				at := fmt.Sprintf("%s's round %d validator %d (the %s)", l.ID, r.Round, vi+1, v.Role)
				if !s.modelsWritten(l, ri, v) {
					return at + " carries writers or models the rest of the file does not give"
				}
				switch v.ModelStatus {
				case "":
				case ModelAsked, ModelReported:
					if v.Class == "" || answeredByReviewRoute(v.Route) {
						return at + " carries a model status the verifier never writes"
					}
				case ModelUnsent:
					if v.Class == "" || !answeredByReviewRoute(v.Route) {
						return at + " carries a model status the verifier never writes"
					}
				default:
					return at + " carries a model status the verifier never writes"
				}
			}
			for _, c := range r.Comparisons {
				if !slices.Contains(reviewRoles, c.Role) || c.Reviewers < 1 || c.Reviewers > maxReviewers || len(c.Verdicts) != c.Reviewers ||
					!slices.Contains([]string{ComparisonAgree, ComparisonDisagree, ComparisonNone}, c.Outcome) {
					return fmt.Sprintf("%s's round %d carries a comparison the loop never writes", l.ID, r.Round)
				}
			}
		}
	}
	return ""
}

// modelsWritten reports whether validator v of lane l's round ri carries the
// models the loop writes, each held to what the same file says elsewhere:
//
//   - a reviewer that records no model is one not yet handed out, the
//     intent-auditor, or one an earlier version handed out (ModelsUnrecorded,
//     set only as such a file is read), which keeps the set's class;
//   - the writers equal laneWriters over the lane's receipts as they stood
//     when the round could be handed out: every receipt up to the previous
//     round's fix, and none from its own fix on;
//   - a reviewer asked a class is at the set's class, or at the one otherClass
//     takes instead, recomputed from the checked writers, the checked class
//     map and its siblings;
//   - the reviewer's model is the one it was asked for, or once it returned
//     the one its runner reported or, where the reviewer route answered, the
//     one its return named; the named model is kept only where it is another
//     model than that one; and every model held for it is another model than
//     every writer's.
func (s State) modelsWritten(l Lane, ri int, v ValidatorRun) bool {
	reviewer := s.setReviewer
	r := l.Validation[ri]
	if len(v.WriterModels) == 0 {
		if v.ReviewerModel != "" || v.ReviewerNamed != "" || !reviewer(v.Role, v.Reviewer, v.Class, v.Model, v.Effort) {
			return false
		}
		if v.ModelsUnrecorded {
			// Only a run started before version 27, whose review set keeps no
			// class map, can hold a reviewer an earlier version handed out.
			return slices.Contains(reviewRoles, v.Role) && (s.Review == nil || len(s.Review.Classes) == 0)
		}
		// Not yet handed out on a code-review role; the intent-auditor never
		// records a model.
		return !slices.Contains(reviewRoles, v.Role) || v.Verdict == "" && !l.awaited(v.Return)
	}
	if v.ModelsUnrecorded || !slices.Contains(reviewRoles, v.Role) || len(v.WriterModels) > maxStoredWriters {
		return false
	}
	for _, w := range v.WriterModels {
		if !writerKnown(w) {
			return false
		}
	}
	if _, ok := writersAt(l, ri, v.WriterModels); !ok {
		return false
	}
	held := []string{v.ReviewerModel}
	if v.ReviewerNamed != "" {
		if v.Verdict == "" || !modelWritten(v.ReviewerNamed) || !matrix.OtherModel(v.ReviewerNamed, v.ReviewerModel) {
			return false
		}
		held = append(held, v.ReviewerNamed)
	}
	reported := ""
	if v.Route != nil {
		reported = v.Route.Model
	}
	if v.Class == "" {
		if !reviewer(v.Role, v.Reviewer, "", v.Model, v.Effort) {
			return false
		}
		if v.Verdict == "" {
			return v.ReviewerModel == ""
		}
		if reported != "" && v.ReviewerModel != reported {
			return false
		}
		return modelWritten(v.ReviewerModel) && differsFromAll(v.WriterModels, v.ReviewerModel) &&
			(v.ReviewerNamed == "" || differsFromAll(v.WriterModels, v.ReviewerNamed))
	}
	if s.Review == nil || v.Reviewer < 0 || v.Reviewer >= maxReviewers {
		return false
	}
	set, ok := s.Review.Role(v.Role)
	if !ok || v.Reviewer >= len(set.Reviewers) {
		return false
	}
	want, ok := otherClass(*s.Review, v.Role, set.Reviewers[v.Reviewer], v.WriterModels, siblingModels(*s.Review, r, v))
	if !ok || v.Class != want.Class || v.Model != want.Model || v.Effort != want.Effort {
		return false
	}
	switch {
	case v.Verdict == "" || v.ModelStatus == ModelAsked:
		if v.ReviewerModel != v.Model {
			return false
		}
	case v.ModelStatus == ModelReported:
		if reported == "" || v.ReviewerModel != reported {
			return false
		}
	case v.ModelStatus == ModelUnsent:
		if v.ReviewerModel == "" {
			return false
		}
	default:
		return false
	}
	held = append(held, v.Model)
	for _, m := range held {
		if !modelWritten(m) || !differsFromAll(v.WriterModels, m) {
			return false
		}
	}
	return true
}

// maxStoredWriters bounds the writers a reviewer records: one implementer and
// each fix round's and conflict resolver's, each with at most two names.
const maxStoredWriters = 64

// writersAt reports whether writers are the lane's writers as laneWriters
// reads them from the lane's receipts as they stood when round ri was handed
// out: every receipt verified before it (an earlier round's fix, a sync's
// conflict resolver whose fresh round is this one or earlier, and the first),
// and none verified after it (its own or a later round's fix, a later sync's
// resolver). It returns how many receipts stood.
func writersAt(l Lane, ri int, writers []string) (int, bool) {
	at := func(path string) int {
		if path == "" {
			return -1
		}
		return slices.IndexFunc(l.Receipts, func(r ReceiptRecord) bool { return r.Receipt == path })
	}
	round := l.Validation[ri].Round
	lo, hi := 1, len(l.Receipts)
	before := func(i int) {
		if i >= 0 {
			lo = max(lo, i+1)
		}
	}
	after := func(i int) {
		if i >= 0 {
			hi = min(hi, i)
		}
	}
	for j, r := range l.Validation {
		if j < ri {
			before(at(r.Fix))
		} else {
			after(at(r.Fix))
		}
	}
	for _, sy := range l.Syncs {
		if sy.Round > 0 && sy.Round <= round {
			before(at(sy.Receipt))
		} else {
			after(at(sy.Receipt))
		}
	}
	for k := lo; k <= hi; k++ {
		if got, why := laneWriters(l, k); why == "" && slices.Equal(got, writers) {
			return k, true
		}
	}
	return 0, false
}

// recordReviewerModel checks, as a code-review role's verdict is about to be
// recorded, that every model the loop holds for the reviewer is another model
// than every writer's: the one it was asked for (its class's), the one its
// runner reported, and the one its return names on its model line, which
// every return must carry. A return naming none, or any of them counting as a
// writer's, is refused, so the verdict is not recorded. The reviewer then
// records the model its runner reported, else the one it was asked for, else
// (where the reviewer route answered, unsent, or no class was asked of it) the
// one its return named, and the named one beside it where that is another
// model. A reviewer an earlier version handed out is checked against the
// lane's writers and records none.
func (v *ValidatorRun) recordReviewerModel(l Lane, rel, reported, text string, unsent bool) error {
	if !slices.Contains(reviewRoles, v.Role) {
		return nil
	}
	writers := v.WriterModels
	if len(writers) == 0 {
		w, why := laneWriters(l, len(l.Receipts))
		if why != "" {
			return refuse("receipt", "", l.ID, fmt.Sprintf("%s cannot be recorded: %s, so no reviewer can be shown to be another model", rel, why),
				fmt.Sprintf("discard the lane with `abcd implement step --discard %s` and build it again with an implementer whose receipt names the model it runs on as a plain id", l.ID))
		}
		writers = w
	}
	named := namedReviewerModel(text)
	refused := func(what string) error {
		return refuse("receipt", "", l.ID, fmt.Sprintf("%s %s: a model other than every writer's reviews every change", rel, what),
			fmt.Sprintf("start a fresh %s with the brief %s on a model other than %s; its return names that model on a line `%s <model>` among its first lines, then hand it back",
				v.Role, v.Brief, writersPhrase(writers), ReviewerModelPrefix))
	}
	if named == "" {
		return refused("names no model it ran on")
	}
	type held struct{ model, from string }
	var hs []held
	if v.Class != "" {
		hs = append(hs, held{v.Model, "it was asked for"})
	}
	if reported != "" {
		hs = append(hs, held{reported, "its runner reported"})
	}
	hs = append(hs, held{named, "its return names"})
	for _, h := range hs {
		if !modelWritten(h.model) || !differsFromAll(writers, h.model) {
			return refused(fmt.Sprintf("holds %s, the model %s, which counts as a writer's (%s)", spannedModel(h.model), h.from, writersPhrase(writers)))
		}
	}
	if len(v.WriterModels) == 0 {
		return nil
	}
	rec := reported
	switch {
	case rec != "":
	case v.Class != "" && !unsent:
		rec = v.Model
	default:
		rec = named
	}
	v.ReviewerModel = rec
	if matrix.OtherModel(named, rec) {
		v.ReviewerNamed = named
	}
	return nil
}

// modelsPhrase names a reviewer's writers and models as the record and the
// pull request's body show them, each name in a code span where it is shown.
// The reviewer reads "reviewed by" its model where a runner reported it or the
// return named the same model; "asked of" its class's model only where the
// loop asked for one and holds no report or return; and, where the return
// named another model, both: the model asked for (or its runner reported) and
// the one its return named. A reviewer handed out before reviewers' models
// were recorded (unrecorded) says so in words, naming no model. "" when
// nothing is recorded.
func modelsPhrase(writers []string, reviewer, named, status string, classed, unrecorded bool) string {
	if unrecorded {
		return "the reviewer's models were not recorded: it was handed out before the loop recorded them"
	}
	if len(writers) == 0 && reviewer == "" {
		return ""
	}
	out := "written by " + writersPhrase(writers)
	if len(writers) == 0 {
		out = "written by " + shownModel("")
	}
	// A reviewer no class was asked of records a report as its model where
	// its return named another (recordReviewerModel).
	reported := status == ModelReported || !classed && named != ""
	switch {
	case named != "" && reported:
		return out + ", run on, as its runner reported, " + spannedModel(reviewer) + ", its return naming " + spannedModel(named)
	case named != "":
		return out + ", asked of " + spannedModel(reviewer) + ", its return naming " + spannedModel(named)
	case classed && status == "":
		return out + ", asked of " + spannedModel(reviewer)
	}
	return out + ", reviewed by " + spannedModel(reviewer)
}

// models is modelsPhrase for the validator.
func (v ValidatorRun) models() string {
	return modelsPhrase(v.WriterModels, v.ReviewerModel, v.ReviewerNamed, v.ModelStatus, v.Class != "", v.ModelsUnrecorded)
}

// resettleWriters keeps lane l's stored writers whole after a model the loop
// learned late was filled in or replaced on one of its receipts: a runner's
// report stamped on a receipt a step collected before it arrived
// (answerCollected), and the landing's re-read of a receipt's model
// (rereadModels). before is the lane as it stood. A model learned so is a
// writer from that moment, so each reviewer that records writers takes the
// lane's writers recomputed over the receipts it was handed out under, in the
// same change; and every reviewer whose writers can no longer be read, any
// model it holds (asked for, reported or named) is not another model than
// every one of them, or whose class the move no longer takes, is withdrawn
// (withdrawReviewer): its hand-out is dropped and its verdict, if it had one,
// no longer counts. It returns each reviewer withdrawn, named; settled
// reports whether the next step hands each out again by the same move as at
// dispatch, which it does only where every one withdrawn is of the lane's
// open round at its validate stage, with no fix or sync out for it. Where it
// is not, the caller hands the lane back as stuck (lateWriterHandBack).
func resettleWriters(st State, before Lane, l *Lane) (withdrawn []string, settled bool) {
	l.Validation = slices.Clone(l.Validation)
	for ri := range l.Validation {
		l.Validation[ri].Validators = slices.Clone(l.Validation[ri].Validators)
	}
	settled = true
	pull := func(ri, vi int) {
		r := &l.Validation[ri]
		withdrawn = append(withdrawn, fmt.Sprintf("round %d's %s", r.Round, withdrawReviewer(st, l, ri, vi)))
		if ri != len(l.Validation)-1 || r.Fix != "" || l.Stage != StageValidate || l.awaitsRole(RoleImplementer) {
			settled = false
		}
	}
	for ri := range l.Validation {
		for vi := range l.Validation[ri].Validators {
			v := &l.Validation[ri].Validators[vi]
			if len(v.WriterModels) == 0 {
				continue
			}
			k, ok := writersAt(before, ri, v.WriterModels)
			if !ok {
				continue
			}
			if got, why := laneWriters(*l, k); why == "" {
				v.WriterModels = got
			} else {
				pull(ri, vi)
			}
		}
	}
	// A reviewer withdrawn moves its siblings' recomputed classes, so the
	// check runs again until every one left reads.
	for again := true; again; {
		again = false
		s := st
		s.Lanes = []Lane{*l}
		for ri := range l.Validation {
			for vi, v := range l.Validation[ri].Validators {
				if len(v.WriterModels) > 0 && !s.modelsWritten(*l, ri, v) {
					pull(ri, vi)
					again = true
				}
			}
			if again {
				break
			}
		}
	}
	return withdrawn, settled
}

// withdrawReviewer withdraws validator vi of lane l's round ri as an
// uncounted return is: its await is dropped, so whatever its agent hands in
// later is answered as one nothing awaits; its verdict, findings and models
// are cleared, with the round's comparison for its role; and it stands again
// as the set names it (roundReviewers), to be handed out by the next step. It
// returns the reviewer, named.
func withdrawReviewer(st State, l *Lane, ri, vi int) string {
	r := &l.Validation[ri]
	v := r.Validators[vi]
	name := v.Role
	if v.Class != "" {
		name += fmt.Sprintf(", reviewer %d", v.Reviewer)
	}
	fresh := ValidatorRun{Role: v.Role, Reviewer: v.Reviewer, Brief: v.Brief, Return: v.Return}
	for _, c := range roundReviewers(st, v.Role) {
		if c.Reviewer == v.Reviewer {
			fresh.Class, fresh.Model, fresh.Effort = c.Class, c.Model, c.Effort
		}
	}
	r.Validators[vi] = fresh
	r.Comparisons = slices.DeleteFunc(slices.Clone(r.Comparisons), func(c Comparison) bool { return c.Role == v.Role })
	if j := r.Judge; j != nil {
		// The judge ruled on returns one of which no longer counts: its
		// ruling goes with it, its await too, and the round asks again
		// once its reviewers are back (judge.go).
		r.Judge = nil
		l.Awaits = slices.DeleteFunc(slices.Clone(l.Awaits), func(a Await) bool { return a.Receipt == j.Return })
		for k := range r.Validators {
			u := &r.Validators[k]
			u.Findings = slices.Clone(u.Findings)
			for m := range u.Findings {
				if u.Findings[m].Status == FindingStruck {
					u.Findings[m].Status, u.Findings[m].Reason = "", ""
					if r.Fix != "" {
						u.Findings[m].Status = FindingUnknown
					}
				}
			}
		}
	}
	markCitedBy(r)
	l.Awaits = slices.DeleteFunc(slices.Clone(l.Awaits), func(a Await) bool { return a.Receipt == v.Return })
	return name
}

// lateWriterFound opens the reason of every hand-back lateWriterHandBack makes.
const lateWriterFound = noOtherModel + ": a model learned after the lane's reviewers were handed out is a writer of it"

// lateWriterHandedBack reports whether l was handed back by a late writer
// (lateWriterHandBack): a report that came after its reviewers were handed out
// showed one of them is not another model than every writer.
func lateWriterHandedBack(l Lane) bool {
	return l.HandBack != nil && l.HandBack.Kind == HandBackStuck && strings.HasPrefix(l.HandBack.Reason, lateWriterFound)
}

// lateWriterHandBack is the stuck hand-back of a lane whose reviewer was
// withdrawn by resettleWriters where the next step cannot hand it out again:
// a round already settled, or a lane gone on past its validators.
func lateWriterHandBack(st State, l Lane, withdrawn []string) *HandBack {
	return &HandBack{Kind: HandBackStuck,
		Reason: lateWriterFound + ", and " + strings.Join(withdrawn, " and ") +
			" is not another model than every writer (" + writersPhrase(lastWriters(l)) + "), so its review no longer counts and the round it judged cannot be judged again",
		Home: fmt.Sprintf("a person: discard the lane with `abcd implement step --run %s --discard %s` and build %s again; its reviewers are then chosen against every model that wrote it",
			st.RunID, l.ID, keyOf(st))}
}

// lastWriters are the lane's writers over every receipt it holds, or none
// where one cannot be read.
func lastWriters(l Lane) []string {
	w, _ := laneWriters(l, len(l.Receipts))
	return w
}

// withdrawnNote is the record's words for reviewers resettleWriters
// withdrew.
func withdrawnNote(withdrawn []string, settled bool) string {
	note := "a model learned after the lane's reviewers were handed out is a writer of it, so " + strings.Join(withdrawn, " and ") +
		" was withdrawn: its review does not count"
	if settled {
		note += ", and the next step hands it out again on a class whose model differs from every writer's, or hands the lane back"
	}
	return note
}

// launchedWriter is lane's launched writers with model added, as a runner is
// started on it for the implementer, fix or conflict-resolution role: a
// writer of the lane from its next receipt on, whether or not a receipt comes
// back. Only a name the comparison reads is kept, as every stored name is;
// a runner on its harness's default model ("") or on a family the comparison
// does not list adds none, and leaves the writers to its runner's report and
// its receipt. A model already launched before the same receipt is kept
// once. A lane past the bound refuses the launch, so no writer is dropped.
func launchedWriter(lane Lane, model string) ([]LaunchedWriter, error) {
	w := LaunchedWriter{Model: model, After: len(lane.Receipts)}
	if !writerKnown(model) || slices.Contains(lane.Launched, w) {
		return lane.Launched, nil
	}
	if len(lane.Launched) >= maxLaunchedWriters {
		return nil, refuse("state", "", lane.ID, fmt.Sprintf("%s has had %d writing roles started on its runners, the most the run keeps, so no further one is started", lane.ID, maxLaunchedWriters),
			fmt.Sprintf("discard the lane with `abcd implement step --discard %s` and build it again", lane.ID))
	}
	return append(slices.Clone(lane.Launched), w), nil
}
