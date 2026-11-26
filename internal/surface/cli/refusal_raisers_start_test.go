package cli

import "testing"

// refusal_raisers_start_test.go — the probes for the codes of start
// (TestRaisedByIsWhatTheVerbsRaise): each runs in the world its refusal is
// raised in (startRefusalWorlds).

var _ = registerRaiserProbes("start", startRaiserProbes)

// startRaiserProbes returns one probe for each of start's refusals, readied
// by the world startRefusalWorlds sets up for it, each in a subtest of its own:
// a world replaces test seams that are undone only when its test ends.
func startRaiserProbes(t *testing.T) []raiserProbe {
	var out []raiserProbe
	for _, code := range []string{
		"start/profile-unusable",
		"start/repository-unready",
		"start/host-not-runnable",
		"start/not-logged-in",
		"start/no-verdict",
		"start/wall-not-held",
		"start/outside-a-terminal",
	} {
		setup := startRefusalWorlds[code]
		if setup == nil {
			t.Fatalf("no world readies %s", code)
		}
		out = append(out, probe(code, "start").apart(setup))
	}
	return out
}
