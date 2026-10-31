package cli

import (
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/abcdhome"
	"github.com/intentdriven/abcd/internal/core/claim"
)

// A holder id the reader cannot read, kept or set, is named on every render
// that reads whose holds are the person's (iss-2610290809526033): `abcd
// peers` and `abcd status` each say on stderr, in one line, the class of the
// refusal and that no holder id is read, and carry the class in --json as
// holder_id_refused, never the value or the file's contents; `abcd reserve
// --holder`, which prints the id, refuses naming the file, as an act does.
func TestEveryRenderNamesARefusedHolderID(t *testing.T) {
	for _, c := range []struct {
		name, class string
		setup       func(t *testing.T, home string)
	}{
		{"a kept file another account can write", claim.HolderIDsWritableByOthers, func(t *testing.T, home string) {
			path := abcdhome.Path(home, "holder-id")
			if err := os.Chmod(path, 0o620); err != nil {
				t.Fatal(err)
			}
		}},
		{"a variable set to no id", claim.HolderIDEnvNotID, func(t *testing.T, _ string) {
			t.Setenv(claim.HolderIDEnv, "h_notanidsecret")
		}},
	} {
		t.Run(c.name, func(t *testing.T) {
			w := newReserveWorld(t)
			octo := w.clone(t, "octo", octoMail)
			if code, _, stderr := reserveRun(t, octo, "reserve", "iss-1", "--footprint", "a.go"); code != 0 {
				t.Fatalf("reserve: exit %d, %s", code, stderr)
			}
			c.setup(t, w.home)
			line := claim.HolderIDRefusedLine(c.class)
			if line == "" {
				t.Fatal("no line names the refusal")
			}
			for _, verb := range [][]string{{"peers"}, {"status"}} {
				name := strings.Join(verb, " ")
				code, stdout, stderr := reserveRun(t, octo, verb...)
				if code != 0 || strings.Count(stderr, line) != 1 || !strings.Contains(stderr, "abcd "+verb[0]+": "+line+"\n") {
					t.Fatalf("%s: exit %d, stderr %q; want exit 0 and the refusal named once", name, code, stderr)
				}
				code, stdout, stderr = reserveRun(t, octo, append(verb, "--json")...)
				var got map[string]any
				if err := json.Unmarshal([]byte(stdout), &got); code != 0 || err != nil {
					t.Fatalf("%s --json: exit %d, %v\n%s%s", name, code, err, stdout, stderr)
				}
				if got["holder_id_refused"] != c.class || strings.Contains(stdout+stderr, "notanidsecret") {
					t.Fatalf("%s --json: holder_id_refused = %v, stderr %q; want %q and no value echoed", name, got["holder_id_refused"], stderr, c.class)
				}
			}
			if code, _, stderr := reserveRun(t, octo, "reserve", "--holder"); code != 2 || strings.Contains(stderr, "notanidsecret") ||
				!strings.Contains(stderr, claim.HolderIDFile) && !strings.Contains(stderr, claim.HolderIDEnv) {
				t.Fatalf("reserve --holder: exit %d, %s; want it refused naming the file or the variable", code, stderr)
			}
		})
	}
}
