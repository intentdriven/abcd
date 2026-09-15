package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/mode"
	"github.com/intentdriven/abcd/internal/core/statusline"
)

// managedCheckout is the ground the mode, statusline and board tests stand
// on: a git checkout carrying the marker block (so ahoy.Managed says yes) and
// the local-ephemeral tier (so the mode store has somewhere to write), with
// HOME redirected so nothing reads the developer's own settings. It returns
// the root as git names it and leaves the process standing in it.
func managedCheckout(t *testing.T) string {
	t.Helper()
	t.Setenv("HOME", t.TempDir())
	t.Setenv("ABCD_PLUGIN_ROOT", "")
	t.Setenv("CLAUDE_PLUGIN_ROOT", "")
	repo := t.TempDir()
	gitInitAt(t, repo)
	repo = realPath(t, repo)
	marker := "# Project\n\n<!-- BEGIN ABCD -->\nx\n<!-- END ABCD -->\n"
	if err := os.WriteFile(filepath.Join(repo, "CLAUDE.md"), []byte(marker), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(repo, filepath.FromSlash(mode.TierRelPath)), 0o700); err != nil {
		t.Fatal(err)
	}
	t.Chdir(repo)
	return repo
}

// writeUserSettings lays ~/.abcd/statusline.json under the sandboxed HOME.
func writeUserSettings(t *testing.T, body string) {
	t.Helper()
	dir := filepath.Join(os.Getenv("HOME"), ".abcd")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, statusline.SettingsFileName), []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
}

// runSplit runs the tree with stdin bound and stdout and stderr captured
// separately — the status verb's contract is about which stream carries what.
func runSplit(t *testing.T, stdin string, args ...string) (stdout, stderr string, err error) {
	t.Helper()
	cmd := NewRootCommand()
	var so, se bytes.Buffer
	cmd.SetOut(&so)
	cmd.SetErr(&se)
	cmd.SetIn(strings.NewReader(stdin))
	cmd.SetArgs(args)
	err = cmd.Execute()
	return so.String(), se.String(), err
}

func storedMode(t *testing.T, root string) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(mode.FileRelPath)))
	if err != nil {
		return ""
	}
	return string(b)
}

// TestModePrintsManagedWhenNothingIsStored: the print form on an absent store
// is the quiet state, and --json carries the state alone — no notice field on
// a print, ever.
func TestModePrintsManagedWhenNothingIsStored(t *testing.T) {
	managedCheckout(t)
	if out := string(runCLI(t, "mode")); out != "managed\n" {
		t.Fatalf("mode = %q, want %q", out, "managed\n")
	}
	var got map[string]json.RawMessage
	if err := json.Unmarshal(runCLI(t, "mode", "--json"), &got); err != nil {
		t.Fatal(err)
	}
	if string(got["state"]) != `"managed"` {
		t.Errorf("state = %s", got["state"])
	}
	if _, has := got["notice"]; has {
		t.Errorf("the print form carried a notice: %s", got["notice"])
	}
}

// TestModeSetsTheStateBothWritersRead is ac-2 through ac-4 at the verb: a set
// lands in the store, the next print returns it, and the render reads the
// same file.
func TestModeSetsTheStateBothWritersRead(t *testing.T) {
	root := managedCheckout(t)
	writeUserSettings(t, `{"schema_version":1}`) // a surface exists: no notice.
	for _, st := range []string{"product-thinker", "facilitator", "managed"} {
		out := string(runCLI(t, "mode", st))
		if out != "" {
			t.Errorf("mode %s printed %q with a status surface installed; the set form is silent there", st, out)
		}
		if got := storedMode(t, root); got != st+"\n" {
			t.Errorf("store = %q after `mode %s`", got, st)
		}
		if got := string(runCLI(t, "mode")); got != st+"\n" {
			t.Errorf("mode prints %q after `mode %s`", got, st)
		}
		res, err := statusline.Compose(root, statusline.Payload{}, statusline.Defaults())
		if err != nil {
			t.Fatal(err)
		}
		if string(res.State) != st {
			t.Errorf("the render reads %q after `mode %s`", res.State, st)
		}
	}
}

// TestModeRefusesAnUnknownState: the vocabulary is closed, the refusal names
// the three, the exit is the operand-refusal code, and nothing is written.
func TestModeRefusesAnUnknownState(t *testing.T) {
	root := managedCheckout(t)
	out, err := runCLIErr(t, "mode", "boss")
	if err == nil {
		t.Fatalf("`mode boss` succeeded:\n%s", out)
	}
	if code := exitCodeOf(err); code != 2 {
		t.Errorf("exit = %d, want 2", code)
	}
	for _, want := range []string{"managed", "facilitator", "product-thinker"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("the refusal does not name %q: %v", want, err)
		}
	}
	if storedMode(t, root) != "" {
		t.Error("a refused set wrote the store")
	}
}

// TestModeSetPrintsOneLineWhereNoSurfaceExists is ac-7: with no status surface
// on this machine — no user-level setting, or one with the off switch thrown —
// the set form prints exactly one line naming whose answer is owed. Setting
// `managed` owes nobody and prints nothing. With a surface installed and on,
// nothing is printed: the line is the fallback, not a second channel.
func TestModeSetPrintsOneLineWhereNoSurfaceExists(t *testing.T) {
	managedCheckout(t)

	cases := []struct {
		name     string
		settings string // "" means no file
		state    string
		want     string
	}{
		{"no setting, product thinker", "", "product-thinker", "abcd: waiting on the product thinker — an answer is owed\n"},
		{"no setting, facilitator", "", "facilitator", "abcd: waiting on the facilitator — an answer is owed\n"},
		{"no setting, managed", "", "managed", ""},
		{"disabled setting, product thinker", `{"schema_version":1,"disabled":true}`, "product-thinker", "abcd: waiting on the product thinker — an answer is owed\n"},
		{"enabled setting, product thinker", `{"schema_version":1}`, "product-thinker", ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if err := os.RemoveAll(filepath.Join(os.Getenv("HOME"), ".abcd")); err != nil {
				t.Fatal(err)
			}
			if tc.settings != "" {
				writeUserSettings(t, tc.settings)
			}
			stdout, stderr, err := runSplit(t, "", "mode", tc.state)
			if err != nil {
				t.Fatalf("mode %s: %v\n%s", tc.state, err, stderr)
			}
			if stdout != tc.want {
				t.Errorf("stdout = %q, want %q", stdout, tc.want)
			}
			if strings.Count(stdout, "\n") > 1 {
				t.Errorf("more than one line: %q", stdout)
			}

			so, _, err := runSplit(t, "", "mode", tc.state, "--json")
			if err != nil {
				t.Fatal(err)
			}
			var got struct {
				State  string `json:"state"`
				Notice string `json:"notice"`
			}
			if err := json.Unmarshal([]byte(so), &got); err != nil {
				t.Fatalf("--json: %v\n%s", err, so)
			}
			if got.State != tc.state {
				t.Errorf("--json state = %q", got.State)
			}
			if got.Notice != strings.TrimSuffix(tc.want, "\n") {
				t.Errorf("--json notice = %q, want %q", got.Notice, strings.TrimSuffix(tc.want, "\n"))
			}
		})
	}
}

// TestModeRefusesOutsideAManagedRepository: a git checkout without the
// local-ephemeral tier is not one abcd manages, so the store refuses and the
// tier is never created; outside a checkout the root refusal fires.
func TestModeRefusesOutsideAManagedRepository(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	repo := t.TempDir()
	gitInitAt(t, repo)
	t.Chdir(repo)
	out, err := runCLIErr(t, "mode", "facilitator")
	if err == nil {
		t.Fatalf("set succeeded with no local tier:\n%s", out)
	}
	if !strings.Contains(err.Error(), mode.TierRelPath) {
		t.Errorf("the refusal does not name the tier: %v", err)
	}
	if code := exitCodeOf(err); code != 2 {
		t.Errorf("exit = %d, want 2", code)
	}
	if _, statErr := os.Stat(filepath.Join(repo, ".abcd")); statErr == nil {
		t.Error("the refusal minted an .abcd namespace")
	}
	// The print form still answers there: absent means managed.
	if got := string(runCLI(t, "mode")); got != "managed\n" {
		t.Errorf("print outside a managed repo = %q", got)
	}

	plain := t.TempDir()
	t.Chdir(plain)
	_, err = runCLIErr(t, "mode")
	if code := exitCodeOf(err); err == nil || code != 2 {
		t.Errorf("outside a repository: err=%v code=%d, want the root refusal at exit 2", err, code)
	}
}
