package cli

import (
	"encoding/json"
	"strings"
	"testing"
)

// TestThePromptThatNamesTheSessionCarriesTheStatus: the one prompt that sets
// the session's title is also a prompt the status block may be owed on, and
// one emit carries both, on every path the router has: the plain titling
// object, the --json envelope, and a rules.json that will not load, in either
// form. Neither half may displace the other (spc-2610150728396006 step 3,
// itd-2610111707085098).
func TestThePromptThatNamesTheSessionCarriesTheStatus(t *testing.T) {
	const line = "- @octo (claimed): merging #1012, for 30 min\n"
	for _, c := range []struct {
		name          string
		asJSON, wreck bool
	}{
		{"plain", false, false},
		{"json", true, false},
		{"rules-error-plain", false, true},
		{"rules-error-json", true, true},
	} {
		t.Run(c.name, func(t *testing.T) {
			f := newNameFixture(t)
			w := newReserveWorld(t)
			octo := w.clone(t, "octo", octoMail)
			pat := w.clone(t, "pat", "5+pat@users.noreply.github.com")
			if code, _, stderr := reserveRun(t, octo, "announce", "merging", "1012"); code != 0 {
				t.Fatalf("announce: exit %d, %s", code, stderr)
			}
			if code, _, stderr := reserveRun(t, pat, "status"); code != 0 {
				t.Fatalf("status: exit %d, %s", code, stderr)
			}
			if c.wreck {
				writeRepoRules(t, pat, `{"schema_version": 1, "domains": `)
			}
			f.launch(1000)
			runSessionStart(startJSON(t, "s1", pat, map[string]string{"source": "startup"}), "hook", "session-start")
			args := []string{"hook", "prompt-router"}
			if c.asJSON {
				args = append(args, "--json")
			}
			stdout, stderr := runHook(t, promptJSON(t, "s1", pat, "hello there", ""), args...)
			if c.wreck != strings.Contains(stderr, "injecting nothing") {
				t.Fatalf("rules.json wrecked %v, but stderr:\n%s", c.wreck, stderr)
			}
			if !c.asJSON {
				got := decodeTitled(t, stdout)
				if got.HookSpecificOutput.SessionTitle != "Alice" || !strings.Contains(got.HookSpecificOutput.AdditionalContext, line) {
					t.Fatalf("hook output = %+v; want the title Alice and the status line %q in its context", got, line)
				}
				return
			}
			var env struct {
				Text         string `json:"text"`
				Error        string `json:"error"`
				SessionTitle string `json:"session_title"`
			}
			if err := json.Unmarshal([]byte(stdout), &env); err != nil {
				t.Fatalf("not one envelope: %v\n%s", err, stdout)
			}
			if env.SessionTitle != "Alice" || !strings.Contains(env.Text, line) || c.wreck != (env.Error != "") {
				t.Fatalf("envelope = %+v; want session_title Alice, the status line %q in text, and an error only when rules.json is wrecked", env, line)
			}
		})
	}
}
