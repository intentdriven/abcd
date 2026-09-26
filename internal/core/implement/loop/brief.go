package loop

// brief.go is the brief renderer (spec piece 5; criterion 3): the one file a
// lane's implementer is handed. It is rendered from the lane's base — the
// worktree the worktree step made, on the default branch — so the implementer
// reads exactly the record its branch builds on: the intent, the spec, the
// conventions section of AGENTS.md, and the decisions the intent cites. It
// names each source and where it was read, and tells the implementer where its
// report, the definition of done's output and its receipt go and what shape
// the receipt takes, because the receipt's verifier (piece 7) holds it to that.

import (
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"

	"github.com/intentdriven/abcd/internal/core/intent"
	"github.com/intentdriven/abcd/internal/core/record"
	"github.com/intentdriven/abcd/internal/core/spec"
	"github.com/intentdriven/abcd/internal/fsutil"
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
func laneRel(runID, laneID string, step StepName) (string, error) {
	if !ValidRunID(runID) || !ValidLaneID(laneID) {
		return "", refuse(string(step), "", "", fmt.Sprintf("%q of %q is not a lane the loop opened, so no lane path is built from it", laneID, runID),
			"the loop names its runs and lanes; restore the run's state file")
	}
	return runRel(runID) + "/" + laneID, nil
}

// laneFile is a file of a lane's directory, relative to the checkout root.
func laneFile(runID, laneID string, step StepName, name string) (string, error) {
	dir, err := laneRel(runID, laneID, step)
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
}

// citedADR is one ADR the intent cites, as the lane's base holds it.
type citedADR struct {
	ID, Title, Path string
	Found           bool
}

// briefStep is the brief step's body: it renders the brief from the lane's
// base into the lane's directory, replacing what an interrupted call wrote.
func briefStep(c Context, lane *Lane) (Outcome, error) {
	lw, err := laneWorktree(c.RepoRoot, c.State.RunID, lane.ID)
	if err != nil {
		return Outcome{}, relabel(err, StepBrief)
	}
	if lane.Worktree == "" || lane.Worktree != lw.Path {
		return Outcome{}, refuse(string(StepBrief), "", lane.ID,
			"the lane has no worktree the loop made (its state names "+quoteOrNone(fsutil.RedactHome(lane.Worktree))+")",
			"the worktree step makes it; restore the run's state file")
	}
	src, err := readBriefSources(lw.Path, c.State, lane)
	if err != nil {
		return Outcome{}, err
	}
	dirRel, err := laneRel(c.State.RunID, lane.ID, StepBrief)
	if err != nil {
		return Outcome{}, err
	}
	rel := dirRel + "/" + BriefFileName
	body := renderBrief(c.State, *lane, filepath.Join(c.RepoRoot, filepath.FromSlash(dirRel)), src)
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
	return Outcome{Note: fmt.Sprintf("rendered %s from %s, %s, %s (%s) and %d cited decision(s) at %s",
		rel, c.State.Intent, c.State.Spec, ConventionsFile, src.conventionsFrom, len(src.adrs)+len(src.decisions), shortSHA(lane.BaseSHA))}, nil
}

// readBriefSources reads what a brief is rendered from out of the lane's
// worktree. The lane is built off the default branch, so a record the default
// branch does not carry — an intent planned only on another branch, a spec not
// open there, no AGENTS.md — is refused rather than briefed from elsewhere.
func readBriefSources(worktree string, st State, lane *Lane) (briefSources, error) {
	var src briefSources
	base := "the lane's base (" + lane.Branch + " at " + shortSHA(lane.BaseSHA) + ")"
	corpus, err := intent.Load(worktree)
	if err != nil {
		return src, fmt.Errorf("reading the intents at %s: %w", base, err)
	}
	it, ok := corpus.Lookup(st.Intent)
	if !ok || it.Bucket != "planned" {
		where := "does not carry it"
		if ok {
			where = "holds it in " + it.Bucket + "/"
		}
		return src, refuse(string(StepBrief), "", lane.ID,
			fmt.Sprintf("%s is not planned at %s: the default branch %s", st.Intent, base, where),
			"land the intent's planning on the default branch first; a lane is built off the default branch")
	}
	store, err := spec.Load(worktree)
	if err != nil {
		return src, fmt.Errorf("reading the specs at %s: %w", base, err)
	}
	sp, ok := store.Lookup(st.Spec)
	if !ok || sp.Status != "open" {
		return src, refuse(string(StepBrief), "", lane.ID,
			fmt.Sprintf("%s is not open at %s", st.Spec, base),
			"land the spec on the default branch first; a lane is built off the default branch")
	}
	root, err := os.OpenRoot(worktree)
	if err != nil {
		return src, fmt.Errorf("opening the lane's worktree: %w", err)
	}
	defer root.Close()

	read := func(rel string, limit int64) ([]byte, error) {
		b, err := fsutil.ReadGuardedInRoot(root, rel, limit)
		if err != nil {
			return nil, refuse(string(StepBrief), "", lane.ID, fmt.Sprintf("%s cannot be read at %s: %v", rel, base, err),
				"the brief reads regular files within their caps; restore "+rel+" on the default branch")
		}
		return b, nil
	}
	intentText, err := read(it.Path, maxRecordBytes)
	if err != nil {
		return src, err
	}
	specText, err := read(sp.Path, maxRecordBytes)
	if err != nil {
		return src, err
	}
	agents, err := fsutil.ReadGuardedInRoot(root, ConventionsFile, maxRecordBytes)
	if errors.Is(err, fs.ErrNotExist) {
		return src, refuse(string(StepBrief), "", lane.ID, fmt.Sprintf("%s holds no %s, so the lane has no conventions to be briefed with", base, ConventionsFile),
			"write the repository's conventions into "+ConventionsFile+" on the default branch (`abcd prepare-this-repo` sets one up)")
	} else if err != nil {
		return src, refuse(string(StepBrief), "", lane.ID, fmt.Sprintf("%s cannot be read at %s: %v", ConventionsFile, base, err),
			"the brief reads a regular file within its cap; restore "+ConventionsFile+" on the default branch")
	}
	src.intentPath, src.intentText = it.Path, string(intentText)
	src.specPath, src.specText = sp.Path, string(specText)
	src.conventions, src.conventionsFrom = conventionsSection(string(agents))

	ids := adrCiteRe.FindAllString(src.intentText, -1)
	slices.Sort(ids)
	for _, id := range slices.Compact(ids) {
		a := citedADR{ID: id}
		if d, err := record.Describe(worktree, id); err == nil {
			a.ID, a.Title, a.Path, a.Found = d.ID, d.Title, d.Path, true
		}
		src.adrs = append(src.adrs, a)
	}
	log, err := fsutil.ReadGuardedInRoot(root, DecisionsLogRel, maxDecisionsBytes)
	switch {
	case errors.Is(err, fs.ErrNotExist):
	case err != nil:
		return src, refuse(string(StepBrief), "", lane.ID, fmt.Sprintf("%s cannot be read at %s: %v", DecisionsLogRel, base, err),
			"the brief reads a regular file within its cap; restore "+DecisionsLogRel+" on the default branch")
	default:
		src.decisions = decisionsNaming(string(log), st.Intent, st.Spec)
	}
	return src, nil
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
	p("  \"model\": \"<the model id your harness reports, or leave the field out>\"\n")
	p("}\n")
	p("```\n\n")
	p("`output` and `report` are paths inside `%s`. The loop verifies the receipt before anything else\n", laneDir)
	p("runs: every commit exists on the branch past its base, the definition of done's output exists and\n")
	p("its exit code is 0, and the report exists. A receipt short of any of these is refused, naming what\n")
	p("is missing, and the lane waits for a corrected one.\n\n")

	p("---\n\n## The intent: %s\n\n<!-- begin %s -->\n\n%s\n\n<!-- end %s -->\n\n", st.Intent, src.intentPath, strings.TrimSpace(src.intentText), src.intentPath)
	p("## The spec: %s\n\n<!-- begin %s -->\n\n%s\n\n<!-- end %s -->\n\n", st.Spec, src.specPath, strings.TrimSpace(src.specText), src.specPath)
	p("## The conventions: %s\n\n<!-- begin %s -->\n\n%s\n\n<!-- end %s -->\n\n", ConventionsFile, ConventionsFile, src.conventions, ConventionsFile)

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

// relabel re-labels a refusal made for another step as this step's, so the
// caller is told the step it asked for.
func relabel(err error, step StepName) error {
	if r, ok := AsRefusal(err); ok {
		c := *r
		c.Step = string(step)
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
