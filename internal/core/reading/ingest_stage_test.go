package reading

// ingest_stage_test.go is itd-185's ac-2 and ac-10, plus the item-level
// granularity spc-63 records rather than leaving to be discovered at the audit.

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/capture"
	"github.com/intentdriven/abcd/internal/fsutil"
)

// withFault arms the staged-write protocol's test seam for one step and restores
// it afterwards. The seam is the only way to execute the crash window from
// inside the process; a protocol whose crash behaviour is never run is a
// protocol nobody has checked.
func withFault(t *testing.T, step string) {
	t.Helper()
	prior := ingestFault
	ingestFault = func(at string) error {
		if at == step {
			return errors.New("injected fault at " + at)
		}
		return nil
	}
	t.Cleanup(func() { ingestFault = prior })
}

// TestRunMetadataLandsLast is ac-2 at both of the protocol's windows.
//
// The criterion is a state, not an instant: after the fault AND the next
// invocation, no reading records and no run metadata are durable for that run,
// and the orphaned stage has been named and cleared. Injecting at the earlier
// window alone would leave the later one — where the ledger records have already
// landed — untested, which is the window that actually needs a rollback.
func TestRunMetadataLandsLast(t *testing.T) {
	for _, step := range []string{faultAfterStage, faultAfterLedger} {
		t.Run(step, func(t *testing.T) {
			f := newIngestFixture(t, "detection")
			doc := f.payload(2)

			withFault(t, step)
			if _, err := f.ingest(doc); err == nil {
				t.Fatal("the injected fault did not stop the ingest")
			}
			ingestFault = nil

			// The commit marker is the whole claim: whatever else survived the
			// fault, the run never happened.
			if f.exists(ReadingsRecordDir + "/" + f.runID + "/" + RunFileName) {
				t.Fatal("the run metadata landed despite a fault before it")
			}
			if !f.exists(IngestStageDir + "/" + f.runID) {
				t.Fatal("the fault left no stage, so a later invocation has no evidence to find")
			}

			// The next invocation names the orphan and clears it, and the
			// rollback leaves nothing durable for the run.
			f.parkRun("rdg-2608310000000009", "detection", AssemblerVersion())
			next := f.payload(1)
			next["run_id"] = "rdg-2608310000000009"
			next["manifest_sha256"] = f.manifestHashOf("rdg-2608310000000009")
			res := f.mustIngest(next)

			named := false
			for _, run := range res.ClearedStages {
				if run == f.runID {
					named = true
				}
			}
			if !named {
				t.Errorf("the next invocation cleared %v, which does not name the orphaned run %s",
					res.ClearedStages, f.runID)
			}
			f.nothingDurable(f.runID)
		})
	}
}

// TestOrphanedStageIsReportedAndCleared is the sweep on its own, over an orphan
// this test constructs rather than one an injected fault produced — so the sweep
// is proved against the state, not only against the path that reaches it.
//
// Three things must be true at once: the orphan is named, its half-written
// records are gone, and a directory the sweep has no business in is untouched.
func TestOrphanedStageIsReportedAndCleared(t *testing.T) {
	f := newIngestFixture(t, "detection")
	orphan := "rdg-2608310000000003"

	f.write(IngestStageDir+"/"+orphan+"/"+stageFileName,
		[]byte(`{"_type":"`+StageType+`","run_id":"`+orphan+`","records":["rdi-2608310000000004"]}`))
	f.write(".abcd/work/issues/readings/"+orphan+"/rdi-2608310000000004.md", []byte("---\nid: rdi-2608310000000004\n---\n"))
	// A file the sweep must not touch: it is not a reading record, and a
	// rollback that cleared the directory would take it with the rest.
	f.write(".abcd/work/issues/readings/"+orphan+"/NOTES.md", []byte("a person put this here\n"))
	// A stage directory that is not a run id at all. The sweep leaves it alone
	// rather than deleting somebody's file on a guess.
	f.write(IngestStageDir+"/notes.txt", []byte("not a run\n"))

	res := f.mustIngest(f.payload(1))
	if len(res.ClearedStages) != 1 || res.ClearedStages[0] != orphan {
		t.Fatalf("the sweep cleared %v, want exactly [%s]", res.ClearedStages, orphan)
	}
	if f.exists(IngestStageDir + "/" + orphan) {
		t.Error("the orphaned stage survived the sweep")
	}
	if f.exists(".abcd/work/issues/readings/" + orphan + "/rdi-2608310000000004.md") {
		t.Error("the orphaned run's reading record survived the rollback")
	}
	// A delete in the COMMITTED tier is reported by id. "Cleared an orphaned
	// stage" does not tell an operator that records left the ledger with it.
	if len(res.RolledBack) != 1 || res.RolledBack[0] != "rdi-2608310000000004" {
		t.Errorf("the sweep rolled back %v; it removed rdi-2608310000000004 from the ledger and has to "+
			"say so", res.RolledBack)
	}
	if !f.exists(".abcd/work/issues/readings/" + orphan + "/NOTES.md") {
		t.Error("the rollback removed a file that is not a reading record")
	}
	if !f.exists(IngestStageDir + "/notes.txt") {
		t.Error("the sweep removed a stage entry that is not a run id")
	}
}

// TestOrphanSweepLeavesACommittedRunAlone is the sweep's other half, and it is
// the one that makes the rollback safe to have: a stage beside a run whose
// commit marker DID land is a leftover from a crash after the marker. The run
// happened, and only the stage goes.
func TestOrphanSweepLeavesACommittedRunAlone(t *testing.T) {
	f := newIngestFixture(t, "detection")
	res := f.mustIngest(f.payload(2))
	if len(res.Records) != 2 {
		t.Fatalf("the first run landed %d of 2 records", len(res.Records))
	}
	before := f.ledgerRecords(f.runID)

	// A stage reappears beside the completed run.
	f.write(IngestStageDir+"/"+f.runID+"/"+stageFileName,
		[]byte(`{"_type":"`+StageType+`","run_id":"`+f.runID+`","records":[]}`))

	f.parkRun("rdg-2608310000000005", "detection", AssemblerVersion())
	next := f.payload(1)
	next["run_id"] = "rdg-2608310000000005"
	next["manifest_sha256"] = f.manifestHashOf("rdg-2608310000000005")
	f.mustIngest(next)

	if f.exists(IngestStageDir + "/" + f.runID) {
		t.Error("the leftover stage survived")
	}
	if got := f.ledgerRecords(f.runID); len(got) != len(before) {
		t.Errorf("the sweep rolled back a COMMITTED run: %v became %v", before, got)
	}
	if !f.exists(ReadingsRecordDir + "/" + f.runID + "/" + RunFileName) {
		t.Error("the sweep removed a committed run's commit marker")
	}
}

// TestItemLevelViolationLandsTheRest is the refusal granularity spc-63 records:
// an item-level violation refuses that item and lands the rest, naming the
// refused item's ordinal and the rule.
func TestItemLevelViolationLandsTheRest(t *testing.T) {
	f := newIngestFixture(t, "detection")
	doc := f.payload(3)
	doc["items"].([]any)[1].(map[string]any)[PatternField] = ""

	res := f.mustIngest(doc)
	if len(res.Records) != 2 {
		t.Fatalf("an item-level refusal landed %d records, want the 2 survivors", len(res.Records))
	}
	if len(res.RefusedItems) != 1 {
		t.Fatalf("reported %d refusals, want 1: %v", len(res.RefusedItems), res.RefusedItems)
	}
	if res.RefusedItems[0].Ordinal != 2 {
		t.Errorf("the refusal names ordinal %d, want 2", res.RefusedItems[0].Ordinal)
	}
	if strings.TrimSpace(res.RefusedItems[0].Rule) == "" {
		t.Error("the refusal names no rule")
	}

	// And it is durable on the run record, so the refusal outlives the render.
	run := f.readRunRecord(f.runID)
	if len(run.RefusedItems) != 1 || run.RefusedItems[0].Ordinal != 2 {
		t.Errorf("the run record carries refusals %v", run.RefusedItems)
	}
	if len(run.Records) != 2 {
		t.Errorf("the run record names %d records, want 2", len(run.Records))
	}
}

// TestListLevelRefusalWritesRefusalRecordOnly is ac-10: a run refused at list
// level leaves a refusal record carrying the run metadata and the named reason,
// and no reading records.
func TestListLevelRefusalWritesRefusalRecordOnly(t *testing.T) {
	f := newIngestFixture(t, "detection")
	doc := f.payload(3)
	doc["regime"] = RegimeEvaluative

	res, err := f.ingest(doc)
	if err == nil {
		t.Fatal("a list-level violation was accepted")
	}
	if res.RefusalPath == "" {
		t.Fatal("the refusal reported no record path")
	}

	rec := f.readRefusalRecord(f.runID)
	if rec.Type != RefusalType {
		t.Errorf("the refusal record states _type %q", rec.Type)
	}
	if rec.RunID != f.runID || rec.Position != f.position || rec.Regime != f.regime {
		t.Errorf("the refusal record carries the wrong run metadata: %+v", rec)
	}
	if rec.ManifestSHA256 != f.manifestHash || rec.TargetCommit == "" {
		t.Errorf("the refusal record does not carry the run's manifest reference: %+v", rec)
	}
	// The WHOLE reason, not a prefix of it that happens to contain the word:
	// the recorded reason is what the operator was told, and a record that
	// carries the first hundred runes of it carries a different claim.
	if !strings.Contains(rec.Reason, RegimeEvaluative) || !strings.Contains(rec.Reason, "refuses the run") {
		t.Errorf("the refusal record does not carry the whole named reason: %q", rec.Reason)
	}

	if got := f.ledgerRecords(f.runID); len(got) != 0 {
		t.Errorf("the refused run wrote %v into the reading-record family", got)
	}
	if f.exists(ReadingsRecordDir + "/" + f.runID + "/" + RunFileName) {
		t.Error("a refused run wrote a commit marker")
	}
	if f.exists(IngestStageDir + "/" + f.runID) {
		t.Error("a refused run left a stage behind")
	}

	// A rerun is a NEW run with a new run id, never an amendment, and the
	// refusal record of the refused one stands.
	f.parkRun("rdg-2608310000000006", "detection", AssemblerVersion())
	next := f.payload(1)
	next["run_id"] = "rdg-2608310000000006"
	next["manifest_sha256"] = f.manifestHashOf("rdg-2608310000000006")
	f.mustIngest(next)
	if !f.exists(ReadingsRecordDir + "/" + f.runID + "/" + RefusalFileName) {
		t.Error("a later run erased the earlier run's refusal record")
	}
}

// TestRunIDNeverBuildsAPathBeforeItIsChecked is the trust boundary stated as a
// property: a run id is the one payload value a path is built from, so a
// traversal id must be refused before any file is opened — and must therefore
// leave nothing outside the repository behind.
func TestRunIDNeverBuildsAPathBeforeItIsChecked(t *testing.T) {
	f := newIngestFixture(t, "detection")
	outside := filepath.Join(f.root, "..", "escaped")
	for _, id := range []string{
		"rdg-../../../escaped", "../escaped", "rdg-2608310000000001/../../escaped",
		"rdg-" + strings.Repeat("9", 400),
	} {
		doc := f.payload(1)
		doc["run_id"] = id
		if _, err := f.ingest(doc); err == nil {
			t.Errorf("run_id %q was accepted", id)
		}
	}
	if _, err := os.Stat(outside); !os.IsNotExist(err) {
		t.Errorf("a traversal run id produced %s", outside)
	}
}

// TestARerunOfACommittedRunIsRefused holds "a rerun is a NEW run with a new run
// id, never an amendment" as a check rather than a sentence.
//
// The run id is payload-chosen. Without the check a second ingest lands a second
// batch of records beside the first and rewrites run.json to name only the
// second — and the first batch is then unreachable from any run record AND
// beyond every later sweep, because the rollback bails whenever a commit marker
// exists. The refusal record is guarded for the same reason in the other
// direction: a rerun must not overwrite the refusal of the run it repeats.
func TestARerunOfACommittedRunIsRefused(t *testing.T) {
	t.Run("after a commit marker", func(t *testing.T) {
		f := newIngestFixture(t, "detection")
		doc := f.payload(2)
		first := f.mustIngest(doc)
		if len(first.Records) != 2 {
			t.Fatalf("the first run landed %d of 2", len(first.Records))
		}

		if _, err := f.ingest(doc); err == nil {
			t.Fatal("a run that already committed was ingested again")
		} else if !strings.Contains(err.Error(), "never an amendment") {
			t.Errorf("the refusal does not state the rule: %v", err)
		}
		if got := f.ledgerRecords(f.runID); len(got) != 2 {
			t.Errorf("the ledger holds %v; the rerun duplicated the run's records", got)
		}
		run := f.readRunRecord(f.runID)
		if len(run.Records) != 2 {
			t.Errorf("the run record names %d records", len(run.Records))
		}
	})

	t.Run("after a refusal record", func(t *testing.T) {
		f := newIngestFixture(t, "detection")
		bad := f.payload(1)
		bad["regime"] = RegimeEvaluative
		if _, err := f.ingest(bad); err == nil {
			t.Fatal("a regime mismatch was accepted")
		}
		before := f.readRefusalRecord(f.runID)

		if _, err := f.ingest(f.payload(1)); err == nil {
			t.Fatal("a run that was already refused was ingested again")
		}
		if after := f.readRefusalRecord(f.runID); after.Reason != before.Reason {
			t.Error("the rerun overwrote the refusal record of the run it repeated")
		}
		if f.exists(ReadingsRecordDir + "/" + f.runID + "/" + RunFileName) {
			t.Error("a rerun of a refused run wrote a commit marker beside its refusal")
		}
	})

	// And a DIFFERENT run in the same repository still lands: the rule is about
	// one run id, not about the repository having ingested before.
	f := newIngestFixture(t, "detection")
	f.mustIngest(f.payload(1))
	if res := f.mustIngest(f.nextRun(f.payload(1))); len(res.Records) != 1 {
		t.Fatalf("a second, distinct run landed %d record(s)", len(res.Records))
	}
}

// TestTheStageLockIsHeldAcrossTheSweepAndTheWrite is the peer-invocation guard.
//
// The sweep deletes committed reading records, and its only test for an orphan
// is a stage with no commit marker beside it — which is exactly what a LIVE
// ingest looks like between its ledger write and its marker. A second
// invocation would therefore roll the first one back mid-flight, and the first
// would then write a run record naming records that no longer exist and exit 0.
//
// The lock is probed from inside the fault seam, at the very window the race
// needs, with a zero timeout: a race driven by two real processes would be
// timing-dependent, and this asserts the property the race rests on instead.
func TestTheStageLockIsHeldAcrossTheSweepAndTheWrite(t *testing.T) {
	f := newIngestFixture(t, "detection")
	lock := filepath.Join(f.root, filepath.FromSlash(IngestStageDir), stageLockName)

	probed := map[string]bool{}
	prior := ingestFault
	ingestFault = func(at string) error {
		probed[at] = true
		err := fsutil.WithFileLock(lock, 0, func() error { return nil })
		if !errors.Is(err, fsutil.ErrLockContention) {
			t.Errorf("at %s the stage lock was free (%v); a peer invocation could sweep this run's "+
				"records out from under it", at, err)
		}
		return nil
	}
	t.Cleanup(func() { ingestFault = prior })

	f.mustIngest(f.payload(2))
	for _, at := range []string{faultAfterStage, faultAfterLedger} {
		if !probed[at] {
			t.Errorf("the %s window was never reached, so the lock was never probed there", at)
		}
	}

	// And the lock is RELEASED: the next ingest is not blocked by the last.
	f.mustIngest(f.nextRun(f.payload(1)))
	if err := fsutil.WithFileLock(lock, 0, func() error { return nil }); err != nil {
		t.Errorf("the stage lock was not released after the ingest returned: %v", err)
	}
}

// TestTheLedgerLockIsHeldWhileTheSweepUnlinks.
//
// The stage lock serialises ingest against ingest and says nothing about
// core/capture, whose own verbs READ these records. The sweep unlinks committed
// reading records, so without capture's ledger lock a concurrent disposition or
// promote can be reading a record as it disappears.
//
// The lock is probed from inside the unlink, for the reason the stage-lock case
// gives: a lock held across a window cannot be observed from outside the
// process, and a race driven by two real processes would be timing-dependent.
// The probe goes through capture's own exported helper rather than rebuilding
// the lock path, so it is the same lock by construction.
func TestTheLedgerLockIsHeldWhileTheSweepUnlinks(t *testing.T) {
	f := newIngestFixture(t, "detection")
	orphan := "rdg-2608310000000021"
	f.write(IngestStageDir+"/"+orphan+"/"+stageFileName,
		[]byte(`{"_type":"`+StageType+`","run_id":"`+orphan+`","records":[]}`))
	f.write(".abcd/work/issues/readings/"+orphan+"/rdi-2608310000000022.md",
		[]byte("---\nid: rdi-2608310000000022\n---\n"))

	probed := false
	prior := ingestFault
	ingestFault = func(at string) error {
		if at != faultDuringRollback {
			return nil
		}
		probed = true
		// capture's own timeout bounds this; it returns contention rather than
		// hanging, which is what makes the assertion deterministic.
		if err := capture.WithLedgerLock(f.root, func() error { return nil }); err == nil {
			t.Error("the ledger lock was free while the sweep unlinked committed records; a concurrent " +
				"disposition or promote could be reading one as it disappears")
		}
		return nil
	}
	t.Cleanup(func() { ingestFault = prior })

	f.mustIngest(f.payload(1))
	if !probed {
		t.Fatal("the unlink was never reached, so the lock was never probed")
	}
	if f.exists(".abcd/work/issues/readings/" + orphan + "/rdi-2608310000000022.md") {
		t.Error("the orphaned run's record survived the rollback")
	}
	// And it is released: the next ledger mutation is not blocked by the sweep.
	if err := capture.WithLedgerLock(f.root, func() error { return nil }); err != nil {
		t.Errorf("the ledger lock was not released after the sweep: %v", err)
	}
}

// TestAPayloadRefusedBeforeItValidatesRollsBackNothing is the sweep's safety
// rule: the rollback is a DESTRUCTIVE act in the committed tier, so it rides
// with the commit and never with a refusal.
//
// The failure this closes was durable and silent. With an orphaned stage
// present and its record already in the ledger, an ingest whose payload was
// refused at the `_type` check deleted that committed record and reported the
// `_type` error alone — an operator saw a validation error and had no way to
// learn that a record had been destroyed while it was being reported. A run
// that is being refused for a reason of its own has no business unlinking
// somebody else's records on the way out.
func TestAPayloadRefusedBeforeItValidatesRollsBackNothing(t *testing.T) {
	orphanRecord := ".abcd/work/issues/readings/rdg-2608310000000031/rdi-2608310000000032.md"
	for _, tc := range []struct {
		name   string
		break_ func(doc map[string]any)
	}{
		{"a wrong _type", func(doc map[string]any) { doc["_type"] = "abcd.reading.output/99" }},
		{"a run id that resolves to nothing", func(doc map[string]any) { doc["run_id"] = "rdg-2608310000009999" }},
		{"a regime the definition does not state", func(doc map[string]any) { doc["regime"] = RegimeEvaluative }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := newIngestFixture(t, "detection")
			orphan := "rdg-2608310000000031"
			f.write(IngestStageDir+"/"+orphan+"/"+stageFileName,
				[]byte(`{"_type":"`+StageType+`","run_id":"`+orphan+`","records":["rdi-2608310000000032"]}`))
			f.write(orphanRecord, []byte("---\nid: rdi-2608310000000032\n---\n"))

			doc := f.payload(1)
			tc.break_(doc)
			res, err := f.ingest(doc)
			if err == nil {
				t.Fatal("the broken payload was accepted")
			}
			if !f.exists(orphanRecord) {
				t.Error("a refused run deleted a committed reading record on its way out")
			}
			if len(res.RolledBack) != 0 || len(res.ClearedStages) != 0 {
				t.Errorf("a refused run swept: cleared %v, rolled back %v", res.ClearedStages, res.RolledBack)
			}
		})
	}
}

// TestASweepThatFailsPartWayReportsWhatItAlreadyRemoved is the other half of
// "whatever the sweep did is reported on every exit path". The sweep walks the
// orphans in name order, so a fault on a later one leaves earlier deletes
// already committed — and returning a bare error there loses the only record
// that they happened.
func TestASweepThatFailsPartWayReportsWhatItAlreadyRemoved(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("the denial this rests on is a directory permission, which does not bind root")
	}
	f := newIngestFixture(t, "detection")
	swept, stuck := "rdg-2608310000000041", "rdg-2608310000000042"
	for _, run := range []string{swept, stuck} {
		f.write(IngestStageDir+"/"+run+"/"+stageFileName,
			[]byte(`{"_type":"`+StageType+`","run_id":"`+run+`","records":[]}`))
	}
	f.write(".abcd/work/issues/readings/"+swept+"/rdi-2608310000000043.md",
		[]byte("---\nid: rdi-2608310000000043\n---\n"))
	f.write(".abcd/work/issues/readings/"+stuck+"/rdi-2608310000000044.md",
		[]byte("---\nid: rdi-2608310000000044\n---\n"))

	// The later orphan's directory refuses an unlink, so the sweep fails after
	// it has already removed the earlier one's record.
	stuckDir := filepath.Join(f.root, filepath.FromSlash(".abcd/work/issues/readings/"+stuck))
	if err := os.Chmod(stuckDir, 0o555); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(stuckDir, 0o755) })

	res, err := f.ingest(f.payload(1))
	if err == nil {
		t.Fatal("the sweep reported no fault, so this case proves nothing")
	}
	found := false
	for _, id := range res.RolledBack {
		if id == "rdi-2608310000000043" {
			found = true
		}
	}
	if !found {
		t.Errorf("the sweep removed rdi-2608310000000043 and then failed, reporting %v; a delete in the "+
			"committed tier is reported whatever happens next", res.RolledBack)
	}
}

// TestARefusalRollsBackTheRunsOwnCrashedAttempt is ac-10 on the path holding
// the sweep back to the commit opened up.
//
// The general sweep is deferred because it destroys OTHER runs' records and a
// run on its way to a refusal has no business doing that. This run's own
// records are a different case: a previous ingest of this id may have died
// between the ledger write and its commit marker, refuseARerun has already
// proven the id has no outcome, and ac-10's words are that a refused run leaves
// a refusal record and NO reading records. Without this a refusal.json landed
// beside the earlier attempt's records, asserting a refusal for a run the
// ledger still carried findings for.
func TestARefusalRollsBackTheRunsOwnCrashedAttempt(t *testing.T) {
	f := newIngestFixture(t, "detection")

	// A crashed attempt at this run: its records in the ledger, its stage
	// standing, and no commit marker.
	withFault(t, faultAfterLedger)
	if _, err := f.ingest(f.payload(2)); err == nil {
		t.Fatal("the injected fault did not stop the first attempt")
	}
	ingestFault = nil
	if got := f.ledgerRecords(f.runID); len(got) != 2 {
		t.Fatalf("the crashed attempt left %v in the ledger, want its 2 records", got)
	}

	// The same run again, refused at list level.
	doc := f.payload(1)
	doc["regime"] = RegimeEvaluative
	res, err := f.ingest(doc)
	if err == nil {
		t.Fatal("a regime mismatch was accepted")
	}
	if res.RefusalPath == "" {
		t.Fatal("the refusal wrote no record")
	}
	f.nothingDurableInTheLedger(f.runID)
	if len(res.RolledBack) != 2 {
		t.Errorf("the refusal rolled back %v; it removed the earlier attempt's 2 records from the "+
			"committed ledger and has to say so", res.RolledBack)
	}
}

// TestTheBareRenderNamesAnOrphanedIngestStage.
//
// The sweep is held back to the commit path, so an orphan can now outlive the
// invocation that found it — and nothing named one. `staged_runs` reads the
// ASSEMBLY parking area, which is a different directory and lists committed
// runs alongside uncommitted ones, so an operator had no way to see that a
// crashed ingest had left reading records in the ledger for a run that never
// happened.
func TestTheBareRenderNamesAnOrphanedIngestStage(t *testing.T) {
	f := newIngestFixture(t, "detection")
	status, err := Describe(f.root)
	if err != nil {
		t.Fatal(err)
	}
	if len(status.OrphanedIngests) != 0 {
		t.Errorf("a repository with no orphan reports %v", status.OrphanedIngests)
	}

	withFault(t, faultAfterLedger)
	if _, err := f.ingest(f.payload(1)); err == nil {
		t.Fatal("the injected fault did not stop the ingest")
	}
	ingestFault = nil

	status, err = Describe(f.root)
	if err != nil {
		t.Fatal(err)
	}
	if len(status.OrphanedIngests) != 1 || status.OrphanedIngests[0] != f.runID {
		t.Fatalf("the render reports orphaned ingests %v, want [%s]", status.OrphanedIngests, f.runID)
	}
	// And it clears once a later ingest sweeps it.
	f.mustIngest(f.nextRun(f.payload(1)))
	status, err = Describe(f.root)
	if err != nil {
		t.Fatal(err)
	}
	if len(status.OrphanedIngests) != 0 {
		t.Errorf("the swept orphan is still reported: %v", status.OrphanedIngests)
	}
}

// TestTheBareRenderTellsALeftoverStageFromAnOrphan: a stage directory survives
// in two different states. An ingest that never reached its commit marker
// leaves one, and so does a commit path whose RemoveAll failed AFTER run.json
// landed. The sweep already tells them apart — rollbackRun probes the commit
// marker and leaves a committed run's records alone — so a render that calls
// both "an orphan whose records will be rolled back" is stating something the
// sweep will not do. The render probes the same marker and reports the two
// cases under different keys.
func TestTheBareRenderTellsALeftoverStageFromAnOrphan(t *testing.T) {
	f := newIngestFixture(t, "detection")
	f.mustIngest(f.payload(1))

	// The shape a failed RemoveAll leaves: the commit marker is down and the
	// stage is still there.
	f.write(IngestStageDir+"/"+f.runID+"/"+stageFileName,
		[]byte(`{"_type":"`+StageType+`","run_id":"`+f.runID+`","records":[]}`))
	if !f.exists(ReadingsRecordDir + "/" + f.runID + "/" + RunFileName) {
		t.Fatal("the fixture's run did not commit, so this case proves nothing")
	}

	status, err := Describe(f.root)
	if err != nil {
		t.Fatal(err)
	}
	if len(status.OrphanedIngests) != 0 {
		t.Errorf("a committed run's leftover stage is reported as an orphan: %v", status.OrphanedIngests)
	}
	if len(status.LeftoverStages) != 1 || status.LeftoverStages[0] != f.runID {
		t.Errorf("the render reports leftover stages %v, want [%s]", status.LeftoverStages, f.runID)
	}
}
