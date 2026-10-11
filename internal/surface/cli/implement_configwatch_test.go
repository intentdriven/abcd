package cli

import (
	"strings"
	"testing"
	"time"
)

// TestImplementStepTripwireIsWired: through the CLI, a content filter planted
// in the repository's shared config mid-run refuses the next `implement step`
// at the stage config (exit 2), naming the key and the accepting flag;
// `--accept-config-change` names what it accepted on stderr and takes the
// step; and `implement outage probe` is refused, the probe never run, once a
// remote's upload-pack program is planted.
func TestImplementStepTripwireIsWired(t *testing.T) {
	repo := buildRepo(t)
	repo.Write("AGENTS.md", "# AGENTS.md\n\n- Run make check.\n")
	repo.Commit("conventions")
	mustImplement(t, "build", "itd-10", "--json")
	mustStep(t, "implement", "step", "--json")

	repo.Git("config", "filter.lfs.clean", "git-lfs clean -- %f")
	ref := refusalDocs(t, 2, "implement", "step", "--json")
	if ref["stage"] != "config" || !strings.Contains(ref["reason"].(string), "filter.lfs.clean") ||
		!strings.Contains(ref["remedy"].(string), "--accept-config-change") {
		t.Fatalf("the step is refused at the tripwire: %v", ref)
	}

	code, out, errOut := implementCLI(t, "implement", "step", "--accept-config-change", "--json")
	if code != 0 || !strings.Contains(errOut, "accepted a change to the repository's git configuration: filter.lfs.clean (added)") {
		t.Fatalf("the accept names what it accepted and takes the step (exit %d)\nstdout: %s\nstderr: %s", code, out, errOut)
	}
	mustStep(t, "implement", "step", "--json")

	clock := time.Date(2026, 10, 9, 23, 0, 0, 0, time.UTC)
	calls := 0
	prevNow, prevProbe := outageNow, outageNetworkProbe
	outageNow = func() time.Time { return clock }
	outageNetworkProbe = func(string) func() (bool, string) {
		return func() (bool, string) { calls++; return true, "origin answered" }
	}
	t.Cleanup(func() { outageNow, outageNetworkProbe = prevNow, prevProbe })
	mustImplement(t, "implement", "join", "--session", "alpha", "--role", "first", "--json")
	mustImplement(t, "implement", "outage", "record", "--session", "alpha", "--service", "network", "--kind", "tool",
		"--lane", "lane-1", "--what", "git fetch", "--json")
	clock = clock.Add(time.Minute)
	repo.Git("config", "remote.origin.uploadpack", "touch probe-ran; git-upload-pack")

	code, out, errOut = implementCLI(t, "implement", "outage", "probe", "--session", "alpha")
	if code != 2 || calls != 0 || !strings.Contains(out+errOut, "remote.origin.uploadpack") {
		t.Fatalf("the probe is refused before it runs (exit %d, calls %d)\nstdout: %s\nstderr: %s", code, calls, out, errOut)
	}
}
