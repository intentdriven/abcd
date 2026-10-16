package cli

import (
	"strings"
	"testing"
	"time"

	"github.com/intentdriven/abcd/internal/core/claim"
)

// TestThePromptHookShowsEachAnnouncementOnce is C6 and C12 at the hook: after
// a fetch brought an announcement in, the next prompt's injected text carries
// it, rendered from its fields, with the grants-nothing line, inside a second
// with an origin that would hang any network call; the prompt after shows it
// no more; and a rules.json that will not load does not switch it off.
func TestThePromptHookShowsEachAnnouncementOnce(t *testing.T) {
	t.Setenv("ABCD_RULES_STATE_DIR", t.TempDir())
	w := newReserveWorld(t)
	octo := w.clone(t, "octo", octoMail)
	pat := w.clone(t, "pat", "5+pat@users.noreply.github.com")
	if code, _, stderr := reserveRun(t, octo, "announce", "merging", "1012"); code != 0 {
		t.Fatalf("announce: exit %d, %s", code, stderr)
	}
	if code, _, stderr := reserveRun(t, pat, "status"); code != 0 {
		t.Fatalf("status: exit %d, %s", code, stderr)
	}
	gitCmd(t, pat, "config", "protocol.ext.allow", "always")
	gitCmd(t, pat, "remote", "set-url", "origin", "ext::sh -c sleep\\ 30")

	start := time.Now()
	out, _ := runHook(t, hookInputJSON(t, "s-status", pat, "hello there"), "hook", "prompt-router")
	// The origin would hang for 30 s; the margin is for a loaded machine.
	if took := time.Since(start); took > 5*time.Second {
		t.Fatalf("the hook took %s; want it never to wait on the network", took)
	}
	for _, want := range []string{claim.NotesHeading + "\n", "- @octo (claimed): merging #1012, for 30 min\n", claim.GrantsNothing + "\n"} {
		if !strings.Contains(out, want) {
			t.Fatalf("the hook's text:\n%s\nwant it to carry %q", out, want)
		}
	}
	again, _ := runHook(t, hookInputJSON(t, "s-status", pat, "hello again"), "hook", "prompt-router")
	if strings.Contains(again, claim.NotesHeading) {
		t.Fatalf("the second prompt shows the status again:\n%s", again)
	}

	// A second announcement, then a rules.json that will not load: the
	// status block is still emitted, in both forms.
	gitCmd(t, pat, "remote", "set-url", "origin", w.origin)
	if code, _, stderr := reserveRun(t, octo, "announce", "merged", "1012"); code != 0 {
		t.Fatalf("announce: exit %d, %s", code, stderr)
	}
	if code, _, stderr := reserveRun(t, pat, "status"); code != 0 {
		t.Fatalf("status: exit %d, %s", code, stderr)
	}
	writeRepoRules(t, pat, `{"schema_version": 1, "domains": `)
	env := routerJSON(t, "s-status", pat, "hello")
	if env.Error == "" || !strings.Contains(env.Text, "- @octo (claimed): merged #1012, for 30 min\n") {
		t.Fatalf("with rules.json unreadable, the envelope = %+v; want the error and the status block", env)
	}
	if code, _, stderr := reserveRun(t, octo, "announce", "asking", "1012"); code != 0 {
		t.Fatalf("announce: exit %d, %s", code, stderr)
	}
	if code, _, stderr := reserveRun(t, pat, "status"); code != 0 {
		t.Fatalf("status: exit %d, %s", code, stderr)
	}
	plain, _ := runHook(t, hookInputJSON(t, "s-status", pat, "hello"), "hook", "prompt-router")
	if !strings.Contains(plain, "- @octo (claimed): asking #1012, for 30 min\n") {
		t.Fatalf("with rules.json unreadable, the plain hook:\n%s\nwant the status block", plain)
	}
}
