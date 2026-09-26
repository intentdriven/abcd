package loop

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/gittest"
)

const (
	agentsMarked = "# AGENTS.md\n\nRepo facts outside the section.\n\n" +
		"<!-- working-conventions 2026-09-26 -->\n## Working conventions\n\n- Commit on a branch; never push.\n" +
		"<!-- /working-conventions -->\n\nMore facts outside.\n"
	adrRel       = ".abcd/development/decisions/adrs/0027-the-autonomous-run-is-a-seam.md"
	adrBody      = "---\nid: adr-27\nstatus: accepted\n---\n# The autonomous run is a seam\n"
	decisionsLog = "# DECISIONS\n\nHeader prose that names itd-10 outside any entry.\n\n" +
		"- 2026-09-20 — The loop is a state file (itd-10),\n  continued on a second line.\n" +
		"- 2026-09-21 — Something else entirely, itd-100 only.\n" +
		"- 2026-09-22 — The spec spc-1 takes steps.\n"
)

// briefRepo is loopRepo with the record a brief is rendered from committed on
// the default branch: the conventions, one cited ADR and the decision log.
func briefRepo(t *testing.T, agents string) *gittest.Repo {
	t.Helper()
	repo := loopRepo(t, readyIntent("", settledQuestions+"Refines adr-27, and adr-404 which does not exist.\n\n"), specWithSteps(""))
	if agents != "" {
		repo.Write("AGENTS.md", agents)
	}
	repo.Write(adrRel, adrBody)
	repo.Write(".abcd/work/DECISIONS.md", decisionsLog)
	repo.Commit("the record")
	return repo
}

// advanceTo steps the run until its current lane's next step is want.
func advanceTo(t *testing.T, repo *gittest.Repo, runID string, want StepName) StepResult {
	t.Helper()
	var res StepResult
	for range len(Sequence) {
		st, err := ReadState(repo.Root(), runID)
		if err != nil {
			t.Fatal(err)
		}
		if i := st.current(); i >= 0 && st.Lanes[i].Step == want {
			return res
		}
		if res, err = Advance(repo.Root(), runID, DefaultSteps(), Options{}); err != nil {
			t.Fatalf("advancing to %s: %v", want, err)
		}
	}
	t.Fatalf("the lane never reached %s", want)
	return res
}

// TestTheBriefNamesWhatItWasRenderedFrom is criterion 3's brief half: the brief
// file names the intent, the spec and the conventions it was rendered from,
// carries each, and the decisions the intent cites; it tells the implementer
// its lane and where its report, the definition of done's output and its
// receipt go.
func TestTheBriefNamesWhatItWasRenderedFrom(t *testing.T) {
	repo := briefRepo(t, agentsMarked)
	start, err := Start(repo.Root(), "itd-10", Options{})
	if err != nil {
		t.Fatal(err)
	}
	advanceTo(t, repo, start.RunID, StepBrief)
	res, err := Advance(repo.Root(), start.RunID, DefaultSteps(), Options{})
	if err != nil {
		t.Fatal(err)
	}
	if res.Performed != StepBrief || res.Step != StepImplement {
		t.Fatalf("want the brief step performed: %+v", res)
	}
	st, err := ReadState(repo.Root(), start.RunID)
	if err != nil {
		t.Fatal(err)
	}
	l := st.Lanes[0]
	wantRel := RunRelDir + "/" + start.RunID + "/lane-1/" + BriefFileName
	if l.Brief != wantRel {
		t.Fatalf("the brief lives in the lane's directory: %q, want %q", l.Brief, wantRel)
	}
	path := filepath.Join(repo.Root(), filepath.FromSlash(l.Brief))
	fi, err := os.Stat(path)
	if err != nil || fi.Mode().Perm() != filePerm {
		t.Fatalf("the brief is the caller's own file: %v %v", fi, err)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	brief := string(raw)
	laneDir := filepath.Join(repo.Root(), filepath.FromSlash(RunRelDir), start.RunID, "lane-1")
	for _, want := range []string{
		"the intent: itd-10, `" + plannedRel + "`",
		"the spec: spc-1, `" + specRel + "`",
		"the conventions: `AGENTS.md`, its working-conventions section",
		"Build spec step 1, \"the whole spec\"",
		l.Worktree, l.Branch, l.BaseSHA,
		filepath.Join(laneDir, ReportFileName),
		filepath.Join(laneDir, DoDFileName),
		filepath.Join(laneDir, ReceiptFileName),
		"\"run_id\": \"" + start.RunID + "\"",
		"Given x, when y, then z.",          // the intent, carried
		"A written design record.",          // the spec, carried
		"- Commit on a branch; never push.", // the conventions section, carried
		"- adr-27 — The autonomous run is a seam (`" + adrRel + "`)",
		"- adr-404 — not found at the lane's base",
		"- 2026-09-20 — The loop is a state file (itd-10),\n  continued on a second line.",
		"- 2026-09-22 — The spec spc-1 takes steps.",
	} {
		if !strings.Contains(brief, want) {
			t.Fatalf("the brief must carry %q:\n%s", want, brief)
		}
	}
	for _, absent := range []string{"Repo facts outside the section.", "itd-100 only", "Header prose"} {
		if strings.Contains(brief, absent) {
			t.Fatalf("the brief must not carry %q", absent)
		}
	}
	if !strings.Contains(st.Record[len(st.Record)-1].Note, "rendered "+wantRel) {
		t.Fatalf("the record names the brief: %+v", st.Record[len(st.Record)-1])
	}
}

// TestTheBriefCarriesAnUnmarkedAgentsFileWhole: a repository whose AGENTS.md
// marks no working-conventions section is briefed with the whole file, and the
// brief says so.
func TestTheBriefCarriesAnUnmarkedAgentsFileWhole(t *testing.T) {
	repo := briefRepo(t, "# AGENTS.md\n\n## Definition of done\n\n- make check is clean.\n")
	start, err := Start(repo.Root(), "itd-10", Options{})
	if err != nil {
		t.Fatal(err)
	}
	advanceTo(t, repo, start.RunID, StepImplement)
	st, _ := ReadState(repo.Root(), start.RunID)
	raw, err := os.ReadFile(filepath.Join(repo.Root(), filepath.FromSlash(st.Lanes[0].Brief)))
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"the whole file: it marks no working-conventions section", "- make check is clean."} {
		if !strings.Contains(string(raw), want) {
			t.Fatalf("want %q in the brief", want)
		}
	}
}

// TestTheBriefIsRenderedFromTheLaneBase: the lane is built off the default
// branch, so the brief reads the record there. An intent planned only on the
// branch the checkout has checked out, and a base with no AGENTS.md, are each
// refused rather than briefed from elsewhere, and the lane stays at its brief.
func TestTheBriefIsRenderedFromTheLaneBase(t *testing.T) {
	t.Run("no AGENTS.md at the base", func(t *testing.T) {
		repo := briefRepo(t, "")
		start, err := Start(repo.Root(), "itd-10", Options{})
		if err != nil {
			t.Fatal(err)
		}
		advanceTo(t, repo, start.RunID, StepBrief)
		_, err = Advance(repo.Root(), start.RunID, DefaultSteps(), Options{})
		if r := mustRefusal(t, err); r.Step != string(StepBrief) || !strings.Contains(r.Reason, "AGENTS.md") {
			t.Fatalf("want the missing conventions named: %+v", r)
		}
		if st, _ := ReadState(repo.Root(), start.RunID); st.Lanes[0].Step != StepBrief || st.Lanes[0].Brief != "" {
			t.Fatalf("the lane stays at its brief: %+v", st.Lanes[0])
		}
	})
	t.Run("the intent planned only off the default branch", func(t *testing.T) {
		repo := briefRepo(t, agentsMarked)
		repo.Git("checkout", "-q", "-b", "planning")
		repo.Git("checkout", "-q", "main")
		repo.Remove(plannedRel)
		repo.Write(".abcd/development/intents/drafts/itd-10-alpha.md", readyIntent("", settledQuestions))
		repo.Commit("main still drafts it")
		repo.Git("checkout", "-q", "planning")
		start, err := Start(repo.Root(), "itd-10", Options{})
		if err != nil {
			t.Fatal(err)
		}
		advanceTo(t, repo, start.RunID, StepBrief)
		_, err = Advance(repo.Root(), start.RunID, DefaultSteps(), Options{})
		if r := mustRefusal(t, err); r.Step != string(StepBrief) || !strings.Contains(r.Reason, "not planned") || !strings.Contains(r.Reason, "drafts/") {
			t.Fatalf("want the base's bucket named: %+v", r)
		}
	})
}

// TestDecisionsNamingMatchesWholeIDs: an entry naming a longer id that merely
// starts with the key's digits is not the key's.
func TestDecisionsNamingMatchesWholeIDs(t *testing.T) {
	got := decisionsNaming(decisionsLog, "itd-10", "")
	if len(got) != 1 || !strings.Contains(got[0], "a state file") {
		t.Fatalf("decisionsNaming = %q", got)
	}
}
