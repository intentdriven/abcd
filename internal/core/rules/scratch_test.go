package rules

import (
	"slices"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/abcdhome"
)

// scratchDomain is the bundled domain that says where a session's working
// files live (iss-2610161501104088).
const scratchDomain = "SCRATCH"

// TestScratchDomainRecallsTheHostScratchpad pins iss-2610161501104088: the
// host hands each session a scratchpad under /tmp and asks it to keep its
// temporary files there, a reboot wipes it, and before this domain no bundled
// rule said so. A prompt about the scratchpad, or a helper script under /tmp,
// must recall the domain, and the injected block must carry the rule.
func TestScratchDomainRecallsTheHostScratchpad(t *testing.T) {
	rs := Defaults()
	for _, prompt := range []string{
		"put the review notes in the scratchpad",
		"write a helper script under /tmp to watch the PR",
		"keep this in /private/tmp for now",
		"save a temporary file for the next turn",
		"start a watcher on the merge queue",
		"write the handover before the session ends",
		"leave the scratch copy somewhere",
	} {
		if got := rs.Match(prompt); !has(got, scratchDomain) {
			t.Errorf("%q does not recall %s: %v", prompt, scratchDomain, names(got))
		}
	}
	res := Inject(rs, "write a helper script under /tmp", SessionState{}, 0)
	if !slices.Contains(res.Injected, scratchDomain) {
		t.Fatalf("a helper script under /tmp injected %v, want %s in it", res.Injected, scratchDomain)
	}
	if !strings.Contains(res.Text, "## "+scratchDomain) || !strings.Contains(res.Text, "does not survive a reboot or a new session") {
		t.Fatalf("the injected block does not carry the scratchpad rule:\n%s", res.Text)
	}
	if got := rs.Match("paint a landscape"); has(got, scratchDomain) {
		t.Errorf("an unrelated prompt recalls %s", scratchDomain)
	}
}

// TestScratchDomainRecallKeywords holds the domain to the recall set the issue
// names, each word alone recalling it.
func TestScratchDomainRecallKeywords(t *testing.T) {
	rs := Defaults()
	d, ok := rs.Domains[scratchDomain]
	if !ok {
		t.Fatalf("no bundled %s domain", scratchDomain)
	}
	for _, kw := range []string{"scratch", "scratchpad", "tmp", "temporary", "script", "helper", "watcher", "queue", "handover"} {
		if !slices.Contains(d.Recall, kw) {
			t.Errorf("%s does not recall on %q", scratchDomain, kw)
		}
		if !has(rs.Match("the "+kw), scratchDomain) {
			t.Errorf("a prompt naming %q does not recall %s", kw, scratchDomain)
		}
	}
}

// TestScratchRuleSaysWhereEachFileGoes holds the rule's substance: what a
// later turn rereads goes in the checkout's local tier, a verifier's copy goes
// under abcd's home folder, and the host scratchpad keeps only the turn's
// throwaway files because it is wiped. Where it meets the OPINIONS and LIFEBOAT
// rules it names them instead of restating them.
func TestScratchRuleSaysWhereEachFileGoes(t *testing.T) {
	body := strings.Join(Defaults().Domains[scratchDomain].Rules, "\n")
	for _, want := range []string{
		"later turn, session or agent",
		"helper script", "queue ticket", "log", "pull-request text", "handover",
		".abcd/.work.local/scratch/", "logs/",
		"verifier's copy", abcdhome.Display() + "/",
		"host's scratchpad", "throwaway", "turn at hand",
		"does not survive a reboot or a new session",
		"OPINIONS", "LIFEBOAT",
	} {
		if !strings.Contains(body, want) {
			t.Errorf("the %s rule does not say %q:\n%s", scratchDomain, want, body)
		}
	}
}
