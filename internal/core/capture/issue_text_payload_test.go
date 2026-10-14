package capture

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/gittest"
)

// TestCaptureSurfacesCarryNoRawRuneFromARecord (iss-2610121653499543): `capture
// list --json` and `capture status --json` emit an issue whole, its body and
// its frontmatter strings included, so a hand-edited record whose body,
// remedy and found_during carry U+202E, U+009B or U+034F reaches neither
// raw, each value kept with the rune masked; the mask is the payload's, never
// the record's, so a resolve leaves the stored text as it was.
func TestCaptureSurfacesCarryNoRawRuneFromARecord(t *testing.T) {
	const runes = "\u202e\u009b\u034f"
	src := gittest.NewRepo(t)
	src.Commit("root")
	repo := src.Root()
	f := drainFixture{t: t, repo: repo, ir: filepath.Join(repo, LedgerRelPath)}
	f.file("iss-3", SeverityMinor, "bug", "guard the map")
	matches, err := filepath.Glob(filepath.Join(f.ir, "open", "iss-3-*.md"))
	if err != nil || len(matches) != 1 {
		t.Fatalf("the captured record: %v %v", matches, err)
	}
	data, err := os.ReadFile(matches[0])
	if err != nil {
		t.Fatal(err)
	}
	text := string(data)
	for _, want := range []string{"text of iss-3", `remedy: "guard the map"`, `found_during: "t"`} {
		if !strings.Contains(text, want) {
			t.Fatalf("the record carries %q: %s", want, text)
		}
	}
	text = strings.Replace(text, "text of iss-3", "text\u202e of\u009b iss\u034f-3", 1)
	text = strings.Replace(text, `remedy: "guard the map"`, "remedy: \"guard\u202e the\u009b map\u034f\"", 1)
	text = strings.Replace(text, `found_during: "t"`, "found_during: \"t\u202e\"", 1)
	if err := os.WriteFile(matches[0], []byte(text), 0o644); err != nil {
		t.Fatal(err)
	}

	list, err := List(ListRequest{RepoRoot: repo, IssuesRoot: f.ir})
	if err != nil {
		t.Fatal(err)
	}
	if len(list.Issues) != 1 || len(list.Skipped) != 0 {
		t.Fatalf("the hand-edited record still reads: %+v", list)
	}
	iss := list.Issues[0]
	if !strings.Contains(iss.Body, "text? of? iss?-3") || iss.Remedy != "guard? the? map?" || iss.FoundDuring != "t?" {
		t.Errorf("the record's text is kept, masked: body %q remedy %q found_during %q", iss.Body, iss.Remedy, iss.FoundDuring)
	}
	st, err := Status(StatusRequest{RepoRoot: repo, IssuesRoot: f.ir})
	if err != nil {
		t.Fatal(err)
	}
	for what, v := range map[string]any{"capture list": list, "capture status": st} {
		b, err := json.Marshal(v)
		if err != nil {
			t.Fatal(err)
		}
		if strings.ContainsAny(string(b), runes) {
			t.Errorf("%s: a raw rune reaches the payload: %s", what, b)
		}
	}

	if _, err := Resolve(ResolveRequest{RepoRoot: repo, IssuesRoot: f.ir, ID: "iss-3", Resolution: "fixed", Impact: "fix"}); err != nil {
		t.Fatal(err)
	}
	resolved, err := filepath.Glob(filepath.Join(f.ir, "resolved", "iss-3-*.md"))
	if err != nil || len(resolved) != 1 {
		t.Fatalf("the resolved record: %v %v", resolved, err)
	}
	after, err := os.ReadFile(resolved[0])
	if err != nil {
		t.Fatal(err)
	}
	for _, kept := range []string{"text\u202e of\u009b iss\u034f-3", "remedy: \"guard\u202e the\u009b map\u034f\"", "found_during: \"t\u202e\""} {
		if !strings.Contains(string(after), kept) {
			t.Errorf("the resolve keeps the stored text %q as it was: %s", kept, after)
		}
	}
}
