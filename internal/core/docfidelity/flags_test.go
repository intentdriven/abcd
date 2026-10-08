package docfidelity

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// iss-2610050259233425: --apply flags the drafted replacement, and a person
// then tidies the sentence by hand before committing, so the flag named text
// the brief does not carry. A flag whose replacement no line of its chapter
// contains is reported, on the flag's own line in the flags file.
func TestUnlandedFlagsReportsAHandTidiedReplacement(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, ChaptersDir)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	chapter := "intro\nCapture prints YAML.\nGuard reads stdin.\n"
	if err := os.WriteFile(filepath.Join(dir, "06-capture.md"), []byte(chapter), 0o644); err != nil {
		t.Fatal(err)
	}
	at := time.Date(2026, 10, 5, 0, 0, 0, 0, time.UTC)
	edits := []Edit{
		{Chapter: "06-capture.md", Sentence: "Capture prints YAML.", Replacement: "Capture prints JSON.", Evidence: "cli.go:1"},
		{Chapter: "06-capture.md", Sentence: "Guard reads stdin.", Replacement: "Guard reads its argument.", Evidence: "guard.go:1"},
	}
	if _, err := Apply(root, edits, "c1", at); err != nil {
		t.Fatal(err)
	}
	if misses, err := UnlandedFlags(root); err != nil || len(misses) != 0 {
		t.Fatalf("freshly applied flags reported as unlanded: %+v, %v", misses, err)
	}

	// The person tidies the second sentence by hand and commits it.
	path := filepath.Join(dir, "06-capture.md")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	tidied := strings.Replace(string(data), "Guard reads its argument.", "Guard reads the command it is given.", 1)
	if err := os.WriteFile(path, []byte(tidied), 0o644); err != nil {
		t.Fatal(err)
	}
	misses, err := UnlandedFlags(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(misses) != 1 {
		t.Fatalf("a hand-tidied replacement is not reported: %+v", misses)
	}
	m := misses[0]
	if m.File != FlagsPath || m.Chapter != "06-capture.md" || m.Replacement != "Guard reads its argument." {
		t.Fatalf("the miss names the wrong flag: %+v", m)
	}
	flags, err := os.ReadFile(filepath.Join(root, FlagsPath))
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(string(flags), "\n")
	if m.Line < 1 || m.Line > len(lines) || !strings.Contains(lines[m.Line-1], `"replacement": "Guard reads its argument."`) {
		t.Fatalf("the miss is on line %d, not the flag's replacement line", m.Line)
	}

	// Recording the sentence as committed clears it.
	refreshed := strings.Replace(string(flags), `"Guard reads its argument."`, `"Guard reads the command it is given."`, 1)
	if err := os.WriteFile(filepath.Join(root, FlagsPath), []byte(refreshed), 0o644); err != nil {
		t.Fatal(err)
	}
	if misses, err := UnlandedFlags(root); err != nil || len(misses) != 0 {
		t.Fatalf("a flag recording the committed sentence is still reported: %+v, %v", misses, err)
	}
}

// A replacement split across lines by a rewrap is carried by no one line, and
// a flag naming a chapter the brief no longer holds names nothing it carries.
func TestUnlandedFlagsReadsLinesAndChapters(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, ChaptersDir)
	if err := os.MkdirAll(filepath.Dir(filepath.Join(root, FlagsPath)), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "05-intent.md"), []byte("is reported on the advisory row, naming the\nfield, and never withholds readiness\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	flags := `{"schema_version": 1, "flags": [
  {"chapter": "05-intent.md", "sentence": "s", "replacement": "naming the field, and never", "commit": "c", "applied": "t"},
  {"chapter": "05-intent.md", "sentence": "s", "replacement": "naming the", "commit": "c", "applied": "t"},
  {"chapter": "99-gone.md", "sentence": "s", "replacement": "anything", "commit": "c", "applied": "t"},
  {"chapter": "../escape.md", "sentence": "s", "replacement": "anything", "commit": "c", "applied": "t"}
]}
`
	if err := os.WriteFile(filepath.Join(root, FlagsPath), []byte(flags), 0o644); err != nil {
		t.Fatal(err)
	}
	misses, err := UnlandedFlags(root)
	if err != nil {
		t.Fatal(err)
	}
	var got []string
	for _, m := range misses {
		got = append(got, m.Chapter+"|"+m.Replacement)
		if m.Line != 2 && m.Line != 4 && m.Line != 5 {
			t.Errorf("miss on line %d for %+v", m.Line, m)
		}
		if m.Why == "" {
			t.Errorf("miss without a reason: %+v", m)
		}
	}
	want := []string{"05-intent.md|naming the field, and never", "99-gone.md|anything", "../escape.md|anything"}
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("misses = %v, want %v", got, want)
	}
}

// No flags file is no flag, and a flags file the applier could not read is an
// error, never an empty report.
func TestUnlandedFlagsAbsentAndMalformed(t *testing.T) {
	root := t.TempDir()
	if misses, err := UnlandedFlags(root); err != nil || misses != nil {
		t.Fatalf("no flags file: %+v, %v", misses, err)
	}
	if err := os.MkdirAll(filepath.Dir(filepath.Join(root, FlagsPath)), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, FlagsPath), []byte(`{"schema_version": 2, "flags": []}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := UnlandedFlags(root); err == nil {
		t.Fatal("a flags file of an unknown schema read as no flags")
	}
}
