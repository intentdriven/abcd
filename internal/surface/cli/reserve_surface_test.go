package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/gittest"
)

// The `abcd reserve` front door (spc-2610031753359849 step 5,
// itd-2609091034175565): its flags, usage errors and exit codes, its --json for
// each act, and renders that carry no host or home path. The core's push table
// is proved in internal/core/claim; these tests prove the verb drives it and
// says what happened.

// reserveWorld is a bare origin and a seeded main branch, under a temp HOME so
// every path a render could leak is one the home redaction must catch.
type reserveWorld struct {
	home   string
	origin string
}

func newReserveWorld(t *testing.T) *reserveWorld {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("ABCD_PLUGIN_ROOT", "")
	t.Setenv("CLAUDE_PLUGIN_ROOT", "")
	w := &reserveWorld{home: home, origin: filepath.Join(home, "origin.git")}
	gittest.InitBare(t, w.origin)
	seed := filepath.Join(home, "seed")
	gitInitAt(t, seed)
	writeRel(t, seed, "README.md", "seed\n")
	gitCmd(t, seed, "add", "-A")
	gitCommit(t, seed, "commit", "-q", "-m", "seed")
	gitCmd(t, seed, "remote", "add", "origin", w.origin)
	gitCmd(t, seed, "push", "-q", "origin", "main")
	return w
}

// clone is one working copy under HOME, with its git identity, its local tier
// (ignored, as a managed repository's .gitignore has it) and no hooks.
func (w *reserveWorld) clone(t *testing.T, name, email string) string {
	t.Helper()
	dir := filepath.Join(w.home, name)
	gitCmd(t, w.home, "clone", "-q", w.origin, dir)
	for _, kv := range [][2]string{{"user.name", "Pat Example"}, {"user.email", email}, {"gc.auto", "0"},
		{"maintenance.auto", "false"}, {"commit.gpgsign", "false"}} {
		gitCmd(t, dir, "config", kv[0], kv[1])
	}
	if err := os.MkdirAll(filepath.Join(dir, ".abcd", ".work.local"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, ".git", "info", "exclude"), []byte(".abcd/.work.local/\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	return dir
}

// remoteTip is the reservation branch's head on origin, "" when absent.
func (w *reserveWorld) remoteTip(t *testing.T, id string) string {
	t.Helper()
	return gitCmd(t, w.origin, "for-each-ref", "--format=%(objectname)", "refs/heads/abcd-reserved/"+id)
}

// reserveRun runs the CLI in dir and returns its exit code and both streams.
func reserveRun(t *testing.T, dir string, args ...string) (int, string, string) {
	t.Helper()
	t.Chdir(dir)
	var stdout, stderr bytes.Buffer
	code := Run(args, &stdout, &stderr)
	return code, stdout.String(), stderr.String()
}

// reserveJSON is the --json document of one act.
type reserveJSON struct {
	Act             string   `json:"act"`
	Record          string   `json:"record"`
	Holder          string   `json:"holder"`
	Copy            string   `json:"copy"`
	Footprint       []string `json:"footprint"`
	ClaimedAt       string   `json:"claimed_at"`
	EffectiveExpiry string   `json:"effective_expiry"`
	Commit          string   `json:"commit"`
	Replaced        *struct {
		Holder          string `json:"holder"`
		Copy            string `json:"copy"`
		Commit          string `json:"commit"`
		EffectiveExpiry string `json:"effective_expiry"`
		Unreadable      string `json:"unreadable"`
	} `json:"replaced"`
}

func decodeReserve(t *testing.T, out string) reserveJSON {
	t.Helper()
	var got reserveJSON
	if err := json.Unmarshal([]byte(out), &got); err != nil {
		t.Fatalf("--json is not one JSON document: %v\n%s", err, out)
	}
	return got
}

const (
	octoMail  = "1234+octo@users.noreply.github.com"
	quinnMail = "99+quinn@users.noreply.github.com"
)

// TestReserveWithoutAFootprintIsRefused is C27: a reservation names the files
// the work will change, so one without --footprint is refused at exit 2 and
// nothing reaches origin.
func TestReserveWithoutAFootprintIsRefused(t *testing.T) {
	w := newReserveWorld(t)
	p := w.clone(t, "p", octoMail)
	code, stdout, stderr := reserveRun(t, p, "reserve", "iss-5")
	want := "abcd reserve: a reservation names the files the work will change: pass --footprint <path> once per path; nothing was pushed"
	if code != 2 || !strings.Contains(stderr, want) {
		t.Fatalf("exit %d, stderr %q, stdout %q; want exit 2 and %q", code, stderr, stdout, want)
	}
	if tip := w.remoteTip(t, "iss-5"); tip != "" {
		t.Fatalf("a refused reservation reached origin: %s", tip)
	}
}

// TestReserveUsageErrorsAndExitCodes: the verb's operand and flag errors are
// usage errors at exit 2 that push nothing; a record held by another person is
// the back-off exit 3; a reservation that reaches origin exits 0.
func TestReserveUsageErrorsAndExitCodes(t *testing.T) {
	w := newReserveWorld(t)
	p := w.clone(t, "p", octoMail)
	q := w.clone(t, "q", quinnMail)

	for _, tc := range []struct {
		name string
		args []string
		want string
	}{
		{"bare", []string{"reserve"}, "abcd peers"},
		{"two operands", []string{"reserve", "iss-5", "iss-6", "--footprint", "a.go"}, "one record id"},
		{"release with a footprint", []string{"reserve", "iss-5", "--release", "--footprint", "a.go"}, "--release"},
		{"release with take-over", []string{"reserve", "iss-5", "--release", "--take-over"}, "--release"},
		{"release with a lease", []string{"reserve", "iss-5", "--release", "--lease", "1h"}, "--release"},
		{"not a reservable id", []string{"reserve", "spc-5", "--footprint", "a.go"}, "is not an issue or intent id"},
		{"bad footprint", []string{"reserve", "iss-5", "--footprint", "../escape"}, "is not a repository-relative path"},
		{"lease too long", []string{"reserve", "iss-5", "--footprint", "a.go", "--lease", "25h"}, "is outside 1m0s to 24h0m0s"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			code, _, stderr := reserveRun(t, p, tc.args...)
			if code != 2 || !strings.Contains(stderr, tc.want) {
				t.Fatalf("exit %d, stderr %q; want exit 2 naming %q", code, stderr, tc.want)
			}
			if tip := w.remoteTip(t, "iss-5"); tip != "" {
				t.Fatalf("a refused invocation reached origin: %s", tip)
			}
		})
	}

	code, stdout, stderr := reserveRun(t, p, "reserve", "iss-5", "--footprint", "a.go", "--lease", "90m")
	if code != 0 {
		t.Fatalf("P's reservation: exit %d\n%s%s", code, stdout, stderr)
	}
	if !strings.Contains(stdout, "reserved iss-5") {
		t.Errorf("the reservation does not say what it did:\n%s", stdout)
	}
	code, _, stderr = reserveRun(t, q, "reserve", "iss-5", "--footprint", "b.go")
	if code != 3 || !strings.Contains(stderr, "iss-5 is reserved by octo until") || !strings.Contains(stderr, "back off") {
		t.Fatalf("Q's reservation over P's live one: exit %d, stderr %q; want the back-off exit 3", code, stderr)
	}
	code, _, stderr = reserveRun(t, q, "reserve", "iss-5", "--release")
	if code != 2 || !strings.Contains(stderr, "not by you") {
		t.Fatalf("Q's release of P's reservation: exit %d, stderr %q; want exit 2", code, stderr)
	}
}

// TestReserveJSONForEachAct: --json reports the act, the record, the holder,
// the copy, the footprint and the effective expiry of a reservation, a
// renewal, a release and a release already made.
func TestReserveJSONForEachAct(t *testing.T) {
	w := newReserveWorld(t)
	p := w.clone(t, "p", octoMail)

	for _, step := range []struct {
		args []string
		act  string
	}{
		{[]string{"reserve", "itd-7", "--footprint", "internal/core/claim", "--footprint", "commands/reserve.md", "--json"}, "reserved"},
		{[]string{"reserve", "itd-7", "--footprint", "internal/core/claim", "--json"}, "renewed"},
		{[]string{"reserve", "itd-7", "--release", "--json"}, "released"},
		{[]string{"reserve", "itd-7", "--release", "--json"}, "already_released"},
	} {
		code, stdout, stderr := reserveRun(t, p, step.args...)
		if code != 0 {
			t.Fatalf("%v: exit %d\n%s%s", step.args, code, stdout, stderr)
		}
		got := decodeReserve(t, stdout)
		if got.Act != step.act || got.Record != "itd-7" || got.Holder != "octo" || got.Copy != "this" ||
			len(got.Footprint) == 0 || got.EffectiveExpiry == "" || got.ClaimedAt == "" || got.Commit == "" || got.Replaced != nil {
			t.Fatalf("%v: payload %+v; want act %s for itd-7 held by octo from this copy", step.args, got, step.act)
		}
		if got.Commit != w.remoteTip(t, "itd-7") {
			t.Fatalf("%v: payload names commit %s, origin holds %s", step.args, got.Commit, w.remoteTip(t, "itd-7"))
		}
	}
}

// TestReserveTakeOverNamesTheCopyReplaced is C52 at the front door: the same
// person's second copy is refused without --take-over (exit 3), and with it
// takes the reservation over, naming the copy it replaced as another copy and
// never printing either copy's key.
func TestReserveTakeOverNamesTheCopyReplaced(t *testing.T) {
	w := newReserveWorld(t)
	laptop := w.clone(t, "laptop", octoMail)
	cloud := w.clone(t, "cloud", octoMail)

	if code, stdout, stderr := reserveRun(t, laptop, "reserve", "iss-5", "--footprint", "a.go"); code != 0 {
		t.Fatalf("the laptop's reservation: exit %d\n%s%s", code, stdout, stderr)
	}
	code, _, stderr := reserveRun(t, cloud, "reserve", "iss-5", "--footprint", "a.go")
	if code != 3 || !strings.Contains(stderr, "you are already building iss-5 on another computer") || !strings.Contains(stderr, "--take-over") {
		t.Fatalf("the cloud copy without --take-over: exit %d, stderr %q; want the back-off naming --take-over", code, stderr)
	}

	code, text, stderr := reserveRun(t, cloud, "reserve", "iss-5", "--footprint", "a.go", "--take-over")
	if code != 0 || !strings.Contains(text, "took over iss-5") || !strings.Contains(text, "your copy on another computer") {
		t.Fatalf("the take-over: exit %d, stdout %q, stderr %q; want it to name the copy replaced", code, text, stderr)
	}
	code, raw, stderr := reserveRun(t, laptop, "reserve", "iss-5", "--footprint", "a.go", "--take-over", "--json")
	if code != 0 {
		t.Fatalf("taking it back: exit %d\n%s%s", code, raw, stderr)
	}
	got := decodeReserve(t, raw)
	if got.Act != "taken_over" || got.Copy != "this" || got.Replaced == nil || got.Replaced.Copy != "another" ||
		got.Replaced.Holder != "octo" || got.Replaced.Commit == "" || got.Replaced.EffectiveExpiry == "" {
		t.Fatalf("payload %+v; want a take-over replacing octo's other copy", got)
	}
	for _, dir := range []string{laptop, cloud} {
		key, err := os.ReadFile(filepath.Join(dir, ".abcd", ".work.local", "reservation", "copy-key"))
		if err != nil {
			t.Fatal(err)
		}
		k := strings.TrimSpace(string(key))
		if strings.Contains(text, k) || strings.Contains(raw, k) {
			t.Fatalf("a render printed a copy's key %s", k)
		}
	}
}

// TestReserveRendersCarryNoHostOrHomePath is C28 at the front door: the text
// and --json renders of an act, and a refusal quoting git's own reason about a
// remote under HOME, carry no home path and no host name.
func TestReserveRendersCarryNoHostOrHomePath(t *testing.T) {
	w := newReserveWorld(t)
	p := w.clone(t, "p", octoMail)
	host, _ := os.Hostname()

	check := func(label, out string) {
		t.Helper()
		noHomePath(t, w.home, out)
		if len(host) >= 4 && strings.Contains(out, host) {
			t.Fatalf("%s carries the host name %q:\n%s", label, host, out)
		}
	}
	_, text, stderr := reserveRun(t, p, "reserve", "iss-5", "--footprint", "a.go")
	check("the text render", text+stderr)
	_, raw, stderr := reserveRun(t, p, "reserve", "iss-5", "--footprint", "a.go", "--json")
	check("the --json render", raw+stderr)

	// origin moved away: git's reason names the path it could not read.
	if err := os.Rename(w.origin, w.origin+".gone"); err != nil {
		t.Fatal(err)
	}
	code, stdout, stderr := reserveRun(t, p, "reserve", "iss-6", "--footprint", "a.go")
	if code != 2 || !strings.Contains(stderr, "could not read the reservations from origin") || !strings.Contains(stderr, "no reservation was made") {
		t.Fatalf("a missing origin: exit %d, stderr %q; want exit 2 naming the failed read", code, stderr)
	}
	check("the refusal", stdout+stderr)
	code, stdout, _ = reserveRun(t, p, "reserve", "iss-6", "--footprint", "a.go", "--json")
	if code != 2 {
		t.Fatalf("a missing origin under --json: exit %d", code)
	}
	check("the --json refusal", stdout)
}
