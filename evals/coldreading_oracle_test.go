//go:build smoke || coldreading

package evals

// The oracle: the exclusion table this eval judges by, and the three assertions
// it applies to what the assembler wrote.
//
// The table below is TRANSCRIBED BY HAND from the record — itd-183's exclusion
// list, brief invariants 14 and 15, and adr-55 — and every row carries the
// source it was transcribed from. Transcription is the point. A table generated
// from the assembler's own include list would agree with the assembler by
// construction and could only ever confirm it, never falsify it; that is why
// nothing in this package imports the assembler's package, and why
// TestOracleImportsNothingFromTheAssembler checks rather than promises it.
//
// The disclosed residue: a hand-transcribed table can fall behind the exclusion
// list it mirrors. That staleness is bounded by the holed variant and by the
// anti-vacuity guard, never by the import check. Prose-borne warmth inside an
// included chapter carries no structural signal and stays the residue itd-183
// records.

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// assembleVerb is the verb this eval binds to. The eval binds to the VERB, not
// to the package, for the independence reason above; the name lives here, the
// single place a rename touches.
var assembleVerb = []string{"reading", "assemble"}

// The two artefacts an assembly writes into the operator-named directory.
const (
	bundleFile   = "bundle.json"
	manifestFile = "manifest.json"
)

// excludedKey is one frontmatter key the record refuses, with the source it is
// transcribed from.
type excludedKey struct {
	Key    string
	Source string
}

// excludedKeys is the key half of the exclusion floor.
var excludedKeys = []excludedKey{
	{Key: "origin", Source: "itd-183 exclusion list: `origin`, detected by frontmatter key"},
	{Key: "production_mode", Source: "itd-183 exclusion list: production mode, detected by frontmatter key"},
}

// excludedHeading is one heading the record refuses.
type excludedHeading struct {
	Heading string
	Source  string
}

// excludedHeadings is the heading half of the exclusion floor.
var excludedHeadings = []excludedHeading{
	{Heading: "Audit Notes", Source: "itd-183 exclusion list: Audit Notes, detected by heading"},
	{Heading: "Scope Condition Dispositions", Source: "itd-183 exclusion list: scope-condition dispositions"},
	{Heading: "Open Questions", Source: "itd-183: open questions and settled dead ends are deliberation"},
	{
		Heading: "Why This Matters",
		Source: "itd-183 assembler rule 2: a shipped intent travels as press release, acceptance " +
			"criteria, scope conditions, mechanism and spec_id; every other heading stays behind",
	},
}

// excludedFamily is one path no assembled item may come from.
type excludedFamily struct {
	// Path is repo-relative; an item at it or beneath it is a violation.
	Path string
	// Positions are the positions the exclusion binds at. Empty binds at every
	// position.
	Positions []string
	Source    string
}

// excludedFamilies is the path half of the exclusion floor.
var excludedFamilies = []excludedFamily{
	{
		Path: ".abcd/development/brief/03-evidence",
		Source: "framework 7.1 as itd-194 applies it at chapter grain: the evidence chapter is " +
			"verdict material, and a prior verdict is revision history — the ground the Audit " +
			"Notes exclusion rests on",
	},
	{Path: ".abcd/development/decisions", Source: "itd-183 exclusion list: decisions/"},
	{Path: ".abcd/development/roadmap/rfcs", Source: "itd-183 exclusion list: roadmap/rfcs/"},
	{Path: ".abcd/development/intents/superseded", Source: "itd-183 exclusion list: intents/superseded/"},
	{Path: ".abcd/development/plans", Source: "itd-183 exclusion list: plans/"},
	{Path: ".abcd/development/research/notes", Source: "itd-183 exclusion list: research/notes/"},
	{
		Path:      ".abcd/work/issues",
		Positions: []string{posWidening, posEntailment, posDetection},
		Source: "itd-183 exclusion list: work/issues/ in every state, reading records and " +
			"dispositions included, admission and selection grounds, and the lapse log. " +
			"Not at comparative: adr-2609021016272867 admits one derived widening run's " +
			"items there, so the container row withdraws and the six rows below name each " +
			"family individually — a narrower assertion, not a weaker one",
	},
	// The comparative position's ledger rows, one per family. They mirror the
	// rows the assembler derives from the ledger's own directory list, and they
	// are transcribed BY HAND for the reason this file's header states: a table
	// generated from the assembler's own would agree with it by construction.
	{
		Path:      ".abcd/work/issues/open",
		Positions: []string{posComparative},
		Source: "adr-2609021016272867: the comparative reading receives candidates and never " +
			"their fate, so every ledger family but the derived run is excluded by name",
	},
	{
		Path:      ".abcd/work/issues/resolved",
		Positions: []string{posComparative},
		Source:    "adr-2609021016272867: as above, for the resolved status directory",
	},
	{
		Path:      ".abcd/work/issues/wontfix",
		Positions: []string{posComparative},
		Source:    "adr-2609021016272867: as above, for the wontfix status directory",
	},
	{
		Path:      ".abcd/work/issues/dispositions",
		Positions: []string{posComparative},
		Source: "adr-2609021016272867: a candidate's disposition is the researcher's judgement " +
			"and is exactly what a reading must not see",
	},
	{
		Path:      ".abcd/work/issues/admissions",
		Positions: []string{posComparative},
		Source:    "adr-2609021016272867: an admission is the warm half of a candidate's fate",
	},
	{
		Path:      ".abcd/work/issues/surprises",
		Positions: []string{posComparative},
		Source:    "adr-2609021016272867: a surprise is the researcher's own act, recorded warm",
	},
	{Path: ".abcd/work/DECISIONS.md", Source: "itd-183 assembler rule 1: .abcd/ is excluded but for what the include list names"},
	{
		Path:   ".abcd/development/readings",
		Source: "itd-183: manifests are warm on the next run, so the instrument's own output is never its input",
	},
	{Path: ".abcd/.work.local", Source: "brief invariant 14: the local ledger side, which no reading consumes"},
	{Path: "agents", Source: "itd-183: the reading definitions are the instrument, not its input"},
	{Path: "evals", Source: "itd-183: the evals that guard this assembler are the instrument"},
	{Path: "internal/core/reading", Source: "itd-183: a reading never receives the include table that decides what it sees"},
	{
		Path:      ".abcd/development/intents/drafts",
		Positions: []string{posWidening, posComparative, posDetection},
		Source:    "itd-183's drafts asymmetry: a reading's object excludes what it exists to change",
	},
	{
		Path:      ".abcd/development/intents/planned",
		Positions: []string{posWidening, posComparative, posDetection},
		Source:    "itd-183's drafts asymmetry: a reading's object excludes what it exists to change",
	},
	{
		Path:      ".abcd/development/intents/shipped",
		Positions: []string{posWidening},
		Source: "itd-194: the framework's widening object and the readings companion's section " +
			"5.2 both state that object without the shipped intents, so the widening position " +
			"withdraws from the row and the floor asserts the withdrawal (iss-2609012259587904)",
	},
}

// bindsAt reports whether the exclusion binds at position p.
func (e excludedFamily) bindsAt(p string) bool {
	if len(e.Positions) == 0 {
		return true
	}
	for _, q := range e.Positions {
		if q == p {
			return true
		}
	}
	return false
}

// admittedRecordPath is one record path itd-183 names individually as included.
//
// This is the positive side of assembler rule 1, transcribed: no include may
// name a directory containing a record family, `.abcd/` is excluded wholesale,
// and the record paths a reading legitimately needs are named INDIVIDUALLY. So
// an assembled item under `.abcd/` that lies beneath none of these is a
// violation whether or not the exclusion table above happens to name it — which
// is what makes a record family invented after this table was written excluded
// by default rather than included by oversight.
type admittedRecordPath struct {
	Path      string
	Positions []string
	Source    string
}

// admittedRecordPaths is itd-183's include list, record paths only.
var admittedRecordPaths = []admittedRecordPath{
	{Path: ".abcd/development/brief/01-product", Source: "itd-183 include list; adr-55: the construal as it presently stands is committed record"},
	{Path: ".abcd/development/brief/02-constraints", Source: "itd-183 include list"},
	// itd-194: brief current text is the whole brief bar the evidence chapter
	// and bar the glossary, which is a record family with its own row
	// (framework 7.2, companion 5.2; the corrections ruling of 2026-09-02).
	{Path: ".abcd/development/brief/04-surfaces", Source: "itd-194: brief current text, the surfaces chapter"},
	{Path: ".abcd/development/brief/05-internals", Source: "itd-194: brief current text, the internals chapter"},
	{Path: ".abcd/development/brief/06-delivery", Source: "itd-194: brief current text, the delivery chapter"},
	{Path: ".abcd/development/brief/00-meta.md", Source: "itd-194: brief current text, the meta chapter's one file"},
	{Path: ".abcd/development/brief/glossary", Source: "itd-183 include list; adr-55: the glossary's committed terms"},
	{
		Path:      ".abcd/development/intents/shipped",
		Positions: []string{posEntailment, posComparative, posDetection},
		Source: "itd-183 include list, projected to the claim record; not at widening, whose " +
			"object neither design document states with the shipped intents in it (itd-194)",
	},
	{Path: ".abcd/development/intents/disciplines", Source: "itd-183 include list"},
	{Path: ".abcd/development/specs", Source: "itd-183 include list"},
	{
		Path:      ".abcd/development/intents/drafts",
		Positions: []string{posEntailment},
		Source:    "itd-183: the entailment reading includes the candidate set, because articulation precedes selection",
	},
	{
		Path:      ".abcd/development/intents/planned",
		Positions: []string{posEntailment},
		Source:    "itd-183: the entailment reading includes the candidate set, because articulation precedes selection",
	},
	{
		// The one record path under `.abcd/` that is admitted from the LEDGER,
		// and it is one RUN DIRECTORY rather than the family: the leaf bucket the
		// readings family keys on, which is what assembler rule 1 permits a row
		// to name. An item under any other run directory lies beneath no
		// individually named include and is a violation, which is how the
		// derivation's narrowing is held by path as well as by plant.
		Path:      ".abcd/work/issues/readings/" + derivedCandidateRun,
		Positions: []string{posComparative},
		Source: "adr-2609021016272867: at the comparative position, two body fields of ONE " +
			"derived widening run's items are that reading's candidate set",
	},
}

// admitsAt reports whether the record path is admitted at position p.
func (a admittedRecordPath) admitsAt(p string) bool {
	if len(a.Positions) == 0 {
		return true
	}
	for _, q := range a.Positions {
		if q == p {
			return true
		}
	}
	return false
}

// materialClass is one member of the closed material-class vocabulary and the
// paths that carry it: the directories it lives under (empty for the whole
// tree), the basename extensions or exact names that select it, and the
// basename suffixes that do.
//
// It is the KIND column of itd-183's include list, transcribed by hand like the
// exclusion table above and for the same reason: a mapping read out of the
// assembler's own table would agree with it by construction. Resolution is
// first-match in this order, which is the tie-break the include list itself
// declares — the first row that reaches a path owns it, and that is why the
// suffix row sits above the source row rather than beside it.
type materialClass struct {
	// Kind is the class the manifest must attest for a path this row reaches.
	Kind string
	// Under are the repo-relative directories the row reaches. Empty reaches the
	// whole tree, as the three root rows do.
	Under []string
	// Match are basename extensions (".md") or exact basenames ("Makefile").
	Match []string
	// Suffix are basename suffixes, matched case-sensitively.
	Suffix []string
	// Source is what the row was transcribed from.
	Source string
}

// materialClasses is the kind oracle: one row per member of the closed
// vocabulary, in resolution order.
//
// The manifest attests a material class PER ITEM, and until this table existed
// nothing read that attestation: the per-item kind added by itd-198 had no
// falsifier behind it, so a manifest naming every item `doc` was green
// (iss-2608312019547974). Brief invariant 16 is what makes that more than a
// tidiness point — an attestation states no more than its examination
// establishes, and a kind nobody checks is an assertion the artefact makes and
// the eval does not test.
var materialClasses = []materialClass{
	{
		Kind: "brief-section",
		Under: []string{
			".abcd/development/brief/01-product",
			".abcd/development/brief/02-constraints",
			".abcd/development/brief/04-surfaces",
			".abcd/development/brief/05-internals",
			".abcd/development/brief/06-delivery",
		},
		Match: []string{".md"},
		Source: "itd-183 include list, widened by itd-194 to brief current text: the product, " +
			"constraints, surfaces, internals and delivery chapters, admitted whole",
	},
	{
		// The meta chapter is the brief's one root file, so the row that reaches
		// it is bounded by an exact basename rather than by a directory. It sits
		// above the glossary row because the brief directory CONTAINS the
		// glossary, and the first row that reaches a path owns it.
		Kind:   "brief-section",
		Under:  []string{".abcd/development/brief"},
		Match:  []string{"00-meta.md"},
		Source: "itd-194: brief current text, the meta chapter's one file at the brief's root",
	},
	{
		Kind:   "glossary-term",
		Under:  []string{".abcd/development/brief/glossary"},
		Match:  []string{".md"},
		Source: "itd-183 include list; adr-55: the glossary's committed terms",
	},
	{
		Kind:   "discipline",
		Under:  []string{".abcd/development/intents/disciplines"},
		Match:  []string{".md"},
		Source: "itd-183 include list: a discipline is a standing commitment, named individually",
	},
	{
		// The candidate kind, reached at the comparative position alone. It sits
		// under the readings family's leaf bucket, which is the one ledger
		// directory an include row names (adr-2609021016272867).
		Kind:  "candidate",
		Under: []string{".abcd/work/issues/readings"},
		Match: []string{".md"},
		Source: "adr-2609021016272867: one derived widening run's items, projected to the " +
			"configuration and what admits it, are the comparative reading's candidate set",
	},
	{
		Kind: "intent-projection",
		Under: []string{
			".abcd/development/intents/shipped",
			".abcd/development/intents/drafts",
			".abcd/development/intents/planned",
		},
		Match:  []string{".md"},
		Source: "itd-183 assembler rule 2: an intent travels as its claim record, field by field",
	},
	{
		Kind:   "spec",
		Under:  []string{".abcd/development/specs"},
		Match:  []string{".md"},
		Source: "itd-183 include list: the design record a capability was built against",
	},
	{
		Kind:   "test",
		Suffix: []string{"_test.go"},
		Source: "itd-183 assembler rule 1: source and tests, counted apart because tests are the " +
			"largest single class and admitted identically; spc-68 selects them by basename suffix",
	},
	{
		Kind:   "source",
		Match:  []string{".go"},
		Source: "itd-183 assembler rule 1: the shipped tree is source and tests",
	},
	{
		Kind:   "doc",
		Match:  []string{".md"},
		Source: "itd-183 assembler rule 1: the shipped tree is the delivered documentation and root prose",
	},
	{
		Kind:   "config",
		Match:  []string{".json", ".yml", ".yaml", ".toml", ".mod", ".sum", "Makefile"},
		Source: "itd-183 assembler rule 1: the shipped tree is the delivered configuration and build files",
	},
}

// reaches reports whether the row claims rel.
func (m materialClass) reaches(rel string) bool {
	if len(m.Under) > 0 {
		inside := false
		for _, dir := range m.Under {
			if strings.HasPrefix(rel, dir+"/") {
				inside = true
				break
			}
		}
		if !inside {
			return false
		}
	}
	base := path.Base(rel)
	for _, s := range m.Suffix {
		if strings.HasSuffix(base, s) {
			return true
		}
	}
	for _, m := range m.Match {
		if strings.HasPrefix(m, ".") {
			if strings.EqualFold(path.Ext(base), m) {
				return true
			}
			continue
		}
		if base == m {
			return true
		}
	}
	return false
}

// classOf returns the material class the record says a path carries, and the
// row it was decided by. The first row that reaches the path owns it.
func classOf(rel string) (materialClass, bool) {
	for _, m := range materialClasses {
		if m.reaches(rel) {
			return m, true
		}
	}
	return materialClass{}, false
}

// violation is one finding: what rule was broken, at which position, and by
// what. Assertions RETURN violations rather than failing the test, so the
// negative control can demand exactly the two it expects.
type violation struct {
	Position string
	Rule     string
	Class    string
	Detail   string
	Source   string
}

// String renders a violation so the failure names the leaked sentinel's class
// token and the position at which it leaked.
func (v violation) String() string {
	head := fmt.Sprintf("[%s] %s: %s", v.Position, v.Rule, v.Detail)
	if v.Class != "" {
		head = fmt.Sprintf("[%s] %s: %s leaked (%s)", v.Position, v.Rule, sentinelPrefix+v.Class, v.Detail)
	}
	if v.Source != "" {
		head += " — " + v.Source
	}
	return head
}

// The three rules, named once so a test can demand a rule by name.
const (
	ruleSentinel       = "sentinel absence"
	ruleExcludedKey    = "field absence (excluded key)"
	ruleExcludedHeader = "field absence (excluded heading)"
	ruleFamily         = "family absence"
	ruleUnnamedRecord  = "family absence (record path no include names)"
	ruleMaterialClass  = "material-class attestation"
	ruleScanMark       = "scan-mark attestation"
)

// bundleItem and manifestItem are the two artefacts' item shapes, read as this
// eval needs them rather than as the assembler defines them.
type bundleItem struct {
	ItemKey string `json:"item_key"`
	Kind    string `json:"kind"`
	Text    string `json:"text"`
}

type manifestItem struct {
	ItemKey string `json:"item_key"`
	Path    string `json:"path"`
	Field   string `json:"field"`
	// Kind is the material class the manifest attests for this item. It is read
	// here so the attestation is checked rather than trusted; see
	// checkMaterialClass.
	Kind string `json:"kind"`
	// Scan is whether the exclusion floor EXAMINED this item. It is read here
	// because the manifest's key and heading exclusions are asserted for the
	// items marked `parsed` and for no other, so an oracle that ignored the
	// mark would hold the assertion over items no scan ever reached — which is
	// the artefact brief invariant 16 forbids (itd-194).
	Scan   string `json:"scan"`
	SHA256 string `json:"sha256"`
}

// manifestExclusion is one entry of the exclusion floor the manifest asserts.
type manifestExclusion struct {
	Rule   string `json:"rule"`
	Signal string `json:"signal"`
	Detail string `json:"detail"`
}

// assembled is one run's output as this eval reads it.
type assembled struct {
	Position      string
	BundleRaw     []byte
	ManifestRaw   []byte
	Items         []bundleItem
	ManifestItems []manifestItem
	// Exclusions is what the manifest CLAIMS it refused. A reader checks the
	// exclusions rather than trusting a disclosure, so what the claim omits is
	// as much this eval's business as what it asserts.
	Exclusions []manifestExclusion
}

// assemble runs the assembler over a materialised fixture at one position and
// returns what it wrote.
//
// The invocation is out of process, through the built binary, which is the
// first of the four mechanisms that hold the oracle independent: this package
// cannot read the include table even by accident, because it never links it.
// The output directory is outside the fixture repository, so a run leaves the
// fixture's own tiers untouched.
func assemble(t *testing.T, f fixture, position string) assembled {
	t.Helper()
	outDir := filepath.Join(t.TempDir(), "run-"+position)
	args := append(append([]string{}, assembleVerb...),
		"--position", position, "--target", "HEAD", "--out", outDir, "--dry-run")
	out, code := runIn(t, f.Root, []string{"HOME=" + f.Home}, args...)
	if code != 0 {
		t.Fatalf("`abcd %s` over the %s fixture exited %d\n%s",
			strings.Join(args, " "), f.Variant, code, out)
	}
	a := assembled{Position: position}
	a.BundleRaw = readArtefact(t, filepath.Join(outDir, bundleFile))
	a.ManifestRaw = readArtefact(t, filepath.Join(outDir, manifestFile))

	var bundle struct {
		Items []bundleItem `json:"items"`
	}
	if err := json.Unmarshal(a.BundleRaw, &bundle); err != nil {
		t.Fatalf("decoding the assembled input at %s: %v", position, err)
	}
	var manifest struct {
		Items      []manifestItem      `json:"items"`
		Exclusions []manifestExclusion `json:"exclusions"`
	}
	if err := json.Unmarshal(a.ManifestRaw, &manifest); err != nil {
		t.Fatalf("decoding the manifest at %s: %v", position, err)
	}
	a.Items, a.ManifestItems, a.Exclusions = bundle.Items, manifest.Items, manifest.Exclusions
	requireCarriers(t, a)
	return a
}

// requireCarriers is the floor under every absence assertion: the assembly must
// be non-empty, and every plant-bearing file the include list names must be in
// it at this position.
//
// The weaker floor — any item at all — cannot see the failure that matters. An
// assembler that stopped enumerating one whole class of source would drop the
// files carrying WARM-FIELD, two of the three WARM-KEY plants and both draft
// plants, leave nine walk-row items behind, and turn every assertion in this
// package green because there was nothing left to leak. That is the corpus
// argument one step further on: an absence assertion cannot see a corpus that
// lost its plants, and it cannot see an ASSEMBLY that lost their carriers
// either.
func requireCarriers(t *testing.T, a assembled) {
	t.Helper()
	if len(a.Items) == 0 {
		t.Fatalf("the assembly at %s passed no items; an absence assertion over an empty "+
			"bundle asserts nothing", a.Position)
	}
	seen := make(map[string]bool, len(a.ManifestItems))
	scanOf := map[string]string{}
	projected := map[string]map[string]bool{}
	for _, it := range a.ManifestItems {
		seen[it.Path] = true
		scanOf[it.Path] = it.Scan
		if it.Field == "" {
			continue
		}
		if projected[it.Path] == nil {
			projected[it.Path] = map[string]bool{}
		}
		projected[it.Path][it.Field] = true
	}
	for _, c := range carriers {
		if !c.reachesAt(a.Position) {
			continue
		}
		// The path first, because a missing path and an empty text are different
		// faults and the message should say which.
		if !seen[c.Path] {
			t.Fatalf("the assembly at %s does not carry %s (%s); %s",
				a.Position, c.Path, c.Why, c.stake())
		}
		// Then the mark, where the carrier declares one. A carrier that arrives
		// whole proves the item travelled; only the mark says whether an
		// examination stood behind the manifest's exclusion assertion over it.
		if c.Scan != "" && scanOf[c.Path] != c.Scan {
			t.Fatalf("the assembly at %s carries %s marked %q, and the record makes it %q (%s); "+
				"the mark is what tells a scan that ran from a scan that never ran",
				a.Position, c.Path, scanOf[c.Path], c.Scan, c.Why)
		}
		// Then the bytes. The manifest names what an assembly SAYS it passed; only
		// the bundle says what it actually passed, and an absence assertion over an
		// item whose text is empty is an absence assertion over nothing.
		for _, field := range c.Fields {
			if projected[c.Path][field] {
				continue
			}
			t.Fatalf("the assembly at %s carries %s but the manifest records no %q field for it, "+
				"so that field of the projection contract is unpinned; %s",
				a.Position, c.Path, field, c.stake())
		}
		for _, marker := range c.Markers {
			if bytes.Contains(a.BundleRaw, []byte(marker)) {
				continue
			}
			t.Fatalf("the assembly at %s names %s in its manifest, but the bundle does not carry "+
				"that file's own text (%q is absent); %s", a.Position, c.Path, marker, c.stake())
		}
	}
}

// requireOracleTables refuses a table that has changed size behind the
// assertions that consume it.
//
// A greater-than-zero floor on a table whose size is known is not a floor: a
// two-row table can halve under it, and a table emptied one row at a time never
// trips it at all. Every one of these tables is a declaration, so its size is a
// declaration too, and each of them is load-bearing — emptying carriers restores
// the vacuity the carriers floor exists to close, emptying excludedKeys and
// excludedHeadings disarms the field-absence check entirely, and emptying
// excludedFamilies disarms half the family-absence check.
//
// The count is deliberately duplicated here rather than derived from len(): a
// derived count agrees with the table by construction and could only confirm it,
// which is the same mistake as an oracle that reads the assembler's own table.
func requireOracleTables(t *testing.T) {
	t.Helper()
	for _, tbl := range []struct {
		name string
		got  int
		want int
	}{
		{"sentinelClasses", len(sentinelClasses), 21},
		{"carriers", len(carriers), 19},
		{"materialClasses", len(materialClasses), 11},
		{"holes", len(holes), 3},
		{"refusals", len(refusals), 8},
		{"excludedKeys", len(excludedKeys), 2},
		{"excludedHeadings", len(excludedHeadings), 4},
		{"excludedFamilies", len(excludedFamilies), 22},
		{"admittedRecordPaths", len(admittedRecordPaths), 13},
		{"coverage", len(coverage), 79},
	} {
		if tbl.got != tbl.want {
			t.Fatalf("the %s table holds %d row(s), and this eval is written against %d; "+
				"a table that changes size behind the assertions consuming it is how an "+
				"absence eval goes quietly vacuous, so update the declared count deliberately",
				tbl.name, tbl.got, tbl.want)
		}
	}
}

// readArtefact reads one of the two artefacts, failing if it is absent — the
// assembler is required to write both as separate files.
func readArtefact(t *testing.T, p string) []byte {
	t.Helper()
	data, err := os.ReadFile(p)
	if err != nil {
		t.Fatalf("the assembly wrote no %s: %v", filepath.Base(p), err)
	}
	return data
}

// checkSentinelAbsence is assertion 1: no planted token appears in the RAW
// serialisation of the assembled input.
//
// It is a content oracle, not a path oracle. Nothing in it mentions where a
// plant was, so a plant that moves is still caught — which is exactly the
// failure a path assertion misses.
func checkSentinelAbsence(a assembled) []violation {
	var out []violation
	raw := string(a.BundleRaw)
	for _, c := range sentinelClasses {
		if c.coldAt(a.Position) {
			continue
		}
		if strings.Contains(raw, c.Token()) {
			out = append(out, violation{
				Position: a.Position,
				Rule:     ruleSentinel,
				Class:    c.Name,
				Detail:   "found in the assembled input's raw serialisation",
				Source:   c.Why,
			})
		}
	}
	return out
}

// excludedKeyLine matches a frontmatter key at ANY indentation, quoted or bare.
// The indentation allowance is what spc-64 means by depth-agnostic: a warm field
// nested one level deeper is the same warm field, and a pattern anchored at
// column 0 would call it clean.
//
// The allowance is UNFALSIFIABLE through this eval and is recorded as such in
// the coverage matrix: the assembler fails closed on an indented excluded key
// that survives redaction, so a corpus carrying one makes the assembly exit
// non-zero rather than leak, and the branch is never reached. It is kept because
// spc-64 asks for it and because it costs nothing to be right if that refusal
// ever softens — not because it has been watched catch anything.
var excludedKeyLine = regexp.MustCompile(`^\s*["']?([A-Za-z_][A-Za-z0-9_-]*)["']?\s*:`)

// atxHeading matches an ATX heading, including the one-to-three-space indent
// CommonMark allows.
var atxHeading = regexp.MustCompile(`^[ ]{0,3}#{1,6}\s+(.*?)\s*#*\s*$`)

// The other spellings a heading arrives in. An ATX-only scan would report a
// bundle item carrying a setext-underlined or raw-HTML excluded heading as
// clean, which is assertion 2 satisfied completely by the leak it exists to
// catch. The state is unreachable while the assembler REFUSES those forms rather
// than redacting them — but that refusal is a mechanism of the assembler's, not
// a property of this oracle, and an oracle that can only see what the thing
// under test currently emits is an oracle that agrees with it by construction.
var (
	// setextUnderline matches the underline that turns the line above it into a
	// heading. A blank line above one underlines nothing, which is what keeps a
	// thematic break from reading as an empty heading.
	setextUnderline = regexp.MustCompile(`^[ ]{0,3}(?:=+|-+)[ \t]*$`)
	// rawHeadingOpen matches an element that OPENS a heading: an h1-h6 tag, or
	// any element carrying a heading role, which renders and is announced as a
	// heading while no h-tag appears. Matching the opening tag alone covers the
	// unclosed and multi-line forms — a document need not be well-formed for a
	// reader to see a heading in it.
	rawHeadingOpen = regexp.MustCompile(
		`(?is)<h[1-6](?:\s[^>]*)?/?>|<[a-z][a-z0-9-]*\s[^>]*role\s*=\s*["']?heading\b[^>]*>`)
	// blankLine bounds a raw heading's text where no closing tag does.
	blankLine = regexp.MustCompile(`\n[ \t]*\n`)
	// fenceDelimiter matches a fenced code block's delimiter. A heading inside a
	// fence is an EXAMPLE — a record template showing its own shape — and the
	// assembler leaves one alone on exactly that ground, so reporting it would be
	// a false red rather than a leak. Widening this scan to setext and raw HTML is
	// what makes the fence matter: those are the forms an example most often
	// carries.
	fenceDelimiter = regexp.MustCompile("^[ \t]*```")
	// headingMarkup is the markup a title carries without changing how it reads:
	// HTML comments and tags.
	headingMarkup = regexp.MustCompile(`(?s)<!--.*?-->|</?[A-Za-z][A-Za-z0-9-]*(?:\s[^>]*)?/?>`)
	// headingLink unwraps `[text](target)` to the text a reader sees.
	headingLink = regexp.MustCompile(`\[([^\]]*)\]\([^)]*\)`)
)

// headingsIn returns every heading an item's text carries, in each spelling,
// unnormalised. It is deliberately the broader of the two scans — the assembler
// redacts one form and refuses the rest, while this reports them all, which is
// the safe direction for a transcribed oracle.
func headingsIn(text string) []string {
	var out []string
	lines := strings.Split(text, "\n")
	fenced := fencedLines(lines)
	blockOpen, blockClose := firstDelimitedBlock(lines, fenced)
	for i, line := range lines {
		if fenced[i] {
			continue
		}
		if m := atxHeading.FindStringSubmatch(line); m != nil {
			out = append(out, m[1])
			continue
		}
		if i+1 >= len(lines) || fenced[i+1] || !setextUnderline.MatchString(lines[i+1]) {
			continue
		}
		if strings.TrimSpace(line) == "" {
			continue // a blank line underlines nothing
		}
		// Two shapes inside a record's own frontmatter that a setext scan reads as
		// headings and no reader does. The block's closing `---` sits directly
		// under the block's last line: it CLOSES a block, it underlines nothing.
		// And a line indented inside the block is a block scalar's continuation,
		// which is a value rather than a title — the shape this corpus grew when
		// the block-scalar plant went in.
		if blockClose >= 0 && i+1 == blockClose {
			continue
		}
		if blockOpen >= 0 && i > blockOpen && (blockClose < 0 || i < blockClose) &&
			strings.HasPrefix(line, " ") {
			continue
		}
		out = append(out, line)
	}
	// The raw-HTML scan runs over the text JOINED with the fenced lines blanked,
	// because an opening tag, its text and its close need not share a line.
	unfenced := make([]string, len(lines))
	for i, line := range lines {
		if !fenced[i] {
			unfenced[i] = line
		}
	}
	scan := strings.Join(unfenced, "\n")
	for _, loc := range rawHeadingOpen.FindAllStringIndex(scan, -1) {
		rest := scan[loc[1]:]
		if cut := strings.Index(rest, "<"); cut >= 0 {
			rest = rest[:cut]
		}
		if cut := blankLine.FindStringIndex(rest); cut != nil {
			rest = rest[:cut[0]]
		}
		out = append(out, rest)
	}
	return out
}

// fencedLines reports, per line, whether it sits inside a fenced code block. The
// delimiter itself counts as inside.
func fencedLines(lines []string) []bool {
	mask := make([]bool, len(lines))
	inside := false
	for i, line := range lines {
		if fenceDelimiter.MatchString(line) {
			inside = !inside
			mask[i] = true
			continue
		}
		mask[i] = inside
	}
	return mask
}

// firstDelimitedBlock locates a document's opening `---` block: the line that
// opens it and the line that closes it, or -1 for either. The block must be the
// document's first non-blank content, so a thematic break further down is not
// mistaken for one.
func firstDelimitedBlock(lines []string, fenced []bool) (int, int) {
	open := -1
	for i, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}
		if !fenced[i] && strings.TrimSpace(line) == "---" {
			open = i
		}
		break
	}
	if open < 0 {
		return -1, -1
	}
	for i := open + 1; i < len(lines); i++ {
		if !fenced[i] && strings.HasPrefix(strings.TrimSpace(lines[i]), "---") {
			return open, i
		}
	}
	return open, -1
}

// headingTitle reduces a heading to the text a reader sees: markup removed, link
// wrappers unwrapped, emphasis and code marks dropped, whitespace collapsed.
func headingTitle(raw string) string {
	s := headingMarkup.ReplaceAllString(raw, "")
	s = headingLink.ReplaceAllString(s, "$1")
	s = strings.Trim(s, " \t*_`#")
	return strings.Join(strings.Fields(s), " ")
}

// checkFieldAbsence is assertion 2: a recursive walk of the parsed documents
// reports no key on the excluded-key list at any depth, and no excluded heading
// in any projected body.
//
// Two walks, because a key can be a key in two different documents. The first
// is over the artefacts' own JSON, at any nesting depth. The second is over each
// item's TEXT, which is where a record's frontmatter actually travels, and it
// too matches at any depth.
func checkFieldAbsence(a assembled) []violation {
	var out []violation
	keys := map[string]excludedKey{}
	for _, k := range excludedKeys {
		keys[strings.ToLower(k.Key)] = k
	}

	for _, artefact := range []struct {
		name string
		raw  []byte
	}{{"the assembled input", a.BundleRaw}, {"the manifest", a.ManifestRaw}} {
		var doc any
		if err := json.Unmarshal(artefact.raw, &doc); err != nil {
			continue
		}
		for _, k := range walkJSONKeys(doc) {
			if hit, ok := keys[strings.ToLower(k)]; ok {
				out = append(out, violation{
					Position: a.Position,
					Rule:     ruleExcludedKey,
					Detail:   fmt.Sprintf("%s carries the key %q", artefact.name, hit.Key),
					Source:   hit.Source,
				})
			}
		}
	}

	// A manifest item's `field` NAMES the projected field, so an excluded key
	// that got itself projected shows up there as a value rather than a key.
	for _, it := range a.ManifestItems {
		if hit, ok := keys[strings.ToLower(it.Field)]; ok {
			out = append(out, violation{
				Position: a.Position,
				Rule:     ruleExcludedKey,
				Detail:   fmt.Sprintf("the manifest projects the field %q as item %s", hit.Key, it.ItemKey),
				Source:   hit.Source,
			})
		}
	}

	// The key and heading halves are held over the items the manifest marks
	// `parsed` and over no other. The floor's two signals are read by a scan,
	// and an item the scan never reached carries no claim that it was clean —
	// so an unscanned item carrying an excluded heading is disclosure working,
	// not a leak, and holding it to the same rule would report the disclosure
	// as the failure. The inverse is the assertion that matters: an item the
	// manifest marks `parsed` is one the assembler says it examined, so a hit
	// there is a leak under an attestation (itd-194; brief invariant 16).
	// The set is built from the mark that EXCUSES an item, never from the one
	// that holds it. An item the manifest does not mention at all is held, and
	// so is one whose mark is missing or unrecognised: only an explicit
	// `unscanned` says the assembler never claimed to have examined it. Reading
	// it the other way round — holding the items marked `parsed` — would let a
	// manifest disarm this assertion by omission, and would silently disarm it
	// for a caller that has the bundle and not the manifest.
	unscanned := make(map[string]bool, len(a.ManifestItems))
	for _, it := range a.ManifestItems {
		if it.Scan == "unscanned" {
			unscanned[it.ItemKey] = true
		}
	}

	for _, it := range a.Items {
		if unscanned[it.ItemKey] {
			continue
		}
		for _, line := range strings.Split(it.Text, "\n") {
			if m := excludedKeyLine.FindStringSubmatch(line); m != nil {
				if hit, ok := keys[strings.ToLower(m[1])]; ok {
					out = append(out, violation{
						Position: a.Position,
						Rule:     ruleExcludedKey,
						Detail:   fmt.Sprintf("item %s carries the key %q", it.ItemKey, hit.Key),
						Source:   hit.Source,
					})
				}
			}
		}
		// The heading scan is over the item's whole text rather than per line,
		// because a heading need not be spelled on one: a setext heading is two
		// lines and a raw-HTML one can be any number.
		for _, raw := range headingsIn(it.Text) {
			title := headingTitle(raw)
			if title == "" {
				continue
			}
			for _, h := range excludedHeadings {
				if strings.EqualFold(title, h.Heading) {
					out = append(out, violation{
						Position: a.Position,
						Rule:     ruleExcludedHeader,
						Detail: fmt.Sprintf("item %s carries the heading %q, spelled %q",
							it.ItemKey, h.Heading, strings.TrimSpace(raw)),
						Source: h.Source,
					})
				}
			}
		}
	}
	return out
}

// walkJSONKeys returns every object key in a decoded document, at any depth.
func walkJSONKeys(doc any) []string {
	var out []string
	var walk func(any)
	walk = func(v any) {
		switch t := v.(type) {
		case map[string]any:
			for k, child := range t {
				out = append(out, k)
				walk(child)
			}
		case []any:
			for _, child := range t {
				walk(child)
			}
		}
	}
	walk(doc)
	return out
}

// checkFamilyAbsence is assertion 3: no item's recorded path lies inside an
// excluded record family, and no item under `.abcd/` lies outside the record
// paths the include list names individually.
//
// The second half is assembler rule 1 as a positive test, which is what catches
// a record family invented after this table was written.
func checkFamilyAbsence(a assembled) []violation {
	var out []violation
	for _, it := range a.ManifestItems {
		rel := path.Clean(it.Path)
		for _, fam := range excludedFamilies {
			if !fam.bindsAt(a.Position) {
				continue
			}
			if rel == fam.Path || strings.HasPrefix(rel, fam.Path+"/") {
				out = append(out, violation{
					Position: a.Position,
					Rule:     ruleFamily,
					Detail:   fmt.Sprintf("item %s comes from %s, inside %s", it.ItemKey, rel, fam.Path),
					Source:   fam.Source,
				})
			}
		}
		if !strings.HasPrefix(rel, ".abcd/") {
			continue
		}
		named := false
		for _, adm := range admittedRecordPaths {
			if !adm.admitsAt(a.Position) {
				continue
			}
			if rel == adm.Path || strings.HasPrefix(rel, adm.Path+"/") {
				named = true
				break
			}
		}
		if !named {
			out = append(out, violation{
				Position: a.Position,
				Rule:     ruleUnnamedRecord,
				Detail: fmt.Sprintf("item %s comes from %s, a record path no include names at this position",
					it.ItemKey, rel),
				Source: "itd-183 assembler rule 1: .abcd/ is excluded wholesale and the record paths a " +
					"reading needs are named individually, so a family added later is excluded by construction",
			})
		}
	}
	return out
}

// checkMaterialClass is the kind attestation: every manifest item's `kind`
// names the material class the record says its path carries, and the bundle
// item beside it carries the same class.
//
// The bundle half matters because the two artefacts travel apart. The manifest
// stays with the auditor and the bundle goes to the reader, so a kind correct
// in the manifest and wrong in the bundle is a reading told its material is
// something it is not, with the auditor's copy attesting otherwise.
//
// It is deliberately NOT part of checkReadBlock: the three read-block
// assertions are about what was selected, and this is about how the artefact
// describes what was selected. Keeping it out also keeps the negative control's
// exact-two expectation about leaks alone.
func checkMaterialClass(a assembled) []violation {
	var out []violation
	kindByKey := make(map[string]string, len(a.Items))
	for _, it := range a.Items {
		kindByKey[it.ItemKey] = it.Kind
	}
	for _, it := range a.ManifestItems {
		want, ok := classOf(it.Path)
		if !ok {
			out = append(out, violation{
				Position: a.Position,
				Rule:     ruleMaterialClass,
				Detail: fmt.Sprintf("item %s comes from %s, which no material class in the record's "+
					"vocabulary reaches", it.ItemKey, it.Path),
				Source: "itd-183: the include list decides both what is admitted and the class it " +
					"is admitted as, so a path with no class is a path no include names",
			})
			continue
		}
		if it.Kind != want.Kind {
			out = append(out, violation{
				Position: a.Position,
				Rule:     ruleMaterialClass,
				Detail: fmt.Sprintf("the manifest attests item %s (%s) as %q, and the record makes it %q",
					it.ItemKey, it.Path, it.Kind, want.Kind),
				Source: want.Source,
			})
		}
		if got := kindByKey[it.ItemKey]; got != it.Kind {
			out = append(out, violation{
				Position: a.Position,
				Rule:     ruleMaterialClass,
				Detail: fmt.Sprintf("item %s travels to the reading as %q and is attested to the "+
					"auditor as %q", it.ItemKey, got, it.Kind),
				Source: "brief invariant 16: an attestation never states more than the examination " +
					"behind it establishes, and the two artefacts describe one selection",
			})
		}
	}
	return out
}

// checkScanMarks is the scan attestation: every manifest item carries a mark,
// and the mark is the one the record says its path carries.
//
// The expectation is TRANSCRIBED from the include table's declaration rather
// than read from it, like every other oracle here: the floor's key and heading
// signals are record shapes only a markdown file carries, so a `.md` item is
// `parsed` and every other item is `unscanned`. An oracle that read Row.Scan
// would agree with the assembler by construction and could only ever confirm
// it.
//
// It is what makes the disclosure falsifiable. An assembler that stamped
// `parsed` on every candidate would satisfy the decoder, the size report and
// every absence assertion here, while the manifest's key and heading
// exclusions rested on a scan that never ran over the source and test items —
// which is the artefact adr-56 exists to forbid.
func checkScanMarks(a assembled) []violation {
	var out []violation
	for _, it := range a.ManifestItems {
		want := "unscanned"
		if strings.EqualFold(path.Ext(it.Path), ".md") {
			want = "parsed"
		}
		switch {
		case it.Scan == "":
			out = append(out, violation{
				Position: a.Position,
				Rule:     ruleScanMark,
				Detail: fmt.Sprintf("item %s (%s) carries no scan mark, so the manifest cannot "+
					"say whether the exclusion floor examined it", it.ItemKey, it.Path),
				Source: "brief invariant 16: an attestation never states more than the examination " +
					"behind it establishes, and an unmarked item's exclusion assertion states " +
					"exactly that",
			})
		case it.Scan != want:
			out = append(out, violation{
				Position: a.Position,
				Rule:     ruleScanMark,
				Detail: fmt.Sprintf("the manifest marks item %s (%s) %q, and the record makes it %q",
					it.ItemKey, it.Path, it.Scan, want),
				Source: "itd-194: the floor's key and heading signals are record shapes only a " +
					"markdown file carries, so a markdown item is parsed and every other item " +
					"travels whole and marked unscanned",
			})
		}
	}
	return out
}

// checkReadBlock is the three assertions together.
func checkReadBlock(a assembled) []violation {
	var out []violation
	out = append(out, checkSentinelAbsence(a)...)
	out = append(out, checkFieldAbsence(a)...)
	out = append(out, checkFamilyAbsence(a)...)
	return out
}

// reportViolations renders a violation list for a failure message.
func reportViolations(vs []violation) string {
	lines := make([]string, 0, len(vs))
	for _, v := range vs {
		lines = append(lines, "  "+v.String())
	}
	return strings.Join(lines, "\n")
}
