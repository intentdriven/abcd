package loop

// issuebrief.go is the brief of an issue-keyed lane (decision 10 on the
// parent, for itd-82 scope 4): the brief renderer takes the issue's record and
// its remedy in place of the intent and the spec. The remedy is the work; the
// definition of done is the repository's, with a detector watched to fail
// before the fix and pass after; the landing resolves the issue through the
// receipt's `resolves`. The record is read from the lane's base, as an
// intent's is, so an edit in the lane's worktree never reaches the brief.

import (
	"bytes"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/intentdriven/abcd/internal/adapter/scanner"
	"github.com/intentdriven/abcd/internal/core/issuerecord"
	"github.com/intentdriven/abcd/internal/core/issueschema"
	"github.com/intentdriven/abcd/internal/core/recordid"
	"github.com/intentdriven/abcd/internal/gitutil"
)

// issueBriefSources is what an issue brief is rendered from.
type issueBriefSources struct {
	issuePath, issueText string
	remedy               string
	// conventions is the conventions text, and conventionsFrom says which part
	// of AGENTS.md it is.
	conventions, conventionsFrom string
	decisions                    []string
}

// readIssueBriefSources reads the issue's record, open at the lane's base, and
// the conventions, out of the lane's base commit. An issue the base does not
// hold open, a record the reader refuses, or one without a remedy is refused:
// the remedy is the work, and a lane is built off the default branch.
func readIssueBriefSources(repoRoot string, st State, lane *Lane) (issueBriefSources, error) {
	var src issueBriefSources
	if !gitutil.IsFullSHA(lane.BaseSHA) {
		return src, refuse(string(StageBrief), "", lane.ID, fmt.Sprintf("the lane records no base commit to read the record at (%s)", quoteOrNone(lane.BaseSHA)),
			"the worktree stage records it; restore the run's state file")
	}
	base := "the lane's base (" + lane.Branch + " at " + shortSHA(lane.BaseSHA) + ")"
	at := baseTree{root: repoRoot, sha: lane.BaseSHA}
	id := st.Issue()
	found, err := at.record(recordid.IssuesRelDir, "iss", id)
	if err != nil {
		return src, fmt.Errorf("reading the ledger at %s: %w", base, err)
	}
	if len(found) != 1 || found[0].folder != "open" {
		where := "does not carry it"
		if len(found) > 0 {
			where = "holds it in " + folders(found)
		}
		return src, refuse(string(StageBrief), "", lane.ID, fmt.Sprintf("%s is not open at %s: the default branch %s", id, base, where),
			"land the issue on the default branch first; a lane is built off the default branch")
	}
	read := func(e baseEntry, limit int64) ([]byte, error) {
		b, err := at.blob(e, limit)
		if err != nil {
			return nil, refuse(string(StageBrief), "", lane.ID, fmt.Sprintf("%s cannot be read at %s: %v", e.path, base, err),
				"the brief reads regular files within their caps; restore "+e.path+" on the default branch")
		}
		return b, nil
	}
	text, err := read(found[0], issueschema.RecordReadLimit)
	if err != nil {
		return src, err
	}
	fm, _, err := issuerecord.Parse(string(text))
	if err != nil {
		return src, refuse(string(StageBrief), "", lane.ID, fmt.Sprintf("%s cannot be parsed at %s: %v", found[0].path, base, err),
			"repair the record on the default branch")
	}
	remedy := strings.TrimSpace(issueschema.RemedyOf(fm))
	if remedy == "" || issueschema.IsMachineRemedy(remedy) {
		return src, refuse(string(StageBrief), "", lane.ID, fmt.Sprintf("%s carries no remedy a person wrote at %s, and the remedy is the lane's work", id, base),
			"write the fix it proposes with `abcd capture remedy "+id+" \"<fix>\"` and land it on the default branch")
	}
	agentsEntry, ok, err := at.file(ConventionsFile)
	if err != nil {
		return src, fmt.Errorf("reading %s at %s: %w", ConventionsFile, base, err)
	}
	if !ok {
		return src, refuse(string(StageBrief), "", lane.ID, fmt.Sprintf("%s holds no %s, so the lane has no conventions to be briefed with", base, ConventionsFile),
			conventionsRemedy)
	}
	agents, err := read(agentsEntry, maxRecordBytes)
	if err != nil {
		return src, err
	}
	src.issuePath, src.issueText, src.remedy = found[0].path, string(text), remedy
	src.conventions, src.conventionsFrom = conventionsSection(string(agents))
	if logEntry, ok, err := at.file(DecisionsLogRel); err != nil {
		return src, fmt.Errorf("reading %s at %s: %w", DecisionsLogRel, base, err)
	} else if ok {
		log, err := read(logEntry, maxDecisionsBytes)
		if err != nil {
			return src, err
		}
		src.decisions = decisionsNaming(string(log), id)
	}
	return src, nil
}

// renderIssueBrief writes an issue lane's brief. laneDir is the lane's
// directory as the implementer, working in another checkout, must address it.
func renderIssueBrief(st State, lane Lane, laneDir string, src issueBriefSources) []byte {
	var b bytes.Buffer
	p := func(format string, a ...any) { fmt.Fprintf(&b, format, a...) }
	at := func(name string) string { return filepath.Join(laneDir, name) }
	id := st.Issue()

	p("# Lane brief: %s of %s\n\n", lane.ID, st.RunID)
	p("You are the implementer of %s in the run `abcd build %s` started: you fix one issue. You did not\n", lane.ID, id)
	p("write this brief and no one will answer a question about it: what the record below does not settle,\n")
	p("you settle and say so in your report, or you hand the issue back (below).\n\n")
	p("Rendered from, at the lane's base (%s, %s):\n\n", lane.Branch, lane.BaseSHA)
	p("- the issue: %s, `%s`\n", id, src.issuePath)
	p("- the conventions: `%s`, %s\n", ConventionsFile, src.conventionsFrom)
	p("- %d entr%s of `%s` naming %s\n\n", len(src.decisions), plural(len(src.decisions), "y", "ies"), DecisionsLogRel, id)

	p("## Your lane\n\n")
	p("- Fix %s by its remedy, below, and nothing else.\n", id)
	p("- Work only in the worktree `%s`.\n", lane.Worktree)
	p("- Commit on its branch, `%s`, cut from the default branch at `%s`. Never push, never switch\n", lane.Branch, lane.BaseSHA)
	p("  branch, and never commit anywhere else.\n")
	p("- Do not resolve %s yourself: the landing runs `capture resolve` with the commit your receipt names.\n\n", id)

	p("## The work: %s's remedy\n\n", id)
	p("%s", fenceQuoteNote)
	p("<!-- begin remedy -->\n\n%s\n\n<!-- end remedy -->\n\n", fenceQuote(src.remedy))

	p("## The definition of done\n\n")
	p("Reproduce, then fix: write a detector (a test) that fails before the fix and passes after, and\n")
	p("watch it fail before you change the code. Then run the repository's definition of done, as the\n")
	p("conventions below state it, in your worktree. The validators judge the fix with the detector as\n")
	p("evidence; they, not the detector, are the oracle.\n\n")

	p("## Handing the issue back\n\n")
	p("An issue reaches you because its fields say it needs no decision. If you find one in it, stop:\n")
	p("do not decide it. Write the receipt below with a `handback` in place of `resolves`, and the loop\n")
	p("discards the lane's work and hands the issue back to a person by its kind:\n\n")
	for _, k := range laneHandBackKinds {
		p("- `%s`: %s\n", k.kind, k.means)
	}
	p("\n`reason` says what you found, in a sentence; `home` names where the decision belongs (an intent,\n")
	p("a decision record, a principle, the brief) and is required for %s.\n\n", strings.Join(homeKinds(), " and "))

	p("## What you hand back\n\n")
	p("Write these three files, then stop:\n\n")
	p("1. Your report, `%s`: what you fixed, the detector you watched fail, and what a reviewer should\n", at(ReportFileName))
	p("   look at. It carries no verdict: only the loop records a verdict, from the validators it runs after you.\n")
	p("2. The definition of done's output, `%s`: its whole output.\n", at(DoDFileName))
	p("3. Your receipt, `%s`, in exactly this shape (strict JSON: any other field refuses it):\n\n", at(ReceiptFileName))
	p("```json\n")
	p("{\n")
	p("  \"schema_version\": %d,\n", ReceiptSchemaVersion)
	p("  \"run_id\": %q,\n", st.RunID)
	p("  \"lane\": %q,\n", lane.ID)
	p("  \"branch\": %q,\n", lane.Branch)
	p("  \"commits\": [\"<the full object name of every commit you made on the branch>\"],\n")
	p("  \"definition_of_done\": {\"command\": \"<what you ran>\", \"exit_code\": 0, \"output\": %q},\n", DoDFileName)
	p("  \"report\": %q,\n", ReportFileName)
	p("  \"model\": \"<the model id your harness reports, or leave the field out>\",\n")
	p("  \"resolves\": [{\"issue\": %q, \"commit\": \"<the commit of yours that fixed it>\", \"note\": \"<the resolution>\",\n", id)
	p("                \"impact\": \"additive|breaking|fix|internal\", \"grounds\": \"pursued: <what is expected, and what would show it wrong>\"}]\n")
	p("}\n")
	p("```\n\n")
	p("`resolves` must name %s: a receipt that does not is refused. To hand the issue back instead,\n", id)
	p("leave `resolves` out and add `\"handback\": {\"kind\": \"<kind>\", \"reason\": \"<what you found>\", \"home\": \"<where it belongs>\"}`.\n\n")
	p("`output` and `report` are paths inside `%s`. The loop verifies the receipt before anything else\n", laneDir)
	p("runs, and a receipt short of what it must carry is refused, naming what is missing.\n\n")

	p("## Outward-facing text\n\n")
	p("A pull-request body, an issue, a comment, a commit message and a release note are public the moment\n")
	p("they exist. This holds whatever the conventions below say:\n\n")
	p("> %s\n\n", scanner.OutboundPolicy)

	p("%s", lostConnectionRule)

	p("---\n\n## The issue: %s\n\n<!-- begin %s -->\n\n%s\n\n<!-- end %s -->\n\n", id, src.issuePath, fenceQuote(strings.TrimSpace(src.issueText)), src.issuePath)
	p("## The conventions: %s\n\n<!-- begin %s -->\n\n%s\n\n<!-- end %s -->\n\n", ConventionsFile, ConventionsFile, fenceQuote(src.conventions), ConventionsFile)
	p("### Entries of `%s` naming %s\n\n", DecisionsLogRel, id)
	if len(src.decisions) == 0 {
		p("None.\n")
	}
	for _, d := range src.decisions {
		p("%s\n", d)
	}
	return b.Bytes()
}
