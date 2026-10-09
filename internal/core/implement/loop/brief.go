package loop

// brief.go is the brief renderer (spec piece 5; criterion 3): the one file a
// lane's implementer is handed. It is rendered from the lane's base — the
// worktree the worktree stage made, on the default branch — so the implementer
// reads exactly the record its branch builds on: the intent, the spec, the
// conventions section of AGENTS.md, and the decisions the intent cites. It
// names each source and where it was read, and tells the implementer where its
// report, the definition of done's output and its receipt go and what shape
// the receipt takes, because the receipt's verifier (piece 7) holds it to that.

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"github.com/intentdriven/abcd/internal/adapter/scanner"
	"github.com/intentdriven/abcd/internal/core/decide"
	"github.com/intentdriven/abcd/internal/core/frontmatter"
	"github.com/intentdriven/abcd/internal/core/intent"
	"github.com/intentdriven/abcd/internal/core/recordid"
	"github.com/intentdriven/abcd/internal/core/spec"
	"github.com/intentdriven/abcd/internal/fsutil"
	"github.com/intentdriven/abcd/internal/gitutil"
)

// The files of a lane's directory, .abcd/.work.local/run/<run-id>/<lane-id>/.
// The loop writes the brief; the implementer writes the other three.
const (
	BriefFileName   = "brief.md"
	ReceiptFileName = "receipt.json"
	ReportFileName  = "report.md"
	DoDFileName     = "dod.log"
)

// ConventionsFile is the file a brief's conventions are read from.
const ConventionsFile = "AGENTS.md"

// DecisionsLogRel is the append-only decision log the brief quotes entries of.
const DecisionsLogRel = ".abcd/work/DECISIONS.md"

// The markers the working-conventions section sits between in a managed
// repository's AGENTS.md (commands/prepare-this-repo.md).
const (
	conventionsBegin = "<!-- working-conventions "
	conventionsEnd   = "<!-- /working-conventions -->"
)

// Read caps for the brief's sources.
const (
	maxRecordBytes    = 256 * 1024
	maxDecisionsBytes = 4 << 20
)

// adrCiteRe finds an ADR id cited in an intent.
var adrCiteRe = regexp.MustCompile(`\badr-[0-9]+\b`)

// laneRel is a lane's directory inside its run, relative to the checkout root.
// The lane id is held to the loop's shape before a path is built from it.
func laneRel(runID, laneID string, stage Stage) (string, error) {
	if !ValidRunID(runID) || !ValidLaneID(laneID) {
		return "", refuse(string(stage), "", "", fmt.Sprintf("%q of %q is not a lane the loop opened, so no lane path is built from it", laneID, runID),
			"the loop names its runs and lanes; restore the run's state file")
	}
	return runRel(runID) + "/" + laneID, nil
}

// laneFile is a file of a lane's directory, relative to the checkout root.
func laneFile(runID, laneID string, stage Stage, name string) (string, error) {
	dir, err := laneRel(runID, laneID, stage)
	if err != nil {
		return "", err
	}
	return dir + "/" + name, nil
}

// briefSources is what a brief is rendered from, each read from the lane's base.
type briefSources struct {
	intentPath, intentText string
	specPath, specText     string
	// conventions is the conventions text, and conventionsFrom says which part
	// of AGENTS.md it is.
	conventions, conventionsFrom string
	adrs                         []citedADR
	decisions                    []string
	// steps is the spec's build view at the base (spec.Steps): the steps it
	// lists, or its one implicit step.
	steps []spec.Step
}

// citedADR is one ADR the intent cites, as the lane's base holds it.
type citedADR struct {
	ID, Title, Path string
	Found           bool
}

// briefStage is the brief stage's body: it renders the brief from the lane's
// base into the lane's directory, replacing what an interrupted call wrote.
func briefStage(c Context, lane *Lane) (Outcome, error) {
	lw, err := laneWorktree(c.RepoRoot, c.State.RunID, lane.ID)
	if err != nil {
		return Outcome{}, relabel(err, StageBrief)
	}
	if lane.Worktree == "" || lane.Worktree != lw.Path {
		return Outcome{}, refuse(string(StageBrief), "", lane.ID,
			"the lane has no worktree the loop made (its state names "+quoteOrNone(fsutil.RedactHome(lane.Worktree))+")",
			"the worktree stage makes it; restore the run's state file")
	}
	dirRel, err := laneRel(c.State.RunID, lane.ID, StageBrief)
	if err != nil {
		return Outcome{}, err
	}
	rel := dirRel + "/" + BriefFileName
	laneDir := filepath.Join(c.RepoRoot, filepath.FromSlash(dirRel))
	var body []byte
	var from string
	if c.State.Issue() != "" {
		src, err := readIssueBriefSources(c.RepoRoot, c.State, lane)
		if err != nil {
			return Outcome{}, err
		}
		body = renderIssueBrief(c.State, *lane, laneDir, src)
		from = fmt.Sprintf("%s and %s (%s)", c.State.Issue(), ConventionsFile, src.conventionsFrom)
	} else {
		src, err := readBriefSources(c.RepoRoot, c.State, lane)
		if err != nil {
			return Outcome{}, err
		}
		body = renderBrief(c.State, *lane, laneDir, src)
		from = fmt.Sprintf("%s, %s, %s (%s) and %d cited decision(s)", c.State.Intent, c.State.Spec, ConventionsFile, src.conventionsFrom, len(src.adrs)+len(src.decisions))
	}
	if err := fsutil.EnsureRealDirAll(c.RepoRoot, dirRel, dirPerm); err != nil {
		return Outcome{}, fmt.Errorf("creating the lane's directory: %w", err)
	}
	root, err := os.OpenRoot(c.RepoRoot)
	if err != nil {
		return Outcome{}, fmt.Errorf("opening the checkout: %w", err)
	}
	defer root.Close()
	if err := fsutil.WriteFileAtomicInRoot(root, rel, body, filePerm); err != nil {
		return Outcome{}, fmt.Errorf("writing %s: %w", rel, err)
	}
	lane.Brief = rel
	return Outcome{Note: fmt.Sprintf("rendered %s from %s at %s", rel, from, shortSHA(lane.BaseSHA))}, nil
}

// readBriefSources reads what a brief is rendered from out of the lane's base
// commit — git's objects, not the lane worktree's files — so the brief is the
// base it names however often it is rendered: an implementer's edit, commit,
// move or deletion in the worktree never reaches it. The lane is built off the
// default branch, so a record the base does not carry — an intent planned only
// on another branch, a spec not open there, no AGENTS.md — is refused rather
// than briefed from elsewhere.
func readBriefSources(repoRoot string, st State, lane *Lane) (briefSources, error) {
	var src briefSources
	if !gitutil.IsFullSHA(lane.BaseSHA) {
		return src, refuse(string(StageBrief), "", lane.ID, fmt.Sprintf("the lane records no base commit to read the record at (%s)", quoteOrNone(lane.BaseSHA)),
			"the worktree stage records it; restore the run's state file")
	}
	base := "the lane's base (" + lane.Branch + " at " + shortSHA(lane.BaseSHA) + ")"
	at := baseTree{root: repoRoot, sha: lane.BaseSHA}

	it, err := at.record(intent.IntentsRelDir, "itd", st.Intent)
	if err != nil {
		return src, fmt.Errorf("reading the intents at %s: %w", base, err)
	}
	if len(it) != 1 || it[0].folder != intent.BucketPlanned {
		where := "does not carry it"
		if len(it) > 0 {
			where = "holds it in " + folders(it)
		}
		return src, refuse(string(StageBrief), "", lane.ID,
			fmt.Sprintf("%s is not planned at %s: the default branch %s", st.Intent, base, where),
			"land the intent's planning on the default branch first; a lane is built off the default branch")
	}
	sp, err := at.record(spec.SpecsRelDir, "spc", st.Spec)
	if err != nil {
		return src, fmt.Errorf("reading the specs at %s: %w", base, err)
	}
	if len(sp) != 1 || sp[0].folder != spec.StatusOpen {
		return src, refuse(string(StageBrief), "", lane.ID,
			fmt.Sprintf("%s is not open at %s", st.Spec, base),
			"land the spec on the default branch first; a lane is built off the default branch")
	}

	read := func(e baseEntry, limit int64) ([]byte, error) {
		b, err := at.blob(e, limit)
		if err != nil {
			return nil, refuse(string(StageBrief), "", lane.ID, fmt.Sprintf("%s cannot be read at %s: %v", e.path, base, err),
				"the brief reads regular files within their caps; restore "+e.path+" on the default branch")
		}
		return b, nil
	}
	intentText, err := read(it[0], maxRecordBytes)
	if err != nil {
		return src, err
	}
	specText, err := read(sp[0], maxRecordBytes)
	if err != nil {
		return src, err
	}
	agentsEntry, found, err := at.file(ConventionsFile)
	if err != nil {
		return src, fmt.Errorf("reading %s at %s: %w", ConventionsFile, base, err)
	}
	if !found {
		return src, refuse(string(StageBrief), "", lane.ID, fmt.Sprintf("%s holds no %s, so the lane has no conventions to be briefed with", base, ConventionsFile),
			"write the repository's conventions into "+ConventionsFile+" on the default branch (`abcd prepare-this-repo` sets one up)")
	}
	agents, err := read(agentsEntry, maxRecordBytes)
	if err != nil {
		return src, err
	}
	src.intentPath, src.intentText = it[0].path, string(intentText)
	src.specPath, src.specText = sp[0].path, string(specText)
	if src.steps, err = laneSteps(st, lane, base, src.specText); err != nil {
		return src, err
	}
	src.conventions, src.conventionsFrom = conventionsSection(string(agents))

	ids := adrCiteRe.FindAllString(src.intentText, -1)
	slices.Sort(ids)
	var adrs []baseEntry
	if len(ids) > 0 {
		if adrs, err = at.list(decide.ADRsRelDir+"/", false); err != nil {
			return src, fmt.Errorf("reading the ADRs at %s: %w", base, err)
		}
	}
	for _, id := range slices.Compact(ids) {
		src.adrs = append(src.adrs, at.describeADR(adrs, id))
	}
	logEntry, found, err := at.file(DecisionsLogRel)
	if err != nil {
		return src, fmt.Errorf("reading %s at %s: %w", DecisionsLogRel, base, err)
	}
	if found {
		log, err := read(logEntry, maxDecisionsBytes)
		if err != nil {
			return src, err
		}
		src.decisions = decisionsNaming(string(log), st.Intent, st.Spec)
	}
	return src, nil
}

// laneSteps reads the spec's steps at the lane's base and holds the lane's own
// step to them: the lane was opened for a step as the spec listed it when the
// run started, and the brief names the steps before it, so a base whose spec
// lists that step under another title — reordered or rewritten since, which a
// run does not follow — or whose steps cannot be read is refused rather than
// briefed with the wrong predecessors (unrecognized-input-never-writes).
func laneSteps(st State, lane *Lane, base, specText string) ([]spec.Step, error) {
	steps, err := spec.Steps(specText)
	if err != nil {
		return nil, refuse(string(StageBrief), "", lane.ID, fmt.Sprintf("%s's steps cannot be read at %s: %v", st.Spec, base, err),
			"rewrite "+spec.StepsHeading+" on the default branch as `abcd intent ready "+st.Intent+"` describes")
	}
	remedy := "restore " + st.Spec + "'s steps on the default branch as the run started from them: a run does not follow steps reordered mid-run"
	if n := lane.SpecStep; n < 1 || n > len(steps) {
		return nil, refuse(string(StageBrief), "", lane.ID, fmt.Sprintf("%s at %s lists %d step(s), none numbered %d, the step this lane was opened for", st.Spec, base, len(steps), n), remedy)
	}
	if got := steps[lane.SpecStep-1]; got.Title != lane.StepTitle {
		return nil, refuse(string(StageBrief), "", lane.ID, fmt.Sprintf("%s at %s lists step %d as %q, not the %q this lane was opened for", st.Spec, base, got.Number, got.Title, lane.StepTitle), remedy)
	}
	return steps, nil
}

// baseTree reads the record out of one commit's objects through the isolated
// git environment, every listing and every blob under a cap.
type baseTree struct {
	root, sha string
}

// baseEntry is one file of the base commit as git's tree lists it.
type baseEntry struct {
	mode, kind, object, path string
	size                     int64
	// folder is the record's status folder, for an entry record found.
	folder string
}

// maxBaseListing caps one listing of the base commit's tree.
const maxBaseListing = 8 << 20

// list is `git ls-tree -l` of the base at path: a file names itself, and
// "dir/" names the directory's entries, one level deep unless recursive.
func (b baseTree) list(path string, recursive bool) ([]baseEntry, error) {
	args := []string{"ls-tree", "-l", "-z", "--full-tree"}
	if recursive {
		args = append(args, "-r")
	}
	out, err := gitutil.RunCapped(b.root, maxBaseListing, append(args, b.sha, "--", path)...)
	if err != nil {
		return nil, err
	}
	var entries []baseEntry
	for _, rec := range strings.Split(out, "\x00") {
		if rec == "" {
			continue
		}
		meta, p, ok := strings.Cut(rec, "\t")
		f := strings.Fields(meta) // <mode> <type> <object> <size>
		if !ok || len(f) != 4 {
			return nil, fmt.Errorf("git ls-tree returned a record it does not document: %q", rec)
		}
		e := baseEntry{mode: f[0], kind: f[1], object: f[2], path: p, size: -1}
		if n, err := strconv.ParseInt(f[3], 10, 64); err == nil {
			e.size = n
		}
		entries = append(entries, e)
	}
	return entries, nil
}

// file is the base's entry at rel, and whether the base carries anything
// there.
func (b baseTree) file(rel string) (baseEntry, bool, error) {
	entries, err := b.list(rel, false)
	if err != nil || len(entries) == 0 {
		return baseEntry{}, false, err
	}
	for _, e := range entries {
		if e.path == rel {
			return e, true, nil
		}
	}
	return baseEntry{}, false, nil
}

// record finds id among the family's record files under dir at the base, by
// the id its filename claims (recordid.SplitRecordFilename, the one splitter
// the ledger and the record gates share), with the status folder each copy
// sits in.
func (b baseTree) record(dir, family, id string) ([]baseEntry, error) {
	entries, err := b.list(dir+"/", true)
	if err != nil {
		return nil, err
	}
	var found []baseEntry
	for _, e := range entries {
		rest, ok := strings.CutPrefix(e.path, dir+"/")
		folder, name, ok2 := strings.Cut(rest, "/")
		if !ok || !ok2 || strings.Contains(name, "/") {
			continue
		}
		if got, _, ok := recordid.SplitRecordFilename(family, name); ok && recordid.SameID(got, id) {
			e.folder = folder
			found = append(found, e)
		}
	}
	return found, nil
}

// blob is the content of a regular file the base carries, refused when the
// base holds a link or anything but a file there, or when it passes limit.
func (b baseTree) blob(e baseEntry, limit int64) ([]byte, error) {
	if e.kind != "blob" || (e.mode != "100644" && e.mode != "100755") {
		return nil, fmt.Errorf("the base holds it as mode %s %s, not a regular file", e.mode, e.kind)
	}
	if e.size < 0 || e.size > limit {
		return nil, fmt.Errorf("it is %d bytes, past the %d-byte cap", e.size, limit)
	}
	out, err := gitutil.RunCapped(b.root, int(limit), "cat-file", "blob", e.object)
	if err != nil {
		return nil, err
	}
	return []byte(out), nil
}

// describeADR is the cited ADR as the base holds it, routed by the id its
// filename claims and confirmed by its frontmatter id, as record.Describe
// routes and confirms one on disk; titled by its first heading.
func (b baseTree) describeADR(entries []baseEntry, id string) citedADR {
	canonical := recordid.CanonADRID(id)
	for _, e := range entries {
		name := strings.TrimPrefix(e.path, decide.ADRsRelDir+"/")
		if canonical == "" || strings.Contains(name, "/") || !strings.HasSuffix(name, ".md") || recordid.ADRFileID(name) != canonical {
			continue
		}
		data, err := b.blob(e, maxRecordBytes)
		if err != nil {
			continue
		}
		lines := strings.Split(string(data), "\n")
		if recordid.CanonADRID(strings.Trim(strings.TrimSpace(frontmatter.Fields(lines)["id"].Value), `"'`)) != canonical {
			continue
		}
		title := strings.TrimSuffix(name, ".md")
		for _, ln := range lines {
			if t, ok := strings.CutPrefix(ln, "# "); ok {
				title = strings.TrimSpace(t)
				break
			}
		}
		return citedADR{ID: canonical, Title: title, Path: e.path, Found: true}
	}
	return citedADR{ID: id}
}

// folders names the status folders a record's copies sit in.
func folders(entries []baseEntry) string {
	var fs []string
	for _, e := range entries {
		fs = append(fs, e.folder+"/")
	}
	return strings.Join(fs, " and ")
}

// conventionsSection returns the working-conventions section of an AGENTS.md
// when the file marks one, and the whole file when it does not: a repository
// whose conventions are the whole file (as this one's are) is briefed with all
// of it, and says so.
func conventionsSection(agents string) (text, from string) {
	if i := strings.Index(agents, conventionsBegin); i >= 0 {
		rest := agents[i:]
		if nl := strings.IndexByte(rest, '\n'); nl >= 0 {
			if j := strings.Index(rest[nl+1:], conventionsEnd); j >= 0 {
				return strings.TrimSpace(rest[nl+1 : nl+1+j]), "its working-conventions section"
			}
		}
	}
	return strings.TrimSpace(agents), "the whole file: it marks no working-conventions section"
}

// decisionsNaming returns the decision log's entries — top-level list items,
// with the lines that continue them — that name any of ids as a whole word.
func decisionsNaming(log string, ids ...string) []string {
	var res []*regexp.Regexp
	for _, id := range ids {
		if id != "" {
			res = append(res, regexp.MustCompile(`(^|[^0-9A-Za-z-])`+regexp.QuoteMeta(id)+`($|[^0-9A-Za-z-])`))
		}
	}
	var out []string
	var cur []string
	flush := func() {
		if len(cur) == 0 {
			return
		}
		entry := strings.Join(cur, "\n")
		for _, re := range res {
			if re.MatchString(entry) {
				out = append(out, entry)
				break
			}
		}
		cur = nil
	}
	for _, line := range strings.Split(log, "\n") {
		switch {
		case strings.HasPrefix(line, "- "):
			flush()
			cur = []string{line}
		case cur != nil && (strings.HasPrefix(line, " ") || strings.HasPrefix(line, "\t")):
			cur = append(cur, line)
		default:
			flush()
		}
	}
	flush()
	return out
}

// lostConnectionRule is the implementer's half of a lost connection
// (iss-2610080620372731, ruling Q1 of 2026-10-09): the run waits the outage
// out on one shared probe, so an agent cut off mid-task neither retries nor
// commits; it yields with the one line `implement step --restart --yielded`
// takes, and the fresh agent starts from the lane's last commit with the
// uncommitted edits saved aside for review. The rule never says "aside": the
// brief never names the aside (restart.go).
const lostConnectionRule = "## A lost connection\n\n" +
	"If the network (git, gh, a download) or the model service fails you (a connection error, an API\n" +
	"error, overloaded, a timed-out request): do not retry, and do not commit what you have. Stop, and\n" +
	"end your hand-back with exactly one line, `" + NetworkLinePrefix + " <the failing command>`, and no receipt.\n" +
	"The run waits the outage out and restarts the lane from its last commit; your uncommitted edits are\n" +
	"kept for review, never built on. A usage or rate limit is not a lost connection.\n\n"

// renderBrief writes the brief. laneDir is the lane's directory as the
// implementer, working in another checkout, must address it: absolute.
func renderBrief(st State, lane Lane, laneDir string, src briefSources) []byte {
	var b bytes.Buffer
	p := func(format string, a ...any) { fmt.Fprintf(&b, format, a...) }
	at := func(name string) string { return filepath.Join(laneDir, name) }

	p("# Lane brief: %s of %s\n\n", lane.ID, st.RunID)
	p("You are the implementer of %s in the run `abcd build %s` started. You did not write this brief and\n", lane.ID, st.Key)
	p("no one will answer a question about it: what the record below does not settle, you settle and say so in\n")
	p("your report.\n\n")
	p("Rendered from, at the lane's base (%s, %s):\n\n", lane.Branch, lane.BaseSHA)
	p("- the intent: %s, `%s`\n", st.Intent, src.intentPath)
	p("- the spec: %s, `%s`\n", st.Spec, src.specPath)
	p("- the conventions: `%s`, %s\n", ConventionsFile, src.conventionsFrom)
	p("- the decisions the intent cites: %d ADR(s), and %d entr%s of `%s` naming %s or %s\n\n",
		len(src.adrs), len(src.decisions), plural(len(src.decisions), "y", "ies"), DecisionsLogRel, st.Intent, st.Spec)

	p("## Your lane\n\n")
	p("- Build spec step %d, %q, and nothing else of the spec.\n", lane.SpecStep, lane.StepTitle)
	p("- Work only in the worktree `%s`.\n", lane.Worktree)
	p("- Commit on its branch, `%s`, cut from the default branch at `%s`. Never push, never switch\n", lane.Branch, lane.BaseSHA)
	p("  branch, and never commit anywhere else.\n\n")

	// The steps before the lane's (itd-2609212103565953, criterion 4): what the
	// spec's author ordered ahead of this step, and what landed each, so the
	// implementer builds on them rather than again.
	p("## The spec's steps before yours\n\n")
	renderEarlierSteps(&b, st, lane, src.steps)

	p("## What you hand back\n\n")
	p("Write these three files, then stop:\n\n")
	p("1. Your report, `%s`: what you built, what you did not, and what a reviewer should look at.\n", at(ReportFileName))
	p("   It carries no verdict: only the loop records a verdict, from the validators it runs after you.\n")
	p("2. The definition of done's output, `%s`: run the definition of done the conventions below\n", at(DoDFileName))
	p("   state, in your worktree, and write its whole output there.\n")
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
	p("  \"resolves\": [{\"issue\": \"iss-<N>\", \"commit\": \"<the commit of yours that fixed it>\", \"note\": \"<the resolution>\",\n")
	p("                \"impact\": \"additive|breaking|fix|internal\", \"grounds\": \"pursued: <what is expected, and what would show it wrong>\"}]\n")
	p("}\n")
	p("```\n\n")
	p("`resolves` names each capture your lane fixed, or is left out when it fixed none. Do not resolve\n")
	p("a capture yourself: the landing runs `capture resolve` for each one named here, with its commit.\n\n")
	p("`output` and `report` are paths inside `%s`. The loop verifies the receipt before anything else\n", laneDir)
	p("runs: every commit exists on the branch past its base, the definition of done's output exists and\n")
	p("its exit code is 0, and the report exists. A receipt short of any of these is refused, naming what\n")
	p("is missing, and the lane waits for a corrected one.\n\n")

	// The outbound policy (itd-152): the harness stamps a session URL and an
	// attribution footer onto what an agent posts, outside the agent's own
	// output, so the prompt that starts the agent is where the rule has to be.
	// It is quoted from scanner.OutboundPolicy, the one value the scanner, the
	// lint rules and the commit gates quote, never restated here, and it is the
	// brief's own instruction: a managed repository's conventions need not
	// carry it.
	p("## Outward-facing text\n\n")
	p("A pull-request body, an issue, a comment, a commit message and a release note are public the moment\n")
	p("they exist. This holds whatever the conventions below say:\n\n")
	p("> %s\n\n", scanner.OutboundPolicy)

	p("%s", lostConnectionRule)

	p("%s", fenceQuoteNote)
	p("---\n\n## The intent: %s\n\n<!-- begin %s -->\n\n%s\n\n<!-- end %s -->\n\n", st.Intent, src.intentPath, fenceQuote(strings.TrimSpace(src.intentText)), src.intentPath)
	p("## The spec: %s\n\n<!-- begin %s -->\n\n%s\n\n<!-- end %s -->\n\n", st.Spec, src.specPath, fenceQuote(strings.TrimSpace(src.specText)), src.specPath)
	p("## The conventions: %s\n\n<!-- begin %s -->\n\n%s\n\n<!-- end %s -->\n\n", ConventionsFile, ConventionsFile, fenceQuote(src.conventions), ConventionsFile)

	p("## The decisions the intent cites\n\n")
	if len(src.adrs) == 0 {
		p("The intent cites no ADR.\n")
	}
	for _, a := range src.adrs {
		if a.Found {
			p("- %s — %s (`%s`)\n", a.ID, a.Title, a.Path)
		} else {
			p("- %s — not found at the lane's base\n", a.ID)
		}
	}
	p("\n### Entries of `%s` naming %s or %s\n\n", DecisionsLogRel, st.Intent, st.Spec)
	if len(src.decisions) == 0 {
		p("None.\n")
	}
	for _, d := range src.decisions {
		p("%s\n", d)
	}
	return b.Bytes()
}

// renderEarlierSteps writes the brief's steps section: the lane's step among
// the spec's, and each step before it with what landed it — the `landed:` the
// spec at the base records, and the lane of this run that built it.
func renderEarlierSteps(b *bytes.Buffer, st State, lane Lane, steps []spec.Step) {
	if len(steps) == 1 && steps[0].Implicit {
		fmt.Fprintf(b, "%s lists no steps, so this lane builds the whole spec: no step comes before it.\n\n", st.Spec)
		return
	}
	fmt.Fprintf(b, "This lane builds step %d of the %d %s lists", lane.SpecStep, len(steps), st.Spec)
	if lane.SpecStep <= 1 {
		fmt.Fprintf(b, ": no step comes before it.\n\n")
		return
	}
	fmt.Fprintf(b, ". The steps before it, and what landed each:\n\n")
	for _, s := range steps[:lane.SpecStep-1] {
		var what []string
		if s.Landed != "" {
			what = append(what, "landed: "+s.Landed+" (the spec at the lane's base)")
		}
		for _, l := range st.Lanes {
			if l.ID == lane.ID || l.SpecStep != s.Number {
				continue
			}
			built := "built by " + l.ID + " of this run"
			if l.Branch != "" {
				built += fmt.Sprintf(": branch `%s` at %s", l.Branch, l.HeadSHA)
			}
			if l.PR > 0 {
				built += fmt.Sprintf(", pull request #%d", l.PR)
			}
			what = append(what, built)
		}
		if len(what) == 0 {
			what = append(what, "not marked landed at the lane's base, and no lane of this run built it")
		}
		fmt.Fprintf(b, "%d. %q — %s\n", s.Number, s.Title, strings.Join(what, "; "))
	}
	b.WriteString("\n")
}

// relabel re-labels a refusal made for another stage as this stage's, so the
// caller is told the stage it asked for.
func relabel(err error, stage Stage) error {
	if r, ok := AsRefusal(err); ok {
		c := *r
		c.Stage = string(stage)
		return &c
	}
	return err
}

func quoteOrNone(s string) string {
	if s == "" {
		return "none"
	}
	return fmt.Sprintf("%q", s)
}

func shortSHA(sha string) string {
	if len(sha) > 12 {
		return sha[:12]
	}
	return sha
}

func plural(n int, one, many string) string {
	if n == 1 {
		return one
	}
	return many
}

// fenceMarkerEscaper writes an HTML comment opener or closer inside quoted text
// with its second hyphen as the entity `&#45;` (`<!-&#45;`, `-&#45;>`): a
// markdown view still shows the text as written, and the raw text holds no
// marker, so no quote can close its `<!-- end -->` fence or open another.
var fenceMarkerEscaper = strings.NewReplacer("<!--", "<!-&#45;", "-->", "-&#45;>")

// fenceQuote is text quoted between a brief's `<!-- begin/end -->` markers,
// with every comment marker in it escaped (fenceMarkerEscaper). Every quote a
// brief fences goes through it: the intent, the spec, the issue, its remedy
// and the conventions.
func fenceQuote(s string) string { return fenceMarkerEscaper.Replace(s) }

// fenceQuoteNote tells a brief's reader the one substitution fenceQuote makes.
const fenceQuoteNote = "The records below are quoted as the lane's base holds them, save one substitution: an HTML\n" +
	"comment opener or closer inside a quote has its second hyphen written `&#45;`, so no quote can end\n" +
	"its fence early.\n\n"
