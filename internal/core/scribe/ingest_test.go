package scribe

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/capture"
	"github.com/intentdriven/abcd/internal/core/issueschema"
)

// A ground that clears the capture verbs' substance floor.
const groundA = "the constraint the reading names is real and binds the verb as shipped"

// session is one assembled scribe session over a fixture: the fixture, the
// supplied text and the assembly's result.
type session struct {
	fixture
	supplied string
	dispPath string
	res      AssembleResult
}

// assembleSession builds a fixture of n items at position and assembles over
// the supplied text, which may name the items as {0}, {1}, ... placeholders.
func assembleSession(t *testing.T, position string, n int, supplied string) session {
	t.Helper()
	f := newFixture(t, position, n)
	for i, id := range f.items {
		supplied = strings.ReplaceAll(supplied, "{"+string(rune('0'+i))+"}", id)
	}
	dispPath := supply(t, supplied)
	res, err := Assemble(AssembleRequest{RepoRoot: f.repo, Run: fixtureRun, DispositionsPath: dispPath})
	if err != nil {
		t.Fatalf("Assemble: %v", err)
	}
	return session{fixture: f, supplied: supplied, dispPath: dispPath, res: res}
}

// out is a payload skeleton for the session: the envelope filled, every list
// empty.
func (s session) out() Output {
	return Output{Type: OutputType, Run: fixtureRun, ContextSHA256: s.res.ContextSHA256}
}

// write puts a payload on disk and returns its path.
func (s session) write(t *testing.T, o any) string {
	t.Helper()
	raw, err := json.Marshal(o)
	if err != nil {
		t.Fatal(err)
	}
	return s.writeRaw(t, string(raw))
}

func (s session) writeRaw(t *testing.T, raw string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "scribe-output.json")
	if err := os.WriteFile(p, []byte(raw), 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

func (s session) ingest(t *testing.T, payloadPath string) (IngestResult, error) {
	t.Helper()
	return Ingest(IngestRequest{RepoRoot: s.repo, ScribeJSONPath: payloadPath, DispositionsPath: s.dispPath})
}

func (s session) ledger(t *testing.T) string {
	t.Helper()
	return treeDigest(t, filepath.Join(s.repo, filepath.FromSlash(capture.LedgerRelPath)))
}

func (s session) promoted() string {
	return filepath.Join(s.repo, filepath.FromSlash(issueschema.ReadingsRecordDir), fixtureRun, ManifestFileName)
}

// dispositionFiles lists every disposition record in the ledger.
func (s session) dispositionFiles(t *testing.T) []string {
	t.Helper()
	var out []string
	root := filepath.Join(s.repo, filepath.FromSlash(capture.LedgerRelPath), issueschema.DispositionsDir)
	_ = filepath.Walk(root, func(p string, fi os.FileInfo, err error) error {
		if err == nil && !fi.IsDir() {
			out = append(out, p)
		}
		return nil
	})
	return out
}

// TestScribeIngestWritesASuppliedDisposition is ac-2: a disposition the
// researcher supplied lands through the disposition writer, and the ledger
// differs afterwards by that one record.
func TestScribeIngestWritesASuppliedDisposition(t *testing.T) {
	s := assembleSession(t, positionDetection, 2,
		"{0}: accepted — "+groundA+".\n{1}: I have not decided yet.\n")
	before := s.dispositionFiles(t)
	o := s.out()
	o.Dispositions = []OutDisposition{{Item: s.items[0], State: issueschema.DispositionAccepted, Grounds: groundA}}
	o.Outstanding = []string{s.items[1]}

	res, err := s.ingest(t, s.write(t, o))
	if err != nil {
		t.Fatalf("Ingest: %v", err)
	}
	after := s.dispositionFiles(t)
	if len(before) != 0 || len(after) != 1 || len(res.Dispositions) != 1 {
		t.Fatalf("dispositions before %v, after %v, result %+v; want exactly one written", before, after, res.Dispositions)
	}
	if d := res.Dispositions[0]; d.Item != s.items[0] || d.State != issueschema.DispositionAccepted {
		t.Fatalf("wrote %+v", d)
	}
	raw, _ := os.ReadFile(after[0])
	if !strings.Contains(string(raw), groundA) {
		t.Fatalf("the disposition does not carry the supplied ground:\n%s", raw)
	}
	// Nothing else in the ledger moved: no admission, no surprise, no record for
	// the outstanding item.
	for _, dir := range []string{issueschema.AdmissionsDir, issueschema.SurprisesDir} {
		if _, err := os.Stat(filepath.Join(s.repo, filepath.FromSlash(capture.LedgerRelPath), dir)); err == nil {
			entries, _ := os.ReadDir(filepath.Join(s.repo, filepath.FromSlash(capture.LedgerRelPath), dir))
			if len(entries) != 0 {
				t.Errorf("%s gained %d entries", dir, len(entries))
			}
		}
	}
	if len(res.Outstanding) != 1 || res.Outstanding[0] != s.items[1] {
		t.Errorf("outstanding = %q, want [%s]", res.Outstanding, s.items[1])
	}
}

// TestScribeIngestRefusesAnAuthoredField is ac-3's first half: a key outside the
// closed shapes is refused by name, with the item it was on, and nothing lands.
func TestScribeIngestRefusesAnAuthoredField(t *testing.T) {
	s := assembleSession(t, positionDetection, 1, "{0}: accepted — "+groundA+".\n")
	before := s.ledger(t)
	for _, tc := range []struct {
		name, raw, field string
	}{
		{"a resolution on a disposition", `{"_type":"` + OutputType + `","run":"` + fixtureRun +
			`","context_sha256":"` + s.res.ContextSHA256 + `","dispositions":[{"item":"` + s.items[0] +
			`","state":"accepted","grounds":"` + groundA + `","resolution":"fixed it"}]}`, "resolution"},
		{"a pattern at the top", `{"_type":"` + OutputType + `","run":"` + fixtureRun +
			`","context_sha256":"` + s.res.ContextSHA256 + `","pattern":"mine","outstanding":["` + s.items[0] + `"]}`, "pattern"},
		{"a position on a fidelity flag", `{"_type":"` + OutputType + `","run":"` + fixtureRun +
			`","context_sha256":"` + s.res.ContextSHA256 + `","outstanding":["` + s.items[0] +
			`"],"fidelity_flags":[{"first":"a","second":"b","position":"widening"}]}`, "position"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := s.ingest(t, s.writeRaw(t, tc.raw))
			if err == nil || !strings.Contains(err.Error(), `"`+tc.field+`"`) {
				t.Fatalf("an authored %q was not refused by name: %v", tc.field, err)
			}
			if tc.field == "resolution" && !strings.Contains(err.Error(), s.items[0]) {
				t.Errorf("the refusal does not name the item it was on: %v", err)
			}
			if s.ledger(t) != before {
				t.Fatal("a refused payload changed the ledger")
			}
		})
	}
}

// TestScribeIngestRefusesAnUnsuppliedGround is ac-3's second half, on the
// disposition: a ground the researcher did not write is refused, naming the
// field and the item.
func TestScribeIngestRefusesAnUnsuppliedGround(t *testing.T) {
	s := assembleSession(t, positionDetection, 1, "{0}: accepted — "+groundA+".\n")
	before := s.ledger(t)
	o := s.out()
	o.Dispositions = []OutDisposition{{Item: s.items[0], State: issueschema.DispositionAccepted,
		Grounds: groundA + " and also because the scribe thinks so"}}
	_, err := s.ingest(t, s.write(t, o))
	if err == nil || !strings.Contains(err.Error(), "grounds") || !strings.Contains(err.Error(), s.items[0]) {
		t.Fatalf("an unsupplied ground was not refused naming grounds and the item: %v", err)
	}
	if s.ledger(t) != before {
		t.Fatal("a refused payload changed the ledger")
	}

	// Reformatting is the scribe's licence: whitespace folds, so a ground
	// re-wrapped across lines is the same words.
	o.Dispositions[0].Grounds = strings.ReplaceAll(groundA, " the ", "\n  the ")
	if _, err := s.ingest(t, s.write(t, o)); err != nil {
		t.Fatalf("a re-wrapped supplied ground was refused: %v", err)
	}
}

// TestScribeIngestRefusesAnUnsuppliedAdmissionGround: the same check on an
// admission, which is the other record carrying a ground.
func TestScribeIngestRefusesAnUnsuppliedAdmissionGround(t *testing.T) {
	s := assembleSession(t, issueschema.PositionWidening, 1, "{0}: admit it — "+groundA+".\n")
	before := s.ledger(t)
	o := s.out()
	o.Admissions = []OutAdmission{{Item: s.items[0], Grounds: "a ground nobody wrote down for this proposal"}}
	_, err := s.ingest(t, s.write(t, o))
	if err == nil || !strings.Contains(err.Error(), "grounds") || !strings.Contains(err.Error(), s.items[0]) {
		t.Fatalf("an unsupplied admission ground was not refused: %v", err)
	}
	if s.ledger(t) != before {
		t.Fatal("a refused payload changed the ledger")
	}
}

// TestScribeIngestRefusesAnUnsuppliedDisposition: a disposition for an item the
// supplied text never names is one the researcher did not supply.
func TestScribeIngestRefusesAnUnsuppliedDisposition(t *testing.T) {
	s := assembleSession(t, positionDetection, 2, "{0}: accepted — "+groundA+".\n")
	before := s.ledger(t)
	o := s.out()
	o.Dispositions = []OutDisposition{
		{Item: s.items[0], State: issueschema.DispositionAccepted, Grounds: groundA},
		{Item: s.items[1], State: issueschema.DispositionAccepted, Grounds: groundA},
	}
	_, err := s.ingest(t, s.write(t, o))
	if err == nil || !strings.Contains(err.Error(), s.items[1]) {
		t.Fatalf("a disposition the researcher did not supply was not refused: %v", err)
	}
	if s.ledger(t) != before {
		t.Fatal("a refused payload changed the ledger")
	}
}

// TestScribeIngestReportsOutstandingAndWritesNothing is ac-4.
func TestScribeIngestReportsOutstandingAndWritesNothing(t *testing.T) {
	s := assembleSession(t, positionDetection, 2, "Nothing decided yet.\n")
	before := s.ledger(t)
	o := s.out()
	o.Outstanding = []string{s.items[0], s.items[1]}
	res, err := s.ingest(t, s.write(t, o))
	if err != nil {
		t.Fatalf("Ingest: %v", err)
	}
	if len(res.Outstanding) != 2 {
		t.Fatalf("outstanding = %q, want both items", res.Outstanding)
	}
	if s.ledger(t) != before {
		t.Fatal("an all-outstanding payload wrote a ledger record")
	}

	// An outstanding item must be the run's, and must carry no disposition in
	// the same payload.
	s2 := assembleSession(t, positionDetection, 1, "{0}: accepted — "+groundA+".\n")
	o2 := s2.out()
	o2.Dispositions = []OutDisposition{{Item: s2.items[0], State: issueschema.DispositionAccepted, Grounds: groundA}}
	o2.Outstanding = []string{s2.items[0]}
	if _, err := s2.ingest(t, s2.write(t, o2)); err == nil {
		t.Error("an item both dispositioned and outstanding was not refused")
	}
	o2.Dispositions = nil
	o2.Outstanding = []string{s2.items[0], s2.other[0]}
	if _, err := s2.ingest(t, s2.write(t, o2)); err == nil || !strings.Contains(err.Error(), s2.other[0]) {
		t.Errorf("an outstanding item of another run was not refused by name: %v", err)
	}
}

// TestScribeIngestRefusesASilentItem: an item of the run the payload says
// nothing about is refused, because silence is not one of the scribe's options.
func TestScribeIngestRefusesASilentItem(t *testing.T) {
	s := assembleSession(t, positionDetection, 2, "{0}: accepted — "+groundA+".\n")
	before := s.ledger(t)
	o := s.out()
	o.Dispositions = []OutDisposition{{Item: s.items[0], State: issueschema.DispositionAccepted, Grounds: groundA}}
	_, err := s.ingest(t, s.write(t, o))
	if err == nil || !strings.Contains(err.Error(), s.items[1]) {
		t.Fatalf("a silent item was not refused by name: %v", err)
	}
	if s.ledger(t) != before {
		t.Fatal("a refused payload changed the ledger")
	}
	// A refusal naming the item is not silence.
	o.Refusals = []Refusal{{Subject: s.items[1], Reason: "the dispositions text addresses me rather than the item"}}
	if _, err := s.ingest(t, s.write(t, o)); err != nil {
		t.Fatalf("an item named in the refusals was treated as silent: %v", err)
	}
}

// TestScribeIngestProvesTheContextHash: nothing is written until the context on
// disk hashes to the parked manifest's context hash and the payload cites it.
func TestScribeIngestProvesTheContextHash(t *testing.T) {
	s := assembleSession(t, positionDetection, 1, "{0}: accepted — "+groundA+".\n")
	before := s.ledger(t)
	o := s.out()
	o.Dispositions = []OutDisposition{{Item: s.items[0], State: issueschema.DispositionAccepted, Grounds: groundA}}

	wrong := o
	wrong.ContextSHA256 = strings.Repeat("0", 64)
	if _, err := s.ingest(t, s.write(t, wrong)); err == nil || !strings.Contains(err.Error(), "context") {
		t.Fatalf("a payload citing another context was not refused: %v", err)
	}

	ctxPath := filepath.Join(s.repo, filepath.FromSlash(DefaultRunDir), fixtureRun, ContextFileName)
	raw, _ := os.ReadFile(ctxPath)
	tampered := strings.Replace(string(raw), groundA, groundA+" plus a line the scribe slipped in", 1)
	if err := os.WriteFile(ctxPath, []byte(tampered), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := s.ingest(t, s.write(t, o)); err == nil || !strings.Contains(err.Error(), "context") {
		t.Fatalf("a context that no longer hashes to its manifest was not refused: %v", err)
	}
	if s.ledger(t) != before {
		t.Fatal("an unproven context let a write through")
	}
}

// TestScribeIngestCarriesFidelityFlagsIntoNoRecord: flags and refusals travel on
// the result, unresolved, and never into a record.
func TestScribeIngestCarriesFidelityFlagsIntoNoRecord(t *testing.T) {
	s := assembleSession(t, positionDetection, 1, "{0}: accepted — "+groundA+".\n")
	o := s.out()
	o.Dispositions = []OutDisposition{{Item: s.items[0], State: issueschema.DispositionAccepted, Grounds: groundA}}
	o.FidelityFlags = []FidelityFlag{{First: "FLAG-FIRST-PIECE", Second: "FLAG-SECOND-PIECE"}}
	o.Refusals = []Refusal{{Subject: "REFUSAL-SUBJECT", Reason: "REFUSAL-REASON"}}
	res, err := s.ingest(t, s.write(t, o))
	if err != nil {
		t.Fatalf("Ingest: %v", err)
	}
	if len(res.FidelityFlags) != 1 || res.FidelityFlags[0].First != "FLAG-FIRST-PIECE" || len(res.Refusals) != 1 {
		t.Fatalf("the result carries flags %+v and refusals %+v", res.FidelityFlags, res.Refusals)
	}
	var leaked []string
	_ = filepath.Walk(filepath.Join(s.repo, ".abcd"), func(p string, fi os.FileInfo, err error) error {
		if err != nil || fi.IsDir() || strings.Contains(p, ".work.local") {
			return nil
		}
		raw, _ := os.ReadFile(p)
		for _, tok := range []string{"FLAG-FIRST-PIECE", "FLAG-SECOND-PIECE", "REFUSAL-SUBJECT", "REFUSAL-REASON"} {
			if strings.Contains(string(raw), tok) {
				leaked = append(leaked, tok+" in "+p)
			}
		}
		return nil
	})
	if len(leaked) > 0 {
		t.Fatalf("flags or refusals reached a record: %v", leaked)
	}
}

// TestScribeIngestNamesWhatLandedBeforeARefusal: a write the writer refuses
// stops the ingest, and the result names what landed before it, so a rerun sees
// the partial state rather than minting it twice.
func TestScribeIngestNamesWhatLandedBeforeARefusal(t *testing.T) {
	s := assembleSession(t, positionDetection, 2,
		"{0}: accepted — "+groundA+".\n{1}: accepted — "+groundA+".\n")
	// The second item gains a standing disposition AFTER the context was built,
	// so the writer refuses a second answer that does not cite it.
	if _, err := capture.Disposition(capture.DispositionRequest{RepoRoot: s.repo, Item: s.items[1],
		State: issueschema.DispositionAccepted, Grounds: groundA}); err != nil {
		t.Fatal(err)
	}
	o := s.out()
	o.Dispositions = []OutDisposition{
		{Item: s.items[0], State: issueschema.DispositionAccepted, Grounds: groundA},
		{Item: s.items[1], State: issueschema.DispositionAccepted, Grounds: groundA},
	}
	res, err := s.ingest(t, s.write(t, o))
	if err == nil {
		t.Fatal("a write the writer refuses did not stop the ingest")
	}
	if len(res.Dispositions) != 1 || res.Dispositions[0].Item != s.items[0] {
		t.Fatalf("the result names %+v as landed, want the first item's disposition", res.Dispositions)
	}
	if !strings.Contains(err.Error(), res.Dispositions[0].ID) {
		t.Errorf("the refusal does not name what landed before it: %v", err)
	}
	if _, statErr := os.Stat(s.promoted()); !os.IsNotExist(statErr) {
		t.Error("a refused ingest promoted the manifest")
	}
}

// TestScribeIngestRefusesBeforeTheComparativeRun: the ordering gate the shared
// writer holds refuses a widening payload at its first disposition, before
// anything lands, and names the run it is waiting for.
func TestScribeIngestRefusesBeforeTheComparativeRun(t *testing.T) {
	s := assembleSession(t, issueschema.PositionWidening, 1, "{0}: accepted — "+groundA+".\n")
	before := s.ledger(t)
	o := s.out()
	o.Dispositions = []OutDisposition{{Item: s.items[0], State: issueschema.DispositionAccepted, Grounds: groundA}}
	res, err := s.ingest(t, s.write(t, o))
	if !errors.Is(err, capture.ErrNotCharacterised) || !strings.Contains(err.Error(), fixtureRun) {
		t.Fatalf("a widening payload with no comparative run: %v", err)
	}
	if len(res.Dispositions) != 0 || s.ledger(t) != before {
		t.Fatal("something landed before the ordering gate refused")
	}
}

// TestScribeIngestPromotesTheManifestLast: a refused ingest leaves the manifest
// parked and nothing beside the run; a completed one lands it write-once.
func TestScribeIngestPromotesTheManifestLast(t *testing.T) {
	s := assembleSession(t, positionDetection, 1, "{0}: accepted — "+groundA+".\n")
	bad := s.out()
	bad.Dispositions = []OutDisposition{{Item: s.items[0], State: issueschema.DispositionAccepted, Grounds: "unsupplied words here"}}
	if _, err := s.ingest(t, s.write(t, bad)); err == nil {
		t.Fatal("expected a refusal")
	}
	if _, err := os.Stat(s.promoted()); !os.IsNotExist(err) {
		t.Fatal("a refused ingest promoted the manifest")
	}
	parked := filepath.Join(s.repo, filepath.FromSlash(DefaultRunDir), fixtureRun, ManifestFileName)
	if _, err := os.Stat(parked); err != nil {
		t.Fatalf("a refused ingest moved the parked manifest: %v", err)
	}

	good := s.out()
	good.Dispositions = []OutDisposition{{Item: s.items[0], State: issueschema.DispositionAccepted, Grounds: groundA}}
	res, err := s.ingest(t, s.write(t, good))
	if err != nil {
		t.Fatalf("Ingest: %v", err)
	}
	promoted, err := os.ReadFile(s.promoted())
	if err != nil {
		t.Fatalf("the manifest was not promoted beside the run: %v", err)
	}
	parkedRaw, _ := os.ReadFile(parked)
	if string(promoted) != string(parkedRaw) {
		t.Error("the promoted manifest is not the parked one")
	}
	if res.Manifest == "" {
		t.Error("the result does not name the promoted manifest")
	}
	// Write-once: a second session over the run is refused BEFORE it writes. The
	// payload carries a surprise the supplied text holds, so a refusal that came
	// only at promotion would leave the surprise in the ledger.
	before := s.ledger(t)
	again := s.out()
	again.Surprises = []OutSurprise{{OccasionedBy: s.items[0], Text: groundA}}
	if _, err := s.ingest(t, s.write(t, again)); err == nil || !strings.Contains(err.Error(), ManifestFileName) {
		t.Fatalf("a second ingest over a promoted run was not refused: %v", err)
	}
	if s.ledger(t) != before {
		t.Fatal("the refused second ingest changed the ledger")
	}
}

// TestScribeIngestWritesAdmissionsAndSurprises: the other two record families
// the scribe transcribes land through their own verbs, on supplied words.
func TestScribeIngestWritesAdmissionsAndSurprises(t *testing.T) {
	surprise := "I did not expect the proposal to touch the release gate at all"
	s := assembleSession(t, issueschema.PositionWidening, 1,
		"{0}: admit it — "+groundA+".\nSurprise at {0}: "+surprise+".\n")
	// Characterise the widening run, so the ordering gate lets the admission
	// through.
	writeFile(t, s.repo, filepath.Join(issueschema.ReadingsRecordDir, "rdg-2609250000000009", issueschema.RunRecordFileName),
		`{"run_id":"rdg-2609250000000009","position":"comparative","candidate_run":"`+fixtureRun+`"}`)
	o := s.out()
	o.Admissions = []OutAdmission{{Item: s.items[0], Grounds: groundA}}
	o.Surprises = []OutSurprise{{OccasionedBy: s.items[0], Text: surprise}}
	res, err := s.ingest(t, s.write(t, o))
	if err != nil {
		t.Fatalf("Ingest: %v", err)
	}
	if len(res.Admissions) != 1 || !res.Admissions[0].DispositionWritten || len(res.Surprises) != 1 {
		t.Fatalf("admissions %+v, surprises %+v", res.Admissions, res.Surprises)
	}
	// A surprise text the researcher did not write is refused.
	s2 := assembleSession(t, positionDetection, 1, "{0}: nothing yet.\n")
	o2 := s2.out()
	o2.Outstanding = []string{s2.items[0]}
	o2.Surprises = []OutSurprise{{OccasionedBy: s2.items[0], Text: surprise}}
	if _, err := s2.ingest(t, s2.write(t, o2)); err == nil || !strings.Contains(err.Error(), "text") {
		t.Fatalf("an unsupplied surprise was not refused: %v", err)
	}
}

// TestScribeIngestHoldsTheStateToTheItemsLine: the state is the ruling, and the
// scribe may not author it. A disposition's state must stand, whole-word, on a
// line of the supplied text that names its item: a state another item's line
// carries is not the researcher's answer to this one, and a word that merely
// contains the state is not the state.
func TestScribeIngestHoldsTheStateToTheItemsLine(t *testing.T) {
	s := assembleSession(t, positionDetection, 2,
		"{0}: rejected — "+groundA+".\n{1}: accepted — "+groundA+".\n")
	before := s.ledger(t)
	o := s.out()
	o.Dispositions = []OutDisposition{
		{Item: s.items[0], State: issueschema.DispositionAccepted, Grounds: groundA},
		{Item: s.items[1], State: issueschema.DispositionAccepted, Grounds: groundA},
	}
	_, err := s.ingest(t, s.write(t, o))
	if err == nil || !strings.Contains(err.Error(), "state") || !strings.Contains(err.Error(), s.items[0]) {
		t.Fatalf("a state the item's line does not carry was not refused naming the state and the item: %v", err)
	}
	if s.ledger(t) != before {
		t.Fatal("a refused payload changed the ledger")
	}

	// Whole-word: "unaccepted" does not carry "accepted".
	s2 := assembleSession(t, positionDetection, 1, "{0}: unaccepted — "+groundA+".\n")
	o2 := s2.out()
	o2.Dispositions = []OutDisposition{{Item: s2.items[0], State: issueschema.DispositionAccepted, Grounds: groundA}}
	if _, err := s2.ingest(t, s2.write(t, o2)); err == nil || !strings.Contains(err.Error(), "state") {
		t.Fatalf("a state carried only inside a longer word was not refused: %v", err)
	}

	// The state the line does carry lands, whatever its case.
	s3 := assembleSession(t, positionDetection, 1, "{0}: Accepted — "+groundA+".\n")
	o3 := s3.out()
	o3.Dispositions = []OutDisposition{{Item: s3.items[0], State: issueschema.DispositionAccepted, Grounds: groundA}}
	if _, err := s3.ingest(t, s3.write(t, o3)); err != nil {
		t.Fatalf("a state the item's line carries was refused: %v", err)
	}
}

// TestScribeIngestHoldsAnAdmissionToTheItemsLine: an admission writes an
// accepted disposition, so it is a state too, and the same rule holds it: the
// item's own line must admit or accept the proposal.
func TestScribeIngestHoldsAnAdmissionToTheItemsLine(t *testing.T) {
	s := assembleSession(t, issueschema.PositionWidening, 2,
		"{0}: declined — "+groundA+".\n{1}: admit it — "+groundA+".\n")
	writeFile(t, s.repo, filepath.Join(issueschema.ReadingsRecordDir, "rdg-2609250000000009", issueschema.RunRecordFileName),
		`{"run_id":"rdg-2609250000000009","position":"comparative","candidate_run":"`+fixtureRun+`"}`)
	before := s.ledger(t)
	o := s.out()
	o.Admissions = []OutAdmission{{Item: s.items[0], Grounds: groundA}, {Item: s.items[1], Grounds: groundA}}
	_, err := s.ingest(t, s.write(t, o))
	if err == nil || !strings.Contains(err.Error(), "admission") || !strings.Contains(err.Error(), s.items[0]) {
		t.Fatalf("an admission the item's line does not carry was not refused: %v", err)
	}
	if s.ledger(t) != before {
		t.Fatal("a refused payload changed the ledger")
	}
}

// TestScribeIngestAuthenticatesTheParkedPair: the parked context and manifest
// sit where a scribe with tools can rewrite them, so neither is the witness to
// what the researcher wrote. A scribe that rewrites the context's supplied text
// to carry a ground of its own, and recomputes the manifest's hashes to match,
// is refused, because the ingest re-reads the researcher's own dispositions
// and holds the pair to them.
func TestScribeIngestAuthenticatesTheParkedPair(t *testing.T) {
	authored := "a ground the scribe wrote and the researcher never did"
	for _, tc := range []struct {
		name           string
		rehashSupplied bool
	}{
		{"the manifest's supplied hash left stale", false},
		{"the manifest's supplied hash recomputed", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s := assembleSession(t, positionDetection, 1, "{0}: accepted — "+groundA+".\n")
			before := s.ledger(t)
			dir := filepath.Join(s.repo, filepath.FromSlash(DefaultRunDir), fixtureRun)

			ctx := s.res.Context
			ctx.Supplied.Dispositions = s.items[0] + ": accepted — " + authored + ".\n"
			ctxRaw, err := encode(ctx)
			if err != nil {
				t.Fatal(err)
			}
			m := s.res.Manifest
			m.ContextSHA256 = sha(ctxRaw)
			if tc.rehashSupplied {
				m.Supplied.DispositionsSHA256 = sha([]byte(ctx.Supplied.Dispositions))
			}
			mRaw, err := encode(m)
			if err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(dir, ContextFileName), ctxRaw, 0o644); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(dir, ManifestFileName), mRaw, 0o644); err != nil {
				t.Fatal(err)
			}

			o := s.out()
			o.ContextSHA256 = m.ContextSHA256
			o.Dispositions = []OutDisposition{{Item: s.items[0], State: issueschema.DispositionAccepted, Grounds: authored}}
			_, err = s.ingest(t, s.write(t, o))
			if err == nil || !strings.Contains(err.Error(), "dispositions") {
				t.Fatalf("a rewritten parked pair carrying a scribe-authored ground was not refused: %v", err)
			}
			if s.ledger(t) != before {
				t.Fatal("a rewritten parked pair let a write through")
			}
		})
	}

	// The researcher's text is required: without it there is no witness.
	s := assembleSession(t, positionDetection, 1, "{0}: accepted — "+groundA+".\n")
	o := s.out()
	o.Dispositions = []OutDisposition{{Item: s.items[0], State: issueschema.DispositionAccepted, Grounds: groundA}}
	if _, err := Ingest(IngestRequest{RepoRoot: s.repo, ScribeJSONPath: s.write(t, o)}); err == nil ||
		!strings.Contains(err.Error(), "dispositions") {
		t.Fatalf("an ingest with no supplied dispositions was not refused: %v", err)
	}
}

// TestScribeIngestPromotesOnlyWhenARecordLanded: an ingest that lands nothing
// (every item outstanding, or refused) promotes nothing, so it cannot lock the
// run against the session that answers it later; the ingest that lands a
// record promotes the manifest as before.
func TestScribeIngestPromotesOnlyWhenARecordLanded(t *testing.T) {
	s := assembleSession(t, positionDetection, 1, "{0}: accepted — "+groundA+".\n")
	nothing := s.out()
	nothing.Outstanding = []string{s.items[0]}
	res, err := s.ingest(t, s.write(t, nothing))
	if err != nil {
		t.Fatalf("Ingest: %v", err)
	}
	if res.Manifest != "" {
		t.Errorf("an ingest that landed nothing names a promoted manifest %s", res.Manifest)
	}
	if _, err := os.Stat(s.promoted()); !os.IsNotExist(err) {
		t.Fatal("an ingest that landed nothing promoted the manifest, locking the run")
	}

	refused := s.out()
	refused.Refusals = []Refusal{{Subject: s.items[0], Reason: "the line is ambiguous"}}
	if _, err := s.ingest(t, s.write(t, refused)); err != nil {
		t.Fatalf("an all-refusal ingest after an all-outstanding one was refused: %v", err)
	}
	if _, err := os.Stat(s.promoted()); !os.IsNotExist(err) {
		t.Fatal("an all-refusal ingest promoted the manifest")
	}

	answer := s.out()
	answer.Dispositions = []OutDisposition{{Item: s.items[0], State: issueschema.DispositionAccepted, Grounds: groundA}}
	res, err = s.ingest(t, s.write(t, answer))
	if err != nil {
		t.Fatalf("the answering ingest was refused: %v", err)
	}
	if res.Manifest == "" {
		t.Fatal("the ingest that landed a record did not promote the manifest")
	}
	if _, err := os.Stat(s.promoted()); err != nil {
		t.Fatalf("the manifest is not beside the run: %v", err)
	}
}

// TestScribeIngestRefusesASymlinkedLedgerAncestor: the ingest lists the run's
// items through a plain path, so the ledger's ancestors are judged there too; a
// ledger redirected after the assembly is refused before the listing.
func TestScribeIngestRefusesASymlinkedLedgerAncestor(t *testing.T) {
	s := assembleSession(t, positionDetection, 1, "{0}: accepted — "+groundA+".\n")
	linked := filepath.Join(s.repo, filepath.FromSlash(capture.LedgerRelPath))
	shadow := filepath.Join(s.repo, "docs", "shadow")
	if err := os.Rename(linked, shadow); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("../../docs/shadow", linked); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	o := s.out()
	o.Outstanding = []string{s.items[0]}
	if _, err := s.ingest(t, s.write(t, o)); !errors.Is(err, ErrSymlink) {
		t.Fatalf("an ingest through a symlinked ledger ancestor was not refused: %v", err)
	}
}

// Line terminators, built from their code points so no layer between the
// author and the compiler can decode an escape into the wrong byte.
var (
	termLF   = string(rune(0x0a))
	termCR   = string(rune(0x0d))
	termCRLF = termCR + termLF
	termLS   = string(rune(0x2028))
	termPS   = string(rune(0x2029))
)

// TestScribeIngestHoldsTheStateToTheItemsLineWhateverEndsIt: the state is held
// to the item's own line, and a line ends at any of the terminators a
// researcher's editor writes. A text whose lines end in CR, U+2028 or U+2029 is
// several lines, so one item's state is not another's (iss-2609261205176571).
func TestScribeIngestHoldsTheStateToTheItemsLineWhateverEndsIt(t *testing.T) {
	for name, term := range map[string]string{
		"LF": termLF, "CRLF": termCRLF, "CR": termCR, "U+2028": termLS, "U+2029": termPS,
	} {
		t.Run(name, func(t *testing.T) {
			s := assembleSession(t, positionDetection, 2,
				"{0}: rejected — "+groundA+"."+term+"{1}: accepted — "+groundA+"."+term)
			before := s.ledger(t)
			o := s.out()
			o.Dispositions = []OutDisposition{
				{Item: s.items[0], State: issueschema.DispositionAccepted, Grounds: groundA},
				{Item: s.items[1], State: issueschema.DispositionAccepted, Grounds: groundA},
			}
			_, err := s.ingest(t, s.write(t, o))
			if err == nil || !strings.Contains(err.Error(), "state") || !strings.Contains(err.Error(), s.items[0]) {
				t.Fatalf("a state carried only by the next line (%s-terminated) was granted to the item: %v", name, err)
			}
			if s.ledger(t) != before {
				t.Fatal("a refused payload changed the ledger")
			}

			// The rulings the lines do give land.
			s2 := assembleSession(t, positionDetection, 2,
				"{0}: rejected — "+groundA+"."+term+"{1}: accepted — "+groundA+"."+term)
			o2 := s2.out()
			o2.Dispositions = []OutDisposition{
				{Item: s2.items[0], State: issueschema.DispositionRejected, Grounds: groundA},
				{Item: s2.items[1], State: issueschema.DispositionAccepted, Grounds: groundA},
			}
			if _, err := s2.ingest(t, s2.write(t, o2)); err != nil {
				t.Fatalf("the states the %s-terminated lines carry were refused: %v", name, err)
			}
		})
	}
}

// TestScribeIngestHoldsTheStateToTheItemsPartOfALine: a line that names two
// items gives each only its own part — the text from its id to the next item id
// — so the state one item's part carries is not granted to the other item the
// line mentions in passing (iss-2609261205178776).
func TestScribeIngestHoldsTheStateToTheItemsPartOfALine(t *testing.T) {
	supplied := "{0}: rejected — " + groundA + "." + termLF +
		"{1}: accepted — " + groundA + " (unlike {0})." + termLF
	s := assembleSession(t, positionDetection, 2, supplied)
	before := s.ledger(t)
	o := s.out()
	o.Dispositions = []OutDisposition{
		{Item: s.items[0], State: issueschema.DispositionAccepted, Grounds: groundA},
		{Item: s.items[1], State: issueschema.DispositionAccepted, Grounds: groundA},
	}
	_, err := s.ingest(t, s.write(t, o))
	if err == nil || !strings.Contains(err.Error(), "state") || !strings.Contains(err.Error(), s.items[0]) {
		t.Fatalf("a state another item's part of a line carries was granted to the item it mentions: %v", err)
	}
	if s.ledger(t) != before {
		t.Fatal("a refused payload changed the ledger")
	}

	// The same line still gives the item it opens with its ruling.
	s2 := assembleSession(t, positionDetection, 2, supplied)
	o2 := s2.out()
	o2.Dispositions = []OutDisposition{
		{Item: s2.items[0], State: issueschema.DispositionRejected, Grounds: groundA},
		{Item: s2.items[1], State: issueschema.DispositionAccepted, Grounds: groundA},
	}
	if _, err := s2.ingest(t, s2.write(t, o2)); err != nil {
		t.Fatalf("the ruling each item's own part carries was refused: %v", err)
	}

	// An admission is held by the same rule.
	s3 := assembleSession(t, issueschema.PositionWidening, 2,
		"{0}: declined — "+groundA+"."+termLF+"{1}: admit it — "+groundA+" (not {0})."+termLF)
	writeFile(t, s3.repo, filepath.Join(issueschema.ReadingsRecordDir, "rdg-2609250000000009", issueschema.RunRecordFileName),
		`{"run_id":"rdg-2609250000000009","position":"comparative","candidate_run":"`+fixtureRun+`"}`)
	o3 := s3.out()
	o3.Admissions = []OutAdmission{{Item: s3.items[0], Grounds: groundA}, {Item: s3.items[1], Grounds: groundA}}
	if _, err := s3.ingest(t, s3.write(t, o3)); err == nil || !strings.Contains(err.Error(), "admission") ||
		!strings.Contains(err.Error(), s3.items[0]) {
		t.Fatalf("an admission another item's part of a line carries was granted to the item it mentions: %v", err)
	}

	// A line that names one item is read whole, so a ruling written ahead of
	// the id still counts for it.
	s4 := assembleSession(t, positionDetection, 1, "Accepted: {0} — "+groundA+"."+termLF)
	o4 := s4.out()
	o4.Dispositions = []OutDisposition{{Item: s4.items[0], State: issueschema.DispositionAccepted, Grounds: groundA}}
	if _, err := s4.ingest(t, s4.write(t, o4)); err != nil {
		t.Fatalf("a ruling ahead of the id on a line naming one item was refused: %v", err)
	}
}

// TestScribeIngestRefusesASymlinkedReadingsOrRunDir: the ingest lists the run's
// items through the readings directory and the run directory, so those two are
// judged as the ledger's ancestors are; a link planted at either after the
// assembly is refused before the listing (iss-2609261205185463).
func TestScribeIngestRefusesASymlinkedReadingsOrRunDir(t *testing.T) {
	readings := capture.LedgerRelPath + "/" + issueschema.ReadingsDir
	for name, tc := range map[string]struct{ rel, target string }{
		"readings": {readings, "../../../docs/shadow"},
		"run":      {readings + "/" + fixtureRun, "../../../../docs/shadow"},
	} {
		t.Run(name, func(t *testing.T) {
			s := assembleSession(t, positionDetection, 1, "{0}: accepted — "+groundA+"."+termLF)
			linked := filepath.Join(s.repo, filepath.FromSlash(tc.rel))
			if err := os.Rename(linked, filepath.Join(s.repo, "docs", "shadow")); err != nil {
				t.Fatal(err)
			}
			if err := os.Symlink(tc.target, linked); err != nil {
				t.Skipf("symlinks unavailable: %v", err)
			}
			if _, err := os.ReadDir(linked); err != nil {
				t.Fatalf("the planted link does not resolve, so the probe proves nothing: %v", err)
			}
			o := s.out()
			o.Outstanding = []string{s.items[0]}
			if _, err := s.ingest(t, s.write(t, o)); !errors.Is(err, ErrSymlink) {
				t.Fatalf("an ingest listing through a symlinked %s directory was not refused: %v", name, err)
			}
		})
	}
}
