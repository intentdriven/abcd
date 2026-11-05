package runner

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/abcdhome"
	"github.com/intentdriven/abcd/internal/gittest"
)

const wallBin = "/opt/abcd dev/bin/abcd"

// fakeWall plays the harness the wall's checks launch: a mode names the one
// wall the stand-in breaks; wall-ok breaks none.
func fakeWall(mode string) int {
	args := os.Args[1:]
	if len(args) > 0 && args[0] == "auth" {
		if mode == "wall-login" {
			fmt.Println(`{"loggedIn":false,"email":"never-kept@example.invalid"}`)
			return 1
		}
		if mode == "wall-login-exit" {
			fmt.Println(`{"loggedIn":true}`)
			return 1
		}
		fmt.Println(`{"loggedIn":true,"email":"never-kept@example.invalid"}`)
		return 0
	}
	if log := os.Getenv(fakeLogEnv); log != "" {
		raw, _ := json.Marshal(args)
		_ = os.WriteFile(filepath.Join(log, "claude.argv.json"), raw, 0o600)
	}
	// A harness reads the working tree's settings unless the launch's setting
	// sources leave them out, or the mode has it read them regardless.
	if !slices.Contains(args, "--setting-sources=user") || mode == "wall-tree" {
		var s struct {
			Hooks map[string][]hookGroup `json:"hooks"`
		}
		raw, _ := os.ReadFile(filepath.Join(".claude", "settings.json"))
		_ = json.Unmarshal(raw, &s)
		for _, g := range s.Hooks["SessionStart"] {
			for _, h := range g.Hooks {
				_ = exec.Command("/bin/sh", "-c", h.Command).Run()
			}
		}
	}
	init := map[string]any{"type": "system", "subtype": "init", "session_id": "s", "model": "m",
		"tools": []string{"Read", "Bash", ApproverTool}, "apiKeySource": "none",
		"mcp_servers": []map[string]string{{"name": "abcd", "status": "connected"}}, "plugins": []string{}}
	switch mode {
	case "wall-tools":
		init["tools"] = []string{"Read", "Bash", ApproverTool, "WebFetch"}
	case "wall-servers":
		init["mcp_servers"] = []map[string]string{{"name": "abcd"}, {"name": "persons-own"}}
	case "wall-plugins":
		init["plugins"] = []map[string]string{{"name": "persons-plugin"}}
	case "wall-key":
		init["apiKeySource"] = "ANTHROPIC_API_KEY"
	case "wall-key-unsaid":
		delete(init, "apiKeySource")
	}
	raw, _ := json.Marshal(init)
	fmt.Println(string(raw))
	shellHome := os.Getenv(StageHomeEnv)
	if mode == "wall-home" {
		shellHome = "/the/persons/own/home"
	}
	if mode != "wall-home-unsaid" {
		ev, _ := json.Marshal(map[string]any{"type": "user", "message": map[string]any{"content": []map[string]any{
			{"type": "tool_result", "content": wallHomeMarker + shellHome + "\n"}}}})
		fmt.Println(string(ev))
	}
	if mode != "wall-env-unsaid" {
		names := []string{"PATH", "HOME", "CLAUDECODE"}
		if mode == "wall-env" {
			names = append(names, "GH_TOKEN")
		}
		var lines strings.Builder
		for _, n := range names {
			lines.WriteString(wallEnvMarker + n + "\n")
		}
		ev, _ := json.Marshal(map[string]any{"type": "user", "message": map[string]any{"content": []map[string]any{
			{"type": "tool_result", "content": lines.String()}}}})
		fmt.Println(string(ev))
	}
	if mode != "wall-guard" {
		ev, _ := json.Marshal(map[string]any{"type": "system", "subtype": "hook_response", "hook_event": "PreToolUse",
			"stdout": `{"hookSpecificOutput":{"permissionDecision":"deny","permissionDecisionReason":"` + guardRefusalText + ` (git-push-force)"}}`})
		fmt.Println(string(ev))
	}
	fmt.Println(`{"type":"result","subtype":"success","is_error":false,"result":"done","session_id":"s"}`)
	return 0
}

func wallProfile(t *testing.T) Profile {
	t.Helper()
	home := profileHome(t, "walled")
	p, err := DriveProfile(home)
	if err != nil {
		t.Fatal(err)
	}
	p.Ledger = newLedger(home)
	return p
}

// TestTheWalledSettingsHoldAbcdsHooksAndNothingElse: each event's command names
// the binary by absolute path, the guard's matcher is the plugin's, and no
// other key stands beside hooks.
func TestTheWalledSettingsHoldAbcdsHooksAndNothingElse(t *testing.T) {
	var s map[string]map[string][]hookGroup
	if err := json.Unmarshal(WallSettings(wallBin), &s); err != nil {
		t.Fatal(err)
	}
	if len(s) != 1 || s["hooks"] == nil {
		t.Fatalf("hooks and nothing else: %v", s)
	}
	pre := s["hooks"]["PreToolUse"]
	if len(pre) != 1 || !strings.Contains(pre[0].Matcher, "Bash") || pre[0].Hooks[0].Command != "'"+wallBin+"' guard hook" {
		t.Fatalf("the guard: %+v", pre)
	}
	for event, verb := range map[string]string{"UserPromptSubmit": "hook prompt-router", "SessionEnd": "hook session-end", "Stop": "hook stop"} {
		if g := s["hooks"][event]; len(g) != 1 || g[0].Hooks[0].Command != "'"+wallBin+"' "+verb {
			t.Errorf("%s: %+v", event, g)
		}
	}
	if len(s["hooks"]["SessionStart"][0].Hooks) != 2 {
		t.Errorf("the session start resets the rules loader and runs the session hook: %+v", s["hooks"]["SessionStart"])
	}
	if _, ok := s["hooks"]["PermissionRequest"]; ok {
		t.Error("a walled launch's prompts are the approver's, so no hook answers them first")
	}
}

// TestWritingTheSettingsReWiresANewBinaryAndSaysWhenItReplacedAnEdit: a file
// abcd rendered for another binary is replaced quietly; one a person edited is
// replaced and said; the login state beside them is untouched.
func TestWritingTheSettingsReWiresANewBinaryAndSaysWhenItReplacedAnEdit(t *testing.T) {
	p := wallProfile(t)
	login := filepath.Join(p.Dir, ".credentials.json")
	if err := os.WriteFile(login, []byte("login state"), 0o600); err != nil {
		t.Fatal(err)
	}
	if replaced, err := WriteWallSettings(p, "/old/abcd"); err != nil || replaced {
		t.Fatalf("a first write replaces nothing: %v %v", replaced, err)
	}
	if replaced, err := WriteWallSettings(p, wallBin); err != nil || replaced {
		t.Fatalf("a new binary re-wires quietly: %v %v", replaced, err)
	}
	if err := CheckWallSettings(p, wallBin); err != nil {
		t.Fatalf("what was written holds: %v", err)
	}
	path := filepath.Join(p.Dir, wallSettingsName)
	edited := strings.Replace(string(WallSettings(wallBin)), `"hooks": {`, `"apiKeyHelper": "x", "hooks": {`, 1)
	if err := os.WriteFile(path, []byte(edited), 0o600); err != nil {
		t.Fatal(err)
	}
	var we *WallError
	if err := CheckWallSettings(p, wallBin); !errors.As(err, &we) || we.Wall != WallSettingsFile {
		t.Fatalf("a difference is the settings wall: %v", err)
	}
	if replaced, err := WriteWallSettings(p, wallBin); err != nil || !replaced {
		t.Fatalf("an edit is replaced and said: %v %v", replaced, err)
	}
	if err := CheckWallSettings(p, wallBin); err != nil {
		t.Fatal(err)
	}
	if raw, _ := os.ReadFile(login); string(raw) != "login state" {
		t.Fatal("the login state is never touched")
	}
}

// TestAWalledLaunchRefusesSettingsThatDiffer: nothing is launched on settings
// other than abcd's.
func TestAWalledLaunchRefusesSettingsThatDiffer(t *testing.T) {
	f, c, p := walledFake(t, "ok")
	if err := os.WriteFile(filepath.Join(p.Dir, wallSettingsName), []byte(`{"hooks":{}}`), 0o600); err != nil {
		t.Fatal(err)
	}
	_, _, err := c.Run(context.Background(), shellRequest(f))
	if err == nil || !strings.Contains(err.Error(), "differ") {
		t.Fatalf("the launch is refused: %v", err)
	}
	if f.launched(Claude) {
		t.Fatal("no harness started")
	}
}

// TestTheLoginWallAsksTheHarnessAndKeepsOnlyTheState: the harness's report is
// read for the one state; no account identifier reaches a refusal.
func TestTheLoginWallAsksTheHarnessAndKeepsOnlyTheState(t *testing.T) {
	newFake(t, "wall-ok", Claude)
	p := wallProfile(t)
	if err := CheckLogin(context.Background(), p); err != nil {
		t.Fatalf("a login holds: %v", err)
	}
	t.Setenv(fakeModeEnv, "wall-login")
	err := CheckLogin(context.Background(), p)
	var we *WallError
	if !errors.As(err, &we) || we.Wall != WallLogin {
		t.Fatalf("no login is the login wall: %v", err)
	}
	if strings.Contains(err.Error(), "never-kept") {
		t.Fatalf("no account identifier is kept: %v", err)
	}
	t.Setenv(fakeModeEnv, "wall-login-exit")
	if err := CheckLogin(context.Background(), p); !errors.As(err, &we) || we.Wall != WallLogin {
		t.Fatalf("a harness that failed proves no login, whatever it printed: %v", err)
	}
}

// TestTheProofNamesEachWallThatDoesNotHold: the stand-in reports each wall held
// or broken in turn.
func TestTheProofNamesEachWallThatDoesNotHold(t *testing.T) {
	for mode, wall := range map[string]string{
		"wall-ok":          "",
		"wall-tools":       WallTools,
		"wall-servers":     WallServers,
		"wall-plugins":     WallPlugins,
		"wall-key":         WallKey,
		"wall-key-unsaid":  WallKey,
		"wall-guard":       WallGuard,
		"wall-tree":        WallWorkingTree,
		"wall-home":        WallHome,
		"wall-home-unsaid": WallHome,
		"wall-env":         WallEnv,
		"wall-env-unsaid":  WallEnv,
	} {
		t.Run(mode, func(t *testing.T) {
			f := newFake(t, mode, Claude)
			p := wallProfile(t)
			err := ProveWall(context.Background(), p, wallBin)
			if wall == "" {
				if err != nil {
					t.Fatalf("every wall holds: %v", err)
				}
				argv := f.argv(t, Claude)
				for _, want := range []string{"--setting-sources=user", "--strict-mcp-config", "--tools=Read,Bash", "--include-hook-events"} {
					if !slices.Contains(argv, want) {
						t.Errorf("the proof's launch %q lacks %q", argv, want)
					}
				}
				if !strings.Contains(argv[len(argv)-1], guardProofCommand) {
					t.Errorf("the proof asks for the command the guard refuses: %q", argv[len(argv)-1])
				}
				return
			}
			var we *WallError
			if !errors.As(err, &we) || we.Wall != wall {
				t.Fatalf("the refusal names the wall %q: %v", wall, err)
			}
		})
	}
}

// TestALaunchWithNoSettingSourcesIsCaughtByTheProof holds the proof's working
// tree to what it is for: a harness that reads it runs the hook it holds.
func TestALaunchWithNoSettingSourcesIsCaughtByTheProof(t *testing.T) {
	newFake(t, "wall-ok", Claude)
	p := wallProfile(t)
	p.Args = slices.DeleteFunc(slices.Clone(p.Args), func(a string) bool { return strings.HasPrefix(a, "--setting-sources") })
	var we *WallError
	if err := ProveWall(context.Background(), p, wallBin); !errors.As(err, &we) || we.Wall != WallWorkingTree {
		t.Fatalf("the working tree's hook ran: %v", err)
	}
}

// ledgerWrapsTheProof is what the ledger owes the proof: nil while the package
// has no RecordLaunch; once it has, ProveWall calls it and the end-only hook is
// gone, since that hook sits inside the dispatcher's run and would double-record.
func ledgerWrapsTheProof(files []*ast.File) error {
	var ledger, hook, wrapped bool
	for _, f := range files {
		for _, d := range f.Decls {
			switch d := d.(type) {
			case *ast.FuncDecl:
				if d.Recv == nil && d.Name.Name == "RecordLaunch" {
					ledger = true
				}
				if d.Recv == nil && d.Name.Name == "recordLaunch" {
					hook = true
				}
				if d.Recv == nil && d.Name.Name == "ProveWall" {
					ast.Inspect(d, func(n ast.Node) bool {
						if c, ok := n.(*ast.CallExpr); ok {
							if id, ok := c.Fun.(*ast.Ident); ok && id.Name == "RecordLaunch" {
								wrapped = true
							}
						}
						return true
					})
				}
			case *ast.GenDecl:
				for _, sp := range d.Specs {
					if v, ok := sp.(*ast.ValueSpec); ok && slices.ContainsFunc(v.Names, func(n *ast.Ident) bool { return n.Name == "recordLaunch" }) {
						hook = true
					}
				}
			}
		}
	}
	switch {
	case !ledger:
		return nil
	case !wrapped:
		return errors.New("the package has RecordLaunch, and ProveWall does not call it, so the proof's launch goes unrecorded")
	case hook:
		return errors.New("the package has RecordLaunch and still the end-only recordLaunch hook; delete the hook")
	}
	return nil
}

// TestTheLedgerCannotLandWithoutTheProof fails the merge that brings RecordLaunch
// here (itd-2611031342011261) until the proof's launch goes through it: the hook
// is a no-op, so nothing else would notice the proof going unrecorded.
func TestTheLedgerCannotLandWithoutTheProof(t *testing.T) {
	parse := func(src string) []*ast.File {
		f, err := parser.ParseFile(token.NewFileSet(), "src.go", "package runner\n"+src, 0)
		if err != nil {
			t.Fatal(err)
		}
		return []*ast.File{f}
	}
	const hook, ledger = "var recordLaunch = func() {}\n", "func RecordLaunch(run func() error) error { return run() }\n"
	for _, c := range []struct {
		name, src string
		bad       bool
	}{
		{"no ledger yet", hook + "func ProveWall() { recordLaunch() }\n", false},
		{"ledger beside the hook", hook + ledger + "func ProveWall() { recordLaunch() }\n", true},
		{"ledger, proof unwrapped", ledger + "func ProveWall() {}\n", true},
		{"ledger wraps, hook stays", hook + ledger + "func ProveWall() { RecordLaunch(nil) }\n", true},
		{"ledger wraps, hook gone", ledger + "func ProveWall() { RecordLaunch(nil) }\n", false},
	} {
		if err := ledgerWrapsTheProof(parse(c.src)); (err != nil) != c.bad {
			t.Errorf("%s: %v", c.name, err)
		}
	}
	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatal(err)
	}
	var files []*ast.File
	fset := token.NewFileSet()
	for _, e := range entries {
		if !strings.HasSuffix(e.Name(), ".go") || strings.HasSuffix(e.Name(), "_test.go") {
			continue
		}
		f, err := parser.ParseFile(fset, e.Name(), nil, 0)
		if err != nil {
			t.Fatal(err)
		}
		files = append(files, f)
	}
	if err := ledgerWrapsTheProof(files); err != nil {
		t.Fatal(err)
	}
}

// TestAWalledLaunchCarriesOnlyWhatAStageNeeds: the launch's environment is an
// allow-list, so nothing the person's shell holds passes by default.
func TestAWalledLaunchCarriesOnlyWhatAStageNeeds(t *testing.T) {
	f, c, p := walledFake(t, "ok")
	for _, k := range []string{"GH_TOKEN", "GITHUB_TOKEN", "AWS_SECRET_ACCESS_KEY", "SSH_AUTH_SOCK", "A_PERSONS_OWN_SECRET"} {
		t.Setenv(k, "a-value-never-passed")
	}
	t.Setenv("TMPDIR", "/a/tmp")
	t.Setenv("LC_ALL", "C")
	t.Setenv("GOFLAGS", "-p=1")
	if _, _, err := c.Run(context.Background(), shellRequest(f)); err != nil {
		t.Fatalf("run: %v", err)
	}
	raw, err := os.ReadFile(filepath.Join(f.log, "claude.env"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(raw), "a-value-never-passed") {
		t.Fatalf("something outside the allow-list reached the harness:\n%s", raw)
	}
	env := strings.Split(string(raw), "\n")
	for _, want := range []string{"TMPDIR=/a/tmp", "LC_ALL=C", "GOFLAGS=-p=1", "PATH=" + os.Getenv("PATH"), "HOME=" + os.Getenv("HOME"),
		"CLAUDE_ENV_FILE=" + filepath.Join(p.Dir, stageEnvName)} {
		if !slices.Contains(env, want) {
			t.Errorf("the harness's environment lacks %q:\n%s", want, raw)
		}
	}
}

// TestAStagesShellHasItsOwnHomeAndWhatItNeeds: sourcing the file the launch
// names moves the shell's home under the profile, and git and the Go caches
// still work there: identity and filters carry over, credential helpers do not.
func TestAStagesShellHasItsOwnHomeAndWhatItNeeds(t *testing.T) {
	if _, err := exec.LookPath("go"); err != nil {
		t.Skip("no go")
	}
	gittest.Env(t) // go's telemetry stays off, and out of this test's cleanup
	realGitRead(t)
	home := os.Getenv("HOME")
	cfg := "[user]\n\tname = Pat Example\n\temail = pat@example.invalid\n" +
		"[filter \"lfs\"]\n\tclean = git-lfs clean -- %f\n\trequired = true\n" +
		"[credential]\n\thelper = a-helper-never-passed\n"
	if err := os.WriteFile(filepath.Join(home, ".gitconfig"), []byte(cfg), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(abcdhome.Path(home, "profiles", "walled"), 0o700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("XDG_CONFIG_HOME", "")
	t.Setenv("GIT_CONFIG_GLOBAL", "")
	if err := os.Unsetenv("GIT_CONFIG_GLOBAL"); err != nil {
		t.Fatal(err)
	}
	goCache, err := exec.Command("go", "env", "GOCACHE", "GOMODCACHE").Output()
	if err != nil {
		t.Fatal(err)
	}
	p, err := DriveProfile(home)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := WriteWallSettings(p, wallBin); err != nil {
		t.Fatal(err)
	}
	run := func(script string) string {
		cmd := exec.Command("/bin/sh", "-c", `. "$F"; `+script)
		cmd.Env = append(os.Environ(), "F="+filepath.Join(p.Dir, stageEnvName), "CLAUDE_CONFIG_DIR="+p.Dir)
		out, _ := cmd.CombinedOutput()
		return strings.TrimSpace(string(out))
	}
	// The Go there starts its telemetry under that home, which would race this
	// test's cleanup.
	run(`go telemetry off`)
	if got := run(`printf %s "$CLAUDE_CONFIG_DIR"`); got != "" {
		t.Errorf("the shell keeps the walled directory: %q", got)
	}
	if got := run(`printf %s "$` + abcdhome.PersonHomeEnv + `"`); got != home {
		t.Errorf("the shell names the person's home for abcd's shared records: %q", got)
	}
	if got := run(`printf %s "$HOME"`); got != p.ShellHome {
		t.Fatalf("the shell's home is %q, want %q", got, p.ShellHome)
	}
	if got := run(`git config --global user.email`); got != "pat@example.invalid" {
		t.Errorf("the identity does not carry over: %q", got)
	}
	if got := run(`git config --global filter.lfs.clean`); got != "git-lfs clean -- %f" {
		t.Errorf("the filter does not carry over: %q", got)
	}
	if got := run(`git config --global credential.helper`); got != "" {
		t.Errorf("a credential helper carried over: %q", got)
	}
	if got := run(`go env GOCACHE GOMODCACHE`); got != strings.TrimSpace(string(goCache)) {
		t.Errorf("the Go caches moved: %q, want %q", got, goCache)
	}
	if err := CheckWallSettings(p, wallBin); err != nil {
		t.Fatalf("what was written holds: %v", err)
	}
}

// TestTheCheckComparesWhatALaunchReads: beside the settings, what a stage's shell
// sources, the home's git configuration, anything else a launch would load from
// the walled directory, and the person's machine declarations.
func TestTheCheckComparesWhatALaunchReads(t *testing.T) {
	for name, change := range map[string]func(t *testing.T, p Profile){
		"the shell's file": func(t *testing.T, p Profile) {
			if err := os.WriteFile(filepath.Join(p.Dir, stageEnvName), []byte("export X=1\n"), 0o600); err != nil {
				t.Fatal(err)
			}
		},
		"the home's git configuration": func(t *testing.T, p Profile) {
			if err := os.WriteFile(filepath.Join(p.ShellHome, ".gitconfig"), []byte("[core]\n\tpager = x\n"), 0o600); err != nil {
				t.Fatal(err)
			}
		},
		"a file the launch would load": func(t *testing.T, p Profile) {
			if err := os.MkdirAll(filepath.Join(p.Dir, "agents"), 0o700); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(p.Dir, "agents", "x.md"), []byte("x"), 0o600); err != nil {
				t.Fatal(err)
			}
		},
		"a machine declaration": func(t *testing.T, p Profile) {
			if err := os.WriteFile(abcdhome.Path(p.PersonHome, "config.json"), []byte(`{"x":1}`), 0o600); err != nil {
				t.Fatal(err)
			}
		},
		"the routing declaration": func(t *testing.T, p Profile) {
			if err := os.WriteFile(abcdhome.Path(p.PersonHome, "oracle-routing.json"), []byte(`{"x":1}`), 0o600); err != nil {
				t.Fatal(err)
			}
		},
	} {
		t.Run(name, func(t *testing.T) {
			p := wallProfile(t)
			if _, err := WriteWallSettings(p, wallBin); err != nil {
				t.Fatal(err)
			}
			if err := CheckWallSettings(p, wallBin); err != nil {
				t.Fatalf("what was written holds: %v", err)
			}
			change(t, p)
			var we *WallError
			if err := CheckWallSettings(p, wallBin); !errors.As(err, &we) || we.Wall != WallSettingsFile {
				t.Fatalf("a difference is the settings wall: %v", err)
			}
		})
	}
}

// TestARejectedLimitsResetIsRead: only a rejected limit's reset counts, in
// seconds, and an event with none reports none.
func TestARejectedLimitsResetIsRead(t *testing.T) {
	const rejected = `{"type":"rate_limit_event","rate_limit_info":{"status":"rejected","resetsAt":1791000000}}`
	const warned = `{"type":"rate_limit_event","rate_limit_info":{"status":"allowed_warning","resetsAt":1791000001}}`
	if got := claudeRateLimitReset([]byte(warned + "\n" + rejected + "\n")); got.Unix() != 1791000000 {
		t.Fatalf("the rejected event's reset: %v", got)
	}
	if got := claudeRateLimitReset([]byte(warned + "\n")); !got.IsZero() {
		t.Fatalf("a warning is no limit: %v", got)
	}
	if got := claudeRateLimitReset([]byte(`{"type":"rate_limit_event","rate_limit_info":{"status":"rejected"}}`)); !got.IsZero() {
		t.Fatalf("no reset reported: %v", got)
	}
}
