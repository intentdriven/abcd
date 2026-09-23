package guard

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestResolveWorkdirResolvesAgainstTheSessionDirectory pins the host semantics
// the 2026-09-23 probe observed on the one host whose shell tool takes a per-call
// working directory (opencode 1.18.31, its bash tool driven directly with
// `opencode debug agent build --tool bash`): a relative workdir is resolved
// against the session directory, an empty one means the session directory, and
// an absolute one is used as given. The same relative value names two different
// directories from two different sessions, which is the whole reason the guard
// resolves it rather than reading the string.
func TestResolveWorkdirResolvesAgainstTheSessionDirectory(t *testing.T) {
	session := t.TempDir()
	if err := os.MkdirAll(filepath.Join(session, "scratch"), 0o755); err != nil {
		t.Fatal(err)
	}
	other := t.TempDir() // a session with no scratch/ in it

	cases := []struct {
		name       string
		sessionDir string
		raw        string
		wantPath   string
		wantExists bool
	}{
		{"empty means the session directory: no workdir", session, "", "", false},
		{"relative, present in this session", session, "scratch", filepath.Join(session, "scratch"), true},
		{"the same relative value, absent from another session", other, "scratch", filepath.Join(other, "scratch"), false},
		{"dot segments are cleaned", session, "scratch/../scratch/", filepath.Join(session, "scratch"), true},
		{"absolute is taken as given", other, filepath.Join(session, "scratch"), filepath.Join(session, "scratch"), true},
		{"a name with a space and non-ASCII letters is an ordinary path", session, "café dir", filepath.Join(session, "café dir"), false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			wd, err := ResolveWorkdir(tc.sessionDir, tc.raw)
			if err != nil {
				t.Fatalf("ResolveWorkdir(%q, %q): unexpected refusal: %v", tc.sessionDir, tc.raw, err)
			}
			if wd.Path != tc.wantPath || wd.Exists != tc.wantExists {
				t.Errorf("ResolveWorkdir(%q, %q) = %+v, want {Path:%q Exists:%v}", tc.sessionDir, tc.raw, wd, tc.wantPath, tc.wantExists)
			}
		})
	}
}

// TestResolveWorkdirTreatsAFileAsAbsent: the probe found a workdir naming a
// regular file fails the host call exactly as a missing one does (ENOTDIR at
// spawn), so the guard must not report it as a directory the command runs in.
func TestResolveWorkdirTreatsAFileAsAbsent(t *testing.T) {
	session := t.TempDir()
	if err := os.WriteFile(filepath.Join(session, "afile"), nil, 0o644); err != nil {
		t.Fatal(err)
	}
	wd, err := ResolveWorkdir(session, "afile")
	if err != nil {
		t.Fatalf("unexpected refusal: %v", err)
	}
	if wd.Exists {
		t.Errorf("a workdir naming a regular file must not read as an existing directory: %+v", wd)
	}
}

// TestResolveWorkdirRefusesMalformedValues is the refusal half: the workdir is
// written by the model, so it is untrusted input, and a value no directory can
// be named by is refused with the reason rather than resolved into something
// the host would never run the command in.
func TestResolveWorkdirRefusesMalformedValues(t *testing.T) {
	session := t.TempDir()
	cases := []struct {
		name       string
		sessionDir string
		raw        string
		wantInErr  string
	}{
		{"a NUL byte", session, "scratch\x00/etc", "NUL"},
		{"a newline", session, "scratch\nrm -rf /", "control character"},
		{"an escape sequence", session, "scratch\x1b[2J", "control character"},
		{"a DEL byte", session, "scratch\x7f", "control character"},
		{"invalid UTF-8", session, "scratch\xff", "UTF-8"},
		{"over the length cap", session, strings.Repeat("a", MaxWorkdirBytes+1), "over"},
		{"relative with no session directory to resolve it against", "", "scratch", "relative"},
		{"relative against a relative session directory", "not/absolute", "scratch", "relative"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			wd, err := ResolveWorkdir(tc.sessionDir, tc.raw)
			if err == nil {
				t.Fatalf("ResolveWorkdir(%q) must refuse; got %+v", tc.raw, wd)
			}
			if !errors.Is(err, ErrMalformedWorkdir) {
				t.Errorf("the refusal must wrap ErrMalformedWorkdir; got %v", err)
			}
			if !strings.Contains(err.Error(), tc.wantInErr) {
				t.Errorf("the refusal must name what it found (%q); got %q", tc.wantInErr, err)
			}
			if wd != (Workdir{}) {
				t.Errorf("a refusal must resolve nothing; got %+v", wd)
			}
		})
	}
	// The boundary itself is admitted: exactly the cap is a path, one past it is not.
	if _, err := ResolveWorkdir("/", strings.Repeat("a", MaxWorkdirBytes)); err != nil {
		t.Errorf("a workdir of exactly MaxWorkdirBytes must be admitted; got %v", err)
	}
}

// TestAHostWorkdirIsNeverReadAsACd is the probe's consequence for the registry.
// The rm-rf-after-cd-chain entry exists because a FAILED cd leaves the delete
// running wherever the shell already was. A host-managed workdir that does not
// exist fails the whole call instead (probe: NotFound, nothing ran), so the
// failed-cd hazard is absent and folding the workdir into the string as
// `cd <workdir> && ` would block every workdir'd recursive delete for a hazard
// the host does not have. The command string alone decides the cd-chain entry.
func TestAHostWorkdirIsNeverReadAsACd(t *testing.T) {
	if d := checkOK(t, "rm -rf *"); d.Verdict != VerdictAllow {
		t.Errorf("a bare recursive delete (run in a host workdir) must stay allowed: %+v", d)
	}
	if d := checkOK(t, "cd scratch && rm -rf *"); d.Verdict != VerdictBlock || d.EntryID != "rm-rf-after-cd-chain" {
		t.Errorf("a cd chain inside the command string must still block: %+v", d)
	}
}

// TestStrictestPicksTheMostSevereVerdict: when a command is checked against two
// registries — the session's and the one for the directory it runs in — the
// stricter answer wins, so the second registry can add a hazard but can never
// take one away. On a tie the first (the session's) decision is kept.
func TestStrictestPicksTheMostSevereVerdict(t *testing.T) {
	allow := Decision{Verdict: VerdictAllow}
	warnA := Decision{Verdict: VerdictWarn, EntryID: "a"}
	warnB := Decision{Verdict: VerdictWarn, EntryID: "b"}
	block := Decision{Verdict: VerdictBlock, EntryID: "c"}

	cases := []struct {
		name string
		in   []Decision
		want string
		verd Verdict
	}{
		{"allow then block", []Decision{allow, block}, "c", VerdictBlock},
		{"block then allow", []Decision{block, allow}, "c", VerdictBlock},
		{"warn then block", []Decision{warnA, block}, "c", VerdictBlock},
		{"allow then warn", []Decision{allow, warnB}, "b", VerdictWarn},
		{"tie keeps the first", []Decision{warnA, warnB}, "a", VerdictWarn},
		{"allow and allow", []Decision{allow, allow}, "", VerdictAllow},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := Strictest(tc.in...)
			if got.Verdict != tc.verd || got.EntryID != tc.want {
				t.Errorf("Strictest = {%s %q}, want {%s %q}", got.Verdict, got.EntryID, tc.verd, tc.want)
			}
		})
	}
	if got := Strictest(); got.Verdict != VerdictAllow {
		t.Errorf("Strictest() of nothing must be an allow; got %+v", got)
	}
}
