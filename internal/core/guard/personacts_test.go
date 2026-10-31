package guard

import (
	"strings"
	"testing"
)

// TestThePersonsActsAreRefusedUnderAnyProgram: the acts only the person
// takes (allowing one delete, flipping a source-ledger line, approving a
// head) are refused through the binary the registry names and under any
// other program name the guard can read: a cross-compiled build, the
// source-checkout wrapper, a copy renamed, a wrapper stepped through, a
// program a variable names. Every other spelling of those verbs is allowed.
func TestThePersonsActsAreRefusedUnderAnyProgram(t *testing.T) {
	reg := Defaults()
	for _, c := range []struct {
		cmd   string
		entry string
	}{
		{`abcd guard allow --once --command 'git branch -D feat/x'`, "abcd-guard-allow"},
		{`bin/abcd-darwin-arm64 guard allow --once --command 'git branch -D feat/x'`, guardAllowAnyEntryID},
		{`./bin/abcd-linux-amd64 guard allow --command 'git worktree remove ../wt' --once`, guardAllowAnyEntryID},
		{`scripts/abcd.sh guard allow --once --command 'git branch -D feat/x'`, guardAllowAnyEntryID},
		{`bash scripts/abcd.sh guard allow --once --command 'git branch -D feat/x'`, guardAllowAnyEntryID},
		{`env FOO=1 ./tool guard allow --once --command 'git branch -D feat/x'`, guardAllowAnyEntryID},
		{`cd cmd/abcd && go run . guard allow --once --command 'git branch -D feat/x'`, guardAllowAnyEntryID},
		{`"$B" guard allow --once --command 'git branch -D feat/x'`, guardAllowAnyEntryID},
		{`abcd library ledger --flip 3`, "abcd-source-ledger-flip"},
		{`bin/abcd-darwin-arm64 library ledger --flip 3`, ledgerFlipAnyEntryID},
		{`scripts/abcd.sh library ledger --corpus c --flip 3`, ledgerFlipAnyEntryID},
		{`bash scripts/abcd.sh library ledger --flip=3`, ledgerFlipAnyEntryID},
		{`bin/abcd-darwin-arm64 library ledger --flip="$n"`, ledgerFlipAnyEntryID},
		{`bin/abcd-darwin-arm64 library ledger --flip=$(next-line)`, ledgerFlipAnyEntryID},
		{`bin/abcd-darwin-arm64 source ledger --flip 3`, ledgerFlipRetiredAnyEntryID},
		{`scripts/abcd.sh source ledger --flip=3`, ledgerFlipRetiredAnyEntryID},
		{`nice -n 5 bin/abcd-darwin-arm64 library ledger --flip 3`, ledgerFlipAnyEntryID},
		{`nice -n 5 scripts/abcd.sh guard allow --once --command 'git branch -D feat/x'`, guardAllowAnyEntryID},
		{`bin/abcd-darwin-arm64 land --approve`, approveAnyEntryID},
		{`scripts/abcd.sh land feat/x --approve`, approveAnyEntryID},
	} {
		d, err := reg.Check(c.cmd)
		// A program a variable names is refused as unknown first.
		varProgram := strings.HasPrefix(c.cmd, `"$`) && d.EntryID == unknownProgramEntryID
		if err != nil || d.Verdict != VerdictBlock || (d.EntryID != c.entry && !varProgram) {
			t.Errorf("Check(%q) = %q via %q, %v; want a block via %q", c.cmd, d.Verdict, d.EntryID, err, c.entry)
		}
	}
	for _, cmd := range []string{
		`bin/abcd-darwin-arm64 guard check --command 'git status'`,
		`bin/abcd-darwin-arm64 guard`,
		`scripts/abcd.sh capture "an agent ran guard allow --once"`,
		`grep -rn 'guard allow' commands`,
		`bin/abcd-darwin-arm64 allow guard`,
		`bin/abcd-darwin-arm64 guard allow --command 'git branch -D feat/x'`,
		`git grep -n -e guard -e allow -- commands/`,
		`rg -e guard -e allow internal/`,
		`git log --grep guard --grep allow`,
		`gh search issues guard allow`,
		`bin/abcd-darwin-arm64 library ledger`,
		`scripts/abcd.sh source ledger --corpus c`,
		`bin/abcd-darwin-arm64 capture "library ledger --flip 3"`,
	} {
		if d, err := reg.Check(cmd); err != nil || d.Verdict == VerdictBlock {
			t.Errorf("Check(%q) = %+v, %v; want it allowed", cmd, d, err)
		}
	}
}

// TestThePersonsActsAreRefusedInEveryMode: facilitator.mode decides who
// merges, never who takes the person's acts, so each is refused whichever
// mode the repository declares, and each refuses under an id a repository's
// own guard file cannot claim.
func TestThePersonsActsAreRefusedInEveryMode(t *testing.T) {
	for _, human := range []bool{true, false} {
		reads := 0
		reg := Defaults().WithMergeMode(modeOf(human, &reads))
		for cmd, id := range map[string]string{
			`bin/abcd-darwin-arm64 guard allow --once --command 'git branch -D feat/x'`: guardAllowAnyEntryID,
			`bin/abcd-darwin-arm64 library ledger --flip 3`:                             ledgerFlipAnyEntryID,
			`bin/abcd-darwin-arm64 source ledger --flip 3`:                              ledgerFlipRetiredAnyEntryID,
			`bin/abcd-darwin-arm64 land --approve`:                                      approveAnyEntryID,
		} {
			if d, err := reg.Check(cmd); err != nil || d.Verdict != VerdictBlock || d.EntryID != id {
				t.Errorf("human %v: Check(%q) = %q via %q, %v; want a block via %q", human, cmd, d.Verdict, d.EntryID, err, id)
			}
		}
	}
	for _, a := range personActs {
		if !containsString(reservedEntryIDs, a.id) {
			t.Errorf("reservedEntryIDs lacks %q", a.id)
		}
	}
}

// TestEachPersonsActSaysWhatItsRegistryEntrySays: the refusal under another
// program name gives the reason and the successor the registry entry for the
// same act gives, so the two cannot drift apart.
func TestEachPersonsActSaysWhatItsRegistryEntrySays(t *testing.T) {
	reg := Defaults()
	for _, a := range personActs {
		e, ok := reg.Entries[a.entry]
		if !ok {
			t.Fatalf("%s names registry entry %q, which the defaults do not hold", a.id, a.entry)
		}
		if a.why != e.Why || a.successor != e.Successor {
			t.Errorf("%s: why/successor differ from %s's:\n got %q / %q\nwant %q / %q", a.id, a.entry, a.why, a.successor, e.Why, e.Successor)
		}
	}
}
