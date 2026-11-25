package cli

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/intentdriven/abcd/internal/abcdhome"
	"github.com/intentdriven/abcd/internal/core/interview"
)

// refusal_raisers_intent_test.go — the probes for the codes of the intent
// store's verbs (intent and spec), of the written interviews, of rules, of
// the refusals a peer's checkout answers, and of capture's own outside a
// checkout (TestRaisedByIsWhatTheVerbsRaise, spc-2610301431108175 step 3).

var (
	_ = registerRaiserProbes("intent", intentRaiserProbes)
	_ = registerRaiserProbes("interview", interviewRaiserProbes)
	_ = registerRaiserProbes("peer-held", peerHeldRaiserProbes)
)

// Ids no fixture writes.
const (
	unknownIntentID = "itd-2601010101010101"
	unknownSpecID   = "spc-2601010101010101"
)

// intentRaiserProbes builds an intent store holding a draft (itd-10), a
// planned intent with no impact (itd-2) and the open spec that delivers it
// (spc-1), and returns the probes of intent, spec, rules and capture's
// outside-a-checkout refusal.
func intentRaiserProbes(t *testing.T) []raiserProbe {
	repo := intentTestRepo(t)
	writeRepoFile(t, repo, cliDrafts+"/itd-10-alpha.md", cliDraftWithAC("itd-10", "alpha"))
	writeRepoFile(t, repo, cliPlanned+"/itd-2-beta.md",
		"---\nid: itd-2\nslug: beta\nspec_id: spc-1\nkind: standalone\n---\n# beta\n\n## Acceptance Criteria\n\n- ok\n")
	writeRepoFile(t, repo, cliSpecsOpen+"/spc-1-beta.md", "---\nid: spc-1\nslug: beta\nintent: itd-2\n---\n# beta\n")
	if err := os.MkdirAll(filepath.Join(repo, ".abcd", ".work.local"), 0o755); err != nil {
		t.Fatal(err)
	}
	missing := filepath.Join(t.TempDir(), "absent.json")
	outside := inDir(t.TempDir())
	return []raiserProbe{
		probe("intent/outside-checkout", "intent").from(outside),
		probe("intent/outside-checkout", "intent", "plan", "itd-10").from(outside),
		probe("spec/outside-checkout", "spec", "close", "spc-1").from(outside),
		probe("capture/outside-checkout", "capture", "the parser drops the last line", "--remedy", cliTestRemedy).from(outside),
		probe("capture/outside-checkout", "capture", "list", "--all").from(outside),
		probe("usage/unknown-command", "intent", "lnk", "itd-10").from(inDir(repo)),
		probe("usage/unknown-command", "intent", "zzz"),
		probe("usage/missing-argument", "intent", ""),
		probe("usage/missing-required-flag", "intent", "hold", "itd-10"),
		probe("usage/flag-conflict", "intent", "audit", "--owed", "--strict"),

		probe("intent/unknown-intent", "intent", "plan", unknownIntentID),
		probe("intent/unknown-intent", "intent", "ready", unknownIntentID),
		probe("intent/unknown-intent", "intent", "hold", unknownIntentID, "--reason", "the parser rewrite comes first"),
		probe("intent/unknown-intent", "intent", "unhold", unknownIntentID),
		probe("intent/unknown-intent", "intent", "target", unknownIntentID, "v1.0.0"),
		probe("intent/unknown-intent", "intent", "edge", unknownIntentID, "--builds-on", "itd-2"),
		probe("intent/unknown-intent", "intent", "condition", unknownIntentID),
		probe("intent/unknown-intent", "intent", "prepass", unknownIntentID),
		probe("intent/unknown-intent", "intent", "interview", unknownIntentID, "--answers", missing),

		probe("intent/insufficient-grounds", "intent", "ready", "itd-2", "--grounds", "bogus"),
		probe("intent/invalid-verdict", "intent", "audit", "ingest", "--verdict-json", missing),
		probe("intent/invalid-findings", "intent", "consistency", "ingest", "--findings-json", missing),
		probe("intent/invalid-findings", "intent", "prepass", "itd-10", "--findings-json", missing),
		probe("intent/invalid-request", "intent", "unhold", "itd-2"),
		probe("intent/invalid-request", "intent", "target", "itd-2", "not-a-version"),
		probe("intent/invalid-request", "intent", "link", "itd-2", unknownSpecID),

		probe("spec/missing-impact", "spec", "close", "spc-1"),
		probe("spec/unknown-spec", "spec", "close", unknownSpecID),
		probe("spec/invalid-request", "spec", "close", "spc-1", "--impact", "fix", "--remainder", "the-rest"),
		probe("usage/missing-required-flag", "spec", "close", "spc-1", "--production-mode", "hand-written"),

		probe("rules/unknown-domain", "rules", "zzz-no-such-domain"),
	}
}

// interviewRaiserProbes builds the retrospective's and the planning
// interview's repositories, under a machine that routes no role and then one
// that routes the reflection composer, and returns the written interviews'
// probes and those of the planning interview's own refusals.
func interviewRaiserProbes(t *testing.T) []raiserProbe {
	routeMachine(t, "")
	retro := retroRepo(t)
	planning := planningRepo(t)
	missing := filepath.Join(t.TempDir(), "absent.json")
	answers := planningAnswersFile(t)
	routed := func(t *testing.T) {
		routeMachine(t, interview.RoleReflectionComposer)
		t.Chdir(retro.Root())
	}
	broken := func(t *testing.T) {
		home := routeMachine(t, interview.RoleReflectionComposer)
		if err := os.WriteFile(abcdhome.Path(home, "config.json"), []byte(`{"roles":`), 0o600); err != nil {
			t.Fatal(err)
		}
		t.Chdir(retro.Root())
	}
	return []raiserProbe{
		probe("interview/unreachable-role", "reflect", "interview", "v0.2.0", "--proceed", "--answers", answers).from(inDir(retro.Root())),
		probe("interview/unreachable-role", "intent", "interview", "itd-5", "--answers", answers).from(inDir(planning.Root())),
		probe("intent/invalid-request", "intent", "interview", "itd-6", "--answers", answers).from(inDir(planning.Root())),
		probe("interview/missing-terminal", "reflect", "interview", "v0.2.0", "--proceed").from(routed),
		probe("interview/invalid-answers", "reflect", "interview", "v0.2.0", "--proceed", "--answers", missing).from(routed),
		probe("interview/invalid-config", "reflect", "interview", "v0.2.0", "--proceed", "--answers", answers).from(broken),
	}
}

// peerHeldRaiserProbes builds a checkout whose peer worktree holds iss-100
// and whose side branch ships itd-77, and returns the refusals that name the
// peer that holds a record this checkout lacks.
func peerHeldRaiserProbes(t *testing.T) []raiserProbe {
	home, repo := peerCheckout(t)
	addPeer(t, home, repo, "a")
	gitCmd(t, repo, "switch", "-q", "-c", "side")
	writeRel(t, repo, ".abcd/development/intents/shipped/itd-77-shipped-elsewhere.md", peerIntent("itd-77", "Shipped elsewhere"))
	gitCmd(t, repo, "add", "-A")
	gitCommit(t, repo, "commit", "-q", "-m", "ship itd-77 on a branch")
	gitCmd(t, repo, "switch", "-q", "main")
	return []raiserProbe{
		probe("capture/held-by-peer", "capture", "resolve", "iss-100", "fixed", "--impact", "fix"),
		probe("intent/held-by-peer", "intent", "audit", "itd-77"),
	}
}
