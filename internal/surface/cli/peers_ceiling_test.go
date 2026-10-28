package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/abcdhome"
)

// TestPeersNamesEachHoldsCopyAndWarnsNearTheCeiling (iss-2610231004140527):
// one login's holds come from this copy, a sibling worktree here and another
// clone; peers names the sibling by its worktree, counts the reader's holds by
// copy against this machine's reservation.max_live, and from four fifths of
// it warns that another account on the same login counts too.
func TestPeersNamesEachHoldsCopyAndWarnsNearTheCeiling(t *testing.T) {
	w := newReserveWorld(t)
	cfg := abcdhome.Path(w.home, "config.json")
	if err := os.MkdirAll(filepath.Dir(cfg), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(cfg, []byte(`{"reservation": {"max_live": 5}}`), 0o600); err != nil {
		t.Fatal(err)
	}
	p := w.clone(t, "p", octoMail)
	sib := filepath.Join(w.home, "wt", "lane-b")
	gitCmd(t, p, "worktree", "add", "-q", "-b", "feat/lane-b", sib, "main")
	if err := os.MkdirAll(filepath.Join(sib, ".abcd", ".work.local"), 0o755); err != nil {
		t.Fatal(err)
	}
	far := w.clone(t, "far", octoMail)
	q := w.clone(t, "q", quinnMail)
	for _, c := range []struct {
		dir, id string
	}{{p, "iss-1"}, {p, "iss-2"}, {sib, "iss-3"}, {far, "iss-4"}, {q, "iss-9"}} {
		if code, out, errOut := reserveRun(t, c.dir, "reserve", c.id, "--footprint", c.id+".go"); code != 0 {
			t.Fatalf("reserve %s: exit %d\n%s%s", c.id, code, out, errOut)
		}
	}

	code, raw, errOut := reserveRun(t, p, "peers", "--json")
	if code != 0 {
		t.Fatalf("peers: exit %d\n%s%s", code, raw, errOut)
	}
	var got struct {
		Reservations []struct {
			Record  string `json:"record"`
			Copy    string `json:"copy"`
			Sibling string `json:"sibling"`
		} `json:"reservations"`
		Ceiling *struct {
			Holder   string `json:"holder"`
			Live     int    `json:"live"`
			Limit    int    `json:"limit"`
			ThisCopy int    `json:"this_copy"`
			Siblings []struct {
				Name string `json:"name"`
				Live int    `json:"live"`
			} `json:"siblings"`
			Elsewhere int    `json:"elsewhere"`
			Warn      bool   `json:"warn"`
			Warning   string `json:"warning"`
		} `json:"ceiling"`
	}
	if err := json.Unmarshal([]byte(raw), &got); err != nil {
		t.Fatalf("peers --json: %v\n%s", err, raw)
	}
	for _, r := range got.Reservations {
		want := ""
		if r.Record == "iss-3" {
			want = "lane-b"
		}
		if r.Sibling != want {
			t.Errorf("%s sibling = %q; want %q", r.Record, r.Sibling, want)
		}
	}
	c := got.Ceiling
	if c == nil || c.Holder != "octo" || c.Live != 4 || c.Limit != 5 || c.ThisCopy != 2 || c.Elsewhere != 1 ||
		len(c.Siblings) != 1 || c.Siblings[0].Name != "lane-b" || c.Siblings[0].Live != 1 || !c.Warn ||
		!strings.Contains(c.Warning, "same login") {
		t.Fatalf("ceiling = %+v; want octo 4 of 5 (this copy 2, lane-b 1, elsewhere 1) with the warning", c)
	}

	code, text, _ := reserveRun(t, p, "peers")
	if code != 0 {
		t.Fatalf("peers: exit %d\n%s", code, text)
	}
	for _, want := range []string{`from the worktree "lane-b" on this machine`,
		`your holds: octo 4 of 5 (this copy 2, "lane-b" 1, elsewhere 1)`, "warning: octo holds 4 of the 5"} {
		if !strings.Contains(text, want) {
			t.Errorf("the listing lacks %q:\n%s", want, text)
		}
	}
	if key, err := os.ReadFile(filepath.Join(sib, ".abcd", ".work.local", "reservation", "copy-key")); err != nil ||
		strings.Contains(raw+text, strings.TrimSpace(string(key))) {
		t.Fatalf("the sibling's copy key is shown, or it has none (%v)", err)
	}
}

// TestPeersTextSaysWhyTheCeilingIsTheDefault: a machine configuration that
// does not read leaves the cap at its default, and the text form says so
// beside the holds, as --json does in the ceiling's note, so a reader never
// takes the default for a limit someone set.
func TestPeersTextSaysWhyTheCeilingIsTheDefault(t *testing.T) {
	w := newReserveWorld(t)
	cfg := abcdhome.Path(w.home, "config.json")
	if err := os.MkdirAll(filepath.Dir(cfg), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(cfg, []byte(`{"reservation": `), 0o600); err != nil {
		t.Fatal(err)
	}
	p := w.clone(t, "p", octoMail)
	if code, out, errOut := reserveRun(t, p, "reserve", "iss-1", "--footprint", "iss-1.go"); code != 0 {
		t.Fatalf("reserve: exit %d\n%s%s", code, out, errOut)
	}
	code, text, _ := reserveRun(t, p, "peers")
	if code != 0 {
		t.Fatalf("peers: exit %d\n%s", code, text)
	}
	for _, want := range []string{"your holds: octo 1 of 10", "note: the machine configuration could not be read"} {
		if !strings.Contains(text, want) {
			t.Errorf("the listing lacks %q:\n%s", want, text)
		}
	}
}
