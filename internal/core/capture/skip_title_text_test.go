package capture

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/gittest"
)

// TestCaptureSurfacesCarryNoRawRuneFromTheLedger (iss-2610121542258778): a
// drain disposition's title is a record's first body line, and the ledger
// reader's refusal of an unreadable record reaches `capture list --json`,
// `capture status --json` and the mentions scan as a skipped record; a record
// whose first line, stored schema_version or claimed filename carries U+202E,
// U+009B or U+034F reaches none of them raw, each keeping its place.
func TestCaptureSurfacesCarryNoRawRuneFromTheLedger(t *testing.T) {
	const runes = "‮\u009b͏"
	src := gittest.NewRepo(t)
	src.Commit("root")
	repo := src.Root()
	writeRuleRecord(t, repo, loosenedFields)
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
	if !strings.Contains(string(data), "text of iss-3") {
		t.Fatalf("the record's first line: %s", data)
	}
	if err := os.WriteFile(matches[0], []byte(strings.Replace(string(data), "text of iss-3", "text‮ of\u009b iss͏-3", 1)), 0o644); err != nil {
		t.Fatal(err)
	}
	for name, body := range map[string]string{
		"iss-7-broken.md": "---\nschema_version: \"1‮\u009b͏\"\nid: \"iss-7\"\n---\n\nbody\n",
		"iss-9‮͏.md":      "---\nschema_version: 1\n---\n\nbody\n",
	} {
		if err := os.WriteFile(filepath.Join(f.ir, "open", name), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	p := f.plan()
	v := verdictOf(t, p, "iss-3")
	if strings.ContainsAny(v.Title, runes) || v.Title != "text? of? iss?-3" {
		t.Errorf("the disposition's title is the first line, masked: %q", v.Title)
	}
	res, err := List(ListRequest{RepoRoot: repo, IssuesRoot: f.ir})
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Skipped) != 2 {
		t.Fatalf("both unreadable files are skipped, named: %+v", res.Skipped)
	}
	// The list's skipped records; its issues carry their own stored text,
	// a route of its own.
	for what, v := range map[string]any{"the plan": p, "the list's skipped records": res.Skipped} {
		b, err := json.Marshal(v)
		if err != nil {
			t.Fatal(err)
		}
		if strings.ContainsAny(string(b), runes) {
			t.Errorf("%s: a raw rune reaches the payload: %s", what, b)
		}
	}
}

// TestAReadingRecordsSchemaVersionIsNeverEchoedRaw: the reading and
// disposition families' version check, the issue validator's twin, printed a
// stored schema_version with %v; it is masked like the issue validator's.
func TestAReadingRecordsSchemaVersionIsNeverEchoedRaw(t *testing.T) {
	for _, bad := range []string{"‮", "\u009b", "͏"} {
		err := requireSchemaVersion(map[string]any{"schema_version": "1" + bad})
		if err == nil || strings.Contains(err.Error(), bad) || !strings.Contains(err.Error(), "unsupported schema_version 1?") {
			t.Errorf("%q: the version is refused, masked: %v", bad, err)
		}
	}
}
