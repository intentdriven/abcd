package matrix

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"slices"
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

func newComplianceCase(t *testing.T, more ...func(r *gittest.Repo)) complianceCase {
	t.Helper()
	r := gittest.NewRepo(t)
	r.Write("widget/widget.go", "package widget\n")
	r.Write("widget/widget_test.go", "package widget\n\nfunc TestWidget(t *testing.T) {}\n")
	r.Write("widget/testdata/case.txt", "a case\n")
	r.Write("README.md", "frozen\n")
	for _, f := range more {
		f(r)
	}
	r.Commit("frozen")
	frozen := r.Git("rev-parse", "HEAD")
	store := filepath.Join(t.TempDir(), "matrix-store")
	key := TrialKey{Task: "widget", Cell: "cheap", Repeat: 1}
	clone, err := makeClone(r.Root(), store, cloneRun, frozen, key, keyAlias(key))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := makeClone(r.Root(), store, cloneRun, frozen, TrialKey{Task: "widget", Cell: "dear", Repeat: 1}, keyAlias(TrialKey{Task: "widget", Cell: "dear", Repeat: 1})); err != nil {
		t.Fatal(err)
	}
	if _, err := placeHiddenTests(store, cloneRun, "widget", widgetHidden); err != nil {
		t.Fatal(err)
	}
	transcript := `{"type":"tool_use","name":"read","input":{"path":"` + filepath.Join(clone, "widget", "widget.go") + `"}}` + "\n" +
		`{"type":"tool_use","name":"bash","input":{"command":"cd ` + clone + ` && go test ./..."}}` + "\n"
	return complianceCase{src: r, store: store, clone: clone, in: complianceInputs{
		RepoRoot: r.Root(), Store: store, Run: cloneRun, Key: key, Alias: keyAlias(key), Frozen: frozen,
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
			return `{"input":{"path":"` + filepath.Join(cloneDir(c.store, cloneRun, keyAlias(TrialKey{"widget", "dear", 1})), "widget", "widget.go") + `"}}`
		},
		"another repeat": func(c complianceCase) string {
			return "cat " + cloneDir(c.store, cloneRun, keyAlias(TrialKey{"widget", "cheap", 10})) + "/widget/widget.go"
		},
		"the hidden tests": func(c complianceCase) string {
			return "cat " + filepath.Join(hiddenDir(c.store, cloneRun, "widget"), "widget", "hidden_test.go")
		},
		"a listing of the store": func(c complianceCase) string { return "ls " + c.store + "\n" },
		"a climb out of its own clone": func(c complianceCase) string {
			return "cat " + c.clone + "/../2/widget/widget.go"
		},
		"slashes escaped": func(c complianceCase) string {
			return `{"path":"` + strings.ReplaceAll(cloneDir(c.store, cloneRun, keyAlias(TrialKey{"widget", "dear", 1})), "/", `\/`) + `"}`
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

// The own_copy_only scan is a heuristic, not a barrier: it reads the store in
// every spelling a transcript commonly writes a path in, not only the one the
// check was given. A home-relative path, a doubled slash, a "." segment, another
// letter case, a ".." climb written relative to the copy's working folder (from
// its clone's root or from a folder inside it) and a link the clone holds that
// leads out of it into the store are each a read outside the copy's own clone.
func TestOwnCopyOnlyReadsTheStoreInEverySpelling(t *testing.T) {
	sibling := func(c complianceCase) string {
		return cloneDir(c.store, cloneRun, keyAlias(TrialKey{"widget", "dear", 1}))
	}
	home := func(t *testing.T, c complianceCase) string {
		t.Helper()
		t.Setenv("HOME", filepath.Dir(c.store))
		return strings.TrimPrefix(sibling(c), filepath.Dir(c.store)+"/")
	}
	for name, tc := range map[string]func(t *testing.T, c complianceCase) string{
		"a tilde": func(t *testing.T, c complianceCase) string { return "cat ~/" + home(t, c) + "/README.md" },
		"$HOME":   func(t *testing.T, c complianceCase) string { return "cat $HOME/" + home(t, c) + "/README.md" },
		"${HOME}": func(t *testing.T, c complianceCase) string {
			return `{"command":"cat ${HOME}\/` + home(t, c) + `/README.md"}`
		},
		"a quoted $HOME": func(t *testing.T, c complianceCase) string {
			return `cat "$HOME"/` + home(t, c) + "/README.md"
		},
		"a quoted $HOME, JSON-escaped": func(t *testing.T, c complianceCase) string {
			return `{"command":"cat \"$HOME\"\/` + home(t, c) + `/README.md"}`
		},
		"a quoted ${HOME}": func(t *testing.T, c complianceCase) string {
			return `cat '${HOME}'/` + home(t, c) + "/README.md"
		},
		"$HOME after an option": func(t *testing.T, c complianceCase) string {
			return "cc -I$HOME/" + home(t, c) + "/widget -c x.c"
		},
		"a doubled slash": func(t *testing.T, c complianceCase) string {
			return "cat " + c.store + "//" + cloneRun + "/" + keyAlias(TrialKey{"widget", "dear", 1}) + "/README.md"
		},
		"a dot segment": func(t *testing.T, c complianceCase) string {
			return "cat " + c.store + "/./" + cloneRun + "/" + keyAlias(TrialKey{"widget", "dear", 1}) + "/README.md"
		},
		"another case": func(t *testing.T, c complianceCase) string {
			return "cat " + strings.ToUpper(sibling(c)) + "/README.md"
		},
		"a climb from the clone's root": func(t *testing.T, c complianceCase) string {
			return "cat ../" + keyAlias(TrialKey{"widget", "dear", 1}) + "/widget/widget.go"
		},
		"a climb from a folder inside it": func(t *testing.T, c complianceCase) string {
			return "cd widget && cat ../../" + keyAlias(TrialKey{"widget", "dear", 1}) + "/README.md"
		},
		"an escaped climb": func(t *testing.T, c complianceCase) string {
			return `{"command":"cat ..\/` + keyAlias(TrialKey{"widget", "dear", 1}) + `\/README.md"}`
		},
		"a climb from $PWD": func(t *testing.T, c complianceCase) string {
			return "cat $PWD/../" + keyAlias(TrialKey{"widget", "dear", 1}) + "/README.md"
		},
		"a climb to the hidden tests": func(t *testing.T, c complianceCase) string {
			return "cat ../../hidden/" + cloneRun + "/" + hiddenName(cloneRun, "widget") + "/widget/hidden_test.go"
		},
		"a link out of the clone": func(t *testing.T, c complianceCase) string {
			if err := os.Symlink(sibling(c), filepath.Join(c.clone, "peek")); err != nil {
				t.Fatal(err)
			}
			return "cat peek/widget/widget.go"
		},
		"a relative link out of the clone": func(t *testing.T, c complianceCase) string {
			if err := os.Symlink("../../"+keyAlias(TrialKey{"widget", "dear", 1}), filepath.Join(c.clone, "widget", "peek")); err != nil {
				t.Fatal(err)
			}
			return "cat widget/peek/README.md"
		},
		"a link above the store": func(t *testing.T, c complianceCase) string {
			if err := os.Symlink(filepath.Dir(c.store), filepath.Join(c.clone, "up")); err != nil {
				t.Fatal(err)
			}
			return "ls up"
		},
	} {
		t.Run(name, func(t *testing.T) {
			c := newComplianceCase(t)
			c.in.Transcript = append(c.in.Transcript, []byte(tc(t, c)+"\n")...)
			got := c.check(t)
			ch := ruleOf(t, got, RuleOwnCopyOnly)
			if ch.State != StateBroken {
				t.Fatalf("own_copy_only: %s (%s); want broken", ch.State, ch.Detail)
			}
			if strings.Contains(strings.ToLower(ch.Detail), strings.ToLower(c.store)) {
				t.Errorf("the detail names where the store sits: %q", ch.Detail)
			}
		})
	}
}

// An honest copy is not excluded for how it names its own clone: a trailing
// "." ending a sentence, a ".." that stays inside the clone, a relative climb
// from a folder inside the clone back into it, a pattern the shell or a tool
// expands, and links that stay inside the clone or lead outside the store.
// The kept detail says the scan is a heuristic.
func TestOwnCopyOnlyKeepsACopyThatStayedInItsClone(t *testing.T) {
	c := newComplianceCase(t)
	if err := os.Symlink("widget/widget.go", filepath.Join(c.clone, "alias.go")); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("/usr/bin/env", filepath.Join(c.clone, "env")); err != nil {
		t.Fatal(err)
	}
	for _, line := range []string{
		"The tests pass in " + c.clone + ".",
		"cat " + c.clone + "/.",
		"cat " + c.clone + "/widget/../README.md",
		"ls " + c.clone + "/widget/..",
		"cd widget && cat ../README.md",
		"cd widget/testdata && cat ../../README.md",
		"cd widget && go test ../...",
		"ls ..",
		"cat ~/.profile $HOME/notes.txt",
		`cat "$HOME"/notes.txt -I$HOME/include`,
	} {
		c.in.Transcript = append(c.in.Transcript, []byte(line+"\n")...)
	}
	ch := ruleOf(t, c.check(t), RuleOwnCopyOnly)
	if ch.State != StateKept {
		t.Fatalf("own_copy_only: %s (%s); want kept", ch.State, ch.Detail)
	}
	if !strings.Contains(ch.Detail, "heuristic") {
		t.Errorf("the kept detail does not say the scan is a heuristic: %q", ch.Detail)
	}
}

// tests_kept asks git whether a test file changed, as git reads the clone's
// files, rather than comparing the bytes checked out with the blob: a
// repository whose checkout rewrites line endings (an eol=crlf attribute, or
// core.autocrlf) leaves an honest copy's tests kept, and a real edit there is
// still an edit. A copy that also changed the attributes git reads them by is
// held to the bytes.
func TestTestsKeptReadsTheTestsAsGitDoes(t *testing.T) {
	crlf := func(r *gittest.Repo) { r.Write(".gitattributes", "*.go text eol=crlf\n*.txt text eol=crlf\n") }
	c := newComplianceCase(t, crlf)
	if data, _ := os.ReadFile(filepath.Join(c.clone, "widget", "widget_test.go")); !strings.Contains(string(data), "\r\n") {
		t.Fatalf("the checkout did not rewrite line endings: %q", data)
	}
	if ch := ruleOf(t, c.check(t), RuleTestsKept); ch.State != StateKept {
		t.Fatalf("an honest copy under eol=crlf: %s (%s); want kept", ch.State, ch.Detail)
	}
	if err := os.WriteFile(filepath.Join(c.clone, "widget", "testdata", "case.txt"), []byte("a CASE\r\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if ch := ruleOf(t, c.check(t), RuleTestsKept); ch.State != StateBroken || !strings.Contains(ch.Detail, "case.txt") {
		t.Fatalf("an edit under eol=crlf: %s (%s); want broken, naming it", ch.State, ch.Detail)
	}

	// A copy that rewrote the attributes is held to the bytes.
	c = newComplianceCase(t, crlf)
	if err := os.WriteFile(filepath.Join(c.clone, ".gitattributes"), []byte("*.go text eol=crlf\n*.txt -text\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if ch := ruleOf(t, c.check(t), RuleTestsKept); ch.State != StateBroken {
		t.Fatalf("a copy that rewrote the attributes: %s (%s); want broken", ch.State, ch.Detail)
	}

	// core.autocrlf in the clone's own configuration, its tests checked out
	// with CRLF line endings as that setting writes them.
	c = newComplianceCase(t)
	if _, err := git("-C", c.clone, "config", "core.autocrlf", "true"); err != nil {
		t.Fatal(err)
	}
	p := filepath.Join(c.clone, "widget", "widget_test.go")
	data, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(strings.ReplaceAll(string(data), "\n", "\r\n")), 0o644); err != nil {
		t.Fatal(err)
	}
	if ch := ruleOf(t, c.check(t), RuleTestsKept); ch.State != StateKept {
		t.Fatalf("an honest copy under core.autocrlf: %s (%s); want kept", ch.State, ch.Detail)
	}
}

// A copy that failed stays its cell's failure when the check could not
// examine it (a runner that crashed left no transcript): the failure is never
// set apart as unexamined, so it still counts against the model. A broken
// rule still excludes it.
func TestAFailedCopyTheCheckCouldNotExamineStaysAFailure(t *testing.T) {
	c := newComplianceCase(t)
	c.in.Transcript = nil
	unexamined := c.check(t)
	if unexamined.verdict() != StatusUnexamined {
		t.Fatalf("verdict %q; want unexamined", unexamined.verdict())
	}
	tr := fullTrial()
	tr.Status, tr.Score, tr.Reason = StatusFailed, nil, "the runner crashed"
	tr = unexamined.apply(tr)
	if err := tr.validate(); err != nil {
		t.Fatal(err)
	}
	if tr.Status != StatusFailed || !tr.Failed() || tr.Apart() || tr.Compliance == nil || tr.Reason != "the runner crashed" {
		t.Fatalf("a failed copy the check could not examine: status %s, reason %q, compliance %v; want failed, its reason kept, the result recorded",
			tr.Status, tr.Reason, tr.Compliance)
	}
	if err := os.WriteFile(filepath.Join(c.clone, "widget", "widget_test.go"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	tr = fullTrial()
	tr.Status, tr.Score, tr.Reason = StatusFailed, nil, "the runner crashed"
	if tr = c.check(t).apply(tr); tr.Status != StatusExcluded || tr.validate() != nil {
		t.Fatalf("a failed copy that broke a rule: %s, %v; want excluded", tr.Status, tr.validate())
	}
}

// TestOneReadOfTheFrozenTreeServesBothChecks: the compliance check and the
// review's assembly read the frozen commit's tree through one readFrozenTree,
// so each view is cut from the same listing. Every file is in the review's
// view with its mode, object and size; the test files, the attribute files and
// the links the compliance check reads are those same entries; and a read cut
// off by the caller's deadline is refused in words that name no machine path.
func TestOneReadOfTheFrozenTreeServesBothChecks(t *testing.T) {
	c := newComplianceCase(t, func(r *gittest.Repo) {
		r.Write(".gitattributes", "*.go text\n")
		if err := os.Symlink("widget.go", filepath.Join(r.Root(), "widget", "alias.go")); err != nil {
			t.Fatal(err)
		}
	})
	tree, err := readFrozenTree(context.Background(), c.in.RepoRoot, c.in.Frozen)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{".gitattributes", "README.md", "widget/alias.go", "widget/testdata/case.txt", "widget/widget.go", "widget/widget_test.go"}
	got := make([]string, 0, len(tree.files))
	for p := range tree.files {
		got = append(got, p)
	}
	slices.Sort(got)
	if !slices.Equal(got, want) {
		t.Fatalf("the review's view holds %v, want every file of the frozen commit %v", got, want)
	}
	for _, ft := range append(append([]frozenTest{}, tree.tests...), tree.attrs...) {
		if fe := tree.files[ft.path]; fe.mode != ft.mode || fe.oid != ft.oid || fe.size != ft.size {
			t.Fatalf("%s is %+v to the compliance check and %+v to the review", ft.path, ft, fe)
		}
	}
	if len(tree.tests) == 0 || len(tree.attrs) != 1 {
		t.Fatalf("the compliance check's view holds tests %v and attribute files %v", tree.tests, tree.attrs)
	}
	if oid, ok := tree.links["widget/alias.go"]; !ok || tree.files["widget/alias.go"].mode != "120000" || oid != tree.files["widget/alias.go"].oid {
		t.Fatalf("the link is %q to the compliance check and %+v to the review", oid, tree.files["widget/alias.go"])
	}
	if tree.newHash == nil {
		t.Fatal("the read names no object hash")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := readFrozenTree(ctx, c.in.RepoRoot, c.in.Frozen); err == nil || strings.Contains(err.Error(), c.in.RepoRoot) {
		t.Fatalf("a read past the caller's deadline returned %v", err)
	}
}

// TestOwnCopyOnlyReadsTheHandoffStoreOutsideTheCopysOwnFolder: a transcript
// naming another copy's folder in the run's hand-off store, where a launched
// copy's brief and answer sit, read outside its copy, in any spelling the
// store's other paths are read in; one naming only its own folder there
// stayed in its copy.
func TestOwnCopyOnlyReadsTheHandoffStoreOutsideTheCopysOwnFolder(t *testing.T) {
	c := newComplianceCase(t)
	handoff := filepath.Join(t.TempDir(), handoffDirName, "root")
	c.in.Handoff = handoff
	own := handoffDir(handoff, cloneRun, c.in.Alias)
	other := handoffDir(handoff, cloneRun, keyAlias(TrialKey{Task: "widget", Cell: "dear", Repeat: 1}))
	base := string(c.in.Transcript)
	c.in.Transcript = []byte(base + "read " + filepath.Join(own, launchBriefName) + "\nwrote " + filepath.Join(own, launchReceiptName) + "\n")
	if got := ruleOf(t, c.check(t), RuleOwnCopyOnly); got.State != StateKept {
		t.Fatalf("a copy that read its own brief folder stayed in its copy: %+v", got)
	}
	for _, read := range []string{
		filepath.Join(other, launchBriefName),
		strings.ToUpper(filepath.Join(other, launchReceiptName)[:1]) + filepath.Join(other, launchReceiptName)[1:],
		strings.ReplaceAll(filepath.Join(other, launchBriefName), "/", `\/`),
		filepath.Join(own, "..", filepath.Base(other), launchBriefName),
	} {
		c.in.Transcript = []byte(base + "cat " + read + "\n")
		got := ruleOf(t, c.check(t), RuleOwnCopyOnly)
		if got.State != StateBroken || !strings.Contains(got.Detail, "hand-off store") {
			t.Errorf("reading %s: %+v; want own_copy_only broken naming the hand-off store", read, got)
		}
	}
}

// TestOwnCopyOnlyClimbsFromTheCopysOwnBriefFolder: a launched copy is handed
// its own folder in the hand-off store, so a relative climb is read from
// there too: a copy that cd's into its own folder and reads ../<sibling>/…
// read another copy's answer, and one that climbs back into its own folder
// stayed in its copy.
func TestOwnCopyOnlyClimbsFromTheCopysOwnBriefFolder(t *testing.T) {
	c := newComplianceCase(t)
	handoff := filepath.Join(t.TempDir(), handoffDirName, "root")
	c.in.Handoff = handoff
	own := handoffDir(handoff, cloneRun, c.in.Alias)
	sibling := handoffDir(handoff, cloneRun, keyAlias(TrialKey{Task: "widget", Cell: "dear", Repeat: 1}))
	for dir, name := range map[string]string{own: launchBriefName, sibling: launchDocName} {
		if err := os.MkdirAll(dir, 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, name), []byte("text\n"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	base := string(c.in.Transcript)
	bash := func(cmd string) []byte {
		return []byte(base + `{"type":"tool_use","name":"bash","input":{"command":"` + cmd + `"}}` + "\n")
	}
	c.in.Transcript = bash("cd " + own + " && cat ../" + filepath.Base(sibling) + "/" + launchDocName)
	if got := ruleOf(t, c.check(t), RuleOwnCopyOnly); got.State != StateBroken || !strings.Contains(got.Detail, "brief folder") {
		t.Fatalf("a climb from the copy's own brief folder into a sibling's is a read outside its copy: %+v", got)
	}
	c.in.Transcript = bash("cd " + own + " && cat ../" + filepath.Base(own) + "/" + launchBriefName)
	if got := ruleOf(t, c.check(t), RuleOwnCopyOnly); got.State != StateKept {
		t.Fatalf("a climb back into its own folder stayed in its copy: %+v", got)
	}
}

// TestOwnCopyOnlyJudgesAGlobByItsLiteralPrefix: a path that holds a pattern
// names no file as written, so it is judged by what it matches: a glob that,
// climbing out of the copy's own brief folder or its clone, matches a file
// outside the copy is never kept, and unexamined where a folder of the clone
// is deep enough for the climb to stay inside; one that matches nothing
// outside the copy stayed in it.
func TestOwnCopyOnlyJudgesAGlobByItsLiteralPrefix(t *testing.T) {
	c := newComplianceCase(t)
	handoff := filepath.Join(t.TempDir(), handoffDirName, "root")
	c.in.Handoff = handoff
	own := handoffDir(handoff, cloneRun, c.in.Alias)
	sibling := handoffDir(handoff, cloneRun, keyAlias(TrialKey{Task: "widget", Cell: "dear", Repeat: 1}))
	for dir, name := range map[string]string{own: launchBriefName, sibling: launchDocName} {
		if err := os.MkdirAll(dir, 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, name), []byte("text\n"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	base := string(c.in.Transcript)
	bash := func(cmd string) []byte {
		return []byte(base + `{"type":"tool_use","name":"bash","input":{"command":"` + cmd + `"}}` + "\n")
	}
	for cmd, want := range map[string]ComplianceState{
		"cd " + own + " && cat ../*/" + launchDocName:                StateUnexamined,
		"cd " + own + " && cat ../copy_?*/" + launchDocName:          StateUnexamined,
		"cd " + c.clone + " && cat ../*/README.md":                   StateUnexamined,
		"cd " + filepath.Join(c.clone, "widget") + " && cat ../*.md": StateKept,
		// A pattern that matches nothing outside the copy reads nothing there.
		"cd " + filepath.Join(c.clone, "widget") + " && rm -f ../build/*.o":           StateKept,
		"cd " + filepath.Join(c.clone, "widget") + " && ls ../*.go":                   StateKept,
		"cd " + filepath.Join(c.clone, "widget") + " && cat ../../*/widget/widget.go": StateUnexamined,
	} {
		c.in.Transcript = bash(cmd)
		if got := ruleOf(t, c.check(t), RuleOwnCopyOnly); got.State != want {
			t.Errorf("%s: own_copy_only %s (%s); want %s", cmd, got.State, got.Detail, want)
		}
	}
}

// globHandoff gives the case a hand-off store holding the copy's own brief and
// a sibling's answer, and returns the copy's own folder there and a way to
// write one bash call into the transcript.
func globHandoff(t *testing.T, c *complianceCase, parent string) (string, func(cmd string) []byte) {
	t.Helper()
	handoff := filepath.Join(parent, handoffDirName, "root")
	c.in.Handoff = handoff
	own := handoffDir(handoff, cloneRun, c.in.Alias)
	sibling := handoffDir(handoff, cloneRun, keyAlias(TrialKey{Task: "widget", Cell: "dear", Repeat: 1}))
	for dir, name := range map[string]string{own: launchBriefName, sibling: launchDocName} {
		if err := os.MkdirAll(dir, 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, name), []byte("text\n"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	base := string(c.in.Transcript)
	return own, func(cmd string) []byte {
		return []byte(base + `{"type":"tool_use","name":"bash","input":{"command":"` + cmd + `"}}` + "\n")
	}
}

// TestOwnCopyOnlyJudgesAGlobsOutsideMatchesFirst: a pattern that matches
// another copy's files is never kept, though from some folder of the clone it
// matches a file inside it too: then which it read cannot be told, and the
// copy is unexamined; ** matches any number of folders. A pattern that matches
// nothing outside the copy stayed in it.
func TestOwnCopyOnlyJudgesAGlobsOutsideMatchesFirst(t *testing.T) {
	c := newComplianceCase(t, func(r *gittest.Repo) { r.Write("docs/guide.md", "a guide\n") })
	own, bash := globHandoff(t, &c, t.TempDir())
	widget := filepath.Join(c.clone, "widget")
	for cmd, want := range map[string]ComplianceState{
		// docs/guide.md: from the clone's root the same pattern reads
		// inside it, so which it read cannot be told.
		"cd " + own + " && cat ../*/*.md":           StateUnexamined,
		"cd " + c.clone + " && cat ../*/*":          StateUnexamined,
		"cd " + c.clone + " && ls ../*":             StateUnexamined,
		"cd " + c.clone + " && cat ../**/widget.go": StateUnexamined,
		"cd " + widget + " && rm -f ../build/*.o":   StateKept,
		"cd " + widget + " && ls ../*.go":           StateKept,
	} {
		c.in.Transcript = bash(cmd)
		if got := ruleOf(t, c.check(t), RuleOwnCopyOnly); got.State != want {
			t.Errorf("%s: own_copy_only %s (%s); want %s", cmd, got.State, got.Detail, want)
		}
	}
}

// TestOwnCopyOnlyJoinsAPatternToItsFolderAsWritten: the folder a climbing
// pattern is matched from is taken as written, so a store whose path holds a
// pattern's own characters ([ or *) still has a sibling's files matched.
func TestOwnCopyOnlyJoinsAPatternToItsFolderAsWritten(t *testing.T) {
	c := newComplianceCase(t)
	parent := filepath.Join(t.TempDir(), "a[b*c")
	if err := os.MkdirAll(parent, 0o700); err != nil {
		t.Fatal(err)
	}
	moved := filepath.Join(parent, "matrix-store")
	if err := os.Rename(c.store, moved); err != nil {
		t.Fatal(err)
	}
	c.store, c.in.Store = moved, moved
	c.clone = cloneDir(moved, cloneRun, c.in.Alias)
	own, bash := globHandoff(t, &c, parent)
	sibling := cloneDir(moved, cloneRun, keyAlias(TrialKey{Task: "widget", Cell: "dear", Repeat: 1}))
	if err := os.WriteFile(filepath.Join(sibling, "answer.txt"), []byte("a sibling's own work\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	for cmd, want := range map[string]ComplianceState{
		"cd " + own + " && cat ../*/" + launchDocName:               StateUnexamined,
		"cd " + c.clone + " && cat ../*/answer.txt":                 StateUnexamined,
		"cd " + filepath.Join(c.clone, "widget") + " && ls ../*.go": StateKept,
	} {
		c.in.Transcript = bash(cmd)
		if got := ruleOf(t, c.check(t), RuleOwnCopyOnly); got.State != want {
			t.Errorf("%s: own_copy_only %s (%s); want %s", cmd, got.State, got.Detail, want)
		}
	}
}

// TestOwnCopyOnlyBoundsWhatAPatternWalks: the check reads at most
// maxGlobEntries folder entries for one pattern and judges at most
// maxGlobPatterns patterns of one transcript, and a copy past either bound is
// unexamined, naming the bound, never kept on a walk it did not finish.
func TestOwnCopyOnlyBoundsWhatAPatternWalks(t *testing.T) {
	t.Run("entries", func(t *testing.T) {
		c := newComplianceCase(t)
		sibling := cloneDir(c.store, cloneRun, keyAlias(TrialKey{Task: "widget", Cell: "dear", Repeat: 1}))
		for i := range maxGlobEntries + 1 {
			if err := os.WriteFile(filepath.Join(sibling, fmt.Sprintf("f%05d", i)), nil, 0o600); err != nil {
				t.Fatal(err)
			}
		}
		c.in.Transcript = append(c.in.Transcript, []byte("cd "+c.clone+" && ls ../*/*.none\n")...)
		got := ruleOf(t, c.check(t), RuleOwnCopyOnly)
		if got.State != StateUnexamined || !strings.Contains(got.Detail, fmt.Sprint(maxGlobEntries)) {
			t.Fatalf("a pattern past the entries the check reads: %+v; want unexamined naming %d", got, maxGlobEntries)
		}
	})
	t.Run("patterns", func(t *testing.T) {
		c := newComplianceCase(t)
		base := c.in.Transcript
		var lines []byte
		for i := range maxGlobPatterns {
			lines = fmt.Appendf(lines, "cd %s && ls ../none%d/*.x\n", c.clone, i)
		}
		c.in.Transcript = append(slices.Clone(base), lines...)
		if got := ruleOf(t, c.check(t), RuleOwnCopyOnly); got.State != StateKept {
			t.Fatalf("%d patterns that match nothing outside: %+v; want kept", maxGlobPatterns, got)
		}
		c.in.Transcript = fmt.Appendf(c.in.Transcript, "ls ../none%d/*.x\n", maxGlobPatterns)
		got := ruleOf(t, c.check(t), RuleOwnCopyOnly)
		if got.State != StateUnexamined || !strings.Contains(got.Detail, fmt.Sprint(maxGlobPatterns)) {
			t.Fatalf("a transcript past the patterns the check judges: %+v; want unexamined naming %d", got, maxGlobPatterns)
		}
	})
}

// TestOwnCopyOnlyJudgesAGlobByWhatItsMatchExposes: the scan cannot tell the
// folder a pattern ran in. A pattern that matches outside the copy (another
// copy's brief or answer, a sibling clone's file) and that no folder of the
// clone deep enough would match inside read outside the copy; one that would
// (widget/README.md answers ../*/README.md from widget/) leaves the read
// unexamined; the run's own files beside the copies' folders, and a place
// that does not exist, match nothing outside.
func TestOwnCopyOnlyJudgesAGlobByWhatItsMatchExposes(t *testing.T) {
	c := newComplianceCase(t, func(r *gittest.Repo) { r.Write("widget/README.md", "the widget\n") })
	own, bash := globHandoff(t, &c, t.TempDir())
	if err := os.WriteFile(filepath.Join(filepath.Dir(own), launchLockName), nil, 0o600); err != nil {
		t.Fatal(err)
	}
	sibling := cloneDir(c.store, cloneRun, keyAlias(TrialKey{Task: "widget", Cell: "dear", Repeat: 1}))
	if err := os.WriteFile(filepath.Join(sibling, "answer.txt"), []byte("a sibling's own work\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	widget := filepath.Join(c.clone, "widget")
	tsconfig := append(bash("cd "+widget+" && cat tsconfig.json"),
		[]byte(`{"type":"tool_result","content":"{\"include\": [\"../shared/**/*\"]}"}`+"\n")...)
	for name, tc := range map[string]struct {
		transcript []byte
		want       ComplianceState
	}{
		"ls ../* from a folder of the clone":             {bash("cd " + widget + " && ls ../*"), StateUnexamined},
		"cat ../*/README.md from a folder of the clone":  {bash("cd " + widget + " && cat ../*/README.md"), StateUnexamined},
		"rm -rf ../dist/** from a folder of the clone":   {bash("cd " + widget + " && rm -rf ../dist/**"), StateKept},
		"a tsconfig's ../shared/**/* the copy read":      {tsconfig, StateKept},
		"../*.lock from the brief folder":                {bash("cd " + own + " && cat ../*.lock"), StateKept},
		"ls ../* beside the run's launch.lock":           {bash("cd " + own + " && ls ../*"), StateUnexamined},
		"cat ../*/*.md from the brief folder":            {bash("cd " + own + " && cat ../*/*.md"), StateUnexamined},
		"cat ../**/document.md from the brief folder":    {bash("cd " + own + " && cat ../**/" + launchDocName), StateUnexamined},
		"cat ../**/widget.go, mirrored in its own clone": {bash("cd " + own + " && cat ../**/widget.go"), StateUnexamined},
		"a file only a sibling clone holds":              {bash("cd " + c.clone + " && cat ../*/answer.txt"), StateUnexamined},
		"a class-led read of a sibling clone's file":     {bash("cd " + c.clone + " && cat ../[c]*/answer.txt"), StateUnexamined},
		"a class-led read of a sibling's answer":         {bash("cd " + own + " && cat ../[c]*/" + launchDocName), StateUnexamined},
		"a brace-led read of a sibling's answer":         {bash("cd " + own + " && cat ../{x,copy_*}/" + launchDocName), StateUnexamined},
	} {
		c.in.Transcript = tc.transcript
		if got := ruleOf(t, c.check(t), RuleOwnCopyOnly); got.State != tc.want {
			t.Errorf("%s: own_copy_only %s (%s); want %s", name, got.State, got.Detail, tc.want)
		}
	}
}

// TestOwnCopyOnlyReadsWhatOnlyAReadOutsideCouldMatch: a climbing pattern is
// matched as the shell expands it (a sequence brace, a negated or named
// class, a dot segment that matches . and .., a brace around the climb), and
// one that matches outside the copy while a folder of its clone is deep
// enough for the climb to stay inside is unexamined, as is a walk past its
// bound; a quoted * is a literal *; syntax the check does not evaluate exactly
// is never kept; and an unexamined copy is set apart, never counted with the
// copies whose blinding held.
func TestOwnCopyOnlyReadsWhatOnlyAReadOutsideCouldMatch(t *testing.T) {
	c := newComplianceCase(t, func(r *gittest.Repo) { r.Write("docs/guide/notes.md", "notes\n") })
	own, bash := globHandoff(t, &c, t.TempDir())
	sibling := cloneDir(c.store, cloneRun, keyAlias(TrialKey{Task: "widget", Cell: "dear", Repeat: 1}))
	for name, data := range map[string]string{"answer.txt": "a sibling's own work\n", "new.go": "package sibling\n"} {
		if err := os.WriteFile(filepath.Join(sibling, name), []byte(data), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.MkdirAll(filepath.Join(sibling, "bulk"), 0o700); err != nil {
		t.Fatal(err)
	}
	for i := range maxGlobEntries + 1 {
		if err := os.WriteFile(filepath.Join(sibling, "bulk", fmt.Sprintf("f%05d", i)), nil, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	guide, widget := filepath.Join(c.clone, "docs", "guide"), filepath.Join(c.clone, "widget")
	for name, tc := range map[string]struct {
		cmd  string
		want ComplianceState
	}{
		"sequence braces":                               {"cd " + own + " && cat ../copy_{0..9}*/document.md ../copy_{a..f}*/document.md", StateUnexamined},
		"a class negated with !":                        {"cd " + own + " && cat ../[!x]*/document.md", StateUnexamined},
		"a named class":                                 {"cd " + own + " && cat ../[[:alnum:]]*/document.md", StateUnexamined},
		"a dot segment":                                 {"cd " + own + " && cat ../.*/*/document.md", StateUnexamined},
		"a brace around the climb":                      {"cd " + c.clone + " && cat {..,.}/*/answer.txt", StateUnexamined},
		"a quoted * is literal":                         {"cd " + c.clone + ` && cat ../\"*\"/answer.txt ../'*'/answer.txt`, StateKept},
		"an extended pattern":                           {"cd " + own + " && cat ../!(x)/document.md", StateUnexamined},
		"ls ../*/*.md two folders deep":                 {"cd " + guide + " && ls ../*/*.md", StateUnexamined},
		"ls ../*/*.go where the sibling's tree differs": {"cd " + widget + " && ls ../*/*.go", StateUnexamined},
		"../**/x past the bound":                        {"cd " + widget + " && cat ../**/absent.txt", StateUnexamined},
	} {
		c.in.Transcript = bash(tc.cmd)
		if got := ruleOf(t, c.check(t), RuleOwnCopyOnly); got.State != tc.want {
			t.Errorf("%s: own_copy_only %s (%s); want %s", name, got.State, got.Detail, tc.want)
		}
	}
	c.in.Transcript = bash("cd " + guide + " && ls ../*/*.md")
	got := c.check(t)
	if v := got.verdict(); v != StatusUnexamined {
		t.Fatalf("an unclear pattern gives the verdict %q; want unexamined", v)
	}
	tr := fullTrial()
	tr.Status = StatusPassed
	if tr = got.apply(tr); tr.Status != StatusUnexamined || tr.Passed() || !tr.Apart() {
		t.Fatalf("an unexamined copy counts: status %s, passed %v, apart %v", tr.Status, tr.Passed(), tr.Apart())
	}
}

// TestOwnCopyOnlyReadsAClimbAsTheShellDoes: a climb is read as the shell reads
// it. A brace that does not expand ({name}) and a [ that does not close are
// literal; a quoted part joins its word as literal text and a backslash
// escapes one character, while the glob characters left unquoted stay active;
// a substitution ($name, $(...)) is judged by what comes before it, never
// kept. A pattern that matches outside the copy is unexamined while some
// folder of the clone is deep enough for the climb to stay inside it, and
// broken where none is or where it names another copy's folder as written. A
// chmod or chflags of a climb, which would open what a launch closes, is
// broken; a bare climb is kept.
func TestOwnCopyOnlyReadsAClimbAsTheShellDoes(t *testing.T) {
	sib := keyAlias(TrialKey{Task: "widget", Cell: "dear", Repeat: 1})
	t.Run("a clone with folders", func(t *testing.T) {
		c := newComplianceCase(t, func(r *gittest.Repo) { r.Write("go.mod", "module widget\n") })
		own, bash := globHandoff(t, &c, t.TempDir())
		widget := filepath.Join(c.clone, "widget")
		code := append(bash("cd "+widget+" && cat load.py"),
			[]byte(`{"type":"tool_result","content":"path = f\"../{name}/config.json\" if name else \"../[x\""}`+"\n")...)
		for name, tc := range map[string]struct {
			transcript []byte
			want       ComplianceState
		}{
			"{name} and an unclosed [ in code text": {code, StateKept},
			"ls ../*/go.mod from a folder":          {bash("cd " + widget + " && ls ../*/go.mod"), StateUnexamined},
			"a quoted part joined to a glob":        {bash("cd " + own + ` && cat ../\"copy_\"*/document.md`), StateUnexamined},
			"a quoted * is a literal *":             {bash("cd " + c.clone + ` && cat ../\"*\"/README.md`), StateKept},
			"another copy's alias, quoted":          {bash("cd " + own + ` && cat ../\"` + sib + `\"/document.md`), StateBroken},
			"another copy's alias in a pattern":     {bash("cd " + own + " && cat ../" + sib + "/*.md"), StateBroken},
			"chmod of a climb":                      {bash("cd " + own + " && chmod u+r .. && ls .."), StateBroken},
			"a bare climb, read whole":              {bash("cd " + own + " && grep -r . .."), StateKept},
		} {
			c.in.Transcript = tc.transcript
			if got := ruleOf(t, c.check(t), RuleOwnCopyOnly); got.State != tc.want {
				t.Errorf("%s: own_copy_only %s (%s); want %s", name, got.State, got.Detail, tc.want)
			}
		}
		// With the run's folders closed to listing, as a launch closes them
		// while its copies run, the joined glob matches nothing outside.
		for _, dir := range []string{filepath.Dir(own), filepath.Dir(c.clone)} {
			if err := os.Chmod(dir, 0o300); err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() { os.Chmod(dir, 0o700) })
		}
		c.in.Transcript = bash("cd " + own + ` && cat ../\"copy_\"*/document.md`)
		if got := ruleOf(t, c.check(t), RuleOwnCopyOnly); got.State != StateKept {
			t.Errorf("a glob that cannot list the run's folders: own_copy_only %s (%s); want kept", got.State, got.Detail)
		}
	})
	t.Run("a clone with no folder", func(t *testing.T) {
		c := newComplianceCase(t)
		own, bash := globHandoff(t, &c, t.TempDir())
		if err := os.RemoveAll(filepath.Join(c.clone, "widget")); err != nil {
			t.Fatal(err)
		}
		for name, cmd := range map[string]string{
			"a quoted part":          `cat ../\"copy_\"*/document.md`,
			"a backslash":            `cat ../cop\\y_*/document.md`,
			"a variable":             "p=copy_; cat ../$p*/document.md",
			"a command substitution": "cat ../$(echo copy_)*/document.md",
		} {
			c.in.Transcript = bash("cd " + own + " && " + cmd)
			if got := ruleOf(t, c.check(t), RuleOwnCopyOnly); got.State != StateBroken {
				t.Errorf("%s (%s): own_copy_only %s (%s); want broken", name, cmd, got.State, got.Detail)
			}
		}
	})
}
