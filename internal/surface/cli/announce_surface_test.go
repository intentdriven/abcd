package cli

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/claim"
	"github.com/intentdriven/abcd/internal/core/embargo/storepath"
)

// The `abcd announce` front door (spc-2610150728396006 step 2,
// itd-2609151838312703): its operands, usage errors and exit codes, its --json,
// and the embargo check it wires in. The core's write, race and refusals are
// proved in internal/core/claim; these tests prove the verb drives it and says
// what happened.

// statusTipAt is the status branch's head on origin, "" when absent.
func (w *reserveWorld) statusTipAt(t *testing.T, name string) string {
	t.Helper()
	return gitCmd(t, w.origin, "for-each-ref", "--format=%(objectname)", "refs/heads/abcd-status/"+name)
}

type announceJSON struct {
	Kind      string `json:"kind"`
	Item      string `json:"item"`
	Sender    string `json:"sender"`
	Branch    string `json:"branch"`
	SentAt    string `json:"sent_at"`
	ExpiresAt string `json:"expires_at"`
	Commit    string `json:"commit"`
	Attempts  int    `json:"attempts"`
}

// TestAnnounceSendsAndSaysSo is C1 at the surface: the verb pushes one
// announcement to the sender's status branch, exits 0, and says what went out,
// in text and in --json.
func TestAnnounceSendsAndSaysSo(t *testing.T) {
	w := newReserveWorld(t)
	p := w.clone(t, "p", octoMail)
	code, stdout, stderr := reserveRun(t, p, "announce", "merging", "#1012", "--expires-in", "45m", "--json")
	if code != 0 {
		t.Fatalf("exit %d, stderr %q", code, stderr)
	}
	var got announceJSON
	if err := json.Unmarshal([]byte(stdout), &got); err != nil {
		t.Fatalf("--json is not one JSON document: %v\n%s", err, stdout)
	}
	tip := w.statusTipAt(t, "octo")
	if got.Kind != "merging" || got.Item != "#1012" || got.Sender != "octo" || got.Branch != "abcd-status/octo" ||
		got.Commit != tip || tip == "" || got.Attempts != 1 || got.SentAt == "" || got.ExpiresAt == "" {
		t.Fatalf("--json %+v, origin's tip %q", got, tip)
	}
	code, stdout, stderr = reserveRun(t, p, "announce", "merged", "1012")
	want := regexp.MustCompile(`^abcd announce: sent "merged #1012" as octo on abcd-status/octo, standing until \S+Z \(30m0s\)\n$`)
	if code != 0 || !want.MatchString(stdout) {
		t.Fatalf("exit %d, stdout %q, stderr %q; want %s", code, stdout, stderr, want)
	}
	if strings.Contains(stdout, w.home) {
		t.Fatalf("the render carries the home directory: %q", stdout)
	}
}

// TestAnnounceUsageErrorsPushNothing is C3 at the surface: the verb takes a
// kind and an item and no free text, and every refusal exits 2, names what
// was wrong and pushes nothing.
func TestAnnounceUsageErrorsPushNothing(t *testing.T) {
	w := newReserveWorld(t)
	p := w.clone(t, "p", octoMail)
	for _, tc := range []struct {
		name string
		args []string
		want string
	}{
		{"bare", []string{"announce"}, "abcd announce <kind> <item>"},
		{"kind only", []string{"announce", "merging"}, "abcd announce <kind> <item>"},
		{"free text", []string{"announce", "merging", "#1012", "please", "hold"}, "no free text"},
		{"unknown kind", []string{"announce", "shipping", "#1012"}, "kind \"shipping\" is not one of merging, merged, asking, answered, handed-over"},
		{"malformed item", []string{"announce", "merging", "spc-5"}, "is not an issue or intent id (iss-N, itd-N) or a pull-request number (#N)"},
		{"expiry too long", []string{"announce", "merging", "#1012", "--expires-in", "25h"}, "expiry 25h0m0s is outside 1m0s to 24h0m0s"},
		{"record not public", []string{"announce", "merging", "iss-5"}, "iss-5 is not on origin's default branch"},
	} {
		code, stdout, stderr := reserveRun(t, p, tc.args...)
		if code != 2 || !strings.Contains(stderr, tc.want) || !strings.Contains(stderr, "abcd announce: ") {
			t.Errorf("%s: exit %d, stderr %q, stdout %q; want exit 2 and %q", tc.name, code, stderr, stdout, tc.want)
		}
	}
	if tip := w.statusTipAt(t, "octo"); tip != "" {
		t.Fatalf("a refused announcement reached origin: %s", tip)
	}
}

// TestAnnounceIsHeldToTheEmbargoStore is C5 end to end: the verb asks this
// machine's embargo store, through the embargo check, about the record an
// announcement names, and refuses one it holds; and a store that exists but
// cannot be checked refuses every announcement.
func TestAnnounceIsHeldToTheEmbargoStore(t *testing.T) {
	t.Run("a record the store holds", func(t *testing.T) {
		w := newReserveWorld(t)
		p := w.clone(t, "p", octoMail)
		if err := os.MkdirAll(filepath.Join(p, ".abcd", "work", "issues", "open"), 0o755); err != nil {
			t.Fatal(err)
		}
		// Intake reads the advisories of the GitHub repository origin names, from
		// a stand-in gh; origin then points back at the bare remote.
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
		held := intake.Minted[0].IssueID
		code, stdout, stderr = reserveRun(t, p, "announce", "merged", held)
		if code != 2 || !strings.Contains(stderr, "an announcement about "+held+" is refused: the embargo store on this machine holds it") {
			t.Fatalf("announce %s: exit %d, stderr %q, stdout %q; want the embargo refusal", held, code, stderr, stdout)
		}
		if tip := w.statusTipAt(t, "octo"); tip != "" {
			t.Fatalf("an embargoed announcement reached origin: %s", tip)
		}
		if code, _, stderr := reserveRun(t, p, "announce", "merged", "#9"); code != 0 {
			t.Fatalf("a pull request beside the store: exit %d, %s", code, stderr)
		}
	})
	t.Run("a store that cannot be checked", func(t *testing.T) {
		w := newReserveWorld(t)
		p := w.clone(t, "p", octoMail)
		store, ok := storepath.PathForTest(w.home, p)
		if !ok {
			t.Fatal("the fixture has no root commit to key the store on")
		}
		// A plain directory where the store's worktree belongs is a store the
		// check refuses to read.
		if err := os.MkdirAll(store, 0o700); err != nil {
			t.Fatal(err)
		}
		code, stdout, stderr := reserveRun(t, p, "announce", "merging", "#1012")
		if code != 2 || !strings.Contains(stderr, "the embargo store on this machine could not be checked") {
			t.Fatalf("exit %d, stderr %q, stdout %q; want the refusal", code, stderr, stdout)
		}
		if tip := w.statusTipAt(t, "octo"); tip != "" {
			t.Fatalf("an unchecked announcement reached origin: %s", tip)
		}
	})
}

// TestAnnounceNeverShowsAToolsIdentity: under a tool's git identity, the
// shape a hosted container gives a session by default, the verb refuses at
// exit 2 naming the fix, and the identity appears in no output, commit or ref.
// The identity is built here so its literal sits in no file.
func TestAnnounceNeverShowsAToolsIdentity(t *testing.T) {
	w := newReserveWorld(t)
	name, email := "Cla"+"ude", "noreply"+"@"+"anthro"+"pic.com"
	p := w.clone(t, "p", email)
	gitCmd(t, p, "config", "user.name", name)
	for _, args := range [][]string{{"announce", "merging", "#1012"}, {"announce", "merging", "#1012", "--json"}} {
		code, stdout, stderr := reserveRun(t, p, args...)
		if code != 2 || !strings.Contains(stdout+stderr, "abcd cannot yet work out whose GitHub account this session uses") ||
			strings.Contains(stdout+stderr, "git config") {
			t.Fatalf("%v: exit %d, stderr %q; want the refusal naming the cause", args, code, stderr)
		}
		if out := strings.ToLower(stdout + stderr); strings.Contains(out, "anthro") || strings.Contains(out, strings.ToLower(name)) {
			t.Fatalf("%v: the output shows the rejected identity:\n%s%s", args, stdout, stderr)
		}
	}
	for _, dir := range []string{w.origin, p} {
		if refs := gitCmd(t, dir, "for-each-ref", "--format=%(refname)", "refs/heads/abcd-status/", "refs/remotes/origin/abcd-status/"); refs != "" {
			t.Fatalf("a status ref was written in %s: %s", filepath.Base(dir), refs)
		}
		if all := gitCmd(t, dir, "log", "--all", "--format=%an %ae %cn %ce %B"); strings.Contains(strings.ToLower(all), "anthro") {
			t.Fatalf("a commit in %s carries the rejected identity", filepath.Base(dir))
		}
	}
}

// TestAnnounceSaysWhatBecameOfThePushOnAnUnclassedFailure mirrors reserve: a
// failure the core does not class as a refusal says what became of the push.
// Before any push it says nothing was pushed; after pushes origin refused, it
// says plainly that one may still have reached origin, naming the branch; a
// refusal, which says so itself, is passed through unchanged.
func TestAnnounceSaysWhatBecameOfThePushOnAnUnclassedFailure(t *testing.T) {
	w := newReserveWorld(t)
	p := w.clone(t, "p", octoMail)
	for _, tc := range []struct {
		name    string
		err     error
		want    string
		wantNot string
	}{
		{"before any push", &claim.AnnounceError{Err: errors.New("git said nothing"), Pushes: 0},
			"abcd announce: git said nothing; not sent, and nothing was pushed", "may have"},
		{"after refused pushes", &claim.AnnounceError{Err: errors.New("git said nothing"), Pushes: 2, Branch: "abcd-status/octo"},
			"abcd announce: git said nothing; not sent as far as git reported, but origin refused 2 earlier pushes to abcd-status/octo " +
				"and a refused push may still have reached it: fetch origin and look at abcd-status/octo before announcing again", "nothing was pushed"},
		{"after one refused push", &claim.AnnounceError{Err: errors.New("git said nothing"), Pushes: 1, Branch: "abcd-status/octo"},
			"abcd announce: git said nothing; not sent as far as git reported, but origin refused 1 earlier push to abcd-status/octo " +
				"and a refused push may still have reached it: fetch origin and look at abcd-status/octo before announcing again", "pushes"},
		{"a refusal", &claim.NotSentError{Reason: "origin refused the push: no"},
			"abcd announce: not sent: origin refused the push: no\n", "nothing was pushed"},
	} {
		announceAct = func(string, claim.AnnounceRequest, claim.Options) (claim.AnnounceResult, error) {
			return claim.AnnounceResult{}, tc.err
		}
		t.Cleanup(func() { announceAct = claim.Announce })
		code, stdout, stderr := reserveRun(t, p, "announce", "merging", "#1012")
		if code != 2 || !strings.Contains(stderr, tc.want) || strings.Contains(stderr, tc.wantNot) || stdout != "" {
			t.Errorf("%s: exit %d, stderr %q, stdout %q; want exit 2 and %q, without %q", tc.name, code, stderr, stdout, tc.want, tc.wantNot)
		}
	}
}

// TestAnnounceSaysAPushMayHaveLandedWhenTheBranchCannotBeReadBack drives the
// real core: the checkout's pre-push hook points origin somewhere that does
// not exist and refuses the push, so the fetch after it fails too. Whether the
// push landed is not known, so the verb does not say "not sent" or that nothing
// was pushed: it says a refused push may still have reached origin, naming the
// branch to look at.
func TestAnnounceSaysAPushMayHaveLandedWhenTheBranchCannotBeReadBack(t *testing.T) {
	w := newReserveWorld(t)
	p := w.clone(t, "p", octoMail)
	nowhere := filepath.Join(t.TempDir(), "nowhere.git")
	hook := "#!/bin/sh\ngit config remote.origin.url '" + nowhere + "'\necho 'refused here' >&2\nexit 1\n"
	if err := os.WriteFile(filepath.Join(p, ".git", "hooks", "pre-push"), []byte(hook), 0o755); err != nil {
		t.Fatal(err)
	}
	code, stdout, stderr := reserveRun(t, p, "announce", "merging", "#1012")
	if code != 2 || stdout != "" || strings.Contains(stderr, "nothing was pushed") || strings.Contains(stderr, "not sent: ") ||
		!strings.Contains(stderr, "refused here") ||
		!strings.Contains(stderr, "origin refused 1 earlier push to abcd-status/octo and a refused push may still have reached it") {
		t.Fatalf("exit %d, stderr %q, stdout %q; want exit 2 saying the push may have reached origin", code, stderr, stdout)
	}
}
