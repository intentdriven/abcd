package interview

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/intentdriven/abcd/internal/core/question"
)

func sampleRecord(answeredIn string) Record {
	return Record{
		Interview: "setup",
		Target:    TargetRepository,
		Answers: []Answer{{
			ID: "visibility",
			Ask: question.Ask{Questions: []question.Question{{
				ID: "visibility", Chip: "Setup Q1", Ask: "Which answer does this repository take?",
				Options: []question.Option{{Value: "private", Label: "private"}},
				Later:   question.Option{Value: "later", Label: "Decide later"},
			}}},
			Value:      "private",
			AnsweredIn: answeredIn,
		}},
	}
}

// TestAnswersFileRefusesUnknownAndDuplicateKeys holds the answers file to
// jsonstrict: a key the schema does not name, or one repeated, refuses rather
// than dropping or overwriting an answer without a word.
func TestAnswersFileRefusesUnknownAndDuplicateKeys(t *testing.T) {
	cases := map[string]string{
		"unknown top-level key": `{"schema_version":1,"interview":"setup","answers":[],"extra":1}`,
		"unknown answer key":    `{"schema_version":1,"interview":"setup","answers":[{"id":"visibility","value":"private","why":"x"}]}`,
		"duplicate key":         `{"schema_version":1,"interview":"setup","interview":"setup","answers":[]}`,
		"duplicate answer key":  `{"schema_version":1,"interview":"setup","answers":[{"id":"visibility","value":"private","value":"public"}]}`,
		"duplicate id":          `{"schema_version":1,"interview":"setup","answers":[{"id":"visibility","value":"private"},{"id":"visibility","value":"public"}]}`,
		"another interview":     `{"schema_version":1,"interview":"planning","answers":[]}`,
		"another schema":        `{"schema_version":2,"interview":"setup","answers":[]}`,
		"answered elsewhere":    `{"schema_version":1,"interview":"setup","answers":[{"id":"visibility","value":"private","answered_in":"a browser"}]}`,
		"no value":              `{"schema_version":1,"interview":"setup","answers":[{"id":"visibility","value":""}]}`,
		"no id":                 `{"schema_version":1,"interview":"setup","answers":[{"id":" ","value":"private"}]}`,
		"trailing document":     `{"schema_version":1,"interview":"setup","answers":[]} {}`,
	}
	for name, doc := range cases {
		if _, err := ParseAnswers([]byte(doc), "setup"); err == nil {
			t.Errorf("%s: accepted %s", name, doc)
		}
	}
	good := `{"schema_version":1,"interview":"setup","answers":[{"id":"visibility","value":"private"},` +
		`{"id":"Q2","value":"later","note":"ask again after the bundle","answered_in":"Claude Code"}]}`
	a, err := ParseAnswers([]byte(good), "setup")
	if err != nil {
		t.Fatalf("the spec's own example was refused: %v", err)
	}
	if got, ok := a.Find("Q2"); !ok || got.Note != "ask again after the bundle" || got.AnsweredIn != ClaudeCode {
		t.Fatalf("Find(Q2) = %+v, %v", got, ok)
	}
	if _, ok := a.Find("docs_target"); ok {
		t.Fatal("Find answered an id the file does not hold")
	}
}

// TestWriteRecordsInTheLocalTierAndTheHome writes the record where the spec
// puts it: the repository's local tier, or the home's interviews/ for the
// machine-wide part, the stamp in the name and nowhere in the content, so two
// runs given the same answers write the same bytes.
func TestWriteRecordsInTheLocalTierAndTheHome(t *testing.T) {
	at := time.Date(2026, 10, 4, 9, 30, 15, 0, time.UTC)
	repo := t.TempDir()
	if _, err := Write(Place{Repo: repo}, sampleRecord(Terminal), at); !errors.Is(err, ErrNoLocalTier) {
		t.Fatalf("a repository without its local tier: got %v, want ErrNoLocalTier", err)
	}
	if err := os.MkdirAll(filepath.Join(repo, ".abcd", ".work.local"), 0o755); err != nil {
		t.Fatal(err)
	}
	p, err := Write(Place{Repo: repo}, sampleRecord(Terminal), at)
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(repo, ".abcd", ".work.local", "interviews", "setup-20261004T093015Z.json")
	if p != want {
		t.Fatalf("wrote %s, want %s", p, want)
	}
	again, err := Write(Place{Repo: repo}, sampleRecord(Terminal), at)
	if err != nil || again == p {
		t.Fatalf("a second record in the same second: %s, %v (it must not replace the first)", again, err)
	}
	a, _ := os.ReadFile(p)
	b, _ := os.ReadFile(again)
	if string(a) != string(b) {
		t.Fatalf("the same answers wrote different bytes:\n%s\n%s", a, b)
	}
	if strings.Contains(string(a), "2026") {
		t.Fatalf("the record carries its time in its content:\n%s", a)
	}
	var rec Record
	if err := json.Unmarshal(a, &rec); err != nil || rec.SchemaVersion != SchemaVersion || rec.Answers[0].AnsweredIn != Terminal {
		t.Fatalf("record %+v, %v:\n%s", rec, err, a)
	}
	if fi, err := os.Stat(p); err != nil || fi.Mode().Perm() != 0o600 {
		t.Fatalf("record mode %v, %v; want 0600", fi.Mode().Perm(), err)
	}

	home := t.TempDir()
	hp, err := Write(Place{Home: home}, sampleRecord(ClaudeCode), at)
	if err != nil {
		t.Fatal(err)
	}
	if hp != filepath.Join(home, ".abcd", "interviews", "setup-20261004T093015Z.json") {
		t.Fatalf("home record at %s", hp)
	}

	for name, bad := range map[string]struct {
		p   Place
		rec Record
	}{
		"answered elsewhere": {Place{Home: home}, sampleRecord("a browser")},
		"no place":           {Place{}, sampleRecord(Terminal)},
		"two places":         {Place{Repo: repo, Home: home}, sampleRecord(Terminal)},
		"a path for a name":  {Place{Home: home}, func() Record { r := sampleRecord(Terminal); r.Interview = "../x"; return r }()},
	} {
		if _, err := Write(bad.p, bad.rec, at); err == nil {
			t.Errorf("%s: written", name)
		}
	}
}

// TestWriteRefusesASymlinkedHome holds the home record to the guarded store
// maker: a ~/.abcd that is a symlink is refused, never written through.
func TestWriteRefusesASymlinkedHome(t *testing.T) {
	home := t.TempDir()
	elsewhere := t.TempDir()
	if err := os.Symlink(elsewhere, filepath.Join(home, ".abcd")); err != nil {
		t.Fatal(err)
	}
	if _, err := Write(Place{Home: home}, sampleRecord(Terminal), time.Now()); err == nil {
		t.Fatal("wrote through a symlinked ~/.abcd")
	}
	if ents, _ := os.ReadDir(elsewhere); len(ents) != 0 {
		t.Fatalf("the symlink's target gained %d entries", len(ents))
	}
}
