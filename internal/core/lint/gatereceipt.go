package lint

import (
	"os"

	"github.com/intentdriven/abcd/internal/fsutil"
)

// gatereceipt.go — the receipt gate's reader, offered to a gate other than the
// release (itd-60, ruling DR3). The doc-fidelity verdict is a saved receipt
// labelled with the commit its reviewer read, found automatically by `spec
// close` and `launch ship` the way the pre-push hook finds a preflight receipt.
// It is judged by checkReceiptGate itself, so a doc-fidelity receipt and a
// release receipt can never be read by two readers that disagree: the same
// subject match, detector binding, pinned judge, manifest hash and tier.

// GateReceipt is what CheckGateReceipt read for one gate at one commit.
type GateReceipt struct {
	// Problems is every reason the receipt gate refuses the receipt; empty is
	// a PROMOTE the gate accepts.
	Problems []Finding
	// Parsed reports that a receipt naming the commit was read and parsed, so
	// Verdict and Failing are the receipt's own.
	Parsed  bool
	Verdict string
	Failing []ReceiptFinding
}

// CheckGateReceipt runs the receipt gate for one gate against commit, reading
// receipts under receiptsDir (repo-relative, laid out <dir>/<commit>/<gate>.json).
func CheckGateReceipt(repoRoot, receiptsDir, commit, gate string) (GateReceipt, error) {
	var out GateReceipt
	rc := RuleConfig{
		Enabled: true, Severity: severityBlocker, ReceiptsDir: receiptsDir,
		Commit: commit, RequiredGates: []string{gate},
		onReceipt: func(_ string, verdict string, failing []ReceiptFinding) {
			out.Parsed, out.Verdict, out.Failing = true, verdict, failing
		},
	}
	findings, err := checkReceiptGate(repoRoot, rc)
	if err != nil {
		return GateReceipt{}, err
	}
	out.Problems = findings
	return out, nil
}

// ReleaseGateManifestHash is the hash a manifest-era receipt must echo: sha256
// over the committed release-gate manifest's bytes, "" when the repository has
// no manifest (the pre-manifest era, where no receipt carries one).
func ReleaseGateManifestHash(repoRoot string) (string, error) {
	root, err := os.OpenRoot(repoRoot)
	if err != nil {
		return "", err
	}
	defer root.Close()
	// The receipt gate's own guarded read: a FIFO or a link at the manifest's
	// path is refused, never followed or waited on.
	data, err := fsutil.ReadGuardedInRoot(root, releaseGateManifestPath, maxReceiptBytes)
	if os.IsNotExist(err) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	return hashManifest(data), nil
}
