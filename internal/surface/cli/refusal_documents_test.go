package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/oracle"
)

// The tests in this file hold the refusal document a verb writes under --json
// ahead of the error object, which the error boundary's pass does not read, to
// the boundary's rule: a value the caller typed is named by its flag, its
// position and its length, never by its text.

// TestADrainSkipTargetItRefusesIsNotRepeated: a --skip or --unskip target that
// is neither an issue id nor a clean repository-relative path is refused by
// its flag and position, in the plain and the --json form.
func TestADrainSkipTargetItRefusesIsNotRepeated(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	dir := scratchRepoWithCommit(t)
	if err := os.MkdirAll(filepath.Join(dir, ".abcd", ".work.local"), 0o755); err != nil {
		t.Fatal(err)
	}
	tok := tokenMarker()
	cases := []struct {
		args []string
		want string
	}{
		{[]string{"drain", "--skip", "/srv/" + tok, "--reason", "held back while it is checked"}, "--skip number 1"},
		{[]string{"drain", "--skip", "iss-2608300000000004", "--skip", "../" + tok, "--reason", "held back while it is checked"}, "--skip number 2"},
		{[]string{"drain", "--unskip", "/srv/" + tok}, "--unskip number 1"},
	}
	for _, c := range cases {
		for _, form := range [][]string{nil, {"--json"}} {
			args := append(append([]string(nil), form...), c.args...)
			code, out, errOut := runRefusal(t, "", args...)
			if code != 2 {
				t.Errorf("%v: exit %d, want 2\nstdout: %s\nstderr: %s", args, code, out, errOut)
			}
			if repeats(out, tok[:12]) || repeats(errOut, tok[:12]) {
				t.Errorf("%v: a refusal repeated the target\nstdout: %s\nstderr: %s", args, out, errOut)
			}
			if !strings.Contains(out+errOut, c.want) {
				t.Errorf("%v: the refusal does not name %q\nstdout: %s\nstderr: %s", args, c.want, out, errOut)
			}
		}
	}
}

// TestAStepsRouteItRefusesIsNotRepeated: every way a step's --route can be
// refused names the route by its position and any part of it the machine does
// not know by its length, in the plain and the --json form, and still names
// what to correct.
func TestAStepsRouteItRefusesIsNotRepeated(t *testing.T) {
	runToImplement(t)
	writeMachineConfig(t, routeStepMachine)
	tok := tokenMarker()
	cases := []struct {
		name  string
		route string
		want  string
	}{
		{"no equals sign", tok, "runner:<name>"},
		{"unknown role", strings.ToLower(tok) + "=runner:opencode", "dispatches"},
		{"tier route", strings.ToLower(tok) + "=frontier", "runner:<name>"},
		{"no runner rung", "implementer=" + tok, "runner:<name>"},
		{"runner not shipped", "implementer=runner:" + tok, "does not ship"},
		{"model not provider/model", "implementer=runner:opencode@" + tok, "<provider>/<model>"},
		{"provider not configured", "implementer=runner:opencode@" + tok + "/devstral", "not configured"},
		{"model off the allowlist", "implementer=runner:opencode@local/" + tok, "local's list"},
		{"over the length bound", "implementer=runner:opencode@local/" + tok + strings.Repeat("a", oracle.MaxRouteBytes), "bytes"},
	}
	for _, c := range cases {
		for _, form := range [][]string{nil, {"--json"}} {
			args := append(append([]string(nil), form...), "implement", "step", "--route", c.route)
			code, out, errOut := runRefusal(t, "", args...)
			label := c.name + " " + strings.Join(form, "")
			if code != 2 {
				t.Errorf("%s: exit %d, want 2\nstdout: %s\nstderr: %s", label, code, out, errOut)
			}
			if repeats(out, tok[:12]) || repeats(errOut, tok[:12]) {
				t.Errorf("%s: a refusal repeated the route\nstdout: %s\nstderr: %s", label, out, errOut)
			}
			// The JSON encoder escapes < and > in the refusal document.
			text := strings.NewReplacer(`\u003c`, "<", `\u003e`, ">").Replace(out + errOut)
			if !strings.Contains(text, "--route number 1") || !strings.Contains(text, c.want) {
				t.Errorf("%s: the refusal does not name the route by position and %q\nstdout: %s\nstderr: %s", label, c.want, out, errOut)
			}
		}
	}
}
