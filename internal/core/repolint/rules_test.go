package repolint_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/repolint"
	"github.com/intentdriven/abcd/internal/gittest"
)

// --- fixture repo builder ---------------------------------------------------

type repoBuilder struct {
	t    *testing.T
	root string
}

// newFixtureRepo starts a git repo with an isolated identity. Chain With* calls
// to lay out tiers, then Commit() to make the tracked set real.
func newFixtureRepo(t *testing.T) *repoBuilder {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not on PATH")
	}
	root := t.TempDir()
	git(t, root, "init", "-q")
	return &repoBuilder{t: t, root: root}
}

func git(t *testing.T, root string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-C", root}, args...)...)
	cmd.Env = gittest.Env(t)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v: %s", args, err, out)
	}
}

func (b *repoBuilder) file(rel, body string) *repoBuilder {
	b.t.Helper()
	p := filepath.Join(b.root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		b.t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
		b.t.Fatal(err)
	}
	return b
}

func (b *repoBuilder) dir(rel string) *repoBuilder {
	b.t.Helper()
	if err := os.MkdirAll(filepath.Join(b.root, filepath.FromSlash(rel)), 0o755); err != nil {
		b.t.Fatal(err)
	}
	return b
}

// conforming lays out a repo that satisfies every v1 rule: all three tiers,
// .work.local gitignored, an AGENTS.md router, and a committed DECISIONS.md.
func (b *repoBuilder) conforming() *repoBuilder {
	return b.
		file(".gitignore", ".abcd/.work.local/\n").
		file(".abcd/development/README.md", "# durable record\n").
		file(".abcd/work/DECISIONS.md", "# decisions\n- 2026-07-13: a thing.\n").
		file(".abcd/.work.local/NEXT.md", "# local handoff\n").
		file("AGENTS.md", "# conventions\n")
}

func (b *repoBuilder) commit() *repoBuilder {
	b.t.Helper()
	git(b.t, b.root, "add", "-A")
	git(b.t, b.root, "-c", "user.email=t@example.com", "-c", "user.name=t", "commit", "-q", "-m", "fixture")
	return b
}

func (b *repoBuilder) run() repolint.Result {
	b.t.Helper()
	res, err := repolint.Evaluate(repolint.DefaultRules(), repolint.Context{RepoRoot: b.root})
	if err != nil {
		b.t.Fatalf("Evaluate: %v", err)
	}
	return res
}

func findingFor(res repolint.Result, ruleID string) *repolint.Finding {
	for i := range res.Findings {
		if res.Findings[i].RuleID == ruleID {
			return &res.Findings[i]
		}
	}
	return nil
}

// --- acceptance criteria (itd-85) -------------------------------------------

// AC4: a conforming repo exits 0 with no findings.
func TestAC_ConformingRepoClean(t *testing.T) {
	res := newFixtureRepo(t).conforming().commit().run()
	if len(res.Findings) != 0 {
		t.Fatalf("conforming repo has findings: %+v", res.Findings)
	}
	if res.ExitCode != 0 {
		t.Errorf("exit = %d, want 0", res.ExitCode)
	}
}

// AC1: a repo missing .abcd/work/ → three-tier-layout error, exit 2, fix names
// the missing tier.
func TestAC_MissingWorkTier(t *testing.T) {
	b := newFixtureRepo(t).
		file(".gitignore", ".abcd/.work.local/\n").
		file(".abcd/development/README.md", "x\n").
		file(".abcd/.work.local/NEXT.md", "x\n").
		file("AGENTS.md", "x\n").
		commit()
	res := b.run()

	f := findingFor(res, "three-tier-layout")
	if f == nil {
		t.Fatal("no three-tier-layout finding for a repo missing .abcd/work/")
	}
	if f.Severity != repolint.SeverityError {
		t.Errorf("severity = %q, want error", f.Severity)
	}
	if !strings.Contains(f.Message+f.Fix, "work") {
		t.Errorf("finding does not name the missing work/ tier: %q / %q", f.Message, f.Fix)
	}
	if res.ExitCode != 2 {
		t.Errorf("exit = %d, want 2", res.ExitCode)
	}
}

// AC2: decisions living only in the gitignored layer → decision-durability warn,
// exit 1 (no errors).
func TestAC_DecisionsOnlyInGitignoredLayer(t *testing.T) {
	b := newFixtureRepo(t).
		file(".gitignore", ".abcd/.work.local/\n").
		file(".abcd/development/README.md", "x\n").
		dir(".abcd/work"). // work/ tier present, but no committed DECISIONS.md
		file(".abcd/work/CONTEXT.md", "x\n").
		file(".abcd/.work.local/DECISIONS.md", "- a decision that will not survive a clone\n").
		file("AGENTS.md", "x\n").
		commit()
	res := b.run()

	f := findingFor(res, "decision-durability")
	if f == nil {
		t.Fatal("no decision-durability finding when decisions are only in the gitignored layer")
	}
	if f.Severity != repolint.SeverityWarn {
		t.Errorf("severity = %q, want warn", f.Severity)
	}
	if res.Blockers != 0 {
		t.Errorf("blockers = %d, want 0", res.Blockers)
	}
	if res.ExitCode != 1 {
		t.Errorf("exit = %d, want 1", res.ExitCode)
	}
}

// conventions-router: AGENTS.md absent → error.
func TestRule_ConventionsRouterMissing(t *testing.T) {
	b := newFixtureRepo(t).
		file(".gitignore", ".abcd/.work.local/\n").
		file(".abcd/development/README.md", "x\n").
		file(".abcd/work/DECISIONS.md", "x\n").
		file(".abcd/.work.local/NEXT.md", "x\n").
		commit() // no AGENTS.md
	res := b.run()

	f := findingFor(res, "conventions-router")
	if f == nil {
		t.Fatal("no conventions-router finding when AGENTS.md is absent")
	}
	if f.Severity != repolint.SeverityError {
		t.Errorf("severity = %q, want error", f.Severity)
	}
}

// three-tier-layout: .work.local present but NOT gitignored → error (the leak
// the tier exists to prevent). git is available, so this is a real violation,
// not cannot-tell.
func TestRule_WorkLocalNotGitignored(t *testing.T) {
	b := newFixtureRepo(t).
		file(".gitignore", "# .work.local deliberately NOT ignored here\n").
		file(".abcd/development/README.md", "x\n").
		file(".abcd/work/DECISIONS.md", "x\n").
		file(".abcd/.work.local/NEXT.md", "x\n").
		file("AGENTS.md", "x\n").
		commit()
	res := b.run()

	f := findingFor(res, "three-tier-layout")
	if f == nil {
		t.Fatal("no three-tier-layout finding when .work.local is not gitignored")
	}
	if f.Severity != repolint.SeverityError {
		t.Errorf("severity = %q, want error", f.Severity)
	}
	if !strings.Contains(strings.ToLower(f.Message+f.Fix), "ignore") {
		t.Errorf("finding does not mention gitignore: %q / %q", f.Message, f.Fix)
	}
}

// A tier path that exists but is a regular FILE (not a directory) does not
// satisfy three-tier-layout — the tiers are directories.
func TestRule_TierPresentButNotADirectory(t *testing.T) {
	b := newFixtureRepo(t).
		file(".gitignore", ".abcd/.work.local/\n").
		file(".abcd/development", "I am a file, not the durable-record tier\n"). // a file at a tier path
		file(".abcd/work/DECISIONS.md", "x\n").
		file(".abcd/.work.local/NEXT.md", "x\n").
		file("AGENTS.md", "x\n").
		commit()
	res := b.run()

	f := findingFor(res, "three-tier-layout")
	if f == nil {
		t.Fatal("no three-tier-layout finding when .abcd/development is a file, not a directory")
	}
	if f.Severity != repolint.SeverityError {
		t.Errorf("severity = %q, want error", f.Severity)
	}
}

// The work tier specifically: when .abcd/work is a regular file, decision-durability
// stats .abcd/work/DECISIONS.md (ENOTDIR). That must read as "not present", not
// abort the audit — so the three-tier "must be a directory" finding still surfaces.
func TestRule_WorkTierIsAFileDoesNotAbort(t *testing.T) {
	b := newFixtureRepo(t).
		file(".gitignore", ".abcd/.work.local/\n").
		file(".abcd/development/README.md", "x\n").
		file(".abcd/work", "I am a file, not the shared-working tier\n").
		file(".abcd/.work.local/NEXT.md", "x\n").
		file("AGENTS.md", "x\n").
		commit()
	res := b.run() // must not error/abort

	f := findingFor(res, "three-tier-layout")
	if f == nil {
		t.Fatal("no three-tier-layout finding when .abcd/work is a file (audit aborted instead of reporting)")
	}
	if f.Severity != repolint.SeverityError {
		t.Errorf("severity = %q, want error", f.Severity)
	}
}

// three-tier-layout: local-tier artifacts (NEXT.md, scratch/, logs/) sitting
// directly in a committed tier → one error per misplacement, each naming the
// misplaced path. This is the leak class where a handover file full of
// machine-local detail rides a committed tier into public history.
func TestRule_LocalArtifactsInCommittedTiers(t *testing.T) {
	b := newFixtureRepo(t).conforming().
		file(".abcd/work/NEXT.md", "# handover in the wrong tier\n").
		dir(".abcd/development/scratch").
		dir(".abcd/work/logs").
		commit()
	res := b.run()

	total := 0
	got := map[string]repolint.Finding{}
	for _, f := range res.Findings {
		if f.RuleID != "three-tier-layout" {
			continue
		}
		total++
		got[f.File] = f
		if f.Severity != repolint.SeverityError {
			t.Errorf("severity for %s = %q, want error", f.File, f.Severity)
		}
	}
	// Exactly the three misplacements, no more: a regression that also fired on
	// the tiers themselves, dropped a misplacement, or emitted duplicates must
	// not slip through — count the raw findings, not the deduplicating map.
	if total != 3 {
		t.Errorf("three-tier-layout findings = %d, want 3 (%v)", total, got)
	}
	for want, mustName := range map[string][]string{
		".abcd/work/NEXT.md":        {"NEXT.md", "shared-working tier .abcd/work/"},
		".abcd/development/scratch": {"scratch", "durable-record tier .abcd/development/"},
		".abcd/work/logs":           {"logs", "shared-working tier .abcd/work/"},
	} {
		f, ok := got[want]
		if !ok {
			t.Errorf("no three-tier-layout finding for misplaced %s (got %v)", want, got)
			continue
		}
		for _, name := range mustName {
			if !strings.Contains(f.Message, name) {
				t.Errorf("message for %s does not name %q: %q", want, name, f.Message)
			}
		}
	}
	// The per-finding fix must name the destination tier, not fall back to the
	// rule-level tier-presence remediation.
	if f, ok := got[".abcd/work/NEXT.md"]; ok {
		if !strings.Contains(f.Fix, ".abcd/work/NEXT.md") || !strings.Contains(f.Fix, ".abcd/.work.local/") {
			t.Errorf("per-finding fix does not say what to move where: %q", f.Fix)
		}
	}
	if res.ExitCode != 2 {
		t.Errorf("exit = %d, want 2", res.ExitCode)
	}
}

// A DANGLING symlink named NEXT.md in a committed tier is still a violation:
// the name occupies the path, `git add -A` commits the link, and its target
// string can itself be a private absolute path — the exact leak class. A
// follow-symlinks presence check stats it as absent; the rule must not.
func TestRule_LocalArtifactDanglingSymlinkStillFlagged(t *testing.T) {
	b := newFixtureRepo(t).conforming()
	link := filepath.Join(b.root, ".abcd", "work", "NEXT.md")
	if err := os.Symlink("gone-target.md", link); err != nil { // relative target that does not exist
		t.Skipf("symlinks unsupported: %v", err)
	}
	b.commit()
	res := b.run()

	f := findingFor(res, "three-tier-layout")
	if f == nil {
		t.Fatal("no three-tier-layout finding for a dangling symlink named NEXT.md in .abcd/work/")
	}
	if f.File != ".abcd/work/NEXT.md" {
		t.Errorf("finding file = %q, want .abcd/work/NEXT.md", f.File)
	}
	if f.Severity != repolint.SeverityError {
		t.Errorf("severity = %q, want error", f.Severity)
	}
}

// The same artifact names in their own tier are exactly where they belong:
// NEXT.md, scratch/ and logs/ under .abcd/.work.local/ produce no findings.
func TestRule_LocalArtifactsInLocalTierClean(t *testing.T) {
	b := newFixtureRepo(t).conforming(). // conforming() already has .abcd/.work.local/NEXT.md
						dir(".abcd/.work.local/scratch").
						dir(".abcd/.work.local/logs").
						commit()
	res := b.run()

	if f := findingFor(res, "three-tier-layout"); f != nil {
		t.Fatalf("local-tier artifacts in their own tier were flagged: %+v", f)
	}
	if res.ExitCode != 0 {
		t.Errorf("exit = %d, want 0", res.ExitCode)
	}
}

// A directory named AGENTS.md does not satisfy conventions-router — the router is
// a file.
func TestRule_ConventionsRouterIsADirectory(t *testing.T) {
	b := newFixtureRepo(t).
		file(".gitignore", ".abcd/.work.local/\n").
		file(".abcd/development/README.md", "x\n").
		file(".abcd/work/DECISIONS.md", "x\n").
		file(".abcd/.work.local/NEXT.md", "x\n").
		file("AGENTS.md/keep.txt", "AGENTS.md is a directory here\n"). // dir, not a router file
		commit()
	res := b.run()

	f := findingFor(res, "conventions-router")
	if f == nil {
		t.Fatal("no conventions-router finding when AGENTS.md is a directory")
	}
	if f.Severity != repolint.SeverityError {
		t.Errorf("severity = %q, want error", f.Severity)
	}
}

// docs-currency: docs/ exists but no docs-lint config → a warn that the check
// could not run, never a silent pass.
func TestRule_DocsCurrencyNoConfigWarns(t *testing.T) {
	b := newFixtureRepo(t).conforming().
		file("docs/how-to/thing.md", "clean docs\n"). // docs/ present, but no .abcd/docs-lint.json
		commit()
	res := b.run()

	f := findingFor(res, "docs-currency")
	if f == nil {
		t.Fatal("docs-currency silently passed when docs/ exists but the config is missing")
	}
	if f.Severity != repolint.SeverityWarn {
		t.Errorf("severity = %q, want warn", f.Severity)
	}
	// It must be a warn, not an error: exit 1, not 2.
	if res.ExitCode != 1 {
		t.Errorf("exit = %d, want 1", res.ExitCode)
	}
}

// AC3: a committed file with an absolute local path → privacy-hygiene error
// citing file:line — unless a waiver escape is on that line.
func TestAC_PrivacyAbsolutePath(t *testing.T) {
	// The specimen must NOT be a persona name: a persona home is fixture material
	// the conventions mandate and is exempt (iss-2609100505145554), so spelling
	// this leak `alice` made the test assert the exemption rather than the rule.
	const leak = "see /Users/jdoe/secret/notes.md for context\n" // abcd-audit:allow
	b := newFixtureRepo(t).conforming().
		file("docs/how-to/thing.md", leak).
		commit()
	res := b.run()

	f := findingFor(res, "privacy-hygiene")
	if f == nil {
		t.Fatal("no privacy-hygiene finding for a committed absolute local path")
	}
	if f.Severity != repolint.SeverityError {
		t.Errorf("severity = %q, want error", f.Severity)
	}
	if f.File != "docs/how-to/thing.md" || f.Line != 1 {
		t.Errorf("citation = %s:%d, want docs/how-to/thing.md:1", f.File, f.Line)
	}
	if res.ExitCode != 2 {
		t.Errorf("exit = %d, want 2", res.ExitCode)
	}
}

// A bare home path with no trailing separator (the username IS the leak, e.g.
// `HOME=/home/jdoe` at end of line) must still be flagged — the previous regex abcd-audit:allow
// required a trailing slash and missed it. The specimen is a non-persona name:
// a persona home is exempt fixture material (iss-2609100505145554).
func TestAC_PrivacyBareHomePathNoTrailingSlash(t *testing.T) {
	const leak = "HOME=/home/jdoe\n" // abcd-audit:allow
	b := newFixtureRepo(t).conforming().
		file("reference/env.md", leak).
		commit()
	res := b.run()

	if f := findingFor(res, "privacy-hygiene"); f == nil {
		t.Fatal("bare home path without a trailing separator was not flagged")
	}
	if res.ExitCode != 2 {
		t.Errorf("exit = %d, want 2", res.ExitCode)
	}
}

func TestAC_PrivacyWaiverSuppresses(t *testing.T) {
	const waived = "example path /Users/alice/x is illustrative  abcd-lint:allow\n"
	// The pre-spc-29 waiver spelling is honoured forever (it lives in committed
	// content across managed repos).
	const waivedLegacy = "example path /Users/bob/x is illustrative  abcd-audit:allow\n"
	// Kept out of docs/ so docs-currency stays skipped and does not add a warn —
	// this test isolates the privacy waiver's effect on the exit code.
	b := newFixtureRepo(t).conforming().
		file("reference/paths.md", waived).
		file("reference/legacy.md", waivedLegacy).
		commit()
	res := b.run()

	if f := findingFor(res, "privacy-hygiene"); f != nil {
		t.Fatalf("waiver escape did not suppress the finding: %+v", f)
	}
	if res.ExitCode != 0 {
		t.Errorf("exit = %d, want 0 (waived)", res.ExitCode)
	}
}

// A hostile repo cannot turn privacy-hygiene into an out-of-repo read: a tracked
// symlink pointing outside the work tree is skipped, never followed. Its target
// contains an absolute path, so following it would both leak and (if it were
// /dev/zero) hang.
func TestRule_PrivacySkipsTrackedSymlink(t *testing.T) {
	outside := filepath.Join(t.TempDir(), "secret.txt")
	if err := os.WriteFile(outside, []byte("leak /Users/victim/.ssh/id_rsa\n"), 0o600); err != nil { // abcd-audit:allow
		t.Fatal(err)
	}
	b := newFixtureRepo(t).conforming()
	link := filepath.Join(b.root, "pointer.md")
	if err := os.Symlink(outside, link); err != nil {
		t.Skipf("symlinks unsupported: %v", err)
	}
	b.commit()
	res := b.run()

	if f := findingFor(res, "privacy-hygiene"); f != nil {
		t.Fatalf("privacy-hygiene followed a tracked symlink out of the repo: %+v", f)
	}
}

// A hostile working tree cannot escape the repo via a symlinked INTERMEDIATE
// directory either: sub/ is tracked, then swapped on disk for a symlink to an
// out-of-repo directory. The scan must refuse to read through it.
func TestRule_PrivacyRejectsIntermediateSymlinkEscape(t *testing.T) {
	outsideDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(outsideDir, "data.txt"), []byte("leak /Users/victim/x/\n"), 0o600); err != nil { // abcd-audit:allow
		t.Fatal(err)
	}
	b := newFixtureRepo(t).conforming().
		file("sub/data.txt", "clean in-repo content\n"). // tracked, no leak
		commit()
	// Swap the tracked intermediate directory for a symlink pointing outside.
	if err := os.RemoveAll(filepath.Join(b.root, "sub")); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outsideDir, filepath.Join(b.root, "sub")); err != nil {
		t.Skipf("symlinks unsupported: %v", err)
	}
	res := b.run()

	if f := findingFor(res, "privacy-hygiene"); f != nil {
		t.Fatalf("privacy-hygiene read through a symlinked intermediate directory: %+v", f)
	}
}

// A tracked file larger than the scan cap is skipped rather than loaded whole —
// a large committed binary must not OOM the repolint.
func TestRule_PrivacySkipsOversizeFile(t *testing.T) {
	b := newFixtureRepo(t).conforming()
	big := make([]byte, repolint.MaxScanBytesForTest()+1)
	for i := range big {
		big[i] = 'a'
	}
	// Put a leak on the first line so a naive scanner would flag it; the size cap
	// must skip the file before it is read — but the skip must be SAID, not
	// silent: "conforms" over content nobody scanned is the didn't-scan-
	// reported-clean shape (iss-356 item 4).
	copy(big, []byte("/Users/alice/x/\n")) // abcd-audit:allow
	b.file("huge.txt", string(big)).commit()
	res := b.run()

	f := findingFor(res, "privacy-hygiene")
	if f == nil {
		t.Fatal("an oversize textual file must yield a not-scanned warn, got no finding")
	}
	if f.Severity != repolint.SeverityWarn || !strings.Contains(f.Message, "not scanned") {
		t.Fatalf("want a not-scanned warn for the skipped file, got %+v", f)
	}
	if strings.Contains(f.Message, "/Users/") {
		t.Fatalf("the not-scanned warn must not quote the unscanned content: %+v", f)
	}
}

// An oversize BINARY blob stays quiet: the scanner would skip it below the cap
// too, so the cap loses nothing and a warn would tax every committed asset.
func TestRule_PrivacyOversizeBinaryStaysQuiet(t *testing.T) {
	b := newFixtureRepo(t).conforming()
	big := make([]byte, repolint.MaxScanBytesForTest()+1)
	copy(big, []byte("PNG\x00\x00binary")) // NUL in the probe window
	b.file("asset.bin", string(big)).commit()
	res := b.run()

	if f := findingFor(res, "privacy-hygiene"); f != nil {
		t.Fatalf("privacy-hygiene warned on an oversize binary: %+v", f)
	}
}

// When git reports tracked files but the repo root cannot be opened for reading
// (search-only permission), the privacy scan cannot run over content that exists.
// It must surface the error rather than silently report a clean pass (repolint.go:94).
func TestRule_PrivacyRootUnreadableSurfacesError(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("running as root bypasses permission bits")
	}
	b := newFixtureRepo(t).conforming().
		file("docs/how-to/thing.md", "clean content\n").
		commit()

	// Search-only (--x): git ls-files can still traverse into .git and read the
	// index, but os.OpenRoot(root) needs read permission and fails with EACCES.
	if err := os.Chmod(b.root, 0o311); err != nil {
		t.Fatalf("chmod root: %v", err)
	}
	t.Cleanup(func() { _ = os.Chmod(b.root, 0o755) })

	// Confirm the precondition actually holds on this filesystem; skip otherwise.
	if r, err := os.OpenRoot(b.root); err == nil {
		_ = r.Close()
		t.Skip("filesystem allows opening a search-only root; precondition not met")
	}

	var privacy repolint.Rule
	for _, r := range repolint.DefaultRules() {
		if r.Meta().ID == "privacy-hygiene" {
			privacy = r
			break
		}
	}
	if privacy == nil {
		t.Fatal("privacy-hygiene rule not found in DefaultRules")
	}

	if _, err := repolint.Evaluate([]repolint.Rule{privacy}, repolint.Context{RepoRoot: b.root}); err == nil {
		t.Fatal("privacy-hygiene reported clean when the repo root could not be opened")
	}
}

// AC5: docs/ absent → docs-currency skipped via Where, not failed.
func TestAC_DocsCurrencySkippedWhenNoDocs(t *testing.T) {
	res := newFixtureRepo(t).conforming().commit().run() // conforming() creates no docs/
	for _, f := range res.Findings {
		if f.RuleID == "docs-currency" {
			t.Fatalf("docs-currency produced a finding when docs/ is absent: %+v", f)
		}
	}
	found := false
	for _, id := range res.Skipped {
		if id == "docs-currency" {
			found = true
		}
	}
	if !found {
		t.Errorf("docs-currency not in Skipped when docs/ is absent: skipped=%v", res.Skipped)
	}
}

// A relative path segment or a URL path that merely contains "home/" or "Users/"
// is NOT an absolute local leak and must not hard-fail the privacy rule
// (iss-305). Before the leading-boundary gate, ordinary committed content —
// route directories, import paths, docs URLs — tripped SeverityError.
func TestAC_PrivacyRelativeHomeSegmentNotFlagged(t *testing.T) {
	const benign = "" + // abcd-audit:allow
		"import Hero from \"../components/home/Hero\";\n" +
		"see https://docs.example.com/home/getting-started for more\n" +
		"handler mounted at internal/web/home/handler.go\n" +
		"route file src/pages/home/index.tsx\n"
	b := newFixtureRepo(t).conforming().
		file("reference/routes.md", benign).
		commit()
	res := b.run()

	if f := findingFor(res, "privacy-hygiene"); f != nil {
		t.Fatalf("a relative/URL home segment was flagged as an absolute leak: %+v", f)
	}
	if res.ExitCode != 0 {
		t.Errorf("exit = %d, want 0", res.ExitCode)
	}
}

// The Windows arm of absPathRe is case-folded (iss-308): NTFS is
// case-insensitive and a lowercase `c:\users\<name>` is a real leak that the
// capital-U-only literal missed.
func TestAC_PrivacyWindowsLowercaseUsersPath(t *testing.T) {
	// Non-persona specimen: a persona home is exempt (iss-2609100505145554), and
	// `dave` is on the roster, so it tested the exemption, not the case fold.
	const leak = "cache dir is c:\\users\\jdoe\\AppData\\Local\\thing\n" // abcd-audit:allow
	b := newFixtureRepo(t).conforming().
		file("reference/win.md", leak).
		commit()
	res := b.run()

	if f := findingFor(res, "privacy-hygiene"); f == nil {
		t.Fatal("lowercase c:\\users\\<name> path was not flagged")
	}
	if res.ExitCode != 2 {
		t.Errorf("exit = %d, want 2", res.ExitCode)
	}
}
