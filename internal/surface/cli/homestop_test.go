package cli

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/abcdhome"
	"github.com/intentdriven/abcd/internal/gittest"
)

// homestop_test.go — D2 of itd-2610030720038073: with an old ~/.abcd standing,
// every command, every hook and the status line stop before writing anything,
// name the folder and the one rename command, and create no new folder; with
// both folders standing they stop and name both (spc-2610031309233367, "The
// stop", and its open questions 3 and 6).

// stoppedHome stands up a temporary HOME holding an old ~/.abcd with a file in
// it (and, when both is set, a ~/.abcd.noindex beside it), a committed
// repository the verbs run in, and a session transcript a staging hook would
// otherwise capture into the home. It returns the home, the repository and the
// transcript's path.
func stoppedHome(t *testing.T, both bool) (home, repo, transcript string) {
	t.Helper()
	home = t.TempDir()
	t.Setenv("HOME", home)
	old := filepath.Join(home, ".abcd")
	if err := os.MkdirAll(filepath.Join(old, "runs"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(old, "trusted-roots"), []byte("# mine\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if both {
		if err := os.MkdirAll(abcdhome.Path(home, "lab"), 0o700); err != nil {
			t.Fatal(err)
		}
	}
	r := gittest.NewRepo(t)
	r.Write("README.md", "fixture\n")
	r.Commit("init")
	t.Chdir(r.Root())
	transcript = filepath.Join(t.TempDir(), "session.jsonl")
	if err := os.WriteFile(transcript, []byte(`{"type":"user","message":"hello"}`+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	return home, r.Root(), transcript
}

// homeTree renders every entry under home with its size and modification
// time, so a test can prove a run left the home exactly as it found it.
func homeTree(t *testing.T, home string) string {
	t.Helper()
	var b strings.Builder
	err := filepath.Walk(home, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(home, path)
		b.WriteString(rel + " " + info.Mode().String() + " " + strconv.FormatInt(info.Size(), 10) + " " + info.ModTime().String() + "\n")
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return b.String()
}

// stopRun runs one invocation through the binary's front door with stdin
// bound to payload, and asserts the home is untouched by it.
func stopRun(t *testing.T, home, payload string, args ...string) (code int, stdout, stderr string) {
	t.Helper()
	before := homeTree(t, home)
	var out, errOut bytes.Buffer
	code = run(args, strings.NewReader(payload), &out, &errOut)
	if after := homeTree(t, home); after != before {
		t.Fatalf("abcd %s changed the home while stopped:\nbefore:\n%s\nafter:\n%s", strings.Join(args, " "), before, after)
	}
	return code, out.String(), errOut.String()
}

// expectedStop is the stop the home's state calls for, checked against what
// the spec says it names, so a wrong line cannot pass for the right one.
func expectedStop(t *testing.T, home string, both bool) *abcdhome.Stop {
	t.Helper()
	stop := abcdhome.Check(home)
	if stop == nil {
		t.Fatal("abcdhome.Check reports no stop with ~/.abcd standing")
	}
	if stop.Both != both {
		t.Fatalf("Check(home).Both = %v, want %v", stop.Both, both)
	}
	want := []string{"~/.abcd ", "~/.abcd.noindex", abcdhome.RepairCommand}
	if both {
		want = []string{"Both ~/.abcd and ~/.abcd.noindex exist", abcdhome.RepairCommand}
	} else {
		want = append(want, abcdhome.RenameCommand)
	}
	for _, w := range want {
		if !strings.Contains(stop.Line, w) {
			t.Fatalf("the stop line %q does not name %q", stop.Line, w)
		}
	}
	return stop
}

// ordinaryVerbs are verbs that would write: the run state and its log in the
// home, a capture in the repository, the install's records in both.
var ordinaryVerbs = [][]string{
	{"implement", "join", "--session", "alpha", "--role", "first"},
	{"implement", "log", "stop", "--session", "alpha", "--field", "reason=x"},
	{"capture", "the home rename stranded a worktree", "--severity", "minor", "--category", "bug", "--remedy", "repair it"},
	{"ahoy", "install", "--yes", "--adopt"},
	{"history", "drain"},
	{},
}

func assertOrdinaryStops(t *testing.T, home string, stop *abcdhome.Stop) {
	t.Helper()
	for _, args := range ordinaryVerbs {
		t.Run("plain "+strings.Join(args, " "), func(t *testing.T) {
			code, out, errOut := stopRun(t, home, "", args...)
			if code != 1 || out != "" || errOut != "abcd: "+stop.Line+"\n" {
				t.Fatalf("exit %d\nstdout %q\nstderr %q\nwant exit 1, nothing on stdout and the stop line on stderr", code, out, errOut)
			}
		})
		t.Run("json "+strings.Join(args, " "), func(t *testing.T) {
			code, out, errOut := stopRun(t, home, "", append(append([]string{}, args...), "--json")...)
			var env struct {
				Abcd     string `json:"abcd"`
				Error    string `json:"error"`
				ExitCode int    `json:"exit_code"`
			}
			if err := json.Unmarshal([]byte(out), &env); err != nil {
				t.Fatalf("stdout is not the error envelope (%v): %q", err, out)
			}
			if code != 1 || env.Abcd != "error" || env.Error != stop.Line || env.ExitCode != 1 || errOut != "" {
				t.Fatalf("exit %d, envelope %+v, stderr %q; want exit 1 and the stop line in the envelope", code, env, errOut)
			}
		})
	}
}

// TestOldHomeStopsEveryVerb: an ordinary verb that would write stops with the
// line on stderr and exit 1, or in the --json error envelope on stdout, and
// the home's tree is identical afterwards, with no ~/.abcd.noindex created.
func TestOldHomeStopsEveryVerb(t *testing.T) {
	home, _, _ := stoppedHome(t, false)
	stop := expectedStop(t, home, false)
	assertOrdinaryStops(t, home, stop)
	if _, err := os.Lstat(abcdhome.Path(home)); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("a stopped run created %s: %v", abcdhome.Display(), err)
	}
}

// guardPayload is a pre-tool-use payload for the shell tool.
func guardPayload(t *testing.T, cwd, command string) string {
	t.Helper()
	b, err := json.Marshal(map[string]any{
		"cwd": cwd, "tool_name": "Bash", "tool_input": map[string]any{"command": command},
	})
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

// TestOldHomeStopsEveryHook: each hook verb answers in its own form — the
// prompt and session-start hooks put the line on stdout for the agent, the
// staging hooks report an uncaptured result naming it, the shell guard blocks
// everything but the exact rename command, and the status line shows the short
// form — every one of them leaving the home exactly as it was.
func TestOldHomeStopsEveryHook(t *testing.T) {
	home, repo, transcript := stoppedHome(t, false)
	stop := expectedStop(t, home, false)
	hookPayload, err := json.Marshal(map[string]any{
		"session_id": "s1", "cwd": repo, "prompt": "commit the change",
		"transcript_path": transcript, "agent_transcript_path": transcript, "agent_id": "a1",
		"hook_event_name": "SessionStart",
	})
	if err != nil {
		t.Fatal(err)
	}
	payload := string(hookPayload)

	for _, verb := range []string{"prompt-router", "session-start"} {
		t.Run(verb, func(t *testing.T) {
			code, out, errOut := stopRun(t, home, payload, "hook", verb)
			if code != 0 || out != stop.Line+"\n" || errOut != "" {
				t.Fatalf("exit %d\nstdout %q\nstderr %q\nwant exit 0 and the stop line on stdout, the channel the agent reads", code, out, errOut)
			}
		})
	}
	t.Run("prompt-router --json", func(t *testing.T) {
		code, out, _ := stopRun(t, home, payload, "hook", "prompt-router", "--json")
		var v routerView
		if err := json.Unmarshal([]byte(out), &v); err != nil || code != 0 || v.Text != stop.Line+"\n" || v.Error != stop.Line || len(v.Injected) != 0 {
			t.Fatalf("exit %d, envelope %+v (%v); want the stop line as the text and the error", code, v, err)
		}
	})
	for _, verb := range []string{"prompt-router-reset", "session-end", "subagent-stop"} {
		t.Run(verb, func(t *testing.T) {
			code, out, errOut := stopRun(t, home, payload, "hook", verb)
			if code != 0 || out != "" || errOut != "abcd: "+stop.Line+"\n" {
				t.Fatalf("exit %d\nstdout %q\nstderr %q\nwant exit 0 and the stop line on stderr", code, out, errOut)
			}
			code, out, _ = stopRun(t, home, payload, "hook", verb, "--json")
			var r hookStageResult
			if err := json.Unmarshal([]byte(out), &r); err != nil || code != 0 || r.Hook != verb || r.Outcome != hookOutcomeNotCaptured || r.Captured || r.Reason != stop.Line {
				t.Fatalf("exit %d, result %+v (%v); want the uncaptured result with the stop line as its reason", code, r, err)
			}
		})
	}

	t.Run("guard hook refuses an ordinary command", func(t *testing.T) {
		code, out, errOut := stopRun(t, home, guardPayload(t, repo, "ls"), "guard", "hook")
		if code != 2 || out != "" || errOut != "abcd: "+stop.Line+"\n" {
			t.Fatalf("exit %d\nstdout %q\nstderr %q\nwant the blocking status and the stop line on stderr", code, out, errOut)
		}
	})
	t.Run("guard hook refuses a question", func(t *testing.T) {
		q := `{"cwd":"` + repo + `","tool_name":"AskUserQuestion","tool_input":{"questions":[{"question":"Which?","header":"Product Q1","options":[{"label":"a","description":"a"},{"label":"b","description":"b"}],"multiSelect":false}]}}`
		code, _, errOut := stopRun(t, home, q, "guard", "hook")
		if code != 2 || errOut != "abcd: "+stop.Line+"\n" {
			t.Fatalf("exit %d, stderr %q; want the blocking status and the stop line", code, errOut)
		}
	})
	t.Run("guard hook refuses an unreadable payload", func(t *testing.T) {
		code, _, errOut := stopRun(t, home, "{not json", "guard", "hook")
		if code != 2 || errOut != "abcd: "+stop.Line+"\n" {
			t.Fatalf("exit %d, stderr %q; want the blocking status and the stop line", code, errOut)
		}
	})
	t.Run("guard hook refuses the rename inside a longer command", func(t *testing.T) {
		code, _, _ := stopRun(t, home, guardPayload(t, repo, abcdhome.RenameCommand+"; rm -r ~/x"), "guard", "hook")
		if code != 2 {
			t.Fatalf("exit %d; only the exact rename command is admitted", code)
		}
	})
	t.Run("guard hook refuses the rename padded with a space the shell keeps", func(t *testing.T) {
		// A shell drops only its own white space around a command; a Unicode
		// space it keeps would rename the folder to a name with an invisible
		// character in it, and the stop would lift.
		for _, pad := range []string{"\u00a0", "\u3000", "\u2003", "\u0085"} {
			for _, cmd := range []string{abcdhome.RenameCommand + pad, pad + abcdhome.RenameCommand} {
				code, _, _ := stopRun(t, home, guardPayload(t, repo, cmd), "guard", "hook")
				if code != 2 {
					t.Errorf("exit %d for %q; only the exact rename command is admitted", code, cmd)
				}
			}
		}
	})
	t.Run("guard hook admits the rename with a shell's own white space around it", func(t *testing.T) {
		code, _, _ := stopRun(t, home, guardPayload(t, repo, " \t"+abcdhome.RenameCommand+"\n"), "guard", "hook")
		if code != 0 {
			t.Fatalf("exit %d; the rename with surrounding spaces, tabs or newlines is the rename", code)
		}
	})
	t.Run("guard hook admits the rename command", func(t *testing.T) {
		code, out, errOut := stopRun(t, home, guardPayload(t, repo, abcdhome.RenameCommand), "guard", "hook")
		if code != 0 || out != "" || errOut != "" {
			t.Fatalf("exit %d\nstdout %q\nstderr %q\nwant the rename admitted silently", code, out, errOut)
		}
	})

	t.Run("statusline", func(t *testing.T) {
		code, out, errOut := stopRun(t, home, `{"cwd":"`+repo+`"}`, "statusline")
		if code != 0 || out != "abcd stopped: rename ~/.abcd to ~/.abcd.noindex\n" || errOut != "" {
			t.Fatalf("exit %d\nstdout %q\nstderr %q\nwant the short form on stdout", code, out, errOut)
		}
	})

	if _, err := os.Lstat(abcdhome.Path(home)); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("a stopped hook created %s: %v", abcdhome.Display(), err)
	}
}

// TestBothHomesStopAndNameBoth: with both folders standing, every entry stops
// with the second line, which names both; the guard refuses even the rename
// command, which would move the old folder into the new one.
func TestBothHomesStopAndNameBoth(t *testing.T) {
	home, repo, _ := stoppedHome(t, true)
	stop := expectedStop(t, home, true)
	assertOrdinaryStops(t, home, stop)

	code, out, errOut := stopRun(t, home, `{"session_id":"s1","cwd":"`+repo+`"}`, "hook", "prompt-router")
	if code != 0 || out != stop.Line+"\n" || errOut != "" {
		t.Fatalf("prompt-router: exit %d, stdout %q, stderr %q", code, out, errOut)
	}
	code, _, errOut = stopRun(t, home, guardPayload(t, repo, abcdhome.RenameCommand), "guard", "hook")
	if code != 2 || errOut != "abcd: "+stop.Line+"\n" {
		t.Fatalf("guard hook with both folders standing admitted %q: exit %d, stderr %q", abcdhome.RenameCommand, code, errOut)
	}
	code, out, _ = stopRun(t, home, "", "statusline")
	if code != 0 || out != stop.Short+"\n" || !strings.Contains(stop.Short, "~/.abcd.noindex") || strings.Contains(stop.Short, "rename") {
		t.Fatalf("statusline: exit %d, stdout %q", code, out)
	}
}
