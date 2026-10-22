package interview

import (
	"maps"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/intentdriven/abcd/internal/abcdhome"
)

// answersRecord is a setup record giving each id its value.
func answersRecord(interview string, values map[string]string) Record {
	rec := Record{Interview: interview, Target: TargetRepository}
	for id, v := range values {
		rec.Answers = append(rec.Answers, Answer{ID: id, Value: v, AnsweredIn: Terminal})
	}
	return rec
}

func repoWithLocalTier(t *testing.T) string {
	t.Helper()
	repo := t.TempDir()
	if err := os.MkdirAll(filepath.Join(repo, ".abcd", ".work.local"), 0o700); err != nil {
		t.Fatal(err)
	}
	return repo
}

// TestLatestGivesEachQuestionItsNewestValue: across one interview's records,
// each question id takes the value of the newest record that answers it; an
// id only an older record answers keeps that answer; a second record in the
// same second is the newer; another interview's records are not read.
func TestLatestGivesEachQuestionItsNewestValue(t *testing.T) {
	repo := repoWithLocalTier(t)
	p := Place{Repo: repo}
	at := time.Date(2026, 10, 1, 9, 0, 0, 0, time.UTC)
	write := func(rec Record, when time.Time) {
		t.Helper()
		if _, err := Write(p, rec, when); err != nil {
			t.Fatal(err)
		}
	}
	// Written out of order, so the order is the names' and never the writes'.
	write(answersRecord("setup", map[string]string{"visibility": "later", "docs_target": "skip"}), at.Add(2*time.Hour))
	write(answersRecord("setup", map[string]string{"visibility": "private", "scan_deep": "true"}), at)
	write(answersRecord("setup", map[string]string{"docs_target": "agents_md"}), at.Add(2*time.Hour))
	write(answersRecord("planning", map[string]string{"visibility": "public"}), at.Add(5*time.Hour))

	got, err := Latest(p, "setup")
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]string{"visibility": "later", "docs_target": "agents_md", "scan_deep": "true"}
	if !maps.Equal(got, want) {
		t.Fatalf("Latest = %v, want %v", got, want)
	}
}

// TestLatestWithNoRecords: a checkout with no records directory, or a home
// with none, has answered nothing, and that is not an error.
func TestLatestWithNoRecords(t *testing.T) {
	for name, p := range map[string]Place{
		"a repository": {Repo: t.TempDir()},
		"a home":       {Home: t.TempDir()},
	} {
		got, err := Latest(p, "setup")
		if err != nil || len(got) != 0 {
			t.Errorf("%s: Latest = %v, %v; want nothing and no error", name, got, err)
		}
	}
}

// TestLatestReadsTheHome: the machine's records are read from abcd's home.
func TestLatestReadsTheHome(t *testing.T) {
	home := t.TempDir()
	rec := answersRecord("setup", map[string]string{"statusline.offered": "no"})
	rec.Target = TargetMachine
	if _, err := Write(Place{Home: home}, rec, time.Now()); err != nil {
		t.Fatal(err)
	}
	got, err := Latest(Place{Home: home}, "setup")
	if err != nil || got["statusline.offered"] != "no" {
		t.Fatalf("Latest = %v, %v", got, err)
	}
	if _, err := os.Stat(abcdhome.Path(home, homeRecordsLeaf)); err != nil {
		t.Fatal(err)
	}
}

// TestLatestRefusesWhatItCannotTrust: a record that does not parse, a record
// for another interview under this one's name, a record that is a link, a
// bad interview name and an ambiguous place are each refused, never skipped,
// so an answer is never read from a file abcd did not write.
func TestLatestRefusesWhatItCannotTrust(t *testing.T) {
	plant := func(t *testing.T, name, body string) Place {
		t.Helper()
		repo := repoWithLocalTier(t)
		dir := filepath.Join(repo, filepath.FromSlash(RecordsRel))
		if err := os.MkdirAll(dir, 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
		return Place{Repo: repo}
	}
	cases := map[string]Place{
		"not JSON":          plant(t, "setup-20261001T090000Z.json", "{"),
		"unknown field":     plant(t, "setup-20261001T090000Z.json", `{"schema_version":1,"interview":"setup","target":"repository","answers":[],"x":1}`),
		"another interview": plant(t, "setup-20261001T090000Z.json", `{"schema_version":1,"interview":"planning","target":"repository","answers":[]}`),
		"another schema":    plant(t, "setup-20261001T090000Z.json", `{"schema_version":2,"interview":"setup","target":"repository","answers":[]}`),
		"both places":       {Repo: t.TempDir(), Home: t.TempDir()},
		"no place":          {},
	}
	linked := repoWithLocalTier(t)
	dir := filepath.Join(linked, filepath.FromSlash(RecordsRel))
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	outside := filepath.Join(t.TempDir(), "elsewhere.json")
	if err := os.WriteFile(outside, []byte(`{"schema_version":1,"interview":"setup","target":"repository","answers":[]}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(dir, "setup-20261001T090000Z.json")); err != nil {
		t.Fatal(err)
	}
	cases["a linked record"] = Place{Repo: linked}
	for name, p := range cases {
		if got, err := Latest(p, "setup"); err == nil {
			t.Errorf("%s: accepted, giving %v", name, got)
		}
	}
	if _, err := Latest(Place{Repo: t.TempDir()}, "../setup"); err == nil {
		t.Error("a bad interview name was accepted")
	}
	// A file that is not a record's name is no record, and is not read.
	p := plant(t, "notes.txt", "{")
	if got, err := Latest(p, "setup"); err != nil || len(got) != 0 {
		t.Errorf("a stray file: Latest = %v, %v", got, err)
	}
}
