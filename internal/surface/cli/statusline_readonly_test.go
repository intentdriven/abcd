package cli

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/intentdriven/abcd/internal/abcdhome"
	"github.com/intentdriven/abcd/internal/core/mode"
	"github.com/intentdriven/abcd/internal/core/rules"
	"github.com/intentdriven/abcd/internal/gittest"
	"github.com/intentdriven/abcd/internal/gitutil"
)

// The status line is the one thing abcd wires into the harness's user
// settings, and the harness runs it on every refresh of every session, so it
// must never do damage (iss-2610050556383525, the product thinker's ruling of
// 2026-10-05). The first half of "never do damage" is that the verb writes
// nothing but the session's fill file: no cache, no lock, no log, no ledger,
// no transcript, no mode file, no directory, and above all no home folder —
// neither ~/.abcd.noindex nor the old ~/.abcd, whose recreation by a stale
// hook is what the ruling was made over. The fill file is the one exception
// the product thinker granted on 2026-10-11 ("Allow one fill file";
// spc-2610111627213138 step 5): the context percentage of the session the
// row is rendered for, in ~/.abcd.noindex/fill/<sha256(session)>.json, and
// only into a fill directory the session-start hook already made.
//
// These tests prove it end to end, through run — the same front door main
// calls, the old-home stop included — with the payload on stdin as the
// harness hands it, against a sandboxed HOME and a temporary checkout, and
// compare a snapshot of both trees taken before and after: every path, its
// mode, its size, its mtime and, for a link, its target. A path created,
// removed or touched in any way fails the test and is named, but for the
// session's fill file and the fill directory's own mtime its creation moves.

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

// fillSession is the session id the fill tests' payloads carry.
const fillSession = "sess-fill-1"

// sessionPayloadFor is the status payload the harness hands a session's
// refresh: the checkout, the session id and the context percentage.
func sessionPayloadFor(repo string) string {
	return `{"cwd":"` + repo + `","session_id":"` + fillSession + `","model":{"display_name":"Opus"},"context_window":{"used_percentage":8}}`
}

// layFillDir makes ~/.abcd.noindex/fill/ as the session-start hook's reset
// makes it.
func layFillDir(t *testing.T, home string) {
	t.Helper()
	if err := rules.ResetFill(home, ""); err != nil {
		t.Fatalf("laying the fill directory: %v", err)
	}
}

// fillPath is the session's fill file under home.
func fillPath(home, session string) string {
	sum := sha256.Sum256([]byte(session))
	return abcdhome.Path(home, "fill", hex.EncodeToString(sum[:])+".json")
}

// statuslineWriteCase is one run of the read-only proof: setup lays the case
// under home and repo; payload is the stdin; want is a substring the verb's
// stdout must carry, so the case is proven to have taken the path it names
// rather than an early exit; wantFill says the session's fill file is the one
// write it may make.
type statuslineWriteCase struct {
	name     string
	setup    func(t *testing.T, home, repo string)
	payload  func(repo string) string
	want     string
	wantFill bool
}

// runStatuslineWriteCase builds the case's own HOME and checkout, snapshots
// both, runs the verb through the front door twice (the second run is a
// refresh, and a verb that wrote a cache or a marker on its first run would
// read it on its second), and requires the exit 0 and the snapshot unchanged
// but, where the case expects it, for the session's fill file, created
// holding the percentage and the time alone, and the fill directory's mtime
// its creation moves. It returns the last run's stderr.
func runStatuslineWriteCase(t *testing.T, tc statuslineWriteCase) string {
	t.Helper()
	// Unresolved on purpose: gittest.Env keeps a HOME only while it reads as
	// inside the OS temp area, and the resolved /private form does not, so it
	// would mint a second HOME under the case.
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("ABCD_PLUGIN_ROOT", "")
	t.Setenv("CLAUDE_PLUGIN_ROOT", "")
	t.Setenv(statuslineFallbackEnv, "")
	repo := readonlyCheckout(t)
	tc.setup(t, home, repo)
	// The process stands somewhere else entirely: the harness runs the status
	// command from wherever it was launched, and the checkout is the
	// payload's cwd.
	t.Chdir(realPath(t, t.TempDir()))
	payload := tc.payload
	if payload == nil {
		payload = sessionPayloadFor
	}

	before := snapshotTrees(t, home, repo)
	var stderr string
	for i := 0; i < 2; i++ {
		var stdout string
		var code int
		stdout, stderr, code = runStatuslineFrontDoor(payload(repo))
		if code != 0 {
			t.Fatalf("run %d: exit %d, stderr %q", i+1, code, stderr)
		}
		if !strings.Contains(stdout, tc.want) {
			t.Fatalf("run %d: stdout %q does not carry %q (stderr %q)", i+1, stdout, tc.want, stderr)
		}
	}
	fill := fillPath(home, fillSession)
	var unexpected []string
	created := false
	for _, d := range diffTrees(before, snapshotTrees(t, home, repo)) {
		switch {
		case tc.wantFill && d == "created  "+fill:
			created = true
		case tc.wantFill && strings.HasPrefix(d, "changed  "+filepath.Dir(fill)+" "):
			// The directory's own mtime, which adding the file moves.
		default:
			unexpected = append(unexpected, d)
		}
	}
	if len(unexpected) > 0 {
		t.Errorf("the status verb changed the filesystem beyond the session's fill file:\n%s", strings.Join(unexpected, "\n"))
	}
	if tc.wantFill {
		if !created {
			t.Fatalf("the session's fill file was not written (stderr %q)", stderr)
		}
		data, err := os.ReadFile(fill)
		if err != nil {
			t.Fatal(err)
		}
		var rec map[string]any
		if err := json.Unmarshal(data, &rec); err != nil {
			t.Fatalf("the fill file is not JSON: %v\n%s", err, data)
		}
		if _, ok := rec["at"]; len(rec) != 2 || rec["used_percentage"] != 8.0 || !ok {
			t.Errorf("the fill file = %s, want used_percentage 8 and at alone", data)
		}
	}
	for _, name := range []string{".abcd", abcdhome.Rel()} {
		if _, ok := before[filepath.Join(home, name)]; ok {
			continue
		}
		if _, err := os.Lstat(filepath.Join(home, name)); err == nil {
			t.Errorf("the status verb created ~/%s", name)
		}
	}
	return stderr
}

// TestStatuslineWritesOnlyTheFillFile is the read-only proof, narrowed to the
// one file the ruling of 2026-10-11 allows: on the path that renders abcd's
// row, for a payload carrying the session id and the context percentage, the
// one changed entry is the session's fill file, holding the percentage and
// the time alone; every other case, the pass-through path among them, leaves
// both trees unchanged.
func TestStatuslineWritesOnlyTheFillFile(t *testing.T) {
	for _, tc := range []statuslineWriteCase{
		{
			name: "managed by marker, setting installed",
			setup: func(t *testing.T, home, repo string) {
				manageByMarker(t, repo)
				writeUserSettings(t, `{"schema_version":1,"previous_command":"printf prev"}`)
				layFillDir(t, home)
			},
			want:     "waiting on the product thinker",
			wantFill: true,
		},
		{
			name: "managed by history registration only",
			setup: func(t *testing.T, home, repo string) {
				registerInHistory(t, home, repo)
				layFillDir(t, home)
			},
			want:     "abcd-managed",
			wantFill: true,
		},
		{
			name: "unmanaged, previous command recorded",
			setup: func(t *testing.T, home, repo string) {
				writeUserSettings(t, `{"schema_version":1,"previous_command":"cat >/dev/null; printf prev"}`)
				layFillDir(t, home)
			},
			want: "prev",
		},
		{
			name: "managed, setting off, previous command recorded",
			setup: func(t *testing.T, home, repo string) {
				manageByMarker(t, repo)
				writeUserSettings(t, `{"schema_version":1,"disabled":true,"previous_command":"cat >/dev/null; printf prev"}`)
				layFillDir(t, home)
			},
			want: "prev",
		},
		{
			name:  "unmanaged, no abcd home at all",
			setup: func(t *testing.T, home, repo string) {},
			want:  "",
		},
	} {
		t.Run(tc.name, func(t *testing.T) { runStatuslineWriteCase(t, tc) })
	}
}

// TestStatuslineCreatesNoDirectory: the fill directory is the session-start
// hook's to make, never the status line's. With no abcd home, or a home with
// no fill directory, the verb renders its row, writes nothing, makes no
// directory, and says nothing about the skipped write.
//
// "Says nothing" is judged line by line, and the one line it lets through is
// the merging element dropped because its facilitator.mode read ran out of
// git's share of the refresh (timedOutMergingRead). That read asks git about
// the default branch with no path around it, so a loaded race runner can
// outlast even the widened share, and its note is about a read, not a write:
// it failed this test over a directory it never touched
// (iss-2611020720003337). Every other line, the fill write's own notice
// among them, still fails it.
func TestStatuslineCreatesNoDirectory(t *testing.T) {
	for _, tc := range []statuslineWriteCase{
		{
			name:  "no abcd home at all",
			setup: func(t *testing.T, home, repo string) { manageByMarker(t, repo) },
			want:  "waiting on the product thinker",
		},
		{
			name: "an abcd home with no fill directory",
			setup: func(t *testing.T, home, repo string) {
				manageByMarker(t, repo)
				registerInHistory(t, home, repo)
			},
			want: "waiting on the product thinker",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			// The row this case wants is the whole row with git's answers in
			// it, so git gets a share a loaded machine cannot outrun.
			widenBudgets(t)
			stderr := runStatuslineWriteCase(t, tc)
			for _, line := range strings.Split(strings.TrimSuffix(stderr, "\n"), "\n") {
				if line != "" && !timedOutMergingRead(line) {
					t.Errorf("a skipped write was said: stderr line %q (all of stderr %q)", line, stderr)
				}
			}
		})
	}
}

// timedOutMergingRead says whether one stderr line of the status verb is the
// merging element dropped because git did not answer its facilitator.mode
// read inside the refresh's git share: landing.AutonomousMerging's note, over
// either deadline wording — its own check before the layered read
// ("git did not answer in time ...") or a git the share ended inside that
// read ("... context deadline exceeded"). A drop for any other reason is not
// it.
func timedOutMergingRead(line string) bool {
	const drop = "abcd statusline: statusline: DROPPED the merging element — facilitator.mode could not be read ("
	return strings.HasPrefix(line, drop) &&
		(strings.Contains(line, gitutil.ErrGitTimedOut.Error()) || strings.Contains(line, context.DeadlineExceeded.Error()))
}

// TestStatuslineWritesNothingWithoutASessionID: a payload with no session id
// names no fill file, and one with no context percentage has no figure to
// record, so neither writes, though the fill directory is there.
func TestStatuslineWritesNothingWithoutASessionID(t *testing.T) {
	for _, tc := range []statuslineWriteCase{
		{
			name: "no session id",
			setup: func(t *testing.T, home, repo string) {
				manageByMarker(t, repo)
				layFillDir(t, home)
			},
			payload: payloadFor,
			want:    "waiting on the product thinker",
		},
		{
			name: "no context percentage",
			setup: func(t *testing.T, home, repo string) {
				manageByMarker(t, repo)
				layFillDir(t, home)
			},
			payload: func(repo string) string {
				return `{"cwd":"` + repo + `","session_id":"` + fillSession + `","model":{"display_name":"Opus"},"context_window":{"used_percentage":null}}`
			},
			want: "waiting on the product thinker",
		},
	} {
		t.Run(tc.name, func(t *testing.T) { runStatuslineWriteCase(t, tc) })
	}
}

// TestStatuslineRecordsTheFillForItsSession: the figure the status line was
// handed is the one the prompt hook reads for that session, the latest
// refresh's figure replacing the one before; another session's fill is not
// this one's; and the session id is in no line the verb prints.
func TestStatuslineRecordsTheFillForItsSession(t *testing.T) {
	widenBudgets(t)
	t.Setenv(statuslineFallbackEnv, "")
	repo := managedCheckout(t) // which sets a HOME of its own
	home := os.Getenv("HOME")
	layFillDir(t, home)
	at := time.Date(2026, 11, 1, 9, 30, 0, 0, time.UTC)
	prev := statusFillClock
	statusFillClock = func() time.Time { return at }
	t.Cleanup(func() { statusFillClock = prev })

	for _, pct := range []string{"8", "63.5"} {
		payload := `{"cwd":"` + repo + `","session_id":"` + fillSession + `","model":{"display_name":"Opus"},"context_window":{"used_percentage":` + pct + `}}`
		for _, args := range [][]string{{"statusline"}, {"statusline", "--json"}} {
			stdout, stderr, err := runSplit(t, payload, args...)
			if err != nil {
				t.Fatalf("%v: %v\n%s", args, err, stderr)
			}
			if strings.Contains(stdout+stderr, fillSession) {
				t.Errorf("%v printed the session id:\n%s%s", args, stdout, stderr)
			}
		}
	}
	got := rules.ReadFill(home, fillSession, "")
	if got.Source != rules.FillSourceStatusLine || got.UsedPercentage == nil || *got.UsedPercentage != 63.5 || got.At != "2026-11-01T09:30:00Z" {
		t.Fatalf("the hook reads %+v for the session, want the status line's 63.5 at 09:30", got)
	}
	if other := rules.ReadFill(home, "another-session", ""); other.Known() {
		t.Errorf("another session reads this one's fill: %+v", other)
	}
}

// TestStatuslineRowRendersWhenTheFillWriteFails: a fill write that fails for
// any reason but an absent directory (here a fill directory another account
// could change, which the guarded walk refuses) is named on stderr, and the
// row renders and the verb exits 0 either way.
func TestStatuslineRowRendersWhenTheFillWriteFails(t *testing.T) {
	widenBudgets(t)
	t.Setenv(statuslineFallbackEnv, "")
	repo := managedCheckout(t) // which sets a HOME of its own
	home := os.Getenv("HOME")
	layFillDir(t, home)
	if err := os.Chmod(abcdhome.Path(home, "fill"), 0o777); err != nil {
		t.Fatal(err)
	}
	stdout, stderr, err := runSplit(t, sessionPayloadFor(repo), "statusline")
	if err != nil {
		t.Fatalf("statusline: %v\n%s", err, stderr)
	}
	if !strings.Contains(stdout, "abcd-managed") && !strings.Contains(stdout, "waiting on") {
		t.Errorf("the row did not render: stdout %q", stdout)
	}
	if !strings.Contains(stderr, "context fill was not recorded") || !strings.Contains(stderr, "the row renders without it") {
		t.Errorf("the failed write was not named: stderr %q", stderr)
	}
	if strings.Contains(stderr, home) {
		t.Errorf("the note names the home's path: %q", stderr)
	}
	if _, err := os.Lstat(fillPath(home, fillSession)); !os.IsNotExist(err) {
		t.Errorf("a refused directory was written into: %v", err)
	}
}
