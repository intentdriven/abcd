package reading

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/gitutil"
)

// The fixture repository the assembler tests read. Every warm class the record
// names is planted in it with a distinctive sentinel token, so a leak shows up
// as a token rather than as a judgement call.
const (
	sentinelEvidence    = "SENTINEL-EVIDENCE-CHAPTER"
	sentinelDecision    = "SENTINEL-DECISION-RECORD"
	sentinelIssue       = "SENTINEL-ISSUE-LEDGER"
	sentinelAuditNotes  = "SENTINEL-AUDIT-NOTES"
	sentinelWhyItMatter = "SENTINEL-WHY-THIS-MATTERS"
	sentinelOrigin      = "SENTINEL-ORIGIN-KEY"
	sentinelProdMode    = "SENTINEL-PRODUCTION-MODE"
	sentinelSuperseded  = "SENTINEL-SUPERSEDED-INTENT"
	sentinelPlan        = "SENTINEL-PLAN-RECORD"
	sentinelPriorRun    = "SENTINEL-PRIOR-MANIFEST"
	sentinelDraftBody   = "SENTINEL-DRAFT-BODY"
	sentinelDefinition  = "SENTINEL-READING-DEFINITION"
	sentinelLapse       = "SENTINEL-LAPSE-LOG"
)

// writeFile writes one fixture file, creating its parents.
func writeFile(t *testing.T, root, rel, body string) {
	t.Helper()
	abs := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
		t.Fatalf("mkdir for %s: %v", rel, err)
	}
	if err := os.WriteFile(abs, []byte(body), 0o644); err != nil {
		t.Fatalf("write %s: %v", rel, err)
	}
}

// fixtureRepo materialises a small repository carrying one file of every class
// the include table admits and one of every class the exclusion floor refuses,
// commits it, and returns its root.
func fixtureRepo(t *testing.T) string {
	t.Helper()
	root := t.TempDir()

	// fixtureScope selects every material kind at every assembling position, so
	// a test written before scopes existed sees the item set it was written
	// against. A test that cares about NARROWING names its own preset.
	writeFile(t, root, ".abcd/config/reading-presets.json", fixturePresets())

	writeFile(t, root, ".abcd/record-lint.json", `{
  "schema_version": 1,
  "rules": {
    "record_schema": {
      "enabled": true,
      "severity": "blocker",
      "record_stores": {
        "adr": ".abcd/development/decisions/adrs",
        "itd": ".abcd/development/intents",
        "spc": ".abcd/development/specs",
        "iss": ".abcd/work/issues",
        "rdi": ".abcd/work/issues/readings"
      }
    }
  }
}
`)

	// The admitted floor.
	writeFile(t, root, ".abcd/development/brief/01-product/06-framing.md",
		"# Framing\n\n## Construal\n\nWe are treating this as a gap in what the repository records.\n")
	writeFile(t, root, ".abcd/development/brief/01-product/01-press-release.md",
		"---\nproduction_mode: hand-written "+sentinelProdMode+"\n---\n\n# Press release\n\nThe product, stated.\n")
	writeFile(t, root, ".abcd/development/brief/02-constraints/03-invariants.md",
		"# Invariants\n\n1. The core is transport agnostic.\n")
	// The rest of brief current text (itd-194): the meta chapter at the brief's
	// root and the three chapters below the evidence chapter. A walk row's
	// source directory must exist or the run refuses, so these are not optional
	// decoration — the table names six chapters and the fixture carries six.
	writeFile(t, root, ".abcd/development/brief/00-meta.md",
		"# Meta\n\nHow this brief is organised.\n")
	writeFile(t, root, ".abcd/development/brief/04-surfaces/23-reading.md",
		"# The reading surface\n\nWhat the verb does.\n")
	writeFile(t, root, ".abcd/development/brief/05-internals/03-configuration.md",
		"# Configuration\n\nWhere the settings live.\n")
	writeFile(t, root, ".abcd/development/brief/06-delivery/01-shipping.md",
		"# Shipping\n\nHow a release is cut.\n")
	writeFile(t, root, ".abcd/development/brief/glossary/core/construal.md",
		"# Construal\n\nWhat the situation is treated as.\n")
	writeFile(t, root, ".abcd/development/intents/disciplines/itd-4-selection-criteria.md",
		"---\nid: itd-4\n---\n\n# Selection criteria\n\nSix criteria, recorded.\n")
	// The criteria discipline the comparative position narrows its disciplines
	// row to. It carries the `## The rule` bullets declaredCriteria parses, so
	// the fixture's slate is read off a record exactly as the committed one is.
	writeFile(t, root, ".abcd/development/intents/disciplines/"+fixtureCriteriaFile,
		"---\nid: "+CriteriaDiscipline+"\n---\n\n# The selection criteria\n\n"+
			"## The rule\n\nSelection weighs candidates against the criteria this record declares.\n\n"+
			"- Plausibility — the conjecture could work by a mechanism we can state.\n"+
			"- Generativity — pursuing it opens further conjectures.\n"+
			"- Cost — what building and carrying it consumes.\n\n"+
			"## The gate\n\nThe criteria come from this record and never from an invocation.\n")
	writeFile(t, root, ".abcd/development/specs/open/spc-1-a-design-record.md",
		"---\nid: spc-1\nintent: itd-1\norigin: "+sentinelOrigin+"\n---\n\n# A design record\n\nThe mechanics.\n")
	writeFile(t, root, "README.md", "# The repository\n\nWhat it is.\n")
	writeFile(t, root, "docs/reference/thing.md", "# Thing\n\nReference prose.\n")
	writeFile(t, root, "main.go", "package main\n\nfunc main() {}\n")
	writeFile(t, root, "go.mod", "module example\n\ngo 1.24\n")
	writeFile(t, root, "Makefile", "build:\n\t@echo build\n")

	// A shipped intent carrying both the fields that travel and the fields that
	// must not.
	writeFile(t, root, ".abcd/development/intents/shipped/itd-1-a-shipped-intent.md",
		"---\nid: itd-1\nspec_id: spc-1\norigin: "+sentinelOrigin+"\n---\n\n"+
			"# A shipped intent\n\n"+
			"## Press Release\n\nThe promise, as made.\n\n"+
			"## Why This Matters\n\n"+sentinelWhyItMatter+"\n\n"+
			"## Acceptance Criteria\n\n- Given a state, when it runs, then it holds.\n\n"+
			"## Scope Conditions\n\n- Holds while the record is one repository.\n\n"+
			"## Mechanism\n\nA positive include table.\n\n"+
			"## Audit Notes\n\n"+sentinelAuditNotes+"\n")

	// The candidate set: cold at entailment, warm everywhere else.
	writeFile(t, root, ".abcd/development/intents/drafts/itd-2-a-draft.md",
		"---\nid: itd-2\norigin: "+sentinelOrigin+"\n---\n\n# A draft\n\n## Press Release\n\n"+sentinelDraftBody+"\n")
	writeFile(t, root, ".abcd/development/intents/planned/itd-3-a-planned.md",
		"---\nid: itd-3\nspec_id: spc-1\n---\n\n# A planned intent\n\n## Press Release\n\nA planned promise.\n")

	// Every refused class.
	writeFile(t, root, ".abcd/development/brief/03-evidence/01-open-questions.md",
		"# Open questions\n\n"+sentinelEvidence+"\n")
	writeFile(t, root, ".abcd/development/decisions/adrs/0001-a-decision.md",
		"---\nid: adr-1\n---\n\n# ADR-1\n\n"+sentinelDecision+"\n")
	writeFile(t, root, ".abcd/development/intents/superseded/itd-5-a-retired.md",
		"---\nid: itd-5\n---\n\n# A retired intent\n\n"+sentinelSuperseded+"\n")
	writeFile(t, root, ".abcd/development/plans/2026-08-30-a-plan.md", "# A plan\n\n"+sentinelPlan+"\n")
	writeFile(t, root, ".abcd/development/research/notes/a-note.md", "# A note\n\nresearch prose.\n")
	writeFile(t, root, ".abcd/development/roadmap/rfcs/an-rfc.md", "# An RFC\n\nrfc prose.\n")
	writeFile(t, root, ".abcd/work/issues/open/iss-1-a-defect.md",
		"---\nid: iss-1\n---\n\n"+sentinelIssue+"\n")
	writeFile(t, root, ".abcd/work/issues/open/iss-2-a-lapse.md",
		"---\nid: iss-2\ncategory: lapse-log\n---\n\n"+sentinelLapse+"\n")
	writeFile(t, root, ".abcd/work/DECISIONS.md", "# Decisions\n\n"+sentinelDecision+"\n")
	writeFile(t, root, ".abcd/development/readings/rdg-2608301200000001/manifest.json",
		"{\"note\": \""+sentinelPriorRun+"\"}\n")
	writeFile(t, root, "agents/cold-reading-widening.md", "# Widening\n\n"+sentinelDefinition+"\n")
	writeFile(t, root, "evals/read_block_test.go", "package evals\n\n// "+sentinelDefinition+"\n")

	// The candidate set. Every position assembles, so the fixture carries what
	// the comparative position needs to assemble AT ALL: one committed widening
	// run of three undispositioned items at the fixture's HEAD. A test that cares
	// about the derivation plants its own second run, dispositions an item, or
	// removes this one.
	plantWideningItems(t, root, fixtureCandidateRun, 3)

	// The durable readings family is left OUT of the fixture's git index.
	//
	// A run record names the commit its reading read, and every fixture that
	// adds a file and commits it moves HEAD — so the record has to be rewritten
	// after each commit (restampFixtureRun), and a tracked file rewritten after
	// every commit would leave the tree permanently dirty. Nothing is lost by
	// ignoring it: the family is admitted by no include row and excluded at
	// every position, so it is outside both the walk and the dirty gate, and the
	// candidate records the assembly actually reads are tracked and committed
	// like every other admitted path.
	writeFile(t, root, ".gitignore", ".abcd/development/readings/\n")

	gitInit(t, root)

	// After the commit, because the run's own `target_commit` names the commit
	// its records landed in. The durable readings family is admitted by no row
	// and excluded at every position, so it sits outside the dirty gate and can
	// be written here without chasing HEAD.
	commitRunRecord(t, root, fixtureCandidateRun, string(PositionWidening), headOf(t, root))
	return root
}

// fixtureCandidateRun is the widening run the base fixture commits, and the one
// a comparative assembly over that fixture derives. fixtureCandidateItems are
// the three item ids it holds.
const fixtureCandidateRun = "rdg-2608301200000009"

var fixtureCandidateItems = []string{
	wideningItemID(fixtureCandidateRun, 1),
	wideningItemID(fixtureCandidateRun, 2),
	wideningItemID(fixtureCandidateRun, 3),
}

// wideningItemID seeds an item id from its run, so two planted runs in one
// fixture never share one.
func wideningItemID(run string, i int) string {
	return fmt.Sprintf("rdi-%s%02d", run[len(run)-3:], i)
}

// The criteria discipline the fixture carries, and the slate it declares. The
// names are what declaredCriteria parses out of its `## The rule` bullets, so a
// test asserting the manifest's `criteria` compares against the record rather
// than against a second list (itd-191).
const fixtureCriteriaFile = CriteriaDiscipline + "-the-selection-criteria.md"

var fixtureCriteria = []string{"Plausibility", "Generativity", "Cost"}

// sentinelCandidate classes the fixture's candidate plants: the configuration
// text a comparative reading must receive, the pattern it must not (provenance
// is the envelope's), and the disposition text that is the researcher's
// judgement and never a reading's input.
const (
	sentinelCandidate = "SENTINEL-CANDIDATE-CONFIGURATION"
	sentinelEnvelope  = "SENTINEL-CANDIDATE-PATTERN"
	sentinelFate      = "SENTINEL-CANDIDATE-FATE"
)

// plantWideningItems writes n reading records for one run into the LEDGER and
// returns their ids. Nothing is committed here: the caller commits, because the
// run's own `target_commit` has to name the commit the records landed in.
func plantWideningItems(t *testing.T, root, run string, n int) []string {
	t.Helper()
	ids := make([]string, 0, n)
	for i := 1; i <= n; i++ {
		// The ids are seeded from the RUN so two planted runs never share one.
		// They would otherwise: a reading item id is unique to the LEDGER, not to
		// the run that minted it — mintUnusedItemID probes the whole tree — and
		// the disposition store is keyed by the item alone, so two runs sharing
		// an id would make one run's disposition the other's fate.
		id := wideningItemID(run, i)
		ids = append(ids, id)
		writeFile(t, root, ".abcd/work/issues/readings/"+run+"/"+id+".md",
			"---\nschema_version: 1\nid: \""+id+"\"\nrun: \""+run+"\"\n"+
				"manifest: \""+strings.Repeat("a", 64)+"\"\nposition: \"widening\"\n"+
				"regime: \"generative\"\npattern: \""+sentinelEnvelope+"-"+id+"\"\n"+
				"configuration: \""+sentinelCandidate+"-"+id+"\"\n"+
				"what_admits_it: \"what admits "+id+"\"\n---\n")
	}
	return ids
}

// commitRunRecord writes the run's COMMIT MARKER, naming the position and the
// commit the run read.
//
// It is written after the commit and left untracked on purpose: the durable
// readings family is excluded at every position and admitted by no row, so it is
// outside the dirty gate — which is what lets a fixture name the very commit its
// records landed in without chasing its own HEAD.
func commitRunRecord(t *testing.T, root, run, position, target string) {
	t.Helper()
	writeFile(t, root, ".abcd/development/readings/"+run+"/run.json",
		fmt.Sprintf(`{"_type":%q,"schema_version":%d,"run_id":%q,"position":%q,"target_commit":%q}`+"\n",
			RunType, SchemaVersion, run, position, target))
}

// parkRunManifest writes a run's parked manifest WITHOUT its commit marker: a
// run that reached the assembler and never reached its ingest.
func parkRunManifest(t *testing.T, root, run, position, target string) {
	t.Helper()
	writeFile(t, root, ".abcd/development/readings/"+run+"/manifest.json",
		fmt.Sprintf(`{"_type":%q,"run_id":%q,"position":%q,"target_commit":%q}`+"\n",
			ManifestType, run, position, target))
}

// plantDisposition writes a standing disposition over one item, by hand. By hand
// is the point: spc-2609020626040342's gate in the shared disposition writer
// makes this unreachable through any verb, and the assembler cannot know a
// record was placed by hand — so it refuses what it finds.
func plantDisposition(t *testing.T, root, item, id string) {
	t.Helper()
	writeFile(t, root, ".abcd/work/issues/dispositions/"+item+"/"+id+".md",
		"---\nschema_version: 1\nid: \""+id+"\"\nitem: \""+item+"\"\nstate: \"accepted\"\n"+
			"disposition_grounds: \""+sentinelFate+"\"\n---\n")
}

// plantAdmission writes an admission of one item under one run, by hand.
func plantAdmission(t *testing.T, root, run, item, id string) {
	t.Helper()
	writeFile(t, root, ".abcd/work/issues/admissions/"+run+"/"+id+".md",
		"---\nschema_version: 1\nid: \""+id+"\"\nrun: \""+run+"\"\nproposal: \""+item+"\"\n"+
			"grounds: \""+sentinelFate+"\"\n---\n")
}

// gitInit turns a fixture directory into a repository with one commit, so HEAD
// resolves and no included path is dirty.
func gitInit(t *testing.T, root string) {
	t.Helper()
	steps := [][]string{
		{"init", "-q", "-b", "main"},
		{"-c", "user.name=abcd test", "-c", "user.email=test@example.invalid", "add", "-A"},
		{"-c", "user.name=abcd test", "-c", "user.email=test@example.invalid",
			"commit", "-q", "-m", "fixture"},
	}
	for _, args := range steps {
		if out, err := gitutil.Run(root, args...); err != nil {
			t.Fatalf("git %s: %v (%s)", strings.Join(args, " "), err, out)
		}
	}
}

// headOf returns the fixture repository's HEAD sha.
func headOf(t *testing.T, root string) string {
	t.Helper()
	sha, err := gitutil.Run(root, "rev-parse", "HEAD")
	if err != nil {
		t.Fatalf("rev-parse HEAD: %v", err)
	}
	return sha
}

// assembleFixture runs one assembly over the fixture repository at p.
func assembleFixture(t *testing.T, root string, p Position) AssembleResult {
	t.Helper()
	res, err := Assemble(AssembleRequest{
		RepoRoot: root, Position: p, Target: "HEAD", DryRun: true,
	})
	if err != nil {
		t.Fatalf("assemble at %s: %v", p, err)
	}
	return res
}

// bundleText joins every item's text, for a sentinel scan.
func bundleText(b Bundle) string {
	var sb strings.Builder
	for _, it := range b.Items {
		sb.WriteString(it.Text)
		sb.WriteString("\n")
	}
	return sb.String()
}

// itemPaths returns the manifest's item paths in order.
func itemPaths(m Manifest) []string {
	out := make([]string, 0, len(m.Items))
	for _, it := range m.Items {
		out = append(out, it.Path)
	}
	return out
}

// gitCommitAll commits everything a test added after the fixture was built, so
// the tree stays clean for the dirty-path gate.
func gitCommitAll(t *testing.T, root string) {
	t.Helper()
	ident := []string{"-c", "user.name=abcd test", "-c", "user.email=test@example.invalid"}
	for _, args := range [][]string{
		append(append([]string{}, ident...), "add", "-A"),
		append(append([]string{}, ident...), "commit", "-q", "-m", "fixture update"),
	} {
		if out, err := gitutil.Run(root, args...); err != nil {
			t.Fatalf("git %s: %v (%s)", strings.Join(args, " "), err, out)
		}
	}
	restampFixtureRun(t, root)
}

// restampFixtureRun re-points the fixture's committed widening run at the new
// HEAD.
//
// A run record names the commit its reading ACTUALLY read, and the comparative
// derivation selects on that (adr-2609021016272867). A fixture that adds a file
// and commits it has moved HEAD, so without this every test that extends the
// base fixture would find no widening run at its own target — a fixture
// artefact, not a property, and one that would make the derivation's refusal
// fire everywhere and prove nothing.
func restampFixtureRun(t *testing.T, root string) {
	t.Helper()
	rel := ".abcd/development/readings/" + fixtureCandidateRun + "/run.json"
	if _, err := os.Stat(filepath.Join(root, filepath.FromSlash(rel))); err != nil {
		return // a test that removed the run, or one that never had it
	}
	commitRunRecord(t, root, fixtureCandidateRun, string(PositionWidening), headOf(t, root))
}

// gitRun runs one git command against a fixture repository under the test
// identity, failing the test on error.
func gitRun(t *testing.T, root string, args ...string) {
	t.Helper()
	full := append([]string{"-c", "user.name=abcd test", "-c", "user.email=test@example.invalid"}, args...)
	if out, err := gitutil.Run(root, full...); err != nil {
		t.Fatalf("git %s: %v (%s)", strings.Join(args, " "), err, out)
	}
}

// fixtureScopeName is the all-selecting preset every fixture carries.
const fixtureScopeName = "everything"

// fixturePresets renders a preset file selecting every kind at every position
// that assembles. It is generated from Kinds() and AssemblingPositions() rather
// than written out, so a new kind or position cannot leave the fixture quietly
// narrower than the table it is meant to mirror.
func fixturePresets() string {
	kinds := make([]string, 0, len(Kinds()))
	for _, k := range Kinds() {
		// `candidate` is a material kind and NOT a preset kind: an entry names
		// repository material, and the candidate set is derived from the record
		// (adr-2609021016272867). validateEntries refuses it by name, so
		// generating it here would make every fixture unloadable.
		if k == KindCandidate {
			continue
		}
		kinds = append(kinds, strconv.Quote(string(k)))
	}
	paths := make([]string, 0, len(fixtureTreePaths))
	for _, p := range fixtureTreePaths {
		paths = append(paths, strconv.Quote(p))
	}
	positions := make([]string, 0, len(AssemblingPositions()))
	for _, p := range AssemblingPositions() {
		// The entailment entry declares the admissibility switch, which is how
		// an entry says "every draft and planned intent" now that those two
		// rows narrow by the entry's record list always (ruled 2026-09-02;
		// divergence register 1 as corrected). It is the same all-selecting
		// intent this file has always had — a test written before entries
		// existed sees the item set it was written against — and a test that
		// cares about the narrowing names its own preset.
		switchOn := ""
		if p == PositionEntailment {
			switchOn = `"admit_drafts_and_planned": true, `
		}
		positions = append(positions, fmt.Sprintf(
			`    %q: {"object": {"records": [], "paths": [%s]}, "kinds": [%s], %s`+
				`"window": {"tokens_est": 1000000, "measured_tokens_est": 0, `+
				`"measured_bytes": 0, "measured_at": "0000000"}}`,
			string(p), strings.Join(paths, ", "), strings.Join(kinds, ", "), switchOn))
	}
	return fmt.Sprintf(`{
  "schema_version": %d,
  "positions": {
%s
  }
}
`, PresetSchemaVersion, strings.Join(positions, ",\n"))
}

// fixtureTreePaths is every root-level path the fixture carries tree material
// under. The object set NARROWS the tree rows, and an entry naming no path hands
// nothing from the tree whatever kinds it lists (spc-2609020626048722), so the
// all-selecting fixture entry has to name where the fixture's tree material
// sits. `main_test.go` is listed although the base fixture does not carry it: a
// test that adds one is naming a path the entry already reaches, rather than
// having to rewrite the entry to see it.
var fixtureTreePaths = []string{
	"Gadget_TEST.go", "Makefile", "README.md", "build", "docs", "fence.go",
	"go.mod", "ignored.json", "main.go", "main_test.go", "run-dir", "runs",
	"widget.go", "widget_test.go",
}
