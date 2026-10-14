package capture

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/record/findingkey"
	"github.com/intentdriven/abcd/internal/gittest"
)

// findingkey_test.go covers the ledger's half of G1 (itd-2609291923559186): a
// record the consistency pass files carries the finding's stable key as
// `finding_ref:`, the ledger reader admits it, and capture writes it in its one
// shape only.

const wellFormedRef = "cfk-0123456789abcdef"

// TestCaptureWritesTheFindingKeyAndTheReaderReadsIt: a key on the request is
// written bare into the record's frontmatter and read back on the Issue, and a
// record without one carries none.
func TestCaptureWritesTheFindingKeyAndTheReaderReadsIt(t *testing.T) {
	root := gittest.NewRepo(t).Root()
	keyed, err := testCapture(CaptureRequest{RepoRoot: root, Text: "a keyed finding", Severity: "minor",
		Category: "inconsistency", Source: "agent-finding", FoundDuring: "fixture", FindingRef: wellFormedRef})
	if err != nil {
		t.Fatal(err)
	}
	plain, err := testCapture(CaptureRequest{RepoRoot: root, Text: "an ordinary finding", Severity: "minor",
		Category: "bug", Source: "user-observation", FoundDuring: "fixture"})
	if err != nil {
		t.Fatal(err)
	}
	if raw := readLedgerFile(t, root, keyed.Path); !strings.Contains(raw, "\nfinding_ref: "+wellFormedRef+"\n") {
		t.Fatalf("the record does not carry finding_ref bare in its frontmatter:\n%s", raw)
	}
	if raw := readLedgerFile(t, root, plain.Path); strings.Contains(raw, "finding_ref") {
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
			want = wellFormedRef
		}
		if iss.FindingRef != want {
			t.Errorf("%s reads finding_ref %q, want %q", iss.ID, iss.FindingRef, want)
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
			Category: "inconsistency", Source: "agent-finding", FoundDuring: "fixture", FindingRef: bad})
		if err == nil || !errors.Is(err, ErrRequestRefused) || !strings.Contains(err.Error(), "finding_ref") {
			t.Errorf("capture with finding_ref %q = %v; want a request refusal naming finding_ref", bad, err)
		}
		if list, _ := List(ListRequest{RepoRoot: root, State: StateAll}); len(list.Issues) != 0 {
			t.Errorf("a refused capture wrote %+v", list.Issues)
		}
	}
	rec := "---\nschema_version: 1\nid: iss-5\nslug: keyed\nseverity: minor\ncategory: inconsistency\nsource: agent-finding\nfound_during: \"fixture\"\nfinding_ref: %s\n---\n\nbody\n"
	if err := ReadRefusal(fmt.Sprintf(rec, wellFormedRef), "open", "iss-5-keyed.md"); err != nil {
		t.Fatalf("the reader refuses a well-formed finding_ref: %v", err)
	}
	for _, bad := range []string{"cfk-123", "[]", `""`} {
		if err := ReadRefusal(fmt.Sprintf(rec, bad), "open", "iss-5-keyed.md"); err == nil {
			t.Errorf("the reader accepts finding_ref %s", bad)
		}
	}
}

// TestTheRedactorLeavesAFindingKeyAlone: a finding key is a content hash, not a
// credential, so the ledger's redactor neither rewrites the key capture writes
// as finding_ref nor one a record's prose carries.
func TestTheRedactorLeavesAFindingKeyAlone(t *testing.T) {
	root := gittest.NewRepo(t).Root()
	key := findingkey.Derive("premise_contradiction",
		findingkey.End{Path: ".abcd/development/intents/planned/itd-10-one-spec.md", Quote: "An intent carries exactly one spec."},
		findingkey.End{Path: ".abcd/development/intents/shipped/itd-11-many-specs.md", Quote: "An intent owns one or more specs."})
	if out, n, _ := redactLedgerText(root, "finding_ref: "+key); n != 0 || !strings.Contains(out, key) {
		t.Fatalf("the redactor rewrote a finding key (%d span(s)): %q", n, out)
	}
	res, err := testCapture(CaptureRequest{RepoRoot: root, Text: "Filed under finding key " + key + ".", Severity: "minor",
		Category: "inconsistency", Source: "agent-finding", FoundDuring: "fixture", FindingRef: key})
	if err != nil {
		t.Fatal(err)
	}
	raw := readLedgerFile(t, root, res.Path)
	if res.Redacted != 0 || !strings.Contains(raw, "\nfinding_ref: "+key+"\n") || !strings.Contains(raw, "finding key "+key+".") {
		t.Fatalf("capture redacted %d span(s); want the key verbatim in frontmatter and body:\n%s", res.Redacted, raw)
	}
}

// secretShapedKey is the shape of gitleaks' default generic-api-key rule
// narrowed to a finding key: a credential word (key, token, secret, ...), up to
// twenty word characters, an assignment and then the key. The rule flagged
// `finding_key: cfk-…` on both gitleaks 8.24.3 and 8.30.1, so a repository
// running a default gitleaks failed on every record the pass filed.
var secretShapedKey = regexp.MustCompile(`(?i)(?:access|auth|api|credential|creds|key|passw(?:or)?d|secret|token)[ \t\w.-]{0,20}[\s'"]{0,3}(?:=|>|:{1,3}=|\|\||:|=>|\?=|,)[` + "`" + `'"\s=]{0,5}cfk-[0-9a-f]{16}`)

// TestAFiledRecordCarriesNoSecretShapedKey: the field a record carries its
// finding key under is named so that no default secret scanner reads the line
// as a credential assignment (itd-2609291923559186 G1).
func TestAFiledRecordCarriesNoSecretShapedKey(t *testing.T) {
	root := gittest.NewRepo(t).Root()
	res, err := testCapture(CaptureRequest{RepoRoot: root, Text: "a keyed finding", Severity: "minor",
		Category: "inconsistency", Source: "agent-finding", FoundDuring: "fixture", FindingRef: wellFormedRef})
	if err != nil {
		t.Fatal(err)
	}
	raw := readLedgerFile(t, root, res.Path)
	if m := secretShapedKey.FindString(raw); m != "" {
		t.Fatalf("the filed record carries %q, which a default gitleaks flags as generic-api-key:\n%s", m, raw)
	}
}
