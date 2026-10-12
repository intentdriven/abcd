package lifeboat

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// briefMetaRelPath is the brief file that calls the mapping table "the
// contract". The table rendered there must equal renderMapping().
const briefMetaRelPath = ".abcd/development/brief/00-meta.md"

// repoRoot walks up from the test's working directory to the directory holding
// go.mod.
func repoRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("go.mod not found above the test working directory")
		}
		dir = parent
	}
}

// TestBriefCarriesTheRenderedMappingTable is the anti-drift detector: the brief
// document and the Go table are one contract, and the code is its source of
// truth. Editing either alone fails here.
func TestBriefCarriesTheRenderedMappingTable(t *testing.T) {
	path := filepath.Join(repoRoot(t), briefMetaRelPath)
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", briefMetaRelPath, err)
	}
	doc := string(raw)

	begin := strings.Index(doc, MarkerBegin)
	end := strings.Index(doc, MarkerEnd)
	if begin < 0 || end < 0 {
		t.Fatalf("%s does not carry the generated-mapping markers %q / %q; "+
			"the brief calls the mapping table the contract, so it must render it",
			briefMetaRelPath, MarkerBegin, MarkerEnd)
	}
	if end < begin {
		t.Fatalf("%s: end marker precedes begin marker", briefMetaRelPath)
	}

	got := strings.TrimSpace(doc[begin+len(MarkerBegin) : end])
	want := strings.TrimSpace(renderMapping())
	if got != want {
		t.Errorf("%s has drifted from lifeboat.Table.\n\n--- brief has ---\n%s\n\n--- renderMapping() wants ---\n%s",
			briefMetaRelPath, got, want)
	}
}

// TestTiersDegradeMonotonically holds the claim the tiering makes: a richer
// tier never grounds a section worse than a poorer one. A row that violates
// this is a mistake in the hypothesis, not a discovery.
func TestTiersDegradeMonotonically(t *testing.T) {
	for _, m := range Table {
		prev := Status("")
		for _, tier := range allTiers() {
			s := m.StatusAt(tier)
			if !s.Valid() {
				t.Errorf("%s at tier %s: %q is not a member of the status enum", m.Section, tier, s)
				continue
			}
			if prev != "" && s.rank() < prev.rank() {
				t.Errorf("%s: tier %s grounds to %q, worse than the poorer tier's %q — tiers must degrade, not improve downward",
					m.Section, tier, s, prev)
			}
			prev = s
		}
	}
}

// TestGraveyardIsTheOnlySectionGroundedAtTierGit pins the thesis. Both the
// package doc and the brief assert that the graveyard is the one section a
// repository grounds from git alone — what a project abandoned is written into
// its history whether or not anyone wrote it down, and that is why the
// graveyard earns a section of its own.
//
// The claim is prose, so nothing but a test keeps it true. If a future row
// grounds some other section at Tier 0, this fails and forces the prose to be
// corrected rather than quietly becoming a lie.
func TestGraveyardIsTheOnlySectionGroundedAtTierGit(t *testing.T) {
	var grounded []Section
	for _, m := range Table {
		if m.Git == StatusGrounded {
			grounded = append(grounded, m.Section)
		}
	}
	if len(grounded) != 1 || grounded[0] != "graveyard" {
		t.Errorf("the brief and the package doc both claim graveyard is the ONLY section grounded at tier %s, "+
			"but the table grounds these there: %v — either the table is wrong or the claim is; fix one",
			TierGit, grounded)
	}
}

// TestPersonasPredictionMatchesTheProbe holds the personas row to what the probe
// measures (iss-2610102343100369). The native personas adapter grounds the
// section from an authored brief file, is partial from a stub and blank with
// none, so the best status abcd-native can reach is the best of those three; no
// poorer tier has a personas adapter, so git and conventions stay blank. A row
// that predicts a different ceiling is a prediction the probe has already
// overturned, and the table is what loses.
func TestPersonasPredictionMatchesTheProbe(t *testing.T) {
	const section Section = "product/personas"
	const file = ".abcd/development/brief/01-product/05-personas.md"
	prose := "Who the work is for is written here by the people who know it: the " +
		"roles the product serves, what each needs from it, and why the work is " +
		"shaped for them, so a reader meets the audience without reconstructing " +
		"it from the history or asking the people who settled it.\n"

	fixtures := map[string]map[string]string{
		"authored": {file: "# Personas\n\n" + prose},
		"stub":     {file: "# Personas\n\nTODO.\n"},
		"none":     {"README.md": "# Project\n"},
	}
	want := map[string]Status{"authored": StatusGrounded, "stub": StatusPartial, "none": StatusBlank}

	ceiling := StatusBlank
	for name, tree := range fixtures {
		dir := t.TempDir()
		writeTree(t, dir, tree)
		ctx, err := newSourceContext(dir)
		if err != nil {
			t.Fatal(err)
		}
		ev := nativeSourceForSection(t, section).Probe(ctx)
		ctx.Close()
		if ev.Status != want[name] {
			t.Errorf("%s probe with %s brief file = %s, want %s", section, name, ev.Status, want[name])
		}
		if ev.Status.rank() > ceiling.rank() {
			ceiling = ev.Status
		}
	}

	for _, s := range append(gitSources(), conventionSources()...) {
		if s.Section() == section {
			t.Errorf("a tier-%s source speaks for %s; the blank predictions below abcd-native need revisiting", s.Tier(), section)
		}
	}

	for _, m := range Table {
		if m.Section != section {
			continue
		}
		if m.Native != ceiling {
			t.Errorf("lifeboat.Table predicts %s at tier %s as %q, but the probe reaches %q there "+
				"(grounded from an authored brief file, partial from a stub, blank with none)",
				section, TierNative, m.Native, ceiling)
		}
		if m.Git != StatusBlank || m.Conventions != StatusBlank {
			t.Errorf("lifeboat.Table predicts %s as %q at %s and %q at %s; no adapter below %s reads it, so both are blank",
				section, m.Git, TierGit, m.Conventions, TierConventions, TierNative)
		}
		return
	}
	t.Fatalf("lifeboat.Table has no %s row", section)
}

// TestEveryBriefSectionHasARow is the exhaustiveness detector. A brief section
// with no row in the table is worse than one the probe reports as blank: a blank
// names a question a human must answer, whereas a missing row is never reported
// at all. The contract claims to cover the brief, so this walks the real brief
// tree and insists every content file is accounted for — either by its own row,
// or by a row that covers its whole chapter directory.
func TestEveryBriefSectionHasARow(t *testing.T) {
	briefDir := filepath.Join(repoRoot(t), ".abcd/development/brief")

	covered := func(rel string) bool {
		for _, m := range Table {
			p := strings.TrimPrefix(m.LifeboatPath, "brief/")
			if p == m.LifeboatPath {
				continue // not a brief row (graveyard/, rescue/, docs/adrs/, ...)
			}
			if p == rel {
				return true // an exact per-file row
			}
			if strings.HasSuffix(p, "/") && strings.HasPrefix(rel, p) {
				return true // a chapter-directory row subsumes it
			}
		}
		return false
	}

	err := filepath.WalkDir(briefDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || !strings.HasSuffix(d.Name(), ".md") {
			return nil
		}
		rel, err := filepath.Rel(briefDir, path)
		if err != nil {
			return err
		}
		// 00-meta.md is the contract's own home, and a README is a chapter
		// index rather than a section of the brief's content.
		if rel == "00-meta.md" || rel == "README.md" || d.Name() == "README.md" {
			return nil
		}
		if !covered(filepath.ToSlash(rel)) {
			t.Errorf("brief section %q has no row in lifeboat.Table — it can never be reported, "+
				"not even as blank; give it a row or a chapter-directory row that covers it", rel)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk brief: %v", err)
	}
}

// TestTableIsWellFormed guards the table's own integrity: sections and lifeboat
// paths are unique, and no row is missing its evidence.
func TestTableIsWellFormed(t *testing.T) {
	seenSection := map[Section]bool{}
	seenPath := map[string]bool{}
	for _, m := range Table {
		if m.Section == "" || m.LifeboatPath == "" || m.Reads == "" {
			t.Errorf("row %+v has an empty field; every section names where it lands and what grounds it", m)
		}
		if seenSection[m.Section] {
			t.Errorf("duplicate section %q", m.Section)
		}
		if seenPath[m.LifeboatPath] {
			t.Errorf("duplicate lifeboat path %q", m.LifeboatPath)
		}
		seenSection[m.Section] = true
		seenPath[m.LifeboatPath] = true
	}
}
