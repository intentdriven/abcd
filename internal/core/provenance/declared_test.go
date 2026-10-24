package provenance

import (
	"os"
	"path/filepath"
	"testing"
)

func writeRepoFile(t *testing.T, root, rel, body string) {
	t.Helper()
	p := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

// The repo's declared production mode is committed in its own file, now the
// identity it used to ride lives on each person's machine.
func TestDeclaredModeReadsItsOwnFile(t *testing.T) {
	dir := t.TempDir()
	writeRepoFile(t, dir, DeclaredModeRelPath, `{"production_mode":"dictated-and-formatted"}`)
	got, err := DeclaredMode(dir)
	if err != nil || got != ModeDictatedAndFormatted {
		t.Fatalf("DeclaredMode = %q, %v; want dictated-and-formatted", got, err)
	}
}

// A repository that has not yet migrated keeps the mode its legacy pin
// declared; the new file, once present, wins.
func TestDeclaredModeFallsBackToTheLegacyPin(t *testing.T) {
	dir := t.TempDir()
	writeRepoFile(t, dir, LegacyPinRelPath, `{"name":"A Maintainer","email":"m@example.com","production_mode":"scribe-transcribed"}`)
	got, err := DeclaredMode(dir)
	if err != nil || got != ModeScribeTranscribed {
		t.Fatalf("legacy fallback = %q, %v; want scribe-transcribed", got, err)
	}
	writeRepoFile(t, dir, DeclaredModeRelPath, `{"production_mode":"hand-written"}`)
	got, err = DeclaredMode(dir)
	if err != nil || got != ModeHandWritten {
		t.Fatalf("with both = %q, %v; want the new file's hand-written", got, err)
	}
}

// Absent everywhere means the default; an out-of-vocabulary value in either
// file is an error, never a silent default.
func TestDeclaredModeDefaultsAndRefuses(t *testing.T) {
	if got, err := DeclaredMode(t.TempDir()); err != nil || got != DefaultMode {
		t.Fatalf("no file = %q, %v; want %q", got, err, DefaultMode)
	}
	pinOnly := t.TempDir()
	writeRepoFile(t, pinOnly, LegacyPinRelPath, `{"name":"A Maintainer","email":"m@example.com"}`)
	if got, err := DeclaredMode(pinOnly); err != nil || got != DefaultMode {
		t.Fatalf("a pin with no member = %q, %v; want %q", got, err, DefaultMode)
	}
	for _, rel := range []string{DeclaredModeRelPath, LegacyPinRelPath} {
		dir := t.TempDir()
		writeRepoFile(t, dir, rel, `{"production_mode":"typed"}`)
		if _, err := DeclaredMode(dir); err == nil {
			t.Fatalf("%s: want an error on an out-of-vocabulary mode", rel)
		}
	}
	empty := t.TempDir()
	writeRepoFile(t, empty, DeclaredModeRelPath, `{}`)
	if _, err := DeclaredMode(empty); err == nil {
		t.Fatal("a declared-mode file that declares nothing must be an error")
	}
}

func TestWriteDeclaredModeRoundTrips(t *testing.T) {
	dir := t.TempDir()
	if err := WriteDeclaredMode(dir, ModeScribeTranscribed); err != nil {
		t.Fatal(err)
	}
	got, found, err := ReadDeclaredModeFile(dir)
	if err != nil || !found || got != ModeScribeTranscribed {
		t.Fatalf("round trip = %q found=%v err=%v", got, found, err)
	}
	if err := WriteDeclaredMode(dir, Mode("typed")); err == nil {
		t.Fatal("an out-of-vocabulary mode must not be written")
	}
}

// Both files are read through the guarded reader: a symlinked declaration is
// refused rather than followed.
func TestDeclaredModeRefusesASymlink(t *testing.T) {
	for _, rel := range []string{DeclaredModeRelPath, LegacyPinRelPath} {
		dir := t.TempDir()
		writeRepoFile(t, dir, "real.json", `{"production_mode":"scribe-transcribed"}`)
		if err := os.MkdirAll(filepath.Join(dir, ".abcd", "config"), 0o755); err != nil {
			t.Fatal(err)
		}
		// A relative target inside the working tree, which an os.Root alone follows.
		if err := os.Symlink(filepath.Join("..", "..", "real.json"), filepath.Join(dir, filepath.FromSlash(rel))); err != nil {
			t.Fatal(err)
		}
		if _, err := DeclaredMode(dir); err == nil {
			t.Fatalf("%s: a symlinked declaration was followed", rel)
		}
	}
}
