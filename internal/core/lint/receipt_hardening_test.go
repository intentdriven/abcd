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

// TestReceiptGateRefusesACaseTwinOfAKey: encoding/json binds a struct field
// case-insensitively, last-wins, so "VerificationResult" after
// "verificationResult" is the same field read twice: the reviewer sees REJECT and
// the gate would read PROMOTE (iss-2609252251311346).
func TestReceiptGateRefusesACaseTwinOfAKey(t *testing.T) {
	root, cfg, put := receiptFixture(t)
	const sha, gate = "0123456789abcdef0123456789abcdef01234567", "iss35-brief-surface-crosscheck"
	for _, twin := range []string{"VerificationResult", "VERIFICATIONRESULT", `\u0056erificationResult`} {
		put(strings.Replace(manifestReceipt(sha, gate, releaseTierFull, "x", `[]`),
			`"verificationResult": "PROMOTE",`, `"verificationResult": "REJECT", "`+twin+`": "PROMOTE",`, 1))
		fs := runReceiptGate(t, root, cfg)
		if !findingWith(fs, filepath.Join(".abcd", "work", "reviews", sha, gate+".json"), "receipt_gate", "duplicate key") {
			t.Fatalf("a receipt carrying %q beside verificationResult was not refused as a duplicate key: %+v", twin, fs)
		}
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

// receiptExampleManifestHash is the value receipt.example.json carries for
// manifestHash. It is a placeholder rather than a literal hash: the manifest
// changes with every surface a branch adds, so a literal copy of its sha256 was
// rewritten on the same line by nearly every such branch and made any two of
// them conflict (iss-2610150526192138). A placeholder never goes stale, and it
// can never pass for a real hash, so a receipt copied from the example without
// computing the hash is refused.
const receiptExampleManifestHash = "sha256:<computed: the sha256 of manifest.json at subject.digest.gitCommit>"

// TestReceiptExampleManifestHashIsAPlaceholder holds the example to the
// placeholder, and holds receipt_gate to refusing it: the example dressed as a
// real receipt for the real manifest is refused on manifestHash, and the same
// receipt with the computed hash in its place is not. The second half is the
// control that pins the refusal on the placeholder alone.
func TestReceiptExampleManifestHashIsAPlaceholder(t *testing.T) {
	repo := filepath.Join("..", "..", "..")
	manifest, err := os.ReadFile(filepath.Join(repo, releaseGateManifestPath))
	if err != nil {
		t.Fatal(err)
	}
	example, err := os.ReadFile(filepath.Join(repo, filepath.Dir(releaseGateManifestPath), "receipt.example.json"))
	if err != nil {
		t.Fatal(err)
	}
	var ex struct {
		Subject struct {
			Digest struct {
				GitCommit string `json:"gitCommit"`
			} `json:"digest"`
		} `json:"subject"`
		ManifestHash string `json:"manifestHash"`
		Policy       struct {
			Detector string `json:"detector"`
		} `json:"policy"`
	}
	if err := json.Unmarshal(example, &ex); err != nil {
		t.Fatal(err)
	}
	if ex.ManifestHash != receiptExampleManifestHash {
		t.Fatalf("receipt.example.json manifestHash is %q; it carries the placeholder %q, never a literal hash, "+
			"because a literal is rewritten by every branch that changes the manifest", ex.ManifestHash, receiptExampleManifestHash)
	}
	if ex.ManifestHash == hashManifest(manifest) {
		t.Fatal("the placeholder equals the manifest's hash")
	}

	root := t.TempDir()
	sha, gate := ex.Subject.Digest.GitCommit, ex.Policy.Detector
	reviews := filepath.Join(".abcd", "work", "reviews")
	rel := filepath.Join(reviews, sha, gate+".json")
	writeFile(t, root, releaseGateManifestPath, string(manifest))
	cfg := RuleConfig{Enabled: true, Severity: severityBlocker, ReceiptsDir: reviews, Commit: sha, RequiredGates: []string{gate}}

	writeFile(t, root, rel, string(example))
	if !findingWith(runReceiptGate(t, root, cfg), rel, "receipt_gate", "manifestHash") {
		t.Fatal("the example receipt, copied verbatim, was not refused on its placeholder manifestHash")
	}
	computed := strings.Replace(string(example), receiptExampleManifestHash, hashManifest(manifest), 1)
	writeFile(t, root, rel, computed)
	if findingWith(runReceiptGate(t, root, cfg), rel, "receipt_gate", "manifestHash") {
		t.Fatal("the example with the computed hash in place of the placeholder was still refused on manifestHash")
	}
}
