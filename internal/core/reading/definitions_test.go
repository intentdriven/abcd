package reading

// definitions_test.go holds itd-184/spc-62 to its acceptance criteria over the
// four definitions that actually ship, plus the locator's own contract over a
// temporary root.
//
// The tests read the repository's real agents/ tree deliberately. A definition
// is not code the package can construct; it is a file a host dispatches, and a
// guard that only ever saw a fixture would pass on a tree whose definitions had
// drifted apart.

import (
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/issueschema"
	"github.com/intentdriven/abcd/internal/core/lint"
)

// coreBegin and coreEnd delimit the blindness core inside a definition. They are
// spelled here rather than imported so an edit to the production markers cannot
// silently retarget the test at a span nobody meant.
const (
	coreBegin = "<!-- blindness-core:begin -->"
	coreEnd   = "<!-- blindness-core:end -->"
)

// blindnessConditions are the seven core conditions in the one fixed order
// itd-184 states them in. The bold lead-in is the anchor: it is the condition's
// name, and a silent deletion or a reorder moves it.
var blindnessConditions = []string{
	"**No project context.**",
	"**No ledger access.**",
	"**No memory across runs.**",
	"**No ranking or prioritisation.**",
	"**No selection, explanation or commitment.**",
	"**Named provenance on every item produced.**",
	"**No passed input is authoritative.**",
}

// definitionText reads one definition out of the repository under test.
func definitionText(t *testing.T, root string, p Position) string {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(DefinitionPath(p))))
	if err != nil {
		t.Fatalf("read the %s definition: %v", p, err)
	}
	return string(raw)
}

// coreSpan returns the delimited blindness core, delimiters included, so the
// comparison is over an exact span rather than a heuristic slice.
func coreSpan(t *testing.T, p Position, text string) string {
	t.Helper()
	start := strings.Index(text, coreBegin)
	if start < 0 {
		t.Fatalf("the %s definition carries no %s marker", p, coreBegin)
	}
	end := strings.Index(text, coreEnd)
	if end < 0 {
		t.Fatalf("the %s definition carries no %s marker", p, coreEnd)
	}
	if end < start {
		t.Fatalf("the %s definition's core markers are inverted", p)
	}
	return text[start : end+len(coreEnd)]
}

// section returns the body of one `## ` heading, up to the next `## ` heading.
func section(t *testing.T, p Position, text, heading string) string {
	t.Helper()
	marker := "\n## " + heading + "\n"
	start := strings.Index(text, marker)
	if start < 0 {
		t.Fatalf("the %s definition carries no '## %s' section", p, heading)
	}
	rest := text[start+len(marker):]
	if next := strings.Index(rest, "\n## "); next >= 0 {
		return rest[:next]
	}
	return rest
}

// admittedSources derives the repository sources the include table admits at p,
// deduplicated in table order. It is the truth the definitions' object sections
// are held against: the table is where a reading's object is actually bounded.
func admittedSources(p Position) []string {
	seen := map[string]bool{}
	out := []string{}
	for _, row := range Table {
		if !row.AdmittedAt(p) || seen[row.Source] {
			continue
		}
		seen[row.Source] = true
		out = append(out, row.Source)
	}
	return out
}

// sourceListHeading names the object subsection every definition renders the
// include table's admitted sources into.
const sourceListHeading = "### Repository sources the assembler admits at this position"

// backtickedRe captures the first backticked token on a line.
var backtickedRe = regexp.MustCompile("`([^`]+)`")

// whitespaceRe collapses a hard-wrapped paragraph so a prose assertion is about
// the sentence rather than about where the line happened to break.
var whitespaceRe = regexp.MustCompile(`\s+`)

// flatten collapses every whitespace run to one space.
func flatten(s string) string { return whitespaceRe.ReplaceAllString(s, " ") }

// blockquoteRe matches a markdown blockquote line, which is how each definition
// carries its question.
var blockquoteRe = regexp.MustCompile(`(?m)^> \S`)

// statedSources reads the source list out of a definition's object section.
func statedSources(t *testing.T, p Position, object string) []string {
	t.Helper()
	start := strings.Index(object, sourceListHeading)
	if start < 0 {
		t.Fatalf("the %s definition's object carries no %q subsection", p, sourceListHeading)
	}
	out := []string{}
	for _, line := range strings.Split(object[start:], "\n") {
		if !strings.HasPrefix(line, "- ") {
			continue
		}
		m := backtickedRe.FindStringSubmatch(line)
		if m == nil {
			t.Fatalf("the %s definition's source list has a bullet naming no path: %q", p, line)
		}
		out = append(out, m[1])
	}
	return out
}

// TestBlindnessCoreIsByteIdenticalAcrossDefinitions is ac-1's first half: the
// delimited span is the same bytes in all four files. Separating the four
// definitions by licence must not separate them by blindness, and the cheap
// guarantee of that is that the core cannot be edited in one copy alone.
func TestBlindnessCoreIsByteIdenticalAcrossDefinitions(t *testing.T) {
	root := repoRoot(t)
	var first string
	var firstPos Position
	for _, p := range Positions() {
		span := coreSpan(t, p, definitionText(t, root, p))
		if first == "" {
			first, firstPos = span, p
			continue
		}
		if span != first {
			t.Errorf("the blindness core in the %s definition is not byte-identical to the %s one\n"+
				"--- %s ---\n%s\n--- %s ---\n%s", p, firstPos, firstPos, first, p, span)
		}
	}
	if first == "" {
		t.Fatal("no definition carried a blindness core")
	}
}

// TestBlindnessCoreCarriesSevenConditions is ac-1's second half: the span holds
// the seven conditions, in the fixed order, and holds no eighth. Byte-identity
// alone would be satisfied by four identically empty cores, so a silent deletion
// is caught here rather than nowhere.
func TestBlindnessCoreCarriesSevenConditions(t *testing.T) {
	root := repoRoot(t)
	for _, p := range Positions() {
		span := coreSpan(t, p, definitionText(t, root, p))
		at := -1
		for i, cond := range blindnessConditions {
			idx := strings.Index(span, cond)
			if idx < 0 {
				t.Fatalf("the %s definition's blindness core is missing condition %d, %s", p, i+1, cond)
			}
			if idx <= at {
				t.Fatalf("the %s definition's blindness core carries condition %d, %s, out of the fixed order",
					p, i+1, cond)
			}
			at = idx
		}
		numbered := regexp.MustCompile(`(?m)^\d+\. \*\*`).FindAllString(span, -1)
		if len(numbered) != len(blindnessConditions) {
			t.Errorf("the %s definition's blindness core carries %d numbered conditions, want %d",
				p, len(numbered), len(blindnessConditions))
		}
	}
}

// TestEveryDefinitionStatesItsRegime is ac-2's first half, read through the
// locator rather than through a regexp: every definition states a regime drawn
// from the four, in its own frontmatter, which is what makes the regime the
// definition's property rather than the payload's.
func TestEveryDefinitionStatesItsRegime(t *testing.T) {
	root := repoRoot(t)
	for _, p := range Positions() {
		def, err := LoadDefinition(root, p)
		if err != nil {
			t.Fatalf("load the %s definition: %v", p, err)
		}
		if def.Position != p {
			t.Errorf("the %s definition states position %q", p, def.Position)
		}
		if def.Regime == "" {
			t.Errorf("the %s definition states no regime", p)
		}
		if def.SHA256 == "" {
			t.Errorf("the %s definition resolved to no hash", p)
		}
		if def.Path != DefinitionPath(p) {
			t.Errorf("the %s definition resolved to %q, want %q", p, def.Path, DefinitionPath(p))
		}
	}
}

// TestRegimeValuesAreTheFourAndDistinct is ac-2's second half: the four values
// are distinct, and each is resolvable from the position alone — the definition's
// stated regime and the schema's position table agree, so a run's regime never
// has to be supplied.
func TestRegimeValuesAreTheFourAndDistinct(t *testing.T) {
	root := repoRoot(t)
	seen := map[string]Position{}
	for _, p := range Positions() {
		def, err := LoadDefinition(root, p)
		if err != nil {
			t.Fatalf("load the %s definition: %v", p, err)
		}
		if prior, dup := seen[def.Regime]; dup {
			t.Errorf("the %s and %s definitions both state regime %q; the four are distinct",
				prior, p, def.Regime)
		}
		seen[def.Regime] = p
		if want := issueschema.ReadingRegime(string(p)); def.Regime != want {
			t.Errorf("the %s definition states regime %q; the position resolves to %q",
				p, def.Regime, want)
		}
	}
	if len(seen) != len(Positions()) {
		t.Errorf("%d distinct regimes across %d positions", len(seen), len(Positions()))
	}
}

// TestDefinitionHoldsItsFiveParts: each file carries its object, its question,
// the core, its regime and its item shape, and nothing in the item shape belongs
// to another position. The body field names are read out of the schema the ingest
// verb validates against, so a definition and the contract cannot drift.
func TestDefinitionHoldsItsFiveParts(t *testing.T) {
	root := repoRoot(t)
	for _, p := range Positions() {
		text := definitionText(t, root, p)

		object := section(t, p, text, "Object")
		question := section(t, p, text, "Question")
		regime := section(t, p, text, "Regime")
		itemShape := section(t, p, text, "Item shape")
		coreSpan(t, p, text)

		// The question is QUOTED, as a blockquote line. A bare ">" test cannot
		// fail here: the section runs to the next `## `, which is the core's own
		// heading inside the begin marker, so the section always ends in "-->".
		// The anchor is a line that starts a blockquote.
		if !blockquoteRe.MatchString(question) {
			t.Errorf("the %s definition does not quote its question as a blockquote:\n%s", p, question)
		}
		if want := issueschema.ReadingRegime(string(p)); !strings.Contains(regime, "`"+want+"`") {
			t.Errorf("the %s definition's regime section does not name `%s`", p, want)
		}

		// The object's source list is the include table for this position,
		// which is where a reading's object is actually bounded.
		got, want := statedSources(t, p, object), admittedSources(p)
		sortedGot, sortedWant := append([]string{}, got...), append([]string{}, want...)
		sort.Strings(sortedGot)
		sort.Strings(sortedWant)
		if strings.Join(sortedGot, "|") != strings.Join(sortedWant, "|") {
			t.Errorf("the %s definition states sources %v; the include table admits %v", p, got, want)
		}

		// The item shape carries this position's body fields and no other
		// position's, so a licence cannot be smuggled in as a field.
		for _, field := range issueschema.ReadingBodyFields[string(p)] {
			if !strings.Contains(itemShape, "`"+field+"`") {
				t.Errorf("the %s definition's item shape does not name the field `%s`", p, field)
			}
			if !strings.Contains(itemShape, `"`+field+`"`) {
				t.Errorf("the %s definition's item shape has no %q key in its fenced example", p, field)
			}
		}
		for _, other := range Positions() {
			if other == p {
				continue
			}
			for _, field := range issueschema.ReadingBodyFields[string(other)] {
				// Named AS A FIELD: backticked, or a key in the fenced example.
				// A bare substring would fire on prose that merely uses the
				// word — the entailment position's claim types include the
				// English word "criterion".
				if strings.Contains(itemShape, "`"+field+"`") || strings.Contains(itemShape, `"`+field+`"`) {
					t.Errorf("the %s definition's item shape names %q, which is the %s position's body field",
						p, field, other)
				}
			}
		}
		if n := strings.Count(text, "\n```json\n"); n != 1 {
			t.Errorf("the %s definition carries %d fenced json blocks, want exactly 1", p, n)
		}
	}
}

// TestWideningDefinitionExcludesDraftsAndPlanned pins the object asymmetry's
// first half against the table that enforces it: the widening reading must not
// see the candidate set it is asked to widen.
func TestWideningDefinitionExcludesDraftsAndPlanned(t *testing.T) {
	root := repoRoot(t)
	object := section(t, PositionWidening, definitionText(t, root, PositionWidening), "Object")
	stated := strings.Join(statedSources(t, PositionWidening, object), "|")
	for _, src := range []string{
		".abcd/development/intents/drafts",
		".abcd/development/intents/planned",
	} {
		for _, row := range Table {
			if row.Source == src && row.AdmittedAt(PositionWidening) {
				t.Errorf("the include table admits %s at the widening position", src)
			}
		}
		if strings.Contains(stated, src) {
			t.Errorf("the widening definition lists %s among its admitted sources", src)
		}
	}
	if !strings.Contains(flatten(object), "withheld from you deliberately") {
		t.Error("the widening definition does not state that the candidate set is withheld from it")
	}
}

// TestEntailmentDefinitionIncludesThem is the asymmetry's second half:
// articulation precedes selection, so the entailment reading properly reads
// drafts.
func TestEntailmentDefinitionIncludesThem(t *testing.T) {
	root := repoRoot(t)
	object := section(t, PositionEntailment, definitionText(t, root, PositionEntailment), "Object")
	stated := strings.Join(statedSources(t, PositionEntailment, object), "|")
	for _, src := range []string{
		".abcd/development/intents/drafts",
		".abcd/development/intents/planned",
	} {
		admitted := false
		for _, row := range Table {
			if row.Source == src && row.AdmittedAt(PositionEntailment) {
				admitted = true
			}
		}
		if !admitted {
			t.Errorf("the include table does not admit %s at the entailment position", src)
		}
		if !strings.Contains(stated, src) {
			t.Errorf("the entailment definition does not list %s among its admitted sources", src)
		}
	}
}

// TestComparativeObjectIsTheWideningPreAdmissionOutput pins the settled reading
// of the object the two intents disagreed about, so the disagreement cannot be
// reintroduced by an edit that reads plausibly.
//
// It now pins the CHANNEL as well as the object (adr-2609021016272867;
// spc-2609020626039834, "The comparative definition"). The object was always the
// widening reading's pre-admission output; what moved is that the definition can
// say how the reading receives it, and the two halves the exception rests on
// have to be in the definition's own words: the candidate set is ONE derived
// run's items, and what has happened to them since is withheld. A definition
// stating the first without the second would describe a channel wider than the
// one the ADR admits.
func TestComparativeObjectIsTheWideningPreAdmissionOutput(t *testing.T) {
	root := repoRoot(t)
	object := flatten(section(t, PositionComparative, definitionText(t, root, PositionComparative), "Object"))
	for _, want := range []string{
		"widening reading's pre-admission output",
		"before admission",
		"never supplied at invocation",
		// The exception's limit, stated where the reading reads it.
		"one committed widening run at the target whose items carry no disposition and no admission",
		"no other prior run's stored output is readable",
		"withheld from you",
	} {
		if !strings.Contains(object, want) {
			t.Errorf("the comparative definition's object does not state %q", want)
		}
	}
}

// TestColdReadingDefinitionsSatisfyTheAgentContract runs record-lint's own
// agent_contract rule over the tree from here, so an itd-5 contract failure on
// one of these four fails locally rather than only in the gate.
func TestColdReadingDefinitionsSatisfyTheAgentContract(t *testing.T) {
	root := repoRoot(t)
	cfg, err := lint.LoadConfig(filepath.Join(root, filepath.FromSlash(LintConfigPath)))
	if err != nil {
		t.Fatalf("load %s: %v", LintConfigPath, err)
	}
	// Proof of life: a disabled rule reports nothing, and a test that reads
	// nothing from a rule that ran nowhere passes for the wrong reason.
	if rule, ok := cfg.Rules["agent_contract"]; !ok || !rule.Enabled {
		t.Fatalf("%s does not enable the agent_contract rule, so this case asserts nothing", LintConfigPath)
	}
	findings, err := lint.Lint(cfg, root)
	if err != nil {
		t.Fatalf("lint: %v", err)
	}
	for _, f := range findings {
		if f.RuleID != "agent_contract" {
			continue
		}
		if strings.Contains(f.File, definitionPrefix) || strings.Contains(f.Message, definitionPrefix) {
			t.Errorf("%s:%d [%s] %s", f.File, f.Line, f.RuleID, f.Message)
		}
	}
}

// --- the locator, over a temporary root ---

// writeDefinition writes a minimal well-formed definition under root.
func writeDefinition(t *testing.T, root string, p Position, frontmatter string) {
	t.Helper()
	rel := DefinitionPath(p)
	if err := os.MkdirAll(filepath.Join(root, filepath.Dir(filepath.FromSlash(rel))), 0o755); err != nil {
		t.Fatal(err)
	}
	body := "---\n" + frontmatter + "---\n\n# " + string(p) + "\n"
	if err := os.WriteFile(filepath.Join(root, filepath.FromSlash(rel)), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

// TestLoadDefinitionResolvesUnderAnArbitraryRoot is the signature contract the
// ingest verb needs: a caller supplies the repository root, so a test can point
// the locator at a temporary tree.
func TestLoadDefinitionResolvesUnderAnArbitraryRoot(t *testing.T) {
	root := t.TempDir()
	writeDefinition(t, root, PositionDetection, "position: detection\nregime: registrative\n")

	def, err := LoadDefinition(root, PositionDetection)
	if err != nil {
		t.Fatalf("LoadDefinition: %v", err)
	}
	if def.Position != PositionDetection || def.Regime != "registrative" {
		t.Errorf("resolved %+v", def)
	}

	// A double-quoted scalar is legal YAML and is the shape the record's other
	// writers emit. The value read out of it is the inner text: handing the raw
	// value to the shared decoder keeps the quotes, and the definition then
	// refuses itself with a message reading detection against detection.
	writeDefinition(t, root, PositionDetection, "position: \"detection\"\nregime: \"registrative\"\n")
	quoted, err := LoadDefinition(root, PositionDetection)
	if err != nil {
		t.Fatalf("LoadDefinition over a double-quoted frontmatter: %v", err)
	}
	if quoted.Position != PositionDetection || quoted.Regime != "registrative" {
		t.Errorf("a double-quoted frontmatter resolved to %+v", quoted)
	}
	writeDefinition(t, root, PositionDetection, "position: detection\nregime: registrative\n")
	if def.Path != "agents/cold-reading-detection.md" {
		t.Errorf("path %q", def.Path)
	}
	// The hash is over the file's bytes, so an edit moves it.
	before := def.SHA256
	writeDefinition(t, root, PositionDetection, "position: detection\nregime: registrative\ncolor: cyan\n")
	after, err := LoadDefinition(root, PositionDetection)
	if err != nil {
		t.Fatalf("LoadDefinition after the edit: %v", err)
	}
	if after.SHA256 == before || after.SHA256 == "" {
		t.Errorf("the hash did not move on an edit: %q then %q", before, after.SHA256)
	}
}

// TestLoadDefinitionRefusesAMalformedDefinition: the locator refuses by name
// rather than defaulting, because a defaulted regime is a regime nobody stated.
func TestLoadDefinitionRefusesAMalformedDefinition(t *testing.T) {
	cases := []struct {
		name        string
		frontmatter string
		want        string
	}{
		{"no regime", "position: detection\n", "regime"},
		{"no position", "regime: registrative\n", "position"},
		{"position disagrees with the filename", "position: widening\nregime: registrative\n", "widening"},
		{"regime outside the closed set", "position: detection\nregime: advisory\n", "advisory"},
		// Fields keeps the first occurrence, so a duplicated key would resolve
		// silently to one of two answers.
		{"regime stated twice", "position: detection\nregime: registrative\nregime: generative\n", "more than once"},
		{"position stated twice", "position: detection\nposition: widening\nregime: registrative\n", "more than once"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			root := t.TempDir()
			writeDefinition(t, root, PositionDetection, tc.frontmatter)
			_, err := LoadDefinition(root, PositionDetection)
			if err == nil {
				t.Fatal("a malformed definition resolved without an error")
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Errorf("error %q does not name %q", err, tc.want)
			}
		})
	}
}

// TestLoadDefinitionsSkipsAnAbsentDefinition: absence is a state, not a fault —
// a repository with no definitions has none, and that is what the status render
// reports. A present-but-broken one is a fault, and is reported as one.
func TestLoadDefinitionsSkipsAnAbsentDefinition(t *testing.T) {
	root := t.TempDir()
	writeDefinition(t, root, PositionWidening, "position: widening\nregime: generative\n")

	defs, err := LoadDefinitions(root)
	if err != nil {
		t.Fatalf("LoadDefinitions: %v", err)
	}
	if len(defs) != 1 || defs[0].Position != PositionWidening {
		t.Fatalf("resolved %+v, want the one definition present", defs)
	}

	writeDefinition(t, root, PositionDetection, "position: detection\n")
	if _, err := LoadDefinitions(root); err == nil {
		t.Fatal("a present-but-malformed definition was skipped rather than reported")
	}
}

// TestDescribeReportsTheDefinitionsTheLocatorResolves holds the locator wired:
// the bare verb's render is built from it, so the resolution the ingest verb
// depends on is the resolution an operator can see.
func TestDescribeReportsTheDefinitionsTheLocatorResolves(t *testing.T) {
	root := repoRoot(t)
	status, err := Describe(root)
	if err != nil {
		t.Fatalf("Describe: %v", err)
	}
	want := []string{}
	for _, p := range Positions() {
		want = append(want, definitionPrefix+string(p))
	}
	sort.Strings(want)
	if strings.Join(status.Definitions, "|") != strings.Join(want, "|") {
		t.Errorf("Describe reported %v, want %v", status.Definitions, want)
	}

	// The render is the locator's output, not a directory listing that happens
	// to agree with it: a definition the locator refuses is a definition the
	// render refuses. Without this, a listing and a resolution are
	// indistinguishable from here.
	broken := t.TempDir()
	writeDefinition(t, broken, PositionWidening, "position: widening\n")
	if _, err := Describe(broken); err == nil {
		t.Error("Describe listed a definition the locator refuses, so the render is a directory scan")
	}
}

// TestLoadDefinitionRefusesARegimeThatDisagreesWithItsPosition is the locator's
// second regime question, and the one that matters at runtime.
//
// Membership in the four regimes and AGREEMENT with the position are different
// questions, and only the first stops a file that states a legal regime under
// the wrong position. Such a file resolves confidently and hands its caller the
// wrong licence — and the caller is the ingest verb, whose entire purpose is to
// catch a reading that exceeded the licence it read under. A gate enforcing the
// wrong licence is the failure shape itd-185 exists to close, so the drift is
// refused HERE, in the one place that claims to resolve a position to its
// regime, rather than cross-checked again in every caller.
//
// The locator still never SUBSTITUTES the table's value for the file's. It
// refuses, which is what makes the drift visible at runtime instead of only in
// the tests that pin the pair (iss-2608311145258479).
func TestLoadDefinitionRefusesARegimeThatDisagreesWithItsPosition(t *testing.T) {
	for _, p := range Positions() {
		p := p
		t.Run(string(p), func(t *testing.T) {
			mine := issueschema.ReadingRegime(string(p))
			for _, other := range issueschema.ReadingPositions {
				if other.Regime == mine {
					continue
				}
				root := t.TempDir()
				writeDefinition(t, root, p, "position: "+string(p)+"\nregime: "+other.Regime+"\n")
				_, err := LoadDefinition(root, p)
				if err == nil {
					t.Fatalf("the %s definition stating regime %q resolved without an error; a legal "+
						"regime under the wrong position is still the wrong licence", p, other.Regime)
				}
				for _, want := range []string{other.Regime, mine} {
					if !strings.Contains(err.Error(), want) {
						t.Errorf("the refusal does not name %q: %v", want, err)
					}
				}
			}

			// The agreeing file still resolves. Without this the case above
			// would pass against a locator that refused every definition.
			root := t.TempDir()
			writeDefinition(t, root, p, "position: "+string(p)+"\nregime: "+mine+"\n")
			def, err := LoadDefinition(root, p)
			if err != nil {
				t.Fatalf("the agreeing %s definition was refused: %v", p, err)
			}
			if def.Regime != mine {
				t.Errorf("resolved regime %q, want %q", def.Regime, mine)
			}
		})
	}

	// And the whole-set resolver reports it rather than skipping it: a render
	// listing three instruments where four were meant is worse than an error.
	root := t.TempDir()
	writeDefinition(t, root, PositionWidening, "position: widening\nregime: registrative\n")
	if _, err := LoadDefinitions(root); err == nil {
		t.Error("LoadDefinitions skipped a definition whose regime disagrees with its position")
	}
}

// TestLoadDefinitionReadsInsideTheRepositoryOnly: the hash this returns is the
// definition half of an instrument identity, and the identity is sold as proving
// that two runs read under the same instructions. A symlinked ancestor under
// agents/ would have it hash a file that is not in this repository at all, which
// makes that a claim about an unknown artefact.
//
// The read is harmless in itself. The CLAIM is what is guarded.
func TestLoadDefinitionReadsInsideTheRepositoryOnly(t *testing.T) {
	root := t.TempDir()
	outside := t.TempDir()
	if err := os.WriteFile(filepath.Join(outside, definitionPrefix+"detection.md"),
		[]byte("---\nposition: detection\nregime: registrative\n---\n\n# elsewhere\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(root, DefinitionsDir)); err != nil {
		t.Fatal(err)
	}

	if _, err := LoadDefinition(root, PositionDetection); err == nil {
		t.Fatal("a definition was hashed through a symlink pointing outside the repository, so the " +
			"instrument identity it reports names a file this repository does not hold")
	}

	// The same definition, actually in the repository, still resolves — so this
	// is containment rather than a blanket refusal.
	if err := os.Remove(filepath.Join(root, DefinitionsDir)); err != nil {
		t.Fatal(err)
	}
	writeDefinition(t, root, PositionDetection, "position: detection\nregime: registrative\n")
	if _, err := LoadDefinition(root, PositionDetection); err != nil {
		t.Fatalf("an in-repository definition was refused: %v", err)
	}
}

// promptVersionRe reads a definition's declared prompt version out of its
// frontmatter.
var promptVersionRe = regexp.MustCompile(`(?m)^prompt_version:\s*(\S+)\s*$`)

// fourthConditionSentence is the companion's own wording, taken verbatim.
//
// The readings companion's section 2 says items are returned in the order they
// arise in the object; the blindness core's fourth condition said they come
// back unordered and unweighted, which is a different claim about the same
// thing (iss-2609021153261145, correction (7) of the 2026-09-02 ruling).
const fourthConditionSentence = "Items are returned in the order they arise in the object"

// retiredFourthConditionSentence is what the condition used to say. It is
// spelled out so the test fails on a definition that carries BOTH — a partial
// edit that added the new sentence beside the old one would satisfy a
// presence-only assertion.
const retiredFourthConditionSentence = "Items come back unordered and unweighted"

// promptVersions pins each definition's prompt version. It is a declaration
// rather than a derivation: a version read out of the file it is checking would
// agree with it by construction and could only confirm it.
//
// The four moved PATCH together with the fourth-condition correction, and PATCH
// again with itd-194, which is a change to the object's source list rather than
// to the blindness core: all four gain the brief's surfaces, internals,
// delivery and meta chapters, and widening loses the shipped intents.
//
// The comparative definition is one PATCH ahead of the other three from
// iss-2609021833302981, which moved nothing they share: its object section
// states the derivation rule — the only one of the four that has one to state —
// and the rule now reaches a widening run at an ancestor of the target. A shared
// edit still moves all four together; this one was not shared.
var promptVersions = map[Position]string{
	PositionWidening:    "0.2.2",
	PositionEntailment:  "0.1.2",
	PositionComparative: "0.1.3",
	PositionDetection:   "0.1.2",
}

// TestTheFourthConditionTakesTheCompanionsSentence is itd-2609021003095168 ac-7
// (companion v4 section 2, condition 4; spc-2609021004075744 "The blindness
// core's fourth condition").
//
// The rest of the condition — no severity, no confidence score, no
// most-important-first, no top-N, and the reason — is unchanged, and is held
// unchanged here so the edit cannot quietly take the rest of the condition with
// it. Byte-identity across the four is TestBlindnessCoreIsByteIdenticalAcross-
// Definitions' job and is not restated.
func TestTheFourthConditionTakesTheCompanionsSentence(t *testing.T) {
	root := repoRoot(t)
	for _, p := range Positions() {
		text := definitionText(t, root, p)
		span := coreSpan(t, p, text)
		if !strings.Contains(flatten(span), fourthConditionSentence) {
			t.Errorf("the %s definition's blindness core does not say %q; the companion's "+
				"section 2 says items are returned in the order they arise in the object",
				p, fourthConditionSentence)
		}
		if strings.Contains(flatten(span), retiredFourthConditionSentence) {
			t.Errorf("the %s definition's blindness core still says %q", p, retiredFourthConditionSentence)
		}
		for _, keep := range []string{
			"**No ranking or prioritisation.**",
			"no severity, no confidence score, no most-important-first, no top-N",
			"An ordering is an argument about what matters",
		} {
			if !strings.Contains(flatten(span), flatten(keep)) {
				t.Errorf("the %s definition's fourth condition lost %q; only the first sentence moves",
					p, keep)
			}
		}
		m := promptVersionRe.FindStringSubmatch(text)
		if m == nil {
			t.Fatalf("the %s definition declares no prompt_version", p)
		}
		if want := promptVersions[p]; m[1] != want {
			t.Errorf("the %s definition declares prompt_version %s, want %s: the four move PATCH "+
				"together with the sentence they share", p, m[1], want)
		}
	}
}
