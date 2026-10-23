package cli

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/intentdriven/abcd/internal/abcdhome"
	"github.com/intentdriven/abcd/internal/core/autonomy"
	"github.com/intentdriven/abcd/internal/gittest"
)

// awayCheckout is managedCheckout with a first commit, so the away store has a
// root commit to key on, and a local user.name, so the record names who set
// it.
func awayCheckout(t *testing.T) string {
	t.Helper()
	root := managedCheckout(t)
	gitCommitAt(t, root, "seed")
	gitAt(t, root, "config", "--local", "user.name", "Alice")
	return root
}

func gitAt(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
	cmd.Env = gittest.Env(t)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %v (%s)", args, err, out)
	}
	return strings.TrimSpace(string(out))
}

func awayStore(t *testing.T, root string) autonomy.Store {
	t.Helper()
	_, st, err := autonomy.Resolve(root, autonomy.Options{Home: os.Getenv("HOME")})
	if err != nil {
		t.Fatal(err)
	}
	return st
}

func readAway(t *testing.T, root string) *autonomy.Record {
	t.Helper()
	rec, err := awayStore(t, root).Read()
	if err != nil {
		t.Fatal(err)
	}
	return rec
}

// TestAwayRecordSharedAcrossWorktrees: `abcd mode autonomous --until 09:00`
// writes one record in the away store that a sibling worktree reads, with who
// set it and from which copy.
func TestAwayRecordSharedAcrossWorktrees(t *testing.T) {
	root := awayCheckout(t)
	sibling := filepath.Join(t.TempDir(), "sibling")
	gitAt(t, root, "worktree", "add", "-q", "-b", "feat/sibling", sibling)
	sibling = realPath(t, sibling)

	stdout, stderr, err := runSplit(t, "", "mode", "autonomous", "--until", "09:00")
	if err != nil {
		t.Fatalf("mode autonomous: %v\n%s", err, stderr)
	}
	for _, want := range []string{"Away until", "Alice", "abcd mode back", "grants no rights"} {
		if !strings.Contains(stdout, want) {
			t.Errorf("the set's output misses %q:\n%s", want, stdout)
		}
	}

	t.Chdir(sibling)
	rec, holds, err := autonomy.AwayAt(sibling, autonomy.Options{})
	if err != nil || !holds {
		t.Fatalf("the sibling worktree does not read the record: holds=%v err=%v", holds, err)
	}
	if rec.SetBy != "Alice" || rec.SetFrom != root {
		t.Errorf("record names set_by=%q set_from=%q, want Alice from %s", rec.SetBy, rec.SetFrom, root)
	}
	if h, m := rec.Ends().Local().Hour(), rec.Ends().Local().Minute(); h != 9 || m != 0 {
		t.Errorf("--until 09:00 ends at %s", rec.Ends().Local())
	}
	if entries, _ := os.ReadDir(abcdhome.Path(os.Getenv("HOME"), autonomy.StoreDir)); len(entries) != 1 {
		t.Errorf("the away store holds %d repository folders, want one shared by both copies", len(entries))
	}

	// The JSON form carries the same record.
	t.Chdir(root)
	so, _, err := runSplit(t, "", "mode", "autonomous", "--until", "09:00", "--json")
	if err != nil {
		t.Fatal(err)
	}
	var got struct {
		Away    bool   `json:"away"`
		SetBy   string `json:"set_by"`
		SetFrom string `json:"set_from"`
		Until   string `json:"until"`
	}
	if err := json.Unmarshal([]byte(so), &got); err != nil {
		t.Fatalf("--json: %v\n%s", err, so)
	}
	if !got.Away || got.SetBy != "Alice" || got.Until == "" || got.SetFrom == "" {
		t.Errorf("--json = %+v", got)
	}
}

// TestUntilRefusedPastOrNoTierOrNoCheckout: a past --until, one past seven
// days, an unreadable one, a checkout without the local tier, and no checkout
// are each refused at exit 2 with nothing written.
func TestUntilRefusedPastOrNoTierOrNoCheckout(t *testing.T) {
	root := awayCheckout(t)
	home := os.Getenv("HOME")
	past := time.Now().Add(-time.Hour).Format(time.RFC3339)
	far := time.Now().Add(autonomy.MaxWindow + time.Hour).Format(time.RFC3339)
	for _, args := range [][]string{
		{"mode", "autonomous", "--until", past},
		{"mode", "autonomous", "--until", far},
		{"mode", "autonomous", "--until", "tomorrow"},
	} {
		out, err := runCLIErr(t, args...)
		if err == nil {
			t.Errorf("%v succeeded:\n%s", args, out)
			continue
		}
		if code := exitCodeOf(err); code != 2 {
			t.Errorf("%v: exit = %d, want 2 (%v)", args, code, err)
		}
		if !strings.Contains(err.Error(), "nothing written") {
			t.Errorf("%v: the refusal does not say nothing was written: %v", args, err)
		}
	}
	if err := os.RemoveAll(filepath.Join(root, ".abcd")); err != nil {
		t.Fatal(err)
	}
	if _, err := runCLIErr(t, "mode", "autonomous"); err == nil || exitCodeOf(err) != 2 || !strings.Contains(err.Error(), ".abcd/.work.local") {
		t.Errorf("no tier: err = %v, want exit 2 naming the tier", err)
	}
	t.Chdir(t.TempDir())
	if _, err := runCLIErr(t, "mode", "autonomous"); err == nil || exitCodeOf(err) != 2 {
		t.Errorf("no checkout: err = %v, want exit 2", err)
	}
	if _, err := os.Stat(abcdhome.Path(home, autonomy.StoreDir)); err == nil {
		t.Error("a refused set created the away store")
	}
}

// TestModeBackPrintsQueueAndStops: `abcd mode back` prints one plain line per
// queued step and per recorded stop, the cap label included, records who
// ended the mode and from which copy, and does the same on a record whose
// time has passed.
func TestModeBackPrintsQueueAndStops(t *testing.T) {
	root := awayCheckout(t)
	if _, err := runCLIErr(t, "mode", "autonomous"); err != nil {
		t.Fatal(err)
	}
	for _, step := range []string{"approve the embargo close", "push feat/x"} {
		if out, err := runCLIErr(t, "mode", "queue", step); err != nil {
			t.Fatalf("mode queue: %v\n%s", err, out)
		}
	}
	st := awayStore(t, root)
	stop := autonomy.Stop{Kind: autonomy.StopWorkOwed, Detail: "feat/x is behind main", PossiblyForcedByCap: true}
	for range 2 {
		if ok, err := st.RecordStop(stop, time.Now()); err != nil || !ok {
			t.Fatalf("RecordStop: %v %v", ok, err)
		}
	}
	stdout, stderr, err := runSplit(t, "", "mode", "back")
	if err != nil {
		t.Fatalf("mode back: %v\n%s", err, stderr)
	}
	lines := strings.Split(strings.TrimRight(stdout, "\n"), "\n")
	if len(lines) != 4 {
		t.Fatalf("mode back printed %d lines, want two queued, one stop and the ending:\n%s", len(lines), stdout)
	}
	for _, want := range []string{"queued: approve the embargo close", "queued: push feat/x", "possibly forced by the cap", "2 times", "Alice"} {
		if !strings.Contains(stdout, want) {
			t.Errorf("mode back misses %q:\n%s", want, stdout)
		}
	}
	rec := readAway(t, root)
	if rec.Holds(time.Now()) || rec.Closed == nil || rec.Closed.By != "Alice" || rec.Closed.From != root {
		t.Errorf("after mode back: closed=%+v holds=%v", rec.Closed, rec.Holds(time.Now()))
	}

	// A second `mode back` has nothing to end.
	if out := string(runCLI(t, "mode", "back")); !strings.Contains(out, "not in the away mode") {
		t.Errorf("a second mode back printed %q", out)
	}

	// A record whose time has passed prints the same and is closed.
	start := time.Now().Add(-3 * time.Hour).Truncate(time.Second)
	if _, err := st.Update(func(*autonomy.Record) (*autonomy.Record, error) {
		r := &autonomy.Record{SchemaVersion: autonomy.SchemaVersion, SetBy: "Alice", SetFrom: root,
			Start: start, Until: start.Add(time.Hour)}
		_ = r.AddStep("rotate the release key", root, start)
		return r, nil
	}); err != nil {
		t.Fatal(err)
	}
	out := string(runCLI(t, "mode", "back"))
	if !strings.Contains(out, "queued: rotate the release key") || !strings.Contains(out, "ended at") {
		t.Errorf("mode back on a lapsed record printed:\n%s", out)
	}
	if rec := readAway(t, root); rec.Closed == nil {
		t.Error("mode back left a lapsed record unclosed")
	}
}

// TestHoldPushesRecordedAtTheVerb: `--hold-pushes` is kept in the record and
// said in the set's output, beside the mode granting no rights.
func TestHoldPushesRecordedAtTheVerb(t *testing.T) {
	root := awayCheckout(t)
	stdout, _, err := runSplit(t, "", "mode", "autonomous", "--hold-pushes")
	if err != nil {
		t.Fatal(err)
	}
	if !readAway(t, root).HoldPushes {
		t.Error("--hold-pushes is not in the record")
	}
	if !strings.Contains(stdout, "pushes wait") || !strings.Contains(stdout, "grants no rights") {
		t.Errorf("the set's output does not say pushes wait and the mode grants no rights:\n%s", stdout)
	}
	so, _, err := runSplit(t, "", "mode", "autonomous", "--json")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(so, `"hold_pushes": false`) || readAway(t, root).HoldPushes {
		t.Errorf("setting again without --hold-pushes kept it:\n%s", so)
	}
}

// TestBadgeRefusedWhileAwayAtTheVerb: while away, `abcd mode product-thinker`
// and `abcd mode facilitator` are refused naming `abcd decide`, nothing is
// written, and `managed` is still set.
func TestBadgeRefusedWhileAwayAtTheVerb(t *testing.T) {
	root := awayCheckout(t)
	if _, err := runCLIErr(t, "mode", "autonomous"); err != nil {
		t.Fatal(err)
	}
	for _, st := range []string{"product-thinker", "facilitator"} {
		_, err := runCLIErr(t, "mode", st)
		if err == nil || exitCodeOf(err) != 2 || !strings.Contains(err.Error(), "abcd decide") {
			t.Errorf("mode %s while away: err = %v, want exit 2 naming `abcd decide`", st, err)
		}
		if storedMode(t, root) != "" {
			t.Errorf("a refused `mode %s` wrote the badge", st)
		}
	}
	if _, err := runCLIErr(t, "mode", "managed"); err != nil {
		t.Errorf("mode managed while away refused: %v", err)
	}
	runCLI(t, "mode", "back")
	if _, err := runCLIErr(t, "mode", "facilitator"); err != nil {
		t.Errorf("mode facilitator after mode back refused: %v", err)
	}
}

// TestModeQueueRefusals: a step is refused when no record holds, when it is
// empty, and when the record fails the store's checks, naming the file.
func TestModeQueueRefusals(t *testing.T) {
	root := awayCheckout(t)
	if _, err := runCLIErr(t, "mode", "queue", "push feat/x"); err == nil || exitCodeOf(err) != 2 || !strings.Contains(err.Error(), "not away") {
		t.Errorf("queue while present: err = %v, want exit 2 saying not away", err)
	}
	if _, err := runCLIErr(t, "mode", "autonomous"); err != nil {
		t.Fatal(err)
	}
	if _, err := runCLIErr(t, "mode", "queue", "   "); err == nil || exitCodeOf(err) != 2 {
		t.Errorf("empty step: err = %v, want exit 2", err)
	}
	if err := os.Chmod(awayStore(t, root).RecordPath(), 0o666); err != nil {
		t.Fatal(err)
	}
	_, err := runCLIErr(t, "mode", "queue", "push feat/x")
	if err == nil || exitCodeOf(err) != 2 || !strings.Contains(err.Error(), autonomy.RecordFile) {
		t.Errorf("queue on a record others can write: err = %v, want exit 2 naming %s", err, autonomy.RecordFile)
	}
}

// TestModeBackEndsAnUnreadableRecord: `abcd mode back` on a record that does
// not parse exits 0, says the record could not be read and was replaced, and
// the mode no longer holds; the --json envelope carries the notes.
func TestModeBackEndsAnUnreadableRecord(t *testing.T) {
	root := awayCheckout(t)
	setAway(t)
	if err := os.WriteFile(awayStore(t, root).RecordPath(), []byte("{not json"), 0o600); err != nil {
		t.Fatal(err)
	}
	stdout, stderr, err := runSplit(t, "", "mode", "back", "--json")
	if err != nil {
		t.Fatalf("mode back on an unreadable record: %v\n%s", err, stderr)
	}
	if !strings.Contains(stdout, `"notes"`) || !strings.Contains(stdout, "could not be read") {
		t.Errorf("the envelope lacks the notes:\n%s", stdout)
	}
	if rec := readAway(t, root); rec.Closed == nil || rec.Holds(time.Now()) {
		t.Errorf("after mode back the record holds: %+v", rec)
	}
}
