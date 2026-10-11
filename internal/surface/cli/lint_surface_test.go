package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/gittest"
)

// auditRepo builds a git repo at t.TempDir with the given layout knobs and
// returns its path. A conforming repo satisfies every v1 rule.
func lintRepo(t *testing.T, conforming bool) string {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not on PATH")
	}
	repo := t.TempDir()
	runGitT(t, repo, "init", "-q")
	write := func(rel, body string) {
		p := filepath.Join(repo, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write(".gitignore", ".abcd/.work.local/\n")
	write(".abcd/development/README.md", "x\n")
	write(".abcd/.work.local/NEXT.md", "x\n")
	write("AGENTS.md", "x\n")
	if conforming {
		write(".abcd/work/DECISIONS.md", "# decisions\n")
	}
	runGitT(t, repo, "add", "-A")
	runGitT(t, repo, "-c", "user.email=t@example.com", "-c", "user.name=t", "commit", "-q", "-m", "fixture")
	return repo
}

func runGitT(t *testing.T, repo string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-C", repo}, args...)...)
	cmd.Env = gittest.Env(t)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v: %s", args, err, out)
	}
}

// A conforming repo: `abcd lint` exits 0, and `--json` emits {"findings": []}.
func TestLintConformingExitsZero(t *testing.T) {
	repo := lintRepo(t, true)
	t.Chdir(repo)

	var stdout, stderr bytes.Buffer
	code := Run([]string{"lint", "--json"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("exit = %d, want 0\nstdout:%s\nstderr:%s", code, stdout.String(), stderr.String())
	}
	var res struct {
		Findings []any `json:"findings"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &res); err != nil {
		t.Fatalf("stdout not JSON: %v\n%s", err, stdout.String())
	}
	if res.Findings == nil {
		t.Error(`clean repo must emit "findings": [] (present empty array), not null`)
	}
	if len(res.Findings) != 0 {
		t.Errorf("conforming repo findings = %d, want 0", len(res.Findings))
	}
}

// A repo missing the committed work tier: exit 2, and the JSON carries the
// three-tier-layout rule id at error severity.
func TestLintMissingWorkTierExitsTwo(t *testing.T) {
	repo := lintRepo(t, false) // no .abcd/work/DECISIONS.md, so no work/ tier
	t.Chdir(repo)

	var stdout, stderr bytes.Buffer
	code := Run([]string{"lint", "--json"}, &stdout, &stderr)
	if code != 2 {
		t.Fatalf("exit = %d, want 2\nstdout:%s", code, stdout.String())
	}
	var res struct {
		Findings []struct {
			RuleID   string `json:"ruleId"`
			Severity string `json:"severity"`
		} `json:"findings"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &res); err != nil {
		t.Fatalf("stdout not JSON: %v\n%s", err, stdout.String())
	}
	found := false
	for _, f := range res.Findings {
		if f.RuleID == "three-tier-layout" && f.Severity == "error" {
			found = true
		}
	}
	if !found {
		t.Errorf("no three-tier-layout error in JSON:\n%s", stdout.String())
	}
}

// A cobra usage error (a stray positional argument, an unknown flag) must exit 2,
// not 1: lint documents Conftest's tri-state where exit 1 means "warnings only",
// so a mistyped invocation landing on 1 would let a CI gate record a clean-ish
// pass for a lint that never ran (B13). These fail before RunE, so they need no
// repo fixture.
func TestLintUsageErrorsExitTwo(t *testing.T) {
	cases := [][]string{
		{"lint", "unexpected-arg"}, // stray positional under cobra.NoArgs
		{"lint", "--nosuchflag"},   // unknown flag
	}
	for _, args := range cases {
		t.Run(strings.Join(args, " "), func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			code := Run(args, &stdout, &stderr)
			if code != 2 {
				t.Fatalf("usage error exit = %d, want 2 (must not collide with the lint tri-state's exit-1 'warnings only')\nstderr:%s", code, stderr.String())
			}
		})
	}
}

// `abcd lint --root <missing>` must report a usage error, not fabricate
// convention violations against a directory that is not there (B41).
func TestLintNonexistentRootIsUsageError(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "gone")
	var stdout, stderr bytes.Buffer
	code := Run([]string{"lint", "--root", missing}, &stdout, &stderr)
	if code != 2 {
		t.Fatalf("exit = %d, want 2\nstderr:%s", code, stderr.String())
	}
	if !strings.Contains(stderr.String(), "is not a directory") {
		t.Errorf("want an 'is not a directory' diagnostic, got stderr:\n%s", stderr.String())
	}
	if strings.Contains(stdout.String(), "conventions-router") || strings.Contains(stdout.String(), "three-tier-layout") {
		t.Errorf("lint fabricated convention findings against a missing dir:\n%s", stdout.String())
	}
}

// The human render (no --json) is grouped and readable, and stdout stays free of
// JSON braces.
func TestLintHumanRender(t *testing.T) {
	repo := lintRepo(t, false)
	t.Chdir(repo)

	var stdout, stderr bytes.Buffer
	code := Run([]string{"lint"}, &stdout, &stderr)
	if code != 2 {
		t.Fatalf("exit = %d, want 2", code)
	}
	out := stdout.String()
	if strings.Contains(out, "{") {
		t.Errorf("human render leaked JSON braces:\n%s", out)
	}
	if !strings.Contains(out, "three-tier-layout") {
		t.Errorf("human render omits the failing rule id:\n%s", out)
	}
}

// TestAuditRenameCleanBreak (spc-29): `abcd lint` is the conformance verb and
// `abcd audit` is an unknown command — the /abcd:audit seat returns to itd-16.
func TestAuditRenameCleanBreak(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := Run([]string{"audit"}, &stdout, &stderr)
	if code != 2 || !strings.Contains(stderr.String(), "unknown command") {
		t.Fatalf("abcd audit must be unknown (exit 2), got exit %d stderr %q", code, stderr.String())
	}
}

// TestLintEngineFaultExitsTwo is the S1 regression: a rule that cannot run (here a
// stat that ELOOPs on a symlink loop at a path a layout rule reads) is a check
// that never happened, not a warnings-only pass. It must exit 2 — the tri-state's
// "any error" — so a CI gate keying on >=2 fails closed, not 1, which the doc
// reserves for "warnings only" and would read as a clean-ish pass.
func TestLintEngineFaultExitsTwo(t *testing.T) {
	repo := lintRepo(t, true)
	// Replace a file a layout rule stats with a self-referential symlink: os.Stat
	// returns ELOOP (not ENOENT), which the rule propagates as an engine fault.
	if err := os.Remove(filepath.Join(repo, "AGENTS.md")); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("AGENTS.md", filepath.Join(repo, "AGENTS.md")); err != nil {
		t.Skipf("symlink unsupported: %v", err)
	}
	t.Chdir(repo)

	var stdout, stderr bytes.Buffer
	code := Run([]string{"lint"}, &stdout, &stderr)
	if code != 2 {
		t.Fatalf("engine-fault exit = %d, want 2 (must not land on the tri-state's exit-1 'warnings only')\nstdout:%s\nstderr:%s", code, stdout.String(), stderr.String())
	}
}

// TestLintSurfacesDocsTargetRefusal is the reporter's reproduction
// (iss-2610100649479892): CLAUDE.md retired while .abcd/docs-lint.json still
// names it in roots. `abcd lint docs` refuses; bare `abcd lint` once reported
// "findings": [] at exit 0 over it, so the name check checked nothing,
// silently. It must exit 2 with an error finding naming the target and its
// refusal, in both renders.
func TestLintSurfacesDocsTargetRefusal(t *testing.T) {
	repo := lintRepo(t, true)
	for rel, body := range map[string]string{
		"README.md": "# readme\n",
		".abcd/docs-lint.json": `{"roots": ["CLAUDE.md", "README.md"], ` +
			`"rules": {"links_resolve": {"enabled": true, "severity": "blocker"}}}`,
	} {
		if err := os.WriteFile(filepath.Join(repo, filepath.FromSlash(rel)), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	t.Chdir(repo)

	var stdout, stderr bytes.Buffer
	code := Run([]string{"lint", "--json"}, &stdout, &stderr)
	if code != 2 {
		t.Fatalf("exit = %d, want 2\nstdout:%s\nstderr:%s", code, stdout.String(), stderr.String())
	}
	var res struct {
		Findings []struct {
			RuleID   string `json:"ruleId"`
			Severity string `json:"severity"`
			Message  string `json:"message"`
		} `json:"findings"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &res); err != nil {
		t.Fatalf("stdout not JSON: %v\n%s", err, stdout.String())
	}
	found := false
	for _, f := range res.Findings {
		if f.RuleID == "docs-currency" && f.Severity == "error" &&
			strings.Contains(f.Message, "abcd lint docs") && strings.Contains(f.Message, `"CLAUDE.md" does not exist`) {
			found = true
		}
	}
	if !found {
		t.Fatalf("no docs-currency error naming the docs target's refusal:\n%s", stdout.String())
	}

	stdout.Reset()
	stderr.Reset()
	if code := Run([]string{"lint"}, &stdout, &stderr); code != 2 {
		t.Fatalf("human render exit = %d, want 2\n%s", code, stdout.String())
	}
	if out := stdout.String(); strings.Contains(out, "conforms") || !strings.Contains(out, "abcd lint docs") {
		t.Errorf("human render does not name the docs target's refusal:\n%s", out)
	}
}
