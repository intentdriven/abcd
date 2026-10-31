package cli

import (
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/abcdhome"
	"github.com/intentdriven/abcd/internal/core/claim"
)

// A ~/.abcd.noindex/holder-ids the guarded reader refuses is named on every
// render that reads whose holds are the person's (iss-2610290809526033):
// `abcd peers`, `abcd status` and `abcd reserve --holder` each say on stderr,
// in one line, the class of the refusal and that no declared id is read, and
// carry the class in --json as holder_ids_refused; an act still refuses.
func TestEveryRenderNamesARefusedHolderIDsFile(t *testing.T) {
	w := newReserveWorld(t)
	octo := w.clone(t, "octo", octoMail)
	if code, _, stderr := reserveRun(t, octo, "reserve", "iss-1", "--footprint", "a.go"); code != 0 {
		t.Fatalf("reserve: exit %d, %s", code, stderr)
	}
	path := abcdhome.Path(w.home, "holder-ids")
	if err := os.WriteFile(path, []byte("h_0123456789abcdef0123456789abcdef mayasecret\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(path, 0o620); err != nil {
		t.Fatal(err)
	}
	line := claim.HolderIDsRefusedLine(claim.HolderIDsWritableByOthers)
	if line == "" {
		t.Fatal("no line names the refusal")
	}
	clean := func(name, out string) {
		t.Helper()
		if strings.Contains(out, "mayasecret") || strings.Contains(out, "h_0123456789abcdef") {
			t.Fatalf("%s carries the declaration's contents:\n%s", name, out)
		}
	}
	for _, verb := range [][]string{{"peers"}, {"status"}, {"reserve", "--holder"}} {
		name := strings.Join(verb, " ")
		code, stdout, stderr := reserveRun(t, octo, verb...)
		if code != 0 || strings.Count(stderr, line) != 1 || !strings.Contains(stderr, "abcd "+verb[0]+": "+line+"\n") {
			t.Fatalf("%s: exit %d, stderr %q; want exit 0 and the refusal named once", name, code, stderr)
		}
		clean(name, stdout+stderr)
		code, stdout, stderr = reserveRun(t, octo, append(verb, "--json")...)
		var got map[string]any
		if err := json.Unmarshal([]byte(stdout), &got); code != 0 || err != nil {
			t.Fatalf("%s --json: exit %d, %v\n%s%s", name, code, err, stdout, stderr)
		}
		if got["holder_ids_refused"] != claim.HolderIDsWritableByOthers || strings.Count(stderr, line) != 1 {
			t.Fatalf("%s --json: holder_ids_refused = %v, stderr %q; want %q and the line once", name, got["holder_ids_refused"], stderr, claim.HolderIDsWritableByOthers)
		}
		clean(name+" --json", stdout+stderr)
	}
	if code, _, stderr := reserveRun(t, octo, "reserve", "iss-2", "--footprint", "b.go"); code != 2 || !strings.Contains(stderr, abcdhome.Display("holder-ids")) {
		t.Fatalf("an act over the refused declaration: exit %d, %s; want it refused naming the file", code, stderr)
	}
}

// TestPeersAndStatusHelpNameBothHolderIDRefusals: the help of each render
// that names a refused holder-ids file or a refused holder id says both, and
// the JSON field each is carried in.
func TestPeersAndStatusHelpNameBothHolderIDRefusals(t *testing.T) {
	var asJSON bool
	for name, long := range map[string]string{"peers": newPeersCommand(&asJSON).Long, "status": newStatusCommand(&asJSON).Long} {
		flat := strings.Join(strings.Fields(long), " ")
		for _, want := range []string{"(holder_ids_refused in --json)", "(holder_id_refused in --json)", "ABCD_HOLDER_ID"} {
			if !strings.Contains(flat, want) {
				t.Errorf("abcd %s --help does not say %q", name, want)
			}
		}
	}
}
