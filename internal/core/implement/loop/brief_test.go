package loop

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/adapter/scanner"
	"github.com/intentdriven/abcd/internal/core/decisions"
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

// The decision records briefRepo commits in the folder: two that name the
// intent or its spec, one naming only a longer id, and one whose text names
// the intent but which its own id does not date, so the reader refuses it.
var briefDecisionRecords = map[string]string{
	"dcn-2609230000000001-the-run-holds-one-lane.md": decisionRecord("dcn-2609230000000001", "the-run-holds-one-lane", "2026-09-23",
		"The run holds one lane per step (itd-10)."),
	"dcn-2609240000000001-spec-steps-land-in-order.md": decisionRecord("dcn-2609240000000001", "spec-steps-land-in-order", "2026-09-24",
		"spc-1's steps land in order,\nas the second line says."),
	"dcn-2609250000000001-a-longer-id.md": decisionRecord("dcn-2609250000000001", "a-longer-id", "2026-09-25",
		"A record for itd-100 only."),
	"dcn-2609260000000001-misdated.md": decisionRecord("dcn-2609260000000001", "misdated", "2026-09-01",
		"A MISDATED RECORD naming itd-10."),
}

// decisionRecord is a decision record as the folder's writer lays one out.
func decisionRecord(id, slug, date, text string) string {
	return "---\nid: " + id + "\nslug: " + slug + "\ndate: " + date + "\n---\n" + date + " — " + text + "\n"
}

// briefRepo is loopRepo with the record a brief is rendered from committed on
// the default branch: the conventions, one cited ADR, the decision records in
// the folder and a frozen decision log.
func briefRepo(t *testing.T, agents string) *gittest.Repo {
	t.Helper()
	repo := loopRepo(t, readyIntent("", settledQuestions+"Refines adr-27, and adr-404 which does not exist.\n\n"), specWithSteps(""))
	if agents != "" {
		repo.Write("AGENTS.md", agents)
	}
	repo.Write(adrRel, adrBody)
	repo.Write(decisions.FrozenLog, decisionsLog)
	for name, body := range briefDecisionRecords {
		repo.Write(decisions.RelDir+"/"+name, body)
	}
	repo.Commit("the record")
	return repo
}

// advanceTo steps the run until its current lane's next step is want.
func advanceTo(t *testing.T, repo *gittest.Repo, runID string, want Stage) StepResult {
	t.Helper()
	var res StepResult
	for range len(Sequence) {
		st, err := ReadState(repo.Root(), runID)
		if err != nil {
			t.Fatal(err)
		}
		if i := st.current(); i >= 0 && st.Lanes[i].Stage == want {
			return res
		}
		if res, err = advance(repo.Root(), runID, DefaultStages(), Options{}); err != nil {
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
	advanceTo(t, repo, start.RunID, StageBrief)
	res, err := advance(repo.Root(), start.RunID, DefaultStages(), Options{})
	if err != nil {
		t.Fatal(err)
	}
	if res.PerformedStage != StageBrief || res.Stage != StageImplement {
		t.Fatalf("want the brief stage performed: %+v", res)
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

// firstLaneBrief starts a run on key and returns the brief its first lane is
// handed.
func firstLaneBrief(t *testing.T, repo *gittest.Repo, key string) string {
	t.Helper()
	start, err := Start(repo.Root(), key, Options{})
	if err != nil {
		t.Fatal(err)
	}
	advanceTo(t, repo, start.RunID, StageImplement)
	return laneBrief(t, repo, start.RunID, 0)
}

// TestBriefQuotesDecisionsNamingTheIntent is spc-2610110931381316 step 4: the
// brief quotes, in id order, every decision record in the folder whose body
// names the intent or its spec, names a record the reader refused rather than
// dropping it, and then quotes the entries of a frozen log naming either; a
// base with no frozen log has no frozen section.
func TestBriefQuotesDecisionsNamingTheIntent(t *testing.T) {
	brief := firstLaneBrief(t, briefRepo(t, agentsMarked), "itd-10")
	folder := "### Decision records in `" + decisions.RelDir + "/` naming itd-10 or spc-1\n\n"
	first := "- dcn-2609230000000001: 2026-09-23 — The run holds one lane per step (itd-10).\n"
	second := "- dcn-2609240000000001: 2026-09-24 — spc-1's steps land in order,\n  as the second line says.\n"
	frozen := "### Entries of the frozen log `" + decisions.FrozenLog + "` naming itd-10 or spc-1\n\n"
	for _, want := range []string{
		folder + first + second,
		"`" + decisions.RelDir + "/dcn-2609260000000001-misdated.md` names it but is not read as a record:",
		frozen + "- 2026-09-20 — The loop is a state file (itd-10),\n  continued on a second line.\n- 2026-09-22 — The spec spc-1 takes steps.\n",
		"2 decision record(s) in `" + decisions.RelDir + "/` and 2 entries of the frozen log `" + decisions.FrozenLog + "` naming itd-10 or spc-1",
	} {
		if !strings.Contains(brief, want) {
			t.Fatalf("the brief must carry %q:\n%s", want, brief)
		}
	}
	if strings.Index(brief, folder) > strings.Index(brief, frozen) {
		t.Fatalf("the folder's records come before the frozen log's entries:\n%s", brief)
	}
	for _, absent := range []string{"A record for itd-100 only", "A MISDATED RECORD", "itd-100 only"} {
		if strings.Contains(brief, absent) {
			t.Fatalf("the brief must not quote %q:\n%s", absent, brief)
		}
	}

	repo := briefRepo(t, agentsMarked)
	repo.Remove(decisions.FrozenLog)
	repo.Commit("no frozen log")
	brief = firstLaneBrief(t, repo, "itd-10")
	if !strings.Contains(brief, folder+first+second) || strings.Contains(brief, "frozen log") {
		t.Fatalf("a base without a frozen log quotes the folder alone:\n%s", brief)
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
	advanceTo(t, repo, start.RunID, StageImplement)
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

// TestTheBriefCarriesTheOutboundPolicy is itd-152's fifth criterion: the prompt
// an autonomous run hands its implementer carries the policy that bans a live
// session URL and a tool's attribution footer in public text and mandates the
// re-read-and-strip of every pull request, issue and comment it creates. The
// policy is quoted from scanner.OutboundPolicy, the one value the scanner, the
// lint rules and the gates already quote, so the text is compared to that value
// rather than to a copy of it. A managed repository's AGENTS.md need not say a
// word about it (this one says nothing), and the policy is the brief's own
// instruction, outside every block it carries from the record.
func TestTheBriefCarriesTheOutboundPolicy(t *testing.T) {
	repo := briefRepo(t, agentsMarked)
	start, err := Start(repo.Root(), "itd-10", Options{})
	if err != nil {
		t.Fatal(err)
	}
	advanceTo(t, repo, start.RunID, StageImplement)
	st, err := ReadState(repo.Root(), start.RunID)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(filepath.Join(repo.Root(), filepath.FromSlash(st.Lanes[0].Brief)))
	if err != nil {
		t.Fatal(err)
	}
	brief := string(raw)
	if n := strings.Count(brief, scanner.OutboundPolicy); n != 1 {
		t.Fatalf("the brief must quote the outbound policy once, verbatim; it does %d time(s):\n%s", n, brief)
	}
	at := strings.Index(brief, scanner.OutboundPolicy)
	if record := strings.Index(brief, "<!-- begin "); record < 0 || at > record {
		t.Fatalf("the policy is the brief's own instruction, before the record it carries (policy at %d, record at %d)", at, record)
	}
	heading := strings.Index(brief, "## Outward-facing text\n")
	if heading < 0 || heading > at {
		t.Fatalf("the policy sits under its own heading:\n%s", brief)
	}
}

// TestTheBriefTellsTheImplementerToYieldOnALostConnection is the agent's half
// of a lost connection (iss-2610080620372731, ruling Q1 of 2026-10-09): an
// implementer whose network or model service drops mid-task neither retries
// nor commits what it has; it stops and ends its hand-back with exactly one
// `NETWORK: <cmd>` line and no receipt, the line `implement step --restart
// --yielded` takes. A usage or rate limit is not a lost connection. The rule is
// the brief's own instruction, before the record it carries.
func TestTheBriefTellsTheImplementerToYieldOnALostConnection(t *testing.T) {
	repo := briefRepo(t, agentsMarked)
	start, err := Start(repo.Root(), "itd-10", Options{})
	if err != nil {
		t.Fatal(err)
	}
	advanceTo(t, repo, start.RunID, StageImplement)
	st, err := ReadState(repo.Root(), start.RunID)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(filepath.Join(repo.Root(), filepath.FromSlash(st.Lanes[0].Brief)))
	if err != nil {
		t.Fatal(err)
	}
	brief := string(raw)
	if n := strings.Count(brief, lostConnectionRule); n != 1 {
		t.Fatalf("the brief must carry the lost-connection rule once; it does %d time(s):\n%s", n, brief)
	}
	for _, want := range []string{
		"## A lost connection\n",
		"do not retry",
		"do not commit",
		"`" + NetworkLinePrefix + " <the failing command>`",
		"no receipt",
		"usage or rate limit",
	} {
		if !strings.Contains(lostConnectionRule, want) {
			t.Errorf("the lost-connection rule says %q:\n%s", want, lostConnectionRule)
		}
	}
	if at, record := strings.Index(brief, lostConnectionRule), strings.Index(brief, "<!-- begin "); record < 0 || at > record {
		t.Fatalf("the rule is the brief's own instruction, before the record it carries (rule at %d, record at %d)", at, record)
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
		advanceTo(t, repo, start.RunID, StageBrief)
		_, err = advance(repo.Root(), start.RunID, DefaultStages(), Options{})
		if r := mustRefusal(t, err); r.Stage != string(StageBrief) || !strings.Contains(r.Reason, "AGENTS.md") {
			t.Fatalf("want the missing conventions named: %+v", r)
		}
		if st, _ := ReadState(repo.Root(), start.RunID); st.Lanes[0].Stage != StageBrief || st.Lanes[0].Brief != "" {
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
		advanceTo(t, repo, start.RunID, StageBrief)
		_, err = advance(repo.Root(), start.RunID, DefaultStages(), Options{})
		if r := mustRefusal(t, err); r.Stage != string(StageBrief) || !strings.Contains(r.Reason, "not planned") || !strings.Contains(r.Reason, "drafts/") {
			t.Fatalf("want the base's bucket named: %+v", r)
		}
	})
}

// TestABriefRenderedAgainRendersTheBaseNotTheWorktree: the brief says it was
// rendered at the lane's base, so it is read from the base commit's objects,
// not the lane worktree's files. A brief rendered again after the implementer
// edited, committed, moved and deleted the record in its worktree carries the
// base's text, byte for byte the first rendering.
func TestABriefRenderedAgainRendersTheBaseNotTheWorktree(t *testing.T) {
	repo := briefRepo(t, agentsMarked)
	start, err := Start(repo.Root(), "itd-10", Options{})
	if err != nil {
		t.Fatal(err)
	}
	advanceTo(t, repo, start.RunID, StageImplement)
	st, err := ReadState(repo.Root(), start.RunID)
	if err != nil {
		t.Fatal(err)
	}
	lane := st.Lanes[0]
	briefPath := filepath.Join(repo.Root(), filepath.FromSlash(lane.Brief))
	first, err := os.ReadFile(briefPath)
	if err != nil {
		t.Fatal(err)
	}

	wt := lane.Worktree
	write := func(rel, body string) {
		t.Helper()
		if err := os.WriteFile(filepath.Join(wt, filepath.FromSlash(rel)), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	git := func(args ...string) {
		t.Helper()
		repo.Git(append([]string{"-C", wt}, args...)...)
	}
	// Committed on the lane branch: a rewritten spec and ADR.
	write(specRel, specWithSteps("")+"\nIMPLEMENTER'S SPEC EDIT\n")
	write(adrRel, "---\nid: adr-27\nstatus: accepted\n---\n# IMPLEMENTER'S ADR TITLE\n")
	git("add", "-A")
	git("commit", "-q", "-m", "the implementer's commit")
	// Uncommitted: the intent moved on, AGENTS.md gone, the log rewritten.
	if err := os.MkdirAll(filepath.Join(wt, ".abcd", "development", "intents", "shipped"), 0o755); err != nil {
		t.Fatal(err)
	}
	git("mv", plannedRel, ".abcd/development/intents/shipped/itd-10-alpha.md")
	if err := os.Remove(filepath.Join(wt, "AGENTS.md")); err != nil {
		t.Fatal(err)
	}
	write(decisions.FrozenLog, "- 2026-09-27 — IMPLEMENTER'S DECISION on itd-10.\n")
	write(decisions.RelDir+"/dcn-2609270000000001-implementers.md",
		decisionRecord("dcn-2609270000000001", "implementers", "2026-09-27", "IMPLEMENTER'S RECORD on itd-10."))

	c := Context{RepoRoot: repo.Root(), RunDir: runRel(st.RunID), State: st}
	if _, err := briefStage(c, &lane); err != nil {
		t.Fatalf("the brief renders the base whatever the worktree holds: %v", err)
	}
	again, err := os.ReadFile(briefPath)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(again), "IMPLEMENTER'S") {
		t.Fatalf("the brief carries the implementer's tree under the base's label:\n%s", again)
	}
	if string(again) != string(first) {
		t.Fatalf("the brief rendered again is the base's brief:\nfirst:\n%s\nagain:\n%s", first, again)
	}
}

// TestABriefSourceTheBaseHoldsAsASymlinkIsRefused: a record the base commits
// as a link (mode 120000) is not read through, whatever it points at.
func TestABriefSourceTheBaseHoldsAsASymlinkIsRefused(t *testing.T) {
	repo := briefRepo(t, "")
	if err := os.Symlink("/etc/hosts", filepath.Join(repo.Root(), "AGENTS.md")); err != nil {
		t.Fatal(err)
	}
	repo.Git("add", "AGENTS.md")
	repo.Commit("a linked AGENTS.md")
	start, err := Start(repo.Root(), "itd-10", Options{})
	if err != nil {
		t.Fatal(err)
	}
	advanceTo(t, repo, start.RunID, StageBrief)
	_, err = advance(repo.Root(), start.RunID, DefaultStages(), Options{})
	if r := mustRefusal(t, err); r.Stage != string(StageBrief) || !strings.Contains(r.Reason, "AGENTS.md") {
		t.Fatalf("want the linked AGENTS.md refused: %+v", r)
	}
}
