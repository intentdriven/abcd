package main

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/gittest"
)

const (
	longIss  = "a-record-s-file-name-has-no-cap-tied-to-the-length-of-its"
	shortIss = "a-record-s-file-name-has-no-cap-tied-to"
	longItd  = "the-general-rewriter-shortens-a-title-so-the-cut-reads-well"
	shortItd = "the-general-rewriter-shortens-a-title-so"
	longADR  = "a-reading-is-commissioned-about-something-so-the-invocation"
	shortADR = "a-reading-is-commissioned-about"
)

// fixture lays a small committed repository holding one long-slugged record in
// each family, one short-slugged record, and files that name them.
func fixture(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	files := map[string]string{
		".abcd/work/issues/open/iss-2610100626320367-" + longIss + ".md":           "---\nschema_version: 1\nid: \"iss-2610100626320367\"\nslug: \"" + longIss + "\"\n---\n\nBody naming [the intent](../../../development/intents/drafts/itd-2610100627454580-" + longItd + ".md).\n",
		".abcd/work/issues/resolved/iss-2609010000000001-short-one.md":             "---\nid: \"iss-2609010000000001\"\nslug: \"short-one\"\n---\n\nslug: not-frontmatter-" + longIss + "\n",
		".abcd/development/intents/drafts/itd-2610100627454580-" + longItd + ".md": "---\nid: itd-2610100627454580\nslug: " + longItd + "\n---\n\n# Title\n",
		".abcd/development/specs/open/spc-2610100627454581-" + longItd + ".md":     "---\nid: spc-2610100627454581\nslug: " + longItd + "\nintent: itd-2610100627454580\n---\n",
		".abcd/development/decisions/adrs/0058-" + longADR + ".md":                 "---\nid: adr-58\nslug: " + longADR + "\n---\n",
		"AGENTS.md": "See .abcd/work/issues/open/iss-2610100626320367-" + longIss + ".md and\n" +
			"[adr](.abcd/development/decisions/adrs/0058-" + longADR + ".md#context).\n" +
			"A derived name iss-2610100626320367-" + longIss + "-review.json is not a reference.\n",
		".abcd/work/DECISIONS.md": "- 2026-10-10 decided per iss-2610100626320367-" + longIss + ".md\n",
		"internal/x/x_test.go":    "package x\n\nconst p = \"spc-2610100627454581-" + longItd + ".md\"\n",
		"docs/prose.md":           "The slug " + longItd + " alone is not a path.\n",
	}
	for rel, body := range files {
		abs := filepath.Join(root, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(abs, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	git(t, root, "init", "-q")
	git(t, root, "add", "-A")
	git(t, root, "-c", "user.name=t", "-c", "user.email=t@example.invalid", "commit", "-q", "-m", "fixture")
	return root
}

func git(t *testing.T, root string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-C", root}, args...)...)
	cmd.Env = gittest.Env(t)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
	return string(out)
}

func read(t *testing.T, root, rel string) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(rel)))
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

// TestPlanNamesEveryLongSlugAndOnlyThose: the plan renames each family's long
// slug to the shared cut and leaves a short one alone.
func TestPlanNamesEveryLongSlugAndOnlyThose(t *testing.T) {
	root := fixture(t)
	plan, err := buildPlan(root)
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]string{
		".abcd/work/issues/open/iss-2610100626320367-" + longIss + ".md":           ".abcd/work/issues/open/iss-2610100626320367-" + shortIss + ".md",
		".abcd/development/intents/drafts/itd-2610100627454580-" + longItd + ".md": ".abcd/development/intents/drafts/itd-2610100627454580-" + shortItd + ".md",
		".abcd/development/specs/open/spc-2610100627454581-" + longItd + ".md":     ".abcd/development/specs/open/spc-2610100627454581-" + shortItd + ".md",
		".abcd/development/decisions/adrs/0058-" + longADR + ".md":                 ".abcd/development/decisions/adrs/0058-" + shortADR + ".md",
	}
	if len(plan) != len(want) {
		t.Fatalf("plan has %d renames, want %d: %+v", len(plan), len(want), plan)
	}
	for _, r := range plan {
		if want[r.OldRel] != r.NewRel {
			t.Errorf("%s -> %s, want %s", r.OldRel, r.NewRel, want[r.OldRel])
		}
	}
}

// TestApplyRenamesRewritesAndReports is the whole program on the fixture: every
// long-slugged record is moved with git, its slug field follows, every
// reference to the old name is rewritten, and what it must not rewrite — a
// derived name, the append-only decision log, a bare slug — is left and
// reported.
func TestApplyRenamesRewritesAndReports(t *testing.T) {
	root := fixture(t)
	rep, err := run(root, true)
	if err != nil {
		t.Fatal(err)
	}
	if rep.Renamed["iss"] != 1 || rep.Renamed["itd"] != 1 || rep.Renamed["spc"] != 1 || rep.Renamed["adr"] != 1 {
		t.Fatalf("renamed per family = %v, want one in each", rep.Renamed)
	}
	// Git records no rename, only the index's paths, so the move is proved by
	// the index: the old path gone from it and the new one staged.
	index := git(t, root, "ls-files", "--", ".abcd/work/issues/open")
	if strings.Contains(index, longIss) || !strings.Contains(index, "iss-2610100626320367-"+shortIss+".md") {
		t.Fatalf("the issue was not moved in the index:\n%s", index)
	}
	if unstaged := git(t, root, "diff", "--name-only"); unstaged != "" {
		t.Fatalf("the run left changes unstaged:\n%s", unstaged)
	}
	if got := read(t, root, ".abcd/work/issues/open/iss-2610100626320367-"+shortIss+".md"); !strings.Contains(got, "slug: \""+shortIss+"\"\n") ||
		!strings.Contains(got, "intents/drafts/itd-2610100627454580-"+shortItd+".md") {
		t.Fatalf("issue slug field or its link was not rewritten:\n%s", got)
	}
	if got := read(t, root, ".abcd/development/intents/drafts/itd-2610100627454580-"+shortItd+".md"); !strings.Contains(got, "slug: "+shortItd+"\n") {
		t.Fatalf("intent slug field not rewritten:\n%s", got)
	}
	if got := read(t, root, ".abcd/work/issues/resolved/iss-2609010000000001-short-one.md"); !strings.Contains(got, "slug: \"short-one\"") ||
		!strings.Contains(got, "slug: not-frontmatter-"+longIss) {
		t.Fatalf("a short record, or a body line, was touched:\n%s", got)
	}
	agents := read(t, root, "AGENTS.md")
	for _, want := range []string{
		".abcd/work/issues/open/iss-2610100626320367-" + shortIss + ".md and",
		"adrs/0058-" + shortADR + ".md#context",
		"iss-2610100626320367-" + longIss + "-review.json",
	} {
		if !strings.Contains(agents, want) {
			t.Errorf("AGENTS.md lacks %q:\n%s", want, agents)
		}
	}
	if got := read(t, root, "internal/x/x_test.go"); !strings.Contains(got, "spc-2610100627454581-"+shortItd+".md") {
		t.Errorf("a path in a test was not rewritten:\n%s", got)
	}
	if got := read(t, root, ".abcd/work/DECISIONS.md"); !strings.Contains(got, longIss) {
		t.Errorf("the append-only decision log was rewritten:\n%s", got)
	}
	// The issue's link to the intent, AGENTS.md's issue path and ADR link, and
	// the test's spec path; the derived name and the decision log are left.
	if rep.References != 4 || rep.Files != 3 {
		t.Errorf("rewrote %d reference(s) in %d file(s), want 4 in 3", rep.References, rep.Files)
	}
	var left bytes.Buffer
	for _, l := range rep.Leftovers {
		left.WriteString(l.File + ": " + l.Slug + " (" + l.Why + ")\n")
	}
	for _, want := range []string{"AGENTS.md: " + longIss, ".abcd/work/DECISIONS.md: " + longIss, "docs/prose.md: " + longItd} {
		if !strings.Contains(left.String(), want) {
			t.Errorf("leftovers lack %q:\n%s", want, left.String())
		}
	}
}

// TestASecondRunChangesNothing: once the rename is committed the program finds
// nothing to do.
func TestASecondRunChangesNothing(t *testing.T) {
	root := fixture(t)
	if _, err := run(root, true); err != nil {
		t.Fatal(err)
	}
	git(t, root, "-c", "user.name=t", "-c", "user.email=t@example.invalid", "commit", "-q", "-m", "rename")
	rep, err := run(root, true)
	if err != nil {
		t.Fatal(err)
	}
	if n := rep.total(); n != 0 || rep.References != 0 {
		t.Fatalf("second run renamed %d and rewrote %d", n, rep.References)
	}
	if s := git(t, root, "status", "--porcelain"); s != "" {
		t.Fatalf("second run left the tree dirty:\n%s", s)
	}
}

// TestApplyRefusesADirtyTree: a rename over uncommitted work would fold that
// work into the rename, so the program refuses and changes nothing.
func TestApplyRefusesADirtyTree(t *testing.T) {
	root := fixture(t)
	if err := os.WriteFile(filepath.Join(root, "stray.txt"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := run(root, true); err == nil {
		t.Fatal("run applied over a dirty tree")
	}
	if _, err := os.Stat(filepath.Join(root, ".abcd/work/issues/open/iss-2610100626320367-"+longIss+".md")); err != nil {
		t.Fatalf("a refused run moved a record: %v", err)
	}
}

// TestPlanRefusesASlugFieldThatDisagrees: a record whose slug field is not its
// filename's slug is refused before anything moves, so the rename is never
// left part way.
func TestPlanRefusesASlugFieldThatDisagrees(t *testing.T) {
	root := fixture(t)
	rel := ".abcd/development/decisions/adrs/0058-" + longADR + ".md"
	if err := os.WriteFile(filepath.Join(root, rel), []byte("---\nid: adr-58\nslug: something-else\n---\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	git(t, root, "-c", "user.name=t", "-c", "user.email=t@example.invalid", "commit", "-q", "-am", "drift")
	if _, err := run(root, true); err == nil || !strings.Contains(err.Error(), "something-else") {
		t.Fatalf("run = %v, want a refusal naming the disagreeing field", err)
	}
	if s := git(t, root, "status", "--porcelain"); s != "" {
		t.Fatalf("a refused run changed the tree:\n%s", s)
	}
}
