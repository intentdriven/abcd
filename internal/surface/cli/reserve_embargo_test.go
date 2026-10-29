package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/embargo"
	"github.com/intentdriven/abcd/internal/core/embargo/storepath"
)

// A reservation consults this machine's embargo store, as an announcement
// does: the reserve verb's and the implement loop's, which build their own
// options and get the check the front door registers.

// heldRecord takes in one advisory from a stand-in gh into the store under
// w's HOME and returns the private record it minted.
func heldRecord(t *testing.T, w *reserveWorld, p string) string {
	t.Helper()
	if err := os.MkdirAll(filepath.Join(p, ".abcd", "work", "issues", "open"), 0o755); err != nil {
		t.Fatal(err)
	}
	bin := t.TempDir()
	advisory := `{"ghsa_id":"` + surfaceGHSA + `","cve_id":null,"summary":"a sample report","description":"the report",` +
		`"severity":"high","state":"triage","vulnerabilities":[]}`
	script := "#!/bin/sh\nfor a in \"$@\"; do case \"$a\" in\n  state=triage) echo '[" + advisory + "]'; exit 0 ;;\n" +
		"  state=*) echo '[]'; exit 0 ;;\nesac; done\necho '" + advisory + "'\n"
	if err := os.WriteFile(filepath.Join(bin, "gh"), []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
	gitCmd(t, p, "remote", "set-url", "origin", "https://github.com/example/example.git")
	code, stdout, stderr := reserveRun(t, p, "embargo", "intake", "--json")
	gitCmd(t, p, "remote", "set-url", "origin", w.origin)
	var intake struct {
		Minted []struct {
			IssueID string `json:"issue_id"`
		} `json:"minted"`
	}
	if code != 0 || json.Unmarshal([]byte(stdout), &intake) != nil || len(intake.Minted) != 1 {
		t.Fatalf("intake: exit %d, %s%s", code, stdout, stderr)
	}
	return intake.Minted[0].IssueID
}

// unreadableStore puts a state file that does not parse where p's store
// belongs, a store the check refuses to read, and returns that place as a
// refusal names it: under ~ for the home.
func unreadableStore(t *testing.T, w *reserveWorld, p string) string {
	t.Helper()
	loc, err := storepath.Locate(w.home, p)
	if err != nil || loc.Registered {
		t.Fatalf("the fixture's store place: %+v, %v", loc, err)
	}
	state := filepath.Join(loc.Path, filepath.FromSlash(embargo.StateRel))
	if err := os.MkdirAll(filepath.Dir(state), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(state, []byte("{not json"), 0o600); err != nil {
		t.Fatal(err)
	}
	return "~" + strings.TrimPrefix(loc.Path, w.home)
}

func TestReserveConsultsTheEmbargoStore(t *testing.T) {
	t.Run("a record the store holds", func(t *testing.T) {
		w := newReserveWorld(t)
		p := w.clone(t, "p", octoMail)
		held := heldRecord(t, w, p)
		code, stdout, stderr := reserveRun(t, p, "reserve", held, "--footprint", "internal/a.go")
		if code != 2 || !strings.Contains(stderr, "a reservation of "+held+" is refused: the embargo store on this machine holds it") {
			t.Fatalf("reserve %s: exit %d, stderr %q, stdout %q; want the embargo refusal", held, code, stderr, stdout)
		}
		if tip := w.remoteTip(t, held); tip != "" {
			t.Fatalf("a reservation of a held record reached origin: %s", tip)
		}
		if code, _, stderr := reserveRun(t, p, "reserve", "iss-9", "--footprint", "a.go"); code != 0 {
			t.Fatalf("a record the store does not hold: exit %d, %s", code, stderr)
		}
	})
	t.Run("a store that cannot be checked", func(t *testing.T) {
		w := newReserveWorld(t)
		reservedRecords(t, w)
		p := w.clone(t, "p", octoMail)
		store := unreadableStore(t, w, p)
		code, stdout, stderr := reserveRun(t, p, "reserve", "iss-5", "--footprint", "a.go")
		if code != 2 || !strings.Contains(stderr, "the embargo store on this machine could not be checked") {
			t.Fatalf("reserve: exit %d, stderr %q, stdout %q; want the refusal", code, stderr, stdout)
		}
		// The refusal names the store that could not be read, and its repair.
		if !strings.Contains(stderr, "the store at "+store) || !strings.Contains(stderr, "settle it") {
			t.Fatalf("reserve: stderr %q; want the store's path and its repair", stderr)
		}
		// The build loop builds its own options and names no check: it is
		// held to the one the front door registered.
		code, stdout, stderr = reserveRun(t, p, "build", "itd-8")
		if code == 0 || !strings.Contains(stdout+stderr, "the embargo store on this machine could not be checked") {
			t.Fatalf("build: exit %d, stderr %q, stdout %q; want the refusal", code, stderr, stdout)
		}
		for _, id := range []string{"iss-5", "itd-8"} {
			if tip := w.remoteTip(t, id); tip != "" {
				t.Fatalf("an unchecked reservation of %s reached origin: %s", id, tip)
			}
		}
	})
}
