package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/ideate"
)

// The store an `ideate record` run writes into is the CHECKOUT's research store,
// and the log its pointer is appended to is the CHECKOUT's decision log —
// wherever in the checkout the caller happens to stand. These are
// iss-2609091729516940's detectors for the ideate family: the front door handed
// its working directory to the core as the repo root, verbatim, so a run from a
// package directory grilled the wrong tree and reported the cited records as
// records that "do not exist in this repository" — a plausible wrong answer,
// blaming the operator's grill for the verb's own misaddressing — and, where the
// caller's directory happened to carry a decision log, wrote the verdict and its
// pointer into a store below the root, reporting repo-relative paths that read
// exactly like the checkout's. Outside every repository it did the same in a
// plain directory. A verdict filed that way reaches no gate, no release cut and
// no reader — and a killed idea nobody can find is an idea that gets proposed
// again.

// ideateStoreFixture builds a git working tree carrying everything a verdict
// record needs — the decision log its pointer is appended to and one planned
// intent a grill hit can legitimately cite — with a package-shaped subdirectory
// two levels down, and chdirs into the root.
func ideateStoreFixture(t *testing.T) (repo, sub string) {
	t.Helper()
	t.Setenv("HOME", t.TempDir())
	repo = t.TempDir()
	gitInitAt(t, repo)
	repo = realPath(t, repo)
	seedIdeateStore(t, repo)
	sub = filepath.Join(repo, "internal", "core")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	t.Chdir(repo)
	return repo, sub
}

// seedIdeateStore lays out everything a verdict record needs in a tree: the
// decision log its pointer is appended to, and one planned intent a grill hit
// can legitimately cite. It is used on the NON-repository fixtures too, so that
// a refusal there is provably the root resolution's and not the core's own
// missing-decision-log or unresolvable-citation refusal wearing its clothes.
func seedIdeateStore(t *testing.T, root string) {
	t.Helper()
	writeRepoFile(t, root, ideate.DecisionsRelDir, "# Decisions\n")
	writeRepoFile(t, root, ".abcd/development/intents/planned/itd-104-ideate.md",
		"---\nid: itd-104\nslug: ideate\nspec_id: spc-18\nimpact: additive\n---\n\n# itd-104\n")
}

// recordedVerdict returns the paths a landed verdict occupies in the checkout:
// the dated research note, and the decision log that must now point at it.
func recordedVerdict(t *testing.T, repo, slug string) (note string) {
	t.Helper()
	dir := filepath.Join(repo, filepath.FromSlash(ideate.ResearchRelDir))
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("the checkout's research store holds no verdict: %v", err)
	}
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), "-ideate-"+slug+".md") {
			return filepath.Join(dir, e.Name())
		}
	}
	t.Fatalf("no verdict record for %q under %s: %v", slug, ideate.ResearchRelDir, entries)
	return ""
}

// TestIdeateRecordFromSubdirectoryWritesIntoTheCheckoutStore is the write half
// of the reproduction: the verdict and its decision-log pointer must land in the
// CHECKOUT's store, and no second store may appear under the subdirectory the
// caller happened to stand in. It also pins the grill: the cited record resolves,
// because the grill reads the checkout's record and not the caller's directory.
func TestIdeateRecordFromSubdirectoryWritesIntoTheCheckoutStore(t *testing.T) {
	repo, sub := ideateStoreFixture(t)
	payload := writeVerdict(t, ideateVerdictJSON("itd-104"))

	t.Chdir(sub)
	out, err := runCLIErr(t, "ideate", "record", "filed-from-a-package-directory", "--verdict-json", payload)
	if err != nil {
		t.Fatalf("`abcd ideate record` from a package directory: %v\n%s", err, out)
	}
	note := recordedVerdict(t, repo, "filed-from-a-package-directory")
	if _, serr := os.Stat(note); serr != nil {
		t.Errorf("the verdict record is not in the checkout's research store: %v", serr)
	}
	log, rerr := os.ReadFile(filepath.Join(repo, filepath.FromSlash(ideate.DecisionsRelDir)))
	if rerr != nil {
		t.Fatal(rerr)
	}
	if !strings.Contains(string(log), "filed-from-a-package-directory") {
		t.Errorf("the checkout's decision log gained no pointer at the verdict:\n%s", log)
	}
	if _, serr := os.Stat(filepath.Join(sub, ".abcd")); serr == nil {
		t.Errorf("`ideate record` laid a second store at %s/.abcd — a verdict filed there reaches no gate, no release cut and no reader", sub)
	}
}

// TestIdeateRecordOutsideAnyRepositoryRefuses pins the no-repository ruling: with
// no checkout anywhere above, the run REFUSES with exit 2 and writes nothing —
// and refuses for the RIGHT reason, naming the missing repository rather than the
// missing decision log, which is what a caller standing outside a checkout was
// told before.
func TestIdeateRecordOutsideAnyRepositoryRefuses(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	plain := realPath(t, t.TempDir())
	payload := writeVerdict(t, ideateVerdictJSON("itd-104"))
	// Everything the core demands is present — the decision log its pointer is
	// appended to, and the record its grill hit cites — so the ONLY thing
	// standing between this run and a written verdict is the root resolution.
	seedIdeateStore(t, plain)
	t.Chdir(plain)

	out, err := runCLIErr(t, "ideate", "record", "outside-any-repository", "--verdict-json", payload)
	if err == nil {
		t.Fatalf("`abcd ideate record` outside a repository succeeded; it must refuse:\n%s", out)
	}
	if code := exitCodeOf(err); code != 2 {
		t.Errorf("`abcd ideate record` outside a repository exited %d, want 2", code)
	}
	if !strings.Contains(err.Error(), "not inside a git repository") {
		t.Errorf("the refusal does not name the missing repository as the reason: %v", err)
	}
	if _, serr := os.Stat(filepath.Join(plain, filepath.FromSlash(ideate.ResearchRelDir))); serr == nil {
		t.Errorf("a refused run still laid a research store at %s — the refusal must write nothing", ideate.ResearchRelDir)
	}
}

// TestIdeateRecordRefusesARepoShapedTreeGitWillNotAnswerFor pins the third state
// a repo root can be in. A directory carrying a .git that git will not answer for
// is repo-SHAPED, and a marker walk would hand it back as a root without checking
// either its shape or its owner (iss-2609090947359464). Resolving this front door
// must not make such a walk live.
func TestIdeateRecordRefusesARepoShapedTreeGitWillNotAnswerFor(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	planted := realPath(t, t.TempDir())
	if err := os.Mkdir(filepath.Join(planted, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	seedIdeateStore(t, planted)
	payload := writeVerdict(t, ideateVerdictJSON("itd-104"))
	t.Chdir(planted)

	out, err := runCLIErr(t, "ideate", "record", "beneath-a-planted-marker", "--verdict-json", payload)
	if err == nil {
		t.Fatalf("a repo-shaped tree git will not answer for was accepted as a root:\n%s", out)
	}
	if !strings.Contains(err.Error(), "git could not name the repository root") {
		t.Errorf("the refusal does not say that git could not answer: %v", err)
	}
	if _, serr := os.Stat(filepath.Join(planted, filepath.FromSlash(ideate.ResearchRelDir))); serr == nil {
		t.Errorf("the refusal still laid a research store under the planted marker")
	}
}

// TestIdeateRecordFromTheCheckoutRootIsUnchanged is the anti-vacuity control: the
// resolution must be invisible where the caller already stood at the root.
func TestIdeateRecordFromTheCheckoutRootIsUnchanged(t *testing.T) {
	repo, _ := ideateStoreFixture(t)

	out, err := runCLIErr(t, "ideate", "record", "filed-from-the-root", "--verdict-json", writeVerdict(t, ideateVerdictJSON("itd-104")))
	if err != nil {
		t.Fatalf("`abcd ideate record` at the checkout root: %v\n%s", err, out)
	}
	body, rerr := os.ReadFile(recordedVerdict(t, repo, "filed-from-the-root"))
	if rerr != nil {
		t.Fatal(rerr)
	}
	if !strings.Contains(string(body), "survives") {
		t.Errorf("the verdict record does not carry its verdict:\n%s", body)
	}
}

// TestIdeateRecordReportsAStrayResearchStoreBelowTheCheckoutRoot: a store this
// defect already laid under a subdirectory holds verdicts nobody will ever see
// again, because the verb now correctly addresses the checkout's. The resolution
// therefore SAYS the stray store is there, on stderr. It reports; it moves
// nothing.
func TestIdeateRecordReportsAStrayResearchStoreBelowTheCheckoutRoot(t *testing.T) {
	repo, sub := ideateStoreFixture(t)
	stray := filepath.Join(sub, filepath.FromSlash(ideate.ResearchRelDir))
	if err := os.MkdirAll(stray, 0o755); err != nil {
		t.Fatal(err)
	}
	orphan := filepath.Join(stray, "2026-09-09-ideate-an-orphaned-verdict.md")
	if err := os.WriteFile(orphan, []byte("# an orphaned verdict\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	t.Chdir(sub)
	out, err := runCLIErr(t, "ideate", "record", "filed-beside-a-stray-store", "--verdict-json", writeVerdict(t, ideateVerdictJSON("itd-104")))
	if err != nil {
		t.Fatalf("ideate record: %v\n%s", err, out)
	}
	if !strings.Contains(string(out), filepath.Join("internal", "core", ".abcd")) {
		t.Errorf("the run did not report the stray research store under %s:\n%s",
			strings.TrimPrefix(sub, repo+string(filepath.Separator)), out)
	}
	if _, serr := os.Stat(orphan); serr != nil {
		t.Errorf("the report moved or removed the orphaned record: %v", serr)
	}
}

// TestIdeateCallsNeverTakeTheRawWorkingDirectory is the static half: EVERY call
// into the ideate core that takes a repo root must take the RESOLVED checkout
// root, never the caller's working directory. Both halves are read out of the
// source, so a second ideate front door written the old way fails here rather
// than shipping.
func TestIdeateCallsNeverTakeTheRawWorkingDirectory(t *testing.T) {
	assertCoreRootsAreResolved(t, coreRootCheck{
		pkg:     "ideate",
		coreDir: filepath.Join("..", "..", "core", "ideate"),
		door:    "ideateStoreRoot",
	})
}
