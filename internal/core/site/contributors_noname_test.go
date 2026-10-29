package site

// The site abcd builds names no person (iss-2610291028185585). The human
// authors of record are published as a COUNT — how many people, and how many
// commits they made — and no author's name, forge login or profile link
// reaches a rendered page or the record JSON. The bots-and-tools row and the
// Assisted-by tally are unchanged: they name tools and models, not people.

import (
	"encoding/json"
	"strings"
	"testing"
)

// TestContributorsNameNoPerson builds the fixture with three persona authors on
// top of its own, each committing under more than one identity, and holds every
// published surface to the rule: no persona name, login or profile anywhere,
// and a humans count that folds each person's identities into one.
func TestContributorsNameNoPerson(t *testing.T) {
	f := newFixture(t)
	// Alice commits from both forms of the forge's privacy address, under two
	// spellings of her name: one person, by the login the address carries.
	f.commitBy("2026-03-08T09:00:00+00:00", "docs: alice one", "Alice <1234+alice@users.noreply.github.com>")
	f.commitBy("2026-03-08T10:00:00+00:00", "docs: alice two", "Alice <1234+alice@users.noreply.github.com>")
	f.commitBy("2026-03-08T11:00:00+00:00", "docs: alice three", "alice <alice@users.noreply.github.com>")
	// Bob commits from one mailbox spelled in two cases, under two names: one
	// person, by the address.
	f.commitBy("2026-03-09T09:00:00+00:00", "docs: bob one", "Bob <bob@example.com>")
	f.commitBy("2026-03-09T10:00:00+00:00", "docs: bob two", "Bob B <Bob@Example.COM>")
	// Carol commits once.
	f.commitBy("2026-03-10T09:00:00+00:00", "docs: carol one", "Carol <carol@example.org>")
	// A forge bot stays named in its own row.
	f.git("2026-03-11T09:00:00+00:00",
		"-c", "user.name=depbot[bot]", "-c", "user.email=depbot@example.invalid",
		"commit", "--allow-empty", "-m", "chore: bump a dependency\n\nAssisted-by: None")

	out := t.TempDir()
	buildFixture(t, f, out)

	// The fixture's own author has six commits on HEAD (its golden record
	// says so); the personas add six more.
	const wantPeople, wantCommits = 4, 12

	forbidden := []string{
		"Alice", "alice", "Bob", "Carol", "carol",
		"github.com/alice", "noreply.github.com", "example.com", "example.org",
	}
	for _, rel := range []string{"contributors/index.html", "record/health/index.html", "record.json"} {
		body := outFile(t, out, rel)
		for _, word := range forbidden {
			if strings.Contains(body, word) {
				t.Errorf("%s carries %q: the site names a person", rel, word)
			}
		}
	}

	page := outFile(t, out, "contributors/index.html")
	if strings.Contains(page, ">Fixture<") {
		t.Error("the contributors page names the fixture's author")
	}
	for _, want := range []string{
		`<b class="tnum">4</b> Authors of record`,
		`<b class="tnum">12</b> commits`,
		`depbot[bot]`,
		`assistant-model-1`,
	} {
		if !strings.Contains(page, want) {
			t.Errorf("the contributors page is missing %q", want)
		}
	}

	var raw struct {
		Authorship map[string]json.RawMessage `json:"authorship"`
	}
	if err := json.Unmarshal([]byte(outFile(t, out, "record.json")), &raw); err != nil {
		t.Fatal(err)
	}
	for _, gone := range []string{"humans", "profile"} {
		if _, ok := raw.Authorship[gone]; ok {
			t.Errorf("record.json authorship still carries %q", gone)
		}
	}
	rec := outFile(t, out, "record.json")
	if strings.Contains(rec, `"profile"`) || strings.Contains(rec, `"Fixture"`) {
		t.Error("record.json carries an author name or profile")
	}
	var people, commits int
	if err := json.Unmarshal(raw.Authorship["human_authors"], &people); err != nil {
		t.Fatalf("record.json authorship has no human_authors count: %v", err)
	}
	if err := json.Unmarshal(raw.Authorship["human_commits"], &commits); err != nil {
		t.Fatalf("record.json authorship has no human_commits count: %v", err)
	}
	if people != wantPeople || commits != wantCommits {
		t.Errorf("humans = %d people / %d commits, want %d / %d", people, commits, wantPeople, wantCommits)
	}
	if !strings.Contains(rec, `"depbot[bot]"`) {
		t.Error("record.json lost the bots row")
	}
}
