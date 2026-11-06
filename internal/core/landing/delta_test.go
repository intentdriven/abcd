package landing

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/recordid"
	"github.com/intentdriven/abcd/internal/gittest"
)

const deltaDefault = "refs/remotes/origin/main"

// deltaWorld is a branch at a head a full review pair passed (base), off a
// default branch as last fetched.
type deltaWorld struct {
	t    *testing.T
	repo *gittest.Repo
	base string
}

// A record of each ledger on the default branch, and a block and body for them.
const (
	ledgerIssue   = ".abcd/work/issues/resolved/iss-1-x.md"
	ledgerWontfix = ".abcd/work/issues/wontfix/iss-2-x.md"
	ledgerHead    = "---\nid: \"iss-1\"\nseverity: \"major\"\n---\n"
	ledgerBody    = "\nThe record's body, as captured.\n"
)

func numberedLines(n int) string {
	var b strings.Builder
	for i := 1; i <= n; i++ {
		fmt.Fprintf(&b, "line %d\n", i)
	}
	return b.String()
}

func newDeltaWorld(t *testing.T) *deltaWorld {
	t.Helper()
	repo := gittest.NewRepo(t)
	// No identity comes from the host: a host that can guess one must not hide a
	// git command here that has none, as a host that cannot guess one refuses it.
	repo.Git("config", "user.useConfigOnly", "true")
	repo.Write(".gitignore", ".abcd/.work.local/\n")
	repo.Write("docs/how-to/a.md", numberedLines(60))
	repo.Write("docs/how-to/run.md", "#!/bin/sh\n")
	repo.Write(ledgerIssue, ledgerHead+ledgerBody)
	repo.Write(ledgerWontfix, ledgerHead+ledgerBody)
	for i := 0; i <= MaxDeltaFiles; i++ {
		repo.Write(fmt.Sprintf("docs/how-to/f%d.md", i), "x\n")
	}
	repo.Write("scripts/x.sh", "#!/bin/sh\necho x\n")
	repo.Write("Makefile", "all:\n")
	repo.Commit("the base")
	if err := os.MkdirAll(filepath.Join(repo.Root(), filepath.FromSlash(TierRelDir)), 0o700); err != nil {
		t.Fatal(err)
	}
	repo.Git("update-ref", deltaDefault, "HEAD")
	repo.Git("checkout", "-q", "-b", "feat/x")
	repo.Write("docs/how-to/feature.md", "the change\n")
	repo.Commit("the change")
	w := &deltaWorld{t: t, repo: repo, base: repo.Git("rev-parse", "HEAD")}
	if _, err := RecordReview(repo.Root(), w.base, fullPair(w.base)); err != nil {
		t.Fatalf("the full pair for the base head was refused: %v", err)
	}
	return w
}

func fullPair(sha string) []ReviewReturn {
	return []ReviewReturn{
		{Name: "ruthless.md", Text: reviewReturn(sha, "SHIP")},
		{Name: "security.md", Text: reviewReturn(sha, "APPROVE") + "\nFrom the security reviewer.\n"},
	}
}

// deltaReturn is a return that states the delta it read.
func deltaReturn(base, head, who, verdict string) string {
	return "Reviewed delta: " + base + ".." + head + "\n\n# " + who + "\n\n## Findings\n\nNone.\n\n### Verdict\n\nHead: " + head + "\n" + verdict + "\n"
}

func deltaPair(base, head string) []ReviewReturn {
	return []ReviewReturn{
		{Name: "ruthless.md", Text: deltaReturn(base, head, "ruthless", "SHIP")},
		{Name: "security.md", Text: deltaReturn(base, head, "security", "APPROVE")},
	}
}

func (w *deltaWorld) head() string { return w.repo.Git("rev-parse", "HEAD") }

// fix commits one edit to docs/how-to/a.md, one line changed.
func (w *deltaWorld) fix() string {
	w.t.Helper()
	body := strings.Replace(numberedLines(60), "line 1\n", "line one\n", 1)
	w.repo.Write("docs/how-to/a.md", body)
	w.repo.Commit("a one-line fix")
	return w.head()
}

// onMain commits one change to the default branch and moves its tracking ref,
// as a fetch would, leaving the branch checked out.
func (w *deltaWorld) onMain(rel, body string) string {
	w.t.Helper()
	w.repo.Git("checkout", "-q", "main")
	w.repo.Write(rel, body)
	w.repo.Commit("a change on the default branch")
	tip := w.head()
	w.repo.Git("update-ref", deltaDefault, tip)
	w.repo.Git("checkout", "-q", "feat/x")
	return tip
}

// has puts a file on the default branch and merges it up, so a commit after it
// changes a file the branch already holds.
func (w *deltaWorld) has(rel, body string) {
	w.t.Helper()
	w.onMain(rel, body)
	if err := w.merge("--no-edit", deltaDefault); err != nil {
		w.t.Fatal(err)
	}
}

// fixLines commits n one-line edits to docs/how-to/a.md, from line first on.
func (w *deltaWorld) fixLines(first, n int) {
	w.t.Helper()
	data, err := os.ReadFile(filepath.Join(w.repo.Root(), "docs", "how-to", "a.md"))
	if err != nil {
		w.t.Fatal(err)
	}
	body := string(data)
	for i := first; i < first+n; i++ {
		body = strings.Replace(body, fmt.Sprintf("line %d\n", i), fmt.Sprintf("line %d, fixed\n", i), 1)
	}
	w.repo.Write("docs/how-to/a.md", body)
	w.repo.Commit("edits")
}

// merge starts a merge that may stop for a conflict, git's verdict being the
// test's to read: exit 1 is returned, and any other failure, git refusing to run the
// merge at all, fails the test with git's own words.
func (w *deltaWorld) merge(args ...string) error {
	w.t.Helper()
	full := append([]string{"-C", w.repo.Root(), "-c", "user.email=fixture@example.invalid", "-c", "user.name=Fixture", "-c", "commit.gpgsign=false", "merge", "-q"}, args...)
	cmd := exec.Command("git", full...)
	cmd.Env = w.repo.Env()
	out, err := cmd.CombinedOutput()
	var exit *exec.ExitError
	if err != nil && (!errors.As(err, &exit) || exit.ExitCode() != 1) {
		w.t.Fatalf("git merge %v: %v\n%s", args, err, out)
	}
	return err
}

func (w *deltaWorld) record() error {
	w.t.Helper()
	head := w.head()
	_, err := RecordDeltaReview(w.repo.Root(), head, w.base, deltaPair(w.base, head))
	return err
}

// wantRefused asserts the delta for the branch's head is refused, naming want,
// that nothing was written, and that a delta receipt for that head does not stand
// at the merge gate either.
func (w *deltaWorld) wantRefused(want string) {
	w.t.Helper()
	err := w.record()
	if err == nil || !strings.Contains(err.Error(), want) {
		w.t.Fatalf("err = %v; want a refusal naming %q", err, want)
	}
	if !strings.Contains(err.Error(), "needs a full review pair") {
		w.t.Fatalf("err = %v; want the refusal to name the full pair the head needs", err)
	}
	head := w.head()
	if got, _ := reviewReceipt(w.repo.Root(), head); got != "" {
		w.t.Fatalf("a refused delta left a receipt at %s", got)
	}
	w.writeDeltaReceipt(head)
	proof := readReviewProof(w.repo.Root(), head, deltaDefault)
	if proof.reviewed || !strings.Contains(proof.deltaWhy, want) {
		w.t.Fatalf("the gate read a delta receipt that does not stand as %+v; want it refused, naming %q", proof, want)
	}
}

// writeDeltaReceipt writes a delta receipt that does not stand.
func (w *deltaWorld) writeDeltaReceipt(head string) {
	w.t.Helper()
	data, err := json.Marshal(reviewRecord{Commit: head, VerdictRule: verdictRule, Delta: &deltaRecord{Base: w.base}, Reviews: []reviewRecordEntry{
		{Name: "a", Verdict: "SHIP", SHA256: "x"}, {Name: "b", Verdict: "SHIP", SHA256: "y"},
	}})
	if err != nil {
		w.t.Fatal(err)
	}
	p := filepath.Join(w.repo.Root(), filepath.FromSlash(ReviewReceiptsRelDir), head)
	if err := os.WriteFile(p, data, 0o600); err != nil {
		w.t.Fatal(err)
	}
}

func TestADeltaWithinTheBoundIsRecordedRestingOnTheFullPair(t *testing.T) {
	w := newDeltaWorld(t)
	head := w.fix()
	path, err := RecordDeltaReview(w.repo.Root(), head, w.base, deltaPair(w.base, head))
	if err != nil {
		t.Fatalf("an accepted delta was refused: %v", err)
	}
	if want := filepath.Join(w.repo.Root(), filepath.FromSlash(ReviewReceiptsRelDir), head); path != want {
		t.Fatalf("receipt at %s, want %s", path, want)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var rec reviewRecord
	if err := json.Unmarshal(data, &rec); err != nil {
		t.Fatal(err)
	}
	if rec.Commit != head || rec.Delta == nil || rec.Delta.Base != w.base || rec.Delta.Files != 1 || rec.Delta.Lines != 2 || rec.Delta.Merges != 0 || len(rec.Reviews) != 2 {
		t.Fatalf("receipt = %+v, delta %+v", rec, rec.Delta)
	}
	proof := readReviewProof(w.repo.Root(), head, deltaDefault)
	if !proof.reviewed || !strings.Contains(proof.note, w.base[:9]) || proof.deltaWhy != "" || proof.unreadable != "" {
		t.Fatalf("the gate read the delta as %+v", proof)
	}
}

// A bound met exactly is within it; a later delta on the branch rests on the
// same full pair and reads the whole way from it.
func TestADeltaAtTheBoundPassesAndALaterOneReadsFromTheSameBase(t *testing.T) {
	w := newDeltaWorld(t)
	w.fixLines(1, MaxDeltaLines/2)
	if err := w.record(); err != nil {
		t.Fatalf("a delta at the line bound was refused: %v", err)
	}
	w.fixLines(MaxDeltaLines/2+1, 1)
	w.wantRefused("past the bound")
}

func TestADeltaPastTheBoundNeedsAFullPair(t *testing.T) {
	t.Run("lines", func(t *testing.T) {
		w := newDeltaWorld(t)
		w.fixLines(1, MaxDeltaLines/2+1)
		w.wantRefused("past the bound")
	})
	t.Run("files", func(t *testing.T) {
		w := newDeltaWorld(t)
		for i := 0; i <= MaxDeltaFiles; i++ {
			w.repo.Write(fmt.Sprintf("docs/how-to/f%d.md", i), "y\n")
		}
		w.repo.Commit("one file too many")
		w.wantRefused("past the bound")
	})
	// A change and its undoing leave the net diff empty and are both read.
	t.Run("churn counted commit by commit", func(t *testing.T) {
		w := newDeltaWorld(t)
		w.fixLines(1, MaxDeltaLines/2)
		w.repo.Write("docs/how-to/a.md", numberedLines(60))
		w.repo.Commit("undo")
		w.wantRefused("past the bound")
	})
}

func TestAHeadThatDoesNotDescendFromTheBaseNeedsAFullPair(t *testing.T) {
	w := newDeltaWorld(t)
	w.repo.Git("checkout", "-q", "-b", "other", "main")
	w.fix()
	w.wantRefused("does not descend")
}

func TestAReturnThatDoesNotNameTheDeltaIsRefused(t *testing.T) {
	w := newDeltaWorld(t)
	head := w.fix()
	good := deltaReturn(w.base, head, "ruthless", "SHIP")
	other := "0123456789abcdef0123456789abcdef01234567"
	for _, c := range []struct{ name, text, want string }{
		{"names only the head", reviewReturn(head, "SHIP"), "Reviewed delta"},
		{"another base", deltaReturn(other, head, "security", "APPROVE"), "does not name the delta"},
		{"another head", deltaReturn(w.base, other, "security", "APPROVE") + "Read " + head[:9] + ".\n", "does not name the delta"},
		{"a base cut too short", deltaReturn(w.base[:8], head, "security", "APPROVE"), "does not name the delta"},
		{"three dots", strings.Replace(deltaReturn(w.base, head, "security", "APPROVE"), "..", "...", 1), "Reviewed delta"},
		{"two deltas stated", deltaReturn(w.base, head, "security", "APPROVE") + "\nReviewed delta: " + w.base + ".." + head + "\n", "2 Reviewed delta lines"},
		{"in prose, not on its own line", "I read Reviewed delta: " + w.base + ".." + head + " today.\n\n### Verdict\n\nAPPROVE\n", "Reviewed delta"},
		{"below the first line", "# security\n\n" + deltaReturn(w.base, head, "security", "APPROVE"), "first line"},
		{"as a list item", "- " + deltaReturn(w.base, head, "security", "APPROVE"), "first line"},
		{"in bold", "**Reviewed delta:** " + w.base + ".." + head + "\n\n### Verdict\n\nAPPROVE\n", "first line"},
		{"in a code span", "Reviewed delta: `" + w.base + ".." + head + "`\n\n### Verdict\n\nAPPROVE\n", "first line"},
		{"in a fence", "```\n" + deltaReturn(w.base, head, "security", "APPROVE") + "```\n", "first line"},
		{"with more on the line", strings.Replace(deltaReturn(w.base, head, "security", "APPROVE"), "\n", " and more\n", 1), "first line"},
		{"another case", strings.Replace(deltaReturn(w.base, head, "security", "APPROVE"), "Reviewed delta", "reviewed delta", 1), "first line"},
		{"a head stated as well", deltaReturn(w.base, head, "security", "APPROVE") + "\nReviewed head: " + head + "\n", "Reviewed head"},
		{"a head stated in a list", deltaReturn(w.base, head, "security", "APPROVE") + "\n- **Reviewed head:** " + head[:9] + "\n", "Reviewed head"},
	} {
		t.Run(c.name, func(t *testing.T) {
			_, err := RecordDeltaReview(w.repo.Root(), head, w.base, []ReviewReturn{{Name: "ruthless.md", Text: good}, {Name: "security.md", Text: c.text}})
			if err == nil || !strings.Contains(err.Error(), c.want) || !strings.Contains(err.Error(), "security.md") {
				t.Fatalf("err = %v; want a refusal of security.md naming %q", err, c.want)
			}
			if got, _ := reviewReceipt(w.repo.Root(), head); got != "" {
				t.Fatalf("a refused delta left a receipt at %s", got)
			}
		})
	}
	// Abbreviated to the nine characters a brief uses, it stands.
	if _, err := RecordDeltaReview(w.repo.Root(), head, w.base, []ReviewReturn{
		{Name: "ruthless.md", Text: "\nReviewed delta: " + w.base[:9] + ".." + head[:9] + "\r\n\n### Verdict\n\nSHIP " + head + "\n"},
		{Name: "security.md", Text: deltaReturn(w.base, head, "security", "APPROVE")},
	}); err != nil {
		t.Fatalf("a delta named by nine characters was refused: %v", err)
	}
}

// The stop rule holds for a delta as for a full pair.
func TestADeltaReturnThatDoesNotPassIsRefused(t *testing.T) {
	w := newDeltaWorld(t)
	head := w.fix()
	for _, c := range []struct {
		name    string
		returns []ReviewReturn
		want    string
	}{
		{"fix first", []ReviewReturn{{Name: "a.md", Text: deltaReturn(w.base, head, "a", "SHIP")}, {Name: "b.md", Text: deltaReturn(w.base, head, "b", "FIX FIRST")}}, "b.md"},
		{"one reviewer", []ReviewReturn{{Name: "a.md", Text: deltaReturn(w.base, head, "a", "SHIP")}}, "two"},
		{"the same text twice", []ReviewReturn{{Name: "a.md", Text: deltaReturn(w.base, head, "a", "SHIP")}, {Name: "b.md", Text: deltaReturn(w.base, head, "a", "SHIP")}}, "same text"},
	} {
		t.Run(c.name, func(t *testing.T) {
			_, err := RecordDeltaReview(w.repo.Root(), head, w.base, c.returns)
			if err == nil || !strings.Contains(err.Error(), c.want) {
				t.Fatalf("err = %v; want a refusal naming %q", err, c.want)
			}
		})
	}
}

// A delta review covers the record tiers and the user-facing docs, by a closed
// list the binary holds. Every other path needs a full pair.
func TestADeltaCoversOnlyTheClosedList(t *testing.T) {
	for _, path := range []string{
		"docs/how-to/guide.md",
		"docs/explanation/guide.md",
		"docs/tutorials/guide.md",
		".abcd/development/plans/2026-11-04-x.md",
		".abcd/development/roadmap/phases/x.md",
	} {
		t.Run("covered "+path, func(t *testing.T) {
			w := newDeltaWorld(t)
			w.has(path, "as it was\n")
			w.repo.Write(path, "changed\n")
			w.repo.Commit("a change on the list")
			if err := w.record(); err != nil {
				t.Fatalf("a path on the list was refused: %v", err)
			}
		})
	}
	for _, path := range []string{
		"a.txt",
		"README.md",
		"AGENTS.md",
		"Makefile",
		"scripts/x.sh",
		"internal/core/landing/delta.go",
		"internal/core/x_test.go",
		"internal/gitutil/repo.go",
		".github/workflows/ci.yml",
		// What a harness or an agent loads as behaviour.
		"commands/land.md",
		"agents/reviewer.md",
		".claude/settings.json",
		".claude/commands/x.md",
		".claude-plugin/plugin.json",
		".claudeignore",
		// The repository's own configuration, none of it a record.
		".abcd/config.json",
		".abcd/guard.json",
		".abcd/rules.json",
		".abcd/record-lint.json",
		".abcd/docs-lint.json",
		".abcd/site.json",
		".abcd/README.md",
		".abcd/work/CONTEXT.md",
		".abcd/work/publish-surface.txt",
		".abcd/work/rulesets/main-protection.json",
		".abcd/work/reviews/x.md",
		".abcd/work/issues.md",
		".abcd/work/decisions-old/x.md",
		// The release gate and its record.
		".abcd/development/release/surface.json",
		".abcd/development/release/README.md",
		".abcd/development/release-gate/manifest.json",
		".abcd/development/release-gate/README.md",
		".abcd/development/releases/v0.1.0.md",
		// The open issues and the decisions.
		".abcd/work/issues/open/iss-1-x.md",
		".abcd/work/decisions/dcn-1-x.md",
		".abcd/work/issues/README.md",
		// The docs outside the listed folders.
		"docs/guide.md",
		"docs/README.md",
		"docs/reference/cli/commands.md",
		"docs/reference/writing-style.md",
		// The research record.
		".abcd/development/research/notes/2026-11-04-x.md",
		".abcd/development/research/ideas/x.md",
		// The intents and the specs.
		".abcd/development/intents/drafts/itd-1-x.md",
		".abcd/development/intents/planned/itd-1-x.md",
		".abcd/development/specs/open/spc-1-x.md",
		".abcd/development/specs/closed/spc-1-x.md",
		// Anything named as an instruction file is, however it is spelt.
		"docs/how-to/CLAUDE.md",
		"docs/how-to/AGENTS.md",
		"docs/how-to/GEMINI.md",
		"docs/how-to/SKILL.md",
		"docs/how-to/AGENTS.override.md",
		"docs/how-to/AGENT.md",
		"docs/how-to/agents-notes.md",
		"docs/how-to/CLAUDE.local.md",
		"docs/how-to/claude-notes.md",
		"docs/how-to/GEMINI.local.md",
		"docs/how-to/Skills.md",
		"docs/how-to/skills/x/guide.md",
		"docs/how-to/CODEX.md",
		"docs/how-to/copilot-instructions.md",
		"docs/how-to/claude.md/x.md",
		".abcd/development/plans/CLAUDE.md",
		".abcd/development/roadmap/GEMINI-x.md",
		".abcd/work/decisions/AGENTS.md",
		".abcd/work/issues/open/skill-x.md",
		// A dot-directory or a dot-file, wherever it sits below a listed directory.
		"docs/how-to/.claude/x.md",
		"docs/how-to/.github/copilot-instructions.md",
		"docs/how-to/.cursor/rules/x.md",
		"docs/how-to/.hidden.md",
		"docs/how-to/how-to/.hidden/x.md",
		".abcd/development/plans/.claude/x.md",
		".abcd/work/issues/.x/iss-1-x.md",
		".abcd/work/decisions/.x.md",
		// The brief, the principles, the ADRs and the rest of the design record.
		".abcd/development/brief/04-surfaces/38-land.md",
		".abcd/development/brief/02-constraints/03-invariants.md",
		".abcd/development/brief/glossary/core/x.md",
		".abcd/development/principles/x.md",
		".abcd/development/decisions/adrs/x.md",
		".abcd/development/agents/README.md",
		".abcd/development/personas.json",
		".abcd/development/README.md",
		// Under a listed directory, but no prose: a listed directory admits
		// markdown alone.
		"docs/how-to/requirements.txt",
		"docs/how-to/.gitattributes",
		"docs/how-to/assets/x.svg",
		"docs/how-to/x.js",
		"docs/how-to/x.md.sh",
		".abcd/development/intents/x.py",
		".abcd/development/research/abcdev-site/compose.py",
		".abcd/development/research/references.csl.json",
		".abcd/work/issues/x.json",
		"docs-old/x.md",
	} {
		t.Run("outside "+path, func(t *testing.T) {
			w := newDeltaWorld(t)
			w.repo.Write(path, "changed\n")
			w.repo.Commit("a change off the list")
			w.wantRefused("outside what a delta review covers")
		})
	}
	// One path off the list among paths on it is enough.
	t.Run("one path of several", func(t *testing.T) {
		w := newDeltaWorld(t)
		w.repo.Write("docs/how-to/feature.md", "x\n")
		w.repo.Write("internal/x.go", "package x\n")
		w.repo.Commit("both")
		w.wantRefused("outside what a delta review covers")
	})
	// The list does not depend on any file in the repository.
	t.Run("a list file changes nothing", func(t *testing.T) {
		w := newDeltaWorld(t)
		w.onMain(".abcd/work/publish-surface.txt", "/docs/\n")
		w.fix()
		if err := w.record(); err != nil {
			t.Fatalf("a publish surface list changed what a delta covers: %v", err)
		}
	})
}

// The list's paths are compared exactly. A fixture cannot commit these on a
// case-insensitive filesystem, so they are asked of the comparison directly.
func TestTheClosedListComparesExactly(t *testing.T) {
	for _, path := range []string{
		"DOCS/x.md", "Docs/x.md", ".ABCD/development/intents/x.md", ".abcd/WORK/issues/x.md", "docs", "x/docs/y.md",
		"docs/how-to/x.MD", "docs/how-to/.md/x", "docs/how-to/ClAuDe.md", "docs/how-to/sub/AgEnTs.x.md", "docs/how-to/SkIlL-x.md", "docs/how-to/.CLAUDE/x.md", "docs/how-to/../internal/x.md", "docs/how-to/./x.md", "docs/how-to//x.md", ".abcd/work/issues/../../config.md",
	} {
		if covered(path) {
			t.Errorf("covered(%q) = true", path)
		}
	}
}

// The command page, the brief and the generated reference name the list a delta
// is held to, entry by entry.
func TestTheLandPagesNameTheClosedList(t *testing.T) {
	if len(deltaCovers) == 0 {
		t.Fatal("the list is empty")
	}
	for _, rel := range []string{"commands/land.md", ".abcd/development/brief/04-surfaces/38-land.md", "docs/reference/cli/commands.md"} {
		data, err := os.ReadFile(filepath.Join("..", "..", "..", filepath.FromSlash(rel)))
		if err != nil {
			t.Fatal(err)
		}
		for _, e := range deltaCovers {
			if !strings.Contains(string(data), e) {
				t.Errorf("%s does not name %s", rel, e)
			}
		}
		// Collapsed, since a figure and its noun can fall either side of a line.
		if size := fmt.Sprintf("%d KiB", maxDeltaPatchBytes>>10); !strings.Contains(strings.Join(strings.Fields(string(data)), " "), size) {
			t.Errorf("%s does not state the patch bound (%s)", rel, size)
		}
	}
}

// What the default branch brought is not the branch's change, however large and
// wherever it falls: a clean merge from it counts against neither the bound nor
// the closed list.
func TestAMergeFromTheDefaultBranchDoesNotCount(t *testing.T) {
	big := numberedLines(MaxDeltaLines * 10)
	t.Run("before a fix", func(t *testing.T) {
		w := newDeltaWorld(t)
		w.onMain("big.txt", big)
		w.onMain("scripts/x.sh", "changed on main\n")
		if err := w.merge("--no-edit", deltaDefault); err != nil {
			t.Fatal(err)
		}
		head := w.fix()
		if _, err := RecordDeltaReview(w.repo.Root(), head, w.base, deltaPair(w.base, head)); err != nil {
			t.Fatalf("a merge from main counted: %v", err)
		}
		rec := w.delta(head)
		if rec.Lines != 2 || rec.Files != 1 || rec.Merges != 1 {
			t.Fatalf("delta = %+v; want the fix alone, one merge", rec)
		}
		if p := readReviewProof(w.repo.Root(), head, deltaDefault); !p.reviewed {
			t.Fatalf("the gate read it as %+v", p)
		}
	})
	t.Run("alone", func(t *testing.T) {
		w := newDeltaWorld(t)
		w.onMain("big.txt", big)
		if err := w.merge("--no-edit", deltaDefault); err != nil {
			t.Fatal(err)
		}
		if err := w.record(); err != nil {
			t.Fatalf("merging main up needed more than the delta pair: %v", err)
		}
	})
	t.Run("after a fix and with main moved on since", func(t *testing.T) {
		w := newDeltaWorld(t)
		w.fix()
		w.onMain("big.txt", big)
		if err := w.merge("--no-edit", deltaDefault); err != nil {
			t.Fatal(err)
		}
		w.onMain("later.txt", big)
		if err := w.record(); err != nil {
			t.Fatalf("a merge from an older tip counted: %v", err)
		}
	})
	t.Run("a merge that takes both sides of one file", func(t *testing.T) {
		w := newDeltaWorld(t)
		w.fix()
		w.onMain("docs/how-to/a.md", strings.Replace(numberedLines(60), "line 50\n", "line fifty\n", 1))
		if err := w.merge("--no-edit", deltaDefault); err != nil {
			t.Fatal(err)
		}
		if err := w.record(); err != nil {
			t.Fatalf("a clean content merge was refused: %v", err)
		}
	})
}

func (w *deltaWorld) delta(head string) deltaRecord {
	w.t.Helper()
	p, err := reviewReceipt(w.repo.Root(), head)
	if err != nil || p == "" {
		w.t.Fatalf("no receipt for %s: %v", head, err)
	}
	data, err := os.ReadFile(p)
	if err != nil {
		w.t.Fatal(err)
	}
	var rec reviewRecord
	if err := json.Unmarshal(data, &rec); err != nil {
		w.t.Fatal(err)
	}
	if rec.Delta == nil {
		w.t.Fatalf("the receipt for %s is not a delta's", head)
	}
	return *rec.Delta
}

// A merge counts for nothing only when it is the merge git makes of a
// default-branch commit with no conflict; every other merge needs a full pair.
func TestAMergeThatIsNotGitsOwnNeedsAFullPair(t *testing.T) {
	t.Run("a tree with a change of its own", func(t *testing.T) {
		w := newDeltaWorld(t)
		w.onMain("big.txt", numberedLines(100))
		if err := w.merge("--no-commit", "--no-ff", deltaDefault); err != nil {
			t.Fatal(err)
		}
		w.repo.Write("internal/x.go", "package x\n")
		w.repo.Commit("Merge the default branch")
		w.wantRefused("no clean merge makes")
	})
	t.Run("a tree with one line of its own", func(t *testing.T) {
		w := newDeltaWorld(t)
		w.onMain("big.txt", numberedLines(3))
		if err := w.merge("--no-commit", "--no-ff", deltaDefault); err != nil {
			t.Fatal(err)
		}
		w.repo.Write("docs/how-to/feature.md", "the change, altered\n")
		w.repo.Commit("Merge the default branch")
		w.wantRefused("no clean merge makes")
	})
	t.Run("a tree without the default branch's change", func(t *testing.T) {
		w := newDeltaWorld(t)
		w.onMain("scripts/x.sh", "#!/bin/sh\necho changed\n")
		if err := w.merge("-s", "ours", "--no-edit", deltaDefault); err != nil {
			t.Fatal(err)
		}
		w.wantRefused("no clean merge makes")
	})
	t.Run("a merge that conflicts", func(t *testing.T) {
		w := newDeltaWorld(t)
		w.repo.Write("docs/how-to/feature.md", "the branch's line\n")
		w.repo.Commit("a fix on the branch")
		w.onMain("docs/how-to/feature.md", "main's line\n")
		if err := w.merge("--no-edit", deltaDefault); err == nil {
			t.Fatal("the fixture's merge should conflict")
		}
		w.repo.Write("docs/how-to/feature.md", "a resolution\n")
		w.repo.Commit("Merge the default branch")
		w.wantRefused("conflicts")
	})
	t.Run("a merge from another branch", func(t *testing.T) {
		w := newDeltaWorld(t)
		w.repo.Git("checkout", "-q", "-b", "side", "main")
		w.repo.Write("side.txt", numberedLines(100))
		w.repo.Commit("work no reviewer read")
		w.repo.Git("checkout", "-q", "feat/x")
		if err := w.merge("--no-edit", "side"); err != nil {
			t.Fatal(err)
		}
		w.wantRefused("not on the default branch")
	})
	t.Run("the branch merged into the default branch's commit", func(t *testing.T) {
		w := newDeltaWorld(t)
		tip := w.onMain("big.txt", numberedLines(100))
		w.repo.Git("checkout", "-q", "--detach", tip)
		if err := w.merge("--no-edit", w.base); err != nil {
			t.Fatal(err)
		}
		w.wantRefused("not on the default branch")
	})
	t.Run("an octopus", func(t *testing.T) {
		w := newDeltaWorld(t)
		w.repo.Git("checkout", "-q", "-b", "other", "main")
		w.repo.Write("two.txt", "x\n")
		w.repo.Commit("another line of work")
		w.repo.Git("checkout", "-q", "feat/x")
		w.onMain("one.txt", "x\n")
		if err := w.merge("--no-edit", "other", deltaDefault); err != nil {
			t.Fatal(err)
		}
		w.wantRefused("3 parents")
	})
}

// A delta covers regular text files. A NUL byte anywhere in a changed blob, on
// either side of the change, refuses it, whatever an attribute says of the file.
func TestADeltaOverABinaryFileNeedsAFullPair(t *testing.T) {
	t.Run("a NUL byte near the start", func(t *testing.T) {
		w := newDeltaWorld(t)
		w.repo.Write("docs/how-to/feature.md", "a\x00b\n")
		w.repo.Commit("a binary")
		w.wantRefused("binary")
	})
	t.Run("a NUL byte past what git samples", func(t *testing.T) {
		w := newDeltaWorld(t)
		w.repo.Write("docs/how-to/feature.md", strings.Repeat("x", 1<<20)+"\x00\n")
		w.repo.Commit("a NUL late in the file")
		w.wantRefused("binary")
	})
	t.Run("a NUL byte in a file an attribute marks as text", func(t *testing.T) {
		w := newDeltaWorld(t)
		w.onMain(".gitattributes", "*.md diff\n")
		if err := w.merge("--no-edit", deltaDefault); err != nil {
			t.Fatal(err)
		}
		w.repo.Write("docs/how-to/feature.md", "a\x00b\n")
		w.repo.Commit("a marked file")
		w.wantRefused("binary")
	})
	t.Run("changing a binary file to text", func(t *testing.T) {
		w := newDeltaWorld(t)
		w.repo.Git("checkout", "-q", "main")
		w.repo.Write("docs/how-to/old.md", "a\x00b\n")
		w.repo.Commit("a binary on the default branch")
		w.repo.Git("update-ref", deltaDefault, "HEAD")
		w.repo.Git("checkout", "-q", "feat/x")
		if err := w.merge("--no-edit", deltaDefault); err != nil {
			t.Fatal(err)
		}
		w.repo.Write("docs/how-to/old.md", "plain\n")
		w.repo.Commit("make it text")
		w.wantRefused("binary")
	})
}

// A delta changes wording, on every path it covers. A changed line naming a
// record, a path or code is a full pair's, added or removed, whichever record it
// is in.
func TestADeltaChangesWordingAlone(t *testing.T) {
	for name, line := range map[string]string{
		"an issue id":          "It follows iss-1 here.",
		"a decision id":        "It follows dcn-1 here.",
		"an intent id":         "It follows itd-1 here.",
		"a spec id":            "It follows spc-1 here.",
		"an ADR id":            "It follows adr-1 here.",
		"a receipt id":         "It follows rcp-0123456789ab here.",
		"a reading run id":     "It follows rdg-1 here.",
		"a reading item id":    "It follows rdi-1 here.",
		"an admission id":      "It follows adm-1 here.",
		"a surprise id":        "It follows srp-1 here.",
		"a reframe id":         "It follows rfm-1 here.",
		"a disposition id":     "It follows dsp-1 here.",
		"a note id":            "It follows mem-1 here.",
		"a frozen log id":      "It follows dec-L12 here.",
		"an id in capitals":    "It follows ISS-12 here.",
		"an id in its slug":    "It follows iss-12-the-slug here.",
		"a family not yet met": "It follows zzz-12 here.",
		"a bare number":        "It follows 2609291342092738 here.",
		"a commit":             "It follows a049b03f1 here.",
		"the receipt words":    "It carries (receipt of the check here.",
		"a path":               "It reads internal/core/x here.",
		"a file by its name":   "It reads delta.go here.",
		"a dotted directory":   "It reads .abcd here.",
		"a root file":          "It reads Makefile here.",
		"a backslash":          "It reads a\\b here.",
		"a code span":          "It runs `make` here.",
		"a fence":              "```",
		"a tilde fence":        "~~~",
		"an indented line":     "    It runs make here.",
		"a tabbed line":        "\tIt runs make here.",
		"a block quote":        "> It runs make here.",
		"a link":               "It reads [the page](the-page) here.",
		"markup":               "It reads <b>this</b> here.",
		"a comment":            "<!-- It reads this here. -->",
		"a table row":          "| It | reads |",
		"a heading":            "# It reads this",
		"a sum":                "It sets x = 1 here.",
		"a variable":           "It reads $HOME here.",
		"a non-breaking space": "It reads\u00a0this here.",
		"a lint waiver":        "It reads this here. abcd-lint:allow",
		"an audit waiver":      "It reads this here. abcd-audit:allow",
		"any abcd waiver":      "It reads this here. abcd-other:allow",
		"a waiver in capitals": "It reads this here. ABCD-LINT:ALLOW",
		"a spaced waiver":      "It reads this here. abcd-lint: allow",
		"a scanner's waiver":   "It reads this here. gitleaks:allow",
		"a docs-lint waiver":   "It reads this here. <!-- docs-lint: allow -->",
		"a record-lint waiver": "It reads this here. <!-- record-lint: illustrative -->",
		"a joined colon":       "It reads this:here.",
		"a variation selector": "It reads this\ufe0f here.",
		"a far selector":       "It reads this\U000e0100 here.",
		"a grapheme joiner":    "It reads this\u034f here.",
		"a Hangul filler":      "It reads\u3164this here.",
		"a Khmer vowel":        "It reads this\u17b4 here.",
		"a zero-width joiner":  "It reads this\u200d here.",
		"a soft hyphen":        "It reads th\u00adis here.",
		"a tag character":      "It reads this\U000e0041 here.",
		"a bidi override":      "It reads \u202ethis here.",
		"a word joiner":        "It reads\u2060 this here.",
	} {
		t.Run("added: "+name, func(t *testing.T) {
			w := newDeltaWorld(t)
			w.repo.Write(ledgerIssue, ledgerHead+ledgerBody+line+"\n")
			w.repo.Commit("a change to a body")
			w.wantRefused("wording")
		})
	}
	t.Run("added in a wontfix record", func(t *testing.T) {
		w := newDeltaWorld(t)
		w.repo.Write(ledgerWontfix, ledgerHead+ledgerBody+"It follows iss-1 here.\n")
		w.repo.Commit("a change to a body")
		w.wantRefused("wording")
	})
	t.Run("added in a page", func(t *testing.T) {
		w := newDeltaWorld(t)
		w.repo.Write("docs/how-to/a.md", numberedLines(60)+"Run `make` here.\n")
		w.repo.Commit("a change to a page")
		w.wantRefused("wording")
	})
	t.Run("removed", func(t *testing.T) {
		w := newDeltaWorld(t)
		w.onMain(ledgerIssue, ledgerHead+ledgerBody+"It carries (receipt of the check here.\n")
		if err := w.merge("--no-edit", deltaDefault); err != nil {
			t.Fatal(err)
		}
		w.repo.Write(ledgerIssue, ledgerHead+ledgerBody)
		w.repo.Commit("a line taken out")
		w.wantRefused("wording")
	})
}

// A fence is a whole file's state, so a line is judged by where it falls in the
// file on each side of the change, not by its own text.
func TestADeltaChangesNoLineInsideACodeFence(t *testing.T) {
	const page = "docs/how-to/fenced.md"
	world := func(t *testing.T, body string) *deltaWorld {
		t.Helper()
		w := newDeltaWorld(t)
		w.onMain(page, body)
		if err := w.merge("--no-edit", deltaDefault); err != nil {
			t.Fatal(err)
		}
		return w
	}
	const fenced = "Before it.\n\n```\nrun this\n```\n\nAfter it.\n"
	for name, c := range map[string]struct{ body, changed string }{
		"a line changed":       {fenced, strings.Replace(fenced, "run this", "run that", 1)},
		"a line added":         {fenced, strings.Replace(fenced, "run this\n", "run this\nrun that\n", 1)},
		"a line removed":       {fenced, strings.Replace(fenced, "run this\n", "", 1)},
		"a tilde fence":        {"~~~\nrun this\n~~~\n", "~~~\nrun that\n~~~\n"},
		"a fence with a tag":   {"```sh\nrun this\n```\n", "```sh\nrun that\n```\n"},
		"a fence never closed": {"```\nrun this\n\nA long way on.\n", "```\nrun this\n\nA long way off.\n"},
		"past a shorter fence": {"````\n```\nrun this\n```\nrun on\n````\n", "````\n```\nrun this\n```\nrun off\n````\n"},
		"past the other mark":  {"```\n~~~\nrun on\n```\n", "```\n~~~\nrun off\n```\n"},
		"indented":             {"   ```\nrun this\n   ```\n", "   ```\nrun that\n   ```\n"},
		"in a list item":       {"- A step:\n\n  ```\n  run this\n  ```\n", "- A step:\n\n  ```\n  run that\n  ```\n"},
		"in a block quote":     {"> ```\nrun this\n> ```\n", "> ```\nrun that\n> ```\n"},
	} {
		t.Run(name, func(t *testing.T) {
			w := world(t, c.body)
			w.repo.Write(page, c.changed)
			w.repo.Commit("a change")
			w.wantRefused("code fence")
		})
	}
	for name, changed := range map[string]string{
		"before the fence": strings.Replace(fenced, "Before it.", "Ahead of it.", 1),
		"after the fence":  strings.Replace(fenced, "After it.", "Behind it.", 1),
		"added after":      fenced + "\nOne more line.\n",
	} {
		t.Run("accepted: "+name, func(t *testing.T) {
			w := world(t, fenced)
			w.repo.Write(page, changed)
			w.repo.Commit("a wording fix")
			if err := w.record(); err != nil {
				t.Fatalf("a line outside the fence was refused: %v", err)
			}
		})
	}
	// Each file is judged by its own fences, in a commit that changes several.
	t.Run("a second file's fence", func(t *testing.T) {
		w := world(t, fenced)
		w.repo.Write("docs/how-to/a.md", strings.Replace(numberedLines(60), "line 1\n", "line one\n", 1))
		w.repo.Write(page, strings.Replace(fenced, "run this", "run that", 1))
		w.repo.Commit("two files")
		w.wantRefused("code fence")
	})
}

// A head's receipts are read from every worktree: a full pair recorded in any of
// them counts, whatever another worktree holds for that head.
func TestEveryWorktreesReceiptIsRead(t *testing.T) {
	lane := func(t *testing.T, w *deltaWorld) string {
		t.Helper()
		other := filepath.Join(t.TempDir(), "lane")
		w.repo.Git("worktree", "add", "-q", "--detach", other, "HEAD")
		if err := os.MkdirAll(filepath.Join(other, filepath.FromSlash(TierRelDir)), 0o700); err != nil {
			t.Fatal(err)
		}
		return other
	}
	t.Run("a delta that does not stand beside a full pair", func(t *testing.T) {
		w := newDeltaWorld(t)
		w.repo.Write("internal/x.go", "package x\n")
		w.repo.Commit("code")
		head := w.head()
		w.writeDeltaReceipt(head)
		if _, err := RecordReview(lane(t, w), head, fullPair(head)); err != nil {
			t.Fatal(err)
		}
		if proof := readReviewProof(w.repo.Root(), head, deltaDefault); !proof.reviewed || proof.deltaWhy != "" || proof.note != "" {
			t.Fatalf("the gate read %+v; want the full pair", proof)
		}
	})
	t.Run("a receipt that does not parse beside a full pair", func(t *testing.T) {
		w := newDeltaWorld(t)
		head := w.fix()
		p := filepath.Join(w.repo.Root(), filepath.FromSlash(ReviewReceiptsRelDir), head)
		if err := os.WriteFile(p, []byte("not a receipt"), 0o600); err != nil {
			t.Fatal(err)
		}
		if _, err := RecordReview(lane(t, w), head, fullPair(head)); err != nil {
			t.Fatal(err)
		}
		if proof := readReviewProof(w.repo.Root(), head, deltaDefault); !proof.reviewed {
			t.Fatalf("the gate read %+v; want the full pair", proof)
		}
	})
	t.Run("a standing delta beside one that does not parse", func(t *testing.T) {
		w := newDeltaWorld(t)
		head := w.fix()
		p := filepath.Join(w.repo.Root(), filepath.FromSlash(ReviewReceiptsRelDir), head)
		if err := os.WriteFile(p, []byte("not a receipt"), 0o600); err != nil {
			t.Fatal(err)
		}
		if _, err := RecordDeltaReview(lane(t, w), head, w.base, deltaPair(w.base, head)); err != nil {
			t.Fatal(err)
		}
		if proof := readReviewProof(w.repo.Root(), head, deltaDefault); !proof.reviewed || proof.note == "" {
			t.Fatalf("the gate read %+v; want the delta", proof)
		}
	})
	t.Run("a base whose full pair sits in another worktree", func(t *testing.T) {
		w := newDeltaWorld(t)
		other := lane(t, w)
		if _, err := RecordReview(other, w.base, fullPair(w.base)); err != nil {
			t.Fatal(err)
		}
		p := filepath.Join(w.repo.Root(), filepath.FromSlash(ReviewReceiptsRelDir), w.base)
		if err := os.WriteFile(p, []byte("not a receipt"), 0o600); err != nil {
			t.Fatal(err)
		}
		w.fix()
		if err := w.record(); err != nil {
			t.Fatalf("the full pair in another worktree was not read: %v", err)
		}
	})
	t.Run("a delta never replaces a full pair in another worktree", func(t *testing.T) {
		w := newDeltaWorld(t)
		head := w.fix()
		w.writeDeltaReceipt(head)
		if _, err := RecordReview(lane(t, w), head, fullPair(head)); err != nil {
			t.Fatal(err)
		}
		if err := w.record(); err == nil || !strings.Contains(err.Error(), "already has a full") {
			t.Fatalf("err = %v; want the delta refused", err)
		}
	})
	t.Run("nothing that stands in any worktree proves nothing", func(t *testing.T) {
		w := newDeltaWorld(t)
		w.repo.Write("internal/x.go", "package x\n")
		w.repo.Commit("code")
		head := w.head()
		w.writeDeltaReceipt(head)
		other := lane(t, w)
		p := filepath.Join(other, filepath.FromSlash(ReviewReceiptsRelDir), head)
		if err := os.MkdirAll(filepath.Dir(p), 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte("not a receipt"), 0o600); err != nil {
			t.Fatal(err)
		}
		if proof := readReviewProof(w.repo.Root(), head, deltaDefault); proof.reviewed || proof.deltaWhy == "" {
			t.Fatalf("the gate read %+v; want the delta's refusal", proof)
		}
	})
}

// Any family the allocator mints is in the shape, whether a list names it or not,
// in any case and with its slug. A word with a number in no id's shape is not one.
func TestMentionsIDKnowsEveryFamilyByItsShape(t *testing.T) {
	minted, err := recordid.Minter{}.Mint("zzz")
	if err != nil {
		t.Fatal(err)
	}
	ids := []string{minted, "rdg-1", "rdi-1", "adm-1", "srp-1", "rfm-1", "dsp-1", "mem-1", "dec-L12", "rcp-0123456789ab", "2609291342092738", "a049b03f1"}
	for _, f := range recordid.CitedFamilies() {
		ids = append(ids, f+"-1", f+"-2609291342092738", f+"-12-its-slug")
	}
	for _, id := range ids {
		for _, text := range []string{id, "see " + id + ".", "(" + id + ")", "see " + strings.ToUpper(id) + " now"} {
			if !mentionsID(text) {
				t.Errorf("mentionsID(%q) = false", text)
			}
		}
	}
	for _, text := range []string{"", "a plain sentence", "3 of 4 runs", "a well-known word", "line 12", "the 2nd try"} {
		if mentionsID(text) {
			t.Errorf("mentionsID(%q) = true", text)
		}
	}
}

// That is the feature's point: a prose fix to a record's body, after a full pair,
// is a delta's, at recording and at the merge gate.
func TestAProseFixToAnIssueBodyStillQualifies(t *testing.T) {
	for _, rel := range []string{ledgerIssue, ledgerWontfix, "docs/how-to/page.md", ".abcd/development/plans/x.md"} {
		t.Run(rel, func(t *testing.T) {
			w := newDeltaWorld(t)
			body := "A corrected sentence: the gate's note said (rightly) why it refused, and 3 of 4 runs \u2014 most \"quiet\" ones \u2014 passed; e.g. the first, i.e. the *only* one!\n- A second line, in a list.\n- Caf\u00e9, with its accent as a mark: cafe\u0301.\n"
			if rel == ledgerIssue || rel == ledgerWontfix {
				body = ledgerHead + "\n" + body
			} else {
				w.has(rel, "An older sentence.\n")
			}
			w.repo.Write(rel, body)
			w.repo.Commit("a wording fix")
			if err := w.record(); err != nil {
				t.Fatalf("a wording fix was refused: %v", err)
			}
			if proof := readReviewProof(w.repo.Root(), w.head(), deltaDefault); !proof.reviewed {
				t.Fatalf("the gate read the delta as %+v", proof)
			}
		})
	}
}

// The grounds of a ledger record are structured and append-only, so a delta changes
// no line of the section on either side, and a prose fix around them is a delta's.
func TestADeltaChangesNoLineOfALedgerRecordsGrounds(t *testing.T) {
	const section = "\n## Grounds\n\n- pursued: the first reason\n- declined: the second reason\n  and its second line\n"
	const text = ledgerHead + "\nA sentence above.\n" + section + "\n## Notes\n\nA sentence below.\n"
	for _, rel := range []string{ledgerIssue, ledgerWontfix} {
		for name, changed := range map[string]string{
			"a token flipped":   strings.Replace(text, "- pursued:", "- deferred:", 1),
			"a bullet removed":  strings.Replace(text, "- pursued: the first reason\n", "", 1),
			"a bullet added":    strings.Replace(text, "- pursued: the first reason\n", "- pursued: the first reason\n- pursued: a new reason\n", 1),
			"a reason reworded": strings.Replace(text, "the first reason", "the better reason", 1),
			"a second line":     strings.Replace(text, "and its second line", "and its better line", 1),
			"a line of prose":   strings.Replace(text, "- pursued: the first reason\n", "A sentence inside.\n- pursued: the first reason\n", 1),
		} {
			t.Run(name+" in "+rel, func(t *testing.T) {
				w := newDeltaWorld(t)
				w.has(rel, text)
				w.repo.Write(rel, changed)
				w.repo.Commit("a change")
				w.wantRefused("Grounds")
			})
		}
		for name, changed := range map[string]string{
			"above the section": strings.Replace(text, "A sentence above.", "A better sentence above.", 1),
			"below the section": strings.Replace(text, "A sentence below.", "A better sentence below.", 1),
		} {
			t.Run("accepted: "+name+" in "+rel, func(t *testing.T) {
				w := newDeltaWorld(t)
				w.has(rel, text)
				w.repo.Write(rel, changed)
				w.repo.Commit("a wording fix")
				if err := w.record(); err != nil {
					t.Fatalf("a prose fix outside the section was refused: %v", err)
				}
				if proof := readReviewProof(w.repo.Root(), w.head(), deltaDefault); !proof.reviewed {
					t.Fatalf("the gate read the delta as %+v", proof)
				}
			})
		}
	}
	// A page is no ledger record: the words are its own.
	t.Run("accepted: a page's section of that name", func(t *testing.T) {
		w := newDeltaWorld(t)
		w.has("docs/how-to/page.md", "## Grounds\n\nA sentence of the page.\n")
		w.repo.Write("docs/how-to/page.md", "## Grounds\n\nA better sentence of the page.\n")
		w.repo.Commit("a wording fix")
		if err := w.record(); err != nil {
			t.Fatalf("a page's own section was refused: %v", err)
		}
	})
}

// A delta reads every gitlink, whatever a repository says to ignore, so a bumped
// submodule is refused by its mode.
func TestASubmoduleBumpNeedsAFullPair(t *testing.T) {
	const link = "docs/how-to/sub.md"
	const module = "[submodule \"sub\"]\n\tpath = " + link + "\n\turl = ./sub\n"
	for name, c := range map[string]struct {
		modules string
		config  []string
	}{
		"ignored in the modules file":        {module + "\tignore = all\n", nil},
		"ignored in the local module config": {module, []string{"submodule.sub.ignore", "all"}},
		"ignored for every diff":             {module, []string{"diff.ignoreSubmodules", "all"}},
	} {
		t.Run(name, func(t *testing.T) {
			w := newDeltaWorld(t)
			w.repo.Git("checkout", "-q", "main")
			w.repo.Write(".gitmodules", c.modules)
			w.repo.Git("add", ".gitmodules")
			w.repo.Git("update-index", "--add", "--cacheinfo", "160000,"+w.base+","+link)
			w.repo.Git("commit", "-q", "-m", "a submodule on the default branch")
			w.repo.Git("update-ref", deltaDefault, "HEAD")
			w.repo.Git("checkout", "-q", "feat/x")
			if err := w.merge("--no-edit", deltaDefault); err != nil {
				t.Fatal(err)
			}
			if c.config != nil {
				w.repo.Git("config", c.config[0], c.config[1])
			}
			w.repo.Git("update-index", "--cacheinfo", "160000,"+w.repo.Git("rev-parse", "main")+","+link)
			w.repo.Git("commit", "-q", "-m", "a bump")
			w.wantRefused("mode")
		})
	}
}

// A delta reads history as stored: a replace ref or a graft changes neither what a
// commit holds nor whose child it is.
func TestADeltaReadsHistoryAsStored(t *testing.T) {
	t.Run("a replace ref", func(t *testing.T) {
		w := newDeltaWorld(t)
		parent := w.head()
		w.repo.Write("internal/x.go", "package x\n")
		w.repo.Commit("code")
		code := w.head()
		w.repo.Git("checkout", "-q", "--detach", parent)
		w.repo.Write("docs/how-to/feature.md", "the change, fixed\n")
		w.repo.Commit("wording")
		wording := w.head()
		w.repo.Git("checkout", "-q", "feat/x")
		w.repo.Git("replace", code, wording)
		w.wantRefused("outside what a delta review covers")
	})
	t.Run("a graft", func(t *testing.T) {
		w := newDeltaWorld(t)
		w.repo.Write("internal/x.go", "package x\n")
		w.repo.Commit("code")
		head := w.fix()
		grafts := filepath.Join(w.repo.Root(), ".git", "info", "grafts")
		if err := os.MkdirAll(filepath.Dir(grafts), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(grafts, []byte(head+" "+w.base+"\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		w.wantRefused("outside what a delta review covers")
	})
}

// A delta keeps the frontmatter of every file it covers, byte for byte. A file with
// no frontmatter stays that way, and a change below the block is a delta's.
func TestADeltaKeepsEveryFilesFrontmatter(t *testing.T) {
	const block = "---\nstatus: open\n---\n"
	const body = "\nA sentence of the body.\n"
	world := func(t *testing.T, rel, text string) *deltaWorld {
		t.Helper()
		w := newDeltaWorld(t)
		w.onMain(rel, text)
		if err := w.merge("--no-edit", deltaDefault); err != nil {
			t.Fatal(err)
		}
		return w
	}
	for _, rel := range []string{".abcd/development/roadmap/rfcs/x.md", ".abcd/development/plans/x.md", "docs/how-to/x.md"} {
		for name, c := range map[string]struct{ from, to string }{
			"a value changed":    {block + body, "---\nstatus: superseded\n---\n" + body},
			"a field added":      {block + body, "---\nstatus: open\nowner: x\n---\n" + body},
			"the block added":    {body, block + body},
			"the block removed":  {block + body, body},
			"below a comment":    {"<!-- a note -->\n" + block + body, "<!-- a note -->\n---\nstatus: superseded\n---\n" + body},
			"below a blank line": {"\n" + block + body, "\n---\nstatus: superseded\n---\n" + body},
			"white space alone":  {block + body, "---\nstatus:  open\n---\n" + body},
		} {
			t.Run(name+" in "+rel, func(t *testing.T) {
				w := world(t, rel, c.from)
				w.repo.Write(rel, c.to)
				w.repo.Commit("a change")
				w.wantRefused("frontmatter")
			})
		}
		t.Run("the body below a block in "+rel, func(t *testing.T) {
			w := world(t, rel, block+body)
			w.repo.Write(rel, block+"\nA better sentence of the body.\n")
			w.repo.Commit("a wording fix")
			if err := w.record(); err != nil {
				t.Fatalf("a wording fix below the block was refused: %v", err)
			}
		})
		t.Run("a file with no block in "+rel, func(t *testing.T) {
			w := world(t, rel, body)
			w.repo.Write(rel, "\nA better sentence of the body.\n")
			w.repo.Commit("a wording fix")
			if err := w.record(); err != nil {
				t.Fatalf("a wording fix was refused: %v", err)
			}
		})
	}
}

// The reviewers' pages say a return has one Verdict section and no other line opening
// with the word, so a reviewer is told before a recording refuses the return.
func TestTheReviewerPagesStateOneVerdictLine(t *testing.T) {
	for _, want := range []string{
		"A return has one `### Verdict` section and no other line opening a verdict, whatever qualifies it.",
		"A return so recorded, a delta's too, names the head in its verdict, by its full object name",
	} {
		for _, rel := range []string{"agents/ruthless-reviewer.md", "agents/security-reviewer.md"} {
			data, err := os.ReadFile(filepath.Join("..", "..", "..", filepath.FromSlash(rel)))
			if err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(strings.Join(strings.Fields(string(data)), " "), want) {
				t.Errorf("%s does not say %q", rel, want)
			}
		}
	}
}

// The reviewers' pages and the command page say, each in one sentence, that a
// return's first line is the line that states what it read, and each line as a page
// states it is what a recording takes and a line below the first is not.
func TestTheReviewPagesStateTheFirstLine(t *testing.T) {
	sentenceEnd := regexp.MustCompile(`\.\s+`)
	for _, rel := range []string{"agents/ruthless-reviewer.md", "agents/security-reviewer.md", "commands/land.md"} {
		data, err := os.ReadFile(filepath.Join("..", "..", "..", filepath.FromSlash(rel)))
		if err != nil {
			t.Fatal(err)
		}
		sentences := sentenceEnd.Split(strings.Join(strings.Fields(string(data)), " "), -1)
		for _, want := range []string{"`Reviewed head: <name>`", "`Reviewed delta: <base>..<head>`"} {
			if !slices.ContainsFunc(sentences, func(s string) bool {
				return strings.Contains(s, want) && strings.Contains(strings.ToLower(s), "first line")
			}) {
				t.Errorf("%s has no sentence saying a return's first line reads %s", rel, want)
			}
		}
	}
	w := newDeltaWorld(t)
	head := w.fix()
	body := "\n\n### Verdict\n\nSHIP " + head + "\n"
	for name, pair := range map[string][]ReviewReturn{
		"the line first":  {{Name: "a.md", Text: "Reviewed delta: " + w.base + ".." + head + body}, {Name: "b.md", Text: "Reviewed delta: " + w.base[:9] + ".." + head[:9] + body}},
		"the line second": {{Name: "a.md", Text: "### Analysis\n\nReviewed delta: " + w.base + ".." + head + body}, {Name: "b.md", Text: "Reviewed delta: " + w.base[:9] + ".." + head[:9] + body}},
	} {
		_, err := RecordDeltaReview(w.repo.Root(), head, w.base, pair)
		if (name == "the line first") != (err == nil) {
			t.Errorf("%s: err = %v", name, err)
		}
	}
}

// A delta changes a ledger record's body alone, and never adds, removes or moves a
// record.
func TestADeltaChangesALedgerRecordsBodyAlone(t *testing.T) {
	for _, rel := range []string{ledgerIssue, ledgerWontfix} {
		t.Run("the body of "+rel, func(t *testing.T) {
			w := newDeltaWorld(t)
			w.repo.Write(rel, ledgerHead+ledgerBody+"One more sentence.\n")
			w.repo.Commit("a fix to the body")
			if err := w.record(); err != nil {
				t.Fatalf("a change to a record's body was refused: %v", err)
			}
		})
		for name, body := range map[string]string{
			"a field added":       "---\nid: \"iss-1\"\nseverity: \"major\"\ndeferred_after: \"v1.0.0\"\ndeferral_reason: \"later\"\n---\n" + ledgerBody,
			"a field changed":     "---\nid: \"iss-1\"\nseverity: \"minor\"\n---\n" + ledgerBody,
			"a field removed":     "---\nid: \"iss-1\"\n---\n" + ledgerBody,
			"white space alone":   "---\nid: \"iss-1\"\nseverity:  \"major\"\n---\n" + ledgerBody,
			"a line end alone":    "---\r\nid: \"iss-1\"\r\nseverity: \"major\"\r\n---\r\n" + ledgerBody,
			"a comment added":     "---\nid: \"iss-1\"\nseverity: \"major\" # ok\n---\n" + ledgerBody,
			"a mark before it":    "\ufeff" + ledgerHead + ledgerBody,
			"the block taken out": ledgerBody,
			"the block left open": "---\nid: \"iss-1\"\nseverity: \"major\"\n" + ledgerBody,
		} {
			t.Run(name+" in "+rel, func(t *testing.T) {
				w := newDeltaWorld(t)
				w.repo.Write(rel, body)
				w.repo.Commit("a change to the frontmatter")
				w.wantRefused("frontmatter")
			})
		}
	}
	t.Run("a record with no frontmatter to hold", func(t *testing.T) {
		w := newDeltaWorld(t)
		w.has(".abcd/work/issues/resolved/README.md", "The ledger.\n")
		w.repo.Write(".abcd/work/issues/resolved/README.md", "The ledger, fixed.\n")
		w.repo.Commit("a fix")
		w.wantRefused("frontmatter")
	})
}

// A delta changes the wording of a file already there: on every path it covers, it
// adds, removes and moves no file.
func TestADeltaAddsRemovesAndMovesNoFile(t *testing.T) {
	const text = "A sentence.\n"
	for _, rel := range []string{ledgerIssue, "docs/how-to/page.md", ".abcd/development/plans/x.md", ".abcd/development/roadmap/x.md"} {
		other := strings.TrimSuffix(rel, ".md") + "-other.md"
		t.Run("a new file beside "+rel, func(t *testing.T) {
			w := newDeltaWorld(t)
			w.repo.Write(other, text)
			w.repo.Commit("a new file")
			w.wantRefused("adds or removes")
		})
		t.Run("a file removed: "+rel, func(t *testing.T) {
			w := newDeltaWorld(t)
			w.has(rel, ledgerHead+text)
			w.repo.Remove(rel)
			w.repo.Commit("gone")
			w.wantRefused("adds or removes")
		})
		t.Run("a file moved: "+rel, func(t *testing.T) {
			w := newDeltaWorld(t)
			w.has(rel, ledgerHead+text)
			w.repo.Remove(rel)
			w.repo.Write(other, ledgerHead+text)
			w.repo.Commit("moved")
			w.wantRefused("adds or removes")
		})
	}
	t.Run("a move between status folders", func(t *testing.T) {
		w := newDeltaWorld(t)
		w.repo.Remove(ledgerIssue)
		w.repo.Write(".abcd/work/issues/wontfix/iss-1-x.md", ledgerHead+ledgerBody)
		w.repo.Commit("moved")
		w.wantRefused("adds or removes")
	})
}

// A delta covers plain text alone, on either side of a change.
func TestADeltaCoversOnlyTextALineCountSpeaksFor(t *testing.T) {
	for name, body := range map[string]string{
		"a bare CR":                "one\rtwo\rthree\n",
		"a form feed":              "one\ftwo\n",
		"a vertical tab":           "one\vtwo\n",
		"an escape sequence":       "one\x1b[2Ktwo\n",
		"a delete":                 "one\x7ftwo\n",
		"a next line":              "one\u0085two\n",
		"a line separator":         "one\u2028two\n",
		"a paragraph separator":    "one\u2029two\n",
		"bytes that are not UTF-8": "\xff\xfe\x41\xd8\n",
	} {
		t.Run("added: "+name, func(t *testing.T) {
			w := newDeltaWorld(t)
			w.repo.Write("docs/how-to/feature.md", body)
			w.repo.Commit("a file")
			w.wantRefused("plain text")
		})
	}
	t.Run("on the side a change removes", func(t *testing.T) {
		w := newDeltaWorld(t)
		w.onMain("docs/how-to/old.md", "one\rtwo\n")
		if err := w.merge("--no-edit", deltaDefault); err != nil {
			t.Fatal(err)
		}
		w.repo.Write("docs/how-to/old.md", "one\n")
		w.repo.Commit("make it plain")
		w.wantRefused("plain text")
	})
	for name, body := range map[string]string{
		"CRLF line ends": "one\r\ntwo\r\n",
		"tabs":           "one\ttwo\n",
		"non-ASCII":      "caf\u00e9 \u2014 na\u00efve\n",
		"no final LF":    "one",
		"an empty file":  "",
	} {
		t.Run("accepted: "+name, func(t *testing.T) {
			w := newDeltaWorld(t)
			w.repo.Write("docs/how-to/feature.md", body)
			w.repo.Commit("a file")
			if err := w.record(); err != nil {
				t.Fatalf("plain text was refused: %v", err)
			}
		})
	}
}

// A delta's patches stay within their bytes, summed over the commits.
func TestADeltaOfLongLinesNeedsAFullPair(t *testing.T) {
	t.Run("one line", func(t *testing.T) {
		w := newDeltaWorld(t)
		w.repo.Write("docs/how-to/feature.md", strings.Repeat("x", maxDeltaPatchBytes)+"\n")
		w.repo.Commit("one long line")
		w.wantRefused("past the bound")
	})
	t.Run("summed over commits", func(t *testing.T) {
		w := newDeltaWorld(t)
		for i := range 3 {
			w.repo.Write(fmt.Sprintf("docs/how-to/f%d.md", i), strings.Repeat("x", maxDeltaPatchBytes/2)+"\n")
			w.repo.Commit("a long line")
		}
		w.wantRefused("past the bound")
	})
}

// A move is a removal and an addition, a file git calls binary is refused, and a
// repository's own settings change neither.
func TestWhatGitsCountCouldHideStillCounts(t *testing.T) {
	t.Run("a move is a removal and an addition, whatever the config says", func(t *testing.T) {
		w := newDeltaWorld(t)
		w.repo.Git("config", "diff.renames", "copies")
		w.repo.Git("mv", "docs/how-to/a.md", "docs/how-to/moved.md")
		w.repo.Commit("a move")
		w.wantRefused("adds or removes")
	})
	t.Run("an attribute that makes git call a text file binary", func(t *testing.T) {
		w := newDeltaWorld(t)
		w.onMain(".gitattributes", "*.md -diff\n")
		if err := w.merge("--no-edit", deltaDefault); err != nil {
			t.Fatal(err)
		}
		w.fix()
		w.wantRefused("binary")
	})
}

// The count stands only when it speaks for exactly the paths the raw list names.
func TestGitsCountMustMatchTheRawList(t *testing.T) {
	paths := []string{"docs/a.md", "docs/b.md"}
	if got, err := countLines([]string{"1\t2\tdocs/a.md", "3\t0\tdocs/b.md"}, paths); err != nil || got != 6 {
		t.Fatalf("countLines = %d, %v; want 6", got, err)
	}
	for name, stat := range map[string][]string{
		"a path it does not count":  {"1\t2\tdocs/a.md"},
		"a path the list omits":     {"1\t2\tdocs/a.md", "3\t0\tdocs/b.md", "1\t0\tinternal/x.go"},
		"another path":              {"1\t2\tdocs/a.md", "3\t0\tinternal/x.go"},
		"a binary count":            {"-\t-\tdocs/a.md", "3\t0\tdocs/b.md"},
		"half a binary count":       {"1\t-\tdocs/a.md", "3\t0\tdocs/b.md"},
		"a negative count":          {"-1\t2\tdocs/a.md", "3\t0\tdocs/b.md"},
		"a line that is not parsed": {"1 2 docs/a.md", "3\t0\tdocs/b.md"},
	} {
		t.Run(name, func(t *testing.T) {
			if got, err := countLines(stat, paths); err == nil {
				t.Fatalf("countLines = %d; want a refusal", got)
			}
		})
	}
}

// A delta covers regular files of mode 100644, on both sides of every change.
func TestADeltaOverAnyOtherModeNeedsAFullPair(t *testing.T) {
	chmod := func(t *testing.T, w *deltaWorld, rel string) {
		t.Helper()
		if err := os.Chmod(filepath.Join(w.repo.Root(), filepath.FromSlash(rel)), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	t.Run("a mode change", func(t *testing.T) {
		w := newDeltaWorld(t)
		chmod(t, w, "docs/how-to/run.md")
		w.repo.Commit("make it runnable")
		w.wantRefused("mode")
	})
	t.Run("a new executable file", func(t *testing.T) {
		w := newDeltaWorld(t)
		w.repo.Write("docs/how-to/new.md", "#!/bin/sh\n")
		chmod(t, w, "docs/how-to/new.md")
		w.repo.Commit("a new script")
		w.wantRefused("adds or removes")
	})
	t.Run("a removal and a return at another mode", func(t *testing.T) {
		w := newDeltaWorld(t)
		w.repo.Remove("docs/how-to/run.md")
		w.repo.Commit("remove it")
		w.repo.Write("docs/how-to/run.md", "#!/bin/sh\n")
		chmod(t, w, "docs/how-to/run.md")
		w.repo.Commit("bring it back")
		w.wantRefused("adds or removes")
	})
	t.Run("a symlink", func(t *testing.T) {
		w := newDeltaWorld(t)
		w.repo.Remove("docs/how-to/run.md")
		if err := os.Symlink("a.md", filepath.Join(w.repo.Root(), "docs", "how-to", "run.md")); err != nil {
			t.Fatal(err)
		}
		w.repo.Commit("a link")
		w.wantRefused("mode")
	})
}

// A delta rests on a full pair and nothing else: no receipt at the base, or a
// receipt that is itself a delta's, is a gap.
func TestADeltaWithNoFullPairAtTheBaseNeedsAFullPair(t *testing.T) {
	t.Run("no receipt at the base", func(t *testing.T) {
		w := newDeltaWorld(t)
		w.fix()
		if err := os.Remove(filepath.Join(w.repo.Root(), filepath.FromSlash(ReviewReceiptsRelDir), w.base)); err != nil {
			t.Fatal(err)
		}
		w.wantRefused("no full review pair")
	})
	t.Run("a base resting on a delta", func(t *testing.T) {
		w := newDeltaWorld(t)
		first := w.fix()
		if err := w.record(); err != nil {
			t.Fatal(err)
		}
		w.repo.Write("docs/how-to/b.md", "x\n")
		w.repo.Commit("a second fix")
		head := w.head()
		_, err := RecordDeltaReview(w.repo.Root(), head, first, deltaPair(first, head))
		if err == nil || !strings.Contains(err.Error(), "no full review pair") {
			t.Fatalf("err = %v; want a chain of deltas refused", err)
		}
	})
	t.Run("a receipt that holds no pair", func(t *testing.T) {
		w := newDeltaWorld(t)
		w.fix()
		p := filepath.Join(w.repo.Root(), filepath.FromSlash(ReviewReceiptsRelDir), w.base)
		if err := os.WriteFile(p, []byte("{}\n"), 0o600); err != nil {
			t.Fatal(err)
		}
		w.wantRefused("no full review pair")
	})
	t.Run("the base itself", func(t *testing.T) {
		w := newDeltaWorld(t)
		_, err := RecordDeltaReview(w.repo.Root(), w.base, w.base, deltaPair(w.base, w.base))
		if err == nil || !strings.Contains(err.Error(), "itself") {
			t.Fatalf("err = %v; want a delta from a head to itself refused", err)
		}
	})
}

// A delta recorded over a head's full pair would weaken it, and any delta
// resting on that head would lose its base.
func TestADeltaNeverReplacesAFullPair(t *testing.T) {
	w := newDeltaWorld(t)
	head := w.fix()
	if _, err := RecordReview(w.repo.Root(), head, fullPair(head)); err != nil {
		t.Fatal(err)
	}
	if _, err := RecordDeltaReview(w.repo.Root(), head, w.base, deltaPair(w.base, head)); err == nil || !strings.Contains(err.Error(), "already") {
		t.Fatalf("err = %v; want a refusal saying the head already has a full pair", err)
	}
	p := readReviewProof(w.repo.Root(), head, deltaDefault)
	if !p.reviewed || p.note != "" {
		t.Fatalf("the full pair's receipt read as %+v", p)
	}
}

// A delta return is not half of a full pair: without a base named, a return that
// states a delta is refused, marked up or not.
func TestARecordedFullPairRefusesADeltaReturn(t *testing.T) {
	w := newDeltaWorld(t)
	head := w.fix()
	for name, line := range map[string]string{
		"plain":       "Reviewed delta: " + w.base + ".." + head,
		"abbreviated": "Reviewed delta: " + w.base[:9] + ".." + head[:9],
		"marked up":   "- **Reviewed delta:** `" + w.base[:9] + ".." + head[:9] + "`",
		"in a quote":  "> Reviewed delta: " + w.base[:9] + ".." + head[:9],
		"indented":    "    Reviewed delta: " + w.base[:9] + ".." + head[:9],
		"other case":  "reviewed delta: " + w.base[:9] + ".." + head[:9],
		"no range":    "Reviewed delta: the fix",
		"bold words":  "**Reviewed delta**: " + w.base[:9] + ".." + head[:9],
		"bold item":   "- **Reviewed delta**: `" + w.base[:9] + ".." + head[:9] + "`",
		"heading":     "## Reviewed delta: " + w.base[:9] + ".." + head[:9],
		"numbered":    "1. Reviewed delta: " + w.base[:9] + ".." + head[:9],
		"table row":   "| Reviewed delta: | " + w.base[:9] + ".." + head[:9] + " |",
		"two spaces":  "Reviewed  delta: " + w.base[:9] + ".." + head[:9],
		"a space":     "Reviewed delta : " + w.base[:9] + ".." + head[:9],
		"no colon":    "Reviewed delta " + w.base[:9] + ".." + head[:9],
	} {
		t.Run(name, func(t *testing.T) {
			_, err := RecordReview(w.repo.Root(), head, []ReviewReturn{
				{Name: "ruthless.md", Text: reviewReturn(head, "SHIP")},
				{Name: "security.md", Text: line + "\n\n" + reviewReturn(head, "APPROVE")},
			})
			if err == nil || !strings.Contains(err.Error(), "security.md") || !strings.Contains(err.Error(), "Reviewed delta") {
				t.Fatalf("err = %v; want security.md refused for stating a delta", err)
			}
			if got, _ := reviewReceipt(w.repo.Root(), head); got != "" {
				t.Fatalf("a refused pair left a receipt at %s", got)
			}
		})
	}
	// The words in a sentence are not the line.
	if _, err := RecordReview(w.repo.Root(), head, []ReviewReturn{
		{Name: "ruthless.md", Text: reviewReturn(head, "SHIP")},
		{Name: "security.md", Text: reviewReturn(head, "APPROVE") + "\nAn earlier note said Reviewed delta: none.\n"},
	}); err != nil {
		t.Fatalf("a full return that mentions the words was refused: %v", err)
	}
}

// A delta refused for its size or its paths needs a full pair, and a delta's
// returns are refused for one.
func TestARefusedDeltasReturnsDoNotMakeAFullPair(t *testing.T) {
	w := newDeltaWorld(t)
	w.repo.Write("internal/x.go", "package x\n")
	w.repo.Commit("code, so no delta covers it")
	head := w.head()
	returns := deltaPair(w.base, head)
	if _, err := RecordDeltaReview(w.repo.Root(), head, w.base, returns); err == nil {
		t.Fatal("the delta was recorded")
	}
	if _, err := RecordReview(w.repo.Root(), head, returns); err == nil || !strings.Contains(err.Error(), "Reviewed delta") {
		t.Fatalf("err = %v; want the delta's returns refused for a full pair", err)
	}
	if got, _ := reviewReceipt(w.repo.Root(), head); got != "" {
		t.Fatalf("the returns left a receipt at %s", got)
	}
}

// A receipt with no delta counts as a full pair only when it holds one for the
// head it names.
func TestAReceiptHoldingNoPairProvesNothing(t *testing.T) {
	for name, body := range map[string]string{
		"empty":             "{}\n",
		"no reviews":        `{"commit":"@HEAD@"}`,
		"one review":        `{"commit":"@HEAD@","reviews":[{"name":"a","verdict":"SHIP","sha256":"x"}]}`,
		"a failing verdict": `{"commit":"@HEAD@","reviews":[{"name":"a","verdict":"SHIP","sha256":"x"},{"name":"b","verdict":"BLOCK","sha256":"y"}]}`,
		"another commit":    `{"commit":"0123456789abcdef0123456789abcdef01234567","reviews":[{"name":"a","verdict":"SHIP","sha256":"x"},{"name":"b","verdict":"APPROVE","sha256":"y"}]}`,
	} {
		t.Run(name, func(t *testing.T) {
			w := newDeltaWorld(t)
			head := w.fix()
			p := filepath.Join(w.repo.Root(), filepath.FromSlash(ReviewReceiptsRelDir), head)
			if err := os.WriteFile(p, []byte(strings.ReplaceAll(body, "@HEAD@", head)), 0o600); err != nil {
				t.Fatal(err)
			}
			got := readReviewProof(w.repo.Root(), head, deltaDefault)
			if got.reviewed || got.unreadable == "" {
				t.Fatalf("a receipt holding no pair read as %+v", got)
			}
			// It does not stop a delta being recorded over it, and the delta
			// then stands.
			if err := w.record(); err != nil {
				t.Fatalf("a receipt holding no pair blocked the delta: %v", err)
			}
		})
	}
}

func TestAReceiptThatDoesNotParseProvesNothing(t *testing.T) {
	w := newDeltaWorld(t)
	head := w.fix()
	p := filepath.Join(w.repo.Root(), filepath.FromSlash(ReviewReceiptsRelDir), head)
	if err := os.WriteFile(p, []byte("not a receipt"), 0o600); err != nil {
		t.Fatal(err)
	}
	if got := readReviewProof(w.repo.Root(), head, deltaDefault); got.reviewed || got.unreadable == "" {
		t.Fatalf("an unreadable receipt read as %+v", got)
	}
}

// The merge gate asks the same questions at landing: a delta stands there on
// its full pair, and stops standing when its full pair goes.
func TestLandArmsOnADeltaOnlyWhileItStands(t *testing.T) {
	world := func(t *testing.T) (*fixture, string) {
		f := newFixtureFiles(t, queueRuleset, map[string]string{".abcd/config.json": autoConfig})
		if err := os.MkdirAll(filepath.Join(f.repo.Root(), filepath.FromSlash(TierRelDir)), 0o700); err != nil {
			t.Fatal(err)
		}
		f.repo.Write("docs/how-to/feature.md", "the change\n")
		f.repo.Commit("the change")
		base := f.head()
		if _, err := RecordReview(f.repo.Root(), base, fullPair(base)); err != nil {
			t.Fatal(err)
		}
		f.repo.Write("docs/how-to/feature.md", "the change, fixed\n")
		f.repo.Commit("a one-line fix")
		if _, err := RecordDeltaReview(f.repo.Root(), f.head(), base, deltaPair(base, f.head())); err != nil {
			t.Fatal(err)
		}
		return f, base
	}
	t.Run("armed", func(t *testing.T) {
		f, base := world(t)
		s := landGate(t, f)
		wantArmed(t, f, s)
		if !strings.Contains(s.Note, "delta") || !strings.Contains(s.Note, short(base)) {
			t.Fatalf("enqueue step %+v: want it to say the head rests on %s", s, short(base))
		}
	})
	t.Run("its full pair gone", func(t *testing.T) {
		f, base := world(t)
		if err := os.Remove(filepath.Join(f.repo.Root(), filepath.FromSlash(ReviewReceiptsRelDir), base)); err != nil {
			t.Fatal(err)
		}
		wantLeft(t, f, landGate(t, f), "no full review pair")
	})
}

// A receipt that holds no standing review is the gate refusing the head, not a read
// that failed: only receipts the gate could not list leave an arming as it is.
func TestAReceiptThatProvesNothingIsARefusalOfTheHead(t *testing.T) {
	for name, body := range map[string]func(head string) string{
		"does not parse": func(string) string { return "not a receipt" },
		"holds no pair":  func(string) string { return "{}\n" },
		"names another": func(string) string {
			return `{"commit":"0123456789abcdef0123456789abcdef01234567","reviews":[{"name":"a","verdict":"SHIP","sha256":"x"},{"name":"b","verdict":"APPROVE","sha256":"y"}]}`
		},
		"a delta refused": func(head string) string {
			return `{"commit":"` + head + `","delta":{"base":"0123456789abcdef0123456789abcdef01234567"},"verdict_rule":2,"reviews":[{"name":"a","verdict":"SHIP","sha256":"x"},{"name":"b","verdict":"APPROVE","sha256":"y"}]}`
		},
	} {
		t.Run(name, func(t *testing.T) {
			f := newFixtureFiles(t, queueRuleset, map[string]string{".abcd/config.json": autoConfig})
			f.forge.head = f.head()
			p := filepath.Join(f.repo.Root(), filepath.FromSlash(ReviewReceiptsRelDir), f.head())
			if err := os.MkdirAll(filepath.Dir(p), 0o700); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(p, []byte(body(f.head())), 0o600); err != nil {
				t.Fatal(err)
			}
			tip := f.repo.Git("rev-parse", "origin/main")
			d, err := DecideMerge(MergeQuery{RepoRoot: f.repo.Root(), Roots: f.options().Roots, Tip: tip, TipName: "origin/main", Default: "main",
				Repo: "o/r", PR: "7", Head: f.head(), Forge: func(args ...string) (string, error) { return f.forge.run("", args...) }})
			if err != nil {
				t.Fatal(err)
			}
			if d.Arm || d.Unread {
				t.Fatalf("decision %+v: want the head refused, and no unread answer", d)
			}
		})
	}
}

// gateAt asks the merge gate about a fixture's head.
func gateAt(t *testing.T, f *fixture) MergeDecision {
	t.Helper()
	d, err := DecideMerge(MergeQuery{RepoRoot: f.repo.Root(), Roots: f.options().Roots, Tip: f.repo.Git("rev-parse", "origin/main"), TipName: "origin/main", Default: "main",
		Repo: "o/r", PR: "7", Head: f.head(), Forge: func(args ...string) (string, error) { return f.forge.run("", args...) }})
	if err != nil {
		t.Fatal(err)
	}
	return d
}

// A receipt written where the gate looks, at a mode this account can read or not.
func writeReceiptAt(t *testing.T, root, sha, body string, mode os.FileMode) string {
	t.Helper()
	p := filepath.Join(root, filepath.FromSlash(ReviewReceiptsRelDir), sha)
	if err := os.MkdirAll(filepath.Dir(p), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(p, mode); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chmod(p, 0o600) })
	return p
}

func fullPairBody(sha string) string {
	return `{"commit":"` + sha + `","verdict_rule":2,"reviews":[{"name":"a","verdict":"SHIP","sha256":"x"},{"name":"b","verdict":"APPROVE","sha256":"y"}]}`
}

// Reading a file at mode 000 fails only for an account that is not root, so a test
// that needs the read to fail says so and stands down under root.
func needsUnreadableFiles(t *testing.T) {
	t.Helper()
	if os.Geteuid() == 0 {
		t.Skip("root reads a file at mode 000")
	}
}

// When the gate could not judge a receipt, a valid one may lie behind the failure, so
// the answer is unread: nothing is armed, and nothing is withdrawn on it.
func TestAReceiptTheGateCouldNotJudgeIsAnUnreadAnswer(t *testing.T) {
	world := func(t *testing.T) *fixture {
		t.Helper()
		f := newFixtureFiles(t, queueRuleset, map[string]string{".abcd/config.json": autoConfig})
		f.forge.head = f.head()
		return f
	}
	wantUnread := func(t *testing.T, f *fixture) {
		t.Helper()
		if d := gateAt(t, f); d.Arm || !d.Unread {
			t.Fatalf("decision %+v: want nothing armed, and an unread answer", d)
		}
	}
	t.Run("a valid receipt that cannot be read", func(t *testing.T) {
		needsUnreadableFiles(t)
		f := world(t)
		writeReceiptAt(t, f.repo.Root(), f.head(), fullPairBody(f.head()), 0)
		wantUnread(t, f)
		if d := gateAt(t, f); !strings.Contains(d.Why, "review receipts cannot be read") {
			t.Fatalf("why = %q; want it to say the receipts could not be read", d.Why)
		}
	})
	t.Run("git cannot say whether a receipt is tracked", func(t *testing.T) {
		f := world(t)
		writeReceiptAt(t, f.repo.Root(), f.head(), fullPairBody(f.head()), 0o600)
		if err := os.WriteFile(filepath.Join(f.repo.Root(), ".git", "index"), []byte("not an index"), 0o600); err != nil {
			t.Fatal(err)
		}
		wantUnread(t, f)
	})
	t.Run("git cannot answer for a delta", func(t *testing.T) {
		f := world(t)
		const base = "0123456789abcdef0123456789abcdef01234567"
		writeReceiptAt(t, f.repo.Root(), base, fullPairBody(base), 0o600)
		writeReceiptAt(t, f.repo.Root(), f.head(), `{"commit":"`+f.head()+`","verdict_rule":2,"delta":{"base":"`+base+`"},"reviews":[{"name":"a","verdict":"SHIP","sha256":"x"},{"name":"b","verdict":"APPROVE","sha256":"y"}]}`, 0o600)
		wantUnread(t, f)
		if d := gateAt(t, f); strings.Contains(d.Why, "does not stand") {
			t.Fatalf("why = %q; a delta git could not judge is not one that does not stand", d.Why)
		}
	})
	t.Run("a delta whose base's receipt cannot be read", func(t *testing.T) {
		needsUnreadableFiles(t)
		f := world(t)
		base := f.head()
		writeReceiptAt(t, f.repo.Root(), base, fullPairBody(base), 0)
		f.repo.Write("docs/how-to/feature.md", "the change\n")
		f.repo.Commit("a fix")
		f.forge.head = f.head()
		writeReceiptAt(t, f.repo.Root(), f.head(), `{"commit":"`+f.head()+`","verdict_rule":2,"delta":{"base":"`+base+`"},"reviews":[{"name":"a","verdict":"SHIP","sha256":"x"},{"name":"b","verdict":"APPROVE","sha256":"y"}]}`, 0o600)
		wantUnread(t, f)
	})
	t.Run("a receipt found wanting beside one that cannot be read", func(t *testing.T) {
		needsUnreadableFiles(t)
		f := world(t)
		writeReceiptAt(t, f.repo.Root(), f.head(), "not a receipt", 0o600)
		other := filepath.Join(t.TempDir(), "lane")
		f.repo.Git("worktree", "add", "-q", "--detach", other, "HEAD")
		writeReceiptAt(t, other, f.head(), fullPairBody(f.head()), 0)
		wantUnread(t, f)
	})
	t.Run("a failed listing", func(t *testing.T) {
		proof := readReviewProof(t.TempDir(), "0123456789abcdef0123456789abcdef01234567", "")
		if proof.reviewed || !proof.unread || proof.unreadable == "" {
			t.Fatalf("proof = %+v; want the listing's failure read as unread", proof)
		}
	})
	// A valid receipt behind a failed read keeps its arming, and comes back when the
	// read does.
	t.Run("the arming is left, and the receipt counts once it can be read", func(t *testing.T) {
		needsUnreadableFiles(t)
		f := world(t)
		p := writeReceiptAt(t, f.repo.Root(), f.head(), fullPairBody(f.head()), 0)
		if refused, unread := gateAtLocalHead(&Result{Default: "main", Head: f.head()}, f.options(), f.repo.Root(), "refs/remotes/origin/main", "o/r", "7", func(args ...string) (string, error) { return f.forge.run("", args...) }); refused != "" || unread == "" {
			t.Fatalf("refused %q, unread %q: want the gate unread, so nothing is withdrawn", refused, unread)
		}
		if err := os.Chmod(p, 0o600); err != nil {
			t.Fatal(err)
		}
		if d := gateAt(t, f); !d.Arm || d.Unread {
			t.Fatalf("decision %+v: want the head armed once its receipt can be read", d)
		}
	})
}

// A receipt read and found wanting is the gate refusing the head, and it says what is
// wrong rather than that it could not read one.
func TestAReceiptFoundWantingSaysWhatIsWrong(t *testing.T) {
	f := newFixtureFiles(t, queueRuleset, map[string]string{".abcd/config.json": autoConfig})
	f.forge.head = f.head()
	for body, reason := range map[string]string{
		"not a receipt":                        "does not parse",
		"{}\n":                                 "holds no passing pair",
		strings.Repeat("x", maxReceiptBytes+1): "exceeds size cap",
	} {
		writeReceiptAt(t, f.repo.Root(), f.head(), body, 0o600)
		d := gateAt(t, f)
		if d.Arm || d.Unread || !strings.Contains(d.Why, "no review receipt for "+short(f.head())+" stands") || !strings.Contains(d.Why, reason) || strings.Contains(d.Why, "cannot be read") {
			t.Fatalf("decision %+v: want the head refused, naming %q", d, reason)
		}
	}
	// A receipt a commit carries is refused for that, with git having answered.
	rel := ReviewReceiptsRelDir + "/" + f.head()
	f2 := newFixtureFiles(t, queueRuleset, map[string]string{".abcd/config.json": autoConfig})
	head := f2.head()
	writeReceiptAt(t, f2.repo.Root(), head, fullPairBody(head), 0o600)
	f2.repo.Git("add", "-f", "--", ReviewReceiptsRelDir+"/"+head)
	f2.repo.Git("commit", "-q", "-m", "an unrelated change")
	proof := readReviewProof(f2.repo.Root(), head, "")
	if proof.reviewed || proof.unread || !strings.Contains(proof.unreadable, "git tracks") {
		t.Fatalf("a tracked receipt read as %+v; want it refused, not unread (%s)", proof, rel)
	}
}

func TestTheGateNamesADeltaThatDoesNotStand(t *testing.T) {
	d := mergeGate(gateFacts{mode: Facilitator{Mode: ModeAuto, Layer: "repo", Origin: "x"}, head: "0123456789abcdef", role: RoleMaintainer, roleWhy: "write", deltaWhy: "past the bound"})
	if d.arm || !strings.Contains(d.why, "past the bound") || !strings.Contains(d.why, "full review pair") {
		t.Fatalf("decision = %+v", d)
	}
}

// A delta's verdict names the head too: the delta's own, not its base.
func TestADeltaVerdictNamesTheHead(t *testing.T) {
	w := newDeltaWorld(t)
	head := w.fix()
	first := "Reviewed delta: " + w.base + ".." + head + "\n\n"
	for name, section := range map[string]string{"no head": "SHIP", "its base": "SHIP " + w.base} {
		t.Run(name, func(t *testing.T) {
			_, err := RecordDeltaReview(w.repo.Root(), head, w.base, []ReviewReturn{
				{Name: "ruthless.md", Text: first + "### Verdict\n\n" + section + "\n"},
				{Name: "security.md", Text: first + "### Verdict\n\nAPPROVE " + head + "\n"},
			})
			if err == nil || !strings.Contains(err.Error(), "ruthless.md") {
				t.Fatalf("err = %v; want ruthless.md refused", err)
			}
		})
	}
	if _, err := RecordDeltaReview(w.repo.Root(), head, w.base, []ReviewReturn{
		{Name: "ruthless.md", Text: first + "### Verdict\n\nSHIP " + head + "\n"},
		{Name: "security.md", Text: first + "### Verdict\n\nAPPROVE " + head + "\n"},
	}); err != nil {
		t.Fatalf("a delta naming its head was refused: %v", err)
	}
}

// A receipt says which reading of a verdict it was recorded under, and the gate counts
// only this one's: its returns are not kept, so an earlier reading cannot be read
// again. The head records its pair again, and a delta cannot rest on it.
func TestAReceiptStandsOnlyUnderTheReadingThatNamesTheHead(t *testing.T) {
	w := newDeltaWorld(t)
	head := w.fix()
	earlier := func(sha string) string {
		return `{"commit":"` + sha + `","reviews":[{"name":"a","verdict":"SHIP","sha256":"x"},{"name":"b","verdict":"APPROVE","sha256":"y"}]}`
	}
	writeReceiptAt(t, w.repo.Root(), head, earlier(head), 0o600)
	proof := readReviewProof(w.repo.Root(), head, w.base)
	if proof.reviewed || proof.unread || !strings.Contains(proof.unreadable, "again") {
		t.Fatalf("a receipt from before the head was named: %+v", proof)
	}
	// A delta's receipt from before it is judged the same way.
	writeReceiptAt(t, w.repo.Root(), head, `{"commit":"`+head+`","delta":{"base":"`+w.base+`"},"reviews":[{"name":"a","verdict":"SHIP","sha256":"x"},{"name":"b","verdict":"APPROVE","sha256":"y"}]}`, 0o600)
	proof = readReviewProof(w.repo.Root(), head, w.base)
	if proof.reviewed || proof.unread || !strings.Contains(proof.deltaWhy, "again") {
		t.Fatalf("a delta's receipt from before the head was named: %+v", proof)
	}
	// A delta does not rest on one.
	writeReceiptAt(t, w.repo.Root(), w.base, earlier(w.base), 0o600)
	if err := hasFullPair(w.repo.Root(), w.base); err == nil || !strings.Contains(err.Error(), "again") {
		t.Fatalf("a delta rests on a receipt from before the head was named: %v", err)
	}
	// Recording the pair again replaces it, and stands.
	if _, err := RecordReview(w.repo.Root(), w.base, fullPair(w.base)); err != nil {
		t.Fatal(err)
	}
	if err := hasFullPair(w.repo.Root(), w.base); err != nil {
		t.Fatalf("the pair recorded again: %v", err)
	}
}
