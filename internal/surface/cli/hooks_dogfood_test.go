package cli

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/intentdriven/abcd/internal/abcdhome"
	"github.com/intentdriven/abcd/internal/core/dogfood"
	"github.com/intentdriven/abcd/internal/gittest"
)

// hooks_dogfood_test.go — the shims' dogfood rung (iss-2610070547057816). With
// a ~/.abcd.noindex/dogfood-roots list in the home, each shim asks the binary
// it resolved, through `hook dogfood-binary`, for a build to run in its place,
// and runs that build when one is named; with no list it asks nothing. A
// guard build that fails to run hands the command back to the installed
// release, out loud. The decision itself (listed, posing, source, trust) is
// internal/core/dogfood's and is tested there; these stubs stand in for it.

// dogfoodEvents is every hooks.json event with the verbs its shim runs.
var dogfoodEvents = stopWrapperEvents

// noopBootstrap provisions nothing, so SessionStart keeps the stub binary.
const noopBootstrap = "#!/bin/sh\nexit 0\n"

// dogfoodRig is a plugin root whose installed-release stub answers
// `hook dogfood-binary` with named, and a dogfood stub that records its calls
// and exits with code.
func dogfoodRig(t *testing.T, named string, code string) (root, build string) {
	t.Helper()
	root = hookRoot(t, noopBootstrap, false)
	build = filepath.Join(t.TempDir(), "abcd")
	if named == "BUILD" {
		named = build
	}
	release := "#!/bin/sh\n" +
		"printf '%s %s\\n' \"$1\" \"$2\" >> \"$ABCD_CALLS\"\n" +
		"if [ \"$1 $2\" = 'hook dogfood-binary' ]; then printf '%s\\n' '" + named + "'; exit 0; fi\n" +
		"cat >/dev/null\nexit 0\n"
	if err := os.WriteFile(filepath.Join(root, "abcd"), []byte(release), 0o755); err != nil {
		t.Fatal(err)
	}
	dog := "#!/bin/sh\ncat >/dev/null\nprintf 'dogfood %s %s\\n' \"$1\" \"$2\" >> \"$ABCD_CALLS\"\nexit " + code + "\n"
	if err := os.WriteFile(build, []byte(dog), 0o755); err != nil {
		t.Fatal(err)
	}
	return root, build
}

// dogfoodListHome is a home holding a dogfood list; the stubs decide what it
// names, so its contents do not matter here.
func dogfoodListHome(t *testing.T) string {
	t.Helper()
	home := sandboxHome(t)
	if err := os.MkdirAll(abcdhome.Path(home), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(abcdhome.Path(home, "dogfood-roots"), []byte("/nowhere\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	return home
}

// releaseRan reports whether the installed-release stub ran verb itself.
func releaseRan(calls, verb string) bool {
	for _, l := range strings.Split(calls, "\n") {
		if l == verb {
			return true
		}
	}
	return false
}

// TestHooksAskNothingWithoutADogfoodList: with no list in the home, no shim
// asks for a dogfood build, so a machine that has not opted in pays nothing.
func TestHooksAskNothingWithoutADogfoodList(t *testing.T) {
	for _, e := range dogfoodEvents {
		t.Run(e.event, func(t *testing.T) {
			root, _ := dogfoodRig(t, "BUILD", "0")
			_, stderr, _ := hookRunHome(t, e.event, root, "", t.TempDir(), sandboxHome(t))
			calls := callLog(t, filepath.Join(root, "calls.log"))
			if strings.Contains(calls, "dogfood") {
				t.Fatalf("with no list the shim asked for or ran a dogfood build; calls %q, stderr %q", calls, stderr)
			}
			for _, v := range e.verbs {
				if !releaseRan(calls, v) {
					t.Errorf("the installed release did not run %q; calls %q", v, calls)
				}
			}
		})
	}
}

// TestHooksRunTheDogfoodBuildTheyAreGiven: with a list, every shim asks the
// installed release which build to run, and runs the build it names for every
// verb, the installed release running none of them.
func TestHooksRunTheDogfoodBuildTheyAreGiven(t *testing.T) {
	for _, e := range dogfoodEvents {
		t.Run(e.event, func(t *testing.T) {
			root, _ := dogfoodRig(t, "BUILD", "0")
			_, stderr, code := hookRunHome(t, e.event, root, "", t.TempDir(), dogfoodListHome(t))
			calls := callLog(t, filepath.Join(root, "calls.log"))
			if code != 0 {
				t.Fatalf("exit %d; stderr %q; calls %q", code, stderr, calls)
			}
			if !strings.Contains(calls, "hook dogfood-binary") {
				t.Fatalf("the shim did not ask for a dogfood build; calls %q", calls)
			}
			for _, v := range e.verbs {
				if !strings.Contains(calls, "dogfood "+v) || releaseRan(calls, v) {
					t.Errorf("%q did not run on the dogfood build alone; calls %q", v, calls)
				}
			}
		})
	}
}

// TestHooksRunTheReleaseWhenNoBuildIsNamed: when the installed release names
// no build, or names something that is not an absolute path to an executable
// file, the installed release runs every verb.
func TestHooksRunTheReleaseWhenNoBuildIsNamed(t *testing.T) {
	for _, named := range []string{"", "relative/abcd", "/nonexistent/abcd"} {
		for _, e := range dogfoodEvents {
			t.Run(e.event+"/"+named, func(t *testing.T) {
				root, _ := dogfoodRig(t, named, "0")
				_, stderr, _ := hookRunHome(t, e.event, root, "", t.TempDir(), dogfoodListHome(t))
				calls := callLog(t, filepath.Join(root, "calls.log"))
				for _, v := range e.verbs {
					if !releaseRan(calls, v) || strings.Contains(calls, "dogfood "+v) {
						t.Errorf("%q did not run on the installed release; calls %q, stderr %q", v, calls, stderr)
					}
				}
			})
		}
	}
}

// TestHooksFallBackWhenTheDogfoodBuildFailsToRun: a dogfood build that dies
// on an event other than the guard's (any exit but 0 and 2, which are its
// answers) never leaves the event unhandled: the shim says so on stderr and
// the installed release runs every verb the event runs. A build that answers
// 2 is taken at its word, with no second run.
func TestHooksFallBackWhenTheDogfoodBuildFailsToRun(t *testing.T) {
	for _, e := range dogfoodEvents {
		if e.event == "PreToolUse" {
			continue // the guard's own fall-back is the test below
		}
		for _, died := range []string{"1", "137"} {
			t.Run(e.event+"/exit-"+died, func(t *testing.T) {
				root, _ := dogfoodRig(t, "BUILD", died)
				_, stderr, code := hookRunHome(t, e.event, root, "", t.TempDir(), dogfoodListHome(t))
				calls := callLog(t, filepath.Join(root, "calls.log"))
				if code != 0 || !strings.Contains(stderr, "FAILED TO RUN") {
					t.Fatalf("exit %d, stderr %q; want 0 and the failure said out loud", code, stderr)
				}
				for _, v := range e.verbs {
					if !strings.Contains(calls, "dogfood "+v) || strings.Count("\n"+calls+"\n", "\n"+v+"\n") != 1 {
						t.Errorf("%q did not fall back to the installed release exactly once; calls %q", v, calls)
					}
				}
			})
		}
		t.Run(e.event+"/answers-2", func(t *testing.T) {
			root, _ := dogfoodRig(t, "BUILD", "2")
			_, stderr, _ := hookRunHome(t, e.event, root, "", t.TempDir(), dogfoodListHome(t))
			calls := callLog(t, filepath.Join(root, "calls.log"))
			for _, v := range e.verbs {
				if releaseRan(calls, v) || strings.Contains(stderr, "FAILED TO RUN") {
					t.Errorf("an exit 2 from the build was run again on the release; calls %q, stderr %q", calls, stderr)
				}
			}
		})
	}
}

// TestGuardFallsBackWhenTheDogfoodBuildFailsToRun: a dogfood guard that dies
// (any exit but the guard's own 0, 1 and 2) never leaves the command
// unguarded: the shim says so on stderr and the installed release judges the
// same command. A dogfood verdict (exit 2, block) stands as given.
func TestGuardFallsBackWhenTheDogfoodBuildFailsToRun(t *testing.T) {
	t.Run("crash", func(t *testing.T) {
		root, _ := dogfoodRig(t, "BUILD", "139")
		_, stderr, code := hookRunHome(t, "PreToolUse", root, "", t.TempDir(), dogfoodListHome(t))
		calls := callLog(t, filepath.Join(root, "calls.log"))
		if !strings.Contains(calls, "dogfood guard hook") || !releaseRan(calls, "guard hook") {
			t.Fatalf("the installed release did not judge the command after the dogfood guard died; calls %q", calls)
		}
		if code != 0 || !strings.Contains(stderr, "dogfood build FAILED TO RUN (exit 139)") || !strings.Contains(stderr, "installed release") {
			t.Fatalf("exit %d, stderr %q; want the installed release's verdict and a line saying the dogfood guard failed", code, stderr)
		}
	})
	t.Run("block", func(t *testing.T) {
		root, _ := dogfoodRig(t, "BUILD", "2")
		_, _, code := hookRunHome(t, "PreToolUse", root, "", t.TempDir(), dogfoodListHome(t))
		calls := callLog(t, filepath.Join(root, "calls.log"))
		if code != 2 || releaseRan(calls, "guard hook") {
			t.Fatalf("exit %d, calls %q; want the dogfood guard's block to stand", code, calls)
		}
	})
}

// dogfoodCLIFixture is a clone of an upstream, with HOME pointed at a home
// holding a list whose line is entry(checkout, upstream): "" names another
// path. It returns the home, the clone and the upstream.
func dogfoodCLIFixture(t *testing.T, entry func(checkout, upstream string) string) (home, checkout, upstream string) {
	t.Helper()
	g := func(dir string, args ...string) string {
		t.Helper()
		cmd := exec.Command("git", append([]string{"-c", "init.defaultBranch=main"}, args...)...)
		cmd.Dir = dir
		cmd.Env = append(gittest.Env(t), "GIT_AUTHOR_NAME=t", "GIT_AUTHOR_EMAIL=t@example.invalid",
			"GIT_COMMITTER_NAME=t", "GIT_COMMITTER_EMAIL=t@example.invalid")
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
		return strings.TrimSpace(string(out))
	}
	resolved := func() string {
		d, err := filepath.EvalSymlinks(t.TempDir())
		if err != nil {
			t.Fatal(err)
		}
		return d
	}
	up := resolved()
	g(up, "init", "-q")
	if err := os.WriteFile(filepath.Join(up, "go.mod"), []byte("module github.com/intentdriven/abcd\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	g(up, "add", ".")
	g(up, "commit", "-q", "-m", "first")
	parent := resolved()
	g(parent, "clone", "-q", up, "checkout")
	checkout = filepath.Join(parent, "checkout")
	home = resolved()
	if err := os.MkdirAll(abcdhome.Path(home), 0o700); err != nil {
		t.Fatal(err)
	}
	line := entry(checkout, up)
	if line == "" {
		line = "/nowhere"
	}
	if err := os.WriteFile(abcdhome.Path(home, "dogfood-roots"), []byte(line+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("HOME", home)
	t.Chdir(checkout)
	return home, checkout, up
}

// dogfoodForgeURL is the reviewed remote a listed line declares: an https
// URL, as the builder accepts, that never resolves. The verb under test reads
// nothing over the network, and a path to the fixture's upstream would be
// refused as a remote this account can write.
const dogfoodForgeURL = "https://forge.example.invalid/intentdriven/abcd.git"

func unlisted(string, string) string         { return "" }
func listedWithRemote(c, _ string) string    { return c + "\t" + dogfoodForgeURL + "\tmain" }
func listedWithoutRemote(c, _ string) string { return c }

// swapSpawn records the build requests `hook dogfood-binary` makes.
func swapSpawn(t *testing.T) *[]string {
	t.Helper()
	var asked []string
	orig := spawnDogfoodBuild
	spawnDogfoodBuild = func(dir string) error { asked = append(asked, dir); return nil }
	t.Cleanup(func() { spawnDogfoodBuild = orig })
	return &asked
}

// TestDogfoodBinaryVerb: in an unlisted checkout the verb prints nothing,
// says nothing and starts nothing; in a listed one with no verified build it
// prints nothing, says the check and build are starting and starts them in the
// background; once the store records a verified tip with its build, it prints
// the build's path and starts nothing; and a line naming no remote starts
// nothing and says why. Every outcome exits 0, so a shim always has the
// installed release.
func TestDogfoodBinaryVerb(t *testing.T) {
	t.Run("unlisted", func(t *testing.T) {
		dogfoodCLIFixture(t, unlisted)
		asked := swapSpawn(t)
		var out, errOut strings.Builder
		if code := Run([]string{"hook", "dogfood-binary"}, &out, &errOut); code != 0 || out.Len() != 0 || errOut.Len() != 0 || len(*asked) != 0 {
			t.Fatalf("exit %d, stdout %q, stderr %q, builds %v; want silence and no build", code, out.String(), errOut.String(), *asked)
		}
	})
	t.Run("listed without a remote", func(t *testing.T) {
		dogfoodCLIFixture(t, listedWithoutRemote)
		asked := swapSpawn(t)
		var out, errOut strings.Builder
		if code := Run([]string{"hook", "dogfood-binary"}, &out, &errOut); code != 0 || out.Len() != 0 || len(*asked) != 0 {
			t.Fatalf("exit %d, stdout %q, builds %v; want no build named and none started", code, out.String(), *asked)
		}
		if !strings.Contains(errOut.String(), "names no remote") {
			t.Fatalf("stderr %q does not say the line names no remote", errOut.String())
		}
	})
	t.Run("listed", func(t *testing.T) {
		home, checkout, _ := dogfoodCLIFixture(t, listedWithRemote)
		asked := swapSpawn(t)
		var out, errOut strings.Builder
		if code := Run([]string{"hook", "dogfood-binary"}, &out, &errOut); code != 0 || out.Len() != 0 || len(*asked) != 1 {
			t.Fatalf("exit %d, stdout %q, builds %v; want no build named and one started", code, out.String(), *asked)
		}
		if !strings.Contains(errOut.String(), "building it in the background") {
			t.Fatalf("stderr %q does not say the build is starting", errOut.String())
		}
		plan := dogfood.Resolve(home, checkout)
		if plan.Checkout == nil {
			t.Fatalf("Resolve = %+v", plan)
		}
		tip := strings.TrimSpace(gitOut(t, checkout, "rev-parse", "origin/main"))
		store := abcdhome.Path(home, "dogfood", plan.Checkout.RootSHA)
		dir := filepath.Join(store, tip)
		if err := os.MkdirAll(dir, 0o700); err != nil {
			t.Fatal(err)
		}
		for _, d := range []string{abcdhome.Path(home, "dogfood"), store} {
			if err := os.Chmod(d, 0o700); err != nil {
				t.Fatal(err)
			}
		}
		bin := filepath.Join(dir, dogfood.BinaryName)
		if err := os.WriteFile(bin, []byte("#!/bin/sh\n"), 0o700); err != nil {
			t.Fatal(err)
		}
		rec := fmt.Sprintf(`{"commit":%q,"url":%q,"branch":"main","tracking":%q,"checked":%q}`,
			tip, dogfoodForgeURL, tip, time.Now().UTC().Format(time.RFC3339Nano))
		if err := os.WriteFile(filepath.Join(store, dogfood.TipName(dogfoodForgeURL, "main")), []byte(rec), 0o600); err != nil {
			t.Fatal(err)
		}
		out.Reset()
		errOut.Reset()
		if code := Run([]string{"hook", "dogfood-binary"}, &out, &errOut); code != 0 || strings.TrimSpace(out.String()) != bin || errOut.Len() != 0 || len(*asked) != 1 {
			t.Fatalf("exit %d, stdout %q, stderr %q, builds %v; want the build named, silence, and no new build", code, out.String(), errOut.String(), *asked)
		}
	})
}

// gitOut runs git in dir under the test's isolated environment.
func gitOut(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	cmd.Env = gittest.Env(t)
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("git %v: %v", args, err)
	}
	return string(out)
}

// guardRig is a plugin root whose installed-release guard writes relOut and
// exits relCode, and a dogfood build whose guard writes dogOut and exits
// dogCode, each recording its call.
func guardRig(t *testing.T, relOut, relCode, dogOut, dogCode string) string {
	t.Helper()
	root := hookRoot(t, noopBootstrap, false)
	build := filepath.Join(t.TempDir(), "abcd")
	release := "#!/bin/sh\n" +
		"printf '%s %s\\n' \"$1\" \"$2\" >> \"$ABCD_CALLS\"\n" +
		"if [ \"$1 $2\" = 'hook dogfood-binary' ]; then printf '%s\\n' '" + build + "'; exit 0; fi\n" +
		"cat >/dev/null\nprintf '%s' '" + relOut + "'\nexit " + relCode + "\n"
	dog := "#!/bin/sh\ncat >/dev/null\nprintf 'dogfood %s %s\\n' \"$1\" \"$2\" >> \"$ABCD_CALLS\"\n" +
		"printf '%s' '" + dogOut + "'\nexit " + dogCode + "\n"
	if err := os.WriteFile(filepath.Join(root, "abcd"), []byte(release), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(build, []byte(dog), 0o755); err != nil {
		t.Fatal(err)
	}
	return root
}

// TestDogfoodGuardVerdicts: the guard's answers are an allow (exit 0, nothing
// on stdout), a deny (exit 0 with the host's deny on stdout, or exit 2), and
// exit 1, which is a warn OR "NOT CHECKED — runs UNGUARDED". A dogfood allow
// or deny stands. A dogfood exit 1 is not trusted to be a verdict: the
// installed release judges the same command, and its deny stands, so a
// dogfood guard that fails open never silences the release.
func TestDogfoodGuardVerdicts(t *testing.T) {
	const deny = `{"hookSpecificOutput":{"permissionDecision":"deny"}}`
	cases := []struct {
		name                             string
		relOut, relCode, dogOut, dogCode string
		wantOut                          string
		wantCode                         int
		wantRelease                      bool
	}{
		{"dogfood allows", "", "0", "", "0", "", 0, false},
		{"dogfood denies on stdout", "", "0", deny, "0", deny, 0, false},
		{"dogfood blocks at exit 2", "", "0", "", "2", "", 2, false},
		{"dogfood not checked, release denies", deny, "0", "", "1", deny, 0, true},
		{"dogfood not checked, release blocks at exit 2", "", "2", "", "1", "", 2, true},
		{"dogfood not checked, release allows", "", "0", "", "1", "", 1, true},
		{"dogfood not checked, release warns", "", "1", "", "1", "", 1, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			root := guardRig(t, c.relOut, c.relCode, c.dogOut, c.dogCode)
			stdout, stderr, code := hookRunHome(t, "PreToolUse", root, "", t.TempDir(), dogfoodListHome(t))
			calls := callLog(t, filepath.Join(root, "calls.log"))
			if code != c.wantCode || strings.TrimSpace(stdout) != c.wantOut {
				t.Fatalf("exit %d, stdout %q; want exit %d, stdout %q (stderr %q, calls %q)", code, stdout, c.wantCode, c.wantOut, stderr, calls)
			}
			if !strings.Contains(calls, "dogfood guard hook") || releaseRan(calls, "guard hook") != c.wantRelease {
				t.Fatalf("calls %q; want the dogfood guard asked, and the release asked: %v", calls, c.wantRelease)
			}
		})
	}
}
