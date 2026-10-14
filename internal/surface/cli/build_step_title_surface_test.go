package cli

import (
	"strings"
	"testing"
)

// TestBuildMasksASpecStepTitlesHiddenRunes (iss-2610121542268630): a spec
// whose step titles carry U+202E or U+009B starts its run, exit 0, and the
// titles reach `build --json` (the lane and the pending steps) and
// `implement status --json` masked, with nothing raw on either stream.
func TestBuildMasksASpecStepTitlesHiddenRunes(t *testing.T) {
	repo := buildRepo(t)
	repo.Write(buildSpec, "---\nid: spc-1\nslug: alpha\nintent: itd-10\n---\n# alpha\n\n## Summary\n\nA written design record.\n\n"+
		"## Steps\n\n1. The state‮ file\n2. The\u009b verb\n")
	repo.Commit("step titles carrying hidden runes")
	for _, args := range [][]string{{"build", "itd-10", "--json"}, {"implement", "status", "--json"}} {
		code, out, errOut := implementCLI(t, args...)
		if strings.ContainsAny(out+errOut, "‮\u009b") {
			t.Fatalf("%v: a raw rune reaches the stream: %q", args, out)
		}
		if code != 0 || !strings.Contains(out, "The state? file") {
			t.Fatalf("%v: the run starts, its titles masked, exit 0: %d %s %s", args, code, out, errOut)
		}
	}
}
