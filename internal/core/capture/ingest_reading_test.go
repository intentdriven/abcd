package capture

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/intentdriven/abcd/internal/adapter/scanner"
	"github.com/intentdriven/abcd/internal/core/issueschema"
	"github.com/intentdriven/abcd/internal/core/recordid"
	"github.com/intentdriven/abcd/internal/fsutil"
)

// One item in, one record out — each under the run's own directory, each
// carrying the run identifier in its envelope. The run-scoped identifier is what
// lets the ledger say WHICH visible world a finding was returned under.
func TestIngestWritesOneRecordPerItem(t *testing.T) {
	repo, ir := ledger(t)
	run := "rdg-2608300000000001"
	res, err := IngestReading(IngestReadingRequest{
		RepoRoot: repo, IssuesRoot: ir,
		Run: run, Manifest: "sha256:beef",
		Position: "detection", Regime: "registrative",
		Items: []ReadingItem{
			{Pattern: "constraint one", Body: bodyFor("detection")},
			{Pattern: "constraint two", Body: bodyFor("detection")},
			{Pattern: "constraint three", Body: bodyFor("detection")},
		},
	})
	if err != nil {
		t.Fatalf("IngestReading: %v", err)
	}
	if len(res.Records) != 3 {
		t.Fatalf("wrote %d records, want one per item (3)", len(res.Records))
	}

	runDir := filepath.Join(ir, issueschema.ReadingsDir, run)
	entries, err := os.ReadDir(runDir)
	if err != nil {
		t.Fatalf("read run dir: %v", err)
	}
	if len(entries) != 3 {
		t.Fatalf("run dir holds %d files, want 3", len(entries))
	}
	seen := map[string]bool{}
	for _, rec := range res.Records {
		if seen[rec.ID] {
			t.Fatalf("id %s written twice", rec.ID)
		}
		seen[rec.ID] = true
		content, err := os.ReadFile(filepath.Join(repo, filepath.FromSlash(rec.Path)))
		if err != nil {
			t.Fatalf("read %s: %v", rec.Path, err)
		}
		if !strings.Contains(string(content), "run: \""+run+"\"") {
			t.Fatalf("%s does not carry its run identifier:\n%s", rec.ID, content)
		}
	}
}

// Two runs returning the same tension carry different ids, because an id is a
// mint (adr-45) and never content-derived. Without that, a re-raise is
// indistinguishable from its first appearance and the recurrence signal dies.
func TestTwoRunsSameTensionMintDistinctIDs(t *testing.T) {
	repo, ir := ledger(t)
	same := ReadingItem{Pattern: "the same stated constraint", Body: bodyFor("detection")}

	var ids []string
	for _, run := range []string{"rdg-2608300000000001", "rdg-2608300000000002"} {
		res, err := IngestReading(IngestReadingRequest{
			RepoRoot: repo, IssuesRoot: ir,
			Run: run, Manifest: "sha256:beef",
			Position: "detection", Regime: "registrative",
			Items: []ReadingItem{same},
		})
		if err != nil {
			t.Fatalf("IngestReading(%s): %v", run, err)
		}
		ids = append(ids, res.Records[0].ID)
	}
	if ids[0] == ids[1] {
		t.Fatalf("two runs returning the same tension minted one id (%s); a re-raise must stay distinguishable", ids[0])
	}
}

// A second disposition for one item must say which one it replaces. The standing
// disposition of an item is the one no sibling supersedes, and a second record
// that cites nothing leaves two standing answers with no way to tell which is in
// force — while a hold that vanished when it was answered would take its own
// exit condition with it, so the superseded record stays in place.
func TestSecondDispositionForOneItemRequiresSupersedes(t *testing.T) {
	repo, ir, item := readingFixture(t, "detection")
	first, err := Disposition(DispositionRequest{
		RepoRoot: repo, IssuesRoot: ir, Item: item,
		State: issueschema.DispositionHeld, ExitCondition: "the closing run returns it again",
	})
	if err != nil {
		t.Fatalf("first disposition: %v", err)
	}

	_, err = Disposition(DispositionRequest{
		RepoRoot: repo, IssuesRoot: ir, Item: item,
		State: issueschema.DispositionAccepted, Grounds: "the closing run returned it",
	})
	if !errors.Is(err, ErrInvariantViolation) {
		t.Fatalf("second disposition without --supersedes: err = %v, want ErrInvariantViolation", err)
	}
	if !strings.Contains(err.Error(), first.ID) {
		t.Fatalf("the refusal must name the standing disposition to supersede; got %v", err)
	}

	second, err := Disposition(DispositionRequest{
		RepoRoot: repo, IssuesRoot: ir, Item: item,
		State: issueschema.DispositionAccepted, Grounds: "the closing run returned it",
		Supersedes: first.ID,
	})
	if err != nil {
		t.Fatalf("second disposition citing %s: %v", first.ID, err)
	}
	if second.ID == first.ID {
		t.Fatal("the superseding disposition must have its own id")
	}
	if _, err := os.Stat(filepath.Join(repo, filepath.FromSlash(first.Path))); err != nil {
		t.Fatalf("the superseded record must stay in place: %v", err)
	}
}

// pinnedMinter installs a minter whose clock is fixed and whose suffix draws are
// scripted, so a same-second id collision is certain rather than rare.
func pinnedMinter(t *testing.T, suffixes ...byte) {
	t.Helper()
	setMinter(t, recordid.Minter{
		Now:     func() time.Time { return time.Date(2026, 8, 30, 11, 0, 0, 0, time.UTC) },
		Entropy: bytes.NewReader(suffixes),
	})
}

// Two items in ONE run can draw the same id: the mint is stateless, so within a
// second the only thing separating two draws is four random digits. At 25 items
// that is a few percent; at 100 it is better than a third of the time. Staging
// two items under one id is not a rare inconvenience — it is a run that cannot be
// reconstructed, so the batch redraws, exactly as the issue allocator does.
func TestIngestRedrawsAnIntraBatchIDCollision(t *testing.T) {
	repo, ir := ledger(t)
	// 0007, 0007 (the collision), then 0008 on the redraw.
	pinnedMinter(t, 0x00, 0x07, 0x00, 0x07, 0x00, 0x08)

	res, err := IngestReading(IngestReadingRequest{
		RepoRoot: repo, IssuesRoot: ir,
		Run: "rdg-2608300000000001", Manifest: "sha256:beef",
		Position: "detection", Regime: "registrative",
		Items: []ReadingItem{
			{Pattern: "the first item", Body: bodyFor("detection")},
			{Pattern: "the second item", Body: bodyFor("detection")},
		},
	})
	if err != nil {
		t.Fatalf("IngestReading: %v", err)
	}
	if len(res.Records) != 2 {
		t.Fatalf("wrote %d records, want one per item (2)", len(res.Records))
	}
	if res.Records[0].ID == res.Records[1].ID {
		t.Fatalf("two items in one run were staged under one id (%s)", res.Records[0].ID)
	}
}

// Two runs ingested inside one UTC second can draw the same suffix: the mint is
// a second plus four digits and nothing sequences ingests. Probing only the
// CURRENT run's directory let that land — one rdi-N in two run directories, an
// item that could afterwards be neither dispositioned nor promoted, and no tree
// gate to refuse it. The probe is ledger-wide, and a hit redraws exactly as an
// intra-batch repeat does.
func TestTwoRunsAtOneInstantMintDistinctIDs(t *testing.T) {
	repo, ir := ledger(t)
	// 0007 for the first run; the second run draws 0007 again (the collision),
	// then 0008 on the redraw.
	pinnedMinter(t, 0x00, 0x07, 0x00, 0x07, 0x00, 0x08)

	ingest := func(run, pattern string) ReadingRecordRef {
		t.Helper()
		res, err := IngestReading(IngestReadingRequest{
			RepoRoot: repo, IssuesRoot: ir, Run: run, Manifest: "sha256:beef",
			Position: "detection", Regime: "registrative",
			Items: []ReadingItem{{Pattern: pattern, Body: bodyFor("detection")}},
		})
		if err != nil {
			t.Fatalf("IngestReading(%s): %v", run, err)
		}
		return res.Records[0]
	}

	first := ingest("rdg-2608300000000001", "the first run's item")
	second := ingest("rdg-2608300000000002", "the second run's item")
	if first.ID == second.ID {
		t.Fatalf("two runs at one instant minted one id (%s); the collision probe must be ledger-wide", first.ID)
	}

	// And the item is findable: one id, one record, in one run directory.
	for _, id := range []string{first.ID, second.ID} {
		if _, err := findReadingItem(ir, id); err != nil {
			t.Fatalf("findReadingItem(%s): %v", id, err)
		}
	}
}

// A mid-batch write failure reports which items LANDED. Returning a bare error
// with no record list leaves the caller unable to say what is on disk, and a
// retry then mints fresh ids for the items that already wrote — duplicating them
// inside the run directory.
func TestIngestReportsWhichItemsLandedWhenAWriteFails(t *testing.T) {
	repo, ir := ledger(t)
	pinnedMinter(t, 0x00, 0x07, 0x00, 0x08, 0x00, 0x09)

	calls := 0
	origWrite := readingWriteHook
	readingWriteHook = func(path string, data []byte) error {
		calls++
		if calls == 2 {
			return errors.New("disk full")
		}
		return fsutil.WriteFileAtomic(path, data, 0o644)
	}
	t.Cleanup(func() { readingWriteHook = origWrite })

	res, err := IngestReading(IngestReadingRequest{
		RepoRoot: repo, IssuesRoot: ir,
		Run: "rdg-2608300000000001", Manifest: "sha256:beef",
		Position: "detection", Regime: "registrative",
		Items: []ReadingItem{
			{Pattern: "the first item", Body: bodyFor("detection")},
			{Pattern: "the second item", Body: bodyFor("detection")},
			{Pattern: "the third item", Body: bodyFor("detection")},
		},
	})
	if err == nil {
		t.Fatal("a failing write must be reported as an error")
	}
	if len(res.Records) != 1 {
		t.Fatalf("the result must name the %d item(s) that landed, got %d: %+v", 1, len(res.Records), res.Records)
	}
	if !strings.Contains(err.Error(), res.Records[0].ID) {
		t.Fatalf("the error must name what landed so a retry is not blind; got %v", err)
	}
}

// The ledger lock serialises every mutation in the repository, so what is done
// while holding it is a budget every other verb pays out of. Redaction is the
// expensive part of an ingest — each scanner probes the machine identity, which
// shells out — and doing it per field per item inside the lock made a large batch
// hold the lock for seconds against a 5-second timeout, failing any concurrent
// capture, disposition or promote with allocator contention.
//
// So the text is redacted BEFORE the lock, with one scanner for the whole batch,
// and the lock holds only the mint, the probe and the write. The assertion is
// structural rather than timed: one scanner per ingest, and none built while the
// lock is held.
func TestIngestRedactsBeforeTakingTheLedgerLock(t *testing.T) {
	repo, ir := ledger(t)

	built := 0
	builtWhenLocked := -1
	origScanner := newLedgerScanner
	newLedgerScanner = func(root string) (*scanner.Scanner, error) {
		built++
		return origScanner(root)
	}
	t.Cleanup(func() { newLedgerScanner = origScanner })

	origWrite := readingWriteHook
	readingWriteHook = func(path string, data []byte) error {
		if builtWhenLocked < 0 {
			builtWhenLocked = built
		}
		return fsutil.WriteFileAtomic(path, data, 0o644)
	}
	t.Cleanup(func() { readingWriteHook = origWrite })

	items := make([]ReadingItem, 0, 50)
	for i := 0; i < 50; i++ {
		items = append(items, ReadingItem{Pattern: "a stated constraint", Body: bodyFor("detection")})
	}
	res, err := IngestReading(IngestReadingRequest{
		RepoRoot: repo, IssuesRoot: ir,
		Run: "rdg-2608300000000001", Manifest: "sha256:beef",
		Position: "detection", Regime: "registrative",
		Items: items,
	})
	if err != nil {
		t.Fatalf("IngestReading: %v", err)
	}
	if len(res.Records) != 50 {
		t.Fatalf("wrote %d records, want 50", len(res.Records))
	}
	if built != 1 {
		t.Fatalf("built %d scanners for a 50-item batch, want 1 — a scanner per field per item is what put seconds inside the lock", built)
	}
	if builtWhenLocked != built {
		t.Fatalf("%d scanner(s) were built while the ledger lock was held; redaction belongs outside it", built-builtWhenLocked)
	}
}

// TestIngestReadingRefusesARecordPastTheReadLimit is the record-size DECISION,
// and it lives here because this is the only place the exact byte count exists:
// the values are already redacted, already escaped, and the assembled string is
// what reaches the disk.
//
// Every check upstream is an estimate over one of those steps, and two attempts
// to decide it upstream failed the same way — each modelled one lengthening step
// and missed the next, writing a record past the cap that every reader of the
// family then refuses. The item is durable and can never be dispositioned, which
// is the split issueschema.RecordReadLimit exists to prevent.
func TestIngestReadingRefusesARecordPastTheReadLimit(t *testing.T) {
	repo, ir := ledger(t)
	run := "rdg-2608310000000031"

	body := bodyFor("detection")
	body["why_a_tension"] = strings.Repeat("x", issueschema.RecordReadLimit)

	res, err := IngestReading(IngestReadingRequest{
		RepoRoot: repo, IssuesRoot: ir,
		Run: run, Manifest: "sha256:beef",
		Position: "detection", Regime: "registrative",
		Items: []ReadingItem{
			{Pattern: "a pattern", Body: bodyFor("detection")},
			{Pattern: "another pattern", Body: body},
		},
	})
	if err == nil {
		t.Fatal("an item whose record exceeds the family's read limit was written")
	}
	if !errors.Is(err, ErrInvariantViolation) {
		t.Errorf("the refusal is not an invariant violation: %v", err)
	}
	for _, want := range []string{"item 2", "never be dispositioned"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("the refusal does not carry %q: %v", want, err)
		}
	}
	if len(res.Records) != 0 {
		t.Errorf("the refused run wrote %d record(s); every item is validated before any is written",
			len(res.Records))
	}

	// Nothing reached the tree: the run directory holds no record at all.
	entries, err := os.ReadDir(filepath.Join(ir, issueschema.ReadingsDir, run))
	if err == nil && len(entries) != 0 {
		t.Errorf("the refused run left %d file(s) in the ledger", len(entries))
	}

	// And the same batch without the oversize item still lands, so this is a
	// size refusal rather than a blanket one.
	ok, err := IngestReading(IngestReadingRequest{
		RepoRoot: repo, IssuesRoot: ir,
		Run: "rdg-2608310000000032", Manifest: "sha256:beef",
		Position: "detection", Regime: "registrative",
		Items: []ReadingItem{{Pattern: "a pattern", Body: bodyFor("detection")}},
	})
	if err != nil {
		t.Fatalf("a legal batch was refused: %v", err)
	}
	if len(ok.Records) != 1 {
		t.Fatalf("the legal batch landed %d record(s)", len(ok.Records))
	}
	info, err := os.Stat(filepath.Join(repo, filepath.FromSlash(ok.Records[0].Path)))
	if err != nil {
		t.Fatal(err)
	}
	if info.Size() > issueschema.RecordReadLimit {
		t.Errorf("a landed record is %d bytes, past the %d-byte limit", info.Size(), issueschema.RecordReadLimit)
	}
}
