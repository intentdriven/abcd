package capture

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/gittest"
)

// findingkey_test.go covers the ledger's half of G1 (itd-2609291923559186): a
// record the consistency pass files carries the finding's stable key as
// `finding_key:`, the ledger reader admits it, and capture writes it in its one
// shape only.

const goodFindingKey = "cfk-0123456789abcdef"

// TestCaptureWritesTheFindingKeyAndTheReaderReadsIt: a key on the request is
// written bare into the record's frontmatter and read back on the Issue, and a
// record without one carries none.
func TestCaptureWritesTheFindingKeyAndTheReaderReadsIt(t *testing.T) {
	root := gittest.NewRepo(t).Root()
	keyed, err := testCapture(CaptureRequest{RepoRoot: root, Text: "a keyed finding", Severity: "minor",
		Category: "inconsistency", Source: "agent-finding", FoundDuring: "fixture", FindingKey: goodFindingKey})
	if err != nil {
		t.Fatal(err)
	}
	plain, err := testCapture(CaptureRequest{RepoRoot: root, Text: "an ordinary finding", Severity: "minor",
		Category: "bug", Source: "user-observation", FoundDuring: "fixture"})
	if err != nil {
		t.Fatal(err)
	}
	if raw := readLedgerFile(t, root, keyed.Path); !strings.Contains(raw, "\nfinding_key: "+goodFindingKey+"\n") {
		t.Fatalf("the record does not carry finding_key bare in its frontmatter:\n%s", raw)
	}
	if raw := readLedgerFile(t, root, plain.Path); strings.Contains(raw, "finding_key") {
		t.Fatalf("a record filed without a key carries one:\n%s", raw)
	}
	list, err := List(ListRequest{RepoRoot: root, State: StateOpen})
	if err != nil {
		t.Fatal(err)
	}
	if len(list.Skipped) != 0 || len(list.Issues) != 2 {
		t.Fatalf("the reader read %d records and skipped %+v; want both read", len(list.Issues), list.Skipped)
	}
	for _, iss := range list.Issues {
		want := ""
		if iss.ID == keyed.ID {
			want = goodFindingKey
		}
		if iss.FindingKey != want {
			t.Errorf("%s reads finding_key %q, want %q", iss.ID, iss.FindingKey, want)
		}
	}
}

// TestCaptureRefusesAMalformedFindingKey: capture refuses a key outside its one
// shape as a request refusal with nothing written, and the ledger reader
// refuses a record carrying one, as record_schema reports it.
func TestCaptureRefusesAMalformedFindingKey(t *testing.T) {
	for _, bad := range []string{"cfk-123", "CFK-0123456789abcdef", "cfk-0123456789ABCDEF", "iss-12", "cfk-0123456789abcdef\nx: y"} {
		root := gittest.NewRepo(t).Root()
		_, err := testCapture(CaptureRequest{RepoRoot: root, Text: "a finding", Severity: "minor",
			Category: "inconsistency", Source: "agent-finding", FoundDuring: "fixture", FindingKey: bad})
		if err == nil || !errors.Is(err, ErrRequestRefused) || !strings.Contains(err.Error(), "finding_key") {
			t.Errorf("capture with finding_key %q = %v; want a request refusal naming finding_key", bad, err)
		}
		if list, _ := List(ListRequest{RepoRoot: root, State: StateAll}); len(list.Issues) != 0 {
			t.Errorf("a refused capture wrote %+v", list.Issues)
		}
	}
	rec := "---\nschema_version: 1\nid: iss-5\nslug: keyed\nseverity: minor\ncategory: inconsistency\nsource: agent-finding\nfound_during: \"fixture\"\nfinding_key: %s\n---\n\nbody\n"
	if err := ReadRefusal(fmt.Sprintf(rec, goodFindingKey), "open", "iss-5-keyed.md"); err != nil {
		t.Fatalf("the reader refuses a well-formed finding_key: %v", err)
	}
	for _, bad := range []string{"cfk-123", "[]", `""`} {
		if err := ReadRefusal(fmt.Sprintf(rec, bad), "open", "iss-5-keyed.md"); err == nil {
			t.Errorf("the reader accepts finding_key %s", bad)
		}
	}
}
