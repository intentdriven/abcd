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
	"slices"
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
	// members are the record and remedy of each member of a run with members
	// (itd-2611201450052326), in the run's order after the lead, each read
	// from the lane's base as the lead's is; empty for a run without members.
	members []memberSource
	// conventions is the conventions text, and conventionsFrom says which part
	// of AGENTS.md it is.
	conventions, conventionsFrom string
	decisions                    namedDecisions
}

// memberSource is one member's record and remedy, as the brief quotes them.
type memberSource struct {
	id, path, text, remedy string
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
	read := func(e baseEntry, limit int64) ([]byte, error) {
		b, err := at.blob(e, limit)
		if err != nil {
			return nil, refuse(string(StageBrief), "", lane.ID, fmt.Sprintf("%s cannot be read at %s: %v", e.path, base, err),
				"the brief reads regular files within their caps; restore "+e.path+" on the default branch")
		}
		return b, nil
	}
	id := st.Issue()
	lead, err := readIssueAtBase(at, base, lane, id, read)
	if err != nil {
		return src, err
	}
	// A run with members quotes each issue it still holds (the brief is
	// rendered before any is handed back), each read the same way.
	ids := []string{id}
	for _, m := range st.Issues() {
		if m == id || !st.hasMembers() {
			continue
		}
		ms, err := readIssueAtBase(at, base, lane, m, read)
		if err != nil {
			return src, err
		}
		src.members = append(src.members, ms)
		ids = append(ids, m)
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
	src.issuePath, src.issueText, src.remedy = lead.path, lead.text, lead.remedy
	src.conventions, src.conventionsFrom = conventionsSection(string(agents))
	if src.decisions, err = at.decisionsNaming(base, read, ids...); err != nil {
		return src, err
	}
	return src, nil
}

// readIssueAtBase reads one issue's record, open at the lane's base, and its
// remedy: an issue the base does not hold open, a record the reader refuses,
// or one without a remedy a person wrote is refused, naming the issue.
func readIssueAtBase(at baseTree, base string, lane *Lane, id string, read func(baseEntry, int64) ([]byte, error)) (memberSource, error) {
	found, err := at.record(recordid.IssuesRelDir, "iss", id)
	if err != nil {
		return memberSource{}, fmt.Errorf("reading the ledger at %s: %w", base, err)
	}
	if len(found) != 1 || found[0].folder != "open" {
		where := "does not carry it"
		if len(found) > 0 {
			where = "holds it in " + folders(found)
		}
		r := refuse(string(StageBrief), "", lane.ID, fmt.Sprintf("%s is not open at %s: the default branch %s", id, base, where),
			"land the issue on the default branch first; a lane is built off the default branch")
		// An issue finished at the base, resolved or wontfix there, is
		// finished for good: the lane's base never moves, so a drain hands
		// the lane back at once (stuck.go).
		if slices.ContainsFunc(found, func(e baseEntry) bool { return e.folder == "resolved" || e.folder == "wontfix" }) {
			r.final = id + " is finished at the lane's base"
		}
		return memberSource{}, r
	}
	text, err := read(found[0], issueschema.RecordReadLimit)
	if err != nil {
		return memberSource{}, err
	}
	fm, _, err := issuerecord.Parse(string(text))
	if err != nil {
		return memberSource{}, refuse(string(StageBrief), "", lane.ID, fmt.Sprintf("%s cannot be parsed at %s: %v", found[0].path, base, err),
			"repair the record on the default branch")
	}
	remedy := strings.TrimSpace(issueschema.RemedyOf(fm))
	if remedy == "" || issueschema.IsMachineRemedy(remedy) {
		return memberSource{}, refuse(string(StageBrief), "", lane.ID, fmt.Sprintf("%s carries no remedy a person wrote at %s, and the remedy is the lane's work", id, base),
			"write the fix it proposes with `abcd capture remedy "+id+" \"<fix>\"` and land it on the default branch")
	}
	return memberSource{id: id, path: found[0].path, text: string(text), remedy: remedy}, nil
}

// renderIssueBrief writes an issue lane's brief. laneDir is the lane's
// directory as the implementer, working in another checkout, must address it.
func renderIssueBrief(st State, lane Lane, laneDir string, src issueBriefSources) []byte {
	if len(src.members) > 0 {
		return renderBundleBrief(st, lane, laneDir, src)
	}
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
	p("- %s\n\n", src.decisions.summary(id))

	p("## Your lane\n\n")
	p("- Fix %s by its remedy, below, and nothing else.\n", id)
	p("- Work only in the worktree `%s`.\n", lane.Worktree)
	p("- Commit on its branch, `%s`, cut from the default branch at `%s`. Never push, never switch\n", lane.Branch, lane.BaseSHA)
	p("  branch, and never commit anywhere else.\n")
	p("- Do not resolve %s yourself: the landing runs `capture resolve` with the commit your receipt names.\n\n", id)

	p("## Files beyond the issue's place\n\n")
	p("The run reserves %s on the shared remote with the path its found_at names, so no other copy\n", id)
	p("edits it while you work. You may change any other file the fix needs, and you run nothing to\n")
	p("reserve it: once you hand your receipt back, the loop reserves every path your commits change\n")
	p("beyond that hold, before it judges the receipt and before the validators start, and again before\n")
	p("each later round and before the landing's push. The decision records you write and the issue's\n")
	p("own record are reserved the same way.\n\n")
	p("- A path another copy holds: the loop does not take it from them. Your receipt is taken all the\n")
	p("  same, the run's record names the path and its holder, and the lane waits before its validators.\n")
	p("  If the hold has not ended by the drain's bound of minutes, the lane is handed back, naming the\n")
	p("  path and its holder.\n")
	p("- A path no reservation may carry: one whose name has a backslash, a control, non-printing,\n")
	p("  bidirectional or zero-width character, or a leading or trailing space, or one with `..` or a\n")
	p("  leading `/`. Your receipt is refused, naming it. Take the change out of the branch.\n")
	p("- A gate file (%s): an unattended drive does not land a lane that changes one, and hands it to\n", gateFilesText())
	p("  a person's own session. Change one only when the remedy names it.\n\n")

	p("## The work: %s's remedy\n\n", id)
	p("%s", fenceQuoteNote)
	p("<!-- begin remedy -->\n\n%s\n\n<!-- end remedy -->\n\n", fenceQuote(src.remedy))

	p("## The definition of done\n\n")
	p("Reproduce, then fix: write a detector (a test) that fails before the fix and passes after, and\n")
	p("watch it fail before you change the code. Then run the repository's definition of done, as the\n")
	p("conventions below state it, in your worktree. The validators judge the fix with the detector as\n")
	p("evidence; they, not the detector, are the oracle.\n\n")
	p("%s\n", narrowTestText(""))

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
	src.decisions.render(&b, id)
	return b.Bytes()
}

// renderBundleBrief writes the brief of a run with members
// (itd-2611201450052326, spc-2611201455051897 step 2): every issue's record
// and remedy, read from the lane's base, the lead first and each member after
// it in the run's order; one commit asked for each issue, named in its
// resolution; and the receipt naming every issue the run still holds exactly
// once, in `resolves` or in `handbacks`, in this receipt and in every fix
// round's. A run without members is rendered by renderIssueBrief as before.
func renderBundleBrief(st State, lane Lane, laneDir string, src issueBriefSources) []byte {
	var b bytes.Buffer
	p := func(format string, a ...any) { fmt.Fprintf(&b, format, a...) }
	at := func(name string) string { return filepath.Join(laneDir, name) }
	id := st.Issue()
	issues := append([]memberSource{{id: id, path: src.issuePath, text: src.issueText, remedy: src.remedy}}, src.members...)
	ids := make([]string, len(issues))
	for i, is := range issues {
		ids[i] = is.id
	}
	all := andList(ids)
	members := andList(ids[1:])

	p("# Lane brief: %s of %s\n\n", lane.ID, st.RunID)
	p("You are the implementer of %s in the run `abcd build %s` started: you fix %d issues of one kind\n", lane.ID, id, len(issues))
	p("in one place, %s and the members it carries, %s. You did not write this brief and no one\n", id, members)
	p("will answer a question about it: what the records below do not settle, you settle and say so in\n")
	p("your report, or you hand that issue back (below).\n\n")
	p("Rendered from, at the lane's base (%s, %s):\n\n", lane.Branch, lane.BaseSHA)
	for i, is := range issues {
		role := "a member"
		if i == 0 {
			role = "the lead"
		}
		p("- %s: %s, `%s`\n", role, is.id, is.path)
	}
	p("- the conventions: `%s`, %s\n", ConventionsFile, src.conventionsFrom)
	p("- %s\n\n", src.decisions.summary(all))

	p("## Your lane\n\n")
	p("- Fix each of %s by its own remedy, below, and nothing else.\n", all)
	p("- Make one commit for each issue, and name that commit in the issue's resolution.\n")
	p("- Work only in the worktree `%s`.\n", lane.Worktree)
	p("- Commit on its branch, `%s`, cut from the default branch at `%s`. Never push, never switch\n", lane.Branch, lane.BaseSHA)
	p("  branch, and never commit anywhere else.\n")
	p("- Do not resolve any of them yourself: the landing runs `capture resolve` for each with the commit\n")
	p("  your receipt names.\n\n")

	p("## Files beyond the issues' places\n\n")
	p("The run reserves each issue on the shared remote with the path its found_at names, so no other\n")
	p("copy edits them while you work. You may change any other file the fixes need, and you run nothing\n")
	p("to reserve it: once you hand your receipt back, the loop reserves every path your commits change\n")
	p("beyond those holds, before it judges the receipt and before the validators start, and again before\n")
	p("each later round and before the landing's push. The decision records you write and the issues'\n")
	p("own records are reserved the same way.\n\n")
	p("- A path another copy holds: the loop does not take it from them. Your receipt is taken all the\n")
	p("  same, the run's record names the path and its holder, and the lane waits before its validators.\n")
	p("  If the hold has not ended by the drain's bound of minutes, the lane is handed back, naming the\n")
	p("  path and its holder.\n")
	p("- A path no reservation may carry: one whose name has a backslash, a control, non-printing,\n")
	p("  bidirectional or zero-width character, or a leading or trailing space, or one with `..` or a\n")
	p("  leading `/`. Your receipt is refused, naming it. Take the change out of the branch.\n")
	p("- A gate file (%s): an unattended drive does not land a lane that changes one, and hands it to\n", gateFilesText())
	p("  a person's own session. Change one only when an issue's remedy names it.\n\n")

	p("## The work: each issue's remedy\n\n")
	p("%s", fenceQuoteNote)
	for i, is := range issues {
		role := ""
		if i == 0 {
			role = " (the lead)"
		}
		p("### %s's remedy%s\n\n", is.id, role)
		p("<!-- begin remedy -->\n\n%s\n\n<!-- end remedy -->\n\n", fenceQuote(is.remedy))
	}

	p("## The definition of done\n\n")
	p("Reproduce, then fix: for each issue, write a detector (a test) that fails before its fix and passes\n")
	p("after, and watch it fail before you change the code. Then run the repository's definition of done,\n")
	p("as the conventions below state it, in your worktree. The validators judge the fixes with the\n")
	p("detectors as evidence; they, not the detectors, are the oracle.\n\n")
	p("%s\n", narrowTestText(""))

	p("## Handing an issue back\n\n")
	p("An issue reaches you because its fields say it needs no decision. If you find one in an issue, stop\n")
	p("work on that issue: do not decide it. Name it in the receipt's `handbacks` in place of `resolves`, and\n")
	p("the loop hands that issue alone back to a person by its kind, while the lane goes on and lands the\n")
	p("others. Take the commits you made for it out of the branch: a receipt that hands back an issue whose\n")
	p("commit an earlier receipt named is refused while that commit is still on the branch. If you hand\n")
	p("back every issue, the loop discards the lane's work. The kinds:\n\n")
	for _, k := range laneHandBackKinds {
		p("- `%s`: %s\n", k.kind, k.means)
	}
	p("\n`reason` says what you found, in a sentence; `home` names where the decision belongs (an intent,\n")
	p("a decision record, a principle, the brief) and is required for %s.\n\n", strings.Join(homeKinds(), " and "))

	p("## What you hand back\n\n")
	p("Write these three files, then stop:\n\n")
	p("1. Your report, `%s`: what you fixed for each issue, the detectors you watched fail, and what a\n", at(ReportFileName))
	p("   reviewer should look at. It carries no verdict: only the loop records a verdict, from the validators it runs after you.\n")
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
	p("  \"resolves\": [\n")
	for i, is := range issues {
		sep := ","
		if i == len(issues)-1 {
			sep = ""
		}
		p("    {\"issue\": %q, \"commit\": \"<the commit of yours that fixed it>\", \"note\": \"<the resolution>\",\n", is.id)
		p("     \"impact\": \"additive|breaking|fix|internal\", \"grounds\": \"pursued: <what is expected, and what would show it wrong>\"}%s\n", sep)
	}
	p("  ],\n")
	p("  \"handbacks\": []\n")
	p("}\n")
	p("```\n\n")
	p("Name every issue of the run, %s, exactly once in every receipt you write, this one and each fix\n", all)
	p("round's: in `resolves`, with the commit that fixes it named afresh, or in `handbacks`. A receipt that\n")
	p("leaves one out, names one in both lists, names an issue outside the run or one already handed back,\n")
	p("or carries a top-level `handback` is refused. To hand an issue back, move it from `resolves` to\n")
	p("`handbacks` as `{\"issue\": \"<id>\", \"kind\": \"<kind>\", \"reason\": \"<what you found>\", \"home\": \"<where it belongs>\"}`.\n\n")
	p("`output` and `report` are paths inside `%s`. The loop verifies the receipt before anything else\n", laneDir)
	p("runs, and a receipt short of what it must carry is refused, naming what is missing.\n\n")

	p("## Outward-facing text\n\n")
	p("A pull-request body, an issue, a comment, a commit message and a release note are public the moment\n")
	p("they exist. This holds whatever the conventions below say:\n\n")
	p("> %s\n\n", scanner.OutboundPolicy)

	p("%s", lostConnectionRule)

	p("---\n\n")
	for i, is := range issues {
		role := "a member"
		if i == 0 {
			role = "the lead"
		}
		p("## The issue: %s, %s\n\n<!-- begin %s -->\n\n%s\n\n<!-- end %s -->\n\n", is.id, role, is.path, fenceQuote(strings.TrimSpace(is.text)), is.path)
	}
	p("## The conventions: %s\n\n<!-- begin %s -->\n\n%s\n\n<!-- end %s -->\n\n", ConventionsFile, ConventionsFile, fenceQuote(src.conventions), ConventionsFile)
	src.decisions.render(&b, all)
	return b.Bytes()
}

// andList joins items as a sentence lists them: "a", "a and b", "a, b and c".
func andList(items []string) string {
	if len(items) < 2 {
		return strings.Join(items, "")
	}
	return strings.Join(items[:len(items)-1], ", ") + " and " + items[len(items)-1]
}

// gateFilesText names every gate file the drive refuses to land a change to
// (DriveGates, and the hooks path the repository configures), for the
// brief's line on them, from the one list the drive reads.
func gateFilesText() string {
	var files, dirs []string
	for _, g := range DriveGates {
		if strings.HasSuffix(g, "/") {
			dirs = append(dirs, "`"+g+"`")
		} else {
			files = append(files, "`"+g+"`")
		}
	}
	return strings.Join(files, ", ") + ", anything under " + orList(dirs) + ", or under the hooks path the repository configures"
}

// orList joins items as a sentence lists them: "a", "a or b", "a, b or c".
func orList(items []string) string {
	if len(items) < 2 {
		return strings.Join(items, "")
	}
	return strings.Join(items[:len(items)-1], ", ") + " or " + items[len(items)-1]
}
