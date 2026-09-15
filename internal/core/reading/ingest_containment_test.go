package reading

// ingest_containment_test.go holds the ingest verb inside the repository it was
// pointed at.
//
// Two untrusted inputs reach this verb, not one. The obvious one is the payload.
// The second is the REPOSITORY TREE: this project already treats a hostile clone
// as in scope — fsutil names "the shape a hostile clone commits as git mode
// 120000", and capture refuses a symlinked run directory before walking it — and
// this verb deletes and writes in that tree. A symlink committed at the ledger's
// run directory, at the durable readings tree, or at the stage root would
// otherwise redirect a delete or a write outside the repository root. The
// orphan sweep runs only once the whole payload has validated
// (iss-2608311517509690), so every case that needs the sweep reached drives it
// with a LEGAL payload: a case that fired the verb on an absent one would pass
// whether or not the containment held, because the sweep would never run.
//
// Every case here works the same way: plant the symlink, run the verb, and
// assert the directory outside the repository is untouched.

import (
	"os"
	"path/filepath"
	"testing"
)

// outsideDir makes a directory outside the repository, holding one file the verb
// must not remove, and returns it.
func (f *ingestFixture) outsideDir(withFile string) string {
	f.t.Helper()
	dir := f.t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, withFile), []byte("a file nobody asked this verb to touch\n"), 0o644); err != nil {
		f.t.Fatal(err)
	}
	return dir
}

// linkTo replaces a repo-relative path with a symlink to target — the shape a
// hostile clone commits as git mode 120000.
func (f *ingestFixture) linkTo(rel, target string) {
	f.t.Helper()
	abs := filepath.Join(f.root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
		f.t.Fatal(err)
	}
	if err := os.RemoveAll(abs); err != nil {
		f.t.Fatal(err)
	}
	if err := os.Symlink(target, abs); err != nil {
		f.t.Fatal(err)
	}
}

// assertUntouched fails unless dir still holds exactly the files it started with.
func assertUntouched(t *testing.T, dir string, want ...string) {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("read the directory outside the repository: %v", err)
	}
	var got []string
	for _, e := range entries {
		got = append(got, e.Name())
	}
	if len(got) != len(want) {
		t.Errorf("the directory outside the repository holds %v, want exactly %v", got, want)
		return
	}
	for i, name := range want {
		if got[i] != name {
			t.Errorf("the directory outside the repository holds %v, want exactly %v", got, want)
			return
		}
	}
}

// TestASymlinkedReadingsTreeCannotRedirectTheDurableWrite: a committed symlink
// at the durable readings tree must not land the manifest, the run metadata or a
// refusal record outside the repository.
func TestASymlinkedReadingsTreeCannotRedirectTheDurableWrite(t *testing.T) {
	t.Run("an accepted run", func(t *testing.T) {
		f := newIngestFixture(t, "detection")
		outside := f.outsideDir("important.txt")
		f.linkTo(ReadingsRecordDir, outside)

		if _, err := f.ingest(f.payload(1)); err == nil {
			t.Error("an ingest wrote its run metadata through a symlinked readings tree")
		}
		assertUntouched(t, outside, "important.txt")
	})

	t.Run("a refused run", func(t *testing.T) {
		f := newIngestFixture(t, "detection")
		outside := f.outsideDir("important.txt")
		f.linkTo(ReadingsRecordDir, outside)

		doc := f.payload(1)
		doc["regime"] = RegimeEvaluative
		if _, err := f.ingest(doc); err == nil {
			t.Error("a regime mismatch was accepted")
		}
		assertUntouched(t, outside, "important.txt")
	})
}

// TestASymlinkedLedgerRunCannotRedirectTheRollback: the orphan sweep deletes
// reading records, and a committed symlink at the run's ledger directory must
// not make it delete them somewhere else. The payload is legal, so the sweep is
// reached; the refusal asserted is the sweep's own.
func TestASymlinkedLedgerRunCannotRedirectTheRollback(t *testing.T) {
	f := newIngestFixture(t, "detection")
	orphan := "rdg-2608310000000011"
	outside := f.outsideDir("rdi-2608310000000012.md")

	f.write(IngestStageDir+"/"+orphan+"/"+stageFileName,
		[]byte(`{"_type":"`+StageType+`","run_id":"`+orphan+`","records":[]}`))
	f.linkTo(".abcd/work/issues/readings/"+orphan, outside)

	if _, err := f.ingest(f.payload(1)); err == nil {
		t.Error("a symlinked ledger run directory was swept rather than refused")
	}
	assertUntouched(t, outside, "rdi-2608310000000012.md")
}

// TestASymlinkedDurableRunCannotRedirectTheRollback is the same escape through
// the other directory the rollback removes from.
func TestASymlinkedDurableRunCannotRedirectTheRollback(t *testing.T) {
	f := newIngestFixture(t, "detection")
	orphan := "rdg-2608310000000013"
	outside := f.outsideDir("manifest.json")

	f.write(IngestStageDir+"/"+orphan+"/"+stageFileName,
		[]byte(`{"_type":"`+StageType+`","run_id":"`+orphan+`","records":[]}`))
	f.linkTo(ReadingsRecordDir+"/"+orphan, outside)

	// The sweep is reached only under a legal payload. The rollback's removes
	// on the durable run directory are best-effort, so the verb may or may not
	// refuse; what must hold either way is that nothing outside moved.
	_, _ = f.ingest(f.payload(1))
	assertUntouched(t, outside, "manifest.json")
}

// TestASymlinkedStageRootCannotRedirectTheSweep: the sweep clears whole stage
// directories, so a symlink at the stage root would clear them outside.
func TestASymlinkedStageRootCannotRedirectTheSweep(t *testing.T) {
	f := newIngestFixture(t, "detection")
	outside := f.t.TempDir()
	runDir := filepath.Join(outside, "rdg-2608310000000014")
	if err := os.MkdirAll(runDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(runDir, "important.txt"), []byte("keep\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	f.linkTo(IngestStageDir, outside)

	// A symlinked stage root is refused before the lock is even taken, so the
	// payload's legality is immaterial here; it is legal anyway, so that if
	// that refusal ever moved later the sweep would still be reached.
	if _, err := f.ingest(f.payload(1)); err == nil {
		t.Error("a symlinked stage root was accepted")
	}
	assertUntouched(t, outside, "rdg-2608310000000014")
	assertUntouched(t, runDir, "important.txt")
}

// TestASymlinkedParkedRunCannotRedirectTheManifestRead: the manifest is read out
// of the local tier at a path built from the payload's run id, so the read is
// contained too — a definition of the run assembled outside the repository is
// not this repository's run.
func TestASymlinkedParkedRunCannotRedirectTheManifestRead(t *testing.T) {
	f := newIngestFixture(t, "detection")
	outside := f.t.TempDir()
	parked := filepath.Join(f.root, filepath.FromSlash(DefaultRunDir), f.runID, ManifestFileName)
	raw, err := os.ReadFile(parked)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(outside, ManifestFileName), raw, 0o644); err != nil {
		t.Fatal(err)
	}
	f.linkTo(DefaultRunDir+"/"+f.runID, outside)

	if _, err := f.ingest(f.payload(1)); err == nil {
		t.Error("a manifest resolved through a symlink pointing outside the repository")
	}
}

// TestASymlinkedRunDirectoryInsideTheRepoIsRefusedToo: os.Root contains a link
// that leaves the repository, and nothing more. A link pointing at ANOTHER
// directory inside the repository is still followed by the kernel, and the
// rollback would then delete reading records out of whatever it points at.
//
// That is why the sweep refuses a symlinked directory outright rather than
// resting on containment alone — the stance core/capture takes on the same
// ledger directory.
//
// The discriminator is readDirIn's Lstat guard AS A WHOLE, not its symlink
// branch: removing that branch alone leaves this green, because the sibling
// `!fi.IsDir()` catches a symlink too (Lstat reports the link, not its target).
// The two branches are mutually redundant here, so the guard is what has to be
// mutated, and removing it turns this case red.
func TestASymlinkedRunDirectoryInsideTheRepoIsRefusedToo(t *testing.T) {
	f := newIngestFixture(t, "detection")
	orphan := "rdg-2608310000000015"

	// A real, in-repo directory holding a reading record that belongs to another
	// run entirely.
	victim := ".abcd/work/issues/readings/rdg-2608310000000016"
	f.write(victim+"/rdi-2608310000000017.md", []byte("---\nid: rdi-2608310000000017\n---\n"))

	f.write(IngestStageDir+"/"+orphan+"/"+stageFileName,
		[]byte(`{"_type":"`+StageType+`","run_id":"`+orphan+`","records":[]}`))
	// RELATIVE, and pointing inside the repository. An absolute symlink is
	// refused by the containment root on its own, so a case using one would
	// pass whether or not the explicit refusal existed — which is what made the
	// first version of this case vacuous (iss-2608311306539528).
	f.linkTo(".abcd/work/issues/readings/"+orphan, "rdg-2608310000000016")

	// Legal payload, so the sweep — and the guard under test — is reached.
	if _, err := f.ingest(f.payload(1)); err == nil {
		t.Error("a symlinked run directory was walked rather than refused")
	}
	if !f.exists(victim + "/rdi-2608310000000017.md") {
		t.Error("the rollback followed a symlink and deleted another run's record")
	}
}
