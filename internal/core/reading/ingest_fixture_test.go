package reading

// ingest_fixture_test.go builds the world one `abcd reading ingest` needs: four
// definitions under agents/, one parked run with its manifest, and a payload
// builder that starts from a LEGAL output and is mutated per case.
//
// Every refusal test in this package mutates a payload this builder produced and
// asserts the adjacent legal payload is still accepted. That is deliberate: a
// refusal test that would pass against a verb refusing everything proves nothing
// about the rule it names.

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/intentdriven/abcd/internal/core/issueschema"
)

// ingestFixture is one prepared repository and the run parked in it.
type ingestFixture struct {
	t        *testing.T
	root     string
	runID    string
	position Position
	regime   string
	// manifestHash is the parked manifest's own content hash — the reference an
	// output cites back — and manifestHashes holds one per parked run, for a
	// test that parks a second.
	manifestHash   string
	manifestHashes map[string]string
	// definitionSHA is the hash of the position's definition file, which is half
	// of the instrument's identity.
	definitionSHA string
	// runCounter numbers the extra runs nextRun parks.
	runCounter int
}

// newIngestFixture prepares a repository at one position.
func newIngestFixture(t *testing.T, pos Position) *ingestFixture {
	t.Helper()
	root := t.TempDir()
	f := &ingestFixture{t: t, root: root, position: pos,
		regime: issueschema.ReadingRegime(string(pos)), manifestHashes: map[string]string{}}

	// All four definitions, so a test can mutate one without the others going
	// missing. The locator reads the frontmatter and nothing else.
	for _, p := range Positions() {
		f.writeDefinition(p, issueschema.ReadingRegime(string(p)))
	}
	def, err := LoadDefinition(root, pos)
	if err != nil {
		t.Fatalf("load the %s definition from the fixture: %v", pos, err)
	}
	f.definitionSHA = def.SHA256

	f.runID = "rdg-2608310000000001"
	f.parkRun(f.runID, pos, AssemblerVersion())
	return f
}

// writeDefinition writes one definition file, with a caller-chosen regime so a
// test can make the file disagree with the payload.
func (f *ingestFixture) writeDefinition(p Position, regime string) {
	f.t.Helper()
	body := "---\nposition: " + string(p) + "\nregime: " + regime + "\n---\n\n# " +
		definitionPrefix + string(p) + "\n"
	f.write(DefinitionPath(p), []byte(body))
}

// parkRun writes one assembled run's manifest into the local-tier run directory,
// exactly where an assembly leaves it, and records its content hash.
func (f *ingestFixture) parkRun(runID string, pos Position, assemblerVersion string) {
	f.t.Helper()
	m := Manifest{
		Type: ManifestType, SchemaVersion: SchemaVersion, RunID: runID, Position: pos,
		TargetCommit:     "0123456789abcdef0123456789abcdef01234567",
		AssemblerVersion: assemblerVersion,
		Items:            []ManifestItem{{ItemKey: "i1", Path: "README.md", Kind: KindDoc, Scan: ScanParsed, SHA256: sha256Hex([]byte("x"))}},
		Exclusions:       []Exclusion{},
	}
	// A COMPARATIVE manifest carries what the comparative ingest checks against:
	// the derived widening run, its candidates, and the slate parsed off the
	// criteria discipline. A parked manifest without them would make every legal
	// comparative payload refuse, so the fixture's manifest is what an assembly
	// at that position actually writes (spc-2609020626039834).
	if pos == PositionComparative {
		exercised := true
		m.CandidateRun = fixtureIngestCandidateRun
		m.Candidates = len(fixtureIngestCandidates)
		m.Exercised = &exercised
		m.CandidateFields = append([]string{}, CandidateFields...)
		m.Criteria = append([]string{}, fixtureIngestCriteria...)
		m.Items = nil
		for _, id := range fixtureIngestCandidates {
			for _, field := range CandidateFields {
				m.Items = append(m.Items, ManifestItem{
					ItemKey:   "i-" + id + "-" + field,
					Path:      CandidateSource + "/" + fixtureIngestCandidateRun + "/" + id + ".md",
					Field:     field,
					Candidate: id,
					Kind:      KindCandidate,
					Scan:      ScanParsed,
					SHA256:    sha256Hex([]byte(id + field)),
				})
			}
		}
	}
	raw, err := EncodeManifest(m)
	if err != nil {
		f.t.Fatalf("encode the parked manifest: %v", err)
	}
	f.write(DefaultRunDir+"/"+runID+"/"+ManifestFileName, raw)
	f.manifestHashes[runID] = sha256Hex(raw)
	if runID == f.runID {
		f.manifestHash = sha256Hex(raw)
	}
}

// parkComparative re-parks one run's manifest with a caller-chosen candidate
// count and exercised flag, for the staged EMPTY comparative run: a manifest
// naming the derived run, its own item count, and `exercised: false`.
//
// It overwrites the manifest parkRun already wrote, so the caller re-reads the
// hash: the reference an output cites is the manifest's own content hash, and a
// re-park is a different manifest.
func (f *ingestFixture) parkComparative(runID, candidateRun string, candidates int,
	exercised *bool, items []ManifestItem) {
	f.t.Helper()
	m := Manifest{
		Type: ManifestType, SchemaVersion: SchemaVersion, RunID: runID,
		Position:         PositionComparative,
		TargetCommit:     "0123456789abcdef0123456789abcdef01234567",
		AssemblerVersion: AssemblerVersion(),
		CandidateRun:     candidateRun,
		Candidates:       candidates,
		Exercised:        exercised,
		CandidateFields:  append([]string{}, CandidateFields...),
		Criteria:         append([]string{}, fixtureIngestCriteria...),
		Items:            items,
		Exclusions:       []Exclusion{},
	}
	if m.Items == nil {
		m.Items = []ManifestItem{}
	}
	raw, err := EncodeManifest(m)
	if err != nil {
		f.t.Fatalf("encode the parked manifest: %v", err)
	}
	f.write(DefaultRunDir+"/"+runID+"/"+ManifestFileName, raw)
	f.manifestHashes[runID] = sha256Hex(raw)
	if runID == f.runID {
		f.manifestHash = sha256Hex(raw)
	}
}

// nextRun parks a FRESH run and points doc at it.
//
// A run id that already has an outcome is refused, by design: a rerun is a new
// run with a new run id, never an amendment. So a case that ingests twice
// ingests two runs, and saying that here keeps the rule out of every case.
func (f *ingestFixture) nextRun(doc map[string]any) map[string]any {
	f.t.Helper()
	f.runCounter++
	id := fmt.Sprintf("rdg-2608319%09d", f.runCounter)
	f.parkRun(id, f.position, AssemblerVersion())
	doc["run_id"] = id
	doc["manifest_sha256"] = f.manifestHashOf(id)
	return doc
}

// manifestHashOf is the parked content hash of one run's manifest.
func (f *ingestFixture) manifestHashOf(runID string) string {
	f.t.Helper()
	h, ok := f.manifestHashes[runID]
	if !ok {
		f.t.Fatalf("no run %s is parked in this fixture", runID)
	}
	return h
}

// readRunRecord reads back the run metadata a completed ingest wrote.
func (f *ingestFixture) readRunRecord(runID string) RunRecord {
	f.t.Helper()
	raw, err := os.ReadFile(filepath.Join(f.root, filepath.FromSlash(
		ReadingsRecordDir+"/"+runID+"/"+RunFileName)))
	if err != nil {
		f.t.Fatalf("read the run record of %s: %v", runID, err)
	}
	var run RunRecord
	if err := json.Unmarshal(raw, &run); err != nil {
		f.t.Fatalf("decode the run record of %s: %v", runID, err)
	}
	return run
}

// readRefusalRecord reads back the record a list-level refusal wrote.
func (f *ingestFixture) readRefusalRecord(runID string) RefusalRecord {
	f.t.Helper()
	raw, err := os.ReadFile(filepath.Join(f.root, filepath.FromSlash(
		ReadingsRecordDir+"/"+runID+"/"+RefusalFileName)))
	if err != nil {
		f.t.Fatalf("read the refusal record of %s: %v", runID, err)
	}
	var rec RefusalRecord
	if err := json.Unmarshal(raw, &rec); err != nil {
		f.t.Fatalf("decode the refusal record of %s: %v", runID, err)
	}
	return rec
}

// write puts one file at a repo-relative slash path.
func (f *ingestFixture) write(rel string, data []byte) {
	f.t.Helper()
	abs := filepath.Join(f.root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
		f.t.Fatal(err)
	}
	if err := os.WriteFile(abs, data, 0o644); err != nil {
		f.t.Fatal(err)
	}
}

// body is a legal item body at the fixture's position: every declared field,
// each with unremarkable text no signature matches.
func (f *ingestFixture) body() map[string]any {
	out := map[string]any{}
	for i, field := range issueschema.ReadingBodyFields[string(f.position)] {
		out[field] = fieldText(field, i)
	}
	return out
}

// The comparative fixture's derived run, its candidates and its declared
// criteria. They are the values the parked manifest records, so a legal payload
// at that position names a candidate of the run and a criterion the discipline
// declares — which is exactly what the two comparative ingest checks compare
// against (spc-2609020626039834).
const fixtureIngestCandidateRun = "rdg-2608301200000009"

var (
	fixtureIngestCandidates = []string{"rdi-1", "rdi-2", "rdi-3"}
	fixtureIngestCriteria   = []string{"Plausibility", "Generativity", "Cost"}
)

// fieldText is plain prose for one body field. `claim_type` takes one of its
// three tokens, `candidate_id` and `criterion` take values the parked
// comparative manifest records, and everything else takes a sentence that trips
// no detector.
func fieldText(field string, i int) string {
	switch field {
	case "claim_type":
		return "criterion"
	case "candidate_id":
		return fixtureIngestCandidates[0]
	case "criterion":
		return fixtureIngestCriteria[0]
	}
	return "the passed material carries this, stated at ordinal " + string(rune('a'+i))
}

// item is one legal flat item: the envelope's pattern field plus the position's
// body, exactly as every definition's item shape instructs.
func (f *ingestFixture) item() map[string]any {
	out := map[string]any{PatternField: "the pattern this reading read under"}
	for k, v := range f.body() {
		out[k] = v
	}
	return out
}

// payload is a legal output carrying n legal items.
func (f *ingestFixture) payload(n int) map[string]any {
	items := make([]any, 0, n)
	for i := 0; i < n; i++ {
		items = append(items, f.item())
	}
	return map[string]any{
		"_type":           OutputType,
		"run_id":          f.runID,
		"position":        string(f.position),
		"regime":          f.regime,
		"manifest_sha256": f.manifestHash,
		"instrument": map[string]any{
			"model":             "a-model",
			"definition_sha256": f.definitionSHA,
			"assembler_version": AssemblerVersion(),
		},
		"items": items,
	}
}

// writePayload renders a payload and returns the absolute path the verb reads.
func (f *ingestFixture) writePayload(doc any) string {
	f.t.Helper()
	raw, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		f.t.Fatal(err)
	}
	path := filepath.Join(f.t.TempDir(), "output.json")
	if err := os.WriteFile(path, raw, 0o644); err != nil {
		f.t.Fatal(err)
	}
	return path
}

// ingest runs the verb over a payload document.
func (f *ingestFixture) ingest(doc any) (IngestResult, error) {
	f.t.Helper()
	return Ingest(IngestRequest{RepoRoot: f.root, OutputPath: f.writePayload(doc)})
}

// mustIngest runs the verb and fails the test if it refused. Every refusal case
// calls this on the adjacent LEGAL payload, so a verb that refused everything
// could not pass the case that names the rule.
func (f *ingestFixture) mustIngest(doc any) IngestResult {
	f.t.Helper()
	res, err := f.ingest(doc)
	if err != nil {
		f.t.Fatalf("the adjacent legal payload was refused: %v", err)
	}
	return res
}

// refusalOf returns the refusal of the item at one ordinal, and fails the test
// when the run refused nothing there. It reads the RESULT rather than the error,
// because an item-level violation refuses the item and lands the rest — the run
// then succeeds, and a test asserting on the error alone would be asserting the
// wrong granularity.
func (f *ingestFixture) refusalOf(res IngestResult, ordinal int) ItemRefusal {
	f.t.Helper()
	for _, r := range res.RefusedItems {
		if r.Ordinal == ordinal {
			return r
		}
	}
	f.t.Fatalf("item %d was not refused; the run refused %v", ordinal, res.RefusedItems)
	return ItemRefusal{}
}

// refusedItem runs a payload whose item at `ordinal` was made illegal and
// returns that item's refusal, having first established that the run itself
// landed and that every OTHER item survived. That second half is the point: a
// refusal case which passed against a verb that refused everything would prove
// nothing about the rule it names.
func (f *ingestFixture) refusedItem(doc any, ordinal, total int) ItemRefusal {
	f.t.Helper()
	res, err := f.ingest(doc)
	if err != nil {
		f.t.Fatalf("an item-level violation refused the whole run: %v", err)
	}
	if len(res.Records) != total-1 {
		f.t.Fatalf("landed %d record(s); one item was illegal and the other %d were not",
			len(res.Records), total-1)
	}
	if len(res.RefusedItems) != 1 {
		f.t.Fatalf("refused %d item(s), want exactly the illegal one: %v", len(res.RefusedItems), res.RefusedItems)
	}
	return f.refusalOf(res, ordinal)
}

// ledgerRecords lists the reading records durable for one run.
func (f *ingestFixture) ledgerRecords(runID string) []string {
	f.t.Helper()
	dir := filepath.Join(f.root, ".abcd", "work", "issues", issueschema.ReadingsDir, runID)
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		f.t.Fatal(err)
	}
	var out []string
	for _, e := range entries {
		out = append(out, e.Name())
	}
	return out
}

// assertEveryRecordIsReadable holds every record the run landed to the limit
// every reader of the family applies. A record past it is durable, committed and
// unreadable — which makes the item it carries permanently unanswerable.
func (f *ingestFixture) assertEveryRecordIsReadable() {
	f.t.Helper()
	for _, name := range f.ledgerRecords(f.runID) {
		info, err := os.Stat(filepath.Join(f.root, ".abcd", "work", "issues",
			issueschema.ReadingsDir, f.runID, name))
		if err != nil {
			f.t.Fatal(err)
		}
		if info.Size() > issueschema.RecordReadLimit {
			f.t.Errorf("%s is %d bytes, past the %d-byte limit every reader of the family applies",
				name, info.Size(), issueschema.RecordReadLimit)
		}
	}
}

// exists reports whether a repo-relative path is present.
func (f *ingestFixture) exists(rel string) bool {
	f.t.Helper()
	_, err := os.Stat(filepath.Join(f.root, filepath.FromSlash(rel)))
	return err == nil
}

// nothingDurableInTheLedger is nothingDurable minus the readings tree, for a
// run refused at LIST level: such a run leaves a refusal record on purpose, and
// what must be empty is the reading-record family and the stage.
func (f *ingestFixture) nothingDurableInTheLedger(runID string) {
	f.t.Helper()
	if got := f.ledgerRecords(runID); len(got) != 0 {
		f.t.Errorf("run %s left %v in the reading-record family", runID, got)
	}
	if f.exists(IngestStageDir + "/" + runID) {
		f.t.Errorf("run %s left a stage behind", runID)
	}
	if f.exists(ReadingsRecordDir + "/" + runID + "/" + RunFileName) {
		f.t.Errorf("run %s left a commit marker behind", runID)
	}
}

// nothingDurable asserts the three places a run could leave a trace are all
// empty for it: the reading-record family, the readings tree, and the stage.
func (f *ingestFixture) nothingDurable(runID string) {
	f.t.Helper()
	if got := f.ledgerRecords(runID); len(got) != 0 {
		f.t.Errorf("run %s left %v in the reading-record family", runID, got)
	}
	for _, rel := range []string{
		ReadingsRecordDir + "/" + runID,
		IngestStageDir + "/" + runID,
	} {
		if f.exists(rel) {
			f.t.Errorf("run %s left %s behind", runID, rel)
		}
	}
}
