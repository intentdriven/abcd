package ahoy

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/abcdhome"
)

// TestReceiptPathRendersEachRootIdentityFree pins the seam's rendering per root:
// the repo relativised, the home directory abbreviated, a location that names no
// developer left exactly as written, and an embedded root redacted out of a note
// that is a sentence rather than a bare path.
func TestReceiptPathRendersEachRootIdentityFree(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	// A repo INSIDE the home directory is the ordinary case, and the one where
	// the two rules could disagree: repo-relative must win.
	repo := filepath.Join(home, "code", "proj")

	cases := []struct {
		name string
		in   string
		want string
	}{
		{"a repo write is repo-relative", filepath.Join(repo, ".abcd", "config.json"), ".abcd/config.json"},
		{"a marker is repo-relative", filepath.Join(repo, "CLAUDE.md"), "CLAUDE.md"},
		{"the user-scope store is home-relative", abcdhome.Path(home, "history", "index.json"), "~/.abcd.noindex/history/index.json"},
		{"a home-rooted PATH entry is home-relative", filepath.Join(home, ".local", "bin", "abcd"), "~/.local/bin/abcd"},
		{"a system location names no developer", "/usr/local/bin/abcd", "/usr/local/bin/abcd"},
		{"an already-relative note is untouched", ".abcd/config/identity.json", ".abcd/config/identity.json"},
		{"a fix-hint note is untouched", "dependency: brew install gitleaks", "dependency: brew install gitleaks"},
		{"an embedded home root is redacted out of a sentence", "dependency: add " + filepath.Join(home, "bin") + " to PATH", "dependency: add ~/bin to PATH"},
		{"a bare home root is redacted", home, "~"},
		// The prefix boundary: "proj-backup" is not inside "proj", so it must not
		// come out repo-relative — it falls through to the home rule instead.
		{"a sibling sharing the repo prefix is not repo-relative", repo + "-backup/config.json", "~/code/proj-backup/config.json"},
		{"empty stays empty", "", ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := receiptPath(repo, tc.in); got != tc.want {
				t.Errorf("receiptPath(%q) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}

// TestReceiptIsWrittenOnlyThroughNote is the structural half of iss-177, and the
// guarantee a later apply step rests on: the scrub cannot be bypassed because
// note is the only writer of the receipt slice. A step that appended to a.writes
// directly would skip it silently, so the single seam is asserted rather than
// left as a convention a reviewer has to notice.
func TestReceiptIsWrittenOnlyThroughNote(t *testing.T) {
	files, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatal(err)
	}
	total := 0
	for _, f := range files {
		if strings.HasSuffix(f, "_test.go") {
			continue
		}
		body, rerr := os.ReadFile(f)
		if rerr != nil {
			t.Fatal(rerr)
		}
		n := strings.Count(string(body), "writes = append(")
		if n > 0 && f != "receipt.go" {
			t.Errorf("%s appends to the receipt directly, bypassing note's path scrub", f)
		}
		total += n
	}
	if total != 1 {
		t.Errorf("the receipt slice is appended to %d times across the package, want exactly 1 (in note)", total)
	}
}

// TestInstallReceiptCarriesNoDeveloperIdentity is iss-177's headline: an install
// receipt is something a user pastes into an issue or a transcript, so no entry
// on it may carry the developer's home directory, username, or the absolute
// location of their repo. Every write is asserted, not a sampled few — the
// receipt is only safe to paste if the WHOLE of it is.
func TestInstallReceiptCarriesNoDeveloperIdentity(t *testing.T) {
	home, _ := setupHermetic(t)
	// Put the PATH entry under the home directory: that is where the no-sudo
	// field-standard location lives, and it is the write most likely to leak a
	// username.
	t.Setenv("ABCD_BIN_TARGET", filepath.Join(home, ".local", "bin", "abcd"))

	repo := t.TempDir()
	if err := os.Mkdir(filepath.Join(repo, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}

	res, err := Install(repo, installOpts(), RefusingPrompter{})
	if err != nil {
		t.Fatal(err)
	}
	if res.Status != "clean" {
		t.Fatalf("install status = %q (remaining=%v), want clean", res.Status, res.Remaining)
	}
	if len(res.Writes) == 0 {
		t.Fatal("install reported no writes; the receipt assertion would be vacuous")
	}

	for _, w := range res.Writes {
		if strings.Contains(w, home) {
			t.Errorf("receipt entry carries the home directory: %q", w)
		}
		if strings.Contains(w, repo) {
			t.Errorf("receipt entry carries the absolute repo path: %q", w)
		}
		if filepath.IsAbs(w) {
			t.Errorf("receipt entry is an absolute path: %q", w)
		}
	}

	// The scrub must not cost the receipt its meaning: it still names what was
	// written, in repo-relative and home-relative form.
	joined := strings.Join(res.Writes, "\n")
	for _, want := range []string{".abcd/config.json", "~/.abcd.noindex/history/index.json"} {
		if !strings.Contains(joined, want) {
			t.Errorf("receipt does not name %q:\n%s", want, joined)
		}
	}
}
