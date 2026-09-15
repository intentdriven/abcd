//go:build smoke || coldreading

package evals

// The opening-run rehearsal: the whole loop a maintainer runs on the morning of
// the first cold reading, driven end to end over a fixture repository through
// the real front door, with no model in it.
//
// Every other eval in this lane falsifies ONE property of ONE verb. This one
// asks the question none of them can: does the sequence work? A loop whose every
// step passes its own test can still be unrunnable — a step can leave the tree
// in a state the next step refuses, and nothing that tests a step in isolation
// can see that. So the invocations here are the operator's own, in the operator's
// order, through the binary the harness builds: `reading assemble` at each
// position, `reading ingest` of a hand-written output, `capture disposition`,
// `capture promote`, and the outstanding-item report.
//
// The outputs are HAND-WRITTEN, which is the point. A reading is a model's
// return and no eval can call one, so what is rehearsed is the contract either
// side of it: what the assembler hands out, and what the ingest accepts back.
// Every payload here is composed from the parked manifest the assembly actually
// wrote, so a payload that would not have been a legal return of that run is not
// one this eval can accidentally build.
//
// The sections each step holds are cited at the step. Framework 8.5 (the
// invocation), 8.9 (the manifest), 11.2 and 11.3 (the disposition vocabulary and
// its grounds), 13 (the clean run); companion 7.6 (the comparative's plural
// candidate set) and 4.3 (what a disposition carries).
//
// The oracle rule holds here as everywhere in this package: nothing under
// evals/ imports the assembler, so every constant below — the four positions'
// regimes, the body fields, the record homes — is TRANSCRIBED from
// commands/reading.md and the record, and disagrees with the code exactly when
// the code disagrees with the record.

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

// The four positions' supply regimes, transcribed from commands/reading.md ("The
// supply regime is the definition's") and from the definitions themselves. The
// regime is the definition's property; a payload declaring another is refused,
// which is one of the refusals below.
var rehearsalRegimes = map[string]string{
	posWidening:    "generative",
	posEntailment:  "explicative",
	posComparative: "evaluative",
	posDetection:   "registrative",
}

// The body fields each position declares, transcribed from the table in
// commands/reading.md, "What the output carries".
var rehearsalBodyFields = map[string][]string{
	posWidening:    {"configuration", "what_admits_it"},
	posEntailment:  {"claim_surfaced", "claim_type", "what_implies_it"},
	posComparative: {"candidate_id", "criterion", "characterisation"},
	posDetection:   {"tension", "constraint_in_play", "why_a_tension"},
}

// Where the plugin page says the artefacts and the records land. Every one of
// these is asserted rather than assumed: a verb that wrote its records somewhere
// else would satisfy every unit test of its own package and leave the operator
// with a ledger nobody reads.
const (
	// rehearsalRunDir is the local-tier directory an assembly parks a run in
	// when no --out is named ("Where the artefacts land").
	rehearsalRunDir = ".abcd/.work.local/scratch/reading-runs"
	// rehearsalReadingsLedger is the committed home of one run's reading
	// records, keyed by the run.
	rehearsalReadingsLedger = ".abcd/work/issues/readings"
	// rehearsalRunRecords is the durable home of a run's own artefacts: the
	// promoted manifest, the run metadata (the commit marker) and a refusal.
	rehearsalRunRecords = ".abcd/development/readings"
	// rehearsalDispositions is keyed by the ITEM, one directory per item.
	rehearsalDispositions = ".abcd/work/issues/dispositions"
	// rehearsalRunMarker is the commit marker: a run without one never happened.
	rehearsalRunMarker = "run.json"
	// rehearsalRefusalRecord is what a list-level refusal leaves behind once the
	// run's identity is proven.
	rehearsalRefusalRecord = "refusal.json"
	// rehearsalDrafts is where a promoted reading item's draft intent lands.
	rehearsalDrafts = ".abcd/development/intents/drafts"
)

// The planted widening run the comparative position derives, and the criteria
// the fixture's discipline declares. Both are read from the corpus by this
// eval's own eyes below; they are named here so a failure says which record went
// missing rather than only that a derivation refused.
var (
	rehearsalCandidates = []string{"rdi-301", "rdi-302", "rdi-303"}
	rehearsalCriteria   = []string{"Plausibility", "Generativity", "Cost"}
)

// TestRehearseTheOpeningRunLoop is the rehearsal itself: one fixture repository,
// one sequence, every command the morning needs.
//
// The steps run in order and share state, so they are sequential `t.Run` calls
// rather than independent tests: the loop is the object under test, and a step
// that only passes when run alone is precisely what this eval exists to catch.
func TestRehearseTheOpeningRunLoop(t *testing.T) {
	f := rehearsalFixture(t)

	// -------------------------------------------------------------------
	// Step 1 — assemble at the three positions whose object is the tree.
	//
	// Framework 8.5: the invocation is a position and a target state and
	// nothing else. Framework 8.9: every run emits a manifest naming what was
	// passed. Both are asserted on what the verb actually wrote, at the paths
	// the plugin page names.
	// -------------------------------------------------------------------
	parked := map[string]parkedRun{}
	for _, position := range []string{posWidening, posEntailment, posDetection} {
		t.Run("assemble-"+position, func(t *testing.T) {
			run := assembleParked(t, f, position)
			parked[position] = run
			requireArtefacts(t, f, run)
		})
	}

	// -------------------------------------------------------------------
	// Step 1b — the comparative runs, parked BEFORE any ingest.
	//
	// The comparative position's candidate row reaches the ledger's readings
	// store, so every reading record in the working tree is an INCLUDED path —
	// and an ingest writes reading records that are, by construction,
	// uncommitted. So once this session's first ingest has run, a comparative
	// assembly refuses on the dirty gate until those records are committed.
	// Committing them is the act the design sequences between an ingest and the
	// next reading, and the assembly that follows it is rehearsed on its own
	// below (TestTheComparativeAssemblyFollowsTheCommittedWideningIngest,
	// iss-2609021833302981). This loop parks its four comparative runs before
	// the first ingest instead, because the three refusal cases below are
	// mutations of one legal payload and need the run they were composed
	// against — not because no other order works.
	//
	// Companion 7.6 wants a PLURAL candidate set for the comparative to be
	// exercised, and the fixture's committed widening run carries three items
	// with no disposition and no admission — which is what the derivation
	// selects (adr-2609021016272867).
	// -------------------------------------------------------------------
	comparative := map[string]parkedRun{}
	for _, use := range []string{
		"clean", "reserved-name-evaluative", "unknown-candidate", "undeclared-criterion",
	} {
		t.Run("assemble-comparative-"+use, func(t *testing.T) {
			run := assembleParked(t, f, posComparative)
			comparative[use] = run
			requireArtefacts(t, f, run)
			requireDerivedCandidates(t, f, run)
		})
	}

	// -------------------------------------------------------------------
	// Step 2 — ingest a hand-written widening output carrying two clean
	// configurations, and assert the records land where the plugin page says:
	// the reading records in the committed ledger under the run, the manifest
	// and the run metadata in the durable run directory, the run metadata last
	// as the commit marker.
	// -------------------------------------------------------------------
	var wideningItems []string
	t.Run("ingest-widening", func(t *testing.T) {
		run := parked[posWidening]
		res := ingestAccepted(t, f, run, []map[string]any{
			{
				"pattern":        "the stated invariant",
				"configuration":  "a first configuration the construal admits and the record does not carry",
				"what_admits_it": "the invariant the constraints chapter already states",
			},
			{
				"pattern":        "the delivery constraint",
				"configuration":  "a second configuration the construal admits and the record does not carry",
				"what_admits_it": "the constraint the delivery chapter already states",
			},
		})
		if len(res.Records) != 2 {
			t.Fatalf("the widening ingest recorded %d item(s), want the 2 the payload carried; "+
				"companion 7.6 needs a plural set for the comparative to have anything to compare",
				len(res.Records))
		}
		for _, r := range res.Records {
			wideningItems = append(wideningItems, r.ID)
		}
		requireCommittedRun(t, f, run.RunID, res)
	})

	// -------------------------------------------------------------------
	// Step 3 — ingest the comparative output: one clean item per
	// candidate-criterion pair for one candidate (companion 7.6).
	// -------------------------------------------------------------------
	t.Run("ingest-comparative", func(t *testing.T) {
		run := comparative["clean"]
		items := make([]map[string]any, 0, len(rehearsalCriteria))
		for _, criterion := range rehearsalCriteria {
			items = append(items, map[string]any{
				"pattern":          "how options of this shape ordinarily behave",
				"candidate_id":     rehearsalCandidates[0],
				"criterion":        criterion,
				"characterisation": "how a configuration of this shape ordinarily behaves against " + criterion,
			})
		}
		res := ingestAccepted(t, f, run, items)
		if len(res.Records) != len(rehearsalCriteria) {
			t.Fatalf("the comparative ingest recorded %d item(s), want one per declared criterion (%d); "+
				"companion 7.6 asks for one characterisation per candidate-criterion pair",
				len(res.Records), len(rehearsalCriteria))
		}
		requireCommittedRun(t, f, run.RunID, res)
	})

	// -------------------------------------------------------------------
	// Step 4 — one clean item at entailment and one at detection.
	// -------------------------------------------------------------------
	var detectionItems []string
	t.Run("ingest-entailment", func(t *testing.T) {
		run := parked[posEntailment]
		res := ingestAccepted(t, f, run, []map[string]any{{
			"pattern":         "what the design commits to by being what it is",
			"claim_surfaced":  "the record commits to a claim its articulation does not state",
			"claim_type":      "criterion",
			"what_implies_it": "the constraint the brief states, taken with the surface it declares",
		}})
		requireCommittedRun(t, f, run.RunID, res)
	})
	t.Run("ingest-detection", func(t *testing.T) {
		run := parked[posDetection]
		res := ingestAccepted(t, f, run, []map[string]any{{
			"pattern":            "a stated constraint the tree does not keep",
			"tension":            "the shipped tree and the claim record disagree about what is enforced",
			"constraint_in_play": "the invariant the constraints chapter states",
			"why_a_tension":      "one of the two has to give, and the record does not say which",
		}})
		if len(res.Records) != 1 {
			t.Fatalf("the detection ingest recorded %d item(s), want 1", len(res.Records))
		}
		for _, r := range res.Records {
			detectionItems = append(detectionItems, r.ID)
		}
		requireCommittedRun(t, f, run.RunID, res)
	})

	// -------------------------------------------------------------------
	// Step 5 — every refusal the verb documents, each at the position it
	// applies at, each asserting that the refusal NAMES the item and that
	// nothing partial was written.
	// -------------------------------------------------------------------
	// The comparative cases run over the runs parked in step 1b, for the reason
	// given there: at that position no run can be assembled once a reading
	// record is uncommitted, and by now several are.
	t.Run("refusals", func(t *testing.T) {
		for _, c := range rehearsalRefusals() {
			t.Run(c.Name, func(t *testing.T) {
				rehearseRefusal(t, f, c, comparative)
			})
		}
	})

	// -------------------------------------------------------------------
	// Step 6 — the clean run (framework 13): an empty item list commits a run
	// with an empty item set at every position, and is never refused. Refusal
	// is reserved for a malformed payload.
	// -------------------------------------------------------------------
	t.Run("empty-output-commits-a-clean-run", func(t *testing.T) {
		run := assembleParked(t, f, posDetection)
		res := ingestAccepted(t, f, run, []map[string]any{})
		if len(res.Records) != 0 {
			t.Fatalf("an empty output recorded %d item(s); framework 13 fixes the null result as a "+
				"run with an EMPTY item set", len(res.Records))
		}
		if res.RefusedCount != 0 {
			t.Fatalf("an empty output refused %d item(s); the clean run is recorded, never refused",
				res.RefusedCount)
		}
		requireCommittedRun(t, f, run.RunID, res)
		if n := readingRecordsUnder(t, f, run.RunID); n != 0 {
			t.Errorf("the clean run left %d reading record(s) under %s/%s; a run with an empty item "+
				"set holds none", n, rehearsalReadingsLedger, run.RunID)
		}
	})

	// -------------------------------------------------------------------
	// Step 7 — the disposition vocabulary (framework 11.2 and 11.3; companion
	// 4.3). Four states, availability by position, grounds on every state but
	// held, an exit condition on held and only on held.
	// -------------------------------------------------------------------
	t.Run("dispositions", func(t *testing.T) {
		if len(detectionItems) == 0 || len(wideningItems) < 2 {
			t.Fatal("the earlier ingests left no item to disposition, so this step would assert nothing")
		}
		rehearseDispositions(t, f, detectionItems[0], wideningItems)
	})

	// -------------------------------------------------------------------
	// Step 8 — promote the accepted detection item into a draft intent whose
	// origin names the pair it was graduated from.
	// -------------------------------------------------------------------
	t.Run("promote", func(t *testing.T) {
		if len(detectionItems) == 0 {
			t.Fatal("no detection item was recorded, so there is nothing to promote")
		}
		rehearsePromote(t, f, parked[posDetection].RunID, detectionItems[0])
	})

	// -------------------------------------------------------------------
	// Step 9 — the outstanding-item report. Every item nobody has answered is
	// named, because nothing in this vocabulary means "already covered".
	// -------------------------------------------------------------------
	t.Run("outstanding-report", func(t *testing.T) {
		rehearseOutstanding(t, f, wideningItems, detectionItems)
	})
}

// TestEveryRehearsedItemCarriesItsPositionsDeclaredBody is the rehearsal's
// anti-vacuity guard.
//
// Every refusal case above starts from a legal item and mutates it, so a "legal"
// item that was in fact illegal would make each of those cases pass for the
// wrong reason — the payload refused, but on a rule nobody named. This holds the
// base items to the body table transcribed from commands/reading.md: the pattern
// and exactly the position's declared fields, no more and no fewer.
func TestEveryRehearsedItemCarriesItsPositionsDeclaredBody(t *testing.T) {
	for _, position := range everyPosition {
		want := append([]string{"pattern"}, rehearsalBodyFields[position]...)
		sort.Strings(want)
		got := make([]string, 0, len(want))
		for k := range legalItem(position) {
			got = append(got, k)
		}
		sort.Strings(got)
		if strings.Join(got, ",") != strings.Join(want, ",") {
			t.Errorf("the rehearsed %s item carries %v and the position declares %v; a base item "+
				"that is not legal makes every refusal mutated from it pass for the wrong reason",
				position, got, want)
		}
	}
}

// TestTheComparativeAssemblyFollowsTheCommittedWideningIngest rehearses the
// step the loop above cannot take in place: the comparative reading over the
// widening run this session just ingested.
//
// The sequence is the design's own — assemble at widening, ingest, COMMIT what
// the ingest wrote, assemble at comparative, ingest that — and each half of the
// middle step is load-bearing. `reading ingest` leaves the run's reading records
// in the committed ledger uncommitted, and the comparative position's candidate
// row reaches that store, so the assembly refuses on the dirty gate until they
// are committed. Committing them moves HEAD off the commit the widening run's
// own record names, and the derivation reads "the one committed widening run at
// the target" (adr-2609021016272867) as reaching an ancestor whose object set
// has not moved — which a commit confined to the readings store and the issue
// ledger has not. That reading is an interpretation and the maintainer's ruling
// is owed (iss-2609021833302981, iss-2609021857343626); the register's entry 27
// is its ground, "the object set, not the commit, names what the readings are
// about", and companion 7.2 fixes the comparative object as the widening run's
// returned items and the criteria discipline, neither of which the tree supplies.
func TestTheComparativeAssemblyFollowsTheCommittedWideningIngest(t *testing.T) {
	f := rehearsalFixtureWithoutPlantedRuns(t)

	widening := assembleParked(t, f, posWidening)
	ingested := ingestAccepted(t, f, widening, []map[string]any{
		{
			"pattern":        "the stated invariant",
			"configuration":  "a first configuration the construal admits",
			"what_admits_it": "the invariant the constraints chapter already states",
		},
		{
			"pattern":        "the delivery constraint",
			"configuration":  "a second configuration the construal admits",
			"what_admits_it": "the constraint the delivery chapter already states",
		},
	})
	if len(ingested.Records) != 2 {
		t.Fatalf("the widening ingest recorded %d item(s), want 2; companion 7.6 needs a plural "+
			"set for the comparative position to be exercised", len(ingested.Records))
	}
	requireCommittedRun(t, f, widening.RunID, ingested)

	// The first half of the middle step, asserted rather than assumed: with the
	// ingest's records still uncommitted the assembly refuses, so the commit
	// below is load-bearing and this test cannot pass by the gate never firing.
	refused, code := runIn(t, f.Root, []string{"HOME=" + f.Home},
		"reading", "assemble", "--position", posComparative, "--target", "HEAD")
	if code == 0 {
		t.Fatalf("the comparative assembly succeeded with the widening run's reading records "+
			"uncommitted; the candidate row reaches that store, so a dirty record there is a "+
			"dirty included path:\n%s", refused)
	}
	if !strings.Contains(refused, "uncommitted") {
		t.Errorf("the assembly refused on something other than the dirty gate:\n%s", refused)
	}

	// The operator's own next act: commit what the ingest wrote, and nothing
	// else. The two paths are the reading records in the committed ledger and
	// the run's own durable artefacts — the whole of what an ingest writes.
	read := gitFixture(t, f.Root, "rev-parse", "HEAD")
	commitFixturePaths(t, f.Root, "the widening run's records, committed",
		rehearsalReadingsLedger, rehearsalRunRecords)
	target := gitFixture(t, f.Root, "rev-parse", "HEAD")
	if target == read {
		t.Fatal("committing the run's records did not move HEAD, so this rehearsal is not the " +
			"sequence it claims to be")
	}

	comparative := assembleParked(t, f, posComparative)
	requireArtefacts(t, f, comparative)
	requireDerivedCandidates(t, f, comparative)

	var m struct {
		CandidateRun       string `json:"candidate_run"`
		CandidateRunTarget string `json:"candidate_run_target"`
		TargetCommit       string `json:"target_commit"`
	}
	raw, err := os.ReadFile(filepath.Join(f.Root, filepath.FromSlash(comparative.OutDir), manifestFile))
	if err != nil {
		t.Fatalf("reading the comparative manifest: %v", err)
	}
	if err := json.Unmarshal(raw, &m); err != nil {
		t.Fatalf("decoding the comparative manifest: %v", err)
	}
	if m.CandidateRun != widening.RunID {
		t.Fatalf("the comparative assembly derived %q; the run whose records this session just "+
			"committed is %q, and no other widening run is in the corpus",
			m.CandidateRun, widening.RunID)
	}
	if m.CandidateRunTarget != read || m.TargetCommit != target {
		t.Errorf("the comparative manifest records candidate_run_target %q and target_commit %q; "+
			"the widening run read %s and this assembly names %s, and a reader needs both "+
			"commits to diff them", m.CandidateRunTarget, m.TargetCommit, read, target)
	}

	// And the step the loop above had to take out of order: the comparative
	// output ingests over the run just assembled (companion 7.6 — one item per
	// candidate-criterion pair).
	items := make([]map[string]any, 0, len(rehearsalCriteria))
	for _, criterion := range rehearsalCriteria {
		items = append(items, map[string]any{
			"pattern":          "how options of this shape ordinarily behave",
			"candidate_id":     ingested.Records[0].ID,
			"criterion":        criterion,
			"characterisation": "how a configuration of this shape ordinarily behaves against " + criterion,
		})
	}
	res := ingestAccepted(t, f, comparative, items)
	if len(res.Records) != len(rehearsalCriteria) {
		t.Fatalf("the comparative ingest recorded %d item(s), want one per declared criterion (%d)",
			len(res.Records), len(rehearsalCriteria))
	}
	requireCommittedRun(t, f, comparative.RunID, res)
}

// rehearsalFixtureWithoutPlantedRuns is the rehearsal fixture with the corpus's
// two planted widening runs removed, so the only widening run in the repository
// is the one the session itself ingests.
//
// The corpus plants two on purpose — one derivable, one already dispositioned —
// so the unit and window evals have something to select and something to reject.
// Here they would make the derivation AMBIGUOUS the moment the session's own run
// is committed, and the refusal that names two qualifying runs is a different
// property from the one this test holds.
func rehearsalFixtureWithoutPlantedRuns(t *testing.T) fixture {
	t.Helper()
	f := materialise(t, variantBaseline, withOperatingDefinitions, withOutstandingReport,
		withoutPlantedWideningRuns)
	// The durable run markers are stamped AFTER the commit, so removing them is
	// this side of materialisation rather than a tree edit.
	for _, run := range []string{derivedCandidateRun, dispositionedWideningRun} {
		if err := os.RemoveAll(filepath.Join(f.Root, filepath.FromSlash(rehearsalRunRecords), run)); err != nil {
			t.Fatalf("removing the planted run %s: %v", run, err)
		}
	}
	return f
}

// withoutPlantedWideningRuns drops the corpus's two planted widening runs from
// the ledger before the fixture is committed. The readings family itself stays —
// the corpus carries a loose prior reading record directly under it, and a walk
// row whose source directory is missing refuses the run.
func withoutPlantedWideningRuns(t *testing.T, root string) {
	t.Helper()
	for _, run := range []string{derivedCandidateRun, dispositionedWideningRun} {
		if err := os.RemoveAll(filepath.Join(root, filepath.FromSlash(rehearsalReadingsLedger), run)); err != nil {
			t.Fatalf("removing the planted run %s: %v", run, err)
		}
	}
}

// commitFixturePaths commits exactly the named paths under the fixture identity,
// leaving everything else in the working tree where it is. The operator's commit
// after an ingest is confined to what the ingest wrote, and a rehearsal that
// committed the whole tree would not be rehearsing that commit.
func commitFixturePaths(t *testing.T, root, message string, paths ...string) {
	t.Helper()
	ident := []string{"-c", "user.name=abcd cold-reading fixture", "-c", "user.email=fixture@example.invalid"}
	add := append(append([]string{}, ident...), "add", "--")
	gitFixture(t, root, append(add, paths...)...)
	gitFixture(t, root, append(append([]string{}, ident...), "commit", "-q", "-m", message)...)
}

// ---------------------------------------------------------------------------
// The fixture
// ---------------------------------------------------------------------------

// rehearsalFixture materialises the shared baseline corpus with the two edits
// this rehearsal needs, both applied before the fixture's commit so whatever
// they write is tracked.
//
// It reuses `materialise` rather than building a second fixture: the corpus, its
// committed preset, its planted widening runs and its transcript store are the
// state every other eval in this lane runs over, and a rehearsal over a
// different repository would rehearse a different thing.
func rehearsalFixture(t *testing.T) fixture {
	t.Helper()
	return materialise(t, variantBaseline, withOperatingDefinitions, withOutstandingReport)
}

// withOperatingDefinitions puts the repository's own four reading definitions
// into the fixture.
//
// The ingest resolves a run's position to its definition and hashes the file:
// the regime it enforces and half the instrument identity both come from there,
// so a rehearsal over stub definitions would rehearse a contract the shipped
// instrument does not have. The corpus carries one definition file as a sentinel
// plant, with no frontmatter at all, which is right for the read-block eval and
// useless here.
func withOperatingDefinitions(t *testing.T, root string) {
	t.Helper()
	for _, position := range everyPosition {
		name := "cold-reading-" + position + ".md"
		copyFile(t, filepath.Join("..", "agents", name), filepath.Join(root, "agents", name))
	}
}

// withOutstandingReport enables the outstanding-item report in the fixture's
// record-lint configuration, at the severity the rule pins in code.
//
// The corpus's configuration enables the schema rule alone, which is all the
// read-block eval needs. The report is a rule a repository turns on, so a
// rehearsal that did not turn it on would report nothing and assert nothing.
func withOutstandingReport(t *testing.T, root string) {
	t.Helper()
	path := filepath.Join(root, ".abcd", "record-lint.json")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading the fixture's record-lint configuration: %v", err)
	}
	var cfg map[string]any
	if err := json.Unmarshal(raw, &cfg); err != nil {
		t.Fatalf("decoding the fixture's record-lint configuration: %v", err)
	}
	rules, ok := cfg["rules"].(map[string]any)
	if !ok {
		t.Fatal("the fixture's record-lint configuration declares no rules block")
	}
	rules["reading_outstanding"] = map[string]any{
		"enabled": true, "severity": "info", "issues_dir": ".abcd/work/issues",
	}
	out, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		t.Fatalf("re-encoding the fixture's record-lint configuration: %v", err)
	}
	if err := os.WriteFile(path, append(out, '\n'), 0o644); err != nil {
		t.Fatalf("writing the fixture's record-lint configuration: %v", err)
	}
}

// ---------------------------------------------------------------------------
// Assembling
// ---------------------------------------------------------------------------

// parkedRun is the part of an assembly's JSON result an operator's next command
// needs. The field names are transcribed from the plugin page's "Report from the
// JSON" list.
type parkedRun struct {
	RunID            string   `json:"run_id"`
	Position         string   `json:"position"`
	TargetCommit     string   `json:"target_commit"`
	AssemblerVersion string   `json:"assembler_version"`
	ItemCount        int      `json:"item_count"`
	ManifestHash     string   `json:"manifest_hash"`
	OutDir           string   `json:"out_dir"`
	Artefacts        []string `json:"artefacts"`
	Written          bool     `json:"written"`
}

// assembleParked runs the operator's assembly — a position, a target, and
// nothing else — and returns the run it parked.
//
// No --out and no --dry-run: the ingest resolves a run id to the manifest the
// assembly parked in the local-tier run directory, so an assembly written
// anywhere else is a run no ingest can accept. That is the invocation the loop
// uses and therefore the one this rehearses (framework 8.5).
func assembleParked(t *testing.T, f fixture, position string) parkedRun {
	t.Helper()
	out, code := runIn(t, f.Root, []string{"HOME=" + f.Home},
		"reading", "assemble", "--position", position, "--target", "HEAD", "--json")
	if code != 0 {
		t.Fatalf("`abcd reading assemble --position %s --target HEAD` exited %d:\n%s",
			position, code, out)
	}
	var run parkedRun
	if err := json.Unmarshal([]byte(out), &run); err != nil {
		t.Fatalf("decoding the assembly result at %s: %v\n%s", position, err, out)
	}
	if run.RunID == "" || run.ManifestHash == "" || !run.Written {
		t.Fatalf("the assembly at %s reported %+v; a parked run needs an id, a manifest hash and "+
			"the two artefacts on disk", position, run)
	}
	if run.Position != position {
		t.Fatalf("the assembly reports the %s position for a run assembled at %s", run.Position, position)
	}
	return run
}

// requireArtefacts holds the assembly to where the plugin page says its two
// artefacts land: the local-tier run directory keyed by the run id, bundle and
// manifest side by side (framework 8.9 — the manifest is what makes the run
// judgeable, and a manifest nobody can find is none).
func requireArtefacts(t *testing.T, f fixture, run parkedRun) {
	t.Helper()
	want := rehearsalRunDir + "/" + run.RunID
	if filepath.ToSlash(run.OutDir) != want {
		t.Errorf("the assembly reports its artefacts at %q; with no --out they land in the "+
			"local-tier run directory %q", run.OutDir, want)
	}
	for _, name := range []string{bundleFile, manifestFile} {
		path := filepath.Join(f.Root, filepath.FromSlash(want), name)
		if !exists(path) {
			t.Errorf("the assembly at %s wrote no %s under %s", run.Position, name, want)
		}
	}
	// The manifest's own hash is what the output cites back, so a report that
	// disagreed with the bytes would send every legal payload to a refusal.
	if got := sha256File(t, filepath.Join(f.Root, filepath.FromSlash(want), manifestFile)); got != run.ManifestHash {
		t.Errorf("the assembly reports manifest_hash %s and the manifest it wrote hashes to %s; "+
			"the reference an output cites is the manifest's own content hash", run.ManifestHash, got)
	}
}

// requireDerivedCandidates asserts the comparative manifest names the derived
// widening run and carries a PLURAL candidate set.
//
// Companion 7.6 is the reason plurality is asserted rather than assumed: a
// widening run that returned one configuration leaves the comparative reading
// nothing to compare, and the verb records that as a run that was not exercised.
// A rehearsal over a single candidate would exercise the not-exercised path and
// report it as the comparative loop.
func requireDerivedCandidates(t *testing.T, f fixture, run parkedRun) {
	t.Helper()
	var m struct {
		CandidateRun string   `json:"candidate_run"`
		Candidates   int      `json:"candidates"`
		Exercised    *bool    `json:"exercised"`
		Criteria     []string `json:"criteria"`
	}
	raw, err := os.ReadFile(filepath.Join(f.Root, filepath.FromSlash(run.OutDir), manifestFile))
	if err != nil {
		t.Fatalf("reading the comparative manifest: %v", err)
	}
	if err := json.Unmarshal(raw, &m); err != nil {
		t.Fatalf("decoding the comparative manifest: %v", err)
	}
	if m.Candidates < 2 {
		t.Fatalf("the comparative manifest names %d candidate(s); companion 7.6 has the position "+
			"NOT EXERCISED below two, so a rehearsal under two rehearses the wrong outcome", m.Candidates)
	}
	if m.Exercised == nil || !*m.Exercised {
		t.Fatalf("the comparative manifest reports exercised=%v over %d candidates", m.Exercised, m.Candidates)
	}
	if m.CandidateRun == "" {
		t.Fatal("the comparative manifest names no candidate_run; the derived run is what makes the " +
			"characterisation attributable to a widening reading")
	}
	sorted := append([]string{}, m.Criteria...)
	sort.Strings(sorted)
	want := append([]string{}, rehearsalCriteria...)
	sort.Strings(want)
	if strings.Join(sorted, ",") != strings.Join(want, ",") {
		t.Errorf("the comparative manifest declares the criteria %v and the fixture's discipline "+
			"declares %v; the criteria are the record's and a reading never authors one (itd-191)",
			m.Criteria, rehearsalCriteria)
	}
}

// ---------------------------------------------------------------------------
// Ingesting
// ---------------------------------------------------------------------------

// readingRecordRef is one record an ingest wrote.
type readingRecordRef struct {
	ID   string `json:"id"`
	Path string `json:"path"`
}

// ingestResult is the part of the ingest's JSON this eval reads, transcribed
// from the plugin page's "Report from the JSON" list.
type ingestResult struct {
	RunID         string             `json:"run_id"`
	Position      string             `json:"position"`
	Regime        string             `json:"regime"`
	Records       []readingRecordRef `json:"records"`
	RefusedItems  []itemRefusal      `json:"refused_items"`
	RefusedCount  int                `json:"refused_count"`
	RunRecordPath string             `json:"run_record"`
	RefusalPath   string             `json:"refusal_record"`
	Error         string             `json:"error"`
}

// itemRefusal is one item the run refused, and why.
type itemRefusal struct {
	Ordinal int    `json:"ordinal"`
	Rule    string `json:"rule"`
	Field   string `json:"field"`
	Detail  string `json:"detail"`
}

// composeOutput builds one legal reading output for a parked run.
//
// It is built FROM the run — the manifest hash the assembly reported, the
// assembler version it stamped, and the definition file's own bytes — so a
// payload this helper produces is one the run could actually have returned. Only
// the deliberate mutations below make an illegal one, which is what keeps a
// refusal test about the rule it names rather than about an unrelated mismatch.
func composeOutput(t *testing.T, f fixture, run parkedRun, items []map[string]any) map[string]any {
	t.Helper()
	defPath := filepath.Join(f.Root, "agents", "cold-reading-"+run.Position+".md")
	if items == nil {
		items = []map[string]any{}
	}
	return map[string]any{
		"_type":           "abcd.reading.output/1",
		"run_id":          run.RunID,
		"position":        run.Position,
		"regime":          rehearsalRegimes[run.Position],
		"manifest_sha256": run.ManifestHash,
		"instrument": map[string]any{
			"model":             "rehearsal/no-model",
			"definition_sha256": sha256File(t, defPath),
			"assembler_version": run.AssemblerVersion,
		},
		"items": items,
	}
}

// writeOutput writes a payload OUTSIDE the fixture repository and returns the
// path.
//
// Outside is not incidental. An output written inside the tree is an untracked
// file the assembler's dirty gate may reach, and the plugin page refuses the
// bundle and the manifest as input wherever they are found for the same reason:
// a run must not read its own exhaust.
func writeOutput(t *testing.T, payload any) string {
	t.Helper()
	raw, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		t.Fatalf("encoding the reading output: %v", err)
	}
	path := filepath.Join(t.TempDir(), "reading-output.json")
	if err := os.WriteFile(path, append(raw, '\n'), 0o644); err != nil {
		t.Fatalf("writing the reading output: %v", err)
	}
	return path
}

// ingest runs the verb over one output file and returns its result and exit
// code.
//
// The stream is decoded as a SEQUENCE of JSON documents rather than one. A
// refusal that has something to disclose renders the result and then the error,
// which is two documents on one stream — and reading only the first would report
// a refusal record as absent exactly when the verb had just named one. The
// plugin page says to read the JSON on exit 2 as well, and this is what that
// costs.
func ingest(t *testing.T, f fixture, path string) (ingestResult, string, int) {
	t.Helper()
	out, code := runIn(t, f.Root, []string{"HOME=" + f.Home},
		"reading", "ingest", "--reading-json", path, "--json")
	var res ingestResult
	dec := json.NewDecoder(strings.NewReader(out))
	for {
		var one ingestResult
		if err := dec.Decode(&one); err != nil {
			break
		}
		if one.RunID != "" {
			res = one
		}
		if one.Error != "" {
			res.Error = one.Error
		}
	}
	if res.RunID == "" && code == 0 {
		t.Fatalf("the ingest exited 0 and reported no run:\n%s", out)
	}
	return res, out, code
}

// ingestAccepted composes a legal output, ingests it, and demands acceptance.
func ingestAccepted(t *testing.T, f fixture, run parkedRun, items []map[string]any) ingestResult {
	t.Helper()
	path := writeOutput(t, composeOutput(t, f, run, items))
	res, out, code := ingest(t, f, path)
	if code != 0 {
		t.Fatalf("`abcd reading ingest` refused a legal %s output (exit %d):\n%s", run.Position, code, out)
	}
	if res.RunID != run.RunID || res.Position != run.Position {
		t.Fatalf("the ingest reports run %s at %s for a run parked as %s at %s",
			res.RunID, res.Position, run.RunID, run.Position)
	}
	if res.Regime != rehearsalRegimes[run.Position] {
		t.Errorf("the ingest records the %s regime at %s; the regime is the definition's and the "+
			"record makes it %s", res.Regime, run.Position, rehearsalRegimes[run.Position])
	}
	return res
}

// requireCommittedRun holds an accepted ingest to where the plugin page says its
// records land: the reading records in the committed ledger keyed by the run,
// the manifest promoted beside the run metadata in the durable run directory,
// and the run metadata written last as the commit marker.
func requireCommittedRun(t *testing.T, f fixture, runID string, res ingestResult) {
	t.Helper()
	marker := rehearsalRunRecords + "/" + runID + "/" + rehearsalRunMarker
	if filepath.ToSlash(res.RunRecordPath) != marker {
		t.Errorf("the ingest reports its commit marker at %q, and the record puts it at %q; a run "+
			"without one never happened", res.RunRecordPath, marker)
	}
	if !exists(filepath.Join(f.Root, filepath.FromSlash(marker))) {
		t.Errorf("no commit marker on disk at %s", marker)
	}
	if !exists(filepath.Join(f.Root, filepath.FromSlash(rehearsalRunRecords+"/"+runID+"/"+manifestFile))) {
		t.Errorf("the run's manifest was not promoted to %s/%s", rehearsalRunRecords, runID)
	}
	for _, r := range res.Records {
		want := rehearsalReadingsLedger + "/" + runID + "/" + r.ID + ".md"
		if filepath.ToSlash(r.Path) != want {
			t.Errorf("the ingest reports record %s at %q; the readings family is keyed by the run "+
				"and the record puts it at %q", r.ID, r.Path, want)
		}
		if !exists(filepath.Join(f.Root, filepath.FromSlash(want))) {
			t.Errorf("no reading record on disk at %s", want)
		}
		// Identifiers are minted at ingest, never by the reading. A payload that
		// supplied one is refused below; this is the other half of that rule.
		if !strings.HasPrefix(r.ID, "rdi-") {
			t.Errorf("the ingest minted the identifier %q, which is not of the reading-item family", r.ID)
		}
	}
}

// ---------------------------------------------------------------------------
// The refusals
// ---------------------------------------------------------------------------

// refusalCase is one documented refusal: the position it applies at, the payload
// that trips it, and what the refusal must name.
type refusalCase struct {
	// Name is the rule, as the verb names it.
	Name string
	// Position is where the rule applies. A rule is exercised at the position
	// that carries it and nowhere else: the reserved names are per REGIME, and
	// the regime is the position's definition's.
	Position string
	// Why cites the document section or the record the rule holds.
	Why string
	// Items are the payload's items; nil leaves the base payload's single legal
	// item in place.
	Items []map[string]any
	// Mutate edits the composed payload, for the list-level rules.
	Mutate func(payload map[string]any)
	// Raw, when non-empty, is written verbatim instead of a composed payload —
	// the only way to rehearse a payload that is not JSON at all.
	Raw string
	// Want are the fragments the refusal must carry. Every case names the item
	// or the field it refused, which is what an operator reads.
	Want []string
	// Proven says the run's identity is proven before the refusal, so the
	// refusal must leave a refusal record. A refusal reached before that point
	// writes nothing durable anywhere.
	Proven bool
}

// legalItem is one clean item at a position, the base every refusal mutates.
//
// Every refusal case starts from a payload that WOULD be accepted, so a case
// proves the rule it names rather than passing against a verb that refuses
// everything.
func legalItem(position string) map[string]any {
	switch position {
	case posWidening:
		return map[string]any{
			"pattern":        "the stated invariant",
			"configuration":  "a configuration the construal admits",
			"what_admits_it": "the invariant the constraints chapter already states",
		}
	case posEntailment:
		return map[string]any{
			"pattern":         "what the design commits to by being what it is",
			"claim_surfaced":  "a claim the articulation does not state",
			"claim_type":      "criterion",
			"what_implies_it": "the constraint the brief already states",
		}
	case posComparative:
		return map[string]any{
			"pattern":          "how options of this shape ordinarily behave",
			"candidate_id":     rehearsalCandidates[0],
			"criterion":        rehearsalCriteria[0],
			"characterisation": "how a configuration of this shape ordinarily behaves",
		}
	default:
		return map[string]any{
			"pattern":            "a stated constraint the tree does not keep",
			"tension":            "the tree and the claim record disagree",
			"constraint_in_play": "the invariant the constraints chapter states",
			"why_a_tension":      "one of the two has to give",
		}
	}
}

// withField returns the position's legal item carrying one extra key.
func withField(position, key string, value any) map[string]any {
	item := legalItem(position)
	item[key] = value
	return item
}

// rehearsalRefusals is every refusal `reading ingest` documents, at the position
// it applies at.
//
// The list is transcribed from commands/reading.md ("What is refused, and how
// far" and "The supply regime is the definition's"), which is the operator's own
// account of the verb. A refusal the page documents and this table does not
// carry is a refusal no rehearsal exercises.
func rehearsalRefusals() []refusalCase {
	return []refusalCase{
		// The reserved names, one row per regime that has one. The generative
		// regime has no row and needs none: its licence is the widest and the
		// constraint on it falls at admission.
		{
			Name:     "reserved-name-evaluative",
			Position: posComparative,
			Why: "an evaluative reading characterises candidates against criteria; ordering, " +
				"scoring or recommending among them is not its licence",
			Items: []map[string]any{
				withField(posComparative, "rank", "2"),
				withField(posComparative, "score", "0.8"),
				withField(posComparative, "order", "first"),
				withField(posComparative, "recommended", "yes"),
			},
			Want:   []string{"item 1", "reserved", "rank", "score", "order", "recommended"},
			Proven: true,
		},
		{
			Name:     "reserved-name-registrative",
			Position: posDetection,
			Why: "a registrative reading names a tension and the constraint in play; proposing " +
				"the resolution is not its licence",
			Items: []map[string]any{
				withField(posDetection, "fix", "rewrite the gate"),
				withField(posDetection, "remedy", "rewrite the gate"),
				withField(posDetection, "resolution", "rewrite the gate"),
			},
			Want:   []string{"item 1", "reserved", "fix", "remedy", "resolution"},
			Proven: true,
		},
		{
			Name:     "reserved-name-explicative",
			Position: posEntailment,
			Why: "an explicative reading surfaces a claim; dispositioning one is the researcher's " +
				"act, and it is a separate record",
			Items: []map[string]any{
				withField(posEntailment, "disposition", "accepted"),
				withField(posEntailment, "status", "open"),
			},
			Want:   []string{"item 1", "reserved", "disposition", "status"},
			Proven: true,
		},
		// The identifier. Ids are minted by the verb; a payload supplying one is
		// refused as an unknown field, and the refusal says why there is no field
		// for it.
		{
			Name:     "payload-supplied-identifier",
			Position: posWidening,
			Why:      "an item carries no identifier: identifiers are minted by the verb",
			Items:    []map[string]any{withField(posWidening, "id", "rdi-2609020000000001")},
			Want:     []string{"item 1", "unknown-field", "id"},
			Proven:   true,
		},
		// The two comparative checks, both against the run's own manifest rather
		// than against the payload's account of itself.
		{
			Name:     "unknown-candidate",
			Position: posComparative,
			Why: "a comparative reading characterises the candidates it was handed and no others " +
				"(the candidate set is the manifest's)",
			Items: []map[string]any{func() map[string]any {
				item := legalItem(posComparative)
				item["candidate_id"] = "rdi-999"
				return item
			}()},
			Want:   []string{"item 1", "unknown-candidate", "rdi-999"},
			Proven: true,
		},
		{
			Name:     "undeclared-criterion",
			Position: posComparative,
			Why:      "the criteria are a declared, recorded discipline and a reading never authors one (itd-191)",
			Items: []map[string]any{func() map[string]any {
				item := legalItem(posComparative)
				item["criterion"] = "Elegance"
				return item
			}()},
			Want:   []string{"item 1", "undeclared-criterion", "Elegance", "itd-191"},
			Proven: true,
		},
		// The item shape.
		{
			Name:     "named-provenance",
			Position: posDetection,
			Why:      "every item at every regime carries the pattern it was read under, without exception",
			Items: []map[string]any{func() map[string]any {
				item := legalItem(posDetection)
				item["pattern"] = "   "
				return item
			}()},
			Want:   []string{"item 1", "named-provenance", "pattern"},
			Proven: true,
		},
		{
			Name:     "missing-body-field",
			Position: posDetection,
			Why:      "a declared body field holding nothing states nothing",
			Items: []map[string]any{func() map[string]any {
				item := legalItem(posDetection)
				item["why_a_tension"] = ""
				return item
			}()},
			Want:   []string{"item 1", "missing-body-field", "why_a_tension"},
			Proven: true,
		},
		{
			Name:     "closed-vocabulary",
			Position: posEntailment,
			Why:      "claim_type is a closed set the definitions instruct and spc-63 tables",
			Items: []map[string]any{func() map[string]any {
				item := legalItem(posEntailment)
				item["claim_type"] = "speculative"
				return item
			}()},
			Want:   []string{"item 1", "closed-vocabulary", "claim_type"},
			Proven: true,
		},
		// The list-level rules. The first three are reached BEFORE the run's
		// identity is proven, so they write nothing durable anywhere.
		{
			Name:     "wrong-type-tag",
			Position: posDetection,
			Why:      "a wrong _type is refused before the run's identity is proven",
			Mutate:   func(p map[string]any) { p["_type"] = "abcd.reading.manifest" },
			Want:     []string{"_type"},
		},
		{
			Name:     "run-resolves-to-no-manifest",
			Position: posDetection,
			Why:      "a run id that resolves to no parked manifest is refused before identity is proven",
			Mutate:   func(p map[string]any) { p["run_id"] = "rdg-2609029999999999" },
			Want:     []string{"rdg-2609029999999999"},
		},
		{
			Name:     "manifest-hash-disagrees",
			Position: posDetection,
			Why:      "a manifest hash that disagrees is refused before identity is proven",
			Mutate: func(p map[string]any) {
				p["manifest_sha256"] = strings.Repeat("0", 64)
			},
			Want: []string{"manifest"},
		},
		{
			Name:     "malformed-payload",
			Position: posDetection,
			Why:      "refusal is reserved for a malformed payload (framework 13)",
			Raw:      "{ this is not JSON",
			Want:     []string{"reading ingest"},
		},
		// These three are reached AFTER the run's identity is proven, so each
		// leaves a refusal record naming the reason and no items.
		{
			Name:     "regime-disagrees-with-the-definition",
			Position: posDetection,
			Why:      "the regime is the definition's property, and a self-declared regime that disagrees refuses the run",
			Mutate:   func(p map[string]any) { p["regime"] = "generative" },
			Want:     []string{"regime", "generative"},
			Proven:   true,
		},
		{
			Name:     "instrument-claims-another-definition",
			Position: posDetection,
			Why:      "the definition's content hash is half of the instrument's identity, and it is recomputed at ingest",
			Mutate: func(p map[string]any) {
				p["instrument"].(map[string]any)["definition_sha256"] = strings.Repeat("a", 64)
			},
			Want:   []string{"definition_sha256"},
			Proven: true,
		},
		{
			Name:     "instrument-claims-another-assembler",
			Position: posDetection,
			Why:      "the assembler version is the other half, and the manifest carries what it was",
			Mutate: func(p map[string]any) {
				p["instrument"].(map[string]any)["assembler_version"] = "0.0.0+not-this-one"
			},
			Want:   []string{"assembler_version"},
			Proven: true,
		},
		{
			Name:     "blank-instrument-part",
			Position: posDetection,
			Why: "blankness is judged on a folded copy everywhere the verb judges it, the " +
				"instrument's three parts included. It is an ENVELOPE rule, checked before the run " +
				"id is resolved to a parked manifest, so it writes nothing durable anywhere \u2014 " +
				"which is why this row is not `Proven` and the zero-width model below is refused " +
				"as though the field were absent",
			Mutate: func(p map[string]any) {
				p["instrument"].(map[string]any)["model"] = "\u200b"
			},
			Want: []string{"model"},
		},
		{
			Name:     "every-item-refused",
			Position: posDetection,
			Why: "a payload in which no item survived is a list-level refusal: a run whose every " +
				"finding was refused is a different fact from a run that returned nothing (framework 13)",
			Items: []map[string]any{
				withField(posDetection, "fix", "rewrite the gate"),
			},
			Want:   []string{"item 1", "reserved", "fix"},
			Proven: true,
		},
	}
}

// rehearseRefusal parks a fresh run at the case's position, ingests the payload
// the case describes, and asserts three things: the verb refused, the refusal
// names what it refused, and no partial record was written.
//
// A fresh run per case is forced by the rerun rule — once a run id has an
// outcome, ingesting it again is refused — and it is also what makes each case
// independent: a case that inherited another's refusal record would assert
// something about the first case's run.
func rehearseRefusal(t *testing.T, f fixture, c refusalCase, pre map[string]parkedRun) {
	t.Helper()
	run, parkedEarlier := pre[c.Name]
	if !parkedEarlier {
		run = assembleParked(t, f, c.Position)
	}
	if run.Position != c.Position {
		t.Fatalf("the %s case runs at %s and was handed a run parked at %s",
			c.Name, c.Position, run.Position)
	}

	var path string
	if c.Raw != "" {
		path = filepath.Join(t.TempDir(), "reading-output.json")
		if err := os.WriteFile(path, []byte(c.Raw), 0o644); err != nil {
			t.Fatalf("writing the malformed payload: %v", err)
		}
	} else {
		items := c.Items
		if items == nil {
			items = []map[string]any{legalItem(c.Position)}
		}
		payload := composeOutput(t, f, run, items)
		if c.Mutate != nil {
			c.Mutate(payload)
		}
		path = writeOutput(t, payload)
	}

	res, out, code := ingest(t, f, path)
	if code == 0 {
		t.Fatalf("`abcd reading ingest` ACCEPTED the %s payload at %s; the verb documents this "+
			"refusal (%s)\n%s", c.Name, c.Position, c.Why, out)
	}
	for _, want := range c.Want {
		if !strings.Contains(out, want) {
			t.Errorf("the %s refusal does not name %q; an operator reads the refusal to find the "+
				"item it refused (%s)\n%s", c.Name, want, c.Why, out)
		}
	}

	// No partial records, whichever kind of refusal this is. No reading record
	// for the run may reach the committed ledger, and the run must not have
	// reached its commit marker: a refused run never happened.
	if n := readingRecordsUnder(t, f, run.RunID); n != 0 {
		t.Errorf("the refused run %s left %d reading record(s) in the committed ledger at %s/%s; a "+
			"refusal writes its refusal record and nothing else",
			run.RunID, n, rehearsalReadingsLedger, run.RunID)
	}
	marker := filepath.Join(f.Root, filepath.FromSlash(rehearsalRunRecords), run.RunID, rehearsalRunMarker)
	if exists(marker) {
		t.Errorf("the refused run %s reached its commit marker; a run without one never happened, "+
			"and a refused run must not have one", run.RunID)
	}

	// The refusal record itself, which is the disclosure half: a refusal after
	// the run is proven leaves a record naming the reason and no items; one
	// before writes nothing durable anywhere.
	refusal := filepath.Join(f.Root, filepath.FromSlash(rehearsalRunRecords), run.RunID, rehearsalRefusalRecord)
	switch {
	case c.Proven && !exists(refusal):
		t.Errorf("the %s refusal left no %s under %s/%s; once the run id resolves to a parked "+
			"manifest whose hash matches, every list-level refusal records itself",
			c.Name, rehearsalRefusalRecord, rehearsalRunRecords, run.RunID)
	case !c.Proven && exists(refusal):
		t.Errorf("the %s refusal wrote a refusal record under %s/%s; it is reached before the "+
			"run's identity is proven, and there is no proven run to record against",
			c.Name, rehearsalRunRecords, run.RunID)
	}
	if c.Proven && res.RefusalPath == "" && exists(refusal) {
		t.Errorf("the %s refusal wrote a refusal record and reported none; the JSON is rendered on "+
			"a refusal exactly so the operator is told", c.Name)
	}
}

// ---------------------------------------------------------------------------
// Dispositioning, promoting, and the outstanding report
// ---------------------------------------------------------------------------

// dispositionResult is the part of the verb's JSON this eval reads.
type dispositionResult struct {
	ID       string `json:"id"`
	Item     string `json:"item"`
	State    string `json:"state"`
	Position string `json:"position"`
	Path     string `json:"path"`
}

// rehearseDispositions runs the researcher's half of the loop: an acceptance
// with its grounds, a hold with its exit condition, and the refusals that hold
// the vocabulary to what the framework fixes.
//
// Framework 11.2 fixes the four states — accepted, rejected, declined, held —
// and 11.3 fixes what each carries: grounds on every state but held, and an
// epistemic exit condition on held, because a hold that cannot say what would
// end it is a parking space. Companion 4.3 is the same rule from the other side.
// Availability is per POSITION and is read off the item's own record, never
// supplied.
func rehearseDispositions(t *testing.T, f fixture, detectionItem string, wideningItems []string) {
	t.Helper()

	// The acceptance, on the detection item, with its grounds.
	accepted := disposition(t, f, detectionItem, "--state", "accepted",
		"--grounds", "the tension is real and the record is the side that has to move")
	if accepted.State != "accepted" || accepted.Position != posDetection {
		t.Errorf("the acceptance records state %q at position %q, want accepted at %s",
			accepted.State, accepted.Position, posDetection)
	}
	want := rehearsalDispositions + "/" + detectionItem + "/" + accepted.ID + ".md"
	if filepath.ToSlash(accepted.Path) != want {
		t.Errorf("the disposition landed at %q; the family is keyed by the ITEM and the record "+
			"puts it at %q", accepted.Path, want)
	}
	if !exists(filepath.Join(f.Root, filepath.FromSlash(want))) {
		t.Errorf("no disposition record on disk at %s", want)
	}

	// The hold, on a widening item, with its exit condition. Held is not
	// available at the widening position, so the hold goes on the SECOND
	// widening item only if it is available there; the record makes held
	// available at entailment, comparative and detection, so the hold is
	// rehearsed where it is available — on the detection run's own item is
	// impossible (it now carries a standing answer), so a second detection run
	// supplies one.
	held := disposition(t, f, mustSecondDetectionItem(t, f), "--state", "held",
		"--exit-condition", "the closing run returns the same tension against a moved record")
	if held.State != "held" {
		t.Errorf("the hold records state %q, want held", held.State)
	}

	// The refusals. A blank grounds and a blank exit condition are refused, and
	// neither writes anything: framework 11.3 makes the grounds and the exit
	// condition constitutive of the answer rather than decoration on it.
	for _, c := range []struct {
		name string
		item string
		args []string
		want string
	}{
		{
			name: "accepted-with-blank-grounds",
			item: mustSecondDetectionItem(t, f),
			args: []string{"--state", "accepted", "--grounds", "   "},
			want: "disposition_grounds",
		},
		{
			name: "held-with-blank-exit-condition",
			item: mustSecondDetectionItem(t, f),
			args: []string{"--state", "held", "--exit-condition", "   "},
			want: "exit_condition",
		},
	} {
		t.Run(c.name, func(t *testing.T) {
			args := append([]string{"capture", "disposition", c.item}, c.args...)
			out, code := runIn(t, f.Root, []string{"HOME=" + f.Home}, args...)
			if code == 0 {
				t.Fatalf("a disposition with a blank %s was accepted:\n%s", c.want, out)
			}
			if !strings.Contains(out, c.want) {
				t.Errorf("the refusal does not name %q:\n%s", c.want, out)
			}
			dir := filepath.Join(f.Root, filepath.FromSlash(rehearsalDispositions), c.item)
			if entries, err := os.ReadDir(dir); err == nil && len(entries) > 0 {
				t.Errorf("the refused disposition wrote %d file(s) under %s/%s; a refusal writes nothing",
					len(entries), rehearsalDispositions, c.item)
			}
		})
	}

	// `declined` is the widening position's own state and is refused everywhere
	// else. It is the answer that costs nothing epistemically — a proposal
	// weighed and not taken up — and at a position whose items are findings
	// rather than proposals there is nothing to decline (framework 11.2).
	t.Run("declined-is-available-at-widening", func(t *testing.T) {
		res := disposition(t, f, wideningItems[0], "--state", "declined",
			"--grounds", "the configuration is admissible and this iteration does not take it up")
		if res.State != "declined" || res.Position != posWidening {
			t.Errorf("the decline records state %q at %q, want declined at %s",
				res.State, res.Position, posWidening)
		}
	})
	t.Run("declined-is-refused-at-detection", func(t *testing.T) {
		item := mustSecondDetectionItem(t, f)
		out, code := runIn(t, f.Root, []string{"HOME=" + f.Home}, "capture", "disposition", item,
			"--state", "declined", "--grounds", "nothing to decline here")
		if code == 0 {
			t.Fatalf("`declined` was accepted at the detection position:\n%s", out)
		}
		if !strings.Contains(out, "declined") {
			t.Errorf("the refusal does not name the state it refused:\n%s", out)
		}
	})
}

// disposition runs the verb and demands success.
func disposition(t *testing.T, f fixture, item string, args ...string) dispositionResult {
	t.Helper()
	full := append([]string{"capture", "disposition", item}, args...)
	full = append(full, "--json")
	out, code := runIn(t, f.Root, []string{"HOME=" + f.Home}, full...)
	if code != 0 {
		t.Fatalf("`abcd capture disposition %s %s` exited %d:\n%s",
			item, strings.Join(args, " "), code, out)
	}
	var res dispositionResult
	if err := json.Unmarshal([]byte(out), &res); err != nil {
		t.Fatalf("decoding the disposition result: %v\n%s", err, out)
	}
	if res.Item != item {
		t.Fatalf("the disposition names item %s for an answer to %s", res.Item, item)
	}
	return res
}

// mustSecondDetectionItem parks and ingests one more detection run and returns
// its single item, so a case needing an unanswered item has one.
//
// An item carrying a standing answer refuses a second one that does not cite it,
// which is right and makes an item a single-use subject. Assembling another run
// is what the operator would do, and it is cheap on this corpus.
func mustSecondDetectionItem(t *testing.T, f fixture) string {
	t.Helper()
	run := assembleParked(t, f, posDetection)
	res := ingestAccepted(t, f, run, []map[string]any{legalItem(posDetection)})
	if len(res.Records) != 1 {
		t.Fatalf("a one-item detection ingest recorded %d record(s)", len(res.Records))
	}
	return res.Records[0].ID
}

// rehearsePromote graduates an accepted detection item into a draft intent and
// asserts the join the promotion mints.
//
// A widening item is NOT promoted here, and the reason is the design's: at that
// position acceptance IS admission, the item is gated on the comparative
// characterisation that precedes it, and the admission record is Phase B. A
// detection item is the case the verb serves this cycle.
func rehearsePromote(t *testing.T, f fixture, runID, item string) {
	t.Helper()
	out, code := runIn(t, f.Root, []string{"HOME=" + f.Home},
		"capture", "promote", item, "--json")
	if code != 0 {
		t.Fatalf("`abcd capture promote %s` exited %d:\n%s", item, code, out)
	}
	var res struct {
		IntentID   string `json:"intent_id"`
		IntentPath string `json:"intent_path"`
		Path       string `json:"path"`
	}
	if err := json.Unmarshal([]byte(out), &res); err != nil {
		t.Fatalf("decoding the promote result: %v\n%s", err, out)
	}
	path := res.IntentPath
	if path == "" {
		path = res.Path
	}
	if path == "" {
		t.Fatalf("the promotion reported no draft path:\n%s", out)
	}
	if !strings.HasPrefix(filepath.ToSlash(path), rehearsalDrafts+"/") {
		t.Errorf("the promotion minted its draft at %q, and a draft intent lives under %q",
			path, rehearsalDrafts)
	}
	raw, err := os.ReadFile(filepath.Join(f.Root, filepath.FromSlash(path)))
	if err != nil {
		t.Fatalf("reading the minted draft: %v", err)
	}
	// The value the record carries, spelled out: a pointer that resolves back to
	// the pair the draft was graduated from. It is one scalar, and both halves
	// are load-bearing — the run alone names no item and the item alone names no
	// run.
	want := "contributed-by-reading " + runID + "/" + item
	if !strings.Contains(string(raw), want) {
		t.Errorf("the minted draft does not carry `origin: %s`; the third arrival path is what "+
			"makes a reading-occasioned intent attributable to the reading that occasioned "+
			"it\n%s", want, string(raw))
	}
	if !strings.Contains(string(raw), item) {
		t.Errorf("the minted draft names no back-edge to %s", item)
	}
}

// rehearseOutstanding runs the outstanding-item report over the fixture and
// demands that every item nobody has answered is named.
//
// The report is a REPORT and never a gate — its severity is pinned to info in
// code — and that is exactly why it is the last step of the loop: nothing in the
// disposition vocabulary means "already covered", so an unanswered item has no
// state to sit in and would otherwise appear nowhere at all.
func rehearseOutstanding(t *testing.T, f fixture, wideningItems, detectionItems []string) {
	t.Helper()
	report := recordLint(t, f)

	// Everything answered above must be absent from the undispositioned roster,
	// and everything not answered must be on it. The second half is the report's
	// purpose; the first is what stops it reporting everything unconditionally.
	answered := map[string]bool{}
	if len(detectionItems) > 0 {
		answered[detectionItems[0]] = true
	}
	if len(wideningItems) > 0 {
		answered[wideningItems[0]] = true
	}

	outstanding := map[string]bool{}
	for _, line := range strings.Split(report, "\n") {
		if !strings.Contains(line, "reading_outstanding") || !strings.Contains(line, "carries no disposition") {
			continue
		}
		for _, id := range readingIDsIn(line) {
			outstanding[id] = true
		}
	}
	if len(outstanding) == 0 {
		t.Fatalf("the outstanding report names no undispositioned item, and this fixture holds "+
			"several; a report that names nothing is one nobody can act on\n%s", report)
	}
	// The unanswered widening item, which is the one the loop deliberately left.
	if len(wideningItems) > 1 && !outstanding[wideningItems[1]] {
		t.Errorf("the outstanding report does not name %s, which carries no disposition\n%s",
			wideningItems[1], report)
	}
	for id := range answered {
		if outstanding[id] {
			t.Errorf("the outstanding report names %s as undispositioned, and it carries a standing "+
				"answer\n%s", id, report)
		}
	}
}

// processHome is the HOME this test process started with, captured at package
// initialisation — which is before TestMain and therefore before any test
// redirects it.
//
// It is needed because `gittest.Env`, which every fixture's git call goes
// through, pins the PROCESS HOME to a test-owned temporary directory the first
// time it runs. That is right for git and wrong for the Go toolchain: a `go`
// invocation inheriting it writes a fresh module cache under the fixture's own
// temporary tree, whose files are read-only by design, and the test then fails
// in `TempDir RemoveAll` cleanup rather than on anything it asserted.
var processHome = os.Getenv("HOME")

// recordLint runs the record gate over the fixture and returns its findings.
//
// It goes through `go run ./cmd/record-lint` because that binary is the gate's
// own front door — `make record-lint` is one line around it — and the harness
// builds only `abcd`. The report's findings are pinned to `info` in code, so the
// command exits 0 with them on stdout; a non-zero exit here would be a blocker
// from some other rule, and the output is returned either way so the assertions
// read what the gate actually said.
func recordLint(t *testing.T, f fixture) string {
	t.Helper()
	cmd := exec.Command("go", "run", "./cmd/record-lint", "--root", f.Root)
	cmd.Dir = ".."
	// The toolchain gets the process's own HOME back (see processHome), so the
	// module cache stays where it belongs.
	env := make([]string, 0, len(os.Environ())+1)
	for _, kv := range os.Environ() {
		if strings.HasPrefix(kv, "HOME=") || strings.HasPrefix(kv, "PWD=") {
			continue
		}
		env = append(env, kv)
	}
	cmd.Env = append(env, "HOME="+processHome)
	out, err := cmd.CombinedOutput()
	if err != nil {
		if _, ok := err.(*exec.ExitError); !ok {
			t.Fatalf("could not launch record-lint: %v", err)
		}
	}
	return string(out)
}

// readingIDsIn returns the reading-item identifiers a line names.
func readingIDsIn(line string) []string {
	var out []string
	for _, field := range strings.FieldsFunc(line, func(r rune) bool {
		return r == ' ' || r == '(' || r == ')' || r == ',' || r == '`' || r == '\t'
	}) {
		if strings.HasPrefix(field, "rdi-") {
			out = append(out, strings.TrimSuffix(field, ".md"))
		}
	}
	return out
}

// ---------------------------------------------------------------------------
// Small shared helpers
// ---------------------------------------------------------------------------

// readingRecordsUnder counts the reading records one run left in the committed
// ledger. It counts FILES rather than probing the directory: an ingest that
// created the run's directory and wrote nothing into it has recorded nothing,
// and a count is what distinguishes those two.
func readingRecordsUnder(t *testing.T, f fixture, runID string) int {
	t.Helper()
	entries, err := os.ReadDir(filepath.Join(f.Root, filepath.FromSlash(rehearsalReadingsLedger), runID))
	if err != nil {
		if os.IsNotExist(err) {
			return 0
		}
		t.Fatalf("reading the ledger directory of run %s: %v", runID, err)
	}
	n := 0
	for _, e := range entries {
		if !e.IsDir() && strings.HasPrefix(e.Name(), "rdi-") && strings.HasSuffix(e.Name(), ".md") {
			n++
		}
	}
	return n
}

// exists reports whether a path is there at all.
func exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// sha256File is the content hash of one file, which is how both the manifest
// reference and the definition half of the instrument identity are spelled.
func sha256File(t *testing.T, path string) string {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("hashing %s: %v", path, err)
	}
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:])
}
