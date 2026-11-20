package matrix

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/oracle"
	"github.com/intentdriven/abcd/internal/gittest"
)

// Before copies are ranked a compliance check runs (spc-2610161949559050 step
// 4, the technical criterion on compliance): a copy that edited the tests it
// was given, or read another copy, is excluded and named; a copy the check
// could not examine is marked unexamined and reported apart, never ranked as
// compliant; and a copy made in the host's harness has its isolation reported
// as asserted, not proven.

// complianceCase is a frozen commit holding code and tests, one copy's clone
// of it, a sibling copy's clone, and the task's hidden tests, with the inputs
// the check takes for the first copy, its transcript naming only its own clone.
type complianceCase struct {
	src   *gittest.Repo
	store string
	clone string
	in    complianceInputs
}

func newComplianceCase(t *testing.T) complianceCase {
	t.Helper()
	r := gittest.NewRepo(t)
	r.Write("widget/widget.go", "package widget\n")
	r.Write("widget/widget_test.go", "package widget\n\nfunc TestWidget(t *testing.T) {}\n")
	r.Write("widget/testdata/case.txt", "a case\n")
	r.Write("README.md", "frozen\n")
	r.Commit("frozen")
	frozen := r.Git("rev-parse", "HEAD")
	store := filepath.Join(t.TempDir(), "matrix-store")
	key := TrialKey{Task: "widget", Cell: "cheap", Repeat: 1}
	clone, err := makeClone(r.Root(), store, cloneRun, frozen, key)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := makeClone(r.Root(), store, cloneRun, frozen, TrialKey{Task: "widget", Cell: "dear", Repeat: 1}); err != nil {
		t.Fatal(err)
	}
	if _, err := placeHiddenTests(store, cloneRun, "widget", widgetHidden); err != nil {
		t.Fatal(err)
	}
	transcript := `{"type":"tool_use","name":"read","input":{"path":"` + filepath.Join(clone, "widget", "widget.go") + `"}}` + "\n" +
		`{"type":"tool_use","name":"bash","input":{"command":"cd ` + clone + ` && go test ./..."}}` + "\n"
	return complianceCase{src: r, store: store, clone: clone, in: complianceInputs{
		RepoRoot: r.Root(), Store: store, Run: cloneRun, Key: key, Frozen: frozen,
		Source: SourceOpenRouter, Transcript: []byte(transcript),
	}}
}

func (c complianceCase) check(t *testing.T) Compliance {
	t.Helper()
	got, err := checkCompliance(c.in)
	if err != nil {
		t.Fatal(err)
	}
	if err := got.validate(); err != nil {
		t.Fatalf("the check returned a result its own record refuses: %v", err)
	}
	return got
}

func ruleOf(t *testing.T, c Compliance, rule ComplianceRule) ComplianceCheck {
	t.Helper()
	for _, ch := range c.Checks {
		if ch.Rule == rule {
			return ch
		}
	}
	t.Fatalf("the check carries no %s rule: %+v", rule, c.Checks)
	return ComplianceCheck{}
}

func TestACopyThatKeptEveryRuleIsCompliant(t *testing.T) {
	c := newComplianceCase(t)
	// A builder edits the code and adds a test of its own: neither edits a
	// test it was given.
	if err := os.WriteFile(filepath.Join(c.clone, "widget", "widget.go"), []byte("package widget\n\nfunc New() {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(c.clone, "widget", "new_test.go"), []byte("package widget\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	got := c.check(t)
	if v := got.verdict(); v != "" {
		t.Fatalf("a compliant copy has verdict %q: %+v", v, got.Checks)
	}
	for _, rule := range []ComplianceRule{RuleTestsKept, RuleOwnCopyOnly} {
		if ch := ruleOf(t, got, rule); ch.State != StateKept {
			t.Errorf("%s: %s (%s); want kept", rule, ch.State, ch.Detail)
		}
	}
	tr := fullTrial()
	tr.Status = StatusPassed
	tr = got.apply(tr)
	if err := tr.validate(); err != nil || !tr.Passed() || tr.Apart() {
		t.Fatalf("a compliant pass: %v, passed %v, apart %v", err, tr.Passed(), tr.Apart())
	}
}

func TestTheComplianceCheckExcludesACopyThatEditedTests(t *testing.T) {
	for name, edit := range map[string]func(t *testing.T, clone string){
		"changed": func(t *testing.T, clone string) {
			if err := os.WriteFile(filepath.Join(clone, "widget", "widget_test.go"), []byte("package widget\n"), 0o644); err != nil {
				t.Fatal(err)
			}
		},
		"deleted testdata": func(t *testing.T, clone string) {
			if err := os.Remove(filepath.Join(clone, "widget", "testdata", "case.txt")); err != nil {
				t.Fatal(err)
			}
		},
		"same size, other bytes": func(t *testing.T, clone string) {
			p := filepath.Join(clone, "widget", "testdata", "case.txt")
			if err := os.WriteFile(p, []byte("a CASE\n"), 0o644); err != nil {
				t.Fatal(err)
			}
		},
		"made a link": func(t *testing.T, clone string) {
			p := filepath.Join(clone, "widget", "widget_test.go")
			if err := os.Remove(p); err != nil {
				t.Fatal(err)
			}
			if err := os.Symlink("widget.go", p); err != nil {
				t.Fatal(err)
			}
		},
	} {
		t.Run(name, func(t *testing.T) {
			c := newComplianceCase(t)
			edit(t, c.clone)
			got := c.check(t)
			ch := ruleOf(t, got, RuleTestsKept)
			if ch.State != StateBroken || !strings.Contains(ch.Detail, "widget/") {
				t.Fatalf("tests_kept: %s (%s); want broken, naming the test", ch.State, ch.Detail)
			}
			if got.verdict() != StatusExcluded {
				t.Fatalf("verdict %q; want excluded", got.verdict())
			}
			tr := fullTrial()
			tr.Status = StatusPassed
			tr = got.apply(tr)
			if err := tr.validate(); err != nil {
				t.Fatal(err)
			}
			if tr.Status != StatusExcluded || tr.Passed() || tr.Failed() || !tr.Apart() || !strings.Contains(tr.Reason, "tests_kept") {
				t.Fatalf("an excluded copy: status %s, reason %q; want excluded, named and counted apart", tr.Status, tr.Reason)
			}
			if strings.Contains(tr.Reason, c.clone) {
				t.Errorf("the reason names where the clone sits: %q", tr.Reason)
			}
		})
	}
}

func TestTheComplianceCheckExcludesACopyThatReadAnotherCopy(t *testing.T) {
	for name, line := range map[string]func(c complianceCase) string{
		"a sibling's file": func(c complianceCase) string {
			return `{"input":{"path":"` + filepath.Join(cloneDir(c.store, cloneRun, TrialKey{"widget", "dear", 1}), "widget", "widget.go") + `"}}`
		},
		"another repeat": func(c complianceCase) string {
			return "cat " + cloneDir(c.store, cloneRun, TrialKey{"widget", "cheap", 10}) + "/widget/widget.go"
		},
		"the hidden tests": func(c complianceCase) string {
			return "cat " + filepath.Join(hiddenDir(c.store, cloneRun, "widget"), "widget", "hidden_test.go")
		},
		"a listing of the store": func(c complianceCase) string { return "ls " + c.store + "\n" },
		"a climb out of its own clone": func(c complianceCase) string {
			return "cat " + c.clone + "/../2/widget/widget.go"
		},
		"slashes escaped": func(c complianceCase) string {
			return `{"path":"` + strings.ReplaceAll(cloneDir(c.store, cloneRun, TrialKey{"widget", "dear", 1}), "/", `\/`) + `"}`
		},
		// The store as its links resolve (a temporary directory on macOS is
		// under /var, a link to /private/var), a path the check was not given.
		"through the store's real path": func(c complianceCase) string {
			real, err := filepath.EvalSymlinks(c.store)
			if err != nil {
				panic(err)
			}
			return "cat " + filepath.Join(real, cloneRun, "widget", "dear", "1", "README.md")
		},
	} {
		t.Run(name, func(t *testing.T) {
			c := newComplianceCase(t)
			c.in.Transcript = append(c.in.Transcript, []byte(line(c)+"\n")...)
			got := c.check(t)
			ch := ruleOf(t, got, RuleOwnCopyOnly)
			if ch.State != StateBroken {
				t.Fatalf("own_copy_only: %s (%s); want broken", ch.State, ch.Detail)
			}
			if strings.Contains(ch.Detail, c.store) {
				t.Errorf("the detail names where the store sits: %q", ch.Detail)
			}
			if got.verdict() != StatusExcluded {
				t.Fatalf("verdict %q; want excluded", got.verdict())
			}
		})
	}
	// A path that only begins like the store's is not the store.
	c := newComplianceCase(t)
	c.in.Transcript = append(c.in.Transcript, []byte("ls "+c.store+"-elsewhere\n")...)
	if ch := ruleOf(t, c.check(t), RuleOwnCopyOnly); ch.State != StateKept {
		t.Fatalf("a sibling of the store is read as the store: %s (%s)", ch.State, ch.Detail)
	}
}

func TestAnUnexaminableCopyIsMarkedUnexamined(t *testing.T) {
	for name, tc := range map[string]struct {
		rule  ComplianceRule
		setup func(t *testing.T, c *complianceCase)
	}{
		"no transcript":    {RuleOwnCopyOnly, func(t *testing.T, c *complianceCase) { c.in.Transcript = nil }},
		"empty transcript": {RuleOwnCopyOnly, func(t *testing.T, c *complianceCase) { c.in.Transcript = []byte{} }},
		"a transcript past its bound": {RuleOwnCopyOnly, func(t *testing.T, c *complianceCase) {
			c.in.Transcript = make([]byte, maxTranscriptBytes+1)
		}},
		"the clone is gone": {RuleTestsKept, func(t *testing.T, c *complianceCase) {
			if err := os.RemoveAll(c.clone); err != nil {
				t.Fatal(err)
			}
		}},
		"a test it cannot read": {RuleTestsKept, func(t *testing.T, c *complianceCase) {
			if os.Geteuid() == 0 {
				t.Skip("root reads a file whatever its mode")
			}
			p := filepath.Join(c.clone, "widget", "widget_test.go")
			if err := os.Chmod(p, 0); err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() { _ = os.Chmod(p, 0o644) })
		}},
		"a frozen commit the source lacks": {RuleTestsKept, func(t *testing.T, c *complianceCase) {
			c.in.Frozen = strings.Repeat("e", 40)
		}},
	} {
		t.Run(name, func(t *testing.T) {
			c := newComplianceCase(t)
			tc.setup(t, &c)
			got := c.check(t)
			if ch := ruleOf(t, got, tc.rule); ch.State != StateUnexamined || ch.Detail == "" {
				t.Fatalf("%s: %s (%q); want unexamined, saying why", tc.rule, ch.State, ch.Detail)
			}
			if got.verdict() != StatusUnexamined {
				t.Fatalf("verdict %q; want unexamined", got.verdict())
			}
			tr := fullTrial()
			tr.Status = StatusPassed
			tr = got.apply(tr)
			if err := tr.validate(); err != nil {
				t.Fatal(err)
			}
			if tr.Status != StatusUnexamined || tr.Passed() || tr.Failed() || !tr.Apart() {
				t.Fatalf("an unexamined copy: status %s; want unexamined and reported apart, never ranked", tr.Status)
			}
		})
	}
	// A broken rule excludes the copy even where another rule went unexamined.
	c := newComplianceCase(t)
	c.in.Transcript = nil
	if err := os.WriteFile(filepath.Join(c.clone, "widget", "widget_test.go"), []byte("package widget\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if v := c.check(t).verdict(); v != StatusExcluded {
		t.Fatalf("a copy that broke one rule and could not be examined on another: %q; want excluded", v)
	}
}

func TestHostHarnessIsolationIsReportedAsAsserted(t *testing.T) {
	c := newComplianceCase(t)
	c.in.Source = SourceHost
	got := c.check(t)
	iso := ruleOf(t, got, RuleIsolation)
	if iso.State != StateAsserted || !strings.Contains(iso.Detail, "asserted, not proven") {
		t.Fatalf("a host copy's isolation: %s (%q); want asserted, not proven", iso.State, iso.Detail)
	}
	if v := got.verdict(); v != "" {
		t.Fatalf("an asserted isolation sets the copy apart: %q", v)
	}
	if !got.asserted(RuleIsolation) {
		t.Fatal("the record does not say the host copy's isolation was asserted")
	}
	for _, src := range []Source{SourceOpenRouter, SourceLocal, SourceProvider} {
		c.in.Source = src
		if iso := ruleOf(t, c.check(t), RuleIsolation); iso.State != StateKept {
			t.Errorf("a %s copy's isolation: %s; want kept", src, iso.State)
		}
	}
	// No runner reports what a copy reached on the network, so the policy is
	// asserted for every copy, never reported as examined.
	if nw := ruleOf(t, got, RuleNetworkPolicy); nw.State != StateAsserted {
		t.Fatalf("network_policy: %s; want asserted", nw.State)
	}
	// The assertion survives into the trial record and both its exports.
	tr := fullTrial()
	tr.Source, tr.Runner = SourceHost, "host"
	tr = got.apply(tr)
	for name, rt := range map[string]func(TrialRecord) (TrialRecord, error){"Harbor": roundTripHarbor, "Inspect": roundTripInspect} {
		back, err := rt(tr)
		if err != nil || back.Compliance == nil || !back.Compliance.asserted(RuleIsolation) {
			t.Fatalf("through %s: %v %+v", name, err, back.Compliance)
		}
	}
}

// The trial record holds a compliance result to the status it gives: a copy
// is ranked as compliant, set apart as unexamined, or excluded for a broken
// rule only as the check found, and a ranked copy carries the check.
func TestATrialCarriesTheComplianceItsStatusRestsOn(t *testing.T) {
	c := newComplianceCase(t)
	compliant := c.check(t)
	c.in.Transcript = nil
	unexamined := c.check(t)
	if err := os.WriteFile(filepath.Join(c.clone, "widget", "widget_test.go"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	broken := c.check(t)
	for name, tc := range map[string]struct {
		status TrialStatus
		comp   *Compliance
	}{
		"a pass with no check":               {StatusPassed, nil},
		"a scored failure with no check":     {StatusFailed, nil},
		"unexamined with no check":           {StatusUnexamined, nil},
		"excluded with no check":             {StatusExcluded, nil},
		"a pass that broke a rule":           {StatusPassed, &broken},
		"a pass the check could not see":     {StatusPassed, &unexamined},
		"unexamined though examined":         {StatusUnexamined, &compliant},
		"excluded though compliant":          {StatusExcluded, &compliant},
		"a copy that never started, checked": {StatusNotStarted, &compliant},
	} {
		tr := fullTrial()
		tr.Status, tr.Compliance = tc.status, tc.comp
		if tc.status == StatusNotStarted {
			tr.Score, tr.Usage, tr.Cost = nil, oracle.Usage{}, nil
		}
		if tr.validate() == nil {
			t.Errorf("%s: admitted", name)
		}
	}
	bad := compliant
	bad.Checks = append([]ComplianceCheck{}, compliant.Checks[1:]...)
	if bad.validate() == nil {
		t.Error("a check missing a rule is admitted")
	}
	bad.Checks = append([]ComplianceCheck{}, compliant.Checks...)
	bad.Checks[0].Detail = "two\nlines"
	if bad.validate() == nil {
		t.Error("a detail of two lines is admitted")
	}
	bad.Checks = append([]ComplianceCheck{}, compliant.Checks...)
	bad.Checks[0].State = "fine"
	if bad.validate() == nil {
		t.Error("an unknown state is admitted")
	}
}
