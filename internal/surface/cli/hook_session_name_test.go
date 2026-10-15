package cli

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/intentdriven/abcd/internal/abcdhome"
	"github.com/intentdriven/abcd/internal/core/machineload"
	"github.com/intentdriven/abcd/internal/core/sessionname"
)

// nameFixture is one machine for the naming hooks: a hermetic HOME holding
// one waiting inbox report (so session start has a stdout line of its own to
// carry), a working directory with no persona list, and a process table in
// which every hook runs under a shell whose parent is the host.
type nameFixture struct {
	home, cwd string
	procs     []machineload.Proc
	tableErr  error
}

func newNameFixture(t *testing.T) *nameFixture {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("CLAUDE_CONFIG_DIR", "")
	t.Setenv("ABCD_RULES_STATE_DIR", t.TempDir())
	noAmbientPluginRoot(t)
	inbox := abcdhome.Path(home, "inbox")
	if err := os.MkdirAll(filepath.Join(inbox, "promoted"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(inbox, "1234567890123456-"+strings.Repeat("a", 40)+".md"), []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	f := &nameFixture{home: home, cwd: t.TempDir()}
	prev := sessionNameMachine
	sessionNameMachine = func() (sessionname.Machine, bool) { return f.machine(), true }
	t.Cleanup(func() { sessionNameMachine = prev })
	return f
}

// hookShell is the pid of the shell every fixture hook runs under, whose
// parent is the session's host.
const hookShell = 4242

// launch makes host the process the next hook runs under.
func (f *nameFixture) launch(host int) {
	f.procs = append(f.procs[:0:0],
		machineload.Proc{PID: host, PPID: 1, Name: "claude", Age: time.Hour},
		machineload.Proc{PID: hookShell, PPID: host, Name: "sh", Age: time.Second})
}

func (f *nameFixture) machine() sessionname.Machine {
	return sessionname.Machine{
		Home: f.home,
		From: hookShell,
		ReadTable: func() (sessionname.Table, error) {
			if f.tableErr != nil {
				return sessionname.Table{}, f.tableErr
			}
			return sessionname.Table{Procs: f.procs, ReadAt: time.Now()}, nil
		},
		LockTimeout: 2 * time.Second,
	}
}

// startJSON is a SessionStart payload; extra fields are merged in.
func startJSON(t *testing.T, sid, cwd string, extra map[string]string) string {
	t.Helper()
	m := map[string]string{"session_id": sid, "cwd": cwd, "hook_event_name": "SessionStart"}
	for k, v := range extra {
		m[k] = v
	}
	b, err := json.Marshal(m)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

// promptJSON is a UserPromptSubmit payload carrying the session's title.
func promptJSON(t *testing.T, sid, cwd, prompt, title string) string {
	t.Helper()
	b, err := json.Marshal(map[string]string{
		"session_id": sid, "cwd": cwd, "prompt": prompt,
		"hook_event_name": "UserPromptSubmit", "session_title": title,
	})
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

// titled is the hook output that sets a title.
type titled struct {
	HookSpecificOutput struct {
		HookEventName     string `json:"hookEventName"`
		AdditionalContext string `json:"additionalContext"`
		SessionTitle      string `json:"sessionTitle"`
	} `json:"hookSpecificOutput"`
}

// decodeTitled reads stdout as exactly one JSON object of the titled shape.
func decodeTitled(t *testing.T, stdout string) titled {
	t.Helper()
	dec := json.NewDecoder(strings.NewReader(stdout))
	dec.DisallowUnknownFields()
	var out titled
	if err := dec.Decode(&out); err != nil {
		t.Fatalf("stdout is not one hook-output object: %v\n%s", err, stdout)
	}
	if dec.More() {
		t.Fatalf("stdout carries more than one JSON value:\n%s", stdout)
	}
	return out
}

// unnamedStart is what session start prints when naming does not run at all:
// the bytes every unnamed outcome must reproduce.
func unnamedStart(t *testing.T, payload string) string {
	t.Helper()
	prev := sessionNameMachine
	sessionNameMachine = func() (sessionname.Machine, bool) { return sessionname.Machine{}, false }
	defer func() { sessionNameMachine = prev }()
	out, _, code := runSessionStart(payload, "hook", "session-start")
	if code != 0 {
		t.Fatalf("unnamed session start exited %d", code)
	}
	return out
}

// TestSessionStartSetsTheTitle is C1: a new session's start hook output is one
// object setting sessionTitle to Alice, carrying the lines the hook would
// otherwise have printed as its additional context.
func TestSessionStartSetsTheTitle(t *testing.T) {
	f := newNameFixture(t)
	f.launch(1000)
	payload := startJSON(t, "s1", f.cwd, map[string]string{"source": "startup"})
	want := unnamedStart(t, payload)
	if !strings.Contains(want, "wait in the inbox") {
		t.Fatalf("the fixture's plain session start has no stdout line to carry: %q", want)
	}
	stdout, stderr, code := runSessionStart(payload, "hook", "session-start")
	if code != 0 {
		t.Fatalf("exit %d, stderr %q", code, stderr)
	}
	got := decodeTitled(t, stdout)
	if got.HookSpecificOutput.HookEventName != "SessionStart" || got.HookSpecificOutput.SessionTitle != "Alice" {
		t.Fatalf("hook output = %+v, want SessionStart titled Alice", got)
	}
	if got.HookSpecificOutput.AdditionalContext != want {
		t.Fatalf("additionalContext = %q, want the plain output %q", got.HookSpecificOutput.AdditionalContext, want)
	}
	// C2 through the hook: the next session, under another host while the
	// first still runs, is Bob.
	f.launch(2000)
	f.procs = append(f.procs, machineload.Proc{PID: 1000, PPID: 1, Name: "claude", Age: time.Hour})
	stdout, _, _ = runSessionStart(startJSON(t, "s2", f.cwd, map[string]string{"source": "startup"}), "hook", "session-start")
	if got := decodeTitled(t, stdout).HookSpecificOutput.SessionTitle; got != "Bob" {
		t.Fatalf("the second session is titled %q, want Bob", got)
	}
}

// TestSessionStartUnnamedIsByteIdentical is C4, C5, C11 and C12 at the hook:
// whenever nothing is named, stdout is byte for byte what it is without
// naming; a failure adds one stderr line, and the exit stays 0.
func TestSessionStartUnnamedIsByteIdentical(t *testing.T) {
	for name, c := range map[string]struct {
		extra    map[string]string
		tableErr error
		noted    bool
	}{
		"a person's own name": {extra: map[string]string{"source": "startup", "session_title": "Zed"}},
		"a sub-agent":         {extra: map[string]string{"source": "startup", "agent_id": "ag-1"}},
		"compaction":          {extra: map[string]string{"source": "compact", "session_title": "Alice"}},
		"a resume":            {extra: map[string]string{"source": "resume", "session_title": "Bob"}},
		"no source":           {extra: map[string]string{}},
		"process table fails": {extra: map[string]string{"source": "startup"}, tableErr: errors.New("ps failed"), noted: true},
		"unsupported":         {extra: map[string]string{"source": "startup"}, tableErr: machineload.ErrUnsupported, noted: true},
	} {
		t.Run(name, func(t *testing.T) {
			f := newNameFixture(t)
			f.launch(1000)
			f.tableErr = c.tableErr
			payload := startJSON(t, "s1", f.cwd, c.extra)
			want := unnamedStart(t, payload)
			stdout, stderr, code := runSessionStart(payload, "hook", "session-start")
			if code != 0 {
				t.Fatalf("exit %d", code)
			}
			if stdout != want {
				t.Fatalf("stdout = %q, want the unnamed %q", stdout, want)
			}
			if n := strings.Count(stderr, "abcd session name:"); (n == 1) != c.noted || n > 1 {
				t.Fatalf("stderr carries %d naming notes, want noted=%v:\n%s", n, c.noted, stderr)
			}
		})
	}
}

// TestPromptRouterResendsTheTitleOnce is C3: the first prompt of a session
// abcd named sets the same title again, which is what puts it in the session
// list, carrying the router's text as additional context; later prompts print
// exactly what they print today.
func TestPromptRouterResendsTheTitleOnce(t *testing.T) {
	f := newNameFixture(t)
	f.launch(1000)
	runSessionStart(startJSON(t, "s1", f.cwd, map[string]string{"source": "startup"}), "hook", "session-start")

	// The reference: the same first prompt, from a session with no title.
	want, _ := runHook(t, promptJSON(t, "other", f.cwd, "commit this change", ""), "hook", "prompt-router")
	if want == "" {
		t.Fatal("the reference prompt injected nothing, so the carried text proves nothing")
	}

	stdout, _ := runHook(t, promptJSON(t, "s1", f.cwd, "commit this change", "Alice"), "hook", "prompt-router")
	got := decodeTitled(t, stdout)
	if got.HookSpecificOutput.HookEventName != "UserPromptSubmit" || got.HookSpecificOutput.SessionTitle != "Alice" {
		t.Fatalf("first prompt output = %+v, want UserPromptSubmit titled Alice", got)
	}
	if got.HookSpecificOutput.AdditionalContext != want {
		t.Fatalf("additionalContext = %q, want the plain text %q", got.HookSpecificOutput.AdditionalContext, want)
	}

	second, _ := runHook(t, promptJSON(t, "s1", f.cwd, "paint a landscape", "Alice"), "hook", "prompt-router")
	if strings.Contains(second, "sessionTitle") {
		t.Fatalf("the second prompt set the title again: %q", second)
	}
}

// TestPromptRouterLeavesARenamedSessionAlone is C4 at the prompt: a session the
// person renamed before its first prompt is owed nothing, and its output is
// the plain text.
func TestPromptRouterLeavesARenamedSessionAlone(t *testing.T) {
	f := newNameFixture(t)
	f.launch(1000)
	runSessionStart(startJSON(t, "s1", f.cwd, map[string]string{"source": "startup"}), "hook", "session-start")
	stdout, _ := runHook(t, promptJSON(t, "s1", f.cwd, "paint a landscape", "Dave"), "hook", "prompt-router")
	if strings.Contains(stdout, "sessionTitle") || strings.HasPrefix(stdout, "{") {
		t.Fatalf("a renamed session's prompt set a title: %q", stdout)
	}
}

// TestPromptRouterJSONCarriesTheSessionTitle: the --json envelope names the
// re-sent title on the prompt that re-sends it, and carries no field otherwise.
func TestPromptRouterJSONCarriesTheSessionTitle(t *testing.T) {
	f := newNameFixture(t)
	f.launch(1000)
	runSessionStart(startJSON(t, "s1", f.cwd, map[string]string{"source": "startup"}), "hook", "prompt-router-reset")
	runSessionStart(startJSON(t, "s1", f.cwd, map[string]string{"source": "startup"}), "hook", "session-start")
	for i, want := range []string{"Alice", ""} {
		out, _ := runHook(t, promptJSON(t, "s1", f.cwd, "paint a landscape", "Alice"), "hook", "prompt-router", "--json")
		var env map[string]json.RawMessage
		if err := json.Unmarshal([]byte(out), &env); err != nil {
			t.Fatalf("prompt %d: not one envelope: %v\n%s", i+1, err, out)
		}
		raw, present := env["session_title"]
		switch {
		case want == "" && present:
			t.Fatalf("prompt %d: the envelope carries session_title %s, want none", i+1, raw)
		case want != "" && string(raw) != `"`+want+`"`:
			t.Fatalf("prompt %d: session_title = %s, want %q", i+1, raw, want)
		}
	}
}

// TestPromptRouterResetIsSilentOnStdout holds the spec's constraint on the
// session-start script: it runs `hook prompt-router-reset` after `hook
// session-start` on the same stdout, so a single byte from the reset would
// turn the titled object into something the host cannot parse.
func TestPromptRouterResetIsSilentOnStdout(t *testing.T) {
	t.Setenv("ABCD_RULES_STATE_DIR", t.TempDir())
	for _, payload := range []string{
		startJSON(t, "s1", t.TempDir(), map[string]string{"source": "startup"}),
		startJSON(t, "s1", t.TempDir(), map[string]string{"source": "compact", "session_title": "Alice"}),
		"not json",
	} {
		stdout, _ := runHook(t, payload, "hook", "prompt-router-reset")
		if stdout != "" {
			t.Fatalf("prompt-router-reset wrote %q to stdout for %s", stdout, payload)
		}
	}
}
