package cli

import (
	"encoding/json"
	"strings"
	"testing"
)

// TestReserveNamesAGeneratedFileItShares (iss-2610282340087862): two people's
// footprints that share only a file origin's default branch declares
// generated are both made, exit 0; the second's plain render names the file
// and the reservation it shares it with on stdout, and --json carries it as
// generated_shared. A real shared source file still backs off.
func TestReserveNamesAGeneratedFileItShares(t *testing.T) {
	w := newReserveWorld(t)
	seed := w.clone(t, "seeder", octoMail)
	writeRel(t, seed, ".gitattributes", "docs/commands.md linguist-generated\n")
	writeRel(t, seed, "docs/commands.md", "generated\n")
	gitCmd(t, seed, "add", "-A")
	gitCommit(t, seed, "commit", "-q", "-m", "declare")
	gitCmd(t, seed, "push", "-q", "origin", "main")
	p := w.clone(t, "p", octoMail)
	q := w.clone(t, "q", quinnMail)
	if code, stdout, stderr := reserveRun(t, p, "reserve", "iss-1", "--footprint", "docs/commands.md", "--footprint", "a.go"); code != 0 {
		t.Fatalf("P's reservation: exit %d\n%s%s", code, stdout, stderr)
	}
	code, stdout, stderr := reserveRun(t, q, "reserve", "iss-2", "--footprint", "docs/commands.md", "--footprint", "b.go")
	const line = `  shared: "docs/commands.md" is also in the footprint of iss-1, but origin's default branch declares it generated, so it is no overlap; whichever lands second regenerates it after merging` + "\n"
	if code != 0 || !strings.Contains(stdout, "reserved iss-2 for "+heldBy(t, q)) || !strings.Contains(stdout, line) || stderr != "" {
		t.Fatalf("the reserve sharing only the generated file: exit %d, stdout %q, stderr %q; want it made, naming the shared file", code, stdout, stderr)
	}
	code, raw, stderr := reserveRun(t, q, "reserve", "iss-3", "--footprint", "docs/commands.md", "--json")
	var got struct {
		GeneratedShared []struct {
			Path   string `json:"path"`
			Record string `json:"record"`
		} `json:"generated_shared"`
	}
	if err := json.Unmarshal([]byte(raw), &got); code != 0 || err != nil || len(got.GeneratedShared) != 1 ||
		got.GeneratedShared[0].Path != "docs/commands.md" || got.GeneratedShared[0].Record != "iss-1" || stderr != "" {
		t.Fatalf("the --json reserve: exit %d, %+v (%v); want generated_shared naming docs/commands.md in iss-1 alone, this copy's own iss-2 being no share\n%s%s", code, got.GeneratedShared, err, raw, stderr)
	}
	// A real shared source file still backs off.
	if code, stdout, stderr := reserveRun(t, q, "reserve", "iss-4", "--footprint", "a.go"); code != 3 || !strings.Contains(stdout+stderr, `"a.go" is in the footprint of iss-1`) {
		t.Fatalf("the reserve sharing a.go: exit %d\n%s%s; want the back-off, exit 3", code, stdout, stderr)
	}
}
