package ahoy

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// GHSA-4q78-ccfv-f374, the second cut. The attestation's whole safety argument
// is that the record is "a write into the caller's own home" — the one place
// the environment does not reach. But HOME *is* the environment, and neither
// home-scoped reader pinned it: os.UserHomeDir() hands back $HOME verbatim, so
//
//   - a RELATIVE HOME resolves ~/.abcd against whatever directory the verb
//     happens to run in. `HOME=fakehome` in a hook makes a committed
//     `fakehome/.abcd/cache-attestation` in the checkout the caller's "own
//     home", and the class the attestation closed is reopened by the same
//     content that could not reach the data dir;
//   - an ABSOLUTE HOME inside the repository being installed is the same shape
//     dataDirHazard already refuses for the data dir ("its cache would be
//     committed bytes") with the same consequence one record further on.
//
// Both readers refuse rather than fall back, and the refusal says why.

// plantHomeRecords writes both home-scoped declaration records under home, in
// exactly the shape their writers produce — so every refusal below is provoked
// by WHERE home is and never by what the records say.
func plantHomeRecords(t *testing.T, home string) {
	t.Helper()
	dir := filepath.Join(home, ".abcd")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	sha := strings.Repeat("a", 64)
	att := "data_dir=/harness/data\nbinary_sha256=" + sha + "\ncache_trust=manifest\n"
	if err := os.WriteFile(filepath.Join(dir, cacheAttestationFile), []byte(att), 0o600); err != nil {
		t.Fatal(err)
	}
	entry := "path=" + filepath.Join(t.TempDir(), "abcd") + "\nbinary_sha256=" + sha + "\n"
	if err := os.WriteFile(filepath.Join(dir, "path-entry"), []byte(entry), 0o600); err != nil {
		t.Fatal(err)
	}
}

// assertHomeScopeRefused: neither record is honoured.
func assertHomeScopeRefused(t *testing.T, why string) {
	t.Helper()
	if rec, ok := readCacheAttestation(); ok {
		t.Errorf("the cache attestation was read from a home that %s: %+v", why, rec)
	}
	if rec, ok := readPathEntry(); ok {
		t.Errorf("the path entry was read from a home that %s: %+v", why, rec)
	}
}

func TestHomeScopedRecordsRefuseARelativeHome(t *testing.T) {
	setupHermetic(t)
	work := t.TempDir()
	plantHomeRecords(t, filepath.Join(work, "fakehome"))
	t.Chdir(work)
	t.Setenv("HOME", "fakehome")
	assertHomeScopeRefused(t, "is a relative path resolved against the working directory")
}

func TestHomeScopedRecordsRefuseAHomeInsideTheRepository(t *testing.T) {
	setupHermetic(t)
	repo := adoptableRepo(t)
	home := filepath.Join(repo, "fakehome")
	plantHomeRecords(t, home)
	t.Chdir(repo)
	t.Setenv("HOME", home)
	assertHomeScopeRefused(t, "lies inside the repository being installed")
}

// TestHomeScopedRecordsReadFromARealHome is the other half: the guard refuses
// those two shapes and NOTHING else. A home outside the working directory reads
// exactly as it did, and so does the ordinary case of a session started in the
// home directory itself — home is not "inside" the repository there, it IS the
// directory, and refusing it would break a real install to close nothing.
func TestHomeScopedRecordsReadFromARealHome(t *testing.T) {
	t.Run("outside the working directory", func(t *testing.T) {
		home, _ := setupHermetic(t)
		plantHomeRecords(t, home)
		t.Chdir(adoptableRepo(t))
		if _, ok := readCacheAttestation(); !ok {
			t.Error("a real home's attestation must still be read")
		}
		if _, ok := readPathEntry(); !ok {
			t.Error("a real home's path entry must still be read")
		}
	})

	t.Run("the working directory is the home", func(t *testing.T) {
		home, _ := setupHermetic(t)
		plantHomeRecords(t, home)
		t.Chdir(home)
		if _, ok := readCacheAttestation(); !ok {
			t.Error("a session started in the home directory must still read its attestation")
		}
		if _, ok := readPathEntry(); !ok {
			t.Error("a session started in the home directory must still read its path entry")
		}
	})
}

// TestCacheBindingProblemNamesARefusedHome: refusal, not fallback. An operator
// whose HOME cannot carry the record must be told that, not sent to "start a
// session with network access" — the hooks would write the attestation into the
// same untrusted home and the next run would refuse identically.
func TestCacheBindingProblemNamesARefusedHome(t *testing.T) {
	setupHermetic(t)
	work := t.TempDir()
	t.Chdir(work)
	t.Setenv("HOME", "fakehome")
	_, problem := cacheBindingProblem("/harness/data")
	if problem == "" {
		t.Fatal("a refused home must not bind any data directory")
	}
	if !strings.Contains(problem, "HOME") {
		t.Errorf("the refusal must name HOME as the reason, got %q", problem)
	}
}

// TestInstallSendsARefusedHomeToTheRightRemedy: the install surface must not
// answer a refused home with "start a session with network access so the hooks
// re-authenticate the cache and attest it". The hooks refuse to write the
// attestation into that home for the same reason this run refuses to read it
// from there, so the reader would be sent round a loop that cannot close.
func TestInstallSendsARefusedHomeToTheRightRemedy(t *testing.T) {
	setupUserScope(t)
	repo := adoptableRepo(t)
	home := filepath.Join(repo, "fakehome")
	if err := os.MkdirAll(home, 0o755); err != nil {
		t.Fatal(err)
	}
	// A real cache, in a directory of the harness's own shape: the only thing
	// standing between it and the PATH copy is the home the record lives in.
	seedDataCache(t, cacheArtefact)
	t.Chdir(repo)
	t.Setenv("HOME", home)

	res, err := Install(repo, installOpts(), RefusingPrompter{})
	if err != nil {
		t.Fatal(err)
	}
	joined := notesJoined(res.Notes)
	if !strings.Contains(joined, "HOME") {
		t.Errorf("the refusal must name HOME; notes = %v", res.Notes)
	}
	if strings.Contains(joined, "Start a session with network access") {
		t.Errorf("a refused home must not be answered with the re-authenticate remedy; notes = %v", res.Notes)
	}
}
