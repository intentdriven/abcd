package ahoy

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestAuditGapsRedactHomeInStalePathDetail: the history.path_stale gap quotes
// two absolute paths — the registered one from index.json and the repo's
// current location — and both must render through displayPath like every
// other path-bearing gap, so `ahoy doctor --json` pasted into an issue carries
// the locations without the username (GHSA-m8pg-chhv-hxvq).
func TestAuditGapsRedactHomeInStalePathDetail(t *testing.T) {
	home, _ := setupHermetic(t)
	if _, err := bootstrapHistory(); err != nil {
		t.Fatal(err)
	}
	sha := strings.Repeat("a", 40)
	registered := filepath.Join(home, "old", "repo")
	idx := &historyIndex{Schema: 1, Repos: []historyRepo{{RootCommit: sha, Path: registered, Status: "active"}}}
	if err := writeHistoryIndex(idx); err != nil {
		t.Fatal(err)
	}
	cwd := filepath.Join(home, "new", "repo")
	if err := os.MkdirAll(cwd, 0o755); err != nil {
		t.Fatal(err)
	}

	gaps := auditGaps(cwd, DetectionResult{RootSHA: sha})
	if len(gaps) != 1 || gaps[0].ID != "history.path_stale" {
		t.Fatalf("gaps = %+v, want exactly one history.path_stale", gaps)
	}
	detail := gaps[0].Detail
	if strings.Contains(detail, home) {
		t.Fatalf("path_stale detail carries the home prefix: %q", detail)
	}
	for _, want := range []string{"~/old/repo", "~/new/repo"} {
		if !strings.Contains(detail, want) {
			t.Fatalf("path_stale detail = %q, want it to name %q", detail, want)
		}
	}
}
