package lint

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The iss-131 hardening of receipt_gate: the manifest and the receipts are read
// through the guarded primitive, a receipt carrying a duplicate JSON key is
// refused rather than read last-wins, and the committed example receipt's
// manifestHash is pinned to the committed manifest.

func receiptFixture(t *testing.T) (root string, cfg RuleConfig, put func(string)) {
	t.Helper()
	root = t.TempDir()
	const sha = "0123456789abcdef0123456789abcdef01234567"
	const gate = "iss35-brief-surface-crosscheck"
	reviews := filepath.Join(".abcd", "work", "reviews")
	manifestBody := `{"schemaVersion":1,"detector":"` + gate + `"}` + "\n"
	writeFile(t, root, releaseGateManifestPath, manifestBody)
	cfg = RuleConfig{Enabled: true, Severity: severityBlocker, ReceiptsDir: reviews, Commit: sha, RequiredGates: []string{gate}}
	put = func(body string) { writeFile(t, root, filepath.Join(reviews, sha, gate+".json"), body) }
	put(manifestReceipt(sha, gate, releaseTierFull, hashManifest([]byte(manifestBody)), `[]`))
	return root, cfg, put
}

func TestReceiptGateRefusesADuplicateKey(t *testing.T) {
	root, cfg, put := receiptFixture(t)
	if n := countRule(runReceiptGate(t, root, cfg), "receipt_gate"); n != 0 {
		t.Fatalf("the conforming fixture is not clean: %d", n)
	}
	// A second verificationResult: last-wins would read PROMOTE over a REJECT.
	put(strings.Replace(manifestReceipt("0123456789abcdef0123456789abcdef01234567", "iss35-brief-surface-crosscheck",
		releaseTierFull, "x", `[]`), `"verificationResult": "PROMOTE",`, `"verificationResult": "REJECT", "verificationResult": "PROMOTE",`, 1))
	fs := runReceiptGate(t, root, cfg)
	if !findingWith(fs, filepath.Join(".abcd", "work", "reviews", "0123456789abcdef0123456789abcdef01234567", "iss35-brief-surface-crosscheck.json"),
		"receipt_gate", "duplicate key") {
		t.Fatalf("a receipt with a duplicate key was not refused by name: %+v", fs)
	}
}

func TestReceiptGateReadsTheManifestGuarded(t *testing.T) {
	root, cfg, _ := receiptFixture(t)
	// The manifest replaced by a symlink: the guarded read refuses the leaf, and
	// the gate fails closed rather than following it.
	manifest := filepath.Join(root, releaseGateManifestPath)
	real := manifest + ".real"
	if err := os.Rename(manifest, real); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(real, manifest); err != nil {
		t.Fatal(err)
	}
	fs := runReceiptGate(t, root, cfg)
	if !findingWith(fs, filepath.Join(".abcd", "work", "reviews"), "receipt_gate", "cannot read the release-gate manifest") {
		t.Fatalf("a symlinked manifest was followed rather than refused: %+v", fs)
	}
}

func TestReceiptExampleManifestHashIsTheCommittedManifests(t *testing.T) {
	repo := filepath.Join("..", "..", "..")
	manifest, err := os.ReadFile(filepath.Join(repo, releaseGateManifestPath))
	if err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(repo, filepath.Dir(releaseGateManifestPath), "receipt.example.json"))
	if err != nil {
		t.Fatal(err)
	}
	var ex struct {
		ManifestHash string `json:"manifestHash"`
	}
	if err := json.Unmarshal(data, &ex); err != nil {
		t.Fatal(err)
	}
	if want := hashManifest(manifest); ex.ManifestHash != want {
		t.Fatalf("receipt.example.json manifestHash is %s, the committed manifest hashes to %s; update the example with the manifest", ex.ManifestHash, want)
	}
}
