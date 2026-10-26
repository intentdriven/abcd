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
//     takes the highest layer that sets it, else the bundled pair (middle and
//     the class below it, the product thinker's ruling of 2026-10-24).
//   - review.judge is one class, or absent. A judge runs only once its agent
//     has cleared itd-81's calibration gate; until one has, a set naming a
//     judge is refused.
//   - review.max_reviewers is the machine's ceiling on reviewers per role,
//     one or two (two when unset). It is read from the machine's file alone:
//     a repository's list longer than the ceiling is clamped, with one line
//     on stderr naming both files, and a repository's own max_reviewers is
//     skipped with a warning.
//   - models.classes is the class map: top, middle, below and smallest, each
//     a <provider>/<model> route. Which model spends the person's key is
//     theirs alone (decision 3), so the map is read from the machine's file
//     or the bundled default alone, and a repository's map is skipped with a
//     warning, as a repository's runner route is (internal/core/runner). The
//     route's shape is checked here; its admission against the provider's
//     allowlist is the runner's, made when a runner is handed the model
//     (spec step 2), since a host-run reviewer is only asked for the class.
//
// A fault in any of them is refused at the review stage before anything is
// written. The two roles are the loop's ruthless-reviewer and
// security-reviewer; the intent-auditor keeps its single run (decision 6).

import (
	"fmt"
	"slices"
	"strings"
	"unicode"

	"github.com/intentdriven/abcd/internal/core/layered"
	"github.com/intentdriven/abcd/internal/termsafe"
)

// The model classes, top to smallest (the scale of iss-2610140652230444's
// ruling of 2026-10-14).
const (
	ClassTop      = "top"
	ClassMiddle   = "middle"
	ClassBelow    = "below"
	ClassSmallest = "smallest"
)

// Classes are the class names the class map carries, top first.
var Classes = []string{ClassTop, ClassMiddle, ClassBelow, ClassSmallest}

// BundledClasses is the bundled class map: one vendor's models, a default and
// not part of the rule (decision 3). For that vendor today the classes are
// Fable, Opus, Sonnet and Haiku (iss-2610140652230444).
var BundledClasses = map[string]string{
	ClassTop:      "anthropic/claude-fable-5",
	ClassMiddle:   "anthropic/claude-opus-5-5",
	ClassBelow:    "anthropic/claude-sonnet-5-5",
	ClassSmallest: "anthropic/claude-haiku-4-5",
}

// BundledReviewers is the bundled list each role runs: the middle class and
// the class below it, the pair the first paired reviews used (decision 1).
var BundledReviewers = []string{ClassMiddle, ClassBelow}

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
	classNamespace  = "models.classes"
)

// ReviewClass is one class of a review set with the model the class map
// resolves it to, and the map layer ("machine" or "bundled") and origin that
// supplied the model.
type ReviewClass struct {
	Class     string `json:"class"`
	Model     string `json:"model"`
	MapLayer  string `json:"map_layer"`
	MapOrigin string `json:"map_origin"`
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
// and the layer and origin of review.judge.
type ReviewJudge struct {
	Class     string `json:"class"`
	Model     string `json:"model"`
	MapLayer  string `json:"map_layer"`
	MapOrigin string `json:"map_origin"`
	Layer     string `json:"layer"`
	Origin    string `json:"origin"`
}

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

// classPhrase is a class as every surface shows it: with the model it holds,
// so the class vocabulary and a vendor's names are never confused unseen
// (decision 2).
func classPhrase(class, model string) string { return class + " (" + model + ")" }

// String renders the set and where each value came from, as the run record
// and the text surfaces name it.
func (rs ReviewSet) String() string {
	parts := make([]string, 0, len(rs.Roles)+2)
	for _, r := range rs.Roles {
		cs := make([]string, 0, len(r.Reviewers))
		for _, c := range r.Reviewers {
			cs = append(cs, classPhrase(c.Class, c.Model))
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
		parts = append(parts, fmt.Sprintf("judge %s (%s)", classPhrase(rs.Judge.Class, rs.Judge.Model), describeLayer(rs.Judge.Layer, rs.Judge.Origin)))
	}
	return strings.Join(parts, "; ")
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
			if !slices.Contains(Classes, c.Class) || !mapLayer(c.MapLayer) || !validRoute(c.Model) || !clean(c.MapOrigin) {
				return false
			}
		}
	}
	if j := rs.Judge; j != nil {
		if !slices.Contains(Classes, j.Class) || !mapLayer(j.MapLayer) || !validRoute(j.Model) ||
			!slices.Contains(paceLayers, j.Layer) || !clean(j.Origin, j.MapOrigin) {
			return false
		}
	}
	return true
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

// resolveReviewSet reads the review set and the class map through the layered
// configuration. diagnostics are its non-fatal reports, one line each, for the
// front door to print on stderr.
func resolveReviewSet(roots layered.Roots, o Options) (ReviewSet, []string, error) {
	s, err := layered.Load(layered.Config, roots)
	if err != nil {
		return ReviewSet{}, nil, reviewRefusal(err.Error())
	}
	if err := s.Claim(reviewNamespace, keyReviewers, keyJudge, keyMaxReviewers); err != nil {
		return ReviewSet{}, nil, reviewRefusal(err.Error())
	}
	if err := s.Claim(classNamespace, Classes...); err != nil {
		return ReviewSet{}, nil, reviewRefusal(err.Error())
	}
	var diags []string
	cm, d, err := resolveClassMap(s)
	if err != nil {
		return ReviewSet{}, nil, err
	}
	diags = append(diags, d...)
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
	j, err := layered.Get(s, reviewNamespace+"."+keyJudge, "", nil)
	if err != nil {
		return ReviewSet{}, nil, reviewRefusal(err.Error())
	}
	if j.Layer != layered.Bundled {
		rc, ok := cm[j.V]
		if !ok {
			return ReviewSet{}, nil, unknownClass(keyJudge, j.V, j.Layer, j.Origin)
		}
		if ok, why := o.judgeGate(); !ok {
			return ReviewSet{}, nil, refuse(StageReview, "", "", fmt.Sprintf("%s.%s is %q in %s (%s layer), but a judge runs only once its agent "+
				"has cleared itd-81's calibration gate, and %s", reviewNamespace, keyJudge, layered.BoundKey(j.V), j.Origin, j.Layer, why),
				fmt.Sprintf("remove %s.%s from %s; without a judge the stricter verdict alone decides; nothing was written", reviewNamespace, keyJudge, j.Origin))
		}
		rs.Judge = &ReviewJudge{Class: rc.Class, Model: rc.Model, MapLayer: rc.MapLayer, MapOrigin: rc.MapOrigin,
			Layer: j.Layer.String(), Origin: j.Origin}
	}
	return rs, diags, nil
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
