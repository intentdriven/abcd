package cli

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/abcdhome"
	"github.com/intentdriven/abcd/internal/core/implement/loop"
	"github.com/intentdriven/abcd/internal/core/runner"
	"github.com/intentdriven/abcd/internal/gittest"
	"github.com/spf13/cobra"
)

// start_test.go is `abcd start` at the front door (spc-2611251757236507). No test starts
// the host: a script named claude stands on PATH, answers the login check, prints a canned
// event stream for the short launch, and keeps what it was handed.

// startConventionsText is the checkout's conventions file: the marker that makes it one
// abcd manages, and words a test looks for byte for byte.
const startConventionsText = "# Conventions\n\nCheck word: plainword.\n\n<!-- BEGIN ABCD -->\nx\n<!-- END ABCD -->\n"

// startCall is one hand-over to the host.
type startCall struct {
	path      string
	argv, env []string
	said      int
}

// startWorld is a managed checkout, a home, a stand-in host on PATH, a terminal on every
// stream, an environment holding no session marker and no variable of the host's, and a
// recorder in place of the exec.
type startWorld struct {
	repo, home, log string
	self            string
	out, errb       bytes.Buffer
	calls           []startCall
	execErr         error
}

// startInit is the first event of a launch that holds what the list names.
func startInit() map[string]any {
	return map[string]any{"type": "system", "subtype": "init", "session_id": "s", "model": "m",
		"tools":        []string{"Task", "TaskStop", "Bash", "Read", "Edit", "Write", "AskUserQuestion", "Monitor", "ScheduleWakeup", "CronCreate", "CronList", "CronDelete"},
		"apiKeySource": "none", "claude_code_version": "2.1.332", "mcp_servers": []any{},
		"plugins": []map[string]string{{"name": "cc-plugin-later", "path": "builtin", "source": "cc-plugin-later@builtin"}},
		"skills":  []string{"simplify"}, "agents": []string{"Explore", "Plan"}, "slash_commands": []string{"simplify", "clear"}}
}

// newStartWorld is a world a start goes through: one profile that is logged in, and a
// short launch that holds.
func newStartWorld(t *testing.T) *startWorld {
	t.Helper()
	for _, kv := range os.Environ() {
		name, _, _ := strings.Cut(kv, "=")
		if slices.Contains(driveSessionMarkers, name) || strings.HasPrefix(name, "CLAUDE_") || strings.HasPrefix(name, "ANTHROPIC_") ||
			strings.HasPrefix(name, "ABCD_STAGE_") || name == abcdhome.PersonHomeEnv || slices.Contains(runner.HeldCredentials(), name) {
			t.Setenv(name, "")
			if err := os.Unsetenv(name); err != nil {
				t.Fatal(err)
			}
		}
	}
	w := &startWorld{home: driveHome(t), log: t.TempDir()}
	t.Setenv("HOME", w.home)
	repo := t.TempDir()
	gitInitAt(t, repo)
	w.repo = realPath(t, repo)
	w.write(t, loop.ConventionsFile, startConventionsText)
	t.Chdir(w.repo)
	w.profiles(t, "walled")

	bin := t.TempDir()
	script := `#!/bin/sh
log='` + w.log + `'
if [ "$1" = auth ]; then
	echo "$*" >> "$log/calls"
	cat "$log/auth"
	exit 0
fi
echo print >> "$log/calls"
if [ -f "$log/during" ]; then /bin/sh "$log/during" > /dev/null 2>&1; fi
for a in "$@"; do printf '%s\0' "$a"; done > "$log/argv"
env > "$log/env"
cat "$log/stream"
if [ -f "$log/fails" ]; then exit 1; fi
`
	if err := os.WriteFile(filepath.Join(bin, "claude"), []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
	w.loggedIn(t, true)
	w.launch(t, startInit(), true)

	w.self = filepath.Join(t.TempDir(), "abcd")
	if err := os.WriteFile(w.self, []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	withTerminal(t)
	wasSelf, wasExec := startSelf, startExec
	t.Cleanup(func() { startSelf, startExec = wasSelf, wasExec })
	startSelf = func() string { return w.self }
	startExec = func(path string, argv, env []string) error {
		w.calls = append(w.calls, startCall{path, slices.Clone(argv), slices.Clone(env), w.out.Len()})
		return w.execErr
	}
	return w
}

// write lays a file in the checkout.
func (w *startWorld) write(t *testing.T, rel, body string) {
	t.Helper()
	path := filepath.Join(w.repo, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

// profiles leaves the account holding exactly the named profiles.
func (w *startWorld) profiles(t *testing.T, names ...string) {
	t.Helper()
	if err := os.RemoveAll(abcdhome.Path(w.home, "profiles")); err != nil {
		t.Fatal(err)
	}
	for _, n := range names {
		if err := os.MkdirAll(abcdhome.Path(w.home, "profiles", n), 0o700); err != nil {
			t.Fatal(err)
		}
	}
}

// loggedIn sets what the stand-in answers about the login.
func (w *startWorld) loggedIn(t *testing.T, in bool) {
	t.Helper()
	body := fmt.Sprintf(`{"loggedIn":%v,"email":"never-kept@example.invalid"}`, in)
	if err := os.WriteFile(filepath.Join(w.log, "auth"), []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
}

// launch sets the event stream the short launch answers with: its first event, none when
// init is nil, whether the guard's hook refused, and the result.
func (w *startWorld) launch(t *testing.T, init map[string]any, refused bool) {
	t.Helper()
	var b bytes.Buffer
	enc := json.NewEncoder(&b)
	if init != nil {
		_ = enc.Encode(init)
	}
	if refused {
		_ = enc.Encode(map[string]any{"type": "system", "subtype": "hook_response", "hook_event": "PreToolUse",
			"stdout": `{"hookSpecificOutput":{"permissionDecision":"deny","permissionDecisionReason":"Blocked by the abcd guard (git-push-force)"}}`})
	}
	_ = enc.Encode(map[string]any{"type": "result", "subtype": "success", "is_error": false, "result": "done", "session_id": "s"})
	if err := os.WriteFile(filepath.Join(w.log, "stream"), b.Bytes(), 0o600); err != nil {
		t.Fatal(err)
	}
}

// launchFails has the short launch end with a failure and no event.
func (w *startWorld) launchFails(t *testing.T) {
	t.Helper()
	for name, body := range map[string]string{"stream": "", "fails": "yes"} {
		if err := os.WriteFile(filepath.Join(w.log, name), []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
	}
}

// noHost leaves git alone on PATH.
func (w *startWorld) noHost(t *testing.T) (bin string) {
	t.Helper()
	git, err := exec.LookPath("git")
	if err != nil {
		t.Skip("no git on PATH")
	}
	bin = t.TempDir()
	if err := os.Symlink(git, filepath.Join(bin, "git")); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin)
	return bin
}

// hostCalls are what the stand-in was asked, in order.
func (w *startWorld) hostCalls() []string {
	raw, _ := os.ReadFile(filepath.Join(w.log, "calls"))
	return strings.Fields(strings.ReplaceAll(strings.TrimSpace(string(raw)), " ", "_"))
}

// run runs `abcd start` with args through the front door.
func (w *startWorld) run(args ...string) int {
	w.out.Reset()
	w.errb.Reset()
	return Run(append([]string{"start"}, args...), &w.out, &w.errb)
}

// opens runs a start that reaches the host, and returns the hand-over.
func (w *startWorld) opens(t *testing.T, args ...string) startCall {
	t.Helper()
	before := len(w.calls)
	if code := w.run(args...); code != 0 || len(w.calls) != before+1 {
		t.Fatalf("exit %d, %d hand-over(s), want 0 and one more than %d:\n%s%s", code, len(w.calls), before, w.out.String(), w.errb.String())
	}
	return w.calls[before]
}

// refuses runs a start that is refused with code, and returns the refusal.
func (w *startWorld) refuses(t *testing.T, code string, args ...string) string {
	t.Helper()
	before := len(w.calls)
	exit := w.run(args...)
	text := w.errb.String()
	if exit != 2 || !strings.Contains(text, "(abcd explain "+code+")") {
		t.Fatalf("exit %d, want 2 and %s:\n%s%s", exit, code, w.out.String(), text)
	}
	if len(w.calls) != before {
		t.Fatal("a refused start opens no session")
	}
	if !strings.HasPrefix(text, "abcd: abcd start: ") || !strings.Contains(text, "no session was started") {
		t.Fatalf("a refusal opens with the verb and says that nothing opened:\n%s", text)
	}
	return text
}

// guardCommitted is the shell line that commits body as the checkout's guard registry.
func (w *startWorld) guardCommitted(body string) string {
	return "cd '" + w.repo + "' && mkdir -p .abcd && printf '%s' '" + body + "' > .abcd/guard.json && git add .abcd/guard.json && " +
		"git -c user.name=t -c user.email=t@example.com -c commit.gpgsign=false commit -q -m guard\n"
}

// commitGuard commits body as the checkout's guard registry.
func (w *startWorld) commitGuard(t *testing.T, body string) {
	t.Helper()
	cmd := exec.Command("/bin/sh", "-c", w.guardCommitted(body))
	cmd.Env = gittest.Env(t)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("%v\n%s", err, out)
	}
}

// Two registries under which the guard does not refuse the proof's command.
const (
	startGuardOff    = `{"schema_version":1,"disabled":true}`
	startGuardWarned = `{"schema_version":1,"entries":{"git-push-force":{"tier":"warn"}}}`
)

// startRefusalWorlds sets up the world each of the verb's refusals is raised in
// (TestRaisedByIsWhatTheVerbsRaise): one command line, seven worlds.
var startRefusalWorlds = map[string]func(t *testing.T){
	"start/outside-a-terminal": func(t *testing.T) {
		newStartWorld(t)
		t.Setenv("CLAUDECODE", "1")
	},
	"start/profile-unusable":   func(t *testing.T) { newStartWorld(t).profiles(t, "alpha", "beta") },
	"start/repository-unready": func(t *testing.T) { newStartWorld(t).write(t, loop.ConventionsFile, "# Conventions\n") },
	"start/host-not-runnable":  func(t *testing.T) { newStartWorld(t).noHost(t) },
	"start/not-logged-in":      func(t *testing.T) { newStartWorld(t).loggedIn(t, false) },
	"start/no-verdict":         func(t *testing.T) { newStartWorld(t).launchFails(t) },
	"start/wall-not-held": func(t *testing.T) {
		init := startInit()
		init["mcp_servers"] = []map[string]string{{"name": "persons-own", "status": "connected"}}
		newStartWorld(t).launch(t, init, true)
	},
}

// argAfter is the argument following name.
func argAfter(argv []string, name string) string {
	if i := slices.Index(argv, name); i >= 0 && i+1 < len(argv) {
		return argv[i+1]
	}
	return ""
}

// startSettings is the settings object among the hand-over's arguments.
func startSettings(t *testing.T, argv []string) (hooks map[string][]struct {
	Matcher string `json:"matcher"`
	Hooks   []struct {
		Command string `json:"command"`
	} `json:"hooks"`
}) {
	t.Helper()
	var whole struct {
		Hooks json.RawMessage `json:"hooks"`
	}
	if err := json.Unmarshal([]byte(argAfter(argv, "--settings")), &whole); err != nil {
		t.Fatalf("the settings are one JSON object: %v", err)
	}
	if err := json.Unmarshal(whole.Hooks, &hooks); err != nil {
		t.Fatal(err)
	}
	return hooks
}

// profileTree is every file below dir by name, mode, size, modification time and content.
func profileTree(t *testing.T, dir string) map[string]string {
	t.Helper()
	tree := map[string]string{}
	err := filepath.WalkDir(dir, func(path string, _ fs.DirEntry, err error) error {
		if errors.Is(err, fs.ErrNotExist) {
			return nil
		}
		if err != nil {
			return err
		}
		fi, err := os.Lstat(path)
		if err != nil {
			return err
		}
		var sum [sha256.Size]byte
		if fi.Mode().IsRegular() {
			raw, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			sum = sha256.Sum256(raw)
		}
		tree[path] = fmt.Sprintf("%v %d %d %x", fi.Mode(), fi.Size(), fi.ModTime().UnixNano(), sum)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return tree
}

// sameProfileTree fails unless the two readings are the same.
func sameProfileTree(t *testing.T, what string, before, after map[string]string) {
	t.Helper()
	for path, was := range before {
		if now, held := after[path]; !held || now != was {
			t.Errorf("%s: %s was %q, is %q", what, path, was, now)
		}
	}
	for path := range after {
		if _, held := before[path]; !held {
			t.Errorf("%s: %s is new", what, path)
		}
	}
}

// TestStartRefusesFromAnAgentsShell: each marker a host sets for what runs inside its
// session, and nothing is made.
func TestStartRefusesFromAnAgentsShell(t *testing.T) {
	for _, marker := range []string{"CLAUDECODE", "CLAUDE_CODE_SESSION_ID", "CLAUDE_CODE_ENTRYPOINT"} {
		t.Run(marker, func(t *testing.T) {
			w := newStartWorld(t)
			w.profiles(t)
			t.Setenv(marker, "1")
			if text := w.refuses(t, "start/outside-a-terminal"); !strings.Contains(text, marker+" is set") {
				t.Fatalf("the refusal names the variable:\n%s", text)
			}
			for _, claim := range []string{"agent's own session", "only from"} {
				if strings.Contains(w.errb.String(), claim) {
					t.Errorf("the refusal claims %q:\n%s", claim, w.errb.String())
				}
			}
			for _, leaf := range []string{"profiles", "session-bin"} {
				if _, err := os.Lstat(abcdhome.Path(w.home, leaf)); !errors.Is(err, fs.ErrNotExist) {
					t.Errorf("%s is made: %v", leaf, err)
				}
			}
			if calls := w.hostCalls(); len(calls) != 0 {
				t.Errorf("the host was asked %q", calls)
			}
		})
	}
}

// TestStartRefusesUnlessAllThreeStreamsAreATerminal: each stream in turn.
func TestStartRefusesUnlessAllThreeStreamsAreATerminal(t *testing.T) {
	w := newStartWorld(t)
	for name, stream := range map[string]any{"input": os.Stdin, "output": &w.out, "error": &w.errb} {
		connectIsTerminal = func(s any) bool { return s != stream }
		if text := w.refuses(t, "start/outside-a-terminal"); !strings.Contains(text, "standard "+name+" ") {
			t.Errorf("the refusal names the %s stream:\n%s", name, text)
		}
	}
	if calls := w.hostCalls(); len(calls) != 0 {
		t.Errorf("the host was asked %q", calls)
	}
}

// TestStartChecksThoseTwoThingsAndNoMore: nothing else about where the command runs is
// read.
func TestStartChecksThoseTwoThingsAndNoMore(t *testing.T) {
	w := newStartWorld(t)
	t.Setenv("CI", "true")
	t.Setenv("TERM", "dumb")
	t.Setenv("SSH_CONNECTION", "192.0.2.1 1 192.0.2.2 22")
	w.opens(t)
}

// TestStartRefusesAModeOutsideTheSet: a usage refusal that repeats no value, whichever flag.
func TestStartRefusesAModeOutsideTheSet(t *testing.T) {
	w := newStartWorld(t)
	for _, args := range [][]string{{"--permission-mode", "zzzmode"}, {"--permission-mode", ""}, {"--permission-mode=Plan"}} {
		text := w.refuses(t, "usage/invalid-flag-value", args...)
		if strings.Contains(text, "zzzmode") || !strings.Contains(text, "acceptEdits") {
			t.Errorf("%q: the refusal lists the modes and repeats no value:\n%s", args, text)
		}
	}
	for _, name := range []string{"../zzzname", "", "zzz name"} {
		if text := w.refuses(t, "usage/invalid-flag-value", "--profile", name); strings.Contains(text, "zzz") {
			t.Errorf("the refusal repeats the name:\n%s", text)
		}
	}
	if exit := w.run("one", "two"); exit != 2 || !strings.Contains(w.errb.String(), "usage/extra-argument") {
		t.Errorf("a second positional is the framework's refusal: exit %d\n%s", exit, w.errb.String())
	}
	if calls := w.hostCalls(); len(calls) != 0 {
		t.Errorf("the host was asked %q", calls)
	}
}

// TestStartWithNoProfileMakesOneAndPrintsTheLoginCommand: the profile and the copy of abcd
// and nothing else, the login that is the person's, and no host run or opened.
func TestStartWithNoProfileMakesOneAndPrintsTheLoginCommand(t *testing.T) {
	w := newStartWorld(t)
	w.profiles(t)
	// The profile is the account's: the checkout is not looked at.
	t.Chdir(t.TempDir())
	if code := w.run(); code != 0 {
		t.Fatalf("exit %d:\n%s%s", code, w.out.String(), w.errb.String())
	}
	out := w.out.String()
	for _, want := range []string{
		"abcd start: this account held no profile, so one was made: " + abcdhome.Display("profiles", "drive") + "\n",
		"a separate login and settings on this one account, not a sandbox",
		"\n\n  CLAUDE_CONFIG_DIR=" + abcdhome.Display("profiles", "drive") + " claude auth login\n\n",
		"abcd start: no session was started; run abcd start again once the login is done (abcd drain --drive runs on the same profile)\n",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("the output lacks %q:\n%s", want, out)
		}
	}
	dir := abcdhome.Path(w.home, "profiles", "drive")
	raw, err := os.ReadFile(filepath.Join(dir, "settings.json"))
	if err != nil {
		t.Fatal(err)
	}
	self, _ := os.ReadFile(w.self)
	sum := sha256.Sum256(self)
	if copyAt := abcdhome.Path(w.home, "session-bin", hex.EncodeToString(sum[:]), "abcd"); !strings.Contains(string(raw), "'"+copyAt+"' guard hook") {
		t.Errorf("the profile's settings name the kept copy:\n%s", raw)
	}
	if calls := w.hostCalls(); len(calls) != 0 || len(w.calls) != 0 {
		t.Fatalf("the host was asked %q and opened %d time(s)", calls, len(w.calls))
	}
}

// TestStartMakesTheProfileUnderTheNameGiven: --profile names the first one.
func TestStartMakesTheProfileUnderTheNameGiven(t *testing.T) {
	w := newStartWorld(t)
	w.profiles(t)
	if code := w.run("--profile", "mine"); code != 0 || len(w.calls) != 0 {
		t.Fatalf("exit %d:\n%s%s", code, w.out.String(), w.errb.String())
	}
	names, err := runner.ProfileNames(w.home)
	if err != nil || !slices.Equal(names, []string{"mine"}) || !strings.Contains(w.out.String(), "CLAUDE_CONFIG_DIR="+abcdhome.Display("profiles", "mine")+" claude auth login") {
		t.Fatalf("the account holds %q %v:\n%s", names, err, w.out.String())
	}
}

// TestStartWithSeveralProfilesRefusesAndListsThem: it never chooses among them.
func TestStartWithSeveralProfilesRefusesAndListsThem(t *testing.T) {
	w := newStartWorld(t)
	w.profiles(t, "alpha", "beta")
	text := w.refuses(t, "start/profile-unusable")
	for _, want := range []string{"alpha", "beta", "--profile <name>"} {
		if !strings.Contains(text, want) {
			t.Errorf("the refusal does not name %q:\n%s", want, text)
		}
	}
	if calls := w.hostCalls(); len(calls) != 0 {
		t.Errorf("the host was asked %q", calls)
	}
}

// TestStartRefusesANameTheAccountDoesNotHold: a mistyped name makes no second login.
func TestStartRefusesANameTheAccountDoesNotHold(t *testing.T) {
	w := newStartWorld(t)
	w.profiles(t, "alpha", "beta")
	text := w.refuses(t, "start/profile-unusable", "--profile", "gamma")
	if !strings.Contains(text, "alpha, beta") || strings.Contains(text, "gamma") {
		t.Errorf("the refusal lists what is held and repeats no name:\n%s", text)
	}
	if names, _ := runner.ProfileNames(w.home); !slices.Equal(names, []string{"alpha", "beta"}) {
		t.Errorf("the account now holds %q", names)
	}
	if _, err := os.Lstat(abcdhome.Path(w.home, "session-bin")); !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("a copy is kept: %v", err)
	}
}

// TestStartRefusesAProfileThatHoldsWhatALaunchWouldLoad: the entry is named, an empty
// folder is not, and synced skills have two ways on.
func TestStartRefusesAProfileThatHoldsWhatALaunchWouldLoad(t *testing.T) {
	w := newStartWorld(t)
	dir := abcdhome.Path(w.home, "profiles", "walled")
	if err := os.Mkdir(filepath.Join(dir, "agents"), 0o700); err != nil {
		t.Fatal(err)
	}
	w.opens(t)
	if err := os.MkdirAll(filepath.Join(dir, "skills", "synced"), 0o700); err != nil {
		t.Fatal(err)
	}
	text := w.refuses(t, "start/profile-unusable")
	for _, want := range []string{"holds skills", "abcd drain --drive", "by hand"} {
		if !strings.Contains(text, want) {
			t.Errorf("the refusal does not name %q:\n%s", want, text)
		}
	}
	if _, err := os.Lstat(filepath.Join(dir, "skills", "synced")); err != nil {
		t.Errorf("a start moves nothing: %v", err)
	}
}

// TestStartRefusesOutsideAManagedCheckout: no checkout, one abcd does not manage, and one
// with no conventions file.
func TestStartRefusesOutsideAManagedCheckout(t *testing.T) {
	w := newStartWorld(t)
	t.Chdir(t.TempDir())
	if text := w.refuses(t, "start/repository-unready"); !strings.Contains(text, "not inside a git checkout") {
		t.Errorf("got\n%s", text)
	}
	t.Chdir(w.repo)
	w.write(t, loop.ConventionsFile, "# Conventions\n")
	if text := w.refuses(t, "start/repository-unready"); !strings.Contains(text, "does not manage") {
		t.Errorf("got\n%s", text)
	}
	w.write(t, "CLAUDE.md", startConventionsText)
	if err := os.Remove(filepath.Join(w.repo, loop.ConventionsFile)); err != nil {
		t.Fatal(err)
	}
	if text := w.refuses(t, "start/repository-unready"); !strings.Contains(text, "is absent") {
		t.Errorf("got\n%s", text)
	}
	if calls := w.hostCalls(); len(calls) != 0 {
		t.Errorf("the host was asked %q", calls)
	}
}

// TestStartRefusesAConventionsFileOverTheBound: the file is handed over whole or not at all.
func TestStartRefusesAConventionsFileOverTheBound(t *testing.T) {
	w := newStartWorld(t)
	w.write(t, "CLAUDE.md", startConventionsText)
	fits := startConventionsText + strings.Repeat("x", startConventionsBound-len(startConventionsText))
	w.write(t, loop.ConventionsFile, fits+"x")
	if text := w.refuses(t, "start/repository-unready"); !strings.Contains(text, "more than 102400 bytes") {
		t.Errorf("got\n%s", text)
	}
	w.write(t, loop.ConventionsFile, startConventionsText+"a\x00b")
	if text := w.refuses(t, "start/repository-unready"); !strings.Contains(text, "NUL byte") {
		t.Errorf("got\n%s", text)
	}
	w.write(t, loop.ConventionsFile, fits)
	w.opens(t)
}

// TestStartRefusesAConventionsFileThatIsALink: a link would hand a file from outside the
// checkout to the host as its instructions.
func TestStartRefusesAConventionsFileThatIsALink(t *testing.T) {
	w := newStartWorld(t)
	w.write(t, "CLAUDE.md", startConventionsText)
	path := filepath.Join(w.repo, loop.ConventionsFile)
	outside := filepath.Join(t.TempDir(), "outside.md")
	if err := os.WriteFile(outside, []byte("outside words\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	for _, target := range []string{outside, "CLAUDE.md"} {
		if err := os.Remove(path); err != nil {
			t.Fatal(err)
		}
		if err := os.Symlink(target, path); err != nil {
			t.Fatal(err)
		}
		if text := w.refuses(t, "start/repository-unready"); !strings.Contains(text, "is a link") {
			t.Errorf("got\n%s", text)
		}
	}
}

// TestStartRefusesAHostItCannotRun: none on PATH, and one the checkout holds.
func TestStartRefusesAHostItCannotRun(t *testing.T) {
	w := newStartWorld(t)
	bin := w.noHost(t)
	if text := w.refuses(t, "start/host-not-runnable"); !strings.Contains(text, "PATH") {
		t.Errorf("got\n%s", text)
	}
	w.write(t, "bin/claude", "#!/bin/sh\nexit 0\n")
	if err := os.Chmod(filepath.Join(w.repo, "bin", "claude"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", filepath.Join(w.repo, "bin")+string(os.PathListSeparator)+bin)
	w.refuses(t, "start/host-not-runnable")
	if calls := w.hostCalls(); len(calls) != 0 {
		t.Errorf("the host was asked %q", calls)
	}
}

// TestStartRefusesALoggedOutProfileWithTheLoginCommand: the login command on its own line,
// after the one question and no launch.
func TestStartRefusesALoggedOutProfileWithTheLoginCommand(t *testing.T) {
	w := newStartWorld(t)
	w.loggedIn(t, false)
	text := w.refuses(t, "start/not-logged-in")
	if !strings.Contains(text, "\n  CLAUDE_CONFIG_DIR="+abcdhome.Display("profiles", "walled")+" claude auth login\n") || strings.Contains(text, "never-kept") {
		t.Errorf("the refusal carries the login command on its own line, and nothing else the host said:\n%s", text)
	}
	if calls := w.hostCalls(); !slices.Equal(calls, []string{"auth_status_--json"}) {
		t.Errorf("the host was asked %q", calls)
	}
	if err := os.WriteFile(filepath.Join(w.log, "auth"), []byte("no answer"), 0o600); err != nil {
		t.Fatal(err)
	}
	w.refuses(t, "start/no-verdict")

	// A name the person typed is not withheld from the command it makes.
	w.loggedIn(t, false)
	w.profiles(t, "a-long-profile-name", "another")
	for _, args := range [][]string{{"--profile", "a-long-profile-name"}, {"--profile=a-long-profile-name", "some words of a prompt"}} {
		if text := w.refuses(t, "start/not-logged-in", args...); !strings.Contains(text, "\n  CLAUDE_CONFIG_DIR="+abcdhome.Display("profiles", "a-long-profile-name")+" claude auth login\n") {
			t.Errorf("%q: the login command is whole:\n%s", args, text)
		}
	}
}

// TestStartRefusesNamingWhatTheShortLaunchReports: a wall that does not hold is named, and
// a launch that came to no verdict is none.
func TestStartRefusesNamingWhatTheShortLaunchReports(t *testing.T) {
	w := newStartWorld(t)
	with := func(key string, value any) map[string]any {
		init := startInit()
		init[key] = value
		return init
	}
	for _, c := range []struct {
		init    map[string]any
		refused bool
		names   string
	}{
		{with("tools", []string{"Bash", "WebFetch"}), true, "WebFetch"},
		{with("mcp_servers", []map[string]string{{"name": "persons-own", "status": "connected"}}), true, "persons-own"},
		{with("plugins", []map[string]string{{"name": "persons-plugin", "path": "/p", "source": "persons-plugin@a-marketplace"}}), true, "persons-plugin"},
		{with("apiKeySource", "user"), true, "key in use"},
		{startInit(), false, "guard"},
	} {
		w.launch(t, c.init, c.refused)
		if text := w.refuses(t, "start/wall-not-held"); !strings.Contains(text, c.names) || !strings.Contains(text, "2.1.332") {
			t.Errorf("the refusal names %q and the host's version:\n%s", c.names, text)
		}
	}
	stream := filepath.Join(w.log, "stream")
	for what, body := range map[string]string{
		"":            "",
		"2026-10-03T": `{"type":"rate_limit_event","rate_limit_info":{"status":"rejected","resetsAt":1791000000,"rateLimitType":"five_hour"},"session_id":"s"}` + "\n" + `{"type":"result","subtype":"success","is_error":true,"result":"limit","session_id":"s"}` + "\n",
	} {
		if err := os.WriteFile(stream, []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
		if body == "" {
			// A launch that ends with no event reports no tool list.
			w.refuses(t, "start/wall-not-held")
			continue
		}
		if text := w.refuses(t, "start/no-verdict"); !strings.Contains(text, what) {
			t.Errorf("the refusal says when the limit resets:\n%s", text)
		}
	}
	w.launchFails(t)
	w.refuses(t, "start/no-verdict")
}

// TestStartRefusesWhereTheCheckoutsGuardDoesNotRefuse: the guard the open session runs is
// the checkout's, so its registry is judged too, before the launch and again before the
// session opens.
func TestStartRefusesWhereTheCheckoutsGuardDoesNotRefuse(t *testing.T) {
	for _, c := range []struct{ what, body, names string }{
		{"off", startGuardOff, "switched off in .abcd/guard.json"},
		{"retiered", startGuardWarned, "does not refuse the command it is meant to refuse"},
	} {
		t.Run(c.what, func(t *testing.T) {
			w := newStartWorld(t)
			w.commitGuard(t, c.body)
			if text := w.refuses(t, "start/wall-not-held"); !strings.Contains(text, c.names) {
				t.Errorf("the refusal says which:\n%s", text)
			}
			if calls := w.hostCalls(); slices.Contains(calls, "print") {
				t.Errorf("a launch was spent: %q", calls)
			}
		})
		t.Run(c.what+" while the launch ran", func(t *testing.T) {
			w := newStartWorld(t)
			if err := os.WriteFile(filepath.Join(w.log, "during"), []byte(w.guardCommitted(c.body)), 0o600); err != nil {
				t.Fatal(err)
			}
			if text := w.refuses(t, "start/wall-not-held"); !strings.Contains(text, c.names) {
				t.Errorf("the refusal says which:\n%s", text)
			}
			if calls := w.hostCalls(); !slices.Contains(calls, "print") {
				t.Errorf("the launch ran first: %q", calls)
			}
		})
	}
	// An edit that is not committed leaves the registry in force.
	w := newStartWorld(t)
	w.write(t, ".abcd/guard.json", startGuardOff)
	w.opens(t)
}

// TestStartNamesTheProfileItRefuses: a name the account holds is printed in each refusal
// of that directory, as the login command is.
func TestStartNamesTheProfileItRefuses(t *testing.T) {
	w := newStartWorld(t)
	w.profiles(t, "a-long-profile-name", "another")
	if err := os.Chmod(abcdhome.Path(w.home, "profiles", "a-long-profile-name"), 0o770); err != nil {
		t.Fatal(err)
	}
	if text := w.refuses(t, "start/profile-unusable", "--profile", "a-long-profile-name"); !strings.Contains(text, abcdhome.Display("profiles", "a-long-profile-name")+" is one group or other can write") {
		t.Errorf("the refusal names the profile:\n%s", text)
	}
}

// TestStartPagesSayTheConventionsShowInTheCommandLine: the conventions file goes over as a
// launch argument, so both pages say who can read it there.
func TestStartPagesSayTheConventionsShowInTheCommandLine(t *testing.T) {
	for _, rel := range []string{"commands/start.md", ".abcd/development/brief/04-surfaces/43-start.md"} {
		raw, err := os.ReadFile(filepath.Join(testRepoRoot(), filepath.FromSlash(rel)))
		if err != nil {
			t.Fatal(err)
		}
		if text := strings.Join(strings.Fields(string(raw)), " "); !strings.Contains(text, "any account on this machine that can list its processes can read it") {
			t.Errorf("%s does not say who can read the conventions text in the command line", rel)
		}
	}
}

// TestStartTakesItsHomeFromTheMachineLayer: a home inside a working tree that is not the
// account's own is no home, and the note says so.
func TestStartTakesItsHomeFromTheMachineLayer(t *testing.T) {
	w := newStartWorld(t)
	tree := t.TempDir()
	gitInitAt(t, tree)
	home := filepath.Join(tree, "home")
	if err := os.Mkdir(home, 0o700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("HOME", home)
	code := w.run()
	if code != 1 || len(w.calls) != 0 || !strings.Contains(w.errb.String(), "HOME lies inside the git working tree") ||
		!strings.Contains(w.errb.String(), "no session was started") {
		t.Fatalf("exit %d:\n%s%s", code, w.out.String(), w.errb.String())
	}
	if _, err := os.Lstat(abcdhome.Path(home, "profiles")); !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("a profile is made there: %v", err)
	}
}

// TestStartExecsTheHostWithTheSessionsArguments: the file that was admitted, the settled
// arguments, the person's environment less what is left out, and hooks by the kept copy
// though the running abcd is a build under the checkout.
func TestStartExecsTheHostWithTheSessionsArguments(t *testing.T) {
	w := newStartWorld(t)
	w.write(t, ".abcd/.work.local/bin/abcd", "#!/bin/sh\nexit 0\n")
	w.self = filepath.Join(w.repo, ".abcd", ".work.local", "bin", "abcd")
	t.Setenv("ANTHROPIC_BASE_URL", "plainword")
	t.Setenv("ANTHROPIC_AUTH_TOKEN", "plainword")
	t.Setenv("ABCD_STAGE_HOME", "/stage/home")
	t.Setenv("EDITOR", "vi")
	call := w.opens(t, "--permission-mode", "plan", "my words")

	host, err := exec.LookPath("claude")
	if err != nil {
		t.Fatal(err)
	}
	if call.path != realPath(t, host) || len(call.argv) == 0 || call.argv[0] != call.path {
		t.Errorf("the host is %q, argv[0] %q, want %q", call.path, call.argv[:min(1, len(call.argv))], host)
	}
	dir := abcdhome.Path(w.home, "profiles", "walled")
	for _, want := range []string{"--strict-mcp-config", `--mcp-config={"mcpServers":{}}`, "--no-chrome",
		"--tools=Bash,Read,Edit,Write,Agent,TaskStop,AskUserQuestion,Monitor,ScheduleWakeup,CronCreate,CronList,CronDelete",
		"--disallowedTools=Read(/" + dir + "/**)"} {
		if !slices.Contains(call.argv, want) {
			t.Errorf("the arguments lack %q", want)
		}
	}
	if i := slices.Index(call.argv, "--setting-sources"); i < 0 || call.argv[i+1] != "" {
		t.Errorf("no settings source is read: %q", call.argv)
	}
	if tail := call.argv[len(call.argv)-3:]; !slices.Equal(tail, []string{"--permission-mode=plan", "--", "my words"}) {
		t.Errorf("the mode and the prompt come last: %q", tail)
	}
	for _, print := range []string{"--print", "--output-format", "--no-session-persistence"} {
		if slices.Contains(call.argv, print) {
			t.Errorf("the open session holds the short launch's %s", print)
		}
	}
	self, _ := os.ReadFile(w.self)
	sum := sha256.Sum256(self)
	kept := abcdhome.Path(w.home, "session-bin", hex.EncodeToString(sum[:]), "abcd")
	hooks := startSettings(t, call.argv)
	if len(hooks) == 0 {
		t.Fatal("the settings wire no hook")
	}
	for event, groups := range hooks {
		for _, g := range groups {
			for _, h := range g.Hooks {
				if !strings.HasPrefix(h.Command, "'"+kept+"' ") {
					t.Errorf("%s runs %q, want the copy %q", event, h.Command, kept)
				}
			}
		}
	}
	if raw, err := os.ReadFile(kept); err != nil || !bytes.Equal(raw, self) {
		t.Errorf("the copy is the running abcd: %v", err)
	}
	for _, want := range []string{"HOME=" + w.home, "EDITOR=vi", "CLAUDE_CONFIG_DIR=" + dir, "CLAUDE_CODE_DISABLE_CLAUDE_MDS=1",
		"CLAUDE_CODE_DISABLE_AUTO_MEMORY=1", "DISABLE_UPDATES=1", "ENABLE_CLAUDEAI_MCP_SERVERS=false"} {
		if !slices.Contains(call.env, want) {
			t.Errorf("the environment lacks %s", want)
		}
	}
	for _, kv := range call.env {
		if strings.Contains(kv, "plainword") || strings.HasPrefix(kv, "ABCD_STAGE_") || strings.HasPrefix(kv, "ANTHROPIC_") {
			t.Errorf("the environment holds %s", kv)
		}
	}
	if line := "abcd start: left out of the session's environment: ABCD_STAGE_HOME, ANTHROPIC_AUTH_TOKEN, ANTHROPIC_BASE_URL\n"; !strings.Contains(w.out.String(), line) {
		t.Errorf("the left-out line is missing:\n%s", w.out.String())
	}
	if strings.Contains(w.out.String()+w.errb.String(), "plainword") {
		t.Error("a line repeats a value")
	}
}

// TestStartWithProfileStartsOnTheOneNamed: the login is asked, the launch run and the
// session opened on it, and nowhere else.
func TestStartWithProfileStartsOnTheOneNamed(t *testing.T) {
	w := newStartWorld(t)
	w.profiles(t, "alpha", "beta")
	call := w.opens(t, "--profile", "beta")
	beta := "CLAUDE_CONFIG_DIR=" + abcdhome.Path(w.home, "profiles", "beta")
	launched, _ := os.ReadFile(filepath.Join(w.log, "env"))
	if !slices.Contains(call.env, beta) || !slices.Contains(strings.Split(string(launched), "\n"), beta) {
		t.Errorf("the session or the short launch is not on beta")
	}
	if !strings.Contains(w.out.String(), "abcd start: found the profile "+abcdhome.Display("profiles", "beta")+" as it was; nothing in it is changed (a separate login and settings on this one account, not a sandbox)\n") {
		t.Errorf("what was found is said:\n%s", w.out.String())
	}
}

// TestStartHandsTheConventionsFileOverWhole: byte for byte inside the one argument.
func TestStartHandsTheConventionsFileOverWhole(t *testing.T) {
	w := newStartWorld(t)
	body := startConventionsText + "  a line with trailing spaces   \n\ttabs, \"quotes\" and $(nothing)\n\n\n"
	w.write(t, loop.ConventionsFile, body)
	// A subdirectory's own file is not the one.
	w.write(t, "sub/"+loop.ConventionsFile, "another file\n")
	t.Chdir(filepath.Join(w.repo, "sub"))
	call := w.opens(t)
	var held []string
	for _, a := range call.argv {
		if strings.HasPrefix(a, "--append-system-prompt=") {
			held = append(held, a)
		}
	}
	if len(held) != 1 || !strings.HasSuffix(held[0], "\n\n"+body) || strings.Contains(held[0], "another file") {
		t.Fatalf("the conventions are in one argument, whole: %q", held)
	}
	if !strings.Contains(w.out.String(), fmt.Sprintf("abcd start: conventions: AGENTS.md, %d bytes, handed over whole as a launch argument, as it stands at this moment\n", len(body))) {
		t.Errorf("the line says how much:\n%s", w.out.String())
	}
}

// TestStartPassesThePermissionModeNamedAndNoneOtherwise: no file is read for it.
func TestStartPassesThePermissionModeNamedAndNoneOtherwise(t *testing.T) {
	w := newStartWorld(t)
	named := func(argv []string) (held []string) {
		for _, a := range argv {
			if strings.HasPrefix(a, "--permission-mode") {
				held = append(held, a)
			}
		}
		return held
	}
	if got := named(w.opens(t).argv); len(got) != 0 {
		t.Errorf("none named passes none: %q", got)
	}
	if !strings.Contains(w.out.String(), "abcd start: permission mode: the host's default (name one with --permission-mode)\n") {
		t.Errorf("the line says so:\n%s", w.out.String())
	}
	for _, mode := range runner.SessionPermissionModes() {
		if got := named(w.opens(t, "--permission-mode", mode).argv); !slices.Equal(got, []string{"--permission-mode=" + mode}) {
			t.Errorf("%s: passed %q", mode, got)
		}
		if !strings.Contains(w.out.String(), "abcd start: permission mode: "+mode+", as you named it\n") {
			t.Errorf("%s: the line says so:\n%s", mode, w.out.String())
		}
		// The short launch never runs in it.
		argv, _ := os.ReadFile(filepath.Join(w.log, "argv"))
		if got := named(strings.Split(string(argv), "\x00")); !slices.Equal(got, []string{"--permission-mode=dontAsk"}) {
			t.Errorf("%s: the short launch ran with %q", mode, got)
		}
	}
}

// TestStartOpensWithThePersonsPromptInPlaceOfAbcd: the prompt is last, past the end of the
// flags, and the short launch never sees it.
func TestStartOpensWithThePersonsPromptInPlaceOfAbcd(t *testing.T) {
	w := newStartWorld(t)
	prompt := "--help is what I ask about\nover two lines"
	call := w.opens(t, "--", prompt)
	if tail := call.argv[len(call.argv)-2:]; !slices.Equal(tail, []string{"--", prompt}) {
		t.Errorf("the arguments end with %q", tail)
	}
	argv, _ := os.ReadFile(filepath.Join(w.log, "argv"))
	if len(argv) == 0 || strings.Contains(string(argv), "what I ask about") {
		t.Error("the short launch has abcd's own prompt")
	}
}

// TestStartWithNoPromptOpensWaiting: nothing follows the flags; an empty prompt is none.
func TestStartWithNoPromptOpensWaiting(t *testing.T) {
	w := newStartWorld(t)
	for _, args := range [][]string{nil, {""}} {
		if call := w.opens(t, args...); slices.Contains(call.argv, "--") {
			t.Errorf("%q: the arguments hold a prompt: %q", args, call.argv[len(call.argv)-2:])
		}
	}
}

// TestStartSaysNothingAfterItsStartLine: the opening line is the last byte written.
func TestStartSaysNothingAfterItsStartLine(t *testing.T) {
	w := newStartWorld(t)
	call := w.opens(t)
	out := w.out.String()
	if call.said != len(out) || !strings.HasSuffix(out, "abcd start: opening the session; abcd says nothing after this line\n") || w.errb.Len() != 0 {
		t.Fatalf("%d of %d bytes were written before the hand-over:\n%s%s", call.said, len(out), out, w.errb.String())
	}
}

// TestStartSaysWhenTheHostCannotBeStarted: a hand-over that returns is a fault.
func TestStartSaysWhenTheHostCannotBeStarted(t *testing.T) {
	w := newStartWorld(t)
	w.execErr = errors.New("exec format error")
	if code := w.run(); code != 1 || len(w.calls) != 1 ||
		w.errb.String() != "abcd: abcd start: the host could not be started (exec format error); no session is open\n" {
		t.Fatalf("exit %d:\n%s", code, w.errb.String())
	}
}

// TestStartSaysANamedToolTheLaunchDidNotReport: said, and the session opens.
func TestStartSaysANamedToolTheLaunchDidNotReport(t *testing.T) {
	w := newStartWorld(t)
	w.launch(t, map[string]any{"type": "system", "subtype": "init", "tools": []string{"Bash", "Read", "Task"}, "apiKeySource": "none",
		"mcp_servers": []any{}, "plugins": []any{}, "agents": []any{}, "skills": []any{}, "slash_commands": []any{}, "claude_code_version": "2.1.332"}, true)
	w.opens(t)
	want := "tools Bash, Read, Task; named and not reported: Edit, Write, TaskStop, AskUserQuestion, Monitor, ScheduleWakeup, CronCreate, CronList, CronDelete;"
	if !strings.Contains(w.out.String(), want) {
		t.Errorf("the output lacks %q:\n%s", want, w.out.String())
	}
}

// TestStartSaysItReadALaunchNotTheOpenSession: what was read, and what cannot be seen.
func TestStartSaysItReadALaunchNotTheOpenSession(t *testing.T) {
	w := newStartWorld(t)
	w.opens(t)
	for _, want := range []string{
		"abcd start: the host reports a login there; abcd asked for that one state and kept nothing else\n",
		"abcd start: tools: Bash, Read, Edit, Write, Agent, TaskStop, AskUserQuestion, Monitor, ScheduleWakeup, CronCreate, CronList, CronDelete; no others; no server and no server's tool; no browser tool; no plugin but the host's own\n",
		"abcd start: one short launch with the session's arguments is starting; it spends a little of the subscription and is written to the launch ledger\n",
		"; no server; no key in use; the guard refused the command it is meant to refuse; plugins, the host's own: cc-plugin-later; built-in skills (1): simplify; built-in agents (2): Explore, Plan; host 2.1.332\n",
		"abcd start: that was read from one launch with the session's arguments (in print mode, with abcd's own prompt and without a permission mode of yours), not from the open session; what the host adds after the start abcd cannot see\n",
	} {
		if !strings.Contains(w.out.String(), want) {
			t.Errorf("the output lacks %q:\n%s", want, w.out.String())
		}
	}
	if strings.Contains(w.out.String(), w.home) {
		t.Errorf("a line carries the home path:\n%s", w.out.String())
	}
}

// TestTheLedgerHoldsTheShortLaunchAndNotTheSession: one entry for a start that opened.
func TestTheLedgerHoldsTheShortLaunchAndNotTheSession(t *testing.T) {
	w := newStartWorld(t)
	w.opens(t)
	listing, err := runner.ReadLedger(w.home, runner.LedgerQuery{All: true})
	if err != nil {
		t.Fatal(err)
	}
	if len(listing.Entries) != 1 || listing.Entries[0].Role != "session-proof" || listing.Entries[0].Repo != w.repo {
		t.Fatalf("the ledger holds %+v", listing.Entries)
	}
}

// TestStartOnAProfileAlreadyThereChangesNoFileInIt: the directory is only read, by a start
// that is refused and by one that opens.
func TestStartOnAProfileAlreadyThereChangesNoFileInIt(t *testing.T) {
	w := newStartWorld(t)
	dir := abcdhome.Path(w.home, "profiles", "walled")
	for name, body := range map[string]string{"settings.json": "{\"mine\":true}\n", ".credentials.json": "login state\n"} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	before := profileTree(t, dir)
	w.loggedIn(t, false)
	w.refuses(t, "start/not-logged-in")
	sameProfileTree(t, "after a refusal", before, profileTree(t, dir))
	w.loggedIn(t, true)
	w.opens(t)
	sameProfileTree(t, "after a start", before, profileTree(t, dir))
}

// TestStartWritesNoFileADriveReads: the profile and the stages' files stand as they were.
func TestStartWritesNoFileADriveReads(t *testing.T) {
	w := newStartWorld(t)
	stages := abcdhome.Path(w.home, "stage-homes")
	if err := os.MkdirAll(filepath.Join(stages, "walled", "x"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(stages, "walled", "x", "stage-env.sh"), []byte("a stage's\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	dirs := []string{abcdhome.Path(w.home, "profiles"), stages}
	before := []map[string]string{profileTree(t, dirs[0]), profileTree(t, dirs[1])}
	w.opens(t)
	for i, dir := range dirs {
		sameProfileTree(t, dir, before[i], profileTree(t, dir))
	}
}

// TestARepositoryFileChangesNothingStartPasses: a file that names a tool, a mode, a server
// or a start adds nothing and starts nothing.
func TestARepositoryFileChangesNothingStartPasses(t *testing.T) {
	w := newStartWorld(t)
	plain := w.opens(t)
	w.write(t, ".claude/settings.json", `{"permissions":{"defaultMode":"bypassPermissions","allow":["WebFetch","Bash(*)"]},"enabledPlugins":{"x@y":true}}`)
	w.write(t, ".mcp.json", `{"mcpServers":{"theirs":{"command":"/bin/sh","args":["-c",":"]}}}`)
	w.write(t, ".abcd/config.json", `{"start":{"tools":["WebFetch"],"permission_mode":"bypassPermissions","profile":"other","prompt":"go"}}`)
	call := w.opens(t)
	if call.path != plain.path || !slices.Equal(call.argv, plain.argv) || !slices.Equal(call.env, plain.env) {
		t.Fatalf("a repository file changed what start passes:\n%q\n%q", call.argv, plain.argv)
	}
}

// TestStartLeavesOutAHookVerbTheBinaryLacks: an event is never wired to a verb that would
// fail, and is said; with the whole tree every verb in the settings resolves.
func TestStartLeavesOutAHookVerbTheBinaryLacks(t *testing.T) {
	// A tree holding every verb the table names but the one at a sub-agent's start.
	tree := &cobra.Command{Use: "abcd"}
	parents := map[string]*cobra.Command{}
	for _, h := range runner.SessionHooks() {
		for _, v := range h.Verbs {
			f := strings.Fields(v)
			if v == "hook subagent-start" {
				continue
			}
			if parents[f[0]] == nil {
				parents[f[0]] = &cobra.Command{Use: f[0]}
				tree.AddCommand(parents[f[0]])
			}
			parents[f[0]].AddCommand(&cobra.Command{Use: f[1]})
		}
	}
	hooks, left := startHooks(tree)
	if len(hooks) != len(runner.SessionHooks())-1 || !slices.Equal(left, []string{"a sub-agent's start"}) {
		t.Fatalf("%d hooks, left out %q", len(hooks), left)
	}
	settings := runner.Session{Hooks: hooks, Bin: "/kept/abcd"}.Settings()
	if strings.Contains(string(settings), "SubagentStart") || strings.Contains(string(settings), "subagent-start") {
		t.Errorf("the settings wire what the binary lacks: %s", settings)
	}

	w := newStartWorld(t)
	call := w.opens(t)
	root := NewRootCommand()
	said := strings.Contains(w.out.String(), "abcd start: this binary has no hook for a sub-agent's start, so the session's settings leave that event out\n")
	if held := findByPath(root, []string{"hook", "subagent-start"}) != nil; said == held {
		t.Errorf("the binary holds the verb: %v; the line is said: %v", held, said)
	}
	for event, groups := range startSettings(t, call.argv) {
		for _, g := range groups {
			for _, h := range g.Hooks {
				_, verb, _ := strings.Cut(h.Command, "' ")
				if findByPath(root, strings.Fields(verb)) == nil {
					t.Errorf("%s is wired to %q, which the binary lacks", event, verb)
				}
			}
		}
	}
}

// TestEveryLineNamingTheProfileSaysASeparateLoginNeverASandbox: the verb's lines, its page
// and its chapter say what a profile is, and nowhere that a profile is spoken of is it
// called a sandbox.
func TestEveryLineNamingTheProfileSaysASeparateLoginNeverASandbox(t *testing.T) {
	texts := map[string]string{}
	says := []string{"the making's lines", "a start's lines", "commands/start.md", ".abcd/development/brief/04-surfaces/43-start.md"}
	for _, rel := range []string{"commands/start.md", "commands/drain.md", "commands/mode.md", "docs/reference/cli/commands.md",
		".abcd/development/brief/04-surfaces/26-mode.md", ".abcd/development/brief/04-surfaces/35-drain.md",
		".abcd/development/brief/04-surfaces/43-start.md"} {
		raw, err := os.ReadFile(filepath.Join(testRepoRoot(), filepath.FromSlash(rel)))
		if err != nil {
			t.Fatal(err)
		}
		texts[rel] = string(raw)
	}
	// The worlds change the working directory, so the pages are read before them.
	made := newStartWorld(t)
	made.profiles(t)
	if code := made.run(); code != 0 {
		t.Fatalf("exit %d:\n%s", code, made.errb.String())
	}
	opened := newStartWorld(t)
	opened.opens(t)
	texts["the making's lines"], texts["a start's lines"] = made.out.String(), opened.out.String()
	// The reference speaks of the host's own sandbox elsewhere; the verb's section is held.
	const heading = "\n### `abcd start`"
	_, section, found := strings.Cut(texts["docs/reference/cli/commands.md"], heading)
	if !found {
		t.Fatal("the reference has no section for the verb")
	}
	section, _, _ = strings.Cut(section, "\n### ")
	texts["docs/reference/cli/commands.md"] = section

	fold := func(s string) string { return strings.Join(strings.Fields(s), " ") }
	for _, name := range says {
		if !strings.Contains(fold(texts[name]), "separate login and settings") {
			t.Errorf("%s does not say a separate login and settings", name)
		}
	}
	for name, text := range texts {
		for _, paragraph := range strings.Split(text, "\n\n") {
			words := strings.ToLower(fold(paragraph))
			if !strings.Contains(words, "profile") {
				continue
			}
			for _, denial := range []string{"not a sandbox", "never a sandbox"} {
				words = strings.ReplaceAll(words, denial, "")
			}
			if strings.Contains(words, "sandbox") {
				t.Errorf("%s calls a profile a sandbox:\n%s", name, paragraph)
			}
		}
	}
}
