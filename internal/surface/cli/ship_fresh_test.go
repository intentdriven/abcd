package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestALocalCutNeedsACutReceiptForItsCommit: in a repository whose preflight mints
// receipts, a cut runs on a commit every test ran on with nothing reused. A receipt
// from a run that reused results does not count, no flag waives it, and a receipt for
// the commit in hand lets the cut on.
func TestALocalCutNeedsACutReceiptForItsCommit(t *testing.T) {
	r := shipRenderableRepo(t)
	script, err := os.ReadFile(filepath.Join(repoRootFromTest(t), "scripts", "preflight-receipt.sh"))
	if err != nil {
		t.Fatal(err)
	}
	r.Write("scripts/preflight-receipt.sh", string(script))
	r.Write(".gitignore", ".abcd/.work.local/\n")
	r.Commit("the repository mints receipts")
	head := strings.TrimSpace(r.Git("rev-parse", "HEAD"))
	before := readFileString(t, filepath.Join(r.Root(), "CHANGELOG.md"))
	payload := composedPayload(t, t.TempDir(), "v0.4.1", "itd-73")
	ship := func(extra ...string) ([]byte, error) {
		t.Helper()
		dest := filepath.Join(t.TempDir(), "payload")
		return shipIn(t, r, append([]string{"launch", "ship", "--changelog-json", payload, "--payload-dir", dest}, extra...)...)
	}
	refused := func(when string, extra ...string) {
		t.Helper()
		out, err := ship(extra...)
		if code := exitCodeOf(err); code != 2 {
			t.Fatalf("%s: exit = %d, want 2\n%s\n%v", when, code, out, err)
		}
		if !strings.Contains(err.Error(), "PREFLIGHT_FRESH") {
			t.Errorf("%s: the refusal does not say what to run: %v", when, err)
		}
		if after := readFileString(t, filepath.Join(r.Root(), "CHANGELOG.md")); after != before {
			t.Fatalf("%s: the refusal wrote the release record anyway", when)
		}
	}
	receipt := func(tests string) {
		t.Helper()
		path := filepath.Join(r.Root(), ".abcd", ".work.local", "preflight-receipts", head)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		body := "commit " + head + "\nminted 2026-11-05T00:00:00Z\nscope full\ntests " + tests + "\nreuse unit 0 of 1\nreuse race 0 of 1\n"
		if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	refused("no receipt")
	refused("no receipt, with --allow-dirty", "--allow-dirty")
	receipt("reuse")
	refused("a receipt that reused results")
	receipt("cut")
	if out, err := ship(); exitCodeOf(err) != 0 {
		t.Fatalf("a cut receipt did not let the cut on: %v\n%s", err, out)
	}
}
