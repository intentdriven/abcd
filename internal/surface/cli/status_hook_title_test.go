package cli

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/intentdriven/abcd/internal/core/claim"
)

// TestThePromptThatNamesTheSessionCarriesTheStatus: the one prompt that sets
// the session's title is also a prompt the status block may be owed on, and
// one emit carries both, on every path the router has: the plain titling
// object, the --json envelope, and a rules.json that will not load, in either
// form. Neither half may displace the other (spc-2610150728396006 step 3,
// itd-2610111707085098). Its budgets are its own (ownHookBudgets), so a
// loaded runner cannot leave the status unavailable.
func TestThePromptThatNamesTheSessionCarriesTheStatus(t *testing.T) {
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
			ownHookBudgets(t, 10*time.Second)
			f := newNameFixture(t)
			w := newReserveWorld(t)
			octo := w.clone(t, "octo", octoMail)
			pat := w.clone(t, "pat", "5+pat@users.noreply.github.com")
			t.Cleanup(claim.WaitPromptReadsForTest)
			if code, _, stderr := reserveRun(t, octo, "announce", "merging", "1012"); code != 0 {
				t.Fatalf("announce: exit %d, %s", code, stderr)
			}
			line := "- " + heldBy(t, octo) + " (claimed): merging #1012, for 30 min\n"
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
			// The prompt forces one rules domain, so the order of the rules
			// and the status block is seen.
			stdout, stderr := runHook(t, promptJSON(t, "s1", pat, "*COMMITTING hello there", ""), args...)
			if c.wreck != strings.Contains(stderr, "injecting nothing") {
				t.Fatalf("rules.json wrecked %v, but stderr:\n%s", c.wreck, stderr)
			}
			if !c.asJSON {
				got := decodeTitled(t, stdout)
				if got.HookSpecificOutput.SessionTitle != "Alice" || !strings.Contains(got.HookSpecificOutput.AdditionalContext, line) {
					t.Fatalf("hook output = %+v; want the title Alice and the status line %q in its context", got, line)
				}
				statusLast(t, got.HookSpecificOutput.AdditionalContext, c.wreck)
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
			statusLast(t, env.Text, c.wreck)
		})
	}
}

// statusLast holds the context a titled prompt carries to its order: the
// rules' text first, when the rules loaded, then the status block, which
// ends it; with the rules unreadable the status block is the whole context.
func statusLast(t *testing.T, context string, wrecked bool) {
	t.Helper()
	at := strings.Index(context, claim.NotesHeading+"\n")
	if at < 0 || !strings.HasSuffix(context, claim.GrantsNothing+"\n") {
		t.Fatalf("context:\n%s\nwant it to end with the status block", context)
	}
	rules := strings.Index(context, "## COMMITTING")
	switch {
	case wrecked && at != 0:
		t.Fatalf("context:\n%s\nwant the status block alone, rules.json being unreadable", context)
	case !wrecked && (rules < 0 || rules > at):
		t.Fatalf("context:\n%s\nwant the rules first, then the status block", context)
	}
}
