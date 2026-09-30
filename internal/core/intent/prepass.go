package intent

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"syscall"

	"github.com/intentdriven/abcd/internal/core/recordid"
	"github.com/intentdriven/abcd/internal/fsutil"
	"github.com/intentdriven/abcd/internal/termsafe"
)

// prepass.go — the pre-pass before the planning interview (itd-42 as re-scoped
// on 2026-09-21, spc-2609211918551301; itd-84's next rung).
//
// Before a draft is interviewed, the pre-pass reads the brief's invariants, the
// principles, a one-line index of every other intent and the draft, and writes
// the coherence questions into the planning brief the interview starts from.
// The judgement rides the host (adr-25); this file is the binary's half, in two
// calls a front door makes either side of the host's pass:
//
//   - ASSEMBLE (AssemblePrepass) reads the four inputs and returns them, with
//     the answers an overlap is asked with, the rules the host's return is held
//     to, and a digest over all of it. It writes nothing.
//   - WRITE (WritePrepassBrief) reads the host's untrusted findings, validates
//     their shape fail-closed against the input as it stands now, and writes the
//     planning brief under the local tier — the only file the pre-pass writes.
//
// Grounded where it asserts, Socratic where it questions (the itd-41 pattern
// the intent names): a conflict is written as one only when the anchor it names
// exists and both of its quotes occur verbatim — the anchor's in the invariant
// or principle, the draft's in the draft. A concern the binary cannot anchor
// that way is not refused: it is written as a question marked unanchored, with
// the reason it could not be anchored, so the person still sees it and nothing
// is asserted on a fiction. A payload whose SHAPE is wrong is refused with
// nothing written.
//
// The loader is the interview's own (decision 3): it is not shared with the
// phase negotiator or the fidelity reviewer, and it reads nothing beyond the
// four inputs, which is the budget.

// PrepassInputType and PrepassFindingsType are the two documents' _type values.
const (
	PrepassInputType    = "abcd/intent-prepass-input/v1"
	PrepassFindingsType = "abcd/intent-prepass-findings/v1"
)

// PlanningBriefsRelDir is where a planning brief lives: the local tier, beside
// the briefs a session writes by hand (commands/intent.md, Autonomous runs).
const PlanningBriefsRelDir = ".abcd/.work.local/scratch/planning-briefs"

// invariantsRelPath and principlesRelDir are two of the pre-pass's four inputs.
const (
	invariantsRelPath = ".abcd/development/brief/02-constraints/03-invariants.md"
	principlesRelDir  = ".abcd/development/principles"
)

// prepassMarker opens every brief the pre-pass writes. A brief without it was
// written by someone else and is never replaced.
const prepassMarker = "<!-- abcd:intent-prepass v1 -->"

// Caps on the untrusted return.
const (
	maxPrepassFindingsBytes = 1 << 20
	maxPrepassBriefBytes    = 4 << 20
	maxPrepassItems         = 100
	prepassProseCap         = 2000
	prepassQuoteCap         = 1000
)

// PrepassAnswers are the four standard answers an overlap is asked with
// (decision 2), in the order the brief lists them.
var PrepassAnswers = []string{"keep-both", "bundle", "supersede", "refine"}

var prepassAnswerLabel = map[string]string{
	"keep-both": "Keep both",
	"bundle":    "Bundle",
	"supersede": "Supersede",
	"refine":    "Refine",
}

// prepassHomes are the record homes a decomposition routes a part to (itd-84).
var prepassHomes = map[string]bool{"intent": true, "adr": true, "principle": true, "brief": true}

// prepassRules is the contract the write enforces, handed to the host in the
// input so the host is told what the binary checks.
var prepassRules = []string{
	"return one JSON object: _type " + PrepassFindingsType + ", intent, input_digest (copied from this input), and any of summary, decomposition, conflicts, overlaps, unanchored, blocks_planning; nothing else is accepted",
	"conflicts: each names exactly one anchor, an invariant by its number or a principle by its path, and carries anchor_quote (verbatim from that anchor), draft_quote (verbatim from the draft) and question",
	"a conflict whose anchor is absent, or whose quotes do not occur verbatim (at least 12 characters once whitespace is collapsed), is written as an unanchored question, never as a conflict",
	"overlaps: each names a sibling from the index and a question; it is asked with the four answers keep-both, bundle, supersede, refine; an optional recommendation names one answer and its reason, and is written as prose beside the question, never as a marked option",
	"unanchored: a concern with no invariant, principle or sibling to anchor it is a question, and is written marked unanchored",
	"decomposition: each part names its home, one of intent, adr, principle, brief; it is an ungraded proposal",
	"at most 100 items in each list; every question, quote and reason is non-empty",
}

// PrepassInvariant is one numbered invariant, as the brief's register states it.
type PrepassInvariant struct {
	Number int    `json:"number"`
	Title  string `json:"title"`
	Text   string `json:"text"`
}

// PrepassPrinciple is one principle: its path, its title and its rule
// paragraph (the stance, without the argument for it).
type PrepassPrinciple struct {
	Path  string `json:"path"`
	Title string `json:"title"`
	Rule  string `json:"rule"`
}

// PrepassIndexEntry is one line of the sibling index.
type PrepassIndexEntry struct {
	ID    string `json:"id"`
	Title string `json:"title"`
	Shelf string `json:"shelf"`
}

// PrepassInput is everything the host's pass reads, and nothing else.
type PrepassInput struct {
	Type       string              `json:"_type"`
	Intent     string              `json:"intent"`
	DraftPath  string              `json:"draft_path"`
	Draft      string              `json:"draft"`
	Invariants []PrepassInvariant  `json:"invariants"`
	Principles []PrepassPrinciple  `json:"principles"`
	Index      []PrepassIndexEntry `json:"index"`
	Answers    []string            `json:"answers"`
	Rules      []string            `json:"rules"`
	Warnings   []string            `json:"warnings"`
	Digest     string              `json:"input_digest"`
}

// PrepassBriefResult reports one written brief.
type PrepassBriefResult struct {
	Intent     string `json:"intent"`
	BriefPath  string `json:"brief_path"`
	Conflicts  int    `json:"conflicts"`
	Overlaps   int    `json:"overlaps"`
	Unanchored int    `json:"unanchored"`
	// Demoted counts the conflicts and overlaps the host returned that could
	// not be anchored, and so are among the Unanchored questions.
	Demoted  int      `json:"demoted"`
	Warnings []string `json:"warnings"`
}

var (
	prepassItemRe  = regexp.MustCompile(`^([0-9]+)\.\s+(.*)$`)
	prepassTitleRe = regexp.MustCompile(`^\*\*(.+?)\*\*`)
	prepassH1Re    = regexp.MustCompile(`^#\s+(.+?)\s*#*\s*$`)
)

// AssemblePrepass reads the four inputs for the draft intentID. It refuses an
// id that is malformed, absent, or not on the drafts shelf; a missing
// invariants file or principles directory degrades the pass with a warning
// instead. It writes nothing.
func AssemblePrepass(repoRoot, intentID string) (PrepassInput, error) {
	if !recordid.ValidIntentID(intentID) {
		return PrepassInput{}, fmt.Errorf("intent prepass: id %q must match ^itd-[0-9]+$", intentID)
	}
	corpus, err := Load(repoRoot)
	if err != nil {
		return PrepassInput{}, err
	}
	it, ok := corpus.Lookup(intentID)
	if !ok {
		return PrepassInput{}, fmt.Errorf("intent prepass: %s not found in any bucket; `abcd intent` renders the store, and the pre-pass runs on a draft it lists", intentID)
	}
	if it.Bucket != BucketDrafts {
		return PrepassInput{}, fmt.Errorf("intent prepass: %s is on %s/; the pre-pass prepares the interview of a draft, and only a record on drafts/ has one to come: run it on a draft `abcd intent` lists", it.ID, it.Bucket)
	}
	draft, err := readRepoFile(filepath.Join(repoRoot, it.Path), it.Path)
	if err != nil {
		return PrepassInput{}, err
	}
	in := PrepassInput{
		Type:       PrepassInputType,
		Intent:     it.ID,
		DraftPath:  filepath.ToSlash(it.Path),
		Draft:      string(draft),
		Invariants: []PrepassInvariant{},
		Principles: []PrepassPrinciple{},
		Index:      []PrepassIndexEntry{},
		Answers:    append([]string(nil), PrepassAnswers...),
		Rules:      append([]string(nil), prepassRules...),
		Warnings:   []string{},
	}
	invariants, err := readRepoFile(filepath.Join(repoRoot, filepath.FromSlash(invariantsRelPath)), invariantsRelPath)
	switch {
	case errors.Is(err, fs.ErrNotExist):
		in.Warnings = append(in.Warnings, "the invariants register "+invariantsRelPath+" is missing: no conflict with an invariant can be anchored, so every such concern is asked as an unanchored question")
	case err != nil:
		return PrepassInput{}, err
	default:
		in.Invariants = parsePrepassInvariants(string(invariants))
	}
	principles, err := readPrepassPrinciples(repoRoot)
	switch {
	case errors.Is(err, fs.ErrNotExist):
		in.Warnings = append(in.Warnings, "the principles directory "+principlesRelDir+" is missing: no conflict with a principle can be anchored")
	case err != nil:
		return PrepassInput{}, err
	default:
		in.Principles = principles
	}
	for _, sib := range corpus.Intents {
		if sib.Path == it.Path {
			continue
		}
		content, err := readRepoFile(filepath.Join(repoRoot, sib.Path), sib.Path)
		if err != nil {
			return PrepassInput{}, err
		}
		title := prepassTitle(string(content))
		if title == "" {
			title = sib.Slug
		}
		in.Index = append(in.Index, PrepassIndexEntry{ID: sib.ID, Title: title, Shelf: sib.Bucket})
	}
	in.Digest = prepassDigest(in)
	return in, nil
}

// prepassDigest is sha256 over the input's canonical JSON with the digest
// blank: it moves when any input moves.
func prepassDigest(in PrepassInput) string {
	in.Digest = ""
	raw, _ := json.Marshal(in) // a struct of strings, ints and slices of them: cannot fail
	sum := sha256.Sum256(raw)
	return "sha256:" + hex.EncodeToString(sum[:])
}

// parsePrepassInvariants reads the register's numbered items: an item opens at
// a column-0 "N. " line and runs to the next item or heading.
func parsePrepassInvariants(content string) []PrepassInvariant {
	var out []PrepassInvariant
	var cur *PrepassInvariant
	var body []string
	flush := func() {
		if cur != nil {
			cur.Text = strings.TrimSpace(strings.Join(body, "\n"))
			out = append(out, *cur)
		}
		cur, body = nil, nil
	}
	for _, line := range strings.Split(content, "\n") {
		if m := prepassItemRe.FindStringSubmatch(line); m != nil {
			flush()
			n, err := strconv.Atoi(m[1])
			if err != nil {
				continue
			}
			cur = &PrepassInvariant{Number: n}
			if t := prepassTitleRe.FindStringSubmatch(m[2]); t != nil {
				cur.Title = strings.TrimSpace(t[1])
			}
			body = []string{m[2]}
			continue
		}
		if strings.HasPrefix(line, "#") {
			flush()
			continue
		}
		if cur != nil {
			body = append(body, line)
		}
	}
	flush()
	if out == nil {
		out = []PrepassInvariant{}
	}
	return out
}

// readPrepassPrinciples reads every principle but the README, in name order.
func readPrepassPrinciples(repoRoot string) ([]PrepassPrinciple, error) {
	dir := filepath.Join(repoRoot, filepath.FromSlash(principlesRelDir))
	fi, err := os.Lstat(dir)
	if err != nil {
		return nil, err
	}
	if !fi.IsDir() {
		return nil, fmt.Errorf("intent prepass: %s is not a directory", principlesRelDir)
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("intent prepass: reading %s: %w", principlesRelDir, err)
	}
	out := []PrepassPrinciple{}
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() || !strings.HasSuffix(name, ".md") || strings.EqualFold(name, "README.md") {
			continue
		}
		rel := principlesRelDir + "/" + name
		data, err := readRepoFile(filepath.Join(dir, name), rel)
		if err != nil {
			return nil, err
		}
		title := prepassTitle(string(data))
		if title == "" {
			title = strings.TrimSuffix(name, ".md")
		}
		out = append(out, PrepassPrinciple{Path: rel, Title: title, Rule: prepassRule(string(data))})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Path < out[j].Path })
	return out, nil
}

// prepassTitle is a record's first level-one heading after its frontmatter.
func prepassTitle(content string) string {
	lines := strings.Split(content, "\n")
	i := 0
	if len(lines) > 0 && strings.TrimSpace(lines[0]) == "---" {
		for i = 1; i < len(lines) && strings.TrimSpace(lines[i]) != "---"; i++ {
		}
		i++
	}
	for ; i < len(lines); i++ {
		if m := prepassH1Re.FindStringSubmatch(lines[i]); m != nil {
			return strings.TrimSpace(m[1])
		}
	}
	return ""
}

// prepassRule is a principle's rule paragraph: the one opening "**The rule.**",
// else the first paragraph that is not a heading.
func prepassRule(content string) string {
	var paras []string
	var cur []string
	for _, line := range strings.Split(content, "\n") {
		if strings.TrimSpace(line) == "" {
			if len(cur) > 0 {
				paras = append(paras, strings.Join(cur, "\n"))
			}
			cur = nil
			continue
		}
		cur = append(cur, line)
	}
	if len(cur) > 0 {
		paras = append(paras, strings.Join(cur, "\n"))
	}
	for _, p := range paras {
		if strings.HasPrefix(p, "**The rule.**") {
			return p
		}
	}
	for _, p := range paras {
		if !strings.HasPrefix(p, "#") {
			return p
		}
	}
	return ""
}

// The host's return, decoded strictly.
type prepassReturn struct {
	Type           string            `json:"_type"`
	Intent         string            `json:"intent"`
	InputDigest    string            `json:"input_digest"`
	Summary        string            `json:"summary"`
	Decomposition  []prepassPart     `json:"decomposition"`
	Conflicts      []prepassConflict `json:"conflicts"`
	Overlaps       []prepassOverlap  `json:"overlaps"`
	Unanchored     []prepassConcern  `json:"unanchored"`
	BlocksPlanning []string          `json:"blocks_planning"`
}

type prepassPart struct {
	Part string `json:"part"`
	Home string `json:"home"`
}

type prepassConflict struct {
	Invariant   *int   `json:"invariant"`
	Principle   string `json:"principle"`
	AnchorQuote string `json:"anchor_quote"`
	DraftQuote  string `json:"draft_quote"`
	Question    string `json:"question"`
}

type prepassOverlap struct {
	Sibling        string                 `json:"sibling"`
	Question       string                 `json:"question"`
	Recommendation *prepassRecommendation `json:"recommendation"`
}

type prepassRecommendation struct {
	Answer string `json:"answer"`
	Reason string `json:"reason"`
}

type prepassConcern struct {
	Question string `json:"question"`
}

// A question as the brief writes it, after validation.
type prepassQuestion struct {
	kind     string // conflict | overlap | unanchored
	heading  string
	quotes   [][2]string // (who says, what)
	question string
	sibling  PrepassIndexEntry
	rec      *prepassRecommendation
	why      string // for an unanchored question the binary demoted
}

// WritePrepassBrief validates the host's findings for the draft intentID
// against the input as it stands now and writes the planning brief. A payload
// that does not validate is refused with nothing written; a brief at the path
// that the pre-pass did not write is never replaced.
func WritePrepassBrief(repoRoot, intentID string, raw []byte) (PrepassBriefResult, error) {
	in, err := AssemblePrepass(repoRoot, intentID)
	if err != nil {
		return PrepassBriefResult{}, err
	}
	f, err := decodePrepassFindings(raw)
	if err != nil {
		return PrepassBriefResult{}, err
	}
	if f.Type != PrepassFindingsType {
		return PrepassBriefResult{}, fmt.Errorf("intent prepass: _type %q, want %q", termsafe.CleanProseLine(f.Type, 80), PrepassFindingsType)
	}
	if !recordid.SameID(f.Intent, in.Intent) {
		return PrepassBriefResult{}, fmt.Errorf("intent prepass: the findings are for %q, not %s", termsafe.CleanProseLine(f.Intent, 80), in.Intent)
	}
	if f.InputDigest != in.Digest {
		return PrepassBriefResult{}, fmt.Errorf("intent prepass: the findings were judged over input %q, and the record now assembles to %s: an input moved since the pass; re-run the pre-pass", termsafe.CleanProseLine(f.InputDigest, 80), in.Digest)
	}
	qs, res, err := validatePrepass(in, f)
	if err != nil {
		return PrepassBriefResult{}, err
	}
	brief := renderPrepassBrief(in, f, qs)

	root, err := os.OpenRoot(repoRoot)
	if err != nil {
		return PrepassBriefResult{}, fmt.Errorf("intent prepass: opening the repository: %w", err)
	}
	defer root.Close()
	rel := PlanningBriefsRelDir + "/" + in.Intent + ".md"
	if err := prepassMayReplace(root, rel); err != nil {
		return PrepassBriefResult{}, err
	}
	if err := fsutil.WriteFileAtomicInRoot(root, rel, []byte(brief), 0o644); err != nil {
		return PrepassBriefResult{}, fmt.Errorf("intent prepass: writing %s: %w", rel, err)
	}
	res.Intent = in.Intent
	res.BriefPath = rel
	res.Warnings = in.Warnings
	return res, nil
}

// ReadPrepassFindings reads the host's findings file for WritePrepassBrief:
// a regular file, never a symlink or a device, within the findings cap. The
// bytes stay untrusted until WritePrepassBrief validates them.
func ReadPrepassFindings(path string) ([]byte, error) {
	if strings.TrimSpace(path) == "" {
		return nil, errors.New("intent prepass: --findings-json names no file; pass the path the host wrote its findings to")
	}
	data, err := fsutil.ReadGuarded(path, maxPrepassFindingsBytes)
	switch {
	case err == nil:
		return data, nil
	case errors.Is(err, fsutil.ErrNotRegular) || errors.Is(err, syscall.ELOOP):
		return nil, fmt.Errorf("intent prepass: the findings %s are not a regular file (a symlink or a device is refused); write them to a plain file", path)
	case errors.Is(err, fsutil.ErrTooBig):
		return nil, fmt.Errorf("intent prepass: the findings %s exceed the %d-byte cap", path, maxPrepassFindingsBytes)
	case errors.Is(err, fs.ErrNotExist):
		return nil, fmt.Errorf("intent prepass: no findings at %s; write the host's findings there, or pass the path they are at", path)
	default:
		return nil, fmt.Errorf("intent prepass: reading the findings %s: %w", path, err)
	}
}

func decodePrepassFindings(raw []byte) (prepassReturn, error) {
	if len(raw) > maxPrepassFindingsBytes {
		return prepassReturn{}, fmt.Errorf("intent prepass: the findings exceed the %d-byte cap", maxPrepassFindingsBytes)
	}
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.DisallowUnknownFields()
	var f prepassReturn
	if err := dec.Decode(&f); err != nil {
		return prepassReturn{}, fmt.Errorf("intent prepass: the findings are not the findings shape: %w", err)
	}
	if _, err := dec.Token(); !errors.Is(err, io.EOF) {
		return prepassReturn{}, errors.New("intent prepass: the findings carry data after the object")
	}
	return f, nil
}

// prepassMayReplace refuses a path that holds anything but a brief the
// pre-pass wrote.
func prepassMayReplace(root *os.Root, rel string) error {
	fi, err := root.Lstat(rel)
	if errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("intent prepass: stat %s: %w", rel, err)
	}
	if !fi.Mode().IsRegular() {
		return fmt.Errorf("intent prepass: %s is not a regular file; nothing was written", rel)
	}
	data, err := fsutil.ReadGuardedInRoot(root, rel, maxPrepassBriefBytes)
	if err != nil {
		return fmt.Errorf("intent prepass: reading %s: %w", rel, err)
	}
	if !strings.HasPrefix(string(data), prepassMarker+"\n") {
		return fmt.Errorf("intent prepass: %s is not written by the pre-pass, so it is someone's brief and is left alone; move it aside and re-run", rel)
	}
	return nil
}

// validatePrepass checks the shape fail-closed and anchors each concern,
// demoting an unanchorable conflict or overlap to an unanchored question.
func validatePrepass(in PrepassInput, f prepassReturn) ([]prepassQuestion, PrepassBriefResult, error) {
	var res PrepassBriefResult
	for name, n := range map[string]int{
		"decomposition": len(f.Decomposition), "conflicts": len(f.Conflicts), "overlaps": len(f.Overlaps),
		"unanchored": len(f.Unanchored), "blocks_planning": len(f.BlocksPlanning),
	} {
		if n > maxPrepassItems {
			return nil, res, fmt.Errorf("intent prepass: %s carries %d items; at most %d", name, n, maxPrepassItems)
		}
	}
	for i, p := range f.Decomposition {
		if prepassBlank(p.Part) || !prepassHomes[p.Home] {
			return nil, res, fmt.Errorf("intent prepass: decomposition[%d] needs a part and a home of intent, adr, principle or brief", i)
		}
	}
	for i, b := range f.BlocksPlanning {
		if prepassBlank(b) {
			return nil, res, fmt.Errorf("intent prepass: blocks_planning[%d] is blank", i)
		}
	}
	var anchored, demoted []prepassQuestion
	for i, c := range f.Conflicts {
		if prepassBlank(c.Question) || prepassBlank(c.AnchorQuote) || prepassBlank(c.DraftQuote) {
			return nil, res, fmt.Errorf("intent prepass: conflicts[%d] needs anchor_quote, draft_quote and question", i)
		}
		if (c.Invariant == nil) == (c.Principle == "") {
			return nil, res, fmt.Errorf("intent prepass: conflicts[%d] names exactly one anchor, an invariant or a principle", i)
		}
		var who, text, why string
		if c.Invariant != nil {
			who = "invariant " + strconv.Itoa(*c.Invariant)
			if inv, ok := prepassInvariantNamed(in, *c.Invariant); ok {
				text = inv.Text
				who = "Invariant " + strconv.Itoa(inv.Number) + prepassDash(inv.Title)
			} else {
				why = "the brief carries no invariant " + strconv.Itoa(*c.Invariant)
			}
		} else {
			if p, ok := prepassPrincipleAt(in, c.Principle); ok {
				text = p.Rule
				who = "Principle" + prepassDash(p.Title)
			} else {
				why = "the record holds no principle at " + c.Principle
			}
		}
		if why == "" {
			why = prepassQuoteMiss(c.AnchorQuote, text, prepassAnchorName(c))
		}
		if why == "" {
			why = prepassQuoteMiss(c.DraftQuote, in.Draft, "the draft")
		}
		if why != "" {
			demoted = append(demoted, prepassQuestion{kind: "unanchored", question: c.Question, why: why})
			continue
		}
		anchored = append(anchored, prepassQuestion{
			kind:     "conflict",
			heading:  "Conflict with " + strings.ToLower(who[:1]) + who[1:],
			quotes:   [][2]string{{who, c.AnchorQuote}, {"The draft", c.DraftQuote}},
			question: c.Question,
		})
		res.Conflicts++
	}
	for i, o := range f.Overlaps {
		if prepassBlank(o.Question) || prepassBlank(o.Sibling) {
			return nil, res, fmt.Errorf("intent prepass: overlaps[%d] needs a sibling and a question", i)
		}
		if r := o.Recommendation; r != nil {
			if prepassAnswerLabel[r.Answer] == "" {
				return nil, res, fmt.Errorf("intent prepass: overlaps[%d] recommends %q; the answers are %s", i, termsafe.CleanProseLine(r.Answer, 40), strings.Join(PrepassAnswers, ", "))
			}
			if prepassBlank(r.Reason) {
				return nil, res, fmt.Errorf("intent prepass: overlaps[%d] recommends %s without its reason", i, r.Answer)
			}
		}
		sib, ok := prepassSibling(in, o.Sibling)
		if !ok {
			why := "no intent on any shelf is " + o.Sibling
			if recordid.SameID(o.Sibling, in.Intent) {
				why = o.Sibling + " is the draft itself"
			}
			demoted = append(demoted, prepassQuestion{kind: "unanchored", question: o.Question, why: why})
			continue
		}
		anchored = append(anchored, prepassQuestion{
			kind: "overlap", heading: "Overlap with " + sib.ID + prepassDash(sib.Title) + " (" + sib.Shelf + ")",
			question: o.Question, sibling: sib, rec: o.Recommendation,
		})
		res.Overlaps++
	}
	var own []prepassQuestion
	for i, u := range f.Unanchored {
		if prepassBlank(u.Question) {
			return nil, res, fmt.Errorf("intent prepass: unanchored[%d] has no question", i)
		}
		own = append(own, prepassQuestion{kind: "unanchored", question: u.Question})
	}
	res.Demoted = len(demoted)
	res.Unanchored = len(demoted) + len(own)
	qs := append(append(anchored, own...), demoted...)
	return qs, res, nil
}

func prepassBlank(s string) bool { return strings.TrimSpace(s) == "" }

func prepassDash(title string) string {
	if title == "" {
		return ""
	}
	return " — " + title
}

func prepassAnchorName(c prepassConflict) string {
	if c.Invariant != nil {
		return "invariant " + strconv.Itoa(*c.Invariant)
	}
	return "principle " + c.Principle
}

// prepassQuoteMiss says why quote does not locate itself in text, or "".
func prepassQuoteMiss(quote, text, where string) string {
	q := collapseSpace(quote)
	if len([]rune(q)) < minQuoteChars {
		return fmt.Sprintf("the quote from %s is shorter than %d characters", where, minQuoteChars)
	}
	if !strings.Contains(collapseSpace(text), q) {
		return "the quote was not found in " + where
	}
	return ""
}

func prepassInvariantNamed(in PrepassInput, n int) (PrepassInvariant, bool) {
	for _, inv := range in.Invariants {
		if inv.Number == n {
			return inv, true
		}
	}
	return PrepassInvariant{}, false
}

func prepassPrincipleAt(in PrepassInput, p string) (PrepassPrinciple, bool) {
	p = path.Clean(strings.TrimSpace(p))
	for _, pr := range in.Principles {
		if pr.Path == p {
			return pr, true
		}
	}
	return PrepassPrinciple{}, false
}

func prepassSibling(in PrepassInput, id string) (PrepassIndexEntry, bool) {
	id = strings.TrimSpace(id)
	if !recordid.ValidIntentID(id) {
		return PrepassIndexEntry{}, false
	}
	for _, e := range in.Index {
		if recordid.SameID(e.ID, id) {
			return e, true
		}
	}
	return PrepassIndexEntry{}, false
}

// prepassLine and prepassProse clean untrusted text for the brief: one line,
// no terminal bytes, no markdown that could open structure.
func prepassLine(s string) string { return termsafe.CleanProseLine(s, prepassProseCap) }

func prepassQuote(s string) string { return termsafe.CleanProseLine(s, prepassQuoteCap) }

func renderPrepassBrief(in PrepassInput, f prepassReturn, qs []prepassQuestion) string {
	var b strings.Builder
	title := prepassLine(prepassTitle(in.Draft))
	b.WriteString(prepassMarker + "\n")
	fmt.Fprintf(&b, "# Planning brief — %s%s\n\n", in.Intent, strings.Replace(prepassDash(title), " — ", ": ", 1))
	fmt.Fprintf(&b, "Written by the pre-pass (`abcd intent prepass %s`) for the planning interview. "+
		"It read the draft, %d invariants, %d principles and a one-line index of %d other intents, input digest `%s`. "+
		"Nothing here is decided: every question below is the product thinker's to answer, and the pre-pass changed no record.\n\n",
		in.Intent, len(in.Invariants), len(in.Principles), len(in.Index), in.Digest)

	b.WriteString("## Summary back\n\n")
	if prepassBlank(f.Summary) {
		b.WriteString("_The pre-pass returned no summary._\n\n")
	} else {
		b.WriteString(termsafe.CleanProse(f.Summary, 4*prepassProseCap) + "\n\n")
	}

	b.WriteString("## Decomposition (an ungraded proposal)\n\n")
	if len(f.Decomposition) == 0 {
		b.WriteString("_None proposed._\n\n")
	} else {
		b.WriteString("| Part | Home |\n| --- | --- |\n")
		for _, p := range f.Decomposition {
			fmt.Fprintf(&b, "| %s | %s |\n", strings.ReplaceAll(prepassLine(p.Part), "|", `\|`), p.Home)
		}
		b.WriteString("\n")
	}

	b.WriteString("## Questions\n\n")
	if len(qs) == 0 {
		b.WriteString("_The pre-pass raised no question._\n\n")
	}
	for i, q := range qs {
		n := i + 1
		switch q.kind {
		case "conflict":
			fmt.Fprintf(&b, "### Q%d. %s\n\n", n, prepassLine(q.heading))
			for _, qt := range q.quotes {
				fmt.Fprintf(&b, "%s says:\n\n> %s\n\n", prepassLine(qt[0]), prepassQuote(qt[1]))
			}
			fmt.Fprintf(&b, "%s\n\n", prepassLine(q.question))
			b.WriteString("Lands as: a decision in the draft's `## Decisions` saying which gives, or a change to the draft's text before it is planned.\n\n")
		case "overlap":
			fmt.Fprintf(&b, "### Q%d. %s\n\n", n, prepassLine(q.heading))
			fmt.Fprintf(&b, "%s\n\n", prepassLine(q.question))
			for _, a := range PrepassAnswers {
				fmt.Fprintf(&b, "- **%s**\n", prepassAnswerLabel[a])
			}
			b.WriteString("\n")
			if q.rec != nil {
				reason := prepassLine(q.rec.Reason)
				if !strings.HasSuffix(reason, ".") && !strings.HasSuffix(reason, "!") && !strings.HasSuffix(reason, "?") {
					reason += "."
				}
				fmt.Fprintf(&b, "The pre-pass leans towards **%s**: %s\n\n", strings.ToLower(prepassAnswerLabel[q.rec.Answer]), reason)
			}
			sib := q.sibling.ID
			fmt.Fprintf(&b, "Lands as: keep both, a decision in the draft's `## Decisions` naming %[1]s and the difference; "+
				"bundle, the two planned as one bundle (`abcd intent plan %[2]s %[1]s --bundle <name>` while both are drafts, "+
				"else `abcd intent reclassify <itd-N> --kind bundle-member --bundle <name>`); "+
				"supersede, `superseded_by` on the record that gives way (`abcd intent reclassify <itd-N> --kind superseded --by <itd-M> --reason \"<why>\"`); "+
				"refine, a decision in the draft's `## Decisions` naming what it refines in %[1]s.\n\n", sib, in.Intent)
		default:
			fmt.Fprintf(&b, "### Q%d. A question (unanchored)\n\n", n)
			fmt.Fprintf(&b, "%s\n\n", prepassLine(q.question))
			if q.why != "" {
				fmt.Fprintf(&b, "Not anchored: %s, so it is asked, not asserted.\n\n", prepassLine(q.why))
			}
			b.WriteString("Lands as: a decision in the draft's `## Decisions`, or nothing when the answer is that the concern does not hold.\n\n")
		}
	}

	b.WriteString("## Blocks planning\n\n")
	if len(f.BlocksPlanning) == 0 {
		b.WriteString("_The pre-pass flagged nothing that blocks planning._\n")
	} else {
		for _, x := range f.BlocksPlanning {
			fmt.Fprintf(&b, "- %s\n", prepassLine(x))
		}
	}
	if len(in.Warnings) > 0 {
		b.WriteString("\n## Warnings\n\n")
		for _, w := range in.Warnings {
			fmt.Fprintf(&b, "- %s\n", w)
		}
	}
	return b.String()
}
