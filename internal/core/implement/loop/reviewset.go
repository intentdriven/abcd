package loop

// reviewset.go is the run's review set (itd-2610240922461281,
// spc-2610240951109420 step 1): how many reviewers each of the loop's two
// code-review roles runs, at which model class, and an optional judge,
// resolved once when a run starts through the one layered configuration
// reader and written into the run's state with the layer each value came
// from, as the pace is (pace.go), so a later invocation honours the set the
// run started on whatever the files say by then.
//
//   - review.reviewers is a list of one or two classes, for both roles, or an
//     object naming a list per role ({"security-reviewer": [...]}); each role
//     takes the highest layer that sets it, else the bundled pair (research
//     and default, itd-2611101419006280 decision 4).
//   - review.judge is one class, else the bundled top. A judge runs only once
//     its agent has cleared itd-81's calibration gate, so the bundled judge is
//     stored waiting, or active once the gate has cleared, decided once when
//     the run starts (itd-2611101419006280 decision 5); a judge a
//     configuration file names while no agent has cleared it is refused.
//   - review.max_reviewers is the machine's ceiling on reviewers per role,
//     one or two (two when unset). It is read from the machine's file alone:
//     a repository's list longer than the ceiling is clamped, with one line
//     on stderr naming both files, and a repository's own max_reviewers is
//     skipped with a warning.
//   - models.effort is how hard each class works (itd-2611101419006280
//     decision 6): one of the levels for every class, or an object naming a
//     level per class, and high for a class nothing names. A higher effort
//     spends more of the person's budget, so it is read from the machine's
//     file or the bundled default alone, as the class map is, and a
//     repository's value is skipped with a warning. The effort rides with
//     each resolved class into the run's state and, where a runner launches a
//     role at the class, into the request it is handed (decision 7).
//   - models.classes is the class map: research, top, default and small, each
//     a <provider>/<model> route. Which model spends the person's key is
//     theirs alone (decision 3), so the map is read from the machine's file
//     or the bundled default alone, and a repository's map is skipped with a
//     warning, as a repository's runner route is (internal/core/runner). The
//     route's shape is checked here; its admission is the runner's, made when
//     a runner is handed the model (spec step 2), since a host-run reviewer is
//     only asked for the class: against the provider's allowlist, except that
//     a drive's walled launch of the claude harness takes a model of the
//     harness's own vendor by name where the class map resolved it during the
//     drive and the machine configures no block for that vendor
//     (iss-2611080057001367), so the bundled map runs in a drive on a
//     subscription-only machine. The drive reads the map at its start
//     (ClassRoutes) and each run's start reads it again, handing its routes
//     on (Options.ClassMapRead). A drive asks the same admission of every
//     reviewer before it starts (CheckReviewersLaunch).
//
// A fault in any of them is refused at the review stage before anything is
// written. The two roles are the loop's ruthless-reviewer and
// security-reviewer; the intent-auditor keeps its single run (decision 6).

import (
	"fmt"
	"maps"
	"slices"
	"strings"
	"unicode"

	"github.com/intentdriven/abcd/internal/core/layered"
	"github.com/intentdriven/abcd/internal/core/runner"
	"github.com/intentdriven/abcd/internal/termsafe"
)

// The model classes, strongest first, named for what each is for
// (itd-2611101419006280 decision 1): research, top, default and small. They
// renamed the scale of iss-2610140652230444's ruling of 2026-10-14 one for one
// (top, middle, below and smallest), and the old names are refused, never
// aliased, since the old top and the new top are different classes.
const (
	ClassResearch = "research"
	ClassTop      = "top"
	ClassDefault  = "default"
	ClassSmall    = "small"
)

// Classes are the class names the class map carries, strongest first.
var Classes = []string{ClassResearch, ClassTop, ClassDefault, ClassSmall}

// renamedClasses maps each class name the classes took before
// itd-2611101419006280 to the name the same class takes now. The old top is
// research, so a top read in a file is the new top, and is caught where a file
// can show which names it was written for (oldClassFault).
var renamedClasses = map[string]string{"top": ClassResearch, "middle": ClassTop, "below": ClassDefault, "smallest": ClassSmall}

// retiredClasses are the old names no class takes any more.
var retiredClasses = []string{"middle", "below", "smallest"}

// renameNote is the whole mapping, as every refusal of an old name says it.
const renameNote = "the classes were renamed for what each is for: `top` is now `research`, `middle` is `top`, `below` is `default` " +
	"and `smallest` is `small`"

// BundledClasses is the bundled class map: one vendor's models, a default and
// not part of the rule (decision 3). For that vendor today the classes are
// Fable, Opus, Sonnet and Haiku (iss-2610140652230444).
var BundledClasses = map[string]string{
	ClassResearch: "anthropic/claude-fable-5",
	ClassTop:      "anthropic/claude-opus-5-5",
	ClassDefault:  "anthropic/claude-sonnet-5-5",
	ClassSmall:    "anthropic/claude-haiku-4-5",
}

// BundledReviewers is the bundled list each role runs: the research class and
// the default class, the widest spread between two reviewers
// (itd-2611101419006280 decision 4, reversing the top and default pair of
// itd-2610240922461281 decision 1).
var BundledReviewers = []string{ClassResearch, ClassDefault}

// BundledJudge is the bundled judge's class: the class between the pair. It
// runs only once a judge agent has cleared itd-81's calibration gate, and
// until then a run starts with it waiting (decision 5).
const BundledJudge = ClassTop

// BundledMaxReviewers is the ceiling on reviewers per role when the machine
// sets none, and maxReviewers the most any ceiling may be.
const (
	BundledMaxReviewers = 2
	maxReviewers        = 2
)

// reviewRoles are the roles a review set governs, in the order the validate
// stage hands them out.
var reviewRoles = []string{RoleRuthless, RoleSecurity}

// The configuration keys the review set claims.
const (
	reviewNamespace = "review"
	keyReviewers    = "reviewers"
	keyJudge        = "judge"
	keyMaxReviewers = "max_reviewers"
	modelsNamespace = "models"
	keyClasses      = "classes"
	keyEffort       = "effort"
	classNamespace  = modelsNamespace + "." + keyClasses
	effortKey       = modelsNamespace + "." + keyEffort
)

// EffortLevels are the levels models.effort takes, lowest first: the five the
// harness takes (decision 6), the runner's own list, which refuses a request
// at any other.
var EffortLevels = runner.EffortLevels

// BundledEffort is every class's effort when the machine names none.
const BundledEffort = "high"

// effortLevelsPhrase names the levels as a refusal says them.
var effortLevelsPhrase = strings.Join(EffortLevels[:len(EffortLevels)-1], ", ") + " and " + EffortLevels[len(EffortLevels)-1]

// ReviewClass is one class of a review set with the model the class map
// resolves it to, and the map layer ("machine" or "bundled") and origin that
// supplied the model; and the effort the class runs at, with the layer and
// origin of models.effort that set it. A run started before efforts carries
// none of the three.
type ReviewClass struct {
	Class        string `json:"class"`
	Model        string `json:"model"`
	MapLayer     string `json:"map_layer"`
	MapOrigin    string `json:"map_origin"`
	Effort       string `json:"effort,omitempty"`
	EffortLayer  string `json:"effort_layer,omitempty"`
	EffortOrigin string `json:"effort_origin,omitempty"`
}

// RoleReviewers is one role's reviewers, with the layer and origin of the
// list. Clamped is the length the list named before the machine's ceiling
// cut it; 0 when it was not cut.
type RoleReviewers struct {
	Role      string        `json:"role"`
	Reviewers []ReviewClass `json:"reviewers"`
	Layer     string        `json:"layer"`
	Origin    string        `json:"origin"`
	Clamped   int           `json:"clamped,omitempty"`
}

// ReviewJudge is a review set's judge: its class, the model it resolves to,
// the layer and origin of review.judge, and its status, fixed when the run
// starts (itd-2611101419006280 decision 5): active once a judge agent has
// cleared itd-81's calibration gate, else waiting. A waiting judge is never
// launched, awaited or counted in an estimate, and a run never gains a judge
// part-way.
type ReviewJudge struct {
	Class        string `json:"class"`
	Model        string `json:"model"`
	MapLayer     string `json:"map_layer"`
	MapOrigin    string `json:"map_origin"`
	Effort       string `json:"effort,omitempty"`
	EffortLayer  string `json:"effort_layer,omitempty"`
	EffortOrigin string `json:"effort_origin,omitempty"`
	Layer        string `json:"layer"`
	Origin       string `json:"origin"`
	Status       string `json:"status"`
}

// The statuses a judge is stored with.
const (
	JudgeWaiting = "waiting"
	JudgeActive  = "active"
)

// Active reports whether the run's judge runs on a disagreement: a judge the
// run started with as active. nil, and a waiting judge, never run.
func (j *ReviewJudge) Active() bool { return j != nil && j.Status == JudgeActive }

// ReviewSet is a run's review set: each role's reviewers, the judge (nil for
// none) and the ceiling the lists were held to.
type ReviewSet struct {
	Roles        []RoleReviewers `json:"roles"`
	Judge        *ReviewJudge    `json:"judge"`
	MaxReviewers PaceValue       `json:"max_reviewers"`
}

// Role returns the reviewers of role, and false when the set names none.
func (rs ReviewSet) Role(role string) (RoleReviewers, bool) {
	for _, r := range rs.Roles {
		if r.Role == role {
			return r, true
		}
	}
	return RoleReviewers{}, false
}

// classPhrase is a class as every surface shows it: with the model it holds
// and the effort it runs at, so the class vocabulary and a vendor's names are
// never confused unseen (decision 2), and a class name is never shown alone
// (itd-2611101419006280). A class of a run started before efforts names its
// model alone.
func classPhrase(class, model, effort string) string {
	if effort == "" {
		return class + " (" + model + ")"
	}
	return class + " (" + model + ", effort " + effort + ")"
}

// phrase is classPhrase for the class.
func (c ReviewClass) phrase() string { return classPhrase(c.Class, c.Model, c.Effort) }

// phrase is classPhrase for the judge.
func (j ReviewJudge) phrase() string { return classPhrase(j.Class, j.Model, j.Effort) }

// effortOrigins names the layers that set the set's efforts, each once, in
// the order the classes are shown; "" for a run started before efforts.
func (rs ReviewSet) effortOrigins() string {
	var seen []string
	add := func(layer, origin string) {
		if layer == "" {
			return
		}
		if d := describeLayer(layer, origin); !slices.Contains(seen, d) {
			seen = append(seen, d)
		}
	}
	for _, r := range rs.Roles {
		for _, c := range r.Reviewers {
			add(c.EffortLayer, c.EffortOrigin)
		}
	}
	if j := rs.Judge; j != nil {
		add(j.EffortLayer, j.EffortOrigin)
	}
	return strings.Join(seen, " and ")
}

// judgePhrase is the judge as the surfaces show it, with its status.
func (j ReviewJudge) judgePhrase() string {
	status := j.Status
	if status == JudgeWaiting {
		status = "waiting on itd-81's calibration gate"
	}
	return "judge " + j.phrase() + ", " + status
}

// Pair is the set as a drive's start names it before it asks: each role's
// reviewers, the two roles together where they run the same classes, and the
// judge with its status.
func (rs ReviewSet) Pair() string {
	lists := make([]string, 0, len(rs.Roles))
	var roles [][]string
	for _, r := range rs.Roles {
		cs := make([]string, 0, len(r.Reviewers))
		for _, c := range r.Reviewers {
			cs = append(cs, c.phrase())
		}
		l := strings.Join(cs, " and ")
		if i := slices.Index(lists, l); i >= 0 {
			roles[i] = append(roles[i], r.Role)
			continue
		}
		lists, roles = append(lists, l), append(roles, []string{r.Role})
	}
	parts := make([]string, 0, len(lists)+1)
	for i, l := range lists {
		parts = append(parts, strings.Join(roles[i], " and ")+": "+l)
	}
	if rs.Judge == nil {
		parts = append(parts, "no judge")
	} else {
		parts = append(parts, rs.Judge.judgePhrase())
	}
	return strings.Join(parts, "; ")
}

// String renders the set and where each value came from, as the run record
// and the text surfaces name it.
func (rs ReviewSet) String() string {
	parts := make([]string, 0, len(rs.Roles)+2)
	for _, r := range rs.Roles {
		cs := make([]string, 0, len(r.Reviewers))
		for _, c := range r.Reviewers {
			cs = append(cs, c.phrase())
		}
		p := fmt.Sprintf("%s %s (%s)", r.Role, strings.Join(cs, " and "), describeLayer(r.Layer, r.Origin))
		if r.Clamped > 0 {
			p += fmt.Sprintf(", clamped from %d", r.Clamped)
		}
		parts = append(parts, p)
	}
	parts = append(parts, fmt.Sprintf("at most %d per role (%s)", rs.MaxReviewers.Value, describeOrigin(rs.MaxReviewers)))
	if rs.Judge == nil {
		parts = append(parts, "no judge")
	} else {
		parts = append(parts, fmt.Sprintf("judge %s (%s), %s", rs.Judge.phrase(),
			describeLayer(rs.Judge.Layer, rs.Judge.Origin), rs.Judge.Status))
	}
	if o := rs.effortOrigins(); o != "" {
		parts = append(parts, "efforts from "+o)
	}
	return strings.Join(parts, "; ")
}

// note is the run record's one line about the judge the run started with.
func (j ReviewJudge) note() string {
	if j.Active() {
		return fmt.Sprintf("the judge, %s, is active: a judge will run where a role's two reviewers disagree", j.phrase())
	}
	return fmt.Sprintf("the judge, %s, waits for the whole run: no judge agent has cleared itd-81's calibration gate, "+
		"so where a role's two reviewers disagree the stricter verdict decides, and no judge is launched, awaited or counted", j.phrase())
}

func describeLayer(layer, origin string) string {
	return describeOrigin(PaceValue{Layer: layer, Origin: origin})
}

// written reports whether a review set read back from a run's state is one
// abcd wrote: each role one the set governs, each list one or two known
// classes, each layer one the resolver writes, and no origin or model
// carrying a rune termsafe masks, so a hand edit is refused when it is read
// rather than echoed or run.
func (rs ReviewSet) written() bool {
	clean := func(s ...string) bool {
		for _, v := range s {
			if termsafe.Sanitize(v) != v {
				return false
			}
		}
		return true
	}
	mapLayer := func(l string) bool { return l == layered.Machine.String() || l == layered.Bundled.String() }
	if !rs.MaxReviewers.written() || rs.MaxReviewers.Value < 1 || rs.MaxReviewers.Value > maxReviewers {
		return false
	}
	if len(rs.Roles) != len(reviewRoles) {
		return false
	}
	for i, r := range rs.Roles {
		if r.Role != reviewRoles[i] || len(r.Reviewers) < 1 || len(r.Reviewers) > rs.MaxReviewers.Value ||
			!slices.Contains(paceLayers, r.Layer) || !clean(r.Origin) {
			return false
		}
		for _, c := range r.Reviewers {
			if !slices.Contains(Classes, c.Class) || !mapLayer(c.MapLayer) || !validRoute(c.Model) || !clean(c.MapOrigin) ||
				!effortWritten(c.Effort, c.EffortLayer, c.EffortOrigin) {
				return false
			}
		}
	}
	if j := rs.Judge; j != nil {
		if !slices.Contains(Classes, j.Class) || !mapLayer(j.MapLayer) || !validRoute(j.Model) ||
			!slices.Contains(paceLayers, j.Layer) || !clean(j.Origin, j.MapOrigin) ||
			(j.Status != JudgeWaiting && j.Status != JudgeActive) || !effortWritten(j.Effort, j.EffortLayer, j.EffortOrigin) {
			return false
		}
	}
	return true
}

// effortWritten reports whether a stored class's effort is one the resolver
// writes: one of the levels, from the machine or the bundled layer, with an
// origin carrying no rune termsafe masks; or none of the three, for a run
// started before efforts.
func effortWritten(effort, layer, origin string) bool {
	if effort == "" && layer == "" && origin == "" {
		return true
	}
	return slices.Contains(EffortLevels, effort) && (layer == layered.Machine.String() || layer == layered.Bundled.String()) &&
		origin != "" && termsafe.Sanitize(origin) == origin
}

// validRoute reports whether s is a <provider>/<model> route: a non-empty
// provider and model around the first slash (a model may carry its own, as an
// OpenRouter id does), with no space or control rune, bounded.
func validRoute(s string) bool {
	p, m, ok := strings.Cut(s, "/")
	if !ok || p == "" || m == "" {
		return false
	}
	for _, r := range s {
		if unicode.IsSpace(r) || unicode.IsControl(r) {
			return false
		}
	}
	return termsafe.Sanitize(s) == s && len(s) <= 200
}

// reviewForm is the accepted form every review-set refusal names.
var reviewForm = "review.reviewers takes a list of one or two classes for both roles, or an object naming such a list per role (" +
	strings.Join(reviewRoles, ", ") + "); review.judge one class; review.max_reviewers 1 or 2, in " + layered.Config.MachineOrigin() +
	" alone; a class is one of " + strings.Join(Classes, ", ") + ", which models.classes in " + layered.Config.MachineOrigin() +
	" maps to <provider>/<model> routes over the bundled map; nothing was written"

func reviewRefusal(reason string) error {
	return refuse(StageReview, "", "", reason, reviewForm)
}

// judgeGate reports whether a judge's agent has cleared itd-81's calibration
// gate, and why not. No agent abcd ships has a calibration record yet, so the
// production gate refuses every judge; a test hands a calibrated fixture
// through Options.judgeCalibrated.
func (o Options) judgeGate() (bool, string) {
	if o.judgeCalibrated != nil {
		return o.judgeCalibrated()
	}
	return false, "no judge agent has a passing calibration record under itd-81's calibration gate " +
		"(a labelled corpus of known-bad and known-good cases, scored for recall and true-negative rate)"
}

// classMap is the resolved class map: each class's route, layer and origin.
type classMap map[string]ReviewClass

// loadClassNames loads the configuration the review set reads, claims its
// keys, and refuses a file that names a class by its old name or a machine map
// that sets top without research (oldClassFault). It is the one reading every
// reader of the class map starts from, the review set's and a drive's
// (ClassRoutes), so each refuses the same files in the same words. named is
// every class the files name.
func loadClassNames(roots layered.Roots) (*layered.Stack, []namedClass, error) {
	s, err := layered.Load(layered.Config, roots)
	if err != nil {
		return nil, nil, reviewRefusal(err.Error())
	}
	if err := s.Claim(reviewNamespace, keyReviewers, keyJudge, keyMaxReviewers); err != nil {
		return nil, nil, reviewRefusal(err.Error())
	}
	if err := s.Claim(modelsNamespace, keyClasses, keyEffort); err != nil {
		return nil, nil, reviewRefusal(err.Error())
	}
	// The retired names are claimed too, so a file naming one is refused below
	// with the whole mapping rather than as a key nothing reads.
	if err := s.Claim(classNamespace, append(slices.Clone(Classes), retiredClasses...)...); err != nil {
		return nil, nil, reviewRefusal(err.Error())
	}
	named, err := classNamesIn(s)
	if err != nil {
		return nil, nil, err
	}
	if err := oldClassFault(named); err != nil {
		return nil, nil, err
	}
	return s, named, nil
}

// resolveReviewSet reads the review set and the class map through the layered
// configuration. diagnostics are its non-fatal reports, one line each, for the
// front door to print on stderr.
func resolveReviewSet(roots layered.Roots, o Options) (ReviewSet, []string, error) {
	s, named, err := loadClassNames(roots)
	if err != nil {
		return ReviewSet{}, nil, err
	}
	var diags []string
	cm, d, err := resolveClassMap(s)
	if err != nil {
		return ReviewSet{}, nil, err
	}
	diags = append(diags, d...)
	if d, err = resolveEffort(s, cm); err != nil {
		return ReviewSet{}, nil, err
	}
	diags = append(diags, d...)
	if o.ClassMapRead != nil {
		o.ClassMapRead(cm.routes())
	}
	ceiling, d, err := resolveCeiling(s)
	if err != nil {
		return ReviewSet{}, nil, err
	}
	diags = append(diags, d...)
	rs := ReviewSet{MaxReviewers: ceiling}
	clamped := map[string][]string{}
	var clampOrder []string
	for _, role := range reviewRoles {
		classes, layer, origin, err := reviewersFor(s, role)
		if err != nil {
			return ReviewSet{}, nil, err
		}
		r := RoleReviewers{Role: role, Layer: layer.String(), Origin: origin}
		for _, c := range classes {
			rc, ok := cm[c]
			if !ok {
				return ReviewSet{}, nil, unknownClass(keyReviewers, c, layer, origin)
			}
			r.Reviewers = append(r.Reviewers, rc)
		}
		if len(r.Reviewers) > ceiling.Value {
			r.Clamped = len(r.Reviewers)
			r.Reviewers = r.Reviewers[:ceiling.Value]
			if layer == layered.Repo {
				if _, seen := clamped[origin]; !seen {
					clampOrder = append(clampOrder, origin)
				}
				clamped[origin] = append(clamped[origin], role)
			}
		}
		rs.Roles = append(rs.Roles, r)
	}
	for _, origin := range clampOrder {
		diags = append(diags, fmt.Sprintf("review: %s.%s in %s names more reviewers for %s than the machine's ceiling of %d (%s.%s, %s); "+
			"the first %d run, and the rest are dropped for this run",
			reviewNamespace, keyReviewers, origin, strings.Join(clamped[origin], " and "), ceiling.Value,
			reviewNamespace, keyMaxReviewers, describeOrigin(ceiling), ceiling.Value))
	}
	j, err := layered.Get(s, reviewNamespace+"."+keyJudge, BundledJudge, nil)
	if err != nil {
		return ReviewSet{}, nil, reviewRefusal(err.Error())
	}
	rc, ok := cm[j.V]
	if !ok {
		return ReviewSet{}, nil, unknownClass(keyJudge, j.V, j.Layer, j.Origin)
	}
	status := JudgeActive
	if calibrated, why := o.judgeGate(); !calibrated {
		// A judge a person wrote into a file is one the run cannot give, so it
		// is refused; the bundled judge waits instead, since refusing it would
		// stop every run.
		if j.Layer != layered.Bundled {
			return ReviewSet{}, nil, refuse(StageReview, "", "", fmt.Sprintf("%s.%s is %q in %s (%s layer), but a judge runs only once its agent "+
				"has cleared itd-81's calibration gate, and %s", reviewNamespace, keyJudge, layered.BoundKey(j.V), j.Origin, j.Layer, why),
				fmt.Sprintf("remove %s.%s from %s; the bundled judge then waits on the gate, and the stricter verdict decides; nothing was written",
					reviewNamespace, keyJudge, j.Origin))
		}
		status = JudgeWaiting
	}
	rs.Judge = &ReviewJudge{Class: rc.Class, Model: rc.Model, MapLayer: rc.MapLayer, MapOrigin: rc.MapOrigin,
		Effort: rc.Effort, EffortLayer: rc.EffortLayer, EffortOrigin: rc.EffortOrigin,
		Layer: j.Layer.String(), Origin: j.Origin, Status: status}
	if d := topNote(named, cm); d != "" {
		diags = append(diags, d)
	}
	return rs, diags, nil
}

// renamed reports the first thing, in a state, that only version 25 and later
// write: a judge's status; "" when there is none.
func (s State) renamed() string {
	if s.Review == nil {
		return ""
	}
	if j := s.Review.Judge; j != nil {
		if j.Status != "" {
			return "a judge's status"
		}
		if j.Effort != "" || j.EffortLayer != "" || j.EffortOrigin != "" {
			return "a judge's effort"
		}
	}
	for _, r := range s.Review.Roles {
		for _, c := range r.Reviewers {
			if c.Effort != "" || c.EffortLayer != "" || c.EffortOrigin != "" {
				return "a class's effort on the " + r.Role
			}
		}
	}
	for _, l := range s.Lanes {
		for _, a := range l.Awaits {
			if a.Effort != "" {
				return "an effort on " + l.ID + "'s await of the " + a.Role
			}
		}
		for _, r := range l.Validation {
			for i, v := range r.Validators {
				if v.Effort != "" || (v.Route != nil && v.Route.EffortAsked != "") {
					return fmt.Sprintf("an effort on %s's round %d validator %d", l.ID, r.Round, i+1)
				}
			}
		}
	}
	return ""
}

// renameClasses migrates a state of version 24 or lower to the class names of
// version 25: every stored label, in the review set, its judge, each await and
// each validator, takes its new name, and a stored judge, which those versions
// kept only once its calibration had cleared, is active. It reports false, and
// the caller refuses the file, when a label is none of the four old names.
func (s *State) renameClasses() bool {
	ok := true
	rename := func(c *string) {
		if *c == "" {
			return
		}
		n, known := renamedClasses[*c]
		if !known {
			ok = false
			return
		}
		*c = n
	}
	if rs := s.Review; rs != nil {
		for i := range rs.Roles {
			for k := range rs.Roles[i].Reviewers {
				rename(&rs.Roles[i].Reviewers[k].Class)
			}
		}
		if rs.Judge != nil {
			rename(&rs.Judge.Class)
			rs.Judge.Status = JudgeActive
		}
	}
	for i := range s.Lanes {
		l := &s.Lanes[i]
		for k := range l.Awaits {
			rename(&l.Awaits[k].Class)
		}
		for r := range l.Validation {
			for v := range l.Validation[r].Validators {
				rename(&l.Validation[r].Validators[v].Class)
			}
		}
	}
	return ok
}

// namedClass is one place a configuration file names a class: the key that
// names it (models.classes.<class>, review.reviewers, review.reviewers.<role>,
// review.judge), the name, and the file's layer and origin.
type namedClass struct {
	key, name string
	layer     layered.Layer
	origin    string
}

// classNamesIn is every class a configuration file names, in a class map, a
// review list or a judge, in the order the layers are read. A value that does
// not decode is left to the reader of its key, which refuses it naming the
// file.
func classNamesIn(s *layered.Stack) ([]namedClass, error) {
	var out []namedClass
	add := func(key, name string, f layered.Found) {
		out = append(out, namedClass{key: key, name: name, layer: f.Layer, origin: f.Origin})
	}
	found, err := s.Lookup(classNamespace)
	if err != nil {
		return nil, reviewRefusal(err.Error())
	}
	for _, f := range found {
		m, err := layered.Decode[map[string]any](f.Raw)
		if err != nil {
			continue
		}
		for _, c := range slices.Sorted(maps.Keys(m)) {
			add(classNamespace+"."+c, c, f)
		}
	}
	if found, err = s.Lookup(effortKey); err != nil {
		return nil, reviewRefusal(err.Error())
	}
	for _, f := range found {
		if m, err := layered.Decode[map[string]any](f.Raw); err == nil {
			for _, c := range slices.Sorted(maps.Keys(m)) {
				add(effortKey+"."+c, c, f)
			}
		}
	}
	key := reviewNamespace + "." + keyReviewers
	if found, err = s.Lookup(key); err != nil {
		return nil, reviewRefusal(err.Error())
	}
	for _, f := range found {
		if per, err := layered.Decode[map[string][]string](f.Raw); err == nil {
			// Only a role the set governs makes a key: any other is refused
			// by reviewersFor, naming it bounded, so a file cannot lengthen
			// a refusal here by the role it writes.
			for _, role := range slices.Sorted(maps.Keys(per)) {
				if !slices.Contains(reviewRoles, role) {
					continue
				}
				for _, c := range per[role] {
					add(key+"."+role, c, f)
				}
			}
		} else if list, err := layered.Decode[[]string](f.Raw); err == nil {
			for _, c := range list {
				add(key, c, f)
			}
		}
	}
	key = reviewNamespace + "." + keyJudge
	if found, err = s.Lookup(key); err != nil {
		return nil, reviewRefusal(err.Error())
	}
	for _, f := range found {
		if c, err := layered.Decode[string](f.Raw); err == nil {
			add(key, c, f)
		}
	}
	return out, nil
}

// oldClassFault refuses a file naming a class by a name it took before the
// rename (decision 1), and a machine class map that sets top without research
// (decision 2): an old file can never hold research, so such a map may have
// been written for the old names, where top was the strongest class.
func oldClassFault(named []namedClass) error {
	for _, n := range named {
		if slices.Contains(retiredClasses, n.name) {
			return refuse(StageReview, "", "", fmt.Sprintf("%s in %s (%s layer) names class %q, a name no class takes any more: %s",
				n.key, n.origin, n.layer, layered.BoundKey(n.name), renameNote),
				fmt.Sprintf("write %q in %s for the class it meant; nothing was written", renamedClasses[n.name], n.origin))
		}
	}
	top, research := "", false
	for _, n := range named {
		if n.layer != layered.Machine || !strings.HasPrefix(n.key, classNamespace+".") {
			continue
		}
		switch n.name {
		case ClassTop:
			top = n.origin
		case ClassResearch:
			research = true
		}
	}
	if top != "" && !research {
		return refuse(StageReview, "", "", fmt.Sprintf("%s.%s in %s (machine layer) is set and %s.%s is not: `top` once named the strongest class "+
			"and now names the second (%s), so a map that sets top alone may have been written for the old names",
			classNamespace, ClassTop, top, classNamespace, ClassResearch, renameNote),
			fmt.Sprintf("a map that sets %s.%s sets %s.%s too, so the file shows which names it was written for: set %s.%s in %s to the strongest "+
				"model (the old top), and %s.%s to the second; nothing was written",
				classNamespace, ClassTop, classNamespace, ClassResearch, classNamespace, ClassResearch, top, classNamespace, ClassTop))
	}
	return nil
}

// topNote is the one line a review list or a judge naming top is announced
// with while no configuration file names research, default or small (decision
// 2): a list cannot show which names it was written for, so it is read as the
// new top and the person is told what that now stands for. "" once a file
// names a new class, or where no list or judge in a file names top. An effort
// object is new with the rename, so a file holding one, under any class name,
// top included, was written for the new names.
func topNote(named []namedClass, cm classMap) string {
	at := ""
	for _, n := range named {
		if strings.HasPrefix(n.key, effortKey+".") || (n.name != ClassTop && slices.Contains(Classes, n.name)) {
			return ""
		}
		if at == "" && n.name == ClassTop && strings.HasPrefix(n.key, reviewNamespace+".") {
			at = fmt.Sprintf("%s in %s", n.key, n.origin)
		}
	}
	if at == "" {
		return ""
	}
	return fmt.Sprintf("review: %s names `top`, which now stands for %s, the second class; the strongest class is now `research` (%s). "+
		"This line stops once a configuration file names research, default or small",
		at, cm[ClassTop].Model, cm[ClassResearch].Model)
}

// unknownClass refuses a class the class map does not carry, naming the class
// and the map's file.
func unknownClass(key, class string, layer layered.Layer, origin string) error {
	return refuse(StageReview, "", "", fmt.Sprintf("%s.%s in %s (%s layer) names class %q, which the class map does not carry: "+
		"%s in %s, over the bundled map, carries %s", reviewNamespace, key, origin, layer, layered.BoundKey(class),
		classNamespace, layered.Config.MachineOrigin(), strings.Join(Classes, ", ")), reviewForm)
}

// resolveClassMap reads models.classes from the machine's file over the
// bundled map. A repository's map is skipped with one warning.
func resolveClassMap(s *layered.Stack) (classMap, []string, error) {
	var diags []string
	cm := classMap{}
	repoNamed := false
	for _, c := range Classes {
		key := classNamespace + "." + c
		found, err := s.Lookup(key)
		if err != nil {
			return nil, nil, reviewRefusal(err.Error())
		}
		rc := ReviewClass{Class: c, Model: BundledClasses[c], MapLayer: layered.Bundled.String(), MapOrigin: layered.Bundled.String()}
		for _, f := range found {
			if f.Layer != layered.Machine {
				repoNamed = true
				continue
			}
			route, err := layered.Decode[string](f.Raw)
			if err == nil && !validRoute(route) {
				err = fmt.Errorf("want a <provider>/<model> route, a provider and a model around a slash, with no space")
			}
			if err != nil {
				return nil, nil, reviewRefusal(fmt.Sprintf("%s (%s layer): %s: %v", f.Origin, f.Layer, key, err))
			}
			rc = ReviewClass{Class: c, Model: route, MapLayer: f.Layer.String(), MapOrigin: f.Origin}
		}
		cm[c] = rc
	}
	if repoNamed {
		diags = append(diags, fmt.Sprintf("review: %s in %s is skipped: which model a class runs spends the person's key, so "+
			"the class map is this machine's alone to say; set it in %s", classNamespace, layered.Config.RepoOrigin(), layered.Config.MachineOrigin()))
	}
	return cm, diags, nil
}

// resolveEffort sets each class of cm to the effort models.effort names for
// it in the machine's file, else the bundled high. A repository's value is
// skipped with one warning; a value that is not one level, or an object
// naming a level per class, is refused naming the key, the value and what is
// accepted.
func resolveEffort(s *layered.Stack, cm classMap) ([]string, error) {
	found, err := s.Lookup(effortKey)
	if err != nil {
		return nil, reviewRefusal(err.Error())
	}
	for c, rc := range cm {
		rc.Effort, rc.EffortLayer, rc.EffortOrigin = BundledEffort, layered.Bundled.String(), layered.Bundled.String()
		cm[c] = rc
	}
	var diags []string
	accepted := fmt.Sprintf("%s takes one of %s for every class, or an object naming one per class (%s), in %s alone",
		effortKey, effortLevelsPhrase, strings.Join(Classes, ", "), layered.Config.MachineOrigin())
	for _, f := range found {
		if f.Layer != layered.Machine {
			diags = append(diags, fmt.Sprintf("review: %s in %s is skipped: a higher effort spends more of the person's budget, so how hard "+
				"each class works is this machine's alone to say; set it in %s", effortKey, f.Origin, layered.Config.MachineOrigin()))
			continue
		}
		set := func(c, level string) {
			rc := cm[c]
			rc.Effort, rc.EffortLayer, rc.EffortOrigin = level, f.Layer.String(), f.Origin
			cm[c] = rc
		}
		bad := func(key, value, why string) error {
			return refuse(StageReview, "", "", fmt.Sprintf("%s (%s layer): %s is %s: %s", f.Origin, f.Layer, key, value, why),
				accepted+"; nothing was written")
		}
		raw := strings.TrimSpace(string(f.Raw))
		if strings.HasPrefix(raw, "{") {
			per, err := layered.Decode[map[string]string](f.Raw)
			if err != nil {
				return nil, bad(effortKey, layered.BoundKey(raw), fmt.Sprintf("want an object naming one of %s per class", effortLevelsPhrase))
			}
			for _, c := range slices.Sorted(maps.Keys(per)) {
				if !slices.Contains(Classes, c) {
					return nil, bad(effortKey, layered.BoundKey(raw), fmt.Sprintf("it names %q, which is no class; an effort object is keyed by %s",
						layered.BoundKey(c), strings.Join(Classes, ", ")))
				}
				if !slices.Contains(EffortLevels, per[c]) {
					return nil, bad(effortKey+"."+c, fmt.Sprintf("%q", layered.BoundKey(per[c])), "an effort is one of "+effortLevelsPhrase)
				}
				set(c, per[c])
			}
			continue
		}
		level, err := layered.Decode[string](f.Raw)
		if err != nil || !slices.Contains(EffortLevels, level) {
			return nil, bad(effortKey, layered.BoundKey(raw), "an effort is one of "+effortLevelsPhrase)
		}
		for _, c := range Classes {
			set(c, level)
		}
	}
	return diags, nil
}

// resolveCeiling reads review.max_reviewers from the machine's file, else the
// bundled two. A repository's value is skipped with one warning.
func resolveCeiling(s *layered.Stack) (PaceValue, []string, error) {
	key := reviewNamespace + "." + keyMaxReviewers
	found, err := s.Lookup(key)
	if err != nil {
		return PaceValue{}, nil, reviewRefusal(err.Error())
	}
	v := PaceValue{Value: BundledMaxReviewers, Layer: layered.Bundled.String(), Origin: layered.Bundled.String()}
	var diags []string
	for _, f := range found {
		if f.Layer != layered.Machine {
			diags = append(diags, fmt.Sprintf("review: %s in %s is skipped: the ceiling on reviewers per role is this machine's alone "+
				"to say, and caps what a repository may ask for; set it in %s", key, f.Origin, layered.Config.MachineOrigin()))
			continue
		}
		n, err := layered.Decode[int](f.Raw)
		if err == nil && (n < 1 || n > maxReviewers) {
			err = fmt.Errorf("want 1 or 2 reviewers per role")
		}
		if err != nil {
			return PaceValue{}, nil, reviewRefusal(fmt.Sprintf("%s (%s layer): %s is %s: %v", f.Origin, f.Layer, key, strings.TrimSpace(string(f.Raw)), err))
		}
		v = PaceValue{Value: n, Layer: f.Layer.String(), Origin: f.Origin}
	}
	return v, diags, nil
}

// reviewersFor resolves one role's list: the highest layer whose
// review.reviewers is a list, or an object naming the role, else the bundled
// pair. A list that is not one or two class names, and an object naming a
// role the set does not govern, are refused naming the file.
func reviewersFor(s *layered.Stack, role string) ([]string, layered.Layer, string, error) {
	key := reviewNamespace + "." + keyReviewers
	found, err := s.Lookup(key)
	if err != nil {
		return nil, 0, "", reviewRefusal(err.Error())
	}
	for _, f := range found {
		where := fmt.Sprintf("%s (%s layer): %s", f.Origin, f.Layer, key)
		raw := strings.TrimSpace(string(f.Raw))
		var list []string
		if strings.HasPrefix(raw, "{") {
			per, err := layered.Decode[map[string][]string](f.Raw)
			if err != nil {
				return nil, 0, "", reviewRefusal(fmt.Sprintf("%s: %v", where, err))
			}
			for r := range per {
				if !slices.Contains(reviewRoles, r) {
					return nil, 0, "", reviewRefusal(fmt.Sprintf("%s names role %q; a review set governs %s, and the intent-auditor keeps its single run",
						where, layered.BoundKey(r), strings.Join(reviewRoles, " and ")))
				}
			}
			l, ok := per[role]
			if !ok {
				continue
			}
			list = l
			where += "." + role
		} else if list, err = layered.Decode[[]string](f.Raw); err != nil {
			return nil, 0, "", reviewRefusal(fmt.Sprintf("%s: %v", where, err))
		}
		if len(list) < 1 || len(list) > maxReviewers {
			return nil, 0, "", reviewRefusal(fmt.Sprintf("%s names %d reviewers; a role runs one or two", where, len(list)))
		}
		return list, f.Layer, f.Origin, nil
	}
	return slices.Clone(BundledReviewers), layered.Bundled, layered.Bundled.String(), nil
}

// CheckReviewersLaunch refuses a review set o's runners cannot launch
// (iss-2611080057001367): each reviewer of each role, at its class's model, is
// put to the runner configuration's CanLaunch, the admission the dispatch makes,
// with no host session, as a drive has none. o.Runners is the configuration the
// lanes will launch on, a drive's walled one. A drive asks it before it takes
// its checkout or opens a lane, so a reviewer no route on this machine can run
// is refused there, naming the role, the model and the configuration to set,
// rather than waiting at review for ever. A review set that does not resolve
// is refused in the review stage's own words.
func CheckReviewersLaunch(o Options, repoRoot string) error {
	if o.Runners == nil {
		return nil
	}
	review, _, err := resolveReviewSet(o.roots(repoRoot), o)
	if err != nil {
		return err
	}
	for _, r := range review.Roles {
		for _, c := range r.Reviewers {
			if err := o.Runners.CanLaunch(r.Role, c.Model, false); err != nil {
				return refuse(StageReview, "", "", fmt.Sprintf("the review set's %s reviewer for %s cannot be launched on this machine: %v",
					c.phrase(), r.Role, err), "set what the reason names in "+layered.Config.MachineOrigin()+
					", or a class map (models.classes) whose models a route here can launch; nothing was written")
			}
		}
	}
	return nil
}

// ClassRoutes is the routes the class map resolves on roots, the machine's
// models.classes over the bundled map, in class order (Classes): the routes a
// drive's walled launch of the claude harness may take by name
// (runner.Config.WithClassRoutes). A map that does not resolve is refused in
// the review stage's own words.
func ClassRoutes(roots layered.Roots) ([]string, error) {
	s, _, err := loadClassNames(roots)
	if err != nil {
		return nil, err
	}
	cm, _, err := resolveClassMap(s)
	if err != nil {
		return nil, err
	}
	return cm.routes(), nil
}

// routes are the map's routes in class order (Classes).
func (cm classMap) routes() []string {
	routes := make([]string, 0, len(Classes))
	for _, c := range Classes {
		routes = append(routes, cm[c].Model)
	}
	return routes
}
