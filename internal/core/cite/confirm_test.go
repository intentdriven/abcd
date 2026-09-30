package cite

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/lint"
)

// TestConfirmWritesADatedManualReceipt is AC 4's floor: a human clears one queue
// line and the baseline records THAT they confirmed it and WHEN.
func TestConfirmWritesADatedManualReceipt(t *testing.T) {
	root := citeRepo(t, "https://paywalled.example.org/x")

	res, err := Confirm(ConfirmRequest{
		RepoRoot: root, Config: testConfig(), Now: now,
		Receipt: Receipt{
			SchemaVersion: ReceiptSchemaVersion,
			Confirmed:     []ConfirmedCitation{{URL: "https://paywalled.example.org/x"}},
		},
	})
	if err != nil {
		t.Fatalf("Confirm: %v", err)
	}
	if len(res.Recorded) != 1 || res.Recorded[0] != "https://paywalled.example.org/x" {
		t.Fatalf("recorded = %v, want the confirmed URL", res.Recorded)
	}

	got := loadBaseline(t, root).Entries["https://paywalled.example.org/x"]
	want := lint.BaselineEntry{
		URL:          "https://paywalled.example.org/x",
		FinalURL:     "https://paywalled.example.org/x",
		LastChecked:  today,
		Outcome:      lint.OutcomeAlive,
		Verification: lint.VerificationManual,
		VerifiedOn:   today,
	}
	if got != want {
		t.Fatalf("entry = %+v, want %+v", got, want)
	}
}

// TestConfirmRecordsADeclaredFinalAddress lets a human report a redirect they
// followed, so a manually-cleared citation carries the same drift information an
// automatic one does.
func TestConfirmRecordsADeclaredFinalAddress(t *testing.T) {
	root := citeRepo(t, "https://paywalled.example.org/x")
	_, err := Confirm(ConfirmRequest{
		RepoRoot: root, Config: testConfig(), Now: now,
		Receipt: Receipt{Confirmed: []ConfirmedCitation{{
			URL: "https://paywalled.example.org/x", FinalURL: "https://paywalled.example.org/x-v2",
			VerifiedOn: "2026-07-20",
		}}},
	})
	if err != nil {
		t.Fatalf("Confirm: %v", err)
	}
	got := loadBaseline(t, root).Entries["https://paywalled.example.org/x"]
	if got.FinalURL != "https://paywalled.example.org/x-v2" {
		t.Errorf("final_url = %q, want the declared address", got.FinalURL)
	}
	// The declared date is the staleness clock: a receipt for work done last week
	// must age from last week, not from whenever it was typed in.
	if got.VerifiedOn != "2026-07-20" || got.LastChecked != "2026-07-20" {
		t.Errorf("dates = %q/%q, want the declared 2026-07-20", got.VerifiedOn, got.LastChecked)
	}
}

// TestConfirmRefusesAnUncitedURL is the anti-forgery clause. Confirming is the
// one place a receipt enters the record without a fetch, so it must not be a
// door through which arbitrary entries arrive — only what the docs actually cite
// can be confirmed.
func TestConfirmRefusesAnUncitedURL(t *testing.T) {
	root := citeRepo(t, "https://example.org/cited")
	_, err := Confirm(ConfirmRequest{
		RepoRoot: root, Config: testConfig(), Now: now,
		Receipt: Receipt{Confirmed: []ConfirmedCitation{{URL: "https://elsewhere.example.org/never-cited"}}},
	})
	if err == nil {
		t.Fatal("Confirm accepted a URL the docs do not cite")
	}
	if !strings.Contains(err.Error(), "not cited") {
		t.Errorf("error = %q, want it to say the URL is not cited", err)
	}
	// Nothing is written when any line is refused: a receipt is all-or-nothing,
	// so a batch with one bad line cannot half-apply.
	if _, statErr := os.Stat(filepath.Join(root, filepath.FromSlash(lint.DefaultBaselinePath))); statErr == nil {
		t.Error("a refused confirm still wrote a baseline")
	}
}

// TestConfirmRefusesAFutureDate stops a receipt from buying itself an extra year
// of freshness by declaring tomorrow.
func TestConfirmRefusesAFutureDate(t *testing.T) {
	root := citeRepo(t, "https://example.org/cited")
	_, err := Confirm(ConfirmRequest{
		RepoRoot: root, Config: testConfig(), Now: now,
		Receipt: Receipt{Confirmed: []ConfirmedCitation{{URL: "https://example.org/cited", VerifiedOn: "2027-01-01"}}},
	})
	if err == nil {
		t.Fatal("Confirm accepted a receipt dated in the future")
	}
}

// TestConfirmPreservesOtherEntries pins that clearing one queue line is a
// surgical edit, not a rewrite: everything else in the baseline survives.
func TestConfirmPreservesOtherEntries(t *testing.T) {
	root := citeRepo(t, "https://example.org/a", "https://paywalled.example.org/x")
	existing := lint.BaselineEntry{
		FinalURL: "https://example.org/a", LastChecked: "2026-07-01",
		Outcome: lint.OutcomeAlive, Verification: lint.VerificationAutomatic, VerifiedOn: "2026-07-01",
	}
	writeBaseline(t, root, map[string]lint.BaselineEntry{"https://example.org/a": existing})

	if _, err := Confirm(ConfirmRequest{
		RepoRoot: root, Config: testConfig(), Now: now,
		Receipt: Receipt{Confirmed: []ConfirmedCitation{{URL: "https://paywalled.example.org/x"}}},
	}); err != nil {
		t.Fatalf("Confirm: %v", err)
	}
	entries := loadBaseline(t, root).Entries
	if len(entries) != 2 {
		t.Fatalf("baseline has %d entries, want 2", len(entries))
	}
	got := entries["https://example.org/a"]
	got.URL = ""
	if got != existing {
		t.Errorf("untouched entry = %+v, want %+v", got, existing)
	}
}

// TestParseReceiptRoundTrip pins the ONE receipt schema both rungs use: the verb
// accepts, verbatim, what the later generated checklist page will hand back.
func TestParseReceiptRoundTrip(t *testing.T) {
	raw := []byte(`{
  "schema_version": 1,
  "confirmed": [
    {"url": "https://a.example.org/x", "final_url": "https://a.example.org/x2", "verified_on": "2026-07-20"},
    {"url": "https://b.example.org/y"}
  ]
}`)
	got, err := parseReceipt(raw)
	if err != nil {
		t.Fatalf("ParseReceipt: %v", err)
	}
	want := Receipt{SchemaVersion: 1, Confirmed: []ConfirmedCitation{
		{URL: "https://a.example.org/x", FinalURL: "https://a.example.org/x2", VerifiedOn: "2026-07-20"},
		{URL: "https://b.example.org/y"},
	}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %+v, want %+v", got, want)
	}
}

// TestParseReceiptRefusesAMethodField carries the schema's central refusal
// through to the ingest path: a receipt is structurally incapable of recording
// HOW a human verified, so a producer that tries is refused at the door rather
// than having the key quietly dropped.
func TestParseReceiptRefusesAMethodField(t *testing.T) {
	raw := []byte(`{"schema_version": 1, "confirmed": [{"url": "https://a.example.org/x", "method": "logged in via the library proxy"}]}`)
	if _, err := parseReceipt(raw); err == nil {
		t.Fatal("ParseReceipt accepted a method field")
	}
}

// TestParseReceiptRefusesAnUnknownSchemaVersion keeps a future producer's record
// from being best-effort parsed by a build that does not understand it.
func TestParseReceiptRefusesAnUnknownSchemaVersion(t *testing.T) {
	if _, err := parseReceipt([]byte(`{"schema_version": 99, "confirmed": []}`)); err == nil {
		t.Fatal("ParseReceipt accepted an unknown schema version")
	}
}

// TestReceiptHasNoMethodFieldByConstruction is the type-level half of the same
// refusal, mirroring the baseline's own construction test.
func TestReceiptHasNoMethodFieldByConstruction(t *testing.T) {
	banned := map[string]bool{
		"method": true, "how": true, "transcript": true, "notes": true,
		"credential": true, "evidence": true, "proof": true,
	}
	typ := reflect.TypeOf(ConfirmedCitation{})
	for i := 0; i < typ.NumField(); i++ {
		tag, _, _ := strings.Cut(typ.Field(i).Tag.Get("json"), ",")
		if banned[tag] {
			t.Errorf("ConfirmedCitation declares %q; a receipt records THAT and WHEN, never how", tag)
		}
	}
}
