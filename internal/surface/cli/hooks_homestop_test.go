package cli

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/abcdhome"
)

// hooks_homestop_test.go — the stop in the two scripts that run ahead of the
// binary (spc-2610031309233367, "The stop"): the hooks' shell wrapper and
// hooks/bootstrap.sh. With an old ~/.abcd standing neither provisions, reads a
// path-entry record, or writes anything under the home; the wrapper hands the
// stop to a plugin-root binary when there is one, and says the line itself
// when there is none.

// stopWrapperEvents is every hooks.json event and the status its missing-binary
// branch exits with, which is what the stop exits with when no binary can say it.
var stopWrapperEvents = []struct {
	event string
	code  int
	verbs []string
}{
	{"UserPromptSubmit", 1, []string{"hook prompt-router"}},
	{"SessionStart", 2, []string{"hook session-start", "hook prompt-router-reset"}},
	{"PreToolUse", 1, []string{"guard hook"}},
	{"PreCompact", 1, []string{"hook prompt-router-reset"}},
	{"SessionEnd", 1, []string{"hook session-end"}},
	{"SubagentStop", 1, []string{"hook subagent-stop"}},
}

// oldHomeWithPathEntry stands up a home holding the old ~/.abcd, whose
// path-entry vouches for an abcd on PATH (so a wrapper that still read it would
// run that binary), and, when both is set, a ~/.abcd.noindex beside it holding
// the same record. It returns the home and the PATH directory.
func oldHomeWithPathEntry(t *testing.T, both bool) (home, pathDir string) {
	t.Helper()
	home = sandboxHome(t)
	pathDir = t.TempDir()
	pathStub(t, pathDir)
	body := "path=" + filepath.Join(pathDir, "abcd") + "\nbinary_sha256=" + strings.Repeat("a", 64) + "\n"
	dirs := []string{filepath.Join(home, ".abcd")}
	if both {
		dirs = append(dirs, abcdhome.Path(home))
	}
	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, "path-entry"), []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	return home, pathDir
}

// TestHookWrapperStopsBeforeProvisioning: with no plugin-root binary and the
// old folder standing, every hooks.json command says the stop line, exits as
// its missing-binary branch does, and leaves no bootstrap attempt marker, no
// bootstrap run, no PATH binary run and no change under HOME. With a
// plugin-root binary it skips the bootstrap and hands the stop to the binary,
// which renders it in its own form.
func TestHookWrapperStopsBeforeProvisioning(t *testing.T) {
	for _, both := range []bool{false, true} {
		for _, e := range stopWrapperEvents {
			name := e.event
			if both {
				name += "/both folders"
			}
			t.Run(name+"/no binary", func(t *testing.T) {
				home, pathDir := oldHomeWithPathEntry(t, both)
				stop := abcdhome.Check(home)
				if stop == nil || stop.Both != both {
					t.Fatalf("abcdhome.Check = %+v, want a stop (both=%v)", stop, both)
				}
				root := hookRoot(t, provisioningBootstrap, false)
				before := homeTree(t, home)
				_, stderr, code := hookRunHome(t, e.event, root, pathDir, t.TempDir(), home)
				if code != e.code || stderr != "abcd: "+stop.Line+"\n" {
					t.Fatalf("exit %d, stderr %q\nwant exit %d and the stop line", code, stderr, e.code)
				}
				for _, f := range []string{".bootstrap.attempt", "boot.log", "calls.log", "abcd"} {
					if _, err := os.Lstat(filepath.Join(root, f)); !errors.Is(err, os.ErrNotExist) {
						t.Errorf("the stopped wrapper left %s in the plugin root (%v)", f, err)
					}
				}
				if after := homeTree(t, home); after != before {
					t.Fatalf("the stopped wrapper changed the home:\nbefore:\n%s\nafter:\n%s", before, after)
				}
			})
			t.Run(name+"/plugin-root binary", func(t *testing.T) {
				home, pathDir := oldHomeWithPathEntry(t, both)
				root := hookRoot(t, provisioningBootstrap, true)
				before := homeTree(t, home)
				_, stderr, code := hookRunHome(t, e.event, root, pathDir, t.TempDir(), home)
				if code != 0 {
					t.Fatalf("exit %d with the stub binary; stderr %q", code, stderr)
				}
				calls := callLog(t, filepath.Join(root, "calls.log"))
				for _, v := range e.verbs {
					if !strings.Contains(calls, v) {
						t.Errorf("the wrapper did not hand %q to the plugin-root binary; calls %q", v, calls)
					}
				}
				for _, f := range []string{".bootstrap.attempt", "boot.log"} {
					if _, err := os.Lstat(filepath.Join(root, f)); !errors.Is(err, os.ErrNotExist) {
						t.Errorf("the stopped wrapper left %s in the plugin root (%v)", f, err)
					}
				}
				if after := homeTree(t, home); after != before {
					t.Fatalf("the stopped wrapper changed the home:\nbefore:\n%s\nafter:\n%s", before, after)
				}
			})
		}
	}
}

// TestBootstrapWritesNothingBesideTheOldHome: the bootstrap a person can also
// run by hand ends with the stop line right after its HOME checks, before any
// download and before any record: no fetch reaches the release host, the plugin
// root and the data dir stay empty, the home is unchanged and no
// ~/.abcd.noindex is created beside the old folder.
func TestBootstrapWritesNothingBesideTheOldHome(t *testing.T) {
	bootstrapRequires(t)
	for _, both := range []bool{false, true} {
		name := "old folder"
		if both {
			name = "both folders"
		}
		t.Run(name, func(t *testing.T) {
			body := []byte("#!/bin/sh\n# abcd release artefact fixture\nexit 0\n")
			fx := bootstrapServer(t, body, bootstrapManifest(body))
			root := bootstrapRoot(t)
			data := t.TempDir()
			home := t.TempDir()
			if err := os.MkdirAll(filepath.Join(home, ".abcd", "runs"), 0o700); err != nil {
				t.Fatal(err)
			}
			if both {
				if err := os.MkdirAll(abcdhome.Path(home), 0o700); err != nil {
					t.Fatal(err)
				}
			}
			stop := abcdhome.Check(home)
			if stop == nil {
				t.Fatal("abcdhome.Check reports no stop with ~/.abcd standing")
			}
			before := homeTree(t, home)
			out, code := runScript(t, bootstrapFixtureScript(t, fx.base), root,
				append(fx.env(), "HOME="+home, "CLAUDE_PLUGIN_DATA="+data), "")
			if code != 1 || out != "abcd: "+stop.Line+"\n" {
				t.Fatalf("exit %d, output %q\nwant exit 1 and the stop line alone", code, out)
			}
			if n := *fx.hits; n != 0 {
				t.Errorf("the stopped bootstrap made %d request(s) to the release host", n)
			}
			for _, dir := range []string{root, data} {
				if entries, err := os.ReadDir(dir); err != nil || len(entries) != 0 {
					t.Errorf("the stopped bootstrap wrote into %s: %v (%v)", dir, entries, err)
				}
			}
			if after := homeTree(t, home); after != before {
				t.Fatalf("the stopped bootstrap changed the home:\nbefore:\n%s\nafter:\n%s", before, after)
			}
		})
	}
}
