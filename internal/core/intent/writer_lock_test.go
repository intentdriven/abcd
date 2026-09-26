package intent

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/condition"
)

// writer_lock_test.go — every writer of an intent record judges and writes the
// bytes it read under the store lock, so a write landing between its early
// reads and its write is seen, not erased. Each test lands a whole verb in that
// window through the lock seam (landAtLockEntry) rather than racing two
// goroutines on the wall clock (iss-2608301301041887): a writer that never takes
// the lock never reaches the seam, which is the finding in its original shape.

// verdictBytes is the payload writeVerdict would hand the ingest for payload:
// the host-issued policy hashes substituted in, read back as bytes.
func verdictBytes(t *testing.T, root, payload string) []byte {
	t.Helper()
	b, err := os.ReadFile(writeVerdict(t, root, payload))
	if err != nil {
		t.Fatal(err)
	}
	return b
}

// reviewMarkers counts the live review markers the record carries.
func reviewMarkers(t *testing.T, content string) int {
	t.Helper()
	_, blocks := readReviewBlocks(content)
	return len(blocks)
}

const (
	rationaleFirst  = "the ship-move writes the OWED stub and request file"
	rationaleSecond = "the second auditor read the ship-move and its request file"
)

// iss-2609261935343851: two ingests of different verdicts on one receipt. The
// one landing in the window is seen under the lock: the second replaces it in
// place and says so (Replaced), rather than reading the stale OWED stub and
// reporting a fresh ingest over a verdict it erased. Exactly one block stands.
func TestIngestSeesAVerdictLandedInTheWindow(t *testing.T) {
	root := t.TempDir()
	rcp := shipOne(t, root)
	first := verdictBytes(t, root, validVerdict(rcp))
	second := verdictBytes(t, root, strings.Replace(validVerdict(rcp), rationaleFirst, rationaleSecond, 1))
	fired := landAtLockEntry(t, func() {
		res, err := IngestVerdictBytes(root, first)
		if err != nil || res.Status != "ingested" || res.Replaced {
			t.Errorf("the ingest landing in the window must be a first ingest: %+v, %v", res, err)
		}
	})

	res, err := IngestVerdictBytes(root, second)
	if !*fired {
		t.Fatal("IngestVerdictBytes never took the store lock: the seam never fired")
	}
	if err != nil {
		t.Fatalf("the second ingest must replace the first under the lock: %v", err)
	}
	if res.Status != "ingested" || !res.Replaced {
		t.Errorf("the second ingest must report the verdict it replaced: %+v", res)
	}
	s := intentBody(t, root)
	if n := reviewMarkers(t, s); n != 1 {
		t.Fatalf("one receipt, one review block: found %d\n%s", n, s)
	}
	if !strings.Contains(s, rationaleSecond) || strings.Contains(s, rationaleFirst) {
		t.Fatalf("exactly the second verdict must stand:\n%s", s)
	}
}

// iss-2609261935343851, the dead-letter half: a malformed verdict whose ingest
// began while the receipt was OWED must not quarantine over a valid verdict
// that landed in the window. Under the lock it reads the receipt INGESTED, and a
// bad re-ingest is refused with nothing written — neither file of the
// dead-letter's two.
func TestADeadLetterNeverOverwritesAVerdictLandedInTheWindow(t *testing.T) {
	root := t.TempDir()
	rcp := shipOne(t, root)
	good := verdictBytes(t, root, validVerdict(rcp))
	bad := verdictBytes(t, root, strings.Replace(validVerdict(rcp), `"verdict": "MET"`, `"verdict": "MAYBE"`, 1))
	var landed string
	fired := landAtLockEntry(t, func() {
		if _, err := IngestVerdictBytes(root, good); err != nil {
			t.Errorf("the ingest landing in the window must succeed: %v", err)
		}
		landed = intentBody(t, root)
	})

	_, err := IngestVerdictBytes(root, bad)
	if !*fired {
		t.Fatal("IngestVerdictBytes never took the store lock: the seam never fired")
	}
	if err == nil {
		t.Fatal("a malformed verdict over an INGESTED receipt must be refused")
	}
	if s := intentBody(t, root); s != landed {
		t.Fatalf("the refused ingest changed the record the landed verdict wrote:\n%s", s)
	}
	if _, statErr := os.Stat(filepath.Join(root, reviewsDir, rcp+".deadletter.json")); !os.IsNotExist(statErr) {
		t.Fatalf("no payload may be retained when nothing is quarantined: %v", statErr)
	}
}

// iss-2609261935343851, across writers: a condition disposition landing in the
// window before an ingest survives the ingest's write.
func TestIngestKeepsAConditionDispositionLandedInTheWindow(t *testing.T) {
	root, rcp := condFixture(t, "\\\"holds while the record is one repository\\\" "+condOne)
	payload := verdictBytes(t, root, verdictWithConditions(t, rcp,
		dispositionOf(condOne, condition.Survived), dispositionOf(condTwo, condition.Survived)))
	fired := landAtLockEntry(t, func() {
		if _, err := DispositionCondition(root, condReq(condition.Falsified)); err != nil {
			t.Errorf("the disposition landing in the window must succeed: %v", err)
		}
	})

	res, err := IngestVerdictBytes(root, payload)
	if !*fired {
		t.Fatal("IngestVerdictBytes never took the store lock: the seam never fired")
	}
	if err != nil || res.Status != "ingested" {
		t.Fatalf("ingest: %+v, %v", res, err)
	}
	s := intentBody(t, root)
	if !strings.Contains(s, "<!-- abcd-condition: "+condOne+" occasion="+condItem+" -->") {
		t.Fatalf("the ingest erased the condition block that landed in the window:\n%s", s)
	}
	if !strings.Contains(s, "abcd-review: INGESTED receipt="+rcp) {
		t.Fatalf("the verdict was not written:\n%s", s)
	}
}

// iss-2609261935343851: the ingest takes the lock only where an intent store
// exists. Taking it creates the store, and a repository without one holds no
// receipt to resolve, so that refusal still writes nothing.
func TestIngestIntoARepositoryWithNoIntentStoreWritesNothing(t *testing.T) {
	root := t.TempDir()
	if _, err := IngestVerdictBytes(root, []byte(validVerdict("rcp-000000000000"))); err == nil {
		t.Fatal("a verdict with no intent store to resolve it in must be refused")
	}
	entries, err := os.ReadDir(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Fatalf("the refusal wrote into the repository: %v", entries)
	}
}
