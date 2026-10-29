package cli

import (
	"encoding/json"
	"strings"
	"testing"
)

// TestReserveOfADirectoryWarnsInBothRenders (iss-2610230016544142): a
// footprint naming a directory git tracks is still reserved, exit 0; the
// plain render warns on stderr, naming the directory and how many tracked
// files it covers, and --json carries it as directory_footprint. A footprint
// of files alone draws no warning.
func TestReserveOfADirectoryWarnsInBothRenders(t *testing.T) {
	w := newReserveWorld(t)
	p := w.clone(t, "p", octoMail)
	writeRel(t, p, "docs/a.md", "a\n")
	writeRel(t, p, "docs/guide/b.md", "b\n")
	gitCmd(t, p, "add", "-A")
	gitCommit(t, p, "commit", "-q", "-m", "docs")
	gitCmd(t, p, "push", "-q", "origin", "main")

	code, stdout, stderr := reserveRun(t, p, "reserve", "iss-1", "--footprint", "docs", "--footprint", "README.md")
	const warning = `abcd reserve: warning: footprint "docs" is a directory covering 2 tracked files on origin/main`
	if code != 0 || !strings.Contains(stdout, "reserved iss-1 for "+heldBy(t, p)) || !strings.HasPrefix(stderr, warning) ||
		!strings.Contains(stderr, "name the files the work changes instead") || strings.Count(stderr, "\n") != 1 ||
		strings.Contains(stdout, "warning") {
		t.Fatalf("the reserve of a directory: exit %d, stdout %q, stderr %q; want it made, warned on stderr", code, stdout, stderr)
	}
	if w.remoteTip(t, "iss-1") == "" {
		t.Fatal("the reservation that warned did not reach origin")
	}

	code, raw, stderr := reserveRun(t, p, "reserve", "iss-2", "--footprint", "docs/guide", "--json")
	var got struct {
		DirectoryFootprint []struct {
			Path         string `json:"path"`
			TrackedFiles int    `json:"tracked_files"`
			On           string `json:"on"`
		} `json:"directory_footprint"`
	}
	if err := json.Unmarshal([]byte(raw), &got); code != 0 || err != nil || len(got.DirectoryFootprint) != 1 ||
		got.DirectoryFootprint[0].Path != "docs/guide" || got.DirectoryFootprint[0].TrackedFiles != 1 || got.DirectoryFootprint[0].On != "origin/main" {
		t.Fatalf("the --json reserve: exit %d, %+v (%v); want directory_footprint naming docs/guide, 1 file\n%s%s", code, got.DirectoryFootprint, err, raw, stderr)
	}
	if stderr != "" {
		t.Fatalf("the --json reserve wrote to stderr: %q", stderr)
	}

	code, stdout, stderr = reserveRun(t, p, "reserve", "iss-3", "--footprint", "README.md", "--footprint", "docs/new.md")
	if code != 0 || stderr != "" {
		t.Fatalf("a footprint of files alone: exit %d, stdout %q, stderr %q; want no warning", code, stdout, stderr)
	}
	code, raw, _ = reserveRun(t, p, "reserve", "iss-4", "--footprint", "README.md", "--json")
	if code != 0 || strings.Contains(raw, "directory_footprint") {
		t.Fatalf("a file footprint's --json names no directory: exit %d\n%s", code, raw)
	}
}
