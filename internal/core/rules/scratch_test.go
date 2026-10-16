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

// scratchRecall is the domain's recall set. helper and queue are not in it: in
// a sample of real prompts they matched Go helper functions and the merge queue
// far more often than a session's working files.
var scratchRecall = []string{"scratch", "scratchpad", "tmp", "temporary", "script", "watcher", "handover"}

// TestScratchDomainRecallsTheHostScratchpad pins iss-2610161501104088: the
// host hands each session a scratchpad and asks it to keep its temporary files
// there, a reboot wipes it, and before this domain no bundled rule said so. A
// prompt about the scratchpad, or a helper script under /tmp, must recall the
// domain, and the injected block must carry the rule.
func TestScratchDomainRecallsTheHostScratchpad(t *testing.T) {
	rs := Defaults()
	for _, prompt := range []string{
		"put the review notes in the scratchpad",
		"write a helper script under /tmp to watch the PR",
		"keep this in /private/tmp for now",
		"save a temporary file for the next turn",
		"start a watcher on the merge queue", // recalled by watcher, not queue
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
}

// TestScratchDomainIgnoresNearMisses holds recall to a session's working
// files: the merge queue, a Go helper function, the TMPDIR variable and words
// that only contain a keyword (javascript, description) must not inject the
// domain. Matching is on word boundaries, so "script" inside "javascript" or
// "description" and "tmp" inside "tmpdir" do not match.
func TestScratchDomainIgnoresNearMisses(t *testing.T) {
	rs := Defaults()
	for _, prompt := range []string{
		"paint a landscape",
		"merge queue",
		"put the PR on the merge queue",
		"a test helper function",
		"extract a helper for the parser",
		"read $TMPDIR",
		"port the javascript client",
		"update the description",
	} {
		if got := rs.Match(prompt); has(got, scratchDomain) {
			t.Errorf("%q recalls %s, which is about a session's working files: %v", prompt, scratchDomain, names(got))
		}
	}
}

// TestScratchDomainRecallKeywords holds the domain to its recall set exactly,
// each word alone recalling it.
func TestScratchDomainRecallKeywords(t *testing.T) {
	rs := Defaults()
	d, ok := rs.Domains[scratchDomain]
	if !ok {
		t.Fatalf("no bundled %s domain", scratchDomain)
	}
	if got := slices.Sorted(slices.Values(d.Recall)); !slices.Equal(got, slices.Sorted(slices.Values(scratchRecall))) {
		t.Errorf("%s recalls on %v, want %v", scratchDomain, d.Recall, scratchRecall)
	}
	for _, kw := range scratchRecall {
		if !has(rs.Match("the "+kw), scratchDomain) {
			t.Errorf("a prompt naming %q does not recall %s", kw, scratchDomain)
		}
	}
}

// TestScratchRuleSaysWhereEachFileGoes holds the rule's substance. What a later
// turn reruns or reads goes in the main checkout's local tier or under abcd's
// home folder, never in a linked worktree's local tier, which git worktree
// remove deletes; a verifier's copy goes in a directory of its own under abcd's
// home, keyed on the root commit; the host scratchpad keeps only the turn's
// throwaway files because it is wiped. The rule stays host-agnostic (no /tmp),
// and where it meets the LIFEBOAT and OPINIONS rules it names them instead of
// restating them.
func TestScratchRuleSaysWhereEachFileGoes(t *testing.T) {
	body := strings.Join(Defaults().Domains[scratchDomain].Rules, "\n")
	home := abcdhome.Display() + "/"
	for _, want := range []string{
		"later turn, session or agent",
		"a helper script, a queue ticket, a log, pull-request text, a handover",
		"the main checkout's .abcd/.work.local/scratch/ (logs/ for run output) or a directory under " + home,
		"never in a linked worktree's .abcd/.work.local/",
		"git worktree remove",
		"a file for that worktree's own work alone may stay in its own tier",
		"verifier's copy goes outside every working tree, in a directory of its own under " + home + " keyed on the root-commit SHA",
		"the session's scratchpad only for a copy the same turn makes and discards",
		"host's scratchpad", "throwaway", "turn at hand",
		"does not survive a reboot or a new session",
		"LIFEBOAT's runtime-artefacts rule says what stays out of tracked directories; OPINIONS' user's-directory rule says where a tool may create one.",
	} {
		if !strings.Contains(body, want) {
			t.Errorf("the %s rule does not say %q:\n%s", scratchDomain, want, body)
		}
	}
	if strings.Contains(body, "/tmp") {
		t.Errorf("the %s rule names /tmp; the bundled rules are host-agnostic:\n%s", scratchDomain, body)
	}
}

// TestScratchRuleSendsAHelperToAbcdsCommandsFirst pins iss-2610161614039496:
// an agent wrote its own preflight queue script although abcd's waiter already
// did that, kept it in the host scratchpad, and a reboot wiped it with the
// queued runs. The SCRATCH domain carries a second rule: look for an abcd
// command first, write a helper only when none does it, keep that helper where
// the first rule says, and capture the gap. It names the preflight case and
// points at the LOAD waiter rule rather than restating it, and a prompt about
// scripting a preflight queue injects it.
func TestScratchRuleSendsAHelperToAbcdsCommandsFirst(t *testing.T) {
	rules := Defaults().Domains[scratchDomain].Rules
	if len(rules) != 2 {
		t.Fatalf("%s carries %d rules, want 2 (where files go; abcd's commands before a helper)", scratchDomain, len(rules))
	}
	helper := rules[1]
	for _, want := range []string{
		"Before writing a helper script, check whether an abcd command already does the job",
		"`abcd --help`", "command page",
		"`abcd implement load --wait -- make preflight`, never a hand-written waiter",
		"LOAD's waiter rule",
		"only when no command does",
		"the main checkout's .abcd/.work.local/scratch/",
		"capture the gap", "so abcd can add the command",
	} {
		if !strings.Contains(helper, want) {
			t.Errorf("the %s helper rule does not say %q:\n%s", scratchDomain, want, helper)
		}
	}
	if strings.Contains(helper, "pgrep") {
		t.Errorf("the %s helper rule restates the LOAD waiter rule instead of pointing at it:\n%s", scratchDomain, helper)
	}

	res := Inject(Defaults(), "write a script to queue the preflight", SessionState{}, 0)
	if !slices.Contains(res.Injected, scratchDomain) {
		t.Fatalf("a script to queue the preflight injected %v, want %s in it", res.Injected, scratchDomain)
	}
	if !strings.Contains(res.Text, "abcd implement load --wait -- make preflight") {
		t.Fatalf("the injected block does not carry the helper rule:\n%s", res.Text)
	}
}
