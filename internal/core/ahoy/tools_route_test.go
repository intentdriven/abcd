package ahoy

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/tools"
)

func depGap(t *testing.T, gaps []Gap, id string) Gap {
	t.Helper()
	for _, g := range gaps {
		if g.ID == id {
			return g
		}
	}
	t.Fatalf("no %s gap among %v", id, gaps)
	return Gap{}
}

// TestDependencyGapCarriesTheRegistryExplanation is itd-63 criterion 1 at
// ahoy's gap: the gitleaks gap states what the registry says, not a bare
// command.
func TestDependencyGapCarriesTheRegistryExplanation(t *testing.T) {
	emptyPath(t)
	g := depGap(t, detectDependencies(t.TempDir()), "deps.gitleaks_missing")
	if g.Tool == nil || g.Tool.Tool != "gitleaks" || !g.Tool.Known {
		t.Fatalf("gap carries no registry explanation: %+v", g)
	}
	if g.Required || g.Tool.Requirement != tools.Optional {
		t.Fatalf("an unarmed repository's gitleaks gap reads as required: %+v", g)
	}
	if !strings.Contains(g.Detail, "native secret scanner") {
		t.Errorf("detail does not say what works without it: %q", g.Detail)
	}
	if !strings.Contains(g.FixHint, g.Tool.StepText()) {
		t.Errorf("fix hint %q does not carry the registry step %q", g.FixHint, g.Tool.StepText())
	}
}

// TestArmedRepositoryMakesGitleaksRequired: a repository that armed gitleaks
// needs it, and the gap says so with the way back to the native scanner.
func TestArmedRepositoryMakesGitleaksRequired(t *testing.T) {
	emptyPath(t)
	repo := t.TempDir()
	if err := os.MkdirAll(filepath.Join(repo, ".abcd", "config"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(repo, ".abcd", "config", "gitleaks.json"),
		[]byte(`{"schema_version":1,"enabled":true}`), 0o644); err != nil {
		t.Fatal(err)
	}
	g := depGap(t, detectDependencies(repo), "deps.gitleaks_missing")
	if !g.Required || g.Tool == nil || g.Tool.Capability != tools.TranscriptScanArmed {
		t.Fatalf("armed repository's gap: %+v", g)
	}
	if !strings.Contains(g.Detail, "enabled to false") {
		t.Errorf("detail does not name the way back: %q", g.Detail)
	}
}

// TestNoTrufflehogGap: nothing in abcd runs trufflehog, so ahoy never asks a
// person to install it (iss-2609261447331434).
func TestNoTrufflehogGap(t *testing.T) {
	emptyPath(t)
	for _, g := range detectDependencies(t.TempDir()) {
		if strings.Contains(g.ID, "trufflehog") {
			t.Fatalf("ahoy still offers trufflehog: %+v", g)
		}
	}
}

// fakeToolRun is a tools.Installer whose programs all "exist" outside the
// repository and whose runs are recorded rather than executed.
func fakeToolRun(t *testing.T, calls *[][]string) {
	t.Helper()
	bin := t.TempDir()
	for _, n := range []string{"brew", "gitleaks"} {
		if err := os.WriteFile(filepath.Join(bin, n), []byte("#!/bin/sh\n"), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	prev := newToolInstaller
	newToolInstaller = func(guard string) *tools.Installer {
		in := tools.Default(guard)
		in.LookPath = func(n string) (string, error) {
			p := filepath.Join(bin, n)
			if _, err := os.Stat(p); err != nil {
				return "", errors.New("absent")
			}
			return p, nil
		}
		in.Getenv = func(string) string { return "" }
		in.Run = func(_ context.Context, argv []string) ([]byte, error) {
			*calls = append(*calls, argv)
			return []byte("8.0.0\n"), nil
		}
		return in
	}
	t.Cleanup(func() { newToolInstaller = prev })
}

func depCtx(t *testing.T, confirm tools.Confirm) *applyCtx {
	t.Helper()
	emptyPath(t)
	repo := t.TempDir()
	return &applyCtx{
		cwd:         repo,
		det:         DetectionResult{Gaps: detectDependencies(repo)},
		approved:    map[GapCategory]bool{Dependency: true},
		confirmTool: confirm,
	}
}

// TestDependencyNoKeepsTheNativeDefaultAndSaysSo is criterion 2's no half at
// ahoy: nothing runs, the explanation is in the notes, and the loud line names
// what the capability continues on.
func TestDependencyNoKeepsTheNativeDefaultAndSaysSo(t *testing.T) {
	var calls [][]string
	fakeToolRun(t, &calls)
	a := depCtx(t, func(tools.Explanation) tools.Answer { return tools.Answer{Why: "answered no"} })
	a.stepDependencies()
	if len(calls) != 0 {
		t.Fatalf("a no ran %v", calls)
	}
	notes := strings.Join(a.notes, "\n")
	for _, want := range []string{
		"what abcd uses it for:",
		"install step",
		"gitleaks not installed (answered no); continuing on the native secret scanner",
	} {
		if !strings.Contains(notes, want) {
			t.Errorf("notes lack %q:\n%s", want, notes)
		}
	}
	if len(a.changes) != 0 || len(a.writes) != 0 {
		t.Errorf("a no reported a change or a write: %v %v", a.changes, a.writes)
	}
}

// TestDependencyWithoutAConfirmationNeverInstalls: a front door that supplies
// no confirmation (--yes, a caller that never asks) installs nothing.
func TestDependencyWithoutAConfirmationNeverInstalls(t *testing.T) {
	var calls [][]string
	fakeToolRun(t, &calls)
	a := depCtx(t, nil)
	a.stepDependencies()
	if len(calls) != 0 {
		t.Fatalf("installed with no confirmation: %v", calls)
	}
	if !strings.Contains(strings.Join(a.notes, "\n"), "continuing on the native secret scanner") {
		t.Fatalf("the decline is silent: %v", a.notes)
	}
}

// TestDependencyYesRunsTheStepAndReportsIt is criterion 2's yes half at ahoy.
func TestDependencyYesRunsTheStepAndReportsIt(t *testing.T) {
	var calls [][]string
	fakeToolRun(t, &calls)
	var shown tools.Explanation
	a := depCtx(t, func(e tools.Explanation) tools.Answer { shown = e; return tools.Answer{Yes: true, Why: "typed yes"} })
	a.stepDependencies()
	if shown.Tool != "gitleaks" {
		t.Fatalf("the confirmation was not shown the explanation: %+v", shown)
	}
	if len(calls) != 2 || filepath.Base(calls[0][0]) != "brew" {
		t.Fatalf("calls = %v, want the brew step and the verify", calls)
	}
	changes := strings.Join(a.changes, "\n")
	if !strings.Contains(changes, "ran brew install gitleaks") || !strings.Contains(changes, "verified") {
		t.Fatalf("the install is not reported as a change: %v (notes %v)", a.changes, a.notes)
	}
}

// TestEveryToolAhoyNamesIsRegistered is the build-time half of criterion 3: a
// tool ahoy names that the registry lacks fails here, naming it, before it can
// reach anyone as the generic explanation. DependencyTools and the gaps
// detectDependencies emits are the same set.
func TestEveryToolAhoyNamesIsRegistered(t *testing.T) {
	emptyPath(t)
	emitted := map[string]bool{}
	for _, g := range detectDependencies(t.TempDir()) {
		if g.Tool == nil {
			t.Fatalf("dependency gap %s carries no explanation", g.ID)
		}
		emitted[g.Tool.Tool] = true
	}
	for _, n := range append(append([]string{}, DependencyTools...), "gh") {
		if !tools.Known(n) {
			t.Errorf("ahoy names %q, which the tool registry does not hold", n)
		}
	}
	for _, n := range DependencyTools {
		if !emitted[n] {
			t.Errorf("DependencyTools names %q but no gap is emitted for it", n)
		}
		delete(emitted, n)
	}
	for n := range emitted {
		t.Errorf("a gap names %q, which DependencyTools omits", n)
	}
}

// TestNamedToolApprovesTheDependencyCategory: naming a tool to install is the
// answer to the dependency question, so a host relaying it with no stdin
// reaches the step; every other category is still asked (and here declined).
func TestNamedToolApprovesTheDependencyCategory(t *testing.T) {
	gaps := []Gap{
		{ID: "deps.gitleaks_missing", Category: Dependency, Resolvable: true},
		{ID: "skeleton.config_missing", Category: SafeAutocreate, Resolvable: true},
	}
	p := &recordingPrompter{}
	approved, declined := resolveApproval(gaps, InstallOptions{ApproveDependency: true}, false, p)
	asked := p.asked
	if !approved[Dependency] {
		t.Fatalf("a named tool did not approve the dependency category (declined %v)", declined)
	}
	for _, q := range asked {
		if strings.Contains(q, string(Dependency)) {
			t.Errorf("the dependency question was still asked: %v", asked)
		}
	}
	if approved[SafeAutocreate] || len(asked) != 1 {
		t.Errorf("the other categories were not left to their own answers: approved %v asked %v", approved, asked)
	}

	approved, _ = resolveApproval(gaps, InstallOptions{ApproveDependency: true, ApprovedCategories: map[GapCategory]bool{}}, false, &recordingPrompter{})
	if !approved[Dependency] {
		t.Fatal("an explicit category subset dropped the named tool's approval")
	}
}
