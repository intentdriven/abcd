package reading

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// refusalLeak is a payload value no refusal may carry back: a marker and a
// third party's absolute home path, the shape a token or a path pasted into the
// wrong field takes.
const refusalLeak = "zzleak-7f3a /Users/zzotherperson/notes" // abcd-lint:allow — a planted home path the refusal must not echo

// assertNoRefusalLeak fails when a refusal carries any part of refusalLeak, or
// no longer names the field it refuses.
func assertNoRefusalLeak(t *testing.T, err error, field string) {
	t.Helper()
	if err == nil {
		t.Fatalf("a payload carrying the leak in %s was accepted", field)
	}
	for _, part := range []string{"zzleak-7f3a", "zzotherperson"} {
		if strings.Contains(err.Error(), part) {
			t.Errorf("the refusal echoes the refused %s: %v", field, err)
		}
	}
	if !strings.Contains(err.Error(), field) {
		t.Errorf("the refusal no longer names %s: %v", field, err)
	}
}

// TestEnvelopeRefusalsDoNotEchoThePayload — iss-2609290043245353. The envelope
// checks run before the run's identity is proven, and each returned the
// payload's own value through echo(), which cleans and caps but does not redact,
// so a token or a home path in a closed-shape field reached the terminal and the
// transcript. A closed-shape value is described now; the one refusal whose value
// the reader needs to find the fault — an undeclared field's NAME — is redacted
// through the canonical scanner and still named.
func TestEnvelopeRefusalsDoNotEchoThePayload(t *testing.T) {
	for _, field := range []string{"_type", "run_id", "position", "manifest_sha256"} {
		t.Run(field, func(t *testing.T) {
			f := newIngestFixture(t, "detection")
			doc := f.payload(1)
			doc[field] = refusalLeak
			_, err := f.ingest(doc)
			assertNoRefusalLeak(t, err, field)
			f.nothingDurable(f.runID)
		})
	}

	t.Run("undeclared envelope field", func(t *testing.T) {
		f := newIngestFixture(t, "detection")
		doc := f.payload(1)
		doc["reviewer_notes /Users/zzotherperson/notes"] = "x" // abcd-lint:allow — a planted home path in a KEY
		_, err := f.ingest(doc)
		if err == nil {
			t.Fatal("an undeclared envelope field was accepted")
		}
		if strings.Contains(err.Error(), "zzotherperson") {
			t.Errorf("the refusal echoes the home path in the undeclared field's name: %v", err)
		}
		if !strings.Contains(err.Error(), "reviewer_notes") {
			t.Errorf("the refusal no longer names the undeclared field: %v", err)
		}
	})
}

// TestWriteRunArtefactDoesNotEchoARefusedRunID — the same class at the run
// artefact writer: a run id that is not one is described, never quoted.
func TestWriteRunArtefactDoesNotEchoARefusedRunID(t *testing.T) {
	_, err := WriteRunArtefact(t.TempDir(), refusalLeak, "scribe-manifest.json", map[string]string{})
	assertNoRefusalLeak(t, err, "run")
}

// TestUndeclaredFieldRefusalFailsClosedOnADegradedScanner — the refusal's
// redactor must consult the scanner's degraded state, as every write-time
// redactor does (iss-2609290043245353). A per-repo scanner config that does not
// parse leaves the scanner degraded; the undeclared field's name is then
// described, never echoed through a weakened pattern set.
func TestUndeclaredFieldRefusalFailsClosedOnADegradedScanner(t *testing.T) {
	f := newIngestFixture(t, "detection")
	cfg := filepath.Join(f.root, ".abcd", "config", "pii.json")
	if err := os.MkdirAll(filepath.Dir(cfg), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(cfg, []byte("{not json"), 0o644); err != nil {
		t.Fatal(err)
	}
	doc := f.payload(1)
	doc["reviewer_notes /Users/zzotherperson/notes"] = "x" // abcd-lint:allow — a planted home path in a KEY
	_, err := f.ingest(doc)
	if err == nil {
		t.Fatal("an undeclared envelope field was accepted")
	}
	for _, part := range []string{"zzotherperson", "reviewer_notes"} {
		if strings.Contains(err.Error(), part) {
			t.Errorf("a degraded scanner let the refusal echo the decoder's message (%q): %v", part, err)
		}
	}
	if !strings.Contains(err.Error(), "not quoted") {
		t.Errorf("a degraded scanner did not describe the message: %v", err)
	}
}
