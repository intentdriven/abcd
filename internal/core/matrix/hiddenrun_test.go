package matrix

import (
	"context"
	"errors"
	"io/fs"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/intentdriven/abcd/internal/gittest"
)

// abcd runs a code copy's hidden tests itself (spc-2610161949559050 step 10):
// the tests placed at the run's start outside every clone are read back by
// content and run with the declared command in an export of the copy, never
// in its clone, as one owned process group bounded by its deadline.

// hiddenRun is a copy's clone with its task's hidden tests placed, and the
// inputs that run them.
type hiddenRun struct {
	src   *gittest.Repo
	store string
	clone string
	in    hiddenRunInputs
}

func newHiddenRun(t *testing.T, edit func(clone string)) hiddenRun {
	t.Helper()
	if _, err := os.Stat("/bin/sh"); err != nil {
		t.Skip("no /bin/sh")
	}
	src, frozen := cloneSource(t)
	store := filepath.Join(t.TempDir(), "matrix-store")
	key := TrialKey{Task: "widget", Cell: "cheap", Repeat: 1}
	clone, err := makeClone(src.Root(), store, cloneRun, frozen, key, keyAlias(key))
	if err != nil {
		t.Fatal(err)
	}
	refs, err := placeHiddenTests(store, cloneRun, "widget", widgetHidden)
	if err != nil {
		t.Fatal(err)
	}
	if edit != nil {
		edit(clone)
	}
	digest, why, err := copyTreeDigest(context.Background(), clone)
	if err != nil {
		t.Fatal(err)
	}
	if why != "" {
		// A copy that cannot be exported whole is recorded under any digest;
		// the run says why it is unexamined before it compares one.
		digest = strings.Repeat("0", 64)
	}
	return hiddenRun{src: src, store: store, clone: clone, in: hiddenRunInputs{
		RepoRoot: src.Root(), Store: store, Run: cloneRun, Task: "widget", Tests: refs,
		Clone: clone, TreeDigest: digest, Deadline: time.Minute,
	}}
}

// sh is the test command running script under /bin/sh, with args as $1....
func sh(script string, args ...string) []string {
	return append([]string{"/bin/sh", "-c", script, "sh"}, args...)
}

// treeOf lists every path under dir with its content, .git aside.
func treeOf(t *testing.T, dir string) map[string]string {
	t.Helper()
	out := map[string]string{}
	err := filepath.WalkDir(dir, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(dir, p)
		if d.IsDir() && d.Name() == ".git" {
			return filepath.SkipDir
		}
		if d.Type().IsRegular() {
			data, err := os.ReadFile(p)
			if err != nil {
				return err
			}
			out[rel] = string(data)
		} else {
			out[rel] = d.Type().String()
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return out
}

func realPath(t *testing.T, p string) string {
	t.Helper()
	r, err := filepath.EvalSymlinks(p)
	if err != nil {
		t.Fatal(err)
	}
	return r
}

// The run happens in an export of the copy with the hidden tests added: the
// command sees the copy's files and the hidden tests, its directory is
// outside the clone and inside the clone store's exports, the clone is left
// exactly as it was, and the export is gone once the run ends. The hidden
// tests were placed outside every clone under their digest name.
func TestHiddenTestsRunInAnExportOfTheCopyNeverInItsClone(t *testing.T) {
	hr := newHiddenRun(t, func(clone string) {
		if err := os.WriteFile(filepath.Join(clone, "widget.go"), []byte("package widget\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	})
	dir := hiddenDir(hr.store, cloneRun, "widget")
	if base := filepath.Base(dir); base != hiddenName(cloneRun, "widget") || !strings.HasPrefix(base, "tests_") || strings.Contains(base, "widget") {
		t.Fatalf("the hidden tests sit under %q; want their digest name", base)
	}
	if within(dir, hr.clone) || within(hr.clone, dir) {
		t.Fatalf("the hidden tests and the clone nest")
	}
	before := treeOf(t, hr.clone)
	where := filepath.Join(t.TempDir(), "where")
	hr.in.Command = sh(`pwd -P > "$1" &&
		grep -q "a case no builder saw" widget/testdata/hidden-case.txt &&
		grep -q TestHidden widget/hidden_test.go &&
		test -f README.md && test -f widget.go && test ! -e .git`, where)
	out, err := runHiddenTests(context.Background(), hr.in)
	if err != nil {
		t.Fatal(err)
	}
	if out.Status != StatusPassed || out.Score == nil || *out.Score != (Score{Passed: 1, Total: 1, Detail: AbcdScoreDetail}) {
		t.Fatalf("outcome %+v; want passed, scored 1 of 1 by abcd's own run", out)
	}
	data, err := os.ReadFile(where)
	if err != nil {
		t.Fatal(err)
	}
	ran := strings.TrimSpace(string(data))
	if within(ran, realPath(t, hr.clone)) {
		t.Fatalf("the tests ran in the copy's clone (%s)", ran)
	}
	if !within(ran, filepath.Join(realPath(t, hr.store), exportsDir, cloneRun)) {
		t.Fatalf("the tests ran at %s, not in an export in the clone store", ran)
	}
	if after := treeOf(t, hr.clone); !maps.Equal(before, after) {
		t.Fatalf("the run changed the copy's clone:\nbefore %v\nafter  %v", before, after)
	}
	for _, h := range widgetHidden {
		if _, err := os.Lstat(filepath.Join(hr.clone, filepath.FromSlash(h.Path))); !os.IsNotExist(err) {
			t.Errorf("the clone holds hidden test %s after the run: %v", h.Path, err)
		}
	}
	if _, err := os.Lstat(ran); !os.IsNotExist(err) {
		t.Errorf("the export is left after the run: %v", err)
	}
	if err := hiddenOutsideClone(hr.clone, dir, hr.in.Tests); err != nil {
		t.Errorf("after the run the clone holds a hidden test: %v", err)
	}
}

// A copy that left its own file at a hidden test's path is run against the
// hidden test, not its own, and its clone keeps its own file.
func TestACopysFileAtAHiddenTestsPathIsReplacedInTheExport(t *testing.T) {
	own := "a case the builder wrote to pass\n"
	hr := newHiddenRun(t, func(clone string) {
		p := filepath.Join(clone, "widget", "testdata", "hidden-case.txt")
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(own), 0o644); err != nil {
			t.Fatal(err)
		}
	})
	hr.in.Command = sh(`grep -q "a case no builder saw" widget/testdata/hidden-case.txt`)
	out, err := runHiddenTests(context.Background(), hr.in)
	if err != nil || out.Status != StatusPassed {
		t.Fatalf("outcome %+v, %v; want the hidden test run in place of the copy's", out, err)
	}
	if data, _ := os.ReadFile(filepath.Join(hr.clone, "widget", "testdata", "hidden-case.txt")); string(data) != own {
		t.Errorf("the clone's own file became %q", data)
	}
}

// The command's exit status is the copy's pass or fail: 0 passes, anything
// else fails, each scored as abcd's run of one command.
func TestTheTestCommandsExitStatusIsThePassOrFail(t *testing.T) {
	for _, tc := range []struct {
		script string
		status TrialStatus
		passed int
		reason string
	}{
		{"exit 0", StatusPassed, 1, "exited 0"},
		{"exit 3", StatusFailed, 0, "exited 3"},
		{"kill -9 $$", StatusFailed, 0, "signal"},
	} {
		hr := newHiddenRun(t, nil)
		hr.in.Command = sh(tc.script)
		out, err := runHiddenTests(context.Background(), hr.in)
		if err != nil {
			t.Fatalf("%s: %v", tc.script, err)
		}
		if out.Status != tc.status || out.Score == nil || out.Score.Passed != tc.passed || out.Score.Total != 1 ||
			out.Score.Detail != AbcdScoreDetail || !strings.Contains(out.Reason, tc.reason) {
			t.Errorf("%s: outcome %+v; want %s, %d of 1, a reason naming %q", tc.script, out, tc.status, tc.passed, tc.reason)
		}
	}
}

// A run that does not end by its deadline is unexamined, never failed: no
// score, a reason naming the deadline, and the call returns at the deadline.
func TestATestRunPastItsDeadlineIsUnexaminedNeverFailed(t *testing.T) {
	hr := newHiddenRun(t, nil)
	hr.in.Command = sh(`echo started; sleep 60`)
	hr.in.Deadline = 500 * time.Millisecond
	start := time.Now()
	out, err := runHiddenTests(context.Background(), hr.in)
	if err != nil {
		t.Fatal(err)
	}
	if took := time.Since(start); took > 20*time.Second {
		t.Errorf("returned after %s; the deadline was %s", took, hr.in.Deadline)
	}
	if out.Status != StatusUnexamined || out.Score != nil || !strings.Contains(out.Reason, "deadline") {
		t.Fatalf("outcome %+v; want unexamined with no score, naming the deadline", out)
	}
	if _, err := os.Stat(filepath.Join(hr.store, exportsDir, cloneRun)); err == nil {
		if ents, _ := os.ReadDir(filepath.Join(hr.store, exportsDir, cloneRun)); len(ents) > 0 {
			t.Errorf("the export is left after a timed-out run: %v", ents)
		}
	}
}

// A stop (the caller's context ending) is the run's error, never the copy's
// score.
func TestAStoppedTestRunIsTheRunsErrorNeverTheCopysScore(t *testing.T) {
	hr := newHiddenRun(t, nil)
	hr.in.Command = sh(`sleep 60`)
	ctx, cancel := context.WithCancel(context.Background())
	time.AfterFunc(300*time.Millisecond, cancel)
	out, err := runHiddenTests(ctx, hr.in)
	if err == nil || !strings.Contains(err.Error(), "stopped") || out.Status != "" || out.Score != nil {
		t.Fatalf("a stopped run gave %+v, %v; want the run's error and no outcome", out, err)
	}
}

// The test run's environment is an allowlist: no credential abcd was started
// with reaches it, a proxy keeps its host but not its credentials, and its
// temporary directory is the export's own.
func TestTheTestRunsEnvironmentCarriesNoCredential(t *testing.T) {
	secrets := map[string]string{
		"OPENROUTER_API_KEY":    "sk-or-v1-" + strings.Repeat("0", 64),
		"ANTHROPIC_API_KEY":     "sk-ant-api03-not-a-real-key",
		"AWS_SECRET_ACCESS_KEY": "not/a/real/aws/secret/key/0000000000",
		"GH_TOKEN":              "ghp_notarealtoken000000000000000000000000",
		"SSH_AUTH_SOCK":         "/tmp/agent.sock",
		"GIT_ASKPASS":           "/bin/echo",
	}
	for k, v := range secrets {
		t.Setenv(k, v)
	}
	t.Setenv("HTTPS_PROXY", "http://who:hunter2@proxy.invalid:8080")
	t.Setenv("GOPROXY", "https://who:hunter2@mod.invalid,direct")
	hr := newHiddenRun(t, nil)
	envFile := filepath.Join(t.TempDir(), "env")
	hr.in.Command = sh(`env > "$1"; echo "tmp=$TMPDIR" >> "$1"; pwd -P >> "$1"`, envFile)
	if out, err := runHiddenTests(context.Background(), hr.in); err != nil || out.Status != StatusPassed {
		t.Fatalf("outcome %+v, %v", out, err)
	}
	data, err := os.ReadFile(envFile)
	if err != nil {
		t.Fatal(err)
	}
	env := string(data)
	for k, v := range secrets {
		if strings.Contains(env, k+"=") || strings.Contains(env, v) {
			t.Errorf("the test run's environment carries %s", k)
		}
	}
	if strings.Contains(env, "hunter2") {
		t.Errorf("a proxy's credentials reached the test run:\n%s", env)
	}
	if !strings.Contains(env, "HTTPS_PROXY=http://proxy.invalid:8080") || !strings.Contains(env, "GOPROXY=https://mod.invalid,direct") {
		t.Errorf("a proxy's host did not reach the test run:\n%s", env)
	}
	if !strings.Contains(env, "PATH=") {
		t.Errorf("the test run has no PATH:\n%s", env)
	}
	lines := strings.Split(strings.TrimSpace(env), "\n")
	ran := lines[len(lines)-1]
	tmp := ""
	for _, l := range lines {
		if v, ok := strings.CutPrefix(l, "tmp="); ok {
			tmp = v
		}
	}
	if filepath.Base(tmp) != "tmp" || filepath.Base(ran) != "tree" || filepath.Base(filepath.Dir(tmp)) != filepath.Base(filepath.Dir(ran)) {
		t.Errorf("the test run's TMPDIR %q is not the export's own beside %q", tmp, ran)
	}
}

// What the command printed is screened before it is kept: where the copy ran
// and the account's home are withheld, a credential is sealed, and the rest
// is kept.
func TestTheTestOutputIsScreenedBeforeItIsKept(t *testing.T) {
	hr := newHiddenRun(t, nil)
	token := "ghp_" + strings.Repeat("A1b2C3d4E5", 3) + "F6g7h8"
	home, _ := os.UserHomeDir()
	hr.in.Command = sh(`echo "--- FAIL: TestHidden"; echo "token $1"; echo "ran in $(pwd -P) and $PWD"; echo "clone $2"; echo "home $3/x"; exit 1`,
		token, hr.clone, home)
	out, err := runHiddenTests(context.Background(), hr.in)
	if err != nil || out.Status != StatusFailed {
		t.Fatalf("outcome %+v, %v", out, err)
	}
	if !strings.Contains(out.Output, "--- FAIL: TestHidden") {
		t.Fatalf("the screened output lost what the tests said: %q", out.Output)
	}
	for what, v := range map[string]string{"the token": token, "the clone": hr.clone, "the clone's real path": realPath(t, hr.clone),
		"the store": hr.store, "the store's real path": realPath(t, hr.store), "the home": home + "/x"} {
		if v != "" && strings.Contains(out.Output, v) {
			t.Errorf("the kept output carries %s: %q", what, out.Output)
		}
	}
	if strings.Contains(out.Reason, hr.store) || strings.Contains(out.Reason, home) {
		t.Errorf("the reason names a place: %q", out.Reason)
	}
}

// Hidden tests that are not what the run recorded are refused with nothing
// run: a test changed, a file the run did not place, a test missing.
func TestHiddenTestsNotAsTheRunRecordedThemAreRefusedWithNothingRun(t *testing.T) {
	for name, tamper := range map[string]func(t *testing.T, dir string){
		"changed": func(t *testing.T, dir string) {
			p := filepath.Join(dir, "widget", "hidden_test.go")
			_ = os.Chmod(p, 0o600)
			if err := os.WriteFile(p, []byte("package widget\n"), 0o600); err != nil {
				t.Fatal(err)
			}
		},
		"added": func(t *testing.T, dir string) {
			if err := os.WriteFile(filepath.Join(dir, "widget", "extra_test.go"), []byte("package widget\n"), 0o600); err != nil {
				t.Fatal(err)
			}
		},
		"missing": func(t *testing.T, dir string) {
			if err := os.Remove(filepath.Join(dir, "widget", "testdata", "hidden-case.txt")); err != nil {
				t.Fatal(err)
			}
		},
		"a link": func(t *testing.T, dir string) {
			p := filepath.Join(dir, "widget", "testdata", "hidden-case.txt")
			other := filepath.Join(t.TempDir(), "case")
			data, _ := os.ReadFile(p)
			if err := os.WriteFile(other, data, 0o600); err != nil {
				t.Fatal(err)
			}
			if err := os.Remove(p); err != nil {
				t.Fatal(err)
			}
			if err := os.Symlink(other, p); err != nil {
				t.Fatal(err)
			}
		},
	} {
		t.Run(name, func(t *testing.T) {
			hr := newHiddenRun(t, nil)
			dir := hiddenDir(hr.store, cloneRun, "widget")
			_ = filepath.WalkDir(dir, func(p string, d fs.DirEntry, err error) error {
				if err == nil && d.IsDir() {
					_ = os.Chmod(p, 0o700)
				}
				return nil
			})
			tamper(t, dir)
			marker := filepath.Join(t.TempDir(), "ran")
			hr.in.Command = sh(`touch "$1"`, marker)
			out, err := runHiddenTests(context.Background(), hr.in)
			if err == nil {
				t.Fatalf("hidden tests %s are run: %+v", name, out)
			}
			if _, serr := os.Stat(marker); serr == nil {
				t.Errorf("the command ran against hidden tests %s", name)
			}
			if strings.Contains(err.Error(), hr.store) {
				t.Errorf("the refusal names where the store sits: %v", err)
			}
		})
	}
}

// Hidden tests that sit inside the copy's clone are refused with nothing run.
func TestHiddenTestsInsideTheCloneAreRefused(t *testing.T) {
	hr := newHiddenRun(t, nil)
	hr.in.Clone = hr.store
	hr.in.Command = sh(`exit 0`)
	if out, err := runHiddenTests(context.Background(), hr.in); err == nil || !strings.Contains(err.Error(), "inside") {
		t.Fatalf("hidden tests inside the clone ran: %+v, %v", out, err)
	}
}

// An export carries a link only where it leads down its own tree; a copy
// holding one that is absolute or climbs is unexamined, with the tests unrun.
func TestAnExportCarriesOnlyALinkLeadingDownItsTree(t *testing.T) {
	hr := newHiddenRun(t, func(clone string) {
		if err := os.Symlink("README.md", filepath.Join(clone, "readme-link")); err != nil {
			t.Fatal(err)
		}
	})
	hr.in.Command = sh(`test -L readme-link && grep -q frozen readme-link`)
	if out, err := runHiddenTests(context.Background(), hr.in); err != nil || out.Status != StatusPassed {
		t.Fatalf("a link down the tree is not carried: %+v, %v", out, err)
	}
	for _, target := range []string{"../outside", "/etc/hosts", "widget/../../x"} {
		hr := newHiddenRun(t, func(clone string) {
			if err := os.Symlink(target, filepath.Join(clone, "escape")); err != nil {
				t.Fatal(err)
			}
		})
		marker := filepath.Join(t.TempDir(), "ran")
		hr.in.Command = sh(`touch "$1"`, marker)
		out, err := runHiddenTests(context.Background(), hr.in)
		if err != nil || out.Status != StatusUnexamined || out.Score != nil || !strings.Contains(out.Reason, "escape") {
			t.Errorf("a link to %s: outcome %+v, %v; want unexamined naming it", target, out, err)
		}
		if _, err := os.Stat(marker); err == nil {
			t.Errorf("a link to %s: the tests ran", target)
		}
	}
}

// The export is pinned to the copy that was made: a clone whose tree changed
// after its digest was recorded (another copy's run writing into it, say) is
// not the copy that was made, so its run is unexamined, naming the mismatch,
// and nothing runs.
func TestAnExportWhoseTreeIsNotTheRecordedCopyIsUnexamined(t *testing.T) {
	for name, edit := range map[string]func(t *testing.T, clone string){
		"a file added": func(t *testing.T, clone string) {
			if err := os.WriteFile(filepath.Join(clone, "planted.go"), []byte("package widget\n"), 0o644); err != nil {
				t.Fatal(err)
			}
		},
		"a file changed": func(t *testing.T, clone string) {
			if err := os.WriteFile(filepath.Join(clone, "README.md"), []byte("not frozen\n"), 0o644); err != nil {
				t.Fatal(err)
			}
		},
		"a file made executable": func(t *testing.T, clone string) {
			if err := os.Chmod(filepath.Join(clone, "README.md"), 0o755); err != nil {
				t.Fatal(err)
			}
		},
	} {
		t.Run(name, func(t *testing.T) {
			hr := newHiddenRun(t, nil)
			edit(t, hr.clone)
			marker := filepath.Join(t.TempDir(), "ran")
			hr.in.Command = sh(`touch "$1"`, marker)
			out, err := runHiddenTests(context.Background(), hr.in)
			if err != nil || out.Status != StatusUnexamined || out.Score != nil || !strings.Contains(out.Reason, "recorded") {
				t.Fatalf("outcome %+v, %v; want unexamined naming the recorded tree", out, err)
			}
			if _, err := os.Stat(marker); err == nil {
				t.Errorf("the tests ran against a tree that is not the copy's")
			}
		})
	}
	hr := newHiddenRun(t, nil)
	again, why, err := copyTreeDigest(context.Background(), hr.clone)
	if err != nil || why != "" || again != hr.in.TreeDigest {
		t.Fatalf("an unchanged clone's digest moved: %s vs %s (%q, %v)", again, hr.in.TreeDigest, why, err)
	}
}

// An export that cannot be removed leaves the copy unexamined, its reason
// naming what was left under the clone store, never where the store sits.
func TestAnExportThatCannotBeRemovedLeavesTheCopyUnexamined(t *testing.T) {
	hr := newHiddenRun(t, nil)
	was := removeExport
	removeExport = func(string) error { return os.ErrPermission }
	t.Cleanup(func() { removeExport = was })
	hr.in.Command = sh(`exit 0`)
	out, err := runHiddenTests(context.Background(), hr.in)
	if err != nil || out.Status != StatusUnexamined || out.Score != nil ||
		!strings.Contains(out.Reason, "<clone store>/"+exportsDir+"/"+cloneRun+"/x_") {
		t.Fatalf("outcome %+v, %v; want unexamined naming the export left behind", out, err)
	}
	if strings.Contains(out.Reason, hr.store) || strings.Contains(out.Reason, realPath(t, hr.store)) {
		t.Errorf("the reason names where the store sits: %q", out.Reason)
	}
	removeExport = was
	_ = removeTree(filepath.Join(hr.store, exportsDir))
}

// A test command that cannot be started is the run's error, never the copy's
// failure; inputs that do not validate are refused.
func TestATestCommandThatCannotRunIsTheRunsError(t *testing.T) {
	hr := newHiddenRun(t, nil)
	hr.in.Command = []string{filepath.Join(t.TempDir(), "no-such-program")}
	if out, err := runHiddenTests(context.Background(), hr.in); err == nil || out.Score != nil {
		t.Fatalf("a missing program gave %+v, %v; want the run's error", out, err)
	}
	for name, edit := range map[string]func(*hiddenRunInputs){
		"no command":       func(in *hiddenRunInputs) { in.Command = nil },
		"an empty program": func(in *hiddenRunInputs) { in.Command = []string{""} },
		"no deadline":      func(in *hiddenRunInputs) { in.Deadline = 0 },
		"a long deadline":  func(in *hiddenRunInputs) { in.Deadline = maxHiddenDeadline + time.Second },
		"a relative store": func(in *hiddenRunInputs) { in.Store = "matrix-store" },
		"a relative clone": func(in *hiddenRunInputs) { in.Clone = "clone" },
		"no tests":         func(in *hiddenRunInputs) { in.Tests = nil },
		"no tree digest":   func(in *hiddenRunInputs) { in.TreeDigest = "" },
		"a bad run":        func(in *hiddenRunInputs) { in.Run = "run" },
		"a NUL":            func(in *hiddenRunInputs) { in.Command = []string{"/bin/sh", "-c", "a\x00b"} },
	} {
		in := hr.in
		in.Command = sh(`exit 0`)
		edit(&in)
		if _, err := runHiddenTests(context.Background(), in); err == nil {
			t.Errorf("%s: the run is not refused", name)
		}
	}
	if ents, _ := os.ReadDir(filepath.Join(hr.store, exportsDir, cloneRun)); len(ents) > 0 {
		t.Errorf("a refused run left an export: %v", ents)
	}
}

// The environment allowlist is a fixed list: no name it carries reads as a
// credential.
func TestTheTestEnvironmentAllowlistNamesNoCredential(t *testing.T) {
	for _, k := range append(slices.Clone(testEnvNames), testProxyNames...) {
		for _, w := range []string{"TOKEN", "KEY", "SECRET", "PASS", "AUTH", "SOCK", "CRED"} {
			if strings.Contains(k, w) {
				t.Errorf("the test environment carries %s", k)
			}
		}
	}
	env := testEnv([]string{"PATH=/bin", "HTTPS_PROXY=http://a:b@[bad", "NOEQUALS", "GH_TOKEN=x"}, "/tmp/t")
	if !slices.Equal(env, []string{"PATH=/bin", "TMPDIR=/tmp/t"}) {
		t.Errorf("testEnv = %q", env)
	}
}

// A deadline that passes once the copy is exported and before the test
// command starts leaves the copy unexamined, as any run abcd did not see end
// does: never the run's error, and never a failure.
func TestADeadlinePassingBeforeTheTestStartsIsUnexamined(t *testing.T) {
	hr := newHiddenRun(t, nil)
	hr.in.Command = sh(`exit 1`)
	hr.in.Deadline = 200 * time.Millisecond
	was := beforeTestStart
	beforeTestStart = func() { time.Sleep(400 * time.Millisecond) }
	t.Cleanup(func() { beforeTestStart = was })
	out, err := runHiddenTests(context.Background(), hr.in)
	if err != nil {
		t.Fatalf("a deadline passed before the start gave the run's error %v; want the copy unexamined", err)
	}
	if out.Status != StatusUnexamined || out.Score != nil || !strings.Contains(out.Reason, "deadline") {
		t.Fatalf("outcome %+v; want unexamined with no score, naming the deadline", out)
	}
}

// A copy's tree read under a context that has ended gives no digest and says
// so as an error, never an empty digest with nothing wrong.
func TestACopyTreeDigestOnAnEndedContextIsAnError(t *testing.T) {
	hr := newHiddenRun(t, nil)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	digest, why, err := copyTreeDigest(ctx, hr.clone)
	if err == nil || digest != "" || why != "" {
		t.Fatalf("copyTreeDigest on an ended context = %q, %q, %v; want no digest and an error", digest, why, err)
	}
	if !errors.Is(err, context.Canceled) {
		t.Errorf("the error %v does not say the context ended", err)
	}
}
