package cli

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/intentdriven/abcd/internal/testsecret"
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
	if got := acceptedInResult(t, out); len(got) != 1 || got[0] != "filter.lfs.clean (added)" {
		t.Fatalf("the step's --json result carries what it accepted: %q\n%s", got, out)
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

// acceptedInResult is the accepted_config_change a --json step result
// carries.
func acceptedInResult(t *testing.T, out string) []string {
	t.Helper()
	var got []string
	for _, d := range jsonStream(t, out) {
		if a, ok := d["accepted_config_change"].([]any); ok {
			for _, v := range a {
				got = append(got, v.(string))
			}
		}
	}
	return got
}

// TestImplementStepNeverPrintsAKeyNamesSecret: a URL rewrite and a credential
// context planted mid-run carry a token in their key names. Through the CLI
// the step is still refused naming both, hidden, and accepting names them
// the same way; the token reaches neither stdout (text or --json), stderr,
// nor any file in the local tier (the fingerprint and the run record).
func TestImplementStepNeverPrintsAKeyNamesSecret(t *testing.T) {
	token, pw := testsecret.Synthetic(71, 32), testsecret.Synthetic(72, 24)
	repo := buildRepo(t)
	repo.Write("AGENTS.md", "# AGENTS.md\n\n- Run make check.\n")
	repo.Commit("conventions")
	mustImplement(t, "build", "itd-10", "--json")
	mustStep(t, "implement", "step", "--json")

	repo.Git("config", "url.https://x-access-token:"+token+"@github.com/.insteadOf", "https://github.com/")
	repo.Git("config", "credential.https://user:"+pw+"@example.invalid.helper", "store")
	var seen []string
	_, text, textErr := implementCLI(t, "implement", "step")
	ref := refusalDocs(t, 2, "implement", "step", "--json")
	reason, _ := ref["reason"].(string)
	if ref["stage"] != "config" || !strings.Contains(reason, "url.<hidden-") || !strings.Contains(reason, "credential.<hidden-") {
		t.Fatalf("the step is refused naming both keys, hidden: %v", ref)
	}
	seen = append(seen, text, textErr, reason)

	code, out, errOut := implementCLI(t, "implement", "step", "--accept-config-change", "--json")
	got := acceptedInResult(t, out)
	if code != 0 || len(got) != 2 || !strings.Contains(errOut, "url.<hidden-") {
		t.Fatalf("the accept names both keys, hidden (exit %d): %q\nstdout: %s\nstderr: %s", code, got, out, errOut)
	}
	seen = append(seen, out, errOut)
	err := filepath.WalkDir(filepath.Join(repo.Root(), ".abcd", ".work.local"), func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		data, err := os.ReadFile(path)
		seen = append(seen, path+"\n"+string(data))
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, text := range seen {
		if strings.Contains(text, token) || strings.Contains(text, pw) || strings.Contains(text, "x-access-token") {
			t.Fatalf("the key's secret was printed or stored:\n%s", text)
		}
	}
}
