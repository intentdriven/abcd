package capture

import (
	"os"
	"reflect"
	"testing"

	"github.com/intentdriven/abcd/internal/core/frontmatter"
)

func TestBuildIssueTextRoundTrip(t *testing.T) {
	tests := []struct {
		name   string
		fields []kv
		body   string
		want   string
	}{
		{
			name: "scalars only",
			fields: []kv{
				{"schema_version", 1},
				{"id", "iss-7"},
				{"slug", "broken-thing"},
				{"found_during", "manual smoke"},
			},
			body: "The body.\n",
			want: "---\nschema_version: 1\nid: \"iss-7\"\nslug: \"broken-thing\"\nfound_during: \"manual smoke\"\n---\n\nThe body.\n",
		},
		{
			name:   "empty list emits bracket-pair",
			fields: []kv{{"id", "iss-1"}, {"related_intents", []string{}}},
			body:   "",
			want:   "---\nid: \"iss-1\"\nrelated_intents: []\n---\n\n",
		},
		{
			name:   "abcd id list is unquoted inline",
			fields: []kv{{"related_intents", []string{"itd-4", "fn-12", "iss-3"}}},
			body:   "b",
			want:   "---\nrelated_intents: [itd-4, fn-12, iss-3]\n---\n\nb\n",
		},
		{
			name:   "non-id list is per-item quoted",
			fields: []kv{{"synthesis_clusters", []string{"cluster a", "cluster b"}}},
			body:   "b",
			want:   "---\nsynthesis_clusters: [\"cluster a\", \"cluster b\"]\n---\n\nb\n",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := buildIssueText(tc.fields, tc.body)
			if err != nil {
				t.Fatalf("buildIssueText: %v", err)
			}
			if got != tc.want {
				t.Fatalf("got:\n%q\nwant:\n%q", got, tc.want)
			}
		})
	}
}

// TestBuildIssueTextTrailingNewline pins the on-disk contract: a serialised
// record ends with exactly one trailing newline, whether or not the caller's body
// carried one (iss-175 — 170 of 174 ledger files lacked an EOF newline because the
// writer appended the body verbatim). A body already ending in a newline is not
// doubled; an empty body keeps the single blank-line separator after the closing
// delimiter.
func TestBuildIssueTextTrailingNewline(t *testing.T) {
	fields := []kv{{"id", "iss-1"}}
	tests := []struct {
		name string
		body string
		want string
	}{
		{"no trailing newline gets exactly one", "body text", "---\nid: \"iss-1\"\n---\n\nbody text\n"},
		{"already-terminated body is not doubled", "body text\n", "---\nid: \"iss-1\"\n---\n\nbody text\n"},
		{"multi-trailing-newline body collapses to one", "body text\n\n\n", "---\nid: \"iss-1\"\n---\n\nbody text\n"},
		{"empty body keeps the separator only", "", "---\nid: \"iss-1\"\n---\n\n"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := buildIssueText(fields, tc.body)
			if err != nil {
				t.Fatalf("buildIssueText: %v", err)
			}
			if got != tc.want {
				t.Fatalf("got:\n%q\nwant:\n%q", got, tc.want)
			}
		})
	}
}

func TestParseFrontmatterAndBody(t *testing.T) {
	text := "---\nschema_version: 1\nid: \"iss-7\"\nrelated_intents: [itd-4, itd-9]\nsynthesis_clusters: [\"a b\"]\n---\n\nHello body\nsecond line\n"
	fm, body, err := parseFrontmatterAndBody(text)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if fm["schema_version"] != 1 {
		t.Errorf("schema_version = %v (%T), want int 1", fm["schema_version"], fm["schema_version"])
	}
	if fm["id"] != "iss-7" {
		t.Errorf("id = %v, want iss-7", fm["id"])
	}
	if !reflect.DeepEqual(fm["related_intents"], []string{"itd-4", "itd-9"}) {
		t.Errorf("related_intents = %#v", fm["related_intents"])
	}
	if !reflect.DeepEqual(fm["synthesis_clusters"], []string{"a b"}) {
		t.Errorf("synthesis_clusters = %#v", fm["synthesis_clusters"])
	}
	if body != "Hello body\nsecond line\n" {
		t.Errorf("body = %q", body)
	}
}

func TestParseRejectsMissingOpener(t *testing.T) {
	if _, _, err := parseFrontmatterAndBody("no frontmatter here"); err == nil {
		t.Fatal("expected error for missing opening ---")
	}
}

func TestYamlScalarRejectsControlChar(t *testing.T) {
	if _, err := yamlScalar("bad\nvalue"); err == nil {
		t.Fatal("expected control-char rejection")
	}
}

func TestYamlScalarEscaping(t *testing.T) {
	got, err := yamlScalar(`he said "hi" \ end`)
	if err != nil {
		t.Fatal(err)
	}
	want := `"he said \"hi\" \\ end"`
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
	// Round-trips through the shared decoder this package's reader uses.
	if back := frontmatter.Unquote(got[1 : len(got)-1]); back != `he said "hi" \ end` {
		t.Fatalf("Unquote = %q", back)
	}
}

func TestSetScalarFieldReplaceAndInsert(t *testing.T) {
	content := "---\nid: \"iss-1\"\ncreated: \"2026-01-01\"\n---\n\nbody\n"
	// Insert a new field before the closing ---.
	got, err := setScalarField(content, "updated", "2026-02-02")
	if err != nil {
		t.Fatal(err)
	}
	want := "---\nid: \"iss-1\"\ncreated: \"2026-01-01\"\nupdated: \"2026-02-02\"\n---\n\nbody\n"
	if got != want {
		t.Fatalf("insert got:\n%q\nwant:\n%q", got, want)
	}
	// Replace an existing field in place.
	got2, err := setScalarField(got, "created", "2026-03-03")
	if err != nil {
		t.Fatal(err)
	}
	if want2 := "---\nid: \"iss-1\"\ncreated: \"2026-03-03\"\nupdated: \"2026-02-02\"\n---\n\nbody\n"; got2 != want2 {
		t.Fatalf("replace got:\n%q\nwant:\n%q", got2, want2)
	}
}

// TestSetMapField: the deterministic nested-map writer renders the members in
// the given order, quoted, indented; refuses a duplicate top-level key and an
// unsafe value.
func TestSetMapField(t *testing.T) {
	base := "---\nid: \"iss-1\"\n---\n\nbody\n"
	out, err := setMapField(base, "resolved_by", []kv{{"intent", "itd-7"}, {"commit", "abc1234"}})
	if err != nil {
		t.Fatalf("setMapField: %v", err)
	}
	want := "---\nid: \"iss-1\"\nresolved_by:\n  intent: \"itd-7\"\n  commit: \"abc1234\"\n---\n\nbody\n"
	if out != want {
		t.Fatalf("setMapField rendered:\n%q\nwant:\n%q", out, want)
	}
	// Round-trips through the parser as a nested object.
	fm, _, err := parseFrontmatterAndBody(out)
	if err != nil {
		t.Fatal(err)
	}
	m, ok := fm["resolved_by"].(map[string]any)
	if !ok || m["intent"] != "itd-7" || m["commit"] != "abc1234" {
		t.Fatalf("round-trip lost the object: %#v", fm["resolved_by"])
	}
	// Duplicate top-level key refused.
	if _, err := setMapField(out, "resolved_by", []kv{{"spec", "spc-1"}}); err == nil {
		t.Fatalf("setMapField must refuse a duplicate key")
	}
	// Control chars in a member value refused.
	if _, err := setMapField(base, "resolved_by", []kv{{"intent", "evil\nkey: injected"}}); err == nil {
		t.Fatalf("setMapField must refuse control chars")
	}
	// Non-scalar member refused.
	if _, err := setMapField(base, "resolved_by", []kv{{"intent", []string{"a"}}}); err == nil {
		t.Fatalf("setMapField must refuse non-scalar members")
	}
}

// TestLapsedAtRoundTrips pins the lapse timestamp end to end: the instant the
// discipline gave way is written, re-read and returned VERBATIM. The whole point
// of the property (spc-60) is that it is not write-up time, so a value the writer
// or reader quietly normalised — to the wall clock, to a truncated date, to a
// re-zoned instant — would defeat the field while every other check stayed green.
func TestLapsedAtRoundTrips(t *testing.T) {
	const lapsedAt = "2026-08-28T09:15:00Z"
	text, err := buildIssueText([]kv{
		{"schema_version", 1},
		{"id", "iss-1"},
		{"slug", "lapse-x"},
		{"severity", "minor"},
		{"category", "lapse"},
		{"source", "user-observation"},
		{"found_during", "preparation"},
		{"lapsed_at", lapsedAt},
	}, "a lapse\n")
	if err != nil {
		t.Fatalf("buildIssueText: %v", err)
	}
	fm, body, err := parseFrontmatterAndBody(text)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if err := validateStrict(fm); err != nil {
		t.Fatalf("a lapse record carrying lapsed_at was refused: %v", err)
	}
	iss := issueFromFrontmatter(fm, StateOpen, "open/iss-1-lapse-x.md", body)
	if iss.LapsedAt != lapsedAt {
		t.Fatalf("lapsed_at round-tripped as %q, want %q", iss.LapsedAt, lapsedAt)
	}
}

// TestCaptureReaderAcceptsProvenanceKeys pins the issue schema against the two
// disclosure keys, proved THROUGH the reader rather than asserted about the
// allow-list map. Without them in issueschema.Known the reader refuses every
// stamped record as carrying an unknown property and skips it — the record sits
// in the ledger, invisible to every capture surface, which is exactly how an
// earlier flag shipped unable to execute (spc-56).
func TestCaptureReaderAcceptsProvenanceKeys(t *testing.T) {
	text, err := buildIssueText([]kv{
		{"schema_version", 1},
		{"id", "iss-1"},
		{"slug", "x"},
		{"severity", "minor"},
		{"category", "observation"},
		{"source", "user-observation"},
		{"found_during", "review"},
		{"origin", rawScalar("researcher-authored")},
		{"production_mode", rawScalar("hand-written")},
	}, "a stamped record\n")
	if err != nil {
		t.Fatalf("buildIssueText: %v", err)
	}
	fm, _, err := parseFrontmatterAndBody(text)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if err := validateStrict(fm); err != nil {
		t.Fatalf("a stamped record was refused by the ledger reader: %v", err)
	}
}

// TestCommitCaptureStampsProvenance proves a captured issue carries both
// disclosure keys, written by the command rather than typed: the arrival path is
// derived from which verb ran, and the production mode is a closed choice
// defaulted when the caller declares none.
func TestCommitCaptureStampsProvenance(t *testing.T) {
	repo, ir := ledger(t)
	res, err := Capture(CaptureRequest{
		RepoRoot: repo, IssuesRoot: ir, Text: "a finding", Severity: SeverityMinor,
		Category: "bug", Source: "user-observation", FoundDuring: "t", Slug: "note",
		ProductionMode: "dictated-and-formatted",
	})
	if err != nil {
		t.Fatal(err)
	}
	fm := readLedgerFrontmatter(t, ir, res.ID)
	if fm["origin"] != "researcher-authored" {
		t.Errorf("origin = %v, want researcher-authored", fm["origin"])
	}
	if fm["production_mode"] != "dictated-and-formatted" {
		t.Errorf("production_mode = %v, want dictated-and-formatted", fm["production_mode"])
	}

	// An unset mode takes the default; both keys are present either way.
	res2, err := Capture(CaptureRequest{
		RepoRoot: repo, IssuesRoot: ir, Text: "another finding", Severity: SeverityMinor,
		Category: "bug", Source: "user-observation", FoundDuring: "t", Slug: "other",
	})
	if err != nil {
		t.Fatal(err)
	}
	fm = readLedgerFrontmatter(t, ir, res2.ID)
	if fm["origin"] != "researcher-authored" || fm["production_mode"] != "hand-written" {
		t.Errorf("defaulted stamp = %v/%v, want researcher-authored/hand-written", fm["origin"], fm["production_mode"])
	}

	// An out-of-vocabulary mode is refused before anything is reserved.
	if _, err := Capture(CaptureRequest{
		RepoRoot: repo, IssuesRoot: ir, Text: "a third finding", Severity: SeverityMinor,
		Category: "bug", Source: "user-observation", FoundDuring: "t", Slug: "third",
		ProductionMode: "typed",
	}); err == nil {
		t.Error("an out-of-vocabulary production mode must be refused")
	}
}

// readLedgerFrontmatter reads a committed ledger record back through the
// ledger's OWN reader, so a key the reader refuses cannot pass as written.
func readLedgerFrontmatter(t *testing.T, issuesRoot, id string) map[string]any {
	t.Helper()
	src, _, err := findIssue(issuesRoot, id)
	if err != nil {
		t.Fatalf("findIssue %s: %v", id, err)
	}
	data, err := os.ReadFile(src)
	if err != nil {
		t.Fatalf("reading %s: %v", src, err)
	}
	fm, _, err := parseFrontmatterAndBody(string(data))
	if err != nil {
		t.Fatalf("parsing %s: %v", src, err)
	}
	if err := validateStrict(fm); err != nil {
		t.Fatalf("the ledger reader refused the record it just wrote: %v", err)
	}
	return fm
}
