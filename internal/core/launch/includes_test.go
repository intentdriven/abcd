package launch

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"
)

func TestLoadIncludesValid(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, ".abcd/config/launch-payload.json", `{"includes": ["./commands/", "docs", "docs", "README.md"]}`)
	inc, err := LoadIncludes(root)
	if err != nil {
		t.Fatal(err)
	}
	// Normalised (./ stripped, trailing / dropped) and de-duplicated.
	want := []string{"commands", "docs", "README.md"}
	if len(inc) != len(want) {
		t.Fatalf("got %v want %v", inc, want)
	}
	for i := range want {
		if inc[i] != want[i] {
			t.Errorf("include[%d]=%q want %q", i, inc[i], want[i])
		}
	}
}

func TestLoadIncludesPreflightErrors(t *testing.T) {
	cases := []struct {
		name    string
		content string
	}{
		{"denied_rooted", `{"includes": [".abcd/x"]}`},
		{"denied_glob", `{"includes": [".a*/x"]}`},
		{"absolute", `{"includes": ["/etc/passwd"]}`},
		{"windows_abs", `{"includes": ["C:\\secret"]}`},
		{"dotdot", `{"includes": ["../outside"]}`},
		{"backslash", `{"includes": ["a\\b"]}`},
		{"empty_array", `{"includes": []}`},
		{"empty_string", `{"includes": [""]}`},
		{"not_object", `["commands"]`},
		{"missing_key", `{"other": 1}`},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			root := t.TempDir()
			writeFile(t, root, ".abcd/config/launch-payload.json", c.content)
			_, err := LoadIncludes(root)
			if err == nil {
				t.Fatalf("expected preflight error for %s", c.name)
			}
			var pe *PreflightError
			if !errors.As(err, &pe) {
				t.Errorf("expected *PreflightError, got %T: %v", err, err)
			}
		})
	}
}

func TestLoadIncludesMissingConfig(t *testing.T) {
	root := t.TempDir()
	_, err := LoadIncludes(root)
	if err == nil {
		t.Fatal("expected preflight error for missing config")
	}
}

// TestAbcdCannotBeReincluded proves there is no allowlist path into the bundle
// for .abcd/** — the config line is refused at preflight AND, even if a raw
// include slice somehow carried it, the resolver still excludes it.
func TestAbcdCannotBeReincluded(t *testing.T) {
	root := t.TempDir()
	// Preflight refusal of the config line.
	writeFile(t, root, ".abcd/config/launch-payload.json", `{"includes": [".abcd/x"]}`)
	if _, err := LoadIncludes(root); err == nil {
		t.Error("LoadIncludes must refuse a .abcd-rooted include")
	}
	// Even a hand-forced include slice cannot promote a .abcd path.
	writeFile(t, root, ".abcd/development/brief.md", "SECRET")
	b, err := ResolveBundle(root, []string{".", ".abcd"})
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range b.Included {
		if firstSegment(f.LogicalPath) == ".abcd" {
			t.Errorf(".abcd path was promoted into the bundle: %s", f.LogicalPath)
		}
	}
}

// TestTheLaunchPayloadConfigIsReadOnlyInsideTheRepository: the include config
// is read only when it is a regular file within its cap, reached without
// leaving the repository. A symlinked leaf, a symlinked .abcd/config, a FIFO
// and an oversize file each refuse at once, none of them as "no launch
// payload", and the outside include list is never used.
func TestTheLaunchPayloadConfigIsReadOnlyInsideTheRepository(t *testing.T) {
	const outsideBody = `{"includes": ["README.md"]}`
	cases := map[string]func(t *testing.T, root, outside string){
		"a symlinked leaf": func(t *testing.T, root, outside string) {
			writeFile(t, outside, "launch-payload.json", outsideBody)
			mkdirAll(t, filepath.Join(root, ".abcd", "config"))
			symlink(t, filepath.Join(outside, "launch-payload.json"), filepath.Join(root, filepath.FromSlash(includeConfigRelPath)))
		},
		"a symlinked config directory": func(t *testing.T, root, outside string) {
			writeFile(t, outside, "launch-payload.json", outsideBody)
			mkdirAll(t, filepath.Join(root, ".abcd"))
			symlink(t, outside, filepath.Join(root, ".abcd", "config"))
		},
		"a FIFO": func(t *testing.T, root, outside string) {
			mkdirAll(t, filepath.Join(root, ".abcd", "config"))
			if err := syscall.Mkfifo(filepath.Join(root, filepath.FromSlash(includeConfigRelPath)), 0o600); err != nil {
				t.Skipf("mkfifo unavailable: %v", err)
			}
		},
		"an oversize file": func(t *testing.T, root, outside string) {
			writeFile(t, root, includeConfigRelPath, `{"includes": ["README.md"], "pad": "`+strings.Repeat("x", maxIncludeConfigBytes)+`"}`)
		},
	}
	for name, arrange := range cases {
		t.Run(name, func(t *testing.T) {
			root, outside := t.TempDir(), t.TempDir()
			arrange(t, root, outside)
			done := make(chan error, 1)
			go func() {
				_, err := LoadIncludes(root)
				done <- err
			}()
			select {
			case err := <-done:
				if err == nil {
					t.Fatal("the include config was read")
				}
				if errors.Is(err, ErrNoLaunchPayload) {
					t.Fatalf("refused as no launch payload: %v", err)
				}
			case <-time.After(5 * time.Second):
				t.Fatal("reading the include config did not return")
			}
		})
	}
}

// TestTheScriptsClosureIsReadOnlyInsideTheRepository: the pinned closure list
// is read on the same terms as the include config, and an absent one still
// means no scoping.
func TestTheScriptsClosureIsReadOnlyInsideTheRepository(t *testing.T) {
	root := t.TempDir()
	if set, err := defaultClosureFn(root); err != nil || set != nil {
		t.Fatalf("an absent closure list: %v, %v; want no scoping", set, err)
	}
	outside := t.TempDir()
	writeFile(t, outside, "scripts-closure.json", `{"closure": ["scripts/x.sh"]}`)
	mkdirAll(t, filepath.Join(root, ".abcd"))
	symlink(t, outside, filepath.Join(root, ".abcd", "config"))
	if set, err := defaultClosureFn(root); err == nil {
		t.Fatalf("the closure list behind a symlinked .abcd/config was read: %v", set)
	}
}

func mkdirAll(t *testing.T, dir string) {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
}

func symlink(t *testing.T, target, link string) {
	t.Helper()
	if err := os.Symlink(target, link); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
}
