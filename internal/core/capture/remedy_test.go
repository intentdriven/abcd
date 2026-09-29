package capture

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The `remedy:` field (itd-82 decision 6, spc-2609212015054359 scope 1): the
// proposed fix a drain reads to decide whether an issue is eligible. capture
// writes it, every reader reads it, and a record written before the field
// existed, which carries the optional `suggested_fix` instead, reads that value
// as its remedy.

// TestCaptureWritesTheRemedyAndListReadsItBack is the round trip: a capture
// given a remedy commits it as the `remedy:` key, and the ledger reader carries
// it on the Issue.
func TestCaptureWritesTheRemedyAndListReadsItBack(t *testing.T) {
	repo, ir := ledger(t)
	res, err := Capture(CaptureRequest{
		RepoRoot: repo, IssuesRoot: ir, Text: "the help text says colour", Severity: SeverityMinor,
		Category: "documentation", Source: "manual-test", Slug: "help-colour", FoundDuring: "t",
		Remedy: "spell colour the British way in the help text",
	})
	if err != nil {
		t.Fatalf("Capture: %v", err)
	}
	raw, err := os.ReadFile(filepath.Join(repo, res.Path))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(raw), "\nremedy: ") {
		t.Fatalf("the record carries no remedy: key:\n%s", raw)
	}
	lr, err := List(ListRequest{RepoRoot: repo, IssuesRoot: ir, State: StateOpen})
	if err != nil {
		t.Fatal(err)
	}
	if len(lr.Issues) != 1 || len(lr.Skipped) != 0 {
		t.Fatalf("want one readable record, got %d (skipped %v)", len(lr.Issues), lr.Skipped)
	}
	if got := lr.Issues[0].Remedy; got != "spell colour the British way in the help text" {
		t.Fatalf("Remedy read back as %q", got)
	}
}

// TestCaptureWithoutARemedyWritesNoKey keeps the field optional at capture: a
// record filed without one carries no `remedy:` line, and reads an empty remedy.
func TestCaptureWithoutARemedyWritesNoKey(t *testing.T) {
	repo, ir := ledger(t)
	res, err := Capture(CaptureRequest{
		RepoRoot: repo, IssuesRoot: ir, Text: "b", Severity: SeverityMinor,
		Category: "bug", Source: "manual-test", Slug: "s", FoundDuring: "t",
	})
	if err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(filepath.Join(repo, res.Path))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(raw), "remedy:") {
		t.Fatalf("a capture without a remedy wrote the key:\n%s", raw)
	}
}

// TestTheOlderSuggestedFixReadsAsTheRemedy is the migration the spec names: a
// record carrying `suggested_fix:` and no `remedy:` reads the old key's value as
// its remedy, and a record carrying both reads `remedy:`.
func TestTheOlderSuggestedFixReadsAsTheRemedy(t *testing.T) {
	repo, ir := ledger(t)
	open := filepath.Join(ir, "open")
	if err := os.MkdirAll(open, 0o755); err != nil {
		t.Fatal(err)
	}
	head := "---\nschema_version: 1\nid: %s\nslug: s\nseverity: minor\ncategory: bug\nsource: manual-test\nfound_during: t\n%s---\n\nbody\n"
	write := func(id, extra string) {
		t.Helper()
		body := fmt.Sprintf(head, id, extra)
		if err := os.WriteFile(filepath.Join(open, id+"-s.md"), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("iss-1", "suggested_fix: \"the old spelling\"\n")
	write("iss-2", "remedy: \"the new spelling\"\nsuggested_fix: \"the old spelling\"\n")
	lr, err := List(ListRequest{RepoRoot: repo, IssuesRoot: ir, State: StateOpen})
	if err != nil {
		t.Fatal(err)
	}
	if len(lr.Skipped) != 0 || len(lr.Issues) != 2 {
		t.Fatalf("want two readable records, got %d (skipped %v)", len(lr.Issues), lr.Skipped)
	}
	want := map[string]string{"iss-1": "the old spelling", "iss-2": "the new spelling"}
	for _, iss := range lr.Issues {
		if iss.Remedy != want[iss.ID] {
			t.Errorf("%s: Remedy = %q, want %q", iss.ID, iss.Remedy, want[iss.ID])
		}
	}
}

// TestARemedyThatIsNotAStringIsRefusedByTheReader holds the field to the type
// the schema declares: a list where a scalar belongs is malformed, and the
// reader skips the record the way it skips every other malformed one.
func TestARemedyThatIsNotAStringIsRefusedByTheReader(t *testing.T) {
	fm := map[string]any{
		"schema_version": 1, "id": "iss-1", "slug": "s", "severity": "minor",
		"category": "bug", "source": "manual-test", "found_during": "t",
		"remedy": []string{"a", "b"},
	}
	if err := validateStrict(fm); err == nil || !strings.Contains(err.Error(), "remedy") {
		t.Fatalf("a list-valued remedy was not refused naming the key: %v", err)
	}
	fm["remedy"] = "one line"
	if err := validateStrict(fm); err != nil {
		t.Fatalf("a string remedy was refused: %v", err)
	}
}
