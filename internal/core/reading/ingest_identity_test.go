package reading

// ingest_identity_test.go is itd-185's ac-3 and ac-13, plus the instrument
// identity ruling (12) rests on.

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// mintedItemID is the shape adr-45 mints: a family tag, a UTC stamp and four
// uniform digits. The verb does not mint it itself — capture.IngestReading does,
// under the ledger lock, where the collision probe can see the tree — so what is
// asserted here is that the id came from a mint and not from the payload.
var mintedItemID = regexp.MustCompile(`^rdi-[0-9]{16}$`)

// TestManifestReferenceMustResolve is the first half of ac-3: a reference that
// resolves to nothing refuses the run.
func TestManifestReferenceMustResolve(t *testing.T) {
	f := newIngestFixture(t, "detection")
	doc := f.payload(1)
	doc["run_id"] = "rdg-2608310000009999"

	_, err := f.ingest(doc)
	if err == nil {
		t.Fatal("an output citing a run nobody parked was accepted")
	}
	if !strings.Contains(err.Error(), "rdg-2608310000009999") {
		t.Errorf("the refusal does not name the unresolvable run: %v", err)
	}
	f.nothingDurable("rdg-2608310000009999")
	f.mustIngest(f.payload(1))
}

// TestManifestHashMismatchRefusesRun is the second half of ac-3: the stored hash
// must equal the content hash of the manifest itself, and a disagreement refuses.
//
// The mismatch is made by editing the PARKED manifest, not by editing the
// citation: that is the direction the check exists for. A manifest swapped after
// the reading ran is exactly the contamination the reference is unforgeable
// against, and a test that only mistyped the hash would pass against a verb that
// merely compared the string to itself.
func TestManifestHashMismatchRefusesRun(t *testing.T) {
	f := newIngestFixture(t, "detection")
	legal := f.payload(1)

	parked := filepath.Join(f.root, filepath.FromSlash(
		DefaultRunDir+"/"+f.runID+"/"+ManifestFileName))
	raw, err := os.ReadFile(parked)
	if err != nil {
		t.Fatal(err)
	}
	swapped := strings.Replace(string(raw), "0123456789abcdef0123456789abcdef01234567",
		"fedcba9876543210fedcba9876543210fedcba98", 1)
	if swapped == string(raw) {
		t.Fatal("the parked manifest was not altered, so this case would prove nothing")
	}
	if err := os.WriteFile(parked, []byte(swapped), 0o644); err != nil {
		t.Fatal(err)
	}

	_, err = f.ingest(legal)
	if err == nil {
		t.Fatal("an output citing the hash of a manifest that has since changed was accepted")
	}
	if !strings.Contains(err.Error(), f.manifestHash) {
		t.Errorf("the refusal does not name the citation it refused: %v", err)
	}
	f.nothingDurable(f.runID)

	// Restore, and the same payload is accepted: the check is about the bytes,
	// not about the run.
	if err := os.WriteFile(parked, raw, 0o644); err != nil {
		t.Fatal(err)
	}
	f.mustIngest(legal)
}

// TestInstrumentIdentityRequiresAllThreeParts is ruling (12): model, definition
// hash and assembler version, all three required and all three checked against
// something that can disagree with them.
func TestInstrumentIdentityRequiresAllThreeParts(t *testing.T) {
	f := newIngestFixture(t, "detection")

	for _, part := range []string{"model", "definition_sha256", "assembler_version"} {
		t.Run("missing "+part, func(t *testing.T) {
			doc := f.nextRun(f.payload(1))
			doc["instrument"].(map[string]any)[part] = ""
			if _, err := f.ingest(doc); err == nil {
				t.Fatalf("an instrument missing %s was accepted", part)
			}
		})
		// A part that renders as nothing is missing too. The hash and the
		// version are compared exactly further on, so only the model could
		// pass on an invisible rune — but "all three present" is one check,
		// and it judges blankness the way every other check in this verb does
		// (iss-2608311518250688).
		t.Run("invisible "+part, func(t *testing.T) {
			doc := f.nextRun(f.payload(1))
			doc["instrument"].(map[string]any)[part] = "\u2800\u200b"
			if _, err := f.ingest(doc); err == nil {
				t.Fatalf("an instrument whose %s renders as nothing was accepted", part)
			}
		})
	}

	t.Run("definition hash is recomputed", func(t *testing.T) {
		doc := f.nextRun(f.payload(1))
		doc["instrument"].(map[string]any)["definition_sha256"] = sha256Hex([]byte("another definition"))
		_, err := f.ingest(doc)
		if err == nil {
			t.Fatal("an instrument claiming a definition hash the file does not have was accepted")
		}
		if !strings.Contains(err.Error(), f.definitionSHA) {
			t.Errorf("the refusal does not name the hash the definition actually has: %v", err)
		}
	})

	t.Run("assembler version is the manifest's", func(t *testing.T) {
		doc := f.nextRun(f.payload(1))
		doc["instrument"].(map[string]any)["assembler_version"] = "0.0.0-not-this-one"
		_, err := f.ingest(doc)
		if err == nil {
			t.Fatal("an instrument claiming an assembler version the manifest does not carry was accepted")
		}
		if !strings.Contains(err.Error(), AssemblerVersion()) {
			t.Errorf("the refusal does not name the manifest's own assembler version: %v", err)
		}
	})

	f.mustIngest(f.nextRun(f.payload(1)))
}

// TestItemIDsAreMintedByTheVerb is ac-13, and it is two claims.
//
// The ids on an accepted run were minted, not carried: every record id has the
// mint's shape and none appears in the payload. And a payload that supplies its
// own item identifier is refused as an unknown field — the payload schema
// carries no item identifier at any level, so an id has nowhere to go.
func TestItemIDsAreMintedByTheVerb(t *testing.T) {
	f := newIngestFixture(t, "detection")
	res := f.mustIngest(f.payload(3))

	if len(res.Records) != 3 {
		t.Fatalf("landed %d of 3 records", len(res.Records))
	}
	seen := map[string]bool{}
	for _, r := range res.Records {
		if !mintedItemID.MatchString(r.ID) {
			t.Errorf("record id %q is not a minted id", r.ID)
		}
		if seen[r.ID] {
			t.Errorf("record id %q was minted twice in one run", r.ID)
		}
		seen[r.ID] = true
	}

	for _, field := range []string{"id", "rdi", "item_id"} {
		doc := f.nextRun(f.payload(1))
		doc["items"].([]any)[0].(map[string]any)[field] = "rdi-2608310000000042"
		_, err := f.ingest(doc)
		if err == nil {
			t.Fatalf("a payload supplying its own item identifier in %q was accepted", field)
		}
		if !strings.Contains(err.Error(), field) {
			t.Errorf("the refusal does not name the field %q: %v", field, err)
		}
	}
}
