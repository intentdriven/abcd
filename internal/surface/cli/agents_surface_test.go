package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/abcdhome"
	"github.com/intentdriven/abcd/internal/core/runner"
)

// agentsWorld is a temporary home holding three launches: two of one session's
// on one run, a key's and a subscription's, and one of a terminal run's, a
// failure carrying a control sequence in its reason. It clears the host's
// session, so a test names the one it wants.
func agentsWorld(t *testing.T) string {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv(runner.HostSessionEnv, "")
	dir := abcdhome.Path(home, "agents")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	for name, line := range map[string]string{
		"lch-261104120000000001-aaaaaaaa.json": `{"schema_version":1,"orchestrator":{"kind":"session","id":"sess-1","name":"Alice"},"run":"run-1","lane":"lane-1","await":"r.json","role":"implementer","runner":"claude","route":"key","key":"build-key","model":"claude-fake","usage":{"prompt_tokens":1,"completion_tokens":2,"cache_read_tokens":3,"cache_write_tokens":4},"cost":{"amount":1.5,"unit":"usd"},"api_ms":2000,"outcome":"finished"}`,
		"lch-261104120000000002-bbbbbbbb.json": `{"schema_version":1,"orchestrator":{"kind":"session","id":"sess-1","name":"Alice"},"run":"run-1","lane":"lane-1","await":"r.json","role":"implementer","runner":"claude","route":"subscription","cost":{"amount":9,"unit":"usd"},"outcome":"finished"}`,
		"lch-261104120000000003-cccccccc.json": "{\"schema_version\":1,\"orchestrator\":{\"kind\":\"run\",\"id\":\"run-2\"},\"run\":\"run-2\",\"role\":\"drain-judge\",\"runner\":\"opencode\",\"route\":\"unknown\",\"route_why\":\"no key\",\"outcome\":\"failed\",\"reason\":\"boom\\u001b[2J\"}",
	} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(line+"\n"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	return t.TempDir()
}

// Bare, the verb lists this session's launches, with totals in which only the
// key's cost is money, and says what its figures count.
func TestAgentsBareListsThisSession(t *testing.T) {
	dir := agentsWorld(t)
	t.Setenv(runner.HostSessionEnv, "sess-1")
	code, out, errOut := reserveRun(t, dir, "agents")
	if code != 0 {
		t.Fatalf("exit %d, stderr %q", code, errOut)
	}
	for _, want := range []string{"sess-1", "Alice", "implementer", "build-key", "claude-fake", "$1.50", "10 tokens", "$9.00",
		"estimate", "invoice", runner.TokenBasis, "two launches for one await"} {
		if !strings.Contains(out, want) {
			t.Errorf("the listing does not say %q:\n%s", want, out)
		}
	}
	if strings.Contains(out, "drain-judge") {
		t.Errorf("another run's launch is listed:\n%s", out)
	}
}

// --run, --session and --all select; --json carries the same listing.
func TestAgentsSelectsAndSpeaksJSON(t *testing.T) {
	dir := agentsWorld(t)
	for _, tc := range []struct {
		args []string
		want int
	}{
		{[]string{"--run", "run-1"}, 2},
		{[]string{"--run", "run-2"}, 1},
		{[]string{"--session", "sess-1"}, 2},
		{[]string{"--all"}, 3},
		{[]string{"--session", "nobody"}, 0},
	} {
		code, out, errOut := reserveRun(t, dir, append([]string{"agents", "--json"}, tc.args...)...)
		if code != 0 {
			t.Fatalf("%v: exit %d, stderr %q", tc.args, code, errOut)
		}
		var got struct {
			Entries []runner.LedgerEntry `json:"entries"`
			Totals  runner.LedgerTotals  `json:"totals"`
		}
		if err := json.Unmarshal([]byte(out), &got); err != nil {
			t.Fatalf("%v: %v\n%s", tc.args, err, out)
		}
		if len(got.Entries) != tc.want || got.Totals.Launches != tc.want {
			t.Errorf("%v: %d entries, totals %+v, want %d", tc.args, len(got.Entries), got.Totals, tc.want)
		}
	}
}

// What an entry says reaches a terminal sanitised: a file in the store is data.
func TestAgentsSanitisesWhatAnEntrySays(t *testing.T) {
	dir := agentsWorld(t)
	code, out, errOut := reserveRun(t, dir, "agents", "--all")
	if code != 0 {
		t.Fatalf("exit %d, stderr %q", code, errOut)
	}
	if strings.Contains(out, "\x1b") {
		t.Errorf("a control sequence reached the terminal: %q", out)
	}
	if !strings.Contains(out, "boom") {
		t.Errorf("the reason is missing:\n%s", out)
	}
}

// An entry it could not read is named, and the verb still lists the rest.
func TestAgentsNamesAnEntryItCouldNotRead(t *testing.T) {
	dir := agentsWorld(t)
	bad := "lch-261104120000000004-dddddddd.json"
	if err := os.WriteFile(filepath.Join(abcdhome.Path(os.Getenv("HOME"), "agents"), bad), []byte("{ no\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	code, out, errOut := reserveRun(t, dir, "agents", "--all")
	if code != 0 {
		t.Fatalf("exit %d, stderr %q", code, errOut)
	}
	if !strings.Contains(out, bad) {
		t.Errorf("the unreadable entry is not named:\n%s", out)
	}
}

// With no session to read, bare refuses and names the flags; two scopes at once
// and a positional refuse too. Each exits 2 and writes nothing.
func TestAgentsRefusesWhatItCannotAnswer(t *testing.T) {
	dir := agentsWorld(t)
	code, _, errOut := reserveRun(t, dir, "agents")
	if code != 2 || !strings.Contains(errOut, "--all") {
		t.Errorf("no session: exit %d, stderr %q", code, errOut)
	}
	for _, args := range [][]string{{"agents", "--all", "--run", "r"}, {"agents", "--run", "r", "--session", "s"}, {"agents", "stray"}} {
		if code, _, errOut := reserveRun(t, dir, args...); code != 2 {
			t.Errorf("%v: exit %d, stderr %q", args, code, errOut)
		}
	}
}

// With no store yet the listing is empty, and reading makes none.
func TestAgentsOnAMachineWithNoLedger(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv(runner.HostSessionEnv, "")
	code, out, errOut := reserveRun(t, t.TempDir(), "agents", "--all")
	if code != 0 || !strings.Contains(out, "no launches") {
		t.Errorf("exit %d, stdout %q, stderr %q", code, out, errOut)
	}
	if _, err := os.Stat(abcdhome.Path(home, "agents")); !os.IsNotExist(err) {
		t.Errorf("a read made the store: %v", err)
	}
}
