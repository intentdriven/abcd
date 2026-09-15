package cli

// reading_help_truth_test.go holds the ingest verb's own Long description to the
// code it describes (iss-2608311518199679).
//
// The description is not decoration. docs/reference/cli/commands.md is GENERATED
// from it, so one wrong sentence in the Long field is a wrong sentence in two
// shipped surfaces at once — the terminal an operator reads and the reference
// page they are pointed at. A schema test cannot catch that: the sentence and
// the behaviour it describes are checked against each other here, and the three
// sentences that were measured false are pinned so the next drift fails a test
// rather than a reader.

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/capture"
	"github.com/intentdriven/abcd/internal/core/issueschema"
	"github.com/intentdriven/abcd/internal/core/reading"
)

// ingestLong returns the ingest verb's Long description, unwrapped: the source
// wraps it for a terminal, and a sentence assertion is about the sentence rather
// than about where the line broke.
func ingestLong(t *testing.T) string {
	t.Helper()
	for _, c := range NewRootCommand().Commands() {
		if c.Name() != "reading" {
			continue
		}
		for _, sub := range c.Commands() {
			if sub.Name() == "ingest" {
				return strings.Join(strings.Fields(sub.Long), " ")
			}
		}
	}
	t.Fatal("the command tree registers no `reading ingest` sub-verb")
	return ""
}

// TestIngestHelpNamesTheReservedTableAsTheGateReadsIt: the help describes the
// reserved-name rule by pointing at the table the gate reads, per regime, and it
// says the generative regime has no row — because it has none, and a help that
// says "each regime has reserved names" sends a reader looking for a generative
// row that does not exist.
func TestIngestHelpNamesTheReservedTableAsTheGateReadsIt(t *testing.T) {
	if got := reading.ReservedNames[reading.RegimeGenerative]; len(got) != 0 {
		t.Fatalf("the generative regime declares reserved names %v; the help says it declares none", got)
	}
	for _, regime := range []string{reading.RegimeEvaluative, reading.RegimeRegistrative, reading.RegimeExplicative} {
		if len(reading.ReservedNames[regime]) == 0 {
			t.Errorf("the %s regime has no row in the reserved-name table, so the help's "+
				"per-regime claim describes a table with one populated row fewer", regime)
		}
	}

	long := ingestLong(t)
	for _, want := range []string{
		"an item carrying a reserved name as one of its own fields is refused with the licence stated",
		"the generative regime has no row",
	} {
		if !strings.Contains(long, want) {
			t.Errorf("the ingest help does not state %q:\n%s", want, long)
		}
	}
	// The prose-signature registry is the half of the gate that is being
	// withdrawn, and the help must not rest a reader's expectation on it: a
	// signature refuses nothing today, and a description of a mechanism in
	// motion is the next false sentence.
	for _, banned := range []string{"registry of named signatures", "raises a review flag"} {
		if strings.Contains(long, banned) {
			t.Errorf("the ingest help restates %q, a mechanism no refusal rests on:\n%s", banned, long)
		}
	}
}

// TestIngestHelpDescribesTheDeletesARefusalMakes: the help's claim about what a
// refusal writes and deletes is checked against a refusal.
//
// A refused run after its identity is proven writes exactly one thing — its
// refusal record — and deletes exactly one: the never-committed records of an
// earlier attempt at its OWN run id. Another run's orphan is left where it was.
// The help said "and nothing else", which omits the delete the code makes on
// every refusal path.
//
// The same shape pins the disclosure the refusal renders. `pending_stages` is
// the operator's list of stages STILL STANDING, so it names the other run's
// orphan and it must not name the stage this run's own rollback just cleared
// (iss-2609020848468450).
func TestIngestHelpDescribesTheDeletesARefusalMakes(t *testing.T) {
	srcRoot := repoRootFromTest(t)
	repo := readingRepo(t)
	t.Chdir(repo)

	runID, manifestHash, def := parkedRunForIngest(t, srcRoot, repo, "detection")

	write := func(rel, body string) {
		abs := filepath.Join(repo, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(abs, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	ledgerRecord := func(run, item string) string {
		return capture.LedgerRelPath + "/" + issueschema.ReadingsDir + "/" + run + "/" + item + ".md"
	}

	// This run's own half-landed record: an earlier attempt at the same run id
	// that died between its ledger write and its commit marker.
	const ownItem = "rdi-2608310000000101"
	ownRecord := ledgerRecord(runID, ownItem)
	write(ownRecord, "---\nid: "+ownItem+"\n---\n\nthe half-landed body\n")
	// and the stage that attempt left standing beside it: the refusal's own
	// rollback clears this one, which is what the pending assertion below turns on.
	write(reading.IngestStageDir+"/"+runID+"/stage.json",
		`{"_type":"`+reading.StageType+`","run_id":"`+runID+`","records":["`+ownItem+`"]}`)

	// Somebody else's orphan: a stage and the record its run had landed.
	const otherRun, otherItem = "rdg-2608310000000102", "rdi-2608310000000103"
	otherRecord := ledgerRecord(otherRun, otherItem)
	const otherBody = "---\nid: " + otherItem + "\n---\n\nanother run's body\n"
	write(reading.IngestStageDir+"/"+otherRun+"/stage.json",
		`{"_type":"`+reading.StageType+`","run_id":"`+otherRun+`","records":["`+otherItem+`"]}`)
	write(otherRecord, otherBody)

	// A regime the definition does not state: a list-level refusal, after the
	// run's identity is proven.
	outPath := detectionPayloadFile(t, runID, manifestHash, "generative", def)
	rendered, err := runCLIErr(t, "reading", "ingest", "--reading-json", outPath, "--json")
	if err == nil {
		t.Fatalf("a regime mismatch exited 0:\n%s", rendered)
	}
	var res reading.IngestResult
	if jsonErr := json.Unmarshal(rendered, &res); jsonErr != nil {
		t.Fatalf("a refusal rendered no JSON result: %v\n%s", jsonErr, rendered)
	}

	// (a) it writes its refusal record, and that is the only thing under the
	// run's own directory.
	if res.RefusalPath == "" {
		t.Fatal("the refusal wrote no refusal record")
	}
	runDir := filepath.Join(repo, filepath.FromSlash(reading.ReadingsRecordDir), runID)
	entries, readErr := os.ReadDir(runDir)
	if readErr != nil {
		t.Fatalf("read the refused run's directory: %v", readErr)
	}
	if len(entries) != 1 || entries[0].Name() != reading.RefusalFileName {
		var names []string
		for _, e := range entries {
			names = append(names, e.Name())
		}
		t.Errorf("the refused run's directory holds %v, want only %s", names, reading.RefusalFileName)
	}

	// (b) it deletes its OWN run's half-landed records, and says so.
	if _, statErr := os.Stat(filepath.Join(repo, filepath.FromSlash(ownRecord))); !os.IsNotExist(statErr) {
		t.Errorf("the refusal left its own never-committed record at %s (%v)", ownRecord, statErr)
	}
	var sawOwn bool
	for _, id := range res.RolledBack {
		if id == ownItem {
			sawOwn = true
		}
	}
	if !sawOwn {
		t.Errorf("the refusal rolled back %v, which does not name %s", res.RolledBack, ownItem)
	}

	// (c) and it touches no OTHER run's durable state.
	got, readErr := os.ReadFile(filepath.Join(repo, filepath.FromSlash(otherRecord)))
	if readErr != nil {
		t.Fatalf("a refused run deleted another run's committed record: %v", readErr)
	}
	if string(got) != otherBody {
		t.Errorf("another run's committed record changed under a refused run:\n%s", got)
	}
	var sawPending bool
	for _, id := range res.PendingStages {
		if id == otherRun {
			sawPending = true
		}
	}
	if !sawPending {
		t.Errorf("the refusal reported pending stages %v, which does not name the orphan %s",
			res.PendingStages, otherRun)
	}

	// (d) and the disclosure is exact: the stage the refusal itself cleared is
	// gone from the tree, so it is not one an operator is told is still pending.
	var clearedOwn bool
	for _, id := range res.ClearedStages {
		if id == runID {
			clearedOwn = true
		}
	}
	if !clearedOwn {
		t.Errorf("the refusal cleared stages %v, which does not name its own run %s; the "+
			"pending assertion below rests on that clear having happened", res.ClearedStages, runID)
	}
	if _, statErr := os.Stat(filepath.Join(repo, filepath.FromSlash(reading.IngestStageDir), runID)); !os.IsNotExist(statErr) {
		t.Errorf("the refusal left its own stage standing at %s/%s (%v)", reading.IngestStageDir, runID, statErr)
	}
	for _, id := range res.PendingStages {
		if id == runID {
			t.Errorf("the refusal reported pending stages %v, which names its own stage %s — "+
				"the stage it had just cleared (cleared: %v)", res.PendingStages, runID, res.ClearedStages)
		}
	}

	// The help has to say all of that, because the operator reads the help.
	long := ingestLong(t)
	for _, want := range []string{
		"a refusal after the run is proven writes its refusal record and nothing else",
		"the one delete it makes is on its OWN run id",
	} {
		if !strings.Contains(long, want) {
			t.Errorf("the ingest help does not state %q:\n%s", want, long)
		}
	}
}

// TestIngestHelpPinsTheThreeMeasuredSentences pins the three sentences an audit
// measured false, so a later edit that reintroduces one fails here. A pin is not
// a substitute for the two behavioural tests above; it is what catches a rewrite
// that quietly drops the qualification they establish.
func TestIngestHelpPinsTheThreeMeasuredSentences(t *testing.T) {
	long := ingestLong(t)
	for _, want := range []string{
		// (1) what a refusal writes and deletes.
		"No OTHER run's durable state is touched until the whole payload validates: a refusal " +
			"after the run is proven writes its refusal record and nothing else, and the one delete " +
			"it makes is on its OWN run id — the records of an earlier attempt at it that never committed.",
		// (2) the orphan sweep, including the committed-tier delete and the run
		// that did commit, whose records the sweep leaves alone.
		"Only the next one whose payload validates sweeps it: where the run reached no commit " +
			"marker the sweep ROLLS THAT RUN'S READING RECORDS OUT OF THE COMMITTED LEDGER, because " +
			"the run never happened; where the marker is there the run stands and only the stage goes.",
		// (3) the reserved-name rule, per regime, with the generative row absent.
		"an item carrying a reserved name as one of its own fields is refused with the licence " +
			"stated. The reserved-name table is read at the run's own regime, one row per regime, " +
			"and the generative regime has no row: no name is reserved at the generative position.",
	} {
		if !strings.Contains(long, want) {
			t.Errorf("the ingest help no longer states:\n%s\n\ngot:\n%s", want, long)
		}
	}
	// The claim that nothing at all is written before the run's identity is
	// proven is false of the stage root and its lock, which are created first.
	// The claim is about the DURABLE tier, and it has to say so.
	if strings.Contains(long, "before that point nothing is written anywhere") {
		t.Error("the ingest help claims nothing is written before the run is proven; the stage " +
			"root and its lock are created before the payload is read")
	}
}
