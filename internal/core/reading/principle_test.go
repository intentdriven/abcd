package reading

import (
	"fmt"
	"strings"
	"testing"
)

// The knowledge record as a read object (spc-2609020626042471): a principle
// travels as its H1 title and its `**The rule.**` paragraph, and its four claim
// keys and every citation stay behind.

const (
	principleRel = ".abcd/development/principles/fix-the-detector.md"
	// The three homes a principle's genealogy has: a claim key, the reasoning
	// below the statement, and a link target inside the statement itself.
	sentinelPrincipleKey  = "SENTINEL-PRINCIPLE-KEY"
	sentinelPrincipleWhy  = "SENTINEL-PRINCIPLE-WHY"
	sentinelPrincipleLink = "SENTINEL-PRINCIPLE-LINK"
	// The statement's own words, which a reading must receive.
	principleStatement = "After a review, the unit of fix is the detector"
)

// principleDoc is a typed principle carrying genealogy in every home it has.
func principleDoc(rule string) string {
	return "---\nid: prn-fix-the-detector\nclaim_type: causal\nreference: \"abcd lint\"\n" +
		"comparison: \"" + sentinelPrincipleKey + " hand fixes against a detector.\"\n" +
		"evidence: [itd-181, cond-2608311949582375]\n---\n\n" +
		"# Fix the detector\n\n" +
		"**The rule.** " + rule + "\n\n" +
		"**Why.** Because adr-1 said so, " + sentinelPrincipleWhy + ".\n\n" +
		"**Bounds.** Nothing else travels.\n"
}

const defaultRule = principleStatement + ", as [the ruling](" + sentinelPrincipleLink + ") says,\n" +
	"not the finding."

// principleFixture is the base fixture with one typed principle committed.
func principleFixture(t *testing.T, rule string) string {
	t.Helper()
	root := fixtureRepo(t)
	writeFile(t, root, principleRel, principleDoc(rule))
	gitCommitAll(t, root)
	return root
}

// TestLabelledParagraphResolves: a field naming a label resolves as the first
// paragraph opening with it, label removed, and nothing after it.
func TestLabelledParagraphResolves(t *testing.T) {
	text, ok, err := projectField(principleRel, principleDoc(defaultRule), "The rule")
	if err != nil || !ok {
		t.Fatalf("projectField(The rule) = %q, %v, %v", text, ok, err)
	}
	if !strings.Contains(text, principleStatement) || !strings.Contains(text, "not the finding.") {
		t.Errorf("the statement did not travel whole: %q", text)
	}
	for _, gone := range []string{"**The rule.**", "**Why.**", sentinelPrincipleWhy, "Bounds", "claim_type"} {
		if strings.Contains(text, gone) {
			t.Errorf("the projected statement carries %q: %q", gone, text)
		}
	}
	// A document without the paragraph contributes no item.
	if _, ok, _ := projectField(principleRel, "# A principle\n\nProse only.\n", "The rule"); ok {
		t.Error("a document with no labelled paragraph projected one")
	}
}

// TestLabelledParagraphCarriesTheTitle: a rule without its name is not readable
// cold, so the H1 title is placed above the statement.
func TestLabelledParagraphCarriesTheTitle(t *testing.T) {
	text, _, _ := projectField(principleRel, principleDoc(defaultRule), "The rule")
	if !strings.HasPrefix(text, "# Fix the detector\n\n"+principleStatement) {
		t.Errorf("the projection does not open with the title above the statement: %q", text)
	}
}

// TestLinksUnwrapInTheStatement: a link target is a citation and the label is
// prose, so the target stays behind and the label travels.
func TestLinksUnwrapInTheStatement(t *testing.T) {
	text, _, _ := projectField(principleRel, principleDoc(defaultRule), "The rule")
	if strings.Contains(text, sentinelPrincipleLink) || strings.Contains(text, "](") {
		t.Errorf("the link target travelled: %q", text)
	}
	if !strings.Contains(text, "as the ruling says") {
		t.Errorf("the link's label did not travel as prose: %q", text)
	}
}

// TestPrincipleProjectsItsStatementOnly is ac-6's item half: at every position
// whose entry admits `principle`, the principle is ONE projected item naming its
// statement field, and neither its keys nor its citations reach the bundle. At
// comparative the row is not admitted at all.
func TestPrincipleProjectsItsStatementOnly(t *testing.T) {
	root := principleFixture(t, defaultRule)
	for _, p := range []Position{PositionWidening, PositionEntailment, PositionDetection} {
		res := assembleFixture(t, root, p)
		var items []ManifestItem
		for _, it := range res.Manifest.Items {
			if it.Path == principleRel {
				items = append(items, it)
			}
		}
		if len(items) != 1 || items[0].Field != "The rule" || items[0].Kind != KindPrinciple || items[0].Scan != ScanParsed {
			t.Fatalf("at %s the principle travelled as %+v, want one parsed %q item of kind %q",
				p, items, "The rule", KindPrinciple)
		}
		text := bundleText(res.Bundle)
		if !strings.Contains(text, principleStatement) {
			t.Errorf("at %s the statement did not reach the bundle", p)
		}
		for _, gone := range []string{sentinelPrincipleKey, sentinelPrincipleWhy, sentinelPrincipleLink,
			"cond-2608311949582375", "claim_type"} {
			if strings.Contains(text, gone) {
				t.Errorf("at %s the bundle carries %q", p, gone)
			}
		}
	}
	res := assembleFixture(t, root, PositionComparative)
	for _, it := range res.Manifest.Items {
		if it.Path == principleRel || it.Kind == KindPrinciple {
			t.Errorf("the comparative assembly carries a principle item: %+v", it)
		}
	}
}

// TestPrincipleIsAScopeToken: `principle` is a kind a committed entry may name,
// the table admits it only where an entry does, and an entry naming the kind
// alone is handed the knowledge record and nothing else.
func TestPrincipleIsAScopeToken(t *testing.T) {
	found := false
	for _, k := range Kinds() {
		if k == KindPrinciple && string(k) == "principle" {
			found = true
		}
	}
	if !found {
		t.Fatalf("Kinds() = %v, which does not carry %q", Kinds(), "principle")
	}

	root := principleFixture(t, defaultRule)
	writeFile(t, root, PresetConfigPath, presetFileNaming(`"principle"`))
	gitCommitAll(t, root)
	res := assembleFixture(t, root, PositionDetection)
	if len(res.Manifest.Items) != 1 || res.Manifest.Items[0].Path != principleRel {
		t.Errorf("an entry naming the principle kind alone was handed %v", itemPaths(res.Manifest))
	}

	// An entry that does not name the kind hands no principle, whatever else it
	// names: the table ADMITS, the entry SELECTS.
	writeFile(t, root, PresetConfigPath, presetFileNaming(`"spec"`))
	gitCommitAll(t, root)
	res = assembleFixture(t, root, PositionDetection)
	for _, it := range res.Manifest.Items {
		if it.Kind == KindPrinciple {
			t.Errorf("an entry not naming the principle kind was handed %s", it.Path)
		}
	}
}

// presetFileNaming renders a preset file whose every assembling position names
// only the given kinds (comparative keeps the discipline its criteria need).
func presetFileNaming(kinds string) string {
	var entries []string
	for _, p := range AssemblingPositions() {
		k := kinds
		if p == PositionComparative {
			k = `"discipline"`
		}
		entries = append(entries, v2Entry(string(p), k, "", "", 1_000_000))
	}
	return "{\n  \"schema_version\": 2,\n  \"positions\": {\n" + strings.Join(entries, ",\n") + "\n  }\n}\n"
}

// TestPrincipleRowExcludesComparative: at comparative the include table is the
// whole account and admits the candidates and the criteria alone, so the
// principle row is admitted at the three assembling positions and no other.
func TestPrincipleRowExcludesComparative(t *testing.T) {
	var rows []Row
	for _, r := range Table {
		if r.Kind == KindPrinciple {
			rows = append(rows, r)
		}
	}
	if len(rows) != 1 {
		t.Fatalf("the table carries %d principle row(s), want 1", len(rows))
	}
	r := rows[0]
	if r.Source != ".abcd/development/principles" || r.Store != "prn" || len(r.Fields) != 1 || r.Fields[0] != "The rule" || r.Scan != ScanParsed {
		t.Errorf("the principle row is %+v", r)
	}
	for _, p := range Positions() {
		want := p != PositionComparative
		if r.AdmittedAt(p) != want {
			t.Errorf("the principle row admitted at %s = %v, want %v", p, r.AdmittedAt(p), want)
		}
	}
}

// TestManifestAssertsPrincipleExclusions is ac-6's exclusion half: the four
// claim keys and the citation entry are asserted in every manifest, so a
// reader checks the withholding rather than trusting it.
func TestManifestAssertsPrincipleExclusions(t *testing.T) {
	root := principleFixture(t, defaultRule)
	res := assembleFixture(t, root, PositionDetection)
	has := map[string]bool{}
	for _, e := range res.Manifest.Exclusions {
		has[e.Signal+"|"+e.Detail+"|"+e.Rule] = true
	}
	for _, k := range []string{"claim_type", "reference", "comparison", "evidence"} {
		if !has["frontmatter key|"+k+"|field projection"] {
			t.Errorf("the manifest does not assert the %s key's exclusion", k)
		}
	}
	if !has["citation|record handles and links in a principle|the statement is knowledge and the citations are genealogy"] {
		t.Errorf("the manifest does not assert the citation exclusion: %+v", res.Manifest.Exclusions)
	}
}

// TestPrincipleItemCarryingAHandleRefuses: the manifest's citation assertion is
// checked rather than trusted, so a principle whose statement still carries a
// record handle after projection refuses the assembly and names both.
func TestPrincipleItemCarryingAHandleRefuses(t *testing.T) {
	root := principleFixture(t, principleStatement+", as adr-1 ruled.")
	_, err := Assemble(AssembleRequest{RepoRoot: root, Position: PositionDetection, Target: "HEAD", DryRun: true})
	if err == nil {
		t.Fatal("a principle whose statement carries a record handle assembled")
	}
	for _, want := range []string{principleRel, "adr-1"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("the refusal does not name %s: %v", want, err)
		}
	}

	// The handle below the statement is genealogy the projection already keeps
	// out, so it refuses nothing.
	root = principleFixture(t, defaultRule)
	assembleFixture(t, root, PositionDetection)
}

// TestManifestAtTheOldSchemaVersionIsRefused: the kind vocabulary is closed and
// gains `principle`, so the shape version moves, and a manifest stamped at the
// previous version is refused rather than read.
func TestManifestAtTheOldSchemaVersionIsRefused(t *testing.T) {
	if SchemaVersion != 11 {
		t.Fatalf("SchemaVersion = %d, want 11: the closed kind vocabulary gains principle", SchemaVersion)
	}
	root := fixtureRepo(t)
	res := assembleFixture(t, root, PositionDetection)
	raw, err := EncodeManifest(res.Manifest)
	if err != nil {
		t.Fatal(err)
	}
	old := strings.Replace(string(raw), fmt.Sprintf("\"schema_version\": %d", SchemaVersion), "\"schema_version\": 10", 1)
	if old == string(raw) {
		t.Fatal("the replacement did nothing, so this case would test nothing")
	}
	if _, err := DecodeManifest([]byte(old)); err == nil {
		t.Error("a manifest at schema version 10 decoded")
	}
}
