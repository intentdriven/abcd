package guard

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/intentdriven/abcd/internal/abcdhome"
)

const testRootSHA = "0123456789abcdef0123456789abcdef01234567"

// TestAllowIsTakenOnceForTheExactLine pins the one-shot allow: it lets the
// exact line through once, within its window, and never a different line or
// a second time, and the log records both acts with no name in them.
func TestAllowIsTakenOnceForTheExactLine(t *testing.T) {
	home := t.TempDir()
	now := time.Date(2026, 10, 29, 12, 0, 0, 0, time.UTC)
	const line = "git push origin --delete feat/x"
	if _, err := GrantAllow(home, testRootSHA, line, now); err != nil {
		t.Fatal(err)
	}
	if _, ok := TakeAllow(home, testRootSHA, line+" ", now); ok {
		t.Error("a line that differs by a space was let through")
	}
	if _, ok := TakeAllow(home, "fedcba9876543210fedcba9876543210fedcba98", line, now); ok {
		t.Error("another repository's line was let through")
	}
	if _, ok := TakeAllow(home, testRootSHA, line, now.Add(time.Minute)); !ok {
		t.Fatal("the allowed line was not let through")
	}
	if _, ok := TakeAllow(home, testRootSHA, line, now.Add(2*time.Minute)); ok {
		t.Error("the allow was used twice")
	}
	log, err := os.ReadFile(abcdhome.Path(home, "guard-allow", testRootSHA, "log.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimSpace(string(log)), "\n")
	if len(lines) != 2 || !strings.Contains(lines[0], `"allowed"`) || !strings.Contains(lines[1], `"used"`) ||
		!strings.Contains(lines[1], `"used_at"`) || !strings.Contains(lines[1], "feat/x") {
		t.Errorf("log = %q, want the grant and the use of the line", lines)
	}
}

// TestAllowLapsesAndIsReadOnlyAsThePersonsOwn pins the record's guards: one
// past its window is not used, and one another account could have written (a
// file or a directory others can write) is never read.
func TestAllowLapsesAndIsReadOnlyAsThePersonsOwn(t *testing.T) {
	now := time.Date(2026, 10, 29, 12, 0, 0, 0, time.UTC)
	const line = "git branch -D feat/x"

	home := t.TempDir()
	if _, err := GrantAllow(home, testRootSHA, line, now); err != nil {
		t.Fatal(err)
	}
	if _, ok := TakeAllow(home, testRootSHA, line, now.Add(AllowWindow+time.Second)); ok {
		t.Error("an allow past its window was used")
	}

	for name, chmod := range map[string]func(home string){
		"file": func(home string) {
			_ = os.Chmod(filepath.Join(abcdhome.Path(home, "guard-allow", testRootSHA), allowName(line)), 0o666)
		},
		"directory": func(home string) { _ = os.Chmod(abcdhome.Path(home, "guard-allow", testRootSHA), 0o777) },
	} {
		home := t.TempDir()
		if _, err := GrantAllow(home, testRootSHA, line, now); err != nil {
			t.Fatal(err)
		}
		chmod(home)
		if _, ok := TakeAllow(home, testRootSHA, line, now); ok {
			t.Errorf("%s others can write: the allow was used", name)
		}
	}
}

// TestGuardRefusesAnAgentGrantingItsOwnAllow pins the person-only verb and
// the store's protection: running the verb, in either spelling, and writing
// into the store from the shell are refused; a line that only mentions the
// store, or removes from it, is not.
func TestGuardRefusesAnAgentGrantingItsOwnAllow(t *testing.T) {
	home := t.TempDir()
	dir := t.TempDir()
	r, err := Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	f := NewFiles(dir)
	f.home = home
	r = r.ReadingFrom(f)
	store := abcdhome.Path(home, "guard-allow", testRootSHA)
	for line, id := range map[string]string{
		"abcd guard allow --once --command 'git branch -D x'":              "abcd-guard-allow",
		"go run ./cmd/abcd guard allow --once --command 'git branch -D x'": "abcd-guard-allow",
		"echo '{}' > " + store + "/allow-x.json":                           allowStoreEntryID,
		"printf x | tee " + store + "/allow-x.json":                        allowStoreEntryID,
		"cp /tmp/a.json " + store + "/allow-x.json":                        allowStoreEntryID,
		"cat a > ~/.abcd.noindex/guard-allow/" + testRootSHA + "/x.json":   allowStoreEntryID,
		`d=~/.abcd.noindex/guard-allow; echo x > "$d/$f"`:                  allowStoreEntryID,
	} {
		d, err := r.Check(line)
		if err != nil {
			t.Fatalf("%q: %v", line, err)
		}
		if d.Verdict != VerdictBlock || d.EntryID != id {
			t.Errorf("%q: verdict %q under %q, want a block under %q", line, d.Verdict, d.EntryID, id)
		}
	}
	for _, line := range []string{
		"ls " + store,
		"cat " + store + "/log.jsonl",
		"rm " + store + "/allow-x.json",
		"echo guard-allow > notes.txt",
	} {
		if d, err := r.Check(line); err != nil || d.Verdict == VerdictBlock {
			t.Errorf("%q: blocked under %q (%v)", line, d.EntryID, err)
		}
	}
}

// TestADeleteBlockIsAllowableOnceOnlyWhenItStandsAlone pins the mark the hook
// reads: a block whose only cause is the unproven delete may be allowed once;
// one with any other blocker beside it may not.
func TestADeleteBlockIsAllowableOnceOnlyWhenItStandsAlone(t *testing.T) {
	r := deleteRegistry(t, t.TempDir(), nil)
	d, err := r.Check("git branch -D feat/x")
	if err != nil {
		t.Fatal(err)
	}
	if d.Verdict != VerdictBlock || !d.AllowableOnce {
		t.Errorf("an unproven delete alone: verdict %q, allowable %v, want an allowable block", d.Verdict, d.AllowableOnce)
	}
	d, err = r.Check("git branch -D feat/x && abcd library ledger --flip 3")
	if err != nil {
		t.Fatal(err)
	}
	if d.Verdict != VerdictBlock || d.AllowableOnce {
		t.Errorf("an unproven delete beside a blocker: verdict %q, allowable %v, want a block no allow passes", d.Verdict, d.AllowableOnce)
	}
}
