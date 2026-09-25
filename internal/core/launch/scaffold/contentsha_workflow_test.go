package scaffold

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// TestReleaseConsumesTheContentShaVerifyGated holds the release job to the
// reviewed content commit verify's semantic gate actually passed
// (iss-2608291814573570). The publish job re-derived it with a second
// `record-lint --derive-content-sha` — a second compile and full-history walk
// per release, and a second derivation that could in principle name a
// different commit from the one the gate judged, which would sign receipts
// nobody gated. verify now exports the sha it gated as a job output, and the
// release job receives it, shape-checks it (it flows into $GITHUB_ENV, a file
// path and a signed predicate) and derives nothing.
func TestReleaseConsumesTheContentShaVerifyGated(t *testing.T) {
	rendered, err := Render(AbcdSubstitutions())
	if err != nil {
		t.Fatal(err)
	}
	wf := string(rendered.ReleaseYML)
	verify := jobSection(t, wf, "verify")
	rel := jobSection(t, wf, "release")

	if !strings.Contains(verify, "    outputs:\n      content_sha: ${{ steps.receipts.outputs.content_sha }}\n") {
		t.Error("verify must export the content sha its semantic gate passed as the content_sha job output")
	}
	gate := indexOf(t, verify, "go run ./cmd/record-lint --release-gate", "verify")
	out := indexOf(t, verify, `echo "content_sha=$content" >> "$GITHUB_OUTPUT"`, "verify")
	if out < gate {
		t.Error("verify must export the content sha only after the release gate admitted it")
	}
	step := verify[strings.LastIndex(verify[:gate], "- name:"):gate]
	if !strings.Contains(step, "id: receipts\n") {
		t.Error("the semantic-gate step must carry id: receipts, the id the job output reads")
	}

	if strings.Contains(rel, "--derive-content-sha") {
		t.Error("the release job must not re-derive the content sha; it consumes verify's output")
	}
	recv := indexOf(t, rel, "CONTENT_SHA_FROM_VERIFY: ${{ needs.verify.outputs.content_sha }}", "release")
	script := runBlock(t, rel[strings.LastIndex(rel[:recv], "- name:"):])

	sha := strings.Repeat("0123456789abcdef", 3)[:40]
	for _, c := range []struct {
		value string
		ok    bool
	}{
		{sha, true},
		{"", false},
		{"not-a-sha", false},
		{strings.ToUpper(sha), false},
		{sha + "\nFORGED=1", false},
	} {
		envFile := filepath.Join(t.TempDir(), "github-env")
		cmd := exec.Command("bash", "-c", script)
		cmd.Env = []string{"CONTENT_SHA_FROM_VERIFY=" + c.value, "GITHUB_ENV=" + envFile, "PATH=/usr/bin:/bin"}
		out, err := cmd.CombinedOutput()
		if got := err == nil; got != c.ok {
			t.Errorf("content sha %q: accepted=%v, want %v\n%s", c.value, got, c.ok, out)
			continue
		}
		written, _ := os.ReadFile(envFile)
		if c.ok && string(written) != "CONTENT_SHA="+sha+"\n" {
			t.Errorf("content sha %q: $GITHUB_ENV holds %q, want exactly the CONTENT_SHA line", c.value, written)
		}
		if !c.ok && len(written) != 0 {
			t.Errorf("content sha %q was refused but still reached $GITHUB_ENV: %q", c.value, written)
		}
	}
}
