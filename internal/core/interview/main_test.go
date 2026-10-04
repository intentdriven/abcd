package interview

// main_test.go is the stub runner the AI-written interview's tests drive. No
// test reaches a real harness or a paid model: the test binary re-executes
// itself under the name claude, which a test puts on PATH, and TestMain,
// seeing stubScriptEnv and claude's own flags, plays that harness. Each turn
// it copies the script's turn-<n>.json to the receipt path the prompt names
// (no file: no receipt), keeps a copy of the turn's brief beside the script,
// and prints the events of a successful run. A turn-<n>.also.json in the
// script, a map of path to content, makes the turn write those files too: a
// path under "turns/" lands in the turn's own directory, any other in the
// directory the runner was started in. A turn-<n>.sleep in the script makes
// the turn mark turn-<n>.started beside the script and sleep, so a test can
// stop it mid-run.

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"
)

const stubScriptEnv = "ABCD_INTERVIEW_STUB_SCRIPT"

func TestMain(m *testing.M) {
	if dir := os.Getenv(stubScriptEnv); dir != "" && slices.Contains(os.Args, "--print") {
		os.Exit(stubRunner(dir))
	}
	os.Exit(m.Run())
}

func stubRunner(dir string) int {
	prompt := os.Args[len(os.Args)-1]
	field := func(prefix string) string {
		sc := bufio.NewScanner(strings.NewReader(prompt))
		for sc.Scan() {
			if v, ok := strings.CutPrefix(sc.Text(), prefix); ok {
				return strings.TrimSpace(v)
			}
		}
		return ""
	}
	receipt, brief := field("Receipt: "), field("Brief: ")
	turn := strings.TrimSuffix(filepath.Base(receipt), ".receipt.json")
	if b, err := os.ReadFile(brief); err == nil {
		_ = os.WriteFile(filepath.Join(dir, turn+".brief.seen.md"), b, 0o600)
	}
	if body, err := os.ReadFile(filepath.Join(dir, turn+".json")); err == nil {
		if err := os.WriteFile(receipt, body, 0o600); err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 3
		}
	}
	if raw, err := os.ReadFile(filepath.Join(dir, turn+".also.json")); err == nil {
		var also map[string]string
		if err := json.Unmarshal(raw, &also); err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 3
		}
		for p, body := range also {
			target := filepath.FromSlash(p)
			if rest, ok := strings.CutPrefix(p, "turns/"); ok {
				target = filepath.Join(filepath.Dir(receipt), filepath.FromSlash(rest))
			}
			if err := os.MkdirAll(filepath.Dir(target), 0o700); err != nil {
				fmt.Fprintln(os.Stderr, err)
				return 3
			}
			if err := os.WriteFile(target, []byte(body), 0o600); err != nil {
				fmt.Fprintln(os.Stderr, err)
				return 3
			}
		}
	}
	if _, err := os.Stat(filepath.Join(dir, turn+".sleep")); err == nil {
		_ = os.WriteFile(filepath.Join(dir, turn+".started"), nil, 0o600)
		time.Sleep(30 * time.Second)
	}
	fmt.Println(`{"type":"system","subtype":"init","session_id":"stub-session-1","model":"stub-model"}`)
	fmt.Println(`{"type":"result","subtype":"success","is_error":false,"result":"done","session_id":"stub-session-1"}`)
	return 0
}

// stubOnPath puts the stub runner on PATH as claude and points it at a script
// of receipts, one per turn (turn-1.json, turn-2.json, ...), returning the
// script's directory.
func stubOnPath(t *testing.T, receipts ...string) string {
	t.Helper()
	self, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	bin, script := t.TempDir(), t.TempDir()
	if err := os.Symlink(self, filepath.Join(bin, "claude")); err != nil {
		t.Fatal(err)
	}
	for i, r := range receipts {
		if err := os.WriteFile(filepath.Join(script, fmt.Sprintf("turn-%d.json", i+1)), []byte(r), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	t.Setenv("PATH", bin)
	t.Setenv(stubScriptEnv, script)
	return script
}

// stubAlso makes the script's turn write files besides its receipt.
func stubAlso(t *testing.T, script string, turn int, files map[string]string) {
	t.Helper()
	b, err := json.Marshal(files)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(script, fmt.Sprintf("turn-%d.also.json", turn)), b, 0o600); err != nil {
		t.Fatal(err)
	}
}

// stubSleep makes the script's turn sleep once it has started, marking
// turn-<n>.started beside the script.
func stubSleep(t *testing.T, script string, turn int) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(script, fmt.Sprintf("turn-%d.sleep", turn)), nil, 0o600); err != nil {
		t.Fatal(err)
	}
}
