package runner

import (
	"context"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/intentdriven/abcd/internal/abcdhome"
)

// walledFake is a fake harness with a profile, for a claude runner the test
// builds with it.
func walledFake(t *testing.T, mode string) (fakeEnv, *ClaudeCLI, Profile) {
	t.Helper()
	f := newFake(t, mode, Claude)
	p, err := DriveProfile(profileHome(t, "walled"), "")
	if err != nil {
		t.Fatal(err)
	}
	c := newClaude("")
	c.profile = &p
	c.approver = "/opt/abcd/bin/abcd"
	if _, err := WriteWallSettings(p, c.approver); err != nil {
		t.Fatal(err)
	}
	return f, c, p
}

// shellRequest is a request for a role that holds the shell.
func shellRequest(f fakeEnv) Request {
	req := f.request("implementer")
	req.Tools = []string{"Read", "Edit", "Write", "Bash", "Grep", "Glob"}
	req.Base = strings.Repeat("a", 40)
	return req
}

// argWith returns the one argument with prefix, "" when none.
func argWith(argv []string, prefix string) string {
	for _, a := range argv {
		if strings.HasPrefix(a, prefix) {
			return strings.TrimPrefix(a, prefix)
		}
	}
	return ""
}

// TestTheWalledLaunchNamesWhatTheAgentMayHold is the argv the wall names: no
// --bare, the tools the stage holds and no others, the shell answered by the
// approver as the only server, no session persistence, and settings read from
// the walled directory's user layer alone.
func TestTheWalledLaunchNamesWhatTheAgentMayHold(t *testing.T) {
	f, c, _ := walledFake(t, "ok")
	req := shellRequest(f)
	if _, _, err := c.Run(context.Background(), req); err != nil {
		t.Fatalf("run: %v", err)
	}
	argv := f.argv(t, Claude)
	if slices.Contains(argv, "--bare") {
		t.Fatalf("a walled launch runs hooks, so it is never bare: %q", argv)
	}
	for _, want := range []string{"--print", "--output-format", "stream-json", "--no-session-persistence",
		"--strict-mcp-config", "--setting-sources=user", "--permission-mode", "default",
		"--permission-prompt-tool=" + ApproverTool} {
		if !slices.Contains(argv, want) {
			t.Errorf("the walled argv %q lacks %q", argv, want)
		}
	}
	if got := argWith(argv, "--tools="); got != "Read,Edit,Write,Bash,Grep,Glob" {
		t.Errorf("the tools the agent holds are %q", got)
	}
	if got := argWith(argv, "--allowedTools="); got != "Read,Edit,Write,Grep,Glob" {
		t.Errorf("the shell is the approver's to answer, never granted outright: %q", got)
	}
	cfg := argWith(argv, "--mcp-config=")
	if !strings.Contains(cfg, `"abcd"`) || strings.Count(cfg, `"command"`) != 1 || !strings.Contains(cfg, "/opt/abcd/bin/abcd") {
		t.Errorf("the approver is the only server: %q", cfg)
	}
	if argv[len(argv)-2] != "--" {
		t.Errorf("the prompt is behind the end-of-options marker: %q", argv)
	}
}

// TestAWalledLaunchWithNoShellHoldsNoServer: a role with no shell starts no
// server, under the strict switch all the same, and its permission rules stay
// on the allow-list while the tools it holds are their bare names.
func TestAWalledLaunchWithNoShellHoldsNoServer(t *testing.T) {
	f, c, _ := walledFake(t, "ok")
	req := f.request("drain-judge")
	req.Tools = []string{"Read(//d/**)", "Edit(//d/answer.json)"}
	req.Denied = []string{"Read(//repo/**)"}
	if _, _, err := c.Run(context.Background(), req); err != nil {
		t.Fatalf("run: %v", err)
	}
	argv := f.argv(t, Claude)
	if got := argWith(argv, "--mcp-config="); got != `{"mcpServers":{}}` {
		t.Errorf("no server is named: %q", got)
	}
	if !slices.Contains(argv, "--strict-mcp-config") || !slices.Contains(argv, "dontAsk") {
		t.Errorf("no shell, no prompts, no other server: %q", argv)
	}
	if got := argWith(argv, "--tools="); got != "Read,Edit,Write" {
		t.Errorf("an edit grant holds the tool a new file is written with: %q", got)
	}
	if got := argWith(argv, "--allowedTools="); got != "Read(//d/**),Edit(//d/answer.json)" {
		t.Errorf("the grants keep their paths: %q", got)
	}
	if got := argWith(argv, "--disallowedTools="); got != "Read(//repo/**)" {
		t.Errorf("the refusals stand: %q", got)
	}
}

// TestTheWalledLaunchsEnvironmentNamesTheDirectoryAndHoldsNoCredential: the
// harness sees the walled directory, memory and instruction files off, and no
// variable it could bill through, whatever the parent holds.
func TestTheWalledLaunchsEnvironmentNamesTheDirectoryAndHoldsNoCredential(t *testing.T) {
	f, c, p := walledFake(t, "ok")
	t.Setenv("CLAUDE_CONFIG_DIR", "/the/persons/own")
	t.Setenv("CLAUDE_CODE_DISABLE_AUTO_MEMORY", "0")
	t.Setenv("DISABLE_UPDATES", "")
	for _, k := range credentialVars {
		t.Setenv(k, "a-value-never-passed")
	}
	if _, _, err := c.Run(context.Background(), shellRequest(f)); err != nil {
		t.Fatalf("run: %v", err)
	}
	raw, err := os.ReadFile(filepath.Join(f.log, "claude.env"))
	if err != nil {
		t.Fatal(err)
	}
	env := strings.Split(string(raw), "\n")
	for _, want := range []string{"CLAUDE_CONFIG_DIR=" + p.Dir, "CLAUDE_CODE_DISABLE_CLAUDE_MDS=1", "CLAUDE_CODE_DISABLE_AUTO_MEMORY=1", "DISABLE_UPDATES=1"} {
		if !slices.Contains(env, want) {
			t.Errorf("the harness's environment lacks %q", want)
		}
	}
	if strings.Contains(string(raw), "a-value-never-passed") {
		t.Fatalf("a credential reached the harness:\n%s", raw)
	}
	for _, k := range credentialVars {
		for _, kv := range env {
			if strings.HasPrefix(kv, k+"=") {
				t.Errorf("%s is sealed out, not set empty: %q", k, kv)
			}
		}
	}
	if strings.Contains(string(raw), "/the/persons/own") {
		t.Error("the person's own configuration directory reached the harness")
	}
}

// TestTheBareLaunchIsUnchanged: a runner with no profile launches as it did.
func TestTheBareLaunchIsUnchanged(t *testing.T) {
	f := newFake(t, "ok", Claude)
	if _, _, err := newClaude("").Run(context.Background(), f.request("ruthless-reviewer")); err != nil {
		t.Fatal(err)
	}
	argv := f.argv(t, Claude)
	if !slices.Contains(argv, "--bare") || argWith(argv, "--setting-sources=") != "" || argWith(argv, "--tools=") != "" {
		t.Fatalf("the bare launch keeps its shape: %q", argv)
	}
}

// TestSealedEnvRemovesANameItHoldsWithNoValue: a bare name takes the variable
// out; a KEY=value sets it.
func TestSealedEnvRemovesANameItHoldsWithNoValue(t *testing.T) {
	got := sealedEnv([]string{"A=1", "B=2", "C=3"}, []string{"B", "C=9"})
	if !slices.Equal(got, []string{"A=1", "C=9"}) {
		t.Fatalf("sealedEnv = %q", got)
	}
}

// TestConfigWalledPutsEveryRoleOnTheWalledClaudeLaunch: every role the drive
// starts runs on claude, on the profile, through the adapter the dispatcher
// builds, none of it read as a --route.
func TestConfigWalledPutsEveryRoleOnTheWalledClaudeLaunch(t *testing.T) {
	c := &Config{roles: map[string]Route{}, timeouts: map[string]time.Duration{}, runners: map[string]RunnerConfig{Claude: {Name: Claude}}}
	p := Profile{Dir: "/d"}
	w := c.Walled(p)
	for _, role := range []string{"implementer", "drain-judge", "ruthless-reviewer", "security-reviewer", "intent-auditor"} {
		if got := w.RouteFor(role).Runner; got != Claude {
			t.Errorf("%s runs on %q", role, got)
		}
	}
	if len(w.FlagRoutes()) != 0 {
		t.Errorf("the yes is no --route: %v", w.FlagRoutes())
	}
	cl, ok := w.adapter(RunnerConfig{Name: Claude}).(*ClaudeCLI)
	if !ok || cl.profile == nil || cl.profile.Dir != "/d" {
		t.Fatalf("the adapter carries the profile: %+v", cl)
	}
	if got := c.RouteFor("implementer").Runner; got != Host {
		t.Errorf("the configuration read stays as it was: %q", got)
	}
}

// TestEveryCheckAndLaunchOnANamedProfileIsOnItsDirectory: the settings written and
// compared, the login asked after, a stage's launch and a limit's probe are the
// named directory's, and its sibling is left as it was.
func TestEveryCheckAndLaunchOnANamedProfileIsOnItsDirectory(t *testing.T) {
	f := newFake(t, "probe-ok", Claude)
	home := profileHome(t, "alpha", "beta")
	alpha := abcdhome.Path(home, "profiles", "alpha")
	p, err := DriveProfile(home, "beta")
	if err != nil {
		t.Fatal(err)
	}
	const bin = "/opt/abcd/bin/abcd"
	if _, err := WriteWallSettings(p, bin); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(p.Dir, wallSettingsName)); err != nil {
		t.Fatalf("the settings are written into the named directory: %v", err)
	}
	if entries, err := os.ReadDir(alpha); err != nil || len(entries) != 0 {
		t.Fatalf("its sibling is left as it was: %v %v", entries, err)
	}
	if err := CheckWallSettings(p, bin); err != nil {
		t.Fatalf("the settings stand: %v", err)
	}
	if err := os.WriteFile(filepath.Join(alpha, wallSettingsName), []byte("{}"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := CheckWallSettings(p, bin); err != nil {
		t.Fatalf("a sibling's settings are no matter: %v", err)
	}
	if err := os.WriteFile(filepath.Join(p.Dir, wallSettingsName), []byte("{}"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := CheckWallSettings(p, bin); err == nil {
		t.Fatal("the named directory's own settings are what is compared")
	}
	if _, err := WriteWallSettings(p, bin); err != nil {
		t.Fatal(err)
	}

	onDir := func(what string) {
		t.Helper()
		if got, _ := envValue(envOf(t, f, "claude.env"), "CLAUDE_CONFIG_DIR"); got != p.Dir {
			t.Errorf("%s ran on %q, want %q", what, got, p.Dir)
		}
	}
	cfg := mustLoad(t, `{"runner":{"claude":{}}}`, "").Walled(p)
	if renewed, fl := cfg.ProbeLimit(context.Background(), "implementer", ""); !renewed {
		t.Fatalf("the probe answered: %+v", fl)
	}
	onDir("a limit's probe")
	t.Setenv(fakeModeEnv, "wall-ok")
	if err := CheckLogin(context.Background(), p); err != nil {
		t.Fatalf("a login holds: %v", err)
	}
	onDir("the login's check")
	t.Setenv(fakeModeEnv, "ok")
	rc, ok := cfg.Runner(Claude)
	if !ok {
		t.Fatal("the runner is enabled")
	}
	c := cfg.adapter(rc).(*ClaudeCLI)
	c.approver = bin
	if _, _, err := c.Run(context.Background(), shellRequest(f)); err != nil {
		t.Fatalf("run: %v", err)
	}
	onDir("a stage's launch")
}
