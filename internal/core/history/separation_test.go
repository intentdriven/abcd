package history

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/sessionkind"
)

// The digests are arbitrary hex: the check matches stamps by kind and run, and
// the digest is part of the exact token a transcript carries.
const (
	sepDigestA = "aaaaaaaaaaaa0000000000000000000000000000000000000000000000000000"
	sepDigestB = "bbbbbbbbbbbb0000000000000000000000000000000000000000000000000000"
)

func mustStamp(t *testing.T, kind sessionkind.Kind, run, digest string) string {
	t.Helper()
	s, err := sessionkind.Stamp(kind, run, digest)
	if err != nil {
		t.Fatal(err)
	}
	return s
}

// captureTranscript captures one transcript whose body carries the given lines,
// as a host retains a tool result that read a bundle or a context.
func captureTranscript(t *testing.T, repoRoot, session string, lines ...string) Record {
	t.Helper()
	body := "user: go\n" + strings.Join(lines, "\n") + "\nassistant: done\n"
	res, err := Capture(repoRoot, testRootSHA, []byte(body), CaptureMeta{SessionID: session, Kind: "native"})
	if err != nil {
		t.Fatalf("Capture %s: %v", session, err)
	}
	if !res.Wrote {
		t.Fatalf("Capture %s wrote nothing", session)
	}
	return res.Record
}

// TestCaptureRecordsContextStamps is the custodian's half: at capture the raw
// transcript is scanned for stamps and every distinct one lands in the record's
// metadata, where the check reads it. A transcript that merely quotes the
// grammar carries none.
func TestCaptureRecordsContextStamps(t *testing.T) {
	repoRoot, _ := setupStore(t)
	reading := mustStamp(t, sessionkind.Reading, "rdg-2609250000000001", sepDigestA)
	scribe := mustStamp(t, sessionkind.Scribe, "rdg-2609250000000002", sepDigestB)

	rec := captureTranscript(t, repoRoot, "sess-stamped",
		`tool_result: {"context_stamp": "`+reading+`"}`,
		`tool_result: {"context_stamp": "`+scribe+`"}`,
		`tool_result: again `+reading)
	want := []string{reading, scribe}
	if !reflect.DeepEqual(rec.ContextStamps, want) {
		t.Fatalf("Capture recorded %q, want %q", rec.ContextStamps, want)
	}
	listed, err := List(repoRoot, testRootSHA)
	if err != nil {
		t.Fatal(err)
	}
	if len(listed) != 1 || !reflect.DeepEqual(listed[0].ContextStamps, want) {
		t.Fatalf("the stored record reads back %+v, want the stamps %q", listed, want)
	}

	plain := captureTranscript(t, repoRoot, "sess-docs",
		"assistant: the stamp reads abcd.context-stamp/<kind>/<rdg-N>/<sha256-12>")
	if len(plain.ContextStamps) != 0 {
		t.Fatalf("a transcript quoting the grammar recorded %q; only a stamp is a stamp", plain.ContextStamps)
	}
}

// TestARecordWithoutContextStampsStillParses holds the field optional on read:
// a record captured before it existed parses clean and carries no stamps, so the
// record schema version does not move.
func TestARecordWithoutContextStampsStillParses(t *testing.T) {
	data := []byte("---\nschema: 3\nsession_id: s1\nroot_commit: " + testRootSHA +
		"\ncaptured_at: 2026-09-01T00:00:00Z\nsource_kind: native\nsource_sha256: abc\n" +
		"redacted_secrets: 0\nredacted_home_paths: 0\n---\nbody\n")
	r, _, err := parseRecord(data)
	if err != nil {
		t.Fatalf("a record without context_stamps does not parse: %v", err)
	}
	if len(r.ContextStamps) != 0 {
		t.Fatalf("a record without context_stamps read back %q", r.ContextStamps)
	}
	if out := string(marshalRecord(r, "body")); strings.Contains(out, fmContextStamps) {
		t.Fatalf("a record with no stamps is written with the key anyway:\n%s", out)
	}
}

// TestSeparationNamesATranscriptCarryingBothStampsOfOneRun is ac-5.
func TestSeparationNamesATranscriptCarryingBothStampsOfOneRun(t *testing.T) {
	repoRoot, _ := setupStore(t)
	run := "rdg-2609250000000001"
	rec := captureTranscript(t, repoRoot, "sess-both",
		mustStamp(t, sessionkind.Reading, run, sepDigestA),
		mustStamp(t, sessionkind.Scribe, run, sepDigestB))

	rep, err := SessionSeparation(repoRoot, testRootSHA)
	if err != nil {
		t.Fatal(err)
	}
	if rep.Unobserved {
		t.Fatalf("a store holding a stamped transcript reported unobserved: %+v", rep)
	}
	want := []Violation{{SessionID: "sess-both", File: filepath.Base(rec.Path), Run: run}}
	if !reflect.DeepEqual(rep.Violations, want) {
		t.Fatalf("violations = %+v, want %+v", rep.Violations, want)
	}
}

// TestSeparationIgnoresTwoStampsOfTwoRuns holds the exact match: a reading
// stamp of one run and a scribe stamp of another is not a session that held a
// reading and the ledger of one run.
func TestSeparationIgnoresTwoStampsOfTwoRuns(t *testing.T) {
	repoRoot, _ := setupStore(t)
	captureTranscript(t, repoRoot, "sess-two-runs",
		mustStamp(t, sessionkind.Reading, "rdg-1", sepDigestA),
		mustStamp(t, sessionkind.Scribe, "rdg-2", sepDigestB))
	rep, err := SessionSeparation(repoRoot, testRootSHA)
	if err != nil {
		t.Fatal(err)
	}
	if len(rep.Violations) != 0 {
		t.Fatalf("two stamps of two runs reported as a violation: %+v", rep.Violations)
	}
	if !reflect.DeepEqual(rep.Runs, []string{"rdg-1", "rdg-2"}) {
		t.Fatalf("runs = %q, want both", rep.Runs)
	}
}

// TestSeparationReportsNoTranscriptCarryingTwoStamps is ac-6: two transcripts,
// one stamp each, is the property held for what was seen, and the report says
// which runs it saw.
func TestSeparationReportsNoTranscriptCarryingTwoStamps(t *testing.T) {
	repoRoot, _ := setupStore(t)
	run := "rdg-2609250000000001"
	captureTranscript(t, repoRoot, "sess-reading", mustStamp(t, sessionkind.Reading, run, sepDigestA))
	captureTranscript(t, repoRoot, "sess-scribe", mustStamp(t, sessionkind.Scribe, run, sepDigestB))

	rep, err := SessionSeparation(repoRoot, testRootSHA)
	if err != nil {
		t.Fatal(err)
	}
	if rep.Unobserved || len(rep.Violations) != 0 {
		t.Fatalf("report = %+v, want the property held with no violation", rep)
	}
	if rep.Transcripts != 2 || rep.Stamped != 2 {
		t.Fatalf("report counts %d transcripts, %d stamped; want 2 and 2", rep.Transcripts, rep.Stamped)
	}
	if !reflect.DeepEqual(rep.Runs, []string{run}) {
		t.Fatalf("runs = %q, want [%s]", rep.Runs, run)
	}
	if got := rep.Summary(); !strings.Contains(got, "no retained transcript carries two stamps of one run") ||
		!strings.Contains(got, run) {
		t.Fatalf("summary %q does not say the property held for what it saw", got)
	}
}

// TestSeparationReportsAnEmptyStoreAsUnobserved is ac-7, and its sibling: a
// store holding transcripts none of which carries a stamp is unobserved too. A
// check that saw nothing and a check that could see nothing must not produce the
// same artefact as a clean one (adr-56).
func TestSeparationReportsAnEmptyStoreAsUnobserved(t *testing.T) {
	repoRoot, _ := setupStore(t)
	rep, err := SessionSeparation(repoRoot, testRootSHA)
	if err != nil {
		t.Fatal(err)
	}
	if !rep.Unobserved || rep.Reason == "" {
		t.Fatalf("an empty store reported %+v, want unobserved with a reason", rep)
	}
	if strings.Contains(rep.Summary(), "no retained transcript carries") {
		t.Fatalf("an empty store's summary %q reads as clean", rep.Summary())
	}

	captureTranscript(t, repoRoot, "sess-plain", "assistant: nothing stamped here")
	rep, err = SessionSeparation(repoRoot, testRootSHA)
	if err != nil {
		t.Fatal(err)
	}
	if !rep.Unobserved || rep.Transcripts != 1 || rep.Stamped != 0 {
		t.Fatalf("a store with no stamped transcript reported %+v, want unobserved", rep)
	}
}

// TestSeparationReadsMetadataOnly holds the check inside the consumer brief
// invariant 15 enumerates — session-separation evidence, metadata only, never
// bodies. Every record's body is rewritten to carry a violation the metadata
// does not, and the report does not move.
func TestSeparationReadsMetadataOnly(t *testing.T) {
	repoRoot, _ := setupStore(t)
	run := "rdg-2609250000000001"
	captureTranscript(t, repoRoot, "sess-reading", mustStamp(t, sessionkind.Reading, run, sepDigestA))
	captureTranscript(t, repoRoot, "sess-scribe", mustStamp(t, sessionkind.Scribe, run, sepDigestB))
	before, err := SessionSeparation(repoRoot, testRootSHA)
	if err != nil {
		t.Fatal(err)
	}

	records, err := List(repoRoot, testRootSHA)
	if err != nil {
		t.Fatal(err)
	}
	planted := mustStamp(t, sessionkind.Reading, "rdg-9", sepDigestA) + "\n" +
		mustStamp(t, sessionkind.Scribe, "rdg-9", sepDigestB) + "\n"
	for _, r := range records {
		if err := os.WriteFile(r.Path, marshalRecord(r, planted), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	after, err := SessionSeparation(repoRoot, testRootSHA)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(before, after) {
		t.Fatalf("the report moved when only the bodies did:\nbefore %+v\nafter  %+v", before, after)
	}
}
