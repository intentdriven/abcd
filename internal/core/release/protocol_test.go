package release_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/release"
)

func writeWorkflow(t *testing.T, root, body string) {
	t.Helper()
	path := filepath.Join(root, ".github", "workflows", "release.yml")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

// TestReceiptsProtocolNamesTheGatesTheCommitAndTheTwoCommitShape is itd-93
// AC8: the emit step ends with the receipts protocol as a numbered checklist —
// the semantic gates to run (read from the release workflow, the list the
// release job enforces), the commit to key the receipts to, and the two-commit
// branch shape — so a first-time operator learns it from the verb.
func TestReceiptsProtocolNamesTheGatesTheCommitAndTheTwoCommitShape(t *testing.T) {
	root := t.TempDir()
	writeWorkflow(t, root, "jobs:\n  verify:\n    steps:\n      - run: |\n"+
		"          go run ./cmd/record-lint --release-gate \"$content\" \\\n"+
		"            --require-gate docs-currency-reviewer \\\n"+
		"            --require-gate iss35-brief-surface-crosscheck\n")
	p, err := release.ReceiptsProtocolFor(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(p.RequiredGates) != 2 {
		t.Fatalf("required gates = %v, want the workflow's two", p.RequiredGates)
	}
	all := strings.Join(p.Steps, "\n")
	for _, want := range []string{
		"docs-currency-reviewer", "iss35-brief-surface-crosscheck", // the gates to run
		"content commit", ".abcd/work/reviews/<content-sha>/<gate>.json", // the commit to key receipts to
		"exactly two commits",  // the branch shape
		"abcd launch receipts", // the local proof before merge
	} {
		if !strings.Contains(all, want) {
			t.Errorf("protocol must carry %q; steps:\n%s", want, all)
		}
	}
	if len(p.Steps) < 4 {
		t.Errorf("protocol is a checklist of at least four steps, got %d", len(p.Steps))
	}
}

// TestReceiptsProtocolWithNoGateRequiresNoReceipt is the AC3 degradation seen
// from the emit step: a workflow that arms no semantic gate gets a protocol
// that says no receipt is required, never one that asks for receipts nothing
// will read.
func TestReceiptsProtocolWithNoGateRequiresNoReceipt(t *testing.T) {
	root := t.TempDir()
	writeWorkflow(t, root, "jobs:\n  verify:\n    steps:\n      - run: go build ./...\n")
	p, err := release.ReceiptsProtocolFor(root)
	if err != nil {
		t.Fatal(err)
	}
	all := strings.Join(p.Steps, "\n")
	if len(p.RequiredGates) != 0 || !strings.Contains(all, "no receipt is required") {
		t.Errorf("an unarmed workflow requires no receipt; got gates %v, steps:\n%s", p.RequiredGates, all)
	}
	if strings.Contains(all, "two commits") {
		t.Errorf("with no receipt there is no second commit to describe; steps:\n%s", all)
	}

	// No release workflow at all names the scaffold that writes one.
	bare := t.TempDir()
	if p, err = release.ReceiptsProtocolFor(bare); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(strings.Join(p.Steps, "\n"), "abcd launch scaffold") {
		t.Errorf("a repository with no release workflow must be pointed at the scaffold; steps: %v", p.Steps)
	}
}

// TestReceiptsProtocolSaysTheReleasePublishesFromTheTaggedMerge is
// iss-2609251939476304: the content commit is the one every receipt names, not
// the one the release publishes from. The release publishes from the tagged
// merge, as the runbook's last step says, and the roll step must not tell the
// operator otherwise.
func TestReceiptsProtocolSaysTheReleasePublishesFromTheTaggedMerge(t *testing.T) {
	root := t.TempDir()
	writeWorkflow(t, root, "jobs:\n  verify:\n    steps:\n      - run: |\n"+
		"          go run ./cmd/record-lint --release-gate \"$content\" \\\n"+
		"            --require-gate docs-currency-reviewer\n")
	p, err := release.ReceiptsProtocolFor(root)
	if err != nil {
		t.Fatal(err)
	}
	roll := p.Steps[0]
	if strings.Contains(roll, "publishes from and") {
		t.Errorf("the roll step says the content commit is what the release publishes from:\n%s", roll)
	}
	for _, want := range []string{"every receipt names", "tagged merge"} {
		if !strings.Contains(roll, want) {
			t.Errorf("the roll step must carry %q:\n%s", want, roll)
		}
	}
}
