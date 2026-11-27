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
// copy against the ceiling, here this machine's reservation.max_live lowering
// the default and named as its layer, and from four fifths of it warns that
// another account on the same login counts too.
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
	if c == nil || c.Holder != heldBy(t, p) || c.Live != 4 || c.Limit != 5 || c.ThisCopy != 2 || c.Elsewhere != 1 ||
		len(c.Siblings) != 1 || c.Siblings[0].Name != "lane-b" || c.Siblings[0].Live != 1 || !c.Warn ||
		!strings.Contains(c.Warning, "whose holds are yours") {
		t.Fatalf("ceiling = %+v; want P's holder id 4 of 5 (this copy 2, lane-b 1, elsewhere 1) with the warning", c)
	}

	code, text, _ := reserveRun(t, p, "peers")
	if code != 0 {
		t.Fatalf("peers: exit %d\n%s", code, text)
	}
	id := heldBy(t, p)
	for _, want := range []string{`from the worktree "lane-b" on this machine`,
		`your holds: ` + id + ` 4 of 5 (machine) (this copy 2, "lane-b" 1, elsewhere 1)`, "warning: " + id + " holds 4 of the 5"} {
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
// does not read shows the cap so far, and the text form says beside the holds
// that an act adding a reservation refuses until the file reads, as --json
// does in the ceiling's note, so a reader never takes the default for a limit
// someone set; the act itself is refused, naming the file.
func TestPeersTextSaysWhyTheCeilingIsTheDefault(t *testing.T) {
	w := newReserveWorld(t)
	p := w.clone(t, "p", octoMail)
	if code, out, errOut := reserveRun(t, p, "reserve", "iss-1", "--footprint", "iss-1.go"); code != 0 {
		t.Fatalf("reserve: exit %d\n%s%s", code, out, errOut)
	}
	cfg := abcdhome.Path(w.home, "config.json")
	if err := os.MkdirAll(filepath.Dir(cfg), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(cfg, []byte(`{"reservation": `), 0o600); err != nil {
		t.Fatal(err)
	}
	code, text, _ := reserveRun(t, p, "peers")
	if code != 0 {
		t.Fatalf("peers: exit %d\n%s", code, text)
	}
	for _, want := range []string{"your holds: " + heldBy(t, p) + " 1 of 10 (bundled)",
		"cannot be read for reservation.max_live", "an act that adds a reservation refuses until it can"} {
		if !strings.Contains(text, want) {
			t.Errorf("the listing lacks %q:\n%s", want, text)
		}
	}
	if code, out, errOut := reserveRun(t, p, "reserve", "iss-2", "--footprint", "iss-2.go"); code != 2 || !strings.Contains(errOut, "config.json cannot be read for reservation.max_live") {
		t.Errorf("a second reservation: exit %d\n%s%s; want it refused naming the file", code, out, errOut)
	}
}

// TestPeersNamesTheRepositorysCeiling (itd-2611251809580662): with the
// repository's file on origin's default branch setting reservation.max_live
// 5, the row names the layer beside the number, the warning names the
// repository's ceiling and not this machine's, and --json carries
// limit_layer and limit_origin.
func TestPeersNamesTheRepositorysCeiling(t *testing.T) {
	w := newReserveWorld(t)
	seed := filepath.Join(w.home, "seed")
	writeRel(t, seed, ".abcd/config.json", `{"reservation": {"max_live": 5}}`)
	gitCmd(t, seed, "add", "-A")
	gitCommit(t, seed, "commit", "-q", "-m", "the repository's ceiling")
	gitCmd(t, seed, "push", "-q", "origin", "main")
	p := w.clone(t, "p", octoMail)
	for _, id := range []string{"iss-1", "iss-2", "iss-3", "iss-4"} {
		if code, out, errOut := reserveRun(t, p, "reserve", id, "--footprint", id+".go"); code != 0 {
			t.Fatalf("reserve %s: exit %d\n%s%s", id, code, out, errOut)
		}
	}
	code, raw, errOut := reserveRun(t, p, "peers", "--json")
	if code != 0 {
		t.Fatalf("peers: exit %d\n%s%s", code, raw, errOut)
	}
	var got struct {
		Ceiling *struct {
			Live        int    `json:"live"`
			Limit       int    `json:"limit"`
			LimitLayer  string `json:"limit_layer"`
			LimitOrigin string `json:"limit_origin"`
			Warning     string `json:"warning"`
		} `json:"ceiling"`
	}
	if err := json.Unmarshal([]byte(raw), &got); err != nil {
		t.Fatalf("peers --json: %v\n%s", err, raw)
	}
	c := got.Ceiling
	if c == nil || c.Live != 4 || c.Limit != 5 || c.LimitLayer != "repository" || c.LimitOrigin != ".abcd/config.json at origin/main" ||
		!strings.Contains(c.Warning, "the repository's ceiling") || strings.Contains(c.Warning, "this machine's") {
		t.Fatalf("ceiling = %+v; want 4 of the repository's 5, its file at the branch, warned of the repository's ceiling", c)
	}
	code, text, _ := reserveRun(t, p, "peers")
	if code != 0 {
		t.Fatalf("peers: exit %d\n%s", code, text)
	}
	id := heldBy(t, p)
	for _, want := range []string{"your holds: " + id + " 4 of 5 (repository) (this copy 4, elsewhere 0)",
		"warning: " + id + " holds 4 of the 5 live reservations the repository's ceiling"} {
		if !strings.Contains(text, want) {
			t.Errorf("the listing lacks %q:\n%s", want, text)
		}
	}
}

// TestPeersNamesTheDrainsLanesInWork (itd-2611251809580662 step 2): with a
// drain in progress in the checkout, a row under the ceiling's names its
// lanes in work against the limit it began with and the layer that set it,
// in text and in --json; with none in progress, or one ended, the row is
// absent.
func TestPeersNamesTheDrainsLanesInWork(t *testing.T) {
	w := newReserveWorld(t)
	p := w.clone(t, "p", octoMail)
	if code, out, errOut := reserveRun(t, p, "reserve", "iss-1", "--footprint", "iss-1.go"); code != 0 {
		t.Fatalf("reserve: exit %d\n%s%s", code, out, errOut)
	}
	code, text, _ := reserveRun(t, p, "peers")
	if code != 0 || strings.Contains(text, "this drain:") {
		t.Fatalf("with no drain in progress the row is absent: exit %d\n%s", code, text)
	}
	drain := func(stopped string) {
		t.Helper()
		ended := ""
		if stopped != "" {
			ended = `, "stopped": "` + stopped + `", "ended_at": "2026-11-26T10:00:00Z"`
		}
		writeRel(t, p, ".abcd/.work.local/run/drain.json", `{"schema_version": 5, "started_at": "2026-11-26T09:00:00Z", "updated_at": "2026-11-26T09:00:00Z",
			"rule": "adr-1", "max": 0, "pace": null, "lanes": [{"issue": "iss-2611260000000001", "run_id": "run-2611260000000001",
			"opened_at": "2026-11-26T09:00:00Z", "outcome": "in-progress"}], "hand_backs": [],
			"in_work_max": {"value": 2, "layer": "repo", "origin": ".abcd/config.json at origin/main"}`+ended+`}`)
	}
	drain("")
	code, text, errOut := reserveRun(t, p, "peers")
	if code != 0 || !strings.Contains(text, "  this drain: 1 of 2 lanes in work (repository)\n") {
		t.Fatalf("the drain's row names its lanes in work and its limit's layer: exit %d\n%s%s", code, text, errOut)
	}
	if strings.Index(text, "this drain:") < strings.Index(text, "your holds:") {
		t.Errorf("the drain's row comes under the ceiling's:\n%s", text)
	}
	code, raw, errOut := reserveRun(t, p, "peers", "--json")
	if code != 0 {
		t.Fatalf("peers --json: exit %d\n%s%s", code, raw, errOut)
	}
	var got struct {
		DrainInWork *struct {
			Count  int    `json:"count"`
			Limit  int    `json:"limit"`
			Layer  string `json:"layer"`
			Origin string `json:"origin"`
		} `json:"drain_in_work"`
	}
	if err := json.Unmarshal([]byte(raw), &got); err != nil {
		t.Fatalf("peers --json: %v\n%s", err, raw)
	}
	if d := got.DrainInWork; d == nil || d.Count != 1 || d.Limit != 2 || d.Layer != "repo" || d.Origin != ".abcd/config.json at origin/main" {
		t.Fatalf("--json carries the drain's lanes in work: %+v", d)
	}
	drain("empty")
	if code, raw, _ := reserveRun(t, p, "peers", "--json"); code != 0 || strings.Contains(raw, "drain_in_work") {
		t.Fatalf("with the drain ended the row is absent: exit %d\n%s", code, raw)
	}
}
