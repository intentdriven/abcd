// Package stub is the claude runner the AI-written interviews' tests stand in
// (internal/surface/cli/written_interview_test.go), in one place for the two
// programs that play it: a small binary built from ../main.go once per test
// process, and, where no go toolchain is found to build that, the test binary
// itself, which TestMain hands to Run.
package stub

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"
)

// ScriptEnv names the stub runner's script directory.
const ScriptEnv = "ABCD_CLI_INTERVIEW_STUB"

// Runs reports whether a program started with args under the script dir is
// the stub runner: a script named and claude's own print flag.
func Runs(dir string, args []string) bool {
	return dir != "" && slices.Contains(args, "--print")
}

// Run plays the claude runner for one turn: it copies the script's
// turn-<n>.json to the receipt path the prompt names (no file, no receipt),
// keeps the turn's brief and its argv beside the script, and reports success.
// A turn-<n>.also.json, a map of path to content, makes the turn write those
// files too, relative to the directory the runner was started in; a
// turn-<n>.sleep makes it mark turn-<n>.started beside the script and sleep,
// so a test can stop it mid-run. args is the program's whole argument list,
// its name first.
func Run(dir string, args []string, stdout io.Writer) int {
	prompt := args[len(args)-1]
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
	if argv, err := json.Marshal(args[1:]); err == nil {
		_ = os.WriteFile(filepath.Join(dir, turn+".argv.json"), argv, 0o600)
	}
	if body, err := os.ReadFile(filepath.Join(dir, turn+".json")); err == nil {
		if err := os.WriteFile(receipt, body, 0o600); err != nil {
			return 3
		}
	}
	if raw, err := os.ReadFile(filepath.Join(dir, turn+".also.json")); err == nil {
		var also map[string]string
		if err := json.Unmarshal(raw, &also); err != nil {
			return 3
		}
		for p, body := range also {
			target := filepath.FromSlash(p)
			if err := os.MkdirAll(filepath.Dir(target), 0o700); err != nil {
				return 3
			}
			if err := os.WriteFile(target, []byte(body), 0o600); err != nil {
				return 3
			}
		}
	}
	if _, err := os.Stat(filepath.Join(dir, turn+".sleep")); err == nil {
		_ = os.WriteFile(filepath.Join(dir, turn+".started"), nil, 0o600)
		time.Sleep(30 * time.Second)
	}
	fmt.Fprintln(stdout, `{"type":"system","subtype":"init","session_id":"stub-session-1","model":"stub-model"}`)
	fmt.Fprintln(stdout, `{"type":"result","subtype":"success","is_error":false,"result":"done","session_id":"stub-session-1"}`)
	return 0
}
