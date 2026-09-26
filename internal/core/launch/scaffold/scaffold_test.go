package scaffold

import (
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/gittest"
)

// repoRoot returns the abcd-cli repo root from this test file's on-disk location
// (internal/core/launch/scaffold → four levels up), so the parity test runs
// against the committed .github/workflows rather than a fixture.
func repoRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", "..", "..", ".."))
}

// TestSelfScaffoldParity is the load-bearing decision-(b) test: rendering the
// shipped templates with abcd-cli's own fact set must reproduce the committed
// release.yml and auto-release.yml BYTE-FOR-BYTE. If they ever drift, the proven
// pattern and the template are no longer one artifact and this fails — so every
// abcd release exercises exactly the machinery a managed repo receives.
func TestSelfScaffoldParity(t *testing.T) {
	root := repoRoot(t)
	rendered, err := Render(AbcdSubstitutions())
	if err != nil {
		t.Fatalf("render abcd profile: %v", err)
	}
	cases := []struct {
		name string
		rel  string
		got  []byte
	}{
		{"release.yml", ReleaseYMLPath, rendered.ReleaseYML},
		{"auto-release.yml", AutoReleaseYMLPath, rendered.AutoReleaseYML},
	}
	for _, c := range cases {
		want, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(c.rel)))
		if err != nil {
			t.Fatalf("read committed %s: %v", c.rel, err)
		}
		if string(c.got) != string(want) {
			t.Errorf("%s: abcd rendering is not byte-identical to the committed workflow.\n%s",
				c.name, firstDiff(string(want), string(c.got)))
		}
	}
}

// firstDiff returns a short human pointer to the first differing line so a parity
// failure is diagnosable without dumping the whole file.
func firstDiff(want, got string) string {
	wl, gl := strings.Split(want, "\n"), strings.Split(got, "\n")
	for i := 0; i < len(wl) || i < len(gl); i++ {
		var w, g string
		if i < len(wl) {
			w = wl[i]
		}
		if i < len(gl) {
			g = gl[i]
		}
		if w != g {
			return "first diff at line " + itoa(i+1) + ":\n  committed: " + w + "\n  rendered:  " + g
		}
	}
	return "(no line diff; trailing bytes differ)"
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var b [20]byte
	i := len(b)
	for n > 0 {
		i--
		b[i] = byte('0' + n%10)
		n /= 10
	}
	return string(b[i:])
}

// TestBareRenderOmitsAbcdMachinery proves the degraded rendering a managed repo
// with no semantic detector receives: the deterministic Go gates alone, a generic
// build, no semantic gate, and no abcd-specific step — yet still a valid, gated
// workflow with the rehearsal.
func TestBareRenderOmitsAbcdMachinery(t *testing.T) {
	rendered, err := Render(BareSubstitutions("trunk"))
	if err != nil {
		t.Fatalf("render bare profile: %v", err)
	}
	rel := string(rendered.ReleaseYML)

	// The abcd-specific detectors and steps must be gone.
	for _, needle := range []string{
		"record-lint", "docs-currency-reviewer", "iss35-brief-surface-crosscheck",
		"scripts/check-reviews.sh", "make smoke", "make build", "make fmt-check", "abcd lint docs",
		"semantic-release-gate", "Cross-compile the four binaries",
		"./internal/...",
	} {
		if strings.Contains(rel, needle) {
			t.Errorf("bare release.yml must not contain abcd-specific %q", needle)
		}
	}
	// The generic build + deterministic gates + rehearsal must be present. The
	// race leg runs over the whole module, not the abcd-specific internal/ tree.
	for _, needle := range []string{
		"go build ./...", "go test -race ./...", "go-version-file: go.mod",
		`"$goroot/bin/gofmt" -l .`,
		"workflow_dispatch:", "rehearsal:", "gh release create",
	} {
		if !strings.Contains(rel, needle) {
			t.Errorf("bare release.yml must contain %q", needle)
		}
	}
	// auto-release must key on the repo's own default branch, not a hard-coded main.
	if !strings.Contains(string(rendered.AutoReleaseYML), "branches: [trunk]") {
		t.Error("bare auto-release.yml must wire the repo's own default branch")
	}
	// The bare runbook must say no semantic detector is configured (AC3).
	if !strings.Contains(string(rendered.Runbook), "No semantic detector is configured") {
		t.Error("bare runbook must state that no semantic detector is configured")
	}
}

// TestBareRunbookGateListMatchesWorkflow is the in-template gate_lockstep
// property: the runbook's numbered deterministic-gate list must be exactly the
// generic five plus the scaffolded reviews-charter shape, the one extra gate a
// managed repo inherits, so a managed repo's own lockstep check stays green.
func TestBareRunbookGateListMatchesWorkflow(t *testing.T) {
	rendered, err := Render(BareSubstitutions("main"))
	if err != nil {
		t.Fatal(err)
	}
	book := string(rendered.Runbook)
	for _, g := range []string{"1. Format (gofmt)", "5. Test (race)", "6. Reviews-charter shape (RD001)"} {
		if !strings.Contains(book, g) {
			t.Errorf("bare runbook must list deterministic gate %q", g)
		}
	}
	// The bare race leg runs over the whole module, never the abcd-specific
	// internal/ package pattern (a generic adopter may have no internal/ tree).
	if strings.Contains(book, "Test (race, internal)") {
		t.Error("bare runbook must not name the abcd-specific internal race leg")
	}
	if strings.Contains(book, "7. Plugin") || strings.Contains(book, "\n7. ") {
		t.Error("bare runbook must not number a seventh gate (the charter is the only extra gate)")
	}
}

// TestAbcdRunbookNumbersExtraGates proves the abcd runbook's numbered list is
// rendered from the same ExtraGates source as the workflow steps (the in-template
// gate_lockstep invariant), continuing 6..9.
func TestAbcdRunbookNumbersExtraGates(t *testing.T) {
	rendered, err := Render(AbcdSubstitutions())
	if err != nil {
		t.Fatal(err)
	}
	book := string(rendered.Runbook)
	for _, want := range []string{
		"6. Record-lint (design-record drift gate)",
		"7. Docs-lint (docs-currency gate)",
		"8. Reviews-charter discipline (RD001-RD004)",
		"9. Smoke every command (self-discovering harness)",
	} {
		if !strings.Contains(book, want) {
			t.Errorf("abcd runbook must number extra gate %q", want)
		}
	}
}

// TestGeneratedYAMLIsGithubTokenOnly proves both profiles run on the built-in
// token alone — no personal access token, no standing secret beyond GITHUB_TOKEN.
func TestGeneratedYAMLIsGithubTokenOnly(t *testing.T) {
	for _, subs := range []Substitutions{AbcdSubstitutions(), BareSubstitutions("main")} {
		rendered, err := Render(subs)
		if err != nil {
			t.Fatal(err)
		}
		for _, doc := range [][]byte{rendered.ReleaseYML, rendered.AutoReleaseYML} {
			s := string(doc)
			// The only secret referenced is GITHUB_TOKEN / github.token.
			for _, ref := range secretRefs(s) {
				if ref != "secrets.GITHUB_TOKEN" && ref != "github.token" {
					t.Errorf("workflow references a non-GITHUB_TOKEN secret %q (Abcd=%v)", ref, subs.Abcd)
				}
			}
			if strings.Contains(strings.ToUpper(s), "PERSONAL_ACCESS_TOKEN") || strings.Contains(s, "secrets.PAT") {
				t.Errorf("workflow references a personal access token (Abcd=%v)", subs.Abcd)
			}
		}
	}
}

// secretRefs returns every `secrets.X` and `github.token` reference in a workflow.
func secretRefs(s string) []string {
	var out []string
	for _, tok := range []string{"secrets.", "github.token"} {
		i := 0
		for {
			j := strings.Index(s[i:], tok)
			if j < 0 {
				break
			}
			start := i + j
			if tok == "github.token" {
				out = append(out, "github.token")
				i = start + len(tok)
				continue
			}
			// read secrets.<IDENT>
			k := start + len(tok)
			end := k
			for end < len(s) && (isIdent(s[end])) {
				end++
			}
			out = append(out, "secrets."+s[k:end])
			i = end
		}
	}
	return out
}

func isIdent(b byte) bool {
	return b == '_' || (b >= 'A' && b <= 'Z') || (b >= 'a' && b <= 'z') || (b >= '0' && b <= '9')
}

// TestRehearsalPublishesNothing is the AC6 property: the rehearsal job is gated to
// workflow_dispatch, holds contents: read only, and contains no publish verb — no
// Release creation, tag push, or attestation. Both profiles.
func TestRehearsalPublishesNothing(t *testing.T) {
	for _, subs := range []Substitutions{AbcdSubstitutions(), BareSubstitutions("main")} {
		rendered, err := Render(subs)
		if err != nil {
			t.Fatal(err)
		}
		job := rehearsalJob(t, string(rendered.ReleaseYML))
		if !strings.Contains(job, "if: github.event_name == 'workflow_dispatch'") {
			t.Errorf("rehearsal must be gated to workflow_dispatch (Abcd=%v)", subs.Abcd)
		}
		if !strings.Contains(job, "permissions:\n      contents: read") {
			t.Errorf("rehearsal must hold contents: read only, so it cannot publish (Abcd=%v)", subs.Abcd)
		}
		for _, forbidden := range []string{
			"gh release create", "gh release upload", "git push", "git tag",
			"actions/attest", "contents: write",
		} {
			if strings.Contains(job, forbidden) {
				t.Errorf("rehearsal must not contain publish verb %q (Abcd=%v)", forbidden, subs.Abcd)
			}
		}
	}
}

// rehearsalJob slices the `rehearsal:` job out of the workflow (from its key to
// EOF, since it is the last job).
func rehearsalJob(t *testing.T, workflow string) string {
	t.Helper()
	idx := strings.Index(workflow, "\n  rehearsal:\n")
	if idx < 0 {
		t.Fatal("no rehearsal job found in rendered release.yml")
	}
	return workflow[idx:]
}

// TestReleaseJobGatedOffRehearsal proves the real publish job never runs on a
// manual rehearsal dispatch — the guard that makes the rehearsal publish nothing.
func TestReleaseJobGatedOffRehearsal(t *testing.T) {
	rendered, err := Render(AbcdSubstitutions())
	if err != nil {
		t.Fatal(err)
	}
	s := string(rendered.ReleaseYML)
	relIdx := strings.Index(s, "\n  release:\n")
	rehIdx := strings.Index(s, "\n  rehearsal:\n")
	if relIdx < 0 || rehIdx < 0 {
		t.Fatal("release/rehearsal jobs not found")
	}
	releaseJob := s[relIdx:rehIdx]
	if !strings.Contains(releaseJob, "if: github.event_name != 'workflow_dispatch'") {
		t.Error("the release (publish) job must be gated off workflow_dispatch so a rehearsal publishes nothing")
	}
}

// TestScaffoldIdempotentAndRefusesHandEdit exercises the AC4 write path against a
// throwaway git repo: first run writes; a re-run is a no-op; a hand-edit is
// refused and left untouched; --confirm overwrites.
func TestScaffoldIdempotentAndRefusesHandEdit(t *testing.T) {
	dir := t.TempDir()
	mustWrite(t, filepath.Join(dir, "go.mod"), "module example.com/x\n\ngo 1.22\n")
	gitInit(t, dir, "release-line")

	// First run: four files written — the two workflows, the runbook and the
	// reviews-charter check.
	rep, err := Scaffold(Request{RepoRoot: dir})
	if err != nil {
		t.Fatalf("first scaffold: %v", err)
	}
	if rep.Wrote != 4 || rep.NoOp {
		t.Fatalf("first run should write 4 files, got wrote=%d noop=%v", rep.Wrote, rep.NoOp)
	}
	// Wired to the repo's own facts.
	if rep.DefaultBranch != "release-line" || rep.GoVersion != "1.22" {
		t.Errorf("expected derived branch=release-line go=1.22, got %s / %s", rep.DefaultBranch, rep.GoVersion)
	}
	relPath := filepath.Join(dir, filepath.FromSlash(ReleaseYMLPath))
	autoPath := filepath.Join(dir, filepath.FromSlash(AutoReleaseYMLPath))
	if !contains(t, autoPath, "branches: [release-line]") {
		t.Error("auto-release.yml must be wired to the derived default branch")
	}

	// Re-run: pure no-op, nothing written.
	rep2, err := Scaffold(Request{RepoRoot: dir})
	if err != nil {
		t.Fatalf("re-run: %v", err)
	}
	if rep2.Wrote != 0 || !rep2.NoOp {
		t.Fatalf("re-run should be a no-op, got wrote=%d noop=%v", rep2.Wrote, rep2.NoOp)
	}

	// Hand-edit a scaffolded file, then re-run: it must refuse and leave the edit.
	edited := "# hand-edited by the operator\n"
	mustWrite(t, relPath, edited)
	rep3, err := Scaffold(Request{RepoRoot: dir})
	if err == nil {
		t.Fatal("scaffold over a hand-edit must return ErrScaffoldBlocked")
	}
	if rep3.Refused == 0 {
		t.Error("the hand-edited file must be refused")
	}
	if got := readFile(t, relPath); got != edited {
		t.Error("a refused file must be left byte-for-byte untouched")
	}
	// The OTHER files must not have been rewritten either (all-or-nothing).
	if rep3.Wrote != 0 {
		t.Errorf("a refusal must abort before any write, got wrote=%d", rep3.Wrote)
	}

	// --confirm overwrites the hand-edit with the machinery.
	rep4, err := Scaffold(Request{RepoRoot: dir, Confirm: true})
	if err != nil {
		t.Fatalf("confirm run: %v", err)
	}
	if rep4.Wrote != 1 {
		t.Errorf("confirm should rewrite exactly the drifted file, got wrote=%d", rep4.Wrote)
	}
	if contains(t, relPath, "hand-edited") {
		t.Error("--confirm must overwrite the hand-edit with the machinery")
	}
}

// TestRefusalAbortReportMatchesDisk pins the report-vs-disk contract on the
// all-or-nothing abort path: when one file is hand-edited (refused) and another
// is still absent, the run writes NOTHING and the report must not claim the absent
// file was written — it is reported as skipped and remains absent on disk.
func TestRefusalAbortReportMatchesDisk(t *testing.T) {
	dir := t.TempDir()
	mustWrite(t, filepath.Join(dir, "go.mod"), "module example.com/x\n\ngo 1.22\n")
	gitInit(t, dir, "main")

	// auto-release.yml exists and differs (a hand-edit); release.yml is absent.
	autoPath := filepath.Join(dir, filepath.FromSlash(AutoReleaseYMLPath))
	relPath := filepath.Join(dir, filepath.FromSlash(ReleaseYMLPath))
	mustWrite(t, autoPath, "# hand-edited auto-release\n")

	rep, err := Scaffold(Request{RepoRoot: dir})
	if err == nil {
		t.Fatal("a hand-edited sibling must abort the run with ErrScaffoldBlocked")
	}
	if rep.Wrote != 0 {
		t.Errorf("nothing must be written on the abort path, got wrote=%d", rep.Wrote)
	}
	// The absent file must NOT be reported as written, and must still be absent.
	for _, f := range rep.Files {
		if f.Path == ReleaseYMLPath {
			if f.Status == StatusWritten {
				t.Errorf("release.yml was NOT written but the report says %q", f.Status)
			}
			if f.Status != StatusSkipped {
				t.Errorf("an unwritten planned file must report as %q, got %q", StatusSkipped, f.Status)
			}
		}
	}
	if _, statErr := os.Stat(relPath); !os.IsNotExist(statErr) {
		t.Error("release.yml must remain absent on disk after an aborted run")
	}
}

// TestDeriveRepoFactsKeepsPatchVersion is iss-386: a go.mod pinning a patch
// (the shape a modern `go mod tidy` writes) must scaffold that exact toolchain,
// not a floating minor that setup-go resolves to whatever patch a runner offers
// on release day.
func TestDeriveRepoFactsKeepsPatchVersion(t *testing.T) {
	dir := t.TempDir()
	mustWrite(t, filepath.Join(dir, "go.mod"), "module example.com/x\n\ngo 1.25.6\n")
	if _, gv := DeriveRepoFacts(dir); gv != "1.25.6" {
		t.Errorf("a patch-pinned go.mod must scaffold the patch, got %q want 1.25.6", gv)
	}
}

// TestDeriveRepoFactsRejectsHostileInputs proves the substitution inputs are
// sanitised before they can reach the YAML — a crafted go.mod or ref name falls
// back to the safe default rather than injecting workflow content.
func TestDeriveRepoFactsRejectsHostileInputs(t *testing.T) {
	dir := t.TempDir()
	// A go.mod whose version carries a YAML-breaking payload.
	mustWrite(t, filepath.Join(dir, "go.mod"), "module x\n\ngo 1.25'\ninjected: true\n")
	if _, gv := DeriveRepoFacts(dir); gv != "1.25" {
		// go 1.25 is a clean prefix match; ensure only the clean major.minor is taken.
		if gv != "1.25" {
			t.Errorf("go version must be sanitised to major.minor, got %q", gv)
		}
	}
	// A hostile HEAD symref must not become the branch value.
	gitDir := filepath.Join(dir, ".git")
	mustMkdir(t, gitDir)
	mustWrite(t, filepath.Join(gitDir, "HEAD"), "ref: refs/heads/main]\ninjected: [x\n")
	if b, _ := DeriveRepoFacts(dir); b != "main" {
		t.Errorf("a hostile branch name must fall back to %q, got %q", "main", b)
	}
}

// --- helpers ---------------------------------------------------------------

func mustWrite(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func mustMkdir(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(path, 0o755); err != nil {
		t.Fatal(err)
	}
}

func readFile(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func contains(t *testing.T, path, needle string) bool {
	t.Helper()
	return strings.Contains(readFile(t, path), needle)
}

// gitInit makes dir a git repo checked out on branchName, so deriveBranch reads a
// real .git/HEAD symref.
func gitInit(t *testing.T, dir, branchName string) {
	t.Helper()
	run := func(args ...string) {
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		cmd.Env = gittest.Env(t) // hermetic git (iss-28): no user/system config leaks in
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	run("init", "-q", "-b", branchName)
}

// TestDeriveRepoFactsResolvesWorktreeGitfile is the S7 regression: in a linked
// worktree (or submodule) ".git" is a FILE holding "gitdir: <path>", not a
// directory. deriveBranch must follow it — reading HEAD from the worktree gitdir
// and origin/HEAD from the shared (commondir) git dir — instead of failing both
// raw reads and silently stamping the "main" fallback into the release workflows.
func TestDeriveRepoFactsResolvesWorktreeGitfile(t *testing.T) {
	root := t.TempDir()
	// Shared (common) git dir with the remote default branch pinned to "mainline".
	commonGit := filepath.Join(root, "common", ".git")
	mustWrite(t, filepath.Join(commonGit, "refs", "remotes", "origin", "HEAD"),
		"ref: refs/remotes/origin/mainline\n")
	// The linked worktree: ".git" is a gitfile; its gitdir carries HEAD (on a
	// different branch) and a commondir pointer back to the shared dir.
	wt := filepath.Join(root, "wt")
	wtGitDir := filepath.Join(commonGit, "worktrees", "wt")
	mustWrite(t, filepath.Join(wtGitDir, "HEAD"), "ref: refs/heads/feature\n")
	mustWrite(t, filepath.Join(wtGitDir, "commondir"), "../..\n")
	mustWrite(t, filepath.Join(wt, ".git"), "gitdir: "+wtGitDir+"\n")

	if b, _ := DeriveRepoFacts(wt); b != "mainline" {
		t.Errorf("worktree default branch = %q, want %q (fell back to main = the bug)", b, "mainline")
	}

	// With no origin/HEAD, it falls back to the per-worktree checked-out branch,
	// still read through the gitfile rather than defaulting to main.
	if err := os.Remove(filepath.Join(commonGit, "refs", "remotes", "origin", "HEAD")); err != nil {
		t.Fatal(err)
	}
	if b, _ := DeriveRepoFacts(wt); b != "feature" {
		t.Errorf("worktree HEAD branch = %q, want %q", b, "feature")
	}
}

// TestClassifySymlinkLeafIsPathFree proves a symlinked target leaf is refused
// with the path-free non-regular reason (folding ELOOP), never the generic
// fallback that would embed a developer-identity absolute path in the scaffold
// report and its --json.
func TestClassifySymlinkLeafIsPathFree(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "release.yml")
	if err := os.Symlink(filepath.Join(dir, "elsewhere"), target); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	disp, reason := classify(target, []byte("machinery"))
	if disp != dispDiffers {
		t.Fatalf("symlink leaf classified %v, want dispDiffers", disp)
	}
	if !strings.Contains(reason, "not a regular file") {
		t.Errorf("reason is not the non-regular message: %q", reason)
	}
	if strings.Contains(reason, dir) {
		t.Errorf("reason leaks the absolute path: %q", reason)
	}
}

// TestRunbookGateListMatchesVerifySteps is the gate_lockstep invariant over
// every profile the templates render: the runbook's numbered deterministic-gate
// list is exactly the verify job's steps, in order, less the setup steps and the
// semantic receipts step (which the runbook describes in its own section and
// abcd's gate_lockstep config ignores). A verify gate added to one side only,
// such as the tag binding (iss-2609251945586202), fails here in every profile,
// not only in abcd's own README.
func TestRunbookGateListMatchesVerifySteps(t *testing.T) {
	semantic := BareSubstitutions("main")
	semantic.SemanticGates = []string{"docs-currency-reviewer"}
	notGates := map[string]bool{
		"Check out the pushed commit": true,
		"Set up Go":                   true,
		"Semantic-gate receipts (fail-closed, before tag)": true,
	}
	itemRe := regexp.MustCompile(`^(\d+)\. (.+)$`)
	for name, subs := range map[string]Substitutions{
		"abcd": AbcdSubstitutions(), "bare": BareSubstitutions("main"), "bare+semantic": semantic,
	} {
		rendered, err := Render(subs)
		if err != nil {
			t.Fatal(err)
		}
		var steps []string
		for _, line := range strings.Split(jobSection(t, string(rendered.ReleaseYML), "verify"), "\n") {
			if s, ok := strings.CutPrefix(strings.TrimSpace(line), "- name: "); ok && !notGates[s] {
				steps = append(steps, s)
			}
		}
		var listed []string
		in := false
		for _, line := range strings.Split(string(rendered.Runbook), "\n") {
			if strings.HasPrefix(line, "#") {
				in = strings.Contains(strings.ToLower(line), "deterministic gate")
				continue
			}
			if m := itemRe.FindStringSubmatch(line); in && m != nil {
				if m[1] != strconv.Itoa(len(listed)+1) {
					t.Errorf("%s: runbook item %q is numbered %s, want %d", name, m[2], m[1], len(listed)+1)
				}
				listed = append(listed, m[2])
			}
		}
		if strings.Join(listed, "\n") != strings.Join(steps, "\n") {
			t.Errorf("%s: runbook deterministic gates\n  %q\nare not the verify job's gate steps\n  %q", name, listed, steps)
		}
	}
}
