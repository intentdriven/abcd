package cli

import (
	"bytes"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/abcdhome"
	"github.com/intentdriven/abcd/internal/core/mode"
	"github.com/intentdriven/abcd/internal/gittest"
)

// The status line is the one thing abcd wires into the harness's user
// settings, and the harness runs it on every refresh of every session, so it
// must never do damage (iss-2610050556383525, the product thinker's ruling of
// 2026-10-05). The first half of "never do damage" is that the verb WRITES
// NOTHING: no cache, no lock, no log, no ledger, no transcript, no mode file,
// and above all no home folder — neither ~/.abcd.noindex nor the old ~/.abcd,
// whose recreation by a stale hook is what the ruling was made over.
//
// These tests prove it end to end, through run — the same front door main
// calls, the old-home stop included — with the payload on stdin as the
// harness hands it, against a sandboxed HOME and a temporary checkout, and
// compare a snapshot of both trees taken before and after: every path, its
// mode, its size, its mtime and, for a link, its target. A path created,
// removed or touched in any way fails the test and is named.

// treeEntry is one path's observable state.
type treeEntry struct {
	mode  fs.FileMode
	size  int64
	mtime int64
	link  string
}

// snapshotTrees records every path under each root (the roots themselves
// included), keyed by absolute path. Lstat throughout: a link is recorded as
// itself and never followed out of the tree.
func snapshotTrees(t *testing.T, roots ...string) map[string]treeEntry {
	t.Helper()
	out := map[string]treeEntry{}
	for _, root := range roots {
		err := filepath.WalkDir(root, func(p string, _ fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			fi, err := os.Lstat(p)
			if err != nil {
				return err
			}
			e := treeEntry{mode: fi.Mode(), size: fi.Size(), mtime: fi.ModTime().UnixNano()}
			if fi.Mode()&fs.ModeSymlink != 0 {
				e.link, _ = os.Readlink(p)
			}
			if fi.IsDir() {
				// A directory's size is the filesystem's bookkeeping, not its
				// content; an entry added or removed shows as a path and an
				// mtime change.
				e.size = 0
			}
			out[p] = e
			return nil
		})
		if err != nil {
			t.Fatalf("snapshot %s: %v", root, err)
		}
	}
	return out
}

// diffTrees names every path created, removed or changed between two
// snapshots, sorted, one per line.
func diffTrees(before, after map[string]treeEntry) []string {
	var d []string
	for p, b := range before {
		a, ok := after[p]
		switch {
		case !ok:
			d = append(d, "removed  "+p)
		case a != b:
			d = append(d, fmt.Sprintf("changed  %s (%+v -> %+v)", p, b, a))
		}
	}
	for p := range after {
		if _, ok := before[p]; !ok {
			d = append(d, "created  "+p)
		}
	}
	sort.Strings(d)
	return d
}

// runStatuslineFrontDoor runs the verb through run, the entry main uses, with
// the payload on stdin, and returns its streams and exit status.
func runStatuslineFrontDoor(stdin string) (stdout, stderr string, code int) {
	var so, se bytes.Buffer
	code = run([]string{"statusline"}, strings.NewReader(stdin), &so, &se)
	return so.String(), se.String(), code
}

// readonlyCheckout lays a git checkout with one commit, so every git question
// the verb asks (toplevel, branch, root commit) has an answer to read.
func readonlyCheckout(t *testing.T) string {
	t.Helper()
	repo := realPath(t, t.TempDir())
	gitInitAt(t, repo)
	gitCommitAt(t, repo, "root")
	return repo
}

// manageByMarker gives the checkout the marker block, the local tier, a
// stored mode and a record with an open issue and a draft intent, so the
// managed row reads every input it has.
func manageByMarker(t *testing.T, repo string) {
	t.Helper()
	marker := "# Project\n\n<!-- BEGIN ABCD -->\nx\n<!-- END ABCD -->\n"
	writeFileAt(t, filepath.Join(repo, "AGENTS.md"), marker)
	if err := os.MkdirAll(filepath.Join(repo, filepath.FromSlash(mode.TierRelPath)), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := mode.SetAt(repo, mode.ProductThinker); err != nil {
		t.Fatal(err)
	}
	writeFileAt(t, filepath.Join(repo, ".abcd/work/issues/open/iss-1-a.md"), "---\n---\n")
	writeFileAt(t, filepath.Join(repo, ".abcd/development/intents/drafts/itd-1-a.md"), "---\n---\n")
}

// registerInHistory records the checkout in the user-level history index by
// its root commit — the second managed signal, which is the one that sends
// ahoy.Managed to git for the root commit.
func registerInHistory(t *testing.T, home, repo string) {
	t.Helper()
	cmd := exec.Command("git", "-C", repo, "rev-list", "-n", "1", "--max-parents=0", "HEAD")
	cmd.Env = gittest.Env(t)
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("root commit: %v", err)
	}
	sha := strings.TrimSpace(string(out))
	if err := os.MkdirAll(abcdhome.Path(home, "history"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(abcdhome.Path(home), 0o700); err != nil {
		t.Fatal(err)
	}
	body := `{"schema":1,"description":"x","repos":[{"root_commit":"` + sha + `","name":"r","path":"` + repo + `","status":"active"}]}`
	if err := os.WriteFile(abcdhome.Path(home, "history", "index.json"), []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
}

func writeFileAt(t *testing.T, p, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

// TestStatuslineWritesNothing is the read-only proof. Each case builds its
// own HOME and checkout, snapshots both, runs the verb through the front door
// twice (the second run is a refresh, and a verb that wrote a cache or a
// marker on its first run would read it on its second), and requires the
// snapshot unchanged and the exit 0.
func TestStatuslineWritesNothing(t *testing.T) {
	cases := []struct {
		name string
		// setup lays the case under home and repo; want is a substring the
		// verb's stdout must carry, so the case is proven to have taken the
		// path it names rather than an early exit.
		setup func(t *testing.T, home, repo string)
		want  string
	}{
		{
			name: "managed by marker, setting installed",
			setup: func(t *testing.T, home, repo string) {
				manageByMarker(t, repo)
				writeUserSettings(t, `{"schema_version":1,"previous_command":"printf prev"}`)
			},
			want: "waiting on the product thinker",
		},
		{
			name: "managed by history registration only",
			setup: func(t *testing.T, home, repo string) {
				registerInHistory(t, home, repo)
			},
			want: "abcd-managed",
		},
		{
			name: "unmanaged, previous command recorded",
			setup: func(t *testing.T, home, repo string) {
				writeUserSettings(t, `{"schema_version":1,"previous_command":"cat >/dev/null; printf prev"}`)
			},
			want: "prev",
		},
		{
			name: "managed by marker, no abcd home at all",
			setup: func(t *testing.T, home, repo string) {
				manageByMarker(t, repo)
			},
			want: "waiting on the product thinker",
		},
		{
			name:  "unmanaged, no abcd home at all",
			setup: func(t *testing.T, home, repo string) {},
			want:  "",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// Unresolved on purpose: gittest.Env keeps a HOME only while it
			// reads as inside the OS temp area, and the resolved /private
			// form does not, so it would mint a second HOME under the case.
			home := t.TempDir()
			t.Setenv("HOME", home)
			t.Setenv("ABCD_PLUGIN_ROOT", "")
			t.Setenv("CLAUDE_PLUGIN_ROOT", "")
			t.Setenv(statuslineFallbackEnv, "")
			repo := readonlyCheckout(t)
			tc.setup(t, home, repo)
			// The process stands somewhere else entirely: the harness runs
			// the status command from wherever it was launched, and the
			// checkout is the payload's cwd.
			t.Chdir(realPath(t, t.TempDir()))

			before := snapshotTrees(t, home, repo)
			for i := 0; i < 2; i++ {
				stdout, stderr, code := runStatuslineFrontDoor(payloadFor(repo))
				if code != 0 {
					t.Fatalf("run %d: exit %d, stderr %q", i+1, code, stderr)
				}
				if !strings.Contains(stdout, tc.want) {
					t.Fatalf("run %d: stdout %q does not carry %q (stderr %q)", i+1, stdout, tc.want, stderr)
				}
			}
			if d := diffTrees(before, snapshotTrees(t, home, repo)); len(d) > 0 {
				t.Errorf("the status verb changed the filesystem:\n%s", strings.Join(d, "\n"))
			}
			for _, name := range []string{".abcd", abcdhome.Rel()} {
				if _, ok := before[filepath.Join(home, name)]; ok {
					continue
				}
				if _, err := os.Lstat(filepath.Join(home, name)); err == nil {
					t.Errorf("the status verb created ~/%s", name)
				}
			}
		})
	}
}
