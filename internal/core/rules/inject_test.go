package rules

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/intentdriven/abcd/internal/abcdhome"
)

// ledgerLeaf is the file name a session's ledger is kept under.
func ledgerLeaf(session string) string {
	sum := sha256.Sum256([]byte(session))
	return hex.EncodeToString(sum[:]) + ".json"
}

// TestStateLivesInTheHomeStore: with no override, a session's ledger is kept
// in the account's own home store, owner-only, and nothing is written under
// the shared temporary folder.
func TestStateLivesInTheHomeStore(t *testing.T) {
	home := t.TempDir()
	tmp := t.TempDir()
	t.Setenv("ABCD_RULES_STATE_DIR", "")
	t.Setenv("HOME", home)
	t.Setenv("XDG_CACHE_HOME", "")
	t.Setenv("TMPDIR", tmp)
	if err := SaveState("s1", SessionState{Count: 3, Ledger: map[string]string{"COMMITTING": "h"}}); err != nil {
		t.Fatal(err)
	}
	if got := LoadState("s1"); got.Count != 3 || got.Ledger["COMMITTING"] != "h" {
		t.Fatalf("LoadState after SaveState = %+v, want the saved ledger", got)
	}
	dir := filepath.Join(home, filepath.FromSlash(abcdhome.Rel("rules-state")))
	fi, err := os.Stat(filepath.Join(dir, ledgerLeaf("s1")))
	if err != nil {
		t.Fatalf("the ledger is not in the home store: %v", err)
	}
	if perm := fi.Mode().Perm(); perm != 0o600 {
		t.Errorf("ledger mode = %04o, want 0600", perm)
	}
	if di, err := os.Stat(dir); err != nil || di.Mode().Perm() != 0o700 {
		t.Errorf("ledger folder = %v, %v; want mode 0700", di, err)
	}
	if entries, _ := os.ReadDir(tmp); len(entries) != 0 {
		t.Errorf("the shared temporary folder holds %d entries, want none", len(entries))
	}
}

// TestStateWithNoHomeIsKeptNowhere: with no home folder there is no private
// place to keep a ledger, so none is kept, and one planted at a predictable
// name under the shared temporary folder is never read: a ledger another
// account wrote could otherwise silence every rule this session matches.
func TestStateWithNoHomeIsKeptNowhere(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("ABCD_RULES_STATE_DIR", "")
	t.Setenv("HOME", "")
	t.Setenv("XDG_CACHE_HOME", "")
	t.Setenv("TMPDIR", tmp)
	planted := filepath.Join(tmp, fmt.Sprintf("abcd-rules-state-%d", os.Getuid()))
	if err := os.MkdirAll(planted, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(planted, ledgerLeaf("s1")), []byte(`{"count":9,"ledger":{"COMMITTING":"h"}}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if got := LoadState("s1"); got.Count != 0 || len(got.Ledger) != 0 {
		t.Fatalf("LoadState read a planted ledger: %+v", got)
	}
	if err := SaveState("s2", SessionState{Count: 1}); !errors.Is(err, ErrNoStateHome) {
		t.Fatalf("SaveState with no home must keep nothing and say why: got %v, want ErrNoStateHome", err)
	}
	if _, err := os.Stat(filepath.Join(planted, ledgerLeaf("s2"))); !os.IsNotExist(err) {
		t.Errorf("SaveState with no home wrote under the shared temporary folder: %v", err)
	}
	if entries, _ := os.ReadDir(tmp); len(entries) != 1 {
		t.Errorf("the shared temporary folder holds %d entries, want only the planted one", len(entries))
	}
}

// TestStateIgnoresALedgerSomeoneElseCouldHaveWritten: in the home store, a
// ledger file others can write, one reached through a link, or one in a
// folder others can write is not this account's word, so it is not read and
// the session starts fresh. A ledger only this account can write is read.
func TestStateIgnoresALedgerSomeoneElseCouldHaveWritten(t *testing.T) {
	home := t.TempDir()
	t.Setenv("ABCD_RULES_STATE_DIR", "")
	t.Setenv("HOME", home)
	t.Setenv("XDG_CACHE_HOME", "")
	t.Setenv("TMPDIR", t.TempDir())
	dir := filepath.Join(home, filepath.FromSlash(abcdhome.Rel("rules-state")))
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(filepath.Dir(dir), 0o700); err != nil {
		t.Fatal(err)
	}
	file := filepath.Join(dir, ledgerLeaf("s1"))
	body := []byte(`{"count":7,"ledger":{"COMMITTING":"h"}}`)
	fresh := func(what string) {
		t.Helper()
		if got := LoadState("s1"); got.Count != 0 || len(got.Ledger) != 0 {
			t.Errorf("%s: LoadState read it: %+v", what, got)
		}
	}
	if err := os.WriteFile(file, body, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(file, 0o666); err != nil {
		t.Fatal(err)
	}
	fresh("a ledger every account can write")
	if err := os.Remove(file); err != nil {
		t.Fatal(err)
	}
	elsewhere := filepath.Join(t.TempDir(), "x.json")
	if err := os.WriteFile(elsewhere, body, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(elsewhere, file); err != nil {
		t.Fatal(err)
	}
	fresh("a ledger reached through a link")
	if err := os.Remove(file); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(file, body, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(dir, 0o777); err != nil {
		t.Fatal(err)
	}
	fresh("a ledger in a folder every account can write")
	if err := os.Chmod(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	if got := LoadState("s1"); got.Count != 7 {
		t.Errorf("a ledger only this account can write: LoadState = %+v, want count 7", got)
	}
}

// TestStateRefusesAFolderSomeoneElseCouldWrite: a save or a reset in a ledger
// folder another account could write fails out loud, rather than writing a
// ledger every later read ignores or reporting a reset it did not make. Once
// the folder is the account's alone, both succeed; with no home, a reset has
// nothing to clear and is not an error.
func TestStateRefusesAFolderSomeoneElseCouldWrite(t *testing.T) {
	home := t.TempDir()
	t.Setenv("ABCD_RULES_STATE_DIR", "")
	t.Setenv("HOME", home)
	t.Setenv("XDG_CACHE_HOME", "")
	if err := ResetState("s1"); err != nil {
		t.Fatalf("ResetState with no ledger folder yet: %v", err)
	}
	if err := SaveState("s1", SessionState{Count: 1}); err != nil {
		t.Fatal(err)
	}
	dir := filepath.Join(home, filepath.FromSlash(abcdhome.Rel("rules-state")))
	if err := os.Chmod(dir, 0o777); err != nil {
		t.Fatal(err)
	}
	if err := SaveState("s1", SessionState{Count: 2}); err == nil {
		t.Error("SaveState wrote into a folder every account can write")
	}
	if err := ResetState("s1"); err == nil {
		t.Error("ResetState reported a reset in a folder every account can write")
	}
	if err := os.Chmod(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := SaveState("s1", SessionState{Count: 3}); err != nil {
		t.Errorf("SaveState in the account's own folder: %v", err)
	}
	if err := ResetState("s1"); err != nil {
		t.Errorf("ResetState in the account's own folder: %v", err)
	}
	t.Setenv("HOME", "")
	if err := ResetState("s1"); err != nil {
		t.Errorf("ResetState with no home: %v", err)
	}
}

func TestInjectFirstTurnRenders(t *testing.T) {
	rs := Defaults()
	res := Inject(rs, "commit and push", SessionState{}, 0)
	if res.Text == "" || !strings.Contains(res.Text, "COMMITTING") {
		t.Fatalf("first turn did not inject COMMITTING:\n%s", res.Text)
	}
	if len(res.Injected) == 0 {
		t.Fatal("Injected list empty on a match")
	}
	if res.State.Count != 1 {
		t.Fatalf("count = %d, want 1", res.State.Count)
	}
}

func TestInjectDedupsWithinSession(t *testing.T) {
	rs := Defaults()
	first := Inject(rs, "commit and push", SessionState{}, 0)
	second := Inject(rs, "commit and push again", first.State, 0)
	if second.Text != "" {
		t.Fatalf("second identical-domain turn re-injected:\n%s", second.Text)
	}
	if len(second.Injected) != 0 {
		t.Fatalf("dedup failed: %v", second.Injected)
	}
}

func TestInjectReinjectsOnClearedLedger(t *testing.T) {
	rs := Defaults()
	first := Inject(rs, "commit", SessionState{}, 0)
	if first.Text == "" {
		t.Fatal("first turn injected nothing")
	}
	// Same session, threaded state: dedup suppresses.
	second := Inject(rs, "commit", first.State, 0)
	if second.Text != "" {
		t.Fatalf("dedup failed on threaded state:\n%s", second.Text)
	}
	// A cleared ledger (what a reset produces) re-injects, even though the prompt
	// is unchanged — this is the reset/refresh contract at the unit level.
	third := Inject(rs, "commit", SessionState{}, 0)
	if third.Text == "" {
		t.Fatal("cleared ledger did not re-inject")
	}
}

func TestLoadBackstop(t *testing.T) {
	// Absent config -> default.
	if got := LoadBackstop(t.TempDir()); got != DefaultRefreshBackstop {
		t.Fatalf("absent config: backstop = %d, want %d", got, DefaultRefreshBackstop)
	}
	// A valid config value is honoured.
	dir := t.TempDir()
	writeConfig(t, dir, `{"rules":{"force_refresh_every_n":7}}`)
	if got := LoadBackstop(dir); got != 7 {
		t.Fatalf("config value not read: %d", got)
	}
	// A non-positive or malformed value falls back to the default.
	bad := t.TempDir()
	writeConfig(t, bad, `{"rules":{"force_refresh_every_n":0}}`)
	if got := LoadBackstop(bad); got != DefaultRefreshBackstop {
		t.Fatalf("non-positive value not defaulted: %d", got)
	}
}

func TestPruneStateRemovesStaleOnly(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("ABCD_RULES_STATE_DIR", dir)
	if err := SaveState("fresh", SessionState{Count: 1}); err != nil {
		t.Fatal(err)
	}
	if err := SaveState("old", SessionState{Count: 1}); err != nil {
		t.Fatal(err)
	}
	// Age the "old" session's file well past the TTL.
	oldFile := filepath.Join(dir, ledgerLeaf("old"))
	past := time.Now().Add(-StateTTL - time.Hour)
	if err := os.Chtimes(oldFile, past, past); err != nil {
		t.Fatal(err)
	}
	PruneState(StateTTL)
	if _, err := os.Stat(oldFile); !os.IsNotExist(err) {
		t.Fatal("stale state file survived prune")
	}
	if LoadState("fresh").Count != 1 {
		t.Fatal("fresh state file was pruned")
	}
}

func writeConfig(t *testing.T, dir, body string) {
	t.Helper()
	abcd := filepath.Join(dir, ".abcd")
	if err := os.MkdirAll(abcd, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(abcd, "config.json"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestInjectContentDriftReinjects(t *testing.T) {
	rs := Defaults()
	first := Inject(rs, "commit", SessionState{}, 0)
	// Mutate COMMITTING's rules; the signature changes, so it must re-inject
	// despite the ledger entry.
	d := rs.Domains["COMMITTING"]
	d.Rules = append([]string{"a brand new rule"}, d.Rules...)
	rs.Domains["COMMITTING"] = d
	second := Inject(rs, "commit", first.State, 0)
	if !strings.Contains(second.Text, "a brand new rule") {
		t.Fatalf("content drift did not re-inject:\n%s", second.Text)
	}
}

func TestInjectBackstopForcesRefresh(t *testing.T) {
	rs := Defaults()
	st := SessionState{}
	var lastText string
	// Backstop of 3: turns 1..2 dedup after the first inject; turn 3 (count%3==0)
	// clears the ledger and re-injects.
	for i := 0; i < 3; i++ {
		res := Inject(rs, "commit", st, 3)
		st = res.State
		lastText = res.Text
	}
	if lastText == "" {
		t.Fatal("backstop turn did not force a re-inject")
	}
}

func TestInjectNoMatchZeroBytes(t *testing.T) {
	rs := Defaults()
	res := Inject(rs, "paint a landscape in oils", SessionState{}, 0)
	if res.Text != "" {
		t.Fatalf("no-match must render zero bytes, got %q", res.Text)
	}
}

func TestInjectNeverReflectsPromptBytes(t *testing.T) {
	rs := Defaults()
	marker := "ZZUNIQUEPROMPTMARKERZZ commit"
	res := Inject(rs, marker, SessionState{}, 0)
	if strings.Contains(res.Text, "ZZUNIQUEPROMPTMARKERZZ") {
		t.Fatalf("injected text reflected prompt bytes:\n%s", res.Text)
	}
}

func TestSessionStateRoundTripAndReset(t *testing.T) {
	t.Setenv("ABCD_RULES_STATE_DIR", t.TempDir())
	const sid = "session-abc/../../etc" // traversal attempt; must be neutralised
	if got := LoadState(sid); got.Count != 0 {
		t.Fatal("fresh session should load zero state")
	}
	want := SessionState{Count: 4, Ledger: map[string]string{"COMMITTING": "deadbeef"}}
	if err := SaveState(sid, want); err != nil {
		t.Fatalf("SaveState: %v", err)
	}
	got := LoadState(sid)
	if got.Count != 4 || got.Ledger["COMMITTING"] != "deadbeef" {
		t.Fatalf("round-trip mismatch: %+v", got)
	}
	if err := ResetState(sid); err != nil {
		t.Fatalf("ResetState: %v", err)
	}
	if LoadState(sid).Count != 0 {
		t.Fatal("state survived reset")
	}
	// A second reset on an absent file is not an error.
	if err := ResetState(sid); err != nil {
		t.Fatalf("reset of absent state errored: %v", err)
	}
}

func TestSessionFileStaysInStateDir(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("ABCD_RULES_STATE_DIR", dir)
	// A traversal-flavoured id must still hash to a file directly inside dir.
	id := "../../../../etc/passwd"
	if err := SaveState(id, SessionState{Count: 1}); err != nil {
		t.Fatal(err)
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || entries[0].Name() != sessionLeaf(id) || !entries[0].Type().IsRegular() {
		t.Fatalf("session file escaped state dir: %v", entries)
	}
	if strings.ContainsAny(sessionLeaf(id), `/\`) || strings.Contains(sessionLeaf(id), "..") {
		t.Fatalf("session file name carries a path: %s", sessionLeaf(id))
	}
}
