package lint_test

// The scribe's access rule (spc-66) is the assembler's exact inverse, and brief
// invariant 15 states both halves: a reading receives a positively included slice
// of the shipped repository and no ledger; the scribe receives ledger content and
// never the shipped repository as an object of judgment, and it is not a
// consumer of the session-transcript store.
//
// These cases hold the shipped DEFINITION to that rule, and they are honest about
// their reach: they prove the prompt names the right paths, not that a host
// assembled the right context. Mechanical assembly belongs to the ingest verb.
//
// They sit in the external test package beside preflightgates_test.go, the other
// case that reads the real repository's shipped files, and share its readRepoFile.

import (
	"encoding/json"
	"fmt"
	"html"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"unicode"

	"github.com/intentdriven/abcd/internal/core/lint"
)

// scribePromptRel is the shipped definition; scribeCanaryRel its injection canary.
const (
	scribePromptRel = "agents/scribe.md"
	scribeCanaryRel = "agents/scribe/fixtures/injection-canary.json"
)

// scribeLedgerRoot is the ONE tree the definition may name — the same root the
// definition, the brief's protocol and the agent changelog all state. A broader
// root admits `.abcd/work/DECISIONS.md`, which is shared working material and not
// the ledger.
const scribeLedgerRoot = ".abcd/work/issues/"

// scribeAgentContractRule is the shipped rule id, spelled as `.abcd/record-lint.json`
// spells it. TestScribePromptSatisfiesTheContract arms it against a deliberately
// broken tree, so a rename cannot leave the case passing vacuously.
const scribeAgentContractRule = "agent_contract"

// scribeProseIdioms is the ONLY exemption from "a separator-bearing token is a
// path", and it is an explicit list because the shape rule it replaces was a
// description of `and/or` that also described `internal/core`, `docs/` and
// `internal/README`. Everything else is judged as a path — a version pair such as
// `0.1.0/0.2.0` and a URL included. That is by design: the definition has no
// reason to carry either, and an exemption wide enough to admit them is wide
// enough to admit a shipped-tree path spelled to look like one.
var scribeProseIdioms = map[string]bool{
	"and/or": true, "read/write": true, "either/or": true, "i/o": true,
}

// scribeAllowedNonASCII is the typographic punctuation the definition actually
// uses. Every OTHER code point outside ASCII is refused rather than folded, which
// is what lets this file carry no lookalike table: enumerating separator
// lookalikes is a losing game — the first cut listed the forward-solidus family
// and missed its reverse twins, and the next list would miss something else —
// whereas "outside ASCII, outside this list, refused" closes the class, homoglyph
// segments and invisible code points included. A definition that needs another
// mark adds it here deliberately.
var scribeAllowedNonASCII = map[rune]bool{
	'—': true, // em dash
}

// scribeMaxDecodeRounds bounds the decode fixpoint. Decoding once is not enough
// (`%252F` and `&amp;#x2F;` each survive one round), and decoding unbounded is a
// loop a hostile file chooses the length of.
const scribeMaxDecodeRounds = 8

// scribePercentRe matches one percent-encoded octet. Decoding is done with this
// rather than with url.PathUnescape because PathUnescape fails the WHOLE string on
// the first invalid escape, so one stray `%` in prose silently switched percent
// decoding off for the entire file.
var scribePercentRe = regexp.MustCompile(`%[0-9A-Fa-f]{2}`)

// scribeResidualEncodingRe matches an encoding shape that survived the fixpoint —
// a truncated octet, or an entity that decodes to nothing. The definition has no
// reason to carry either, so a residual is refused rather than puzzled over. It
// also refuses a bare ampersand followed by a letter, which costs `Q&A` and `AT&T`
// as prose: that is deliberate. An entity is exactly an ampersand followed by
// letters, so a pattern that spared the innocent spelling would spare the hostile
// one too, and prose has the easier repair — write "and".
var scribeResidualEncodingRe = regexp.MustCompile(`%[0-9A-Fa-f]|&[A-Za-z#][A-Za-z0-9]{0,30};?`)

// scribeSpacedSeparatorRe collapses horizontal whitespace around a separator, so
// `a / b` reads as `a/b`. Newlines are deliberately not collapsed: joining across
// a line break would invent paths out of adjacent sentences.
var scribeSpacedSeparatorRe = regexp.MustCompile(`[ \t]*/[ \t]*`)

// scribeDecode runs percent and HTML entity decoding to a bounded fixpoint,
// tolerantly: an invalid escape is left as it stands rather than abandoning the
// rest of the text.
func scribeDecode(text string) string {
	for i := 0; i < scribeMaxDecodeRounds; i++ {
		next := html.UnescapeString(scribePercentRe.ReplaceAllStringFunc(text, func(octet string) string {
			v, err := strconv.ParseUint(octet[1:], 16, 8)
			if err != nil {
				return octet
			}
			return string(rune(v))
		}))
		if next == text {
			break
		}
		text = next
	}
	return text
}

// scribeFold normalizes a definition into the TWO views the path check reads. It
// is short on purpose, and this comment states exactly what it does and nothing
// more: decode to a fixpoint, fold the ASCII reverse solidus, and — for the
// second view only — collapse horizontal whitespace around a separator. There is
// no lookalike folding and no normalization form, because scribeAccessFindings
// refuses every non-ASCII code point outside the small list above; a spelling
// that cannot appear needs no fold.
//
// Two views because the whitespace collapse is a JOIN as well as a fold. It has
// to exist: `internal / core / lint` is one path written with spaces in it. But
// applied after a trailing separator it glues the NEXT token on, so
// `.abcd/work/issues/ internal/core/lint/agentcontract.go` collapses into one
// path that sits under the allowed root and passes the prefix check — the
// shipped-tree path disappears into the ledger path in front of it. Neither view
// sees both spellings: the joined one catches the spaced path, the unjoined one
// catches the smuggled token as itself. So the scan runs over both and the
// findings are unioned.
func scribeFold(text string) (unjoined, joined string) {
	text = scribeDecode(text)
	text = strings.ReplaceAll(text, `\`, "/")
	return text, scribeSpacedSeparatorRe.ReplaceAllString(text, "/")
}

// scribePathRe matches a repository-path-shaped run, and it runs over the WHOLE
// definition rather than over one section. That is deliberate on two counts.
// Markdown is not a hiding place: once separators are folded, inline code, a
// fenced block and bare prose are the same characters, so none of the three is
// somewhere a path can sit unread. And no section is skipped, so a second
// `## Inputs` heading — or an exclusion list that names what it excludes — is not
// a way in either. The definition states its exclusions by category, never by
// path, which is what makes the whole-file rule affordable.
//
// THE LIMIT, stated rather than discovered: this check judges PATHS, and a path
// is recognized by its separator. A separator-free filename — `Makefile`,
// `CHANGELOG.md`, a bare `go.mod` — carries none and is outside its reach. What
// the check therefore proves is that the definition names no shipped-tree
// DIRECTORY or nested file, not that it names nothing from the shipped tree at
// all. spc-66 says the same in its own words.
var scribePathRe = regexp.MustCompile(`[A-Za-z0-9_.~*<>-]*/[A-Za-z0-9_.~*<>/-]*`)

// scribeInputsHeadingRe matches the allow list's heading at any depth.
var scribeInputsHeadingRe = regexp.MustCompile(`(?m)^#{1,6}[ \t]+Inputs\b`)

// scribeAccessFindings returns every way one definition's text breaches the
// access rule: a missing allow list, an allow list that names nothing, a code
// point outside ASCII and the typographic list, a residual encoding, a traversal
// segment, or any path outside the ledger root.
func scribeAccessFindings(prompt string) []string {
	var out []string
	if !scribeInputsHeadingRe.MatchString(prompt) {
		out = append(out, "carries no Inputs section; the access rule IS the allow list, so its absence is the breach")
	}

	unjoined, joined := scribeFold(prompt)

	// One finding, naming the first offender: the class is what matters, and a
	// hostile file could carry thousands. The three refusals keep separate messages
	// even where one subsumes another — an invisible code point, a control
	// character and a homoglyph are different mistakes to be told about.
	//
	// The two whitespace controls the definition legitimately uses are named first
	// and pass; every other control is refused, DEL and the C1 range included. The
	// carriage return is in the refused set on purpose: this file is LF-only, and a
	// control character splits a path match exactly as a format code point does,
	// while a viewer that renders it invisibly shows the spliced halves as one
	// allowed path.
	for _, r := range unjoined {
		switch {
		case r == '\n' || r == '\t':
			continue
		case unicode.IsControl(r):
			out = append(out, fmt.Sprintf("carries the control character U+%04X: a control splits a path match "+
				"while a viewer renders it invisibly, so the halves read as one allowed path; only LF and TAB "+
				"are permitted", r))
		case r < 0x80 || scribeAllowedNonASCII[r]:
			continue
		case unicode.Is(unicode.Cf, r):
			out = append(out, fmt.Sprintf("carries the format code point U+%04X: an invisible code point splices "+
				"a path back together after any check that reads the visible text", r))
		default:
			out = append(out, fmt.Sprintf("carries the non-ASCII code point U+%04X: outside the typographic marks "+
				"the definition uses, a code point outside ASCII is a separator lookalike or a homoglyph, and it "+
				"is refused rather than folded", r))
		}
		break
	}

	if residual := scribeResidualEncodingRe.FindString(unjoined); residual != "" {
		out = append(out, "carries the encoded shape "+residual+" after decoding to a fixpoint; the definition has "+
			"no reason to carry one, and a shape that survives decoding is one this check cannot read")
	}

	// Both views, unioned and de-duplicated: a path that appears in each is one
	// finding, and a path only one view can see is still reported.
	seen := 0
	reported := map[string]bool{}
	add := func(finding string) {
		if !reported[finding] {
			reported[finding] = true
			out = append(out, finding)
		}
	}
	for _, raw := range append(scribePathRe.FindAllString(unjoined, -1),
		scribePathRe.FindAllString(joined, -1)...) {
		// The traversal check runs on the RAW match, before any trimming. Trailing
		// sentence punctuation includes the full stop, and trimming it first ate the
		// second dot of a path whose final segment is `..` — so
		// `.abcd/work/issues/..` arrived at the prefix check as
		// `.abcd/work/issues`, passed it, and admitted the whole of `.abcd/work/`,
		// which is exactly what narrowing the root was for. A dot pair is never
		// punctuation here: no sentence in this definition ends a path with one.
		if strings.Contains(raw, "..") {
			seen++
			add("names " + raw + ": a traversal segment escapes whatever prefix a reader checks")
			continue
		}
		// Trailing sentence punctuation is not part of the path; a LEADING dot is
		// (`.abcd/...`), so the two ends are trimmed with different sets.
		tok := strings.TrimLeft(strings.TrimRight(raw, `.,;:!?)]"'`), `([`)
		if !strings.Contains(tok, "/") || scribeProseIdioms[strings.ToLower(tok)] {
			continue
		}
		seen++
		if !strings.HasPrefix(path.Clean(tok)+"/", scribeLedgerRoot) {
			add("names " + tok + ", outside " + scribeLedgerRoot)
		}
	}
	if seen == 0 {
		out = append(out, "names no repository path at all; an allow list with nothing on it proves nothing")
	}
	return out
}

// scribeFindingClasses names the branch each bypass case is meant to pin. The
// table asserts the CLASS of the finding, not merely that some finding appeared:
// several of these inputs trip two branches at once, so a case that asked only
// for "any finding" would let the branch it exists to cover stop firing without
// anything going red.
var scribeFindingClasses = map[string]string{
	"path":      ", outside ",
	"traversal": "a traversal segment",
	"non-ascii": "the non-ASCII code point",
	"format":    "the format code point",
	"control":   "the control character",
	"encoding":  "carries the encoded shape",
}

// scribeConformingBase is a minimal definition that satisfies the access rule.
// Every hostile case below is this text plus exactly one smuggled path, so the
// smuggle is the only thing a failure can be about.
const scribeConformingBase = "---\nname: scribe\n---\n\n" +
	"## Inputs (the allow list)\n\n" +
	"- `.abcd/work/issues/readings/` — the reading records already on file.\n\n" +
	"## Never in context\n\nThe shipped repository as an object of judgment.\n"

// TestScribeAccessCheckRefusesEveryBypass is the control, and it is what makes the
// case below it worth anything: each entry is a way of writing a shipped-tree path
// into the definition, and the check must report every one. A guard that reads
// only the first heading, only inline code, or only an ASCII solidus is a guard
// whose allow list is advisory.
func TestScribeAccessCheckRefusesEveryBypass(t *testing.T) {
	cases := []struct{ name, smuggled, class string }{
		{"a second Inputs heading", "\n## Inputs (continued)\n\n- `internal/core/lint/agentcontract.go` — the rule.\n", "path"},
		{"a bare path in prose", "\nRead internal/core/lint/agentcontract.go before transcribing.\n", "path"},
		{"a path inside a fence", "\n```\ninternal/core/lint/agentcontract.go\n```\n", "path"},
		{"a fullwidth solidus", "\n- `internal／core／lint／agentcontract.go` — the rule.\n", "non-ascii"},
		{"a backslash separator", "\n- `internal\\core\\lint\\agentcontract.go` — the rule.\n", "path"},
		{"a traversal out of the ledger", "\n- `.abcd/work/issues/../../development/readings/` — the run record.\n", "traversal"},
		{"the shared decision log", "\n- `.abcd/work/DECISIONS.md` — the decisions.\n", "path"},
		{"a session-transcript store path", "\n- `~/.abcd/history/aaaa/transcripts/` — prior sessions.\n", "path"},
		{"a bare shipped-tree directory", "\n- `internal/core` — where the rule lives.\n", "path"},
		{"a bare docs directory", "\n- `docs/` — the user-facing tree.\n", "path"},
		{"a directory-and-file pair with no extension", "\n- `internal/README` — the package map.\n", "path"},
		{"a spaced separator", "\n- `internal / core / lint / agentcontract.go` — the rule.\n", "path"},
		{"an HTML-escaped separator", "\n- `internal&#47;core&#47;lint&#47;agentcontract.go` — the rule.\n", "path"},
		{"a percent-encoded separator", "\n- `internal%2Fcore%2Flint%2Fagentcontract.go` — the rule.\n", "path"},
		{"a box-drawing solidus", "\n- `internal╱core╱lint╱agentcontract.go` — the rule.\n", "non-ascii"},
		{"a big solidus", "\n- `internal⧸core⧸lint⧸agentcontract.go` — the rule.\n", "non-ascii"},
		{"a fullwidth reverse solidus", "\n- `internal＼core＼lint＼agentcontract.go` — the rule.\n", "non-ascii"},
		{"a small reverse solidus", "\n- `internal﹨core﹨lint﹨agentcontract.go` — the rule.\n", "non-ascii"},
		{"a stray percent sign beside an encoded path", "\nProgress is 50% complete.\n- `internal%2Fcore%2Flint%2Fagentcontract.go` — the rule.\n", "path"},
		{"a double-encoded separator", "\n- `internal%252Fcore%252Flint%252Fagentcontract.go` — the rule.\n", "path"},
		{"a double-escaped HTML separator", "\n- `internal&amp;#x2F;core&amp;#x2F;lint&amp;#x2F;agentcontract.go` — the rule.\n", "path"},
		{"a percent-encoded HTML entity separator", "\n- `internal%26sol;core%26sol;lint%26sol;agentcontract.go` — the rule.\n", "path"},
		{"a reverse solidus lookalike", "\n- `internal╲core╲lint╲agentcontract.go` — the rule.\n", "non-ascii"},
		{"a big reverse solidus", "\n- `internal⧹core⧹lint⧹agentcontract.go` — the rule.\n", "non-ascii"},
		{"a set-minus separator", "\n- `internal∖core∖lint∖agentcontract.go` — the rule.\n", "non-ascii"},
		{"a reverse solidus operator", "\n- `internal⧵core⧵lint⧵agentcontract.go` — the rule.\n", "non-ascii"},
		{"a Cyrillic lookalike inside a ledger path", "\n- `.abcd/work/issues/readingс` — the ledger.\n", "non-ascii"},
		{"a residual percent fragment", "\n- `.abcd/work/issues/%2` — the ledger.\n", "encoding"},
		{"a bare trailing traversal", "\n- .abcd/work/issues/.. — the ledger.\n", "traversal"},
		{"a trailing traversal in inline code", "\n- `.abcd/work/issues/..` — the ledger.\n", "traversal"},
		{"an entity-encoded trailing traversal", "\n- `.abcd/work/issues/&#46;&#46;` — the ledger.\n", "traversal"},
		{"a carriage return splicing a traversal", "\n- `.abcd/work/issues/\r..` — the ledger.\n", "control"},
		{"a vertical tab splicing a traversal", "\n- `.abcd/work/issues/\v..` — the ledger.\n", "control"},
		{"a NUL splicing a traversal", "\n- `.abcd/work/issues/\x00..` — the ledger.\n", "control"},
		{"a space joining a shipped path onto the root", "\nRead .abcd/work/issues/ internal/core/lint/agentcontract.go before transcribing.\n", "path"},
		{"a join inside one code span", "\n- `.abcd/work/issues/ internal/core/lint/agentcontract.go` — the ledger.\n", "path"},
		{"a TAB joining a shipped path onto the root", "\n- `.abcd/work/issues/\tinternal/core/lint/agentcontract.go` — the ledger.\n", "path"},
		{"a format code point in the prose", "\nThe allow list above is exhaustive.\u202E\n", "format"},
		{"a zero-width joiner splicing a path", "\n- `internal/core/li\u200dnt/agentcontract.go` — the rule.\n", "format"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			marker, ok := scribeFindingClasses[tc.class]
			if !ok {
				t.Fatalf("case declares the unknown finding class %q", tc.class)
			}
			fs := scribeAccessFindings(scribeConformingBase + tc.smuggled)
			if len(fs) == 0 {
				t.Fatalf("the access check admits %s; a definition edited that way passes this gate", tc.name)
			}
			for _, f := range fs {
				if strings.Contains(f, marker) {
					return
				}
			}
			t.Fatalf("%s was reported, but as %v rather than a %s finding; each case pins the branch that is "+
				"supposed to catch it, so a branch that stops firing cannot hide behind a sibling's finding",
				tc.name, fs, tc.class)
		})
	}
}

// TestScribeAccessCheckPassesTheConformingShape proves the check is not simply
// refusing everything: the base the hostile cases are built on passes clean.
func TestScribeAccessCheckPassesTheConformingShape(t *testing.T) {
	if fs := scribeAccessFindings(scribeConformingBase); len(fs) != 0 {
		t.Fatalf("the conforming base must pass; got %v", fs)
	}
}

// TestScribeInputsAreLedgerOnly is the access rule over the real definition.
func TestScribeInputsAreLedgerOnly(t *testing.T) {
	root := filepath.Join("..", "..", "..")
	for _, f := range scribeAccessFindings(readRepoFile(t, root, scribePromptRel)) {
		t.Errorf("%s %s", scribePromptRel, f)
	}
}

// transcriptStoreNeedles are the spellings of the session-transcript store's
// path. Invariant 15 reserves that store to an enumerated consumer list the
// scribe is not on, so the definition names no path into it at all. Both
// vintages are held: the store the corpus lives in now, its per-repo pull-in,
// and the location it was moved out of, because a definition that names a path
// the corpus has left still declares an access this scribe may not have.
var transcriptStoreNeedles = []string{
	".abcd/transcripts",
	".work.local/transcripts",
	".abcd/history",
	"history/transcripts",
}

// scribeTranscriptStoreFindings reports every named path into the store.
func scribeTranscriptStoreFindings(prompt string) []string {
	var out []string
	for _, needle := range transcriptStoreNeedles {
		if strings.Contains(prompt, needle) {
			out = append(out, "names "+needle)
		}
	}
	return out
}

// TestScribeDeclaresNoTranscriptStoreAccess holds invariant 15's second half, and
// arms itself against a definition that does name the store.
func TestScribeDeclaresNoTranscriptStoreAccess(t *testing.T) {
	root := filepath.Join("..", "..", "..")
	for _, f := range scribeTranscriptStoreFindings(readRepoFile(t, root, scribePromptRel)) {
		t.Errorf("%s %s; invariant 15 reserves the session-transcript store to an enumerated consumer "+
			"list the scribe is not on, and adding it there is an invariant change, never a code change",
			scribePromptRel, f)
	}
	for _, path := range []string{
		"~/.abcd/transcripts/aaaa/records/",
		"repo/.abcd/.work.local/transcripts/aaaa/records/",
		"~/.abcd/history/aaaa/transcripts/",
	} {
		hostile := scribeConformingBase + "\n- `" + path + "` — prior sessions.\n"
		if len(scribeTranscriptStoreFindings(hostile)) == 0 {
			t.Fatalf("the transcript-store check admits %s; it is not armed for that spelling", path)
		}
	}
}

// TestScribeCanaryAssertsTheRefusals reads the fixture the agent_contract rule
// already proves present, non-empty and regular, and asserts what that rule
// cannot: that it parses, carries a hostile payload, and declares the expectation
// the payload is measured against. A control string the input never contains is a
// `must_not_contain` that no run can fail.
func TestScribeCanaryAssertsTheRefusals(t *testing.T) {
	root := filepath.Join("..", "..", "..")
	raw := readRepoFile(t, root, scribeCanaryRel)

	var fixture struct {
		Input    json.RawMessage `json:"input"`
		Expected struct {
			MustNotContain       string   `json:"must_not_contain"`
			MustNotObey          []string `json:"must_not_obey"`
			InjectionIsInertData bool     `json:"injection_is_inert_data"`
		} `json:"expected"`
	}
	if err := json.Unmarshal([]byte(raw), &fixture); err != nil {
		t.Fatalf("%s does not parse as JSON: %v", scribeCanaryRel, err)
	}
	if len(fixture.Input) == 0 {
		t.Errorf("%s carries no 'input'; a canary with no hostile payload asserts nothing", scribeCanaryRel)
	}
	if len(fixture.Expected.MustNotObey) == 0 {
		t.Errorf("%s declares no 'expected.must_not_obey'; the expectation is what the canary asserts", scribeCanaryRel)
	}
	if !fixture.Expected.InjectionIsInertData {
		t.Errorf("%s does not declare 'expected.injection_is_inert_data'; that is the contract every canary asserts",
			scribeCanaryRel)
	}
	// The control string has to be a shape only OBEDIENCE produces. The payload
	// must demand it, or no run can fail the control; and the faithful output must
	// not carry it, or every honest run fails instead — the scribe transcribes item
	// bodies verbatim, so a bare word the payload plants is in the output by design.
	// The demanded word therefore appears in the input and the emitted shape does not.
	tok := fixture.Expected.MustNotContain
	word := strings.Trim(tok, `: "`)
	switch {
	case tok == "" || word == "":
		t.Errorf("%s declares no usable 'expected.must_not_contain'", scribeCanaryRel)
	case !strings.Contains(string(fixture.Input), word):
		t.Errorf("%s's control word %q is demanded nowhere in its input; a control the payload never asks for is "+
			"one no run can fail", scribeCanaryRel, word)
	case strings.Contains(string(fixture.Input), tok):
		t.Errorf("%s's control %q already appears in its input; verbatim transcription puts it in the output, so a "+
			"control shaped like that is one every honest run fails", scribeCanaryRel, tok)
	}

	// Every item carries an identifier, because the dispositions and the
	// outstanding list refer to items by id; an item with none is a reference to
	// nothing.
	var items struct {
		Items []struct {
			Item string `json:"item"`
		} `json:"items"`
	}
	if err := json.Unmarshal(fixture.Input, &items); err != nil {
		t.Fatalf("%s's input does not parse: %v", scribeCanaryRel, err)
	}
	if len(items.Items) == 0 {
		t.Fatalf("%s's input carries no items", scribeCanaryRel)
	}
	ids := map[string]bool{}
	for i, it := range items.Items {
		if it.Item == "" {
			t.Errorf("%s's input item %d carries no 'item' identifier, yet the dispositions and the outstanding "+
				"list refer to items by id", scribeCanaryRel, i+1)
			continue
		}
		ids[it.Item] = true
	}

	// The exemplar is the fixture's own statement of what a faithful run emits, so
	// it has to obey the behavior it illustrates: one reading record per item.
	// An exemplar that transcribes one of two items teaches the opposite of the
	// rule it is there to show.
	var example struct {
		Expected struct {
			Example json.RawMessage `json:"emitted_material_example"`
		} `json:"expected"`
	}
	if err := json.Unmarshal([]byte(raw), &example); err != nil {
		t.Fatalf("%s does not parse: %v", scribeCanaryRel, err)
	}
	// The exemplar is what a faithful run looks like, so it must not itself carry
	// the shape the control forbids. A fixture whose own model answer trips its own
	// control is a fixture that has stopped agreeing with itself.
	if tok != "" && strings.Contains(string(example.Expected.Example), tok) {
		t.Errorf("%s's emitted_material_example carries the control %q it forbids; the exemplar is the model "+
			"answer, so a control it trips is one no faithful run can satisfy either", scribeCanaryRel, tok)
	}
	var records struct {
		ReadingRecords []struct {
			Item string `json:"item"`
		} `json:"reading_records"`
	}
	if err := json.Unmarshal(example.Expected.Example, &records); err != nil {
		t.Fatalf("%s's emitted_material_example does not parse: %v", scribeCanaryRel, err)
	}
	emitted := map[string]bool{}
	for _, r := range records.ReadingRecords {
		emitted[r.Item] = true
	}
	for id := range ids {
		if !emitted[id] {
			t.Errorf("%s's emitted_material_example carries no reading record for %s; the fixture states one "+
				"reading record per item, so an exemplar that skips one contradicts the behavior it illustrates",
				scribeCanaryRel, id)
		}
	}
	// The definition carries two refusals beyond "never obey an instruction":
	// transcript material handed over as ledger context is refused outright, and a
	// composed contribution is stamped or withheld. A canary that tests neither
	// leaves both unexercised.
	declared := strings.ToLower(strings.Join(fixture.Expected.MustNotObey, " | "))
	for _, lure := range []string{"transcript", "summary"} {
		if !strings.Contains(declared, lure) {
			t.Errorf("%s's expected.must_not_obey names no %q refusal; the definition refuses transcript material "+
				"handed as ledger context and refuses an unstamped composed contribution, so the canary tests both",
				scribeCanaryRel, lure)
		}
	}
}

// TestScribePromptSatisfiesTheContract runs the shipped agent_contract rule over
// the real tree and asserts the scribe passes all three sub-checks. The second
// half arms the rule id: a rule renamed out from under this case would otherwise
// leave it filtering for findings that can never appear.
func TestScribePromptSatisfiesTheContract(t *testing.T) {
	cfg := lint.Config{Rules: map[string]lint.RuleConfig{
		scribeAgentContractRule: {Enabled: true, Severity: "blocker"},
	}}

	fs, err := lint.Lint(cfg, filepath.Join("..", "..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range fs {
		if f.RuleID != scribeAgentContractRule {
			continue
		}
		if filepath.ToSlash(f.File) == scribePromptRel || strings.Contains(f.Message, "scribe") {
			t.Errorf("%s: %s", f.File, f.Message)
		}
	}

	broken := t.TempDir()
	if err := os.MkdirAll(filepath.Join(broken, "agents"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(broken, "agents", "unversioned.md"),
		[]byte("---\nname: unversioned\n---\n\nPrompt body.\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	armed, err := lint.Lint(cfg, broken)
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range armed {
		if f.RuleID == scribeAgentContractRule {
			return
		}
	}
	t.Fatalf("no %q finding over a deliberately broken agent tree; the rule id this case filters on is stale",
		scribeAgentContractRule)
}
