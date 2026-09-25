package launch

// gates.go — itd-65's pre-flight gate suite (spc-2609201955279019).
//
// The suite is the part of a launch's refusals that the secret scan, the
// lockstep check and the installability smoke do not already make: marker-block
// sanity and change narration over the shipped Markdown, the dirty-tree refusal,
// and two warn-tier rows (the documentation audit and hook compliance). It runs
// ALL of its gates and collects ALL of their findings before anything decides
// (run-all-collect-all), and one runner serves the preview (DryRun), the core
// ship (Ship) and the render path (PrecheckPayload), so a preview and a cut
// refuse on the same findings.

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/intentdriven/abcd/internal/gitutil"
)

// Gate tiers: what a finding in the row does to the release.
const (
	// TierHardFail — a finding refuses the release.
	TierHardFail = "hard-fail"
	// TierWarn — a finding is surfaced and refuses nothing, unless the
	// repository configures the suite strict.
	TierWarn = "warn"
)

// Gate names the suite registers, in the order it reports them.
const (
	gateMarkerBlock   = "marker-block"
	gateNarration     = "change-narration"
	gateDirtyTree     = "dirty-tree"
	gateDocAuditor    = "documentation-auditor"
	gateHookCompliant = "hook-compliance"
)

// GateFinding is one concern a gate raised, located where it can be fixed.
type GateFinding struct {
	File   string `json:"file,omitempty"`
	Line   int    `json:"line,omitempty"`
	Detail string `json:"detail"`
}

// String renders the finding as file:line: detail, dropping the parts it lacks.
func (f GateFinding) String() string {
	loc := f.File
	if loc != "" && f.Line > 0 {
		loc += ":" + strconv.Itoa(f.Line)
	}
	if loc == "" {
		return f.Detail
	}
	return loc + ": " + f.Detail
}

// ErrPayloadGateRefused reports that the payload failed one of the suite's
// content gates (marker-block sanity, change narration), or a warn-tier gate in
// a repository that configured the suite strict.
var ErrPayloadGateRefused = errors.New("the payload failed a launch pre-flight gate")

// ErrDirtyTree reports that the working tree carries uncommitted changes (or
// that its state could not be read) and the cut was not told to allow them.
var ErrDirtyTree = errors.New("the working tree is not clean")

// DirtyPolicy is how the dirty-tree gate treats uncommitted changes.
type DirtyPolicy int

const (
	// DirtyRefuse is the zero value, so a caller that says nothing fails
	// closed: an uncommitted change, or a tree whose state cannot be read,
	// refuses.
	DirtyRefuse DirtyPolicy = iota
	// DirtyAllow is --allow-dirty: the uncommitted changes are carried, and the
	// pre-flight report records the override and every path it carried. A tree
	// whose state cannot be read still refuses — the override allows dirt, not
	// blindness.
	DirtyAllow
	// DirtySkip leaves the gate out. It exists for the render a cut runs AFTER
	// its own writes (the dated CHANGELOG heading, the release page, the
	// archive pin): those writes are the cut's expected output, not dirt, and
	// the gate already ran before any of them, at the cut's start. The ordering
	// is the point — the cut's own staged changes must never read as dirt. A
	// render caller states it explicitly (PayloadRenderRequest.Dirty); nothing
	// defaults to it.
	DirtySkip
)

// DocAuditPreflight is the documentation audit's result, MEASURED by the
// caller: it needs the docs-lint engine in internal/core/lint, which imports
// this package, so the front door that holds both hands the result in as data
// (the CitationPreflight shape). Nil means the repository has not armed a
// docs-lint configuration.
type DocAuditPreflight struct {
	// Findings are the docs-lint findings over the configured doc roots.
	Findings []GateFinding `json:"findings,omitempty"`
	// Unreadable says why the audit could not be measured at all.
	Unreadable string `json:"unreadable,omitempty"`
	// NotMeasured says why the caller did not measure the audit on this
	// path. The row then reports "not_measured" and makes no claim about the
	// repository's configuration.
	NotMeasured string `json:"not_measured,omitempty"`
}

// GatePolicy is the repository's configuration of the suite, read from the
// launch-payload config beside the include list.
type GatePolicy struct {
	// StrictWarnings makes a warn-tier finding refuse the release.
	StrictWarnings bool `json:"strict_warnings"`
}

// LoadGatePolicy reads the suite's configuration from the launch-payload config.
// An absent key is the default (warnings do not block); a key that is present
// and not a boolean is a PreflightError, like any other malformed launch config.
func LoadGatePolicy(repoRoot string) (GatePolicy, error) {
	var policy GatePolicy
	raw, err := readIncludeConfig(repoRoot)
	if err != nil {
		return policy, err
	}
	v, ok := raw["strict_warnings"]
	if !ok {
		return policy, nil
	}
	if err := json.Unmarshal(v, &policy.StrictWarnings); err != nil {
		return policy, preflight("include config 'strict_warnings' is not a boolean: %s", includeConfigRelPath)
	}
	return policy, nil
}

// suiteRequest is one run of the suite over one resolved bundle.
type suiteRequest struct {
	RepoRoot string
	Bundle   Bundle
	Dirty    DirtyPolicy
	DocAudit *DocAuditPreflight
	Policy   GatePolicy
}

// suiteResult is everything the suite found, collected before anything decides.
type suiteResult struct {
	Gates    []GateSummary
	Refusals []string
	Warnings []string
	// Dirty is every uncommitted path the dirty-tree gate saw.
	Dirty []string
	// errs carries each refusal as a typed error, so the render path can be
	// matched with errors.Is on the gate that refused.
	errs []error
}

// runGateSuite runs every gate in the suite and collects every finding.
func runGateSuite(req suiteRequest) suiteResult {
	var res suiteResult
	hard := func(name, detail string, findings []GateFinding, sentinel error) {
		res.Gates = append(res.Gates, GateSummary{
			Name: name, Status: "ran", Tier: TierHardFail, Detail: detail, Findings: findings,
		})
		for _, f := range findings {
			reason := name + ": " + f.String()
			res.Refusals = append(res.Refusals, reason)
			res.errs = append(res.errs, fmt.Errorf("%w: %s", sentinel, reason))
		}
	}
	warn := func(row GateSummary) {
		row.Tier = TierWarn
		res.Gates = append(res.Gates, row)
		for _, f := range row.Findings {
			reason := row.Name + ": " + f.String()
			res.Warnings = append(res.Warnings, reason)
			if req.Policy.StrictWarnings {
				strict := "strict " + reason
				res.Refusals = append(res.Refusals, strict)
				res.errs = append(res.errs, fmt.Errorf("%w: %s", ErrPayloadGateRefused, strict))
			}
		}
	}

	markers := markerBlockFindings(req.Bundle)
	hard(gateMarkerBlock, countDetail(markers, "malformed marker block"), markers, ErrPayloadGateRefused)

	narration := narrationFindings(req.Bundle)
	hard(gateNarration, countDetail(narration, "change-narration sentence"), narration, ErrPayloadGateRefused)

	if req.Dirty != DirtySkip {
		row, dirty, refusal := dirtyTreeGate(req.RepoRoot, req.Dirty)
		res.Gates = append(res.Gates, row)
		res.Dirty = dirty
		if refusal != "" {
			reason := gateDirtyTree + ": " + refusal
			res.Refusals = append(res.Refusals, reason)
			res.errs = append(res.errs, fmt.Errorf("%w: %s", ErrDirtyTree, reason))
		}
	}

	warn(docAuditGate(req.DocAudit))
	warn(hookComplianceGate(req.Bundle))
	return res
}

// countDetail is a hard-fail row's one-line measurement.
func countDetail(findings []GateFinding, noun string) string {
	return strconv.Itoa(len(findings)) + " " + noun + "(s)"
}

// The managed marker pair abcd writes around its block in an agent
// instructions file. The spelling is the one internal/core/ahoy installs and
// strips; ahoy imports this package, so the pair is declared here and a test
// proves ahoy recognises exactly this spelling (gates_marker_drift_test.go).
const (
	MarkerBlockBegin = "<!-- BEGIN ABCD -->"
	MarkerBlockEnd   = "<!-- END ABCD -->"
)

// isMarkdown reports whether a payload path is a Markdown file.
func isMarkdown(rel string) bool {
	return strings.EqualFold(path.Ext(rel), ".md")
}

// fenceRe opens or closes a fenced code block (up to three spaces of indent).
var fenceRe = regexp.MustCompile("^ {0,3}(```|~~~)")

// inlineCodeRe is an inline code span. Quoting a construct in code is the one
// sanctioned way to mention it in prose without asserting it.
var inlineCodeRe = regexp.MustCompile("`[^`]*`")

// proseLine is one Markdown line outside fenced code, inline code removed.
type proseLine struct {
	n    int // 1-based
	text string
}

// proseLines splits a Markdown document into its prose lines: fenced code
// blocks are dropped and inline code spans blanked, so a construct quoted as an
// example never reads as an assertion. A leading YAML frontmatter block is
// dropped too, since it is metadata rather than a body — but only a block that
// closes: a document that opens with a "---" rule and never repeats it has no
// frontmatter, and is read whole rather than dropped (iss-2609251827296447).
func proseLines(data []byte) []proseLine {
	lines := strings.Split(strings.ReplaceAll(string(data), "\r\n", "\n"), "\n")
	var out []proseLine
	inFence := false
	frontEnd := -1 // index of the closing "---"; -1 when there is no frontmatter
	if len(lines) > 0 && strings.TrimSpace(lines[0]) == "---" {
		for i := 1; i < len(lines); i++ {
			if strings.TrimSpace(lines[i]) == "---" {
				frontEnd = i
				break
			}
		}
	}
	for i, line := range lines {
		if i <= frontEnd {
			continue
		}
		if fenceRe.MatchString(line) {
			inFence = !inFence
			continue
		}
		if inFence {
			continue
		}
		out = append(out, proseLine{n: i + 1, text: inlineCodeRe.ReplaceAllString(line, "")})
	}
	return out
}

// markerBlockFindings checks every shipped Markdown file's marker blocks: a
// BEGIN never closed, an END with no open block, and a BEGIN inside an open
// block (a nested pair) are each a finding naming the file and line. Two
// sequential balanced blocks are well formed; a marker quoted in code is not a
// marker. A file the gate cannot read is a finding too — the gate fails closed.
func markerBlockFindings(bundle Bundle) []GateFinding {
	var out []GateFinding
	for _, f := range bundle.Included {
		if !isMarkdown(f.LogicalPath) {
			continue
		}
		data, err := os.ReadFile(f.ResolvedPath)
		if err != nil {
			out = append(out, GateFinding{File: f.LogicalPath, Detail: "could not be read to check its marker blocks: " + err.Error()})
			continue
		}
		open := 0 // line of the open BEGIN; 0 when no block is open
		for _, line := range proseLines(data) {
			rest := line.text
			for {
				b := strings.Index(rest, MarkerBlockBegin)
				e := strings.Index(rest, MarkerBlockEnd)
				if b < 0 && e < 0 {
					break
				}
				if b >= 0 && (e < 0 || b < e) {
					if open > 0 {
						out = append(out, GateFinding{File: f.LogicalPath, Line: line.n,
							Detail: "nested " + MarkerBlockBegin + " inside the block opened at line " + strconv.Itoa(open)})
					} else {
						open = line.n
					}
					rest = rest[b+len(MarkerBlockBegin):]
					continue
				}
				if open == 0 {
					out = append(out, GateFinding{File: f.LogicalPath, Line: line.n,
						Detail: MarkerBlockEnd + " with no open " + MarkerBlockBegin})
				}
				open = 0
				rest = rest[e+len(MarkerBlockEnd):]
			}
		}
		if open > 0 {
			out = append(out, GateFinding{File: f.LogicalPath, Line: open,
				Detail: MarkerBlockBegin + " is never closed by " + MarkerBlockEnd})
		}
	}
	return out
}

// releaseRecordFiles are the release records a payload may carry. A changelog
// and a release page narrate change by definition — that is what they are for —
// so the change-narration gate never reads them.
var releaseRecordFiles = map[string]struct{}{"changelog.md": {}, "release.md": {}}

// isDocBody reports whether a payload path is a shipped doc body the
// change-narration gate reads: Markdown under docs/, or at the payload root.
// Plugin surface (commands/, agents/, skills/) is instruction to an agent, not
// documentation of the product, and the release records are exempt.
func isDocBody(rel string) bool {
	if !isMarkdown(rel) {
		return false
	}
	if _, ok := releaseRecordFiles[strings.ToLower(path.Base(rel))]; ok {
		return false
	}
	return !strings.Contains(rel, "/") || strings.HasPrefix(rel, "docs/")
}

// narrationEscapeRe is the docs-lint escape. A line carrying it is exempt here
// as it is from docs lint's change-narration tokens: one marker, one meaning.
var narrationEscapeRe = regexp.MustCompile(`(?i)<!--\s*docs-lint:\s*allow\b`)

// narrationConstructs are the deterministic constructs that narrate a change,
// each judged within ONE sentence. A construct that also reads as present
// state carries a check that only its narrating reading passes
// (iss-2609251827286563): "used to" only as the past habit, "no longer" and
// "renamed" only beside a change subject naming abcd or its behaviour, and
// "previously"/"now" only beside a change verb. Bare "now" and bare
// "previously" are not constructs at all (itd-65 AC10); the present-tense
// warning docs lint carries is where they belong.
var narrationConstructs = []struct {
	name  string
	re    *regexp.Regexp
	holds func(sentence string, at []int) bool // nil: the match alone narrates
}{
	{"changed from", regexp.MustCompile(`(?i)\bchanged\s+from\b.*\bto\b`), nil},
	{"no longer", regexp.MustCompile(`(?i)\bno\s+longer\b`), changeSubjectBefore},
	{"migrated from", regexp.MustCompile(`(?i)\bmigrated\s+from\b`), nil},
	{"renamed", regexp.MustCompile(`(?i)\brenamed\b.*\bto\b`), changeSubjectBefore},
	{"previously … now", regexp.MustCompile(`(?i)\bpreviously\b.*\bnow\b|\bnow\b.*\bpreviously\b`), changeVerbBesideEither},
	{"used to", regexp.MustCompile(`(?i)\bused\s+to\b`), pastHabitUsedTo},
}

// narrationEscapeHint names the escape in every narration finding, so a
// sentence that states present state and still trips a construct is one
// marker away from passing, never a rewrite of correct prose.
const narrationEscapeHint = "docs describe present state and the changelog records the change; a sentence that states present state takes <!-- docs-lint: allow --> on its line"

// narrationWordRe is one word of a sentence, or one clause-breaking mark.
var narrationWordRe = regexp.MustCompile(`[\p{L}\p{N}][\p{L}\p{N}'’_.-]*|[,;:()—–"“”]`)

// clauseBreaks end a clause: punctuation, and the words that open a relative
// or subordinate clause, whose subject is not the main clause's.
var clauseBreaks = wordSet(",", ";", ":", "(", ")", "—", "–", "\"", "“", "”",
	"that", "which", "who", "whose", "where", "when", "whenever", "if", "because", "while", "what", "and", "but", "or")

// changeSubjects are the nouns that name abcd or its behaviour. A "no longer"
// or a "renamed" whose clause subject is one of them narrates a change to the
// product; any other subject ("files that are no longer present", "the output
// is renamed") states the present state of something the product handles.
var changeSubjects = wordSet("abcd", "we", "tool", "tools", "verb", "verbs", "command", "commands",
	"subcommand", "subcommands", "flag", "flags", "option", "options", "binary", "plugin", "cli",
	"hook", "hooks", "gate", "gates", "default", "defaults", "behaviour", "behavior", "api",
	"endpoint", "endpoints", "setting", "settings", "feature", "features")

// changeVerbs are the past-tense verbs that make "previously … now" a
// narration ("reports previously went to the log", "the default was
// previously JSON"). A participle used as an adjective ("as previously
// noted", "the previously saved query") is none of them.
var changeVerbs = wordSet("was", "were", "had", "used", "went", "came", "changed", "became", "moved",
	"switched", "replaced", "migrated", "renamed", "removed", "dropped", "defaulted", "lived",
	"wrote", "printed", "returned", "required", "accepted", "stored", "sent", "ran", "pointed")

// subjectPronouns before "used to" make it the finite verb of its clause: the
// past habit ("it used to print", "the classes that used to drift").
var subjectPronouns = wordSet("i", "it", "they", "we", "you", "he", "she", "who", "which", "that", "abcd")

// determiners open a noun phrase that can be a clause's subject.
var determiners = wordSet("the", "a", "an", "this", "that", "these", "those", "its", "their", "our", "his", "her", "each", "every", "some")

// finiteVerbs after a "used to" phrase show the phrase was a participle
// modifying a noun ("the token used to authenticate the request IS read"),
// not the clause's own verb.
var finiteVerbs = wordSet("is", "are", "was", "were", "has", "have", "had", "does", "do", "did", "can", "cannot",
	"could", "must", "will", "would", "should", "may", "might", "shall", "stays", "remains")

func wordSet(words ...string) map[string]struct{} {
	m := make(map[string]struct{}, len(words))
	for _, w := range words {
		m[w] = struct{}{}
	}
	return m
}

// narrationWords splits text into lower-cased words and clause marks.
func narrationWords(text string) []string {
	out := narrationWordRe.FindAllString(strings.ToLower(text), -1)
	for i, w := range out {
		out[i] = strings.TrimRight(w, ".")
	}
	return out
}

// clauseBefore is the words of the clause that runs up to byte offset at.
func clauseBefore(sentence string, at int) []string {
	words := narrationWords(sentence[:at])
	for i := len(words) - 1; i >= 0; i-- {
		if _, stop := clauseBreaks[words[i]]; stop {
			return words[i+1:]
		}
	}
	return words
}

// clauseAfter is the words of the clause that runs on from byte offset at.
func clauseAfter(sentence string, at int) []string {
	words := narrationWords(sentence[at:])
	for i, w := range words {
		if _, stop := clauseBreaks[w]; stop {
			return words[:i]
		}
	}
	return words
}

// anyIn reports whether some word is in the set.
func anyIn(words []string, set map[string]struct{}) bool {
	for _, w := range words {
		if _, ok := set[w]; ok {
			return true
		}
	}
	return false
}

// changeSubjectBefore reports whether the clause leading up to the match names
// abcd or its behaviour.
func changeSubjectBefore(sentence string, at []int) bool {
	return anyIn(clauseBefore(sentence, at[0]), changeSubjects)
}

// changeVerbBesideEither reports whether a change verb stands within three
// words of some "previously" or "now" in the sentence. Beside a "previously"
// that modifies a verb, any past-tense "-ed" verb within two words counts too
// ("it previously lacked", "the rule previously existed"); a "previously"
// after "as" or a determiner qualifies a participle ("as previously noted",
// "the previously saved query") and brings no verb of its own.
func changeVerbBesideEither(sentence string, _ []int) bool {
	words := narrationWords(sentence)
	for i, w := range words {
		if w != "previously" && w != "now" {
			continue
		}
		for j := max(0, i-3); j <= min(len(words)-1, i+3); j++ {
			if _, ok := changeVerbs[words[j]]; ok {
				return true
			}
		}
		if w != "previously" || (i > 0 && (words[i-1] == "as" || isDeterminer(words[i-1]))) {
			continue
		}
		for j := max(0, i-2); j <= min(len(words)-1, i+2); j++ {
			if j != i && len(words[j]) > 3 && strings.HasSuffix(words[j], "ed") {
				return true
			}
		}
	}
	return false
}

// isDeterminer reports whether a word opens a noun phrase.
func isDeterminer(w string) bool {
	_, ok := determiners[w]
	return ok
}

// pastHabitUsedTo reports whether a "used to" is the past-habit construction:
// the finite verb of its clause, after a subject pronoun ("it used to print")
// or after a bare subject noun phrase that opens the clause ("the tool used to
// print") when no finite verb follows in the clause. A passive or adjectival
// "used to" ("is used to sign", "gets used to") and a participle modifying a
// noun ("the token used to authenticate the request is read", "set the token
// used to authenticate") are present state.
func pastHabitUsedTo(sentence string, at []int) bool {
	// The word before "used" is read across a clause break: a relative
	// pronoun ("the classes that used to drift") is the clause's subject.
	preceding := narrationWords(sentence[:at[0]])
	if len(preceding) == 0 {
		return false
	}
	last := preceding[len(preceding)-1]
	if _, aux := passiveAuxiliaries[last]; aux {
		return false
	}
	if _, ok := subjectPronouns[last]; ok {
		return true
	}
	before := clauseBefore(sentence, at[0])
	if len(before) == 0 {
		return false
	}
	det := isDeterminer(before[0])
	subject := len(before) == 1 || (det && len(before) <= 4)
	if !subject {
		return false
	}
	// The phrase's own complement runs until a new subject pronoun opens a
	// clause of its own ("used to die on the laptop they were generated on").
	after := clauseAfter(sentence, at[1])
	for i, w := range after {
		if _, pronoun := subjectPronouns[w]; pronoun {
			after = after[:i]
			break
		}
	}
	return !anyIn(after, finiteVerbs)
}

// passiveAuxiliaries before "used to" make it a passive or an adjective ("is
// used to sign", "gets used to"): present-state prose, not a narrated habit.
var passiveAuxiliaries = wordSet("is", "are", "was", "were", "be", "been", "being", "get", "gets", "got", "getting")

// narrationFindings scans the shipped doc bodies for a sentence narrating a
// change, and names each one with its file, line and text.
func narrationFindings(bundle Bundle) []GateFinding {
	var out []GateFinding
	for _, f := range bundle.Included {
		if !isDocBody(f.LogicalPath) {
			continue
		}
		data, err := os.ReadFile(f.ResolvedPath)
		if err != nil {
			out = append(out, GateFinding{File: f.LogicalPath, Detail: "could not be read to check it for change narration: " + err.Error()})
			continue
		}
		for _, s := range docSentences(proseLines(data)) {
			if name := narrationConstruct(s.text); name != "" {
				out = append(out, GateFinding{File: f.LogicalPath, Line: s.line,
					Detail: "narrates a change (" + name + "): \"" + clip(s.text, 200) + "\" — " + narrationEscapeHint})
			}
		}
	}
	return out
}

// narrationConstruct returns the first construct a sentence carries, or "".
func narrationConstruct(sentence string) string {
	for _, c := range narrationConstructs {
		for _, at := range c.re.FindAllStringIndex(sentence, -1) {
			if c.holds == nil || c.holds(sentence, at) {
				return c.name
			}
		}
	}
	return ""
}

// sentenceEndRe ends a sentence: terminal punctuation followed by space.
var sentenceEndRe = regexp.MustCompile(`[.!?]["')\]]*\s+`)

// listItemRe starts a list item, which starts a new sentence whatever precedes it.
var listItemRe = regexp.MustCompile(`^\s*([-*+]|\d+[.)])\s`)

// docSentence is one sentence of prose, located at the line it starts on.
type docSentence struct {
	line int
	text string
}

// docSentences groups prose lines into sentences. A blank line, a heading, a
// table row or a list item starts a new run of text; within a run, sentences
// split at terminal punctuation. A line carrying the docs-lint escape is
// dropped whole.
func docSentences(lines []proseLine) []docSentence {
	var out []docSentence
	var buf strings.Builder
	// starts[i] is the offset in buf where the text of line lineAt[i] begins.
	var starts, lineAt []int
	flush := func() {
		text := buf.String()
		pos := 0
		for _, loc := range append(sentenceEndRe.FindAllStringIndex(text, -1), []int{len(text), len(text)}) {
			sentence := strings.TrimSpace(text[pos:loc[1]])
			if sentence != "" {
				out = append(out, docSentence{line: lineOf(pos, starts, lineAt), text: sentence})
			}
			pos = loc[1]
		}
		buf.Reset()
		starts, lineAt = starts[:0], lineAt[:0]
	}
	for _, l := range lines {
		t := strings.TrimSpace(l.text)
		if narrationEscapeRe.MatchString(l.text) {
			flush()
			continue
		}
		if t == "" || strings.HasPrefix(t, "#") || strings.HasPrefix(t, "|") || listItemRe.MatchString(l.text) {
			flush()
		}
		if t == "" {
			continue
		}
		if buf.Len() > 0 {
			buf.WriteByte(' ')
		}
		starts = append(starts, buf.Len())
		lineAt = append(lineAt, l.n)
		buf.WriteString(t)
		if strings.HasPrefix(t, "#") || strings.HasPrefix(t, "|") {
			flush()
		}
	}
	flush()
	return out
}

// lineOf maps a byte offset in a run back to the line it came from: the last
// line whose text starts at or before it.
func lineOf(pos int, starts, lineAt []int) int {
	line := 0
	for i, s := range starts {
		if s > pos {
			break
		}
		line = lineAt[i]
	}
	return line
}

// clip shortens s to at most n runes, marking the cut.
func clip(s string, n int) string {
	if utf8.RuneCountInString(s) <= n {
		return s
	}
	r := []rune(s)
	return string(r[:n]) + "…"
}

// localTierPrefix is the gitignored local tier. Nothing in it is ever committed,
// and the pre-flight report itself lands there, so it is never dirt — even in a
// checkout that forgot to ignore it.
const localTierPrefix = ".abcd/.work.local/"

// DirtyTreeFiles lists the working tree's uncommitted paths, repo-relative and
// sorted: every tracked change against HEAD, staged or not, and every untracked
// file git does not ignore. The local tier is never listed. A tree whose state
// git cannot read (no repository, no HEAD) is an error — never an empty list,
// which would read as clean.
func DirtyTreeFiles(repoRoot string) ([]string, error) {
	// Resolve HEAD first: outside a repository `git diff` falls back to its
	// no-index mode and answers with a usage page, not a reason.
	if _, err := gitutil.Run(repoRoot, "rev-parse", "--verify", "HEAD^{commit}"); err != nil {
		return nil, fmt.Errorf("no commit to compare the working tree against: %w", err)
	}
	changed, err := gitutil.Run(repoRoot, "diff", "--name-only", "-z", "HEAD")
	if err != nil {
		return nil, fmt.Errorf("the working tree's changes could not be read: %w", err)
	}
	untracked, err := gitutil.Run(repoRoot, "ls-files", "--others", "--exclude-standard", "-z")
	if err != nil {
		return nil, fmt.Errorf("the working tree's untracked files could not be read: %w", err)
	}
	dirty := map[string]struct{}{}
	for _, list := range []string{changed, untracked} {
		for _, p := range strings.Split(list, "\x00") {
			if p != "" && !strings.HasPrefix(p, localTierPrefix) {
				dirty[p] = struct{}{}
			}
		}
	}
	return sortedKeys(dirty), nil
}

// dirtyTreeGate is the dirty-tree row, the uncommitted paths it saw, and the
// refusal it makes under policy ("" when it makes none).
func dirtyTreeGate(repoRoot string, policy DirtyPolicy) (GateSummary, []string, string) {
	row := GateSummary{Name: gateDirtyTree, Status: "ran", Tier: TierHardFail}
	dirty, err := DirtyTreeFiles(repoRoot)
	if err != nil {
		row.Detail = "could not be read"
		refusal := "the working tree's state could not be read, so it cannot be shown clean: " + err.Error()
		row.Findings = []GateFinding{{Detail: refusal}}
		return row, nil, refusal
	}
	if len(dirty) == 0 {
		row.Detail = "clean"
		return row, nil, ""
	}
	names := dirty
	if len(names) > 10 {
		names = append(append([]string{}, names[:10]...), "and "+strconv.Itoa(len(dirty)-10)+" more")
	}
	if policy == DirtyAllow {
		row.Detail = strconv.Itoa(len(dirty)) + " uncommitted change(s), allowed by --allow-dirty and recorded in the pre-flight report"
		return row, dirty, ""
	}
	for _, p := range dirty {
		row.Findings = append(row.Findings, GateFinding{File: p, Detail: "uncommitted"})
	}
	row.Detail = strconv.Itoa(len(dirty)) + " uncommitted change(s)"
	return row, dirty, strconv.Itoa(len(dirty)) + " uncommitted change(s) in the working tree (" +
		strings.Join(names, ", ") + ") — a release is cut from a commit; commit or discard them, or pass --allow-dirty to cut from this tree anyway"
}

// docAuditGate is the documentation-auditor row: the docs-lint engine's
// findings over the repository's configured doc roots, surfaced at the warn
// tier. The engine is run by the front door and handed in (DocAuditPreflight).
func docAuditGate(pre *DocAuditPreflight) GateSummary {
	row := GateSummary{Name: gateDocAuditor, Status: "ran"}
	switch {
	case pre == nil:
		row.Status = "not_armed"
		row.Detail = "no .abcd/docs-lint.json: the documentation audit (the docs-lint engine over the configured doc roots) has nothing to run"
	case pre.NotMeasured != "":
		row.Status = "not_measured"
		row.Detail = pre.NotMeasured
	case pre.Unreadable != "":
		row.Detail = "could not be measured"
		row.Findings = []GateFinding{{Detail: "the documentation audit could not be measured: " + pre.Unreadable}}
	default:
		row.Findings = pre.Findings
		row.Detail = strconv.Itoa(len(pre.Findings)) + " docs-lint finding(s)"
	}
	return row
}

// hookHandlerTypes are the hook handler types a host runs without a command.
// A handler of another type — "command", or one this gate does not know — is
// expected to carry a non-empty command.
var hookHandlerTypes = map[string]struct{}{"prompt": {}, "agent": {}, "http": {}}

// hookComplianceGate is the hook-compliance row, at the warn tier: every
// payload file a hook command invokes is executable in the payload (a hook that
// tests for an executable silently skips one that is not), every command
// handler names a command, and every timeout is a positive number. What a hook
// DECLARES is the installability smoke's (a missing file refuses there); this
// row judges whether the declared hooks would behave.
func hookComplianceGate(bundle Bundle) GateSummary {
	row := GateSummary{Name: gateHookCompliant, Status: "ran"}
	tree := NewBundleTree(bundle)
	surface, err := ResolveInstallSurface(tree)
	if err != nil {
		row.Findings = []GateFinding{{Detail: "the hook declarations could not be read: " + err.Error()}}
		row.Detail = "could not be read"
		return row
	}
	modes := make(map[string]string, len(bundle.Included))
	for _, f := range bundle.Included {
		modes[f.LogicalPath] = f.GitMode
	}
	seen := map[string]struct{}{}
	for _, e := range surface.Entries {
		if e.Origin != OriginHookCommand || e.Requirement != RequirePayload {
			continue
		}
		mode, carried := modes[e.Path]
		if _, dup := seen[e.Path]; dup || !carried || mode == "100755" {
			continue
		}
		seen[e.Path] = struct{}{}
		row.Findings = append(row.Findings, GateFinding{File: e.Path,
			Detail: "is invoked by a hook command but is not executable in the payload (mode " + mode + "), so the hook cannot run it"})
	}
	for _, e := range surface.Entries {
		if !isHookConfig(e) || !tree.Has(e.Path) {
			continue
		}
		// The resolver above already refuses a config that does not read or
		// parse; the row still fails closed rather than trusting that.
		doc, err := readHookConfig(tree, e.Path)
		if err != nil {
			row.Findings = append(row.Findings, GateFinding{File: e.Path, Detail: err.Error()})
			continue
		}
		row.Findings = append(row.Findings, hookHandlerFindings(e.Path, doc)...)
	}
	row.Detail = strconv.Itoa(len(row.Findings)) + " concern(s)"
	return row
}

// hookHandlerFindings walks a decoded hooks document for handler objects — any
// object carrying a "type" — and judges each one.
func hookHandlerFindings(file string, doc any) []GateFinding {
	var out []GateFinding
	var walk func(any)
	walk = func(v any) {
		switch t := v.(type) {
		case map[string]any:
			if typ, ok := t["type"].(string); ok {
				if _, commandless := hookHandlerTypes[typ]; !commandless {
					if cmd, _ := t["command"].(string); strings.TrimSpace(cmd) == "" {
						out = append(out, GateFinding{File: file, Detail: "a " + strconv.Quote(typ) + " hook handler names no command"})
					}
				}
				if raw, present := t["timeout"]; present {
					if n, ok := raw.(float64); !ok || n <= 0 {
						out = append(out, GateFinding{File: file, Detail: "a hook handler's timeout is not a positive number of seconds"})
					}
				}
			}
			keys := make([]string, 0, len(t))
			for k := range t {
				keys = append(keys, k)
			}
			sort.Strings(keys)
			for _, k := range keys {
				walk(t[k])
			}
		case []any:
			for _, item := range t {
				walk(item)
			}
		}
	}
	walk(doc)
	return out
}

// sortedKeys returns a set's members in order.
func sortedKeys(m map[string]struct{}) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
