package testreuse

import (
	"context"
	"errors"
	"fmt"
	"path"
	"regexp"
	"slices"
	"strings"

	"github.com/intentdriven/abcd/internal/gitutil"
)

// ListPath is where the always-rerun list lives on the default branch.
const ListPath = ".abcd/work/always-rerun-tests.txt"

// Declared is one line of the always-rerun list: a package, or every package
// under one when the pattern ends in /..., optionally narrowed to named tests.
type Declared struct {
	Pattern string
	Tests   []string
	Reason  string
}

// List is the parsed always-rerun list.
type List []Declared

var (
	patternRe  = regexp.MustCompile(`^[A-Za-z0-9_][A-Za-z0-9_./-]*$`)
	testNameRe = regexp.MustCompile(`^(Test|Fuzz|Example)[A-Za-z0-9_]*$`)
)

// ParseList reads the list. One entry a line: a package path relative to the
// module, then the names of the tests when not all of them, then # and why it
// reads what the test cache cannot see. A line that does not read refuses the
// whole list, and with it all reuse: a declaration that cannot be read must not
// become a test that is reused.
func ParseList(text string) (List, error) {
	var list List
	for i, raw := range strings.Split(text, "\n") {
		line := strings.TrimSpace(raw)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		body, reason, ok := strings.Cut(line, "#")
		reason = strings.TrimSpace(reason)
		if !ok || reason == "" {
			return nil, fmt.Errorf("line %d: %q says no reason after #", i+1, line)
		}
		fields := strings.Fields(body)
		if len(fields) == 0 || !patternRe.MatchString(fields[0]) {
			return nil, fmt.Errorf("line %d: %q names no package", i+1, line)
		}
		for _, t := range fields[1:] {
			if !testNameRe.MatchString(t) {
				return nil, fmt.Errorf("line %d: %q is not a test name", i+1, t)
			}
		}
		list = append(list, Declared{Pattern: fields[0], Tests: fields[1:], Reason: reason})
	}
	return list, nil
}

// matches reports whether the entry names the package at rel, its path from the
// module root.
func (d Declared) matches(rel string) bool {
	if prefix, ok := strings.CutSuffix(d.Pattern, "/..."); ok {
		return rel == prefix || strings.HasPrefix(rel, prefix+"/")
	}
	return path.Clean(d.Pattern) == rel
}

// errNoDefaultBranch is a repository with no origin default branch to read from.
var errNoDefaultBranch = errors.New("no default branch on origin to read the always-rerun list from")

// ReadList reads the list from origin's default branch as last fetched, never
// from the working tree, so a change cannot take its own tests off it. The ref
// is local and the same account can move it, so this bounds a change's own
// diff, not an agent rewriting the ref. No ref, or no list on it, is an error,
// which the check takes as no reuse.
func ReadList(ctx context.Context, repo string) (List, string, error) {
	ref := gitutil.TrackedDefaultBranch(repo, "origin")
	if ref == "" {
		return nil, "", errNoDefaultBranch
	}
	short := gitutil.ShortRef(ref)
	out, err := gitOutput(ctx, repo, "show", ref+":"+ListPath)
	if err != nil {
		return nil, short, fmt.Errorf("%s holds no %s", short, ListPath)
	}
	list, err := ParseList(string(out))
	if err != nil {
		return nil, short, fmt.Errorf("%s:%s: %w", short, ListPath, err)
	}
	return list, short, nil
}

// PackageRoute is where a package's tests go.
type PackageRoute struct {
	Path string
	// Rerun is the reason the whole package always reruns, when it does.
	Rerun string
	// Tests are the tests that always rerun when only some do, by name, each
	// with its reason.
	Tests map[string]string
}

// Plan routes a stage's packages: those reused wherever nothing changed, those
// run whole on every check, and those split into the two by test.
type Plan struct {
	Cached []string
	Split  []PackageRoute
	Rerun  []PackageRoute
}

// MakePlan applies the list and the classifier's verdicts to pkgs, in the order
// the stage names them. rels gives each package's path from the module root.
// narrowed says the stage's own -run selects its tests, which leaves no way to
// add a second selection, so a package with any test to rerun reruns whole under
// that -run.
func MakePlan(pkgs []string, rels map[string]string, verdicts map[string]Verdict, list List, narrowed bool) Plan {
	var plan Plan
	for _, p := range pkgs {
		route := PackageRoute{Path: p, Tests: map[string]string{}}
		v := verdicts[p]
		route.Rerun = v.Whole
		for t, why := range v.Tests {
			route.Tests[t] = "undeclared: " + why
		}
		if route.Rerun != "" {
			route.Rerun = "undeclared: " + route.Rerun
		}
		for _, d := range list {
			if !d.matches(rels[p]) {
				continue
			}
			if len(d.Tests) == 0 {
				route.Rerun = "on the always-rerun list: " + d.Reason
				continue
			}
			for _, t := range d.Tests {
				route.Tests[t] = "on the always-rerun list: " + d.Reason
			}
		}
		switch {
		case route.Rerun != "":
			route.Tests = nil
			plan.Rerun = append(plan.Rerun, route)
		case len(route.Tests) == 0:
			plan.Cached = append(plan.Cached, p)
		case narrowed:
			names := route.testNames()
			route.Rerun = route.Tests[names[0]]
			route.Tests = nil
			plan.Rerun = append(plan.Rerun, route)
		default:
			plan.Split = append(plan.Split, route)
		}
	}
	return plan
}

func (r PackageRoute) testNames() []string {
	names := make([]string, 0, len(r.Tests))
	for t := range r.Tests {
		names = append(names, t)
	}
	slices.Sort(names)
	return names
}

// exactly is a -run or -skip pattern matching exactly these top-level tests.
func exactly(names []string) string {
	quoted := make([]string, len(names))
	for i, n := range names {
		quoted[i] = regexp.QuoteMeta(n)
	}
	return "^(" + strings.Join(quoted, "|") + ")$"
}
