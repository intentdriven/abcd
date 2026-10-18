package autonomy

import (
	"strings"
	"testing"
)

// TestAwayReasonListsPersonOnlyActs (spc-2610180914469489, step 6): the
// reroute's reason names every act only the person may approve, each a
// phrase the spec's list carries, says that the away mode is no standing
// instruction, sends those acts to `abcd mode queue` and every other question
// to `abcd decide`, and says nothing detects the act mechanically.
func TestAwayReasonListsPersonOnlyActs(t *testing.T) {
	route := QuestionRoute()
	for _, want := range []struct{ what, phrase string }{
		{"the embargo --approve digest", "--approve <digest>"},
		{"the digest is never supplied on the agent's own", "never supply it on your own"},
		{"the embargo triage verdict", "embargo triage verdict"},
		{"the drain's hand-back", "drain's hand-back"},
		{"the remote-configuration apply verb", "apply verb for remote configuration"},
		{"a release sign-off", "a release sign-off"},
		{"what a release sign-off means", "asks for the person by name, or that no standing instruction of theirs allows"},
		{"a standing permission for automatic cuts stands", "standing permission for automatic cuts stands"},
		{"pushing, opening or merging a pull request", "pushing, opening or merging a pull request without a standing instruction"},
		{"the away mode is no standing instruction", "the away mode is never that instruction"},
		{"publishing and visibility", "publishing anything, changing a repository's visibility"},
		{"an outward comment or message", "posting a comment or message outward"},
		{"a new dependency", "adding a dependency"},
		{"secrets and Apps", "a secret, a credential or an App"},
		{"a website step", "only a website or a browser can do"},
		{"a destructive act", "a destructive act"},
		{"a force push", "a force push"},
		{"a peer's work", "discarding a peer's work"},
		{"spending", "spends money"},
		{"the catch-all", "any other act a command page or a brief invariant names as the person's"},
		{"the queue route", "abcd mode queue \"<step>\""},
		{"carrying on meanwhile", "carry on with other work"},
		{"the decide route", "abcd decide"},
		{"grounds and how to reverse it", "its grounds and how to reverse it"},
		{"nothing detects the act", "Nothing detects the act"},
		{"the mode grants nothing", "grants no rights"},
	} {
		if !strings.Contains(route, want.phrase) {
			t.Errorf("the reason does not name %s (%q):\n%s", want.what, want.phrase, route)
		}
	}
	acts := PersonOnlyActs()
	if len(acts) == 0 {
		t.Fatal("PersonOnlyActs is empty")
	}
	for _, a := range acts {
		if !strings.Contains(route, a) {
			t.Errorf("the reason leaves out the act %q", a)
		}
		if strings.Contains(a, "\n") {
			t.Errorf("an act runs over one line: %q", a)
		}
	}
	if last := acts[len(acts)-1]; !strings.Contains(last, "any other act") {
		t.Errorf("the list does not end on the catch-all: %q", last)
	}
	// The queue route comes before the decide route, so the agent reads
	// which acts are the person's before it reads that it decides the rest.
	if q, d := strings.Index(route, "abcd mode queue"), strings.Index(route, "abcd decide"); q < 0 || d < 0 || q > d {
		t.Errorf("the queue route (at %d) does not come before the decide route (at %d)", q, d)
	}
}
