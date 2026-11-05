package guard

import (
	"github.com/intentdriven/abcd/internal/abcdhome"
	"github.com/intentdriven/abcd/internal/core/runstop"
	"os"
	"path/filepath"
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

// TestStopStorePathNamesWhatTheFileToolsMayNotWrite: the file tools are held to
// the same names a shell line is, wherever the shared folder stands.
func TestStopStorePathNamesWhatTheFileToolsMayNotWrite(t *testing.T) {
	for p, want := range map[string]bool{
		"/Users/Shared/abcd-stop/abcd-stop.json":        true,
		"/home/x/.abcd.noindex/stop.json":               true,
		"/home/x/.abcd.noindex/shared-folder.json":      true,
		"/home/x/.abcd.noindex/sub/../stop.json":        true,
		"/home/x/.abcd.noindex/STOP.JSON":               true,
		"/home/x/.ABCD.NOINDEX/Shared-Folder.json":      true,
		"/Users/Shared/ABCD-STOP/ABCD-STOP.JSON":        true,
		"/home/x/.abcd.noindex/\u017ftop.json":          true,
		"/home/x/.abcd.noindex/\u017fhared-folder.json": true,
		"/home/x/project/stop.json":                     false,
		"/home/x/.abcd.noindex/rules.json":              false,
		"/home/x/.abcd.noindex/worktrees/a/stop.json":   false,
	} {
		if got := StopStorePath(p); got != want {
			t.Errorf("StopStorePath(%q) = %v, want %v", p, got, want)
		}
	}
}

// TestTheStopIsThePersonsAct: setting or lifting a stop, and choosing where a
// stop is read, are the person's acts under any program name, and a shell write
// to the stop file or the setting is refused, a removal included, since removing
// a stop is lifting it (itd-2611041549471712). Naming the verbs in passing, or
// another implement verb, is not.
func TestTheStopIsThePersonsAct(t *testing.T) {
	files := NewFiles(t.TempDir())
	files.home = t.TempDir()
	reg := Defaults().ReadingFrom(files)
	for _, c := range []struct{ cmd, entry string }{
		{`abcd implement stop`, "abcd-implement-stop"},
		{`abcd implement stop --lift`, "abcd-implement-stop"},
		{`go run ./cmd/abcd implement stop --lift`, "abcd-implement-stop"},
		{`bin/abcd-darwin-arm64 implement stop --lift`, implementStopAnyEntryID},
		{`scripts/abcd.sh implement stop`, implementStopAnyEntryID},
		{`bash scripts/abcd.sh --json implement stop --lift`, implementStopAnyEntryID},
		{`abcd ahoy shared-folder /Users/Shared`, "abcd-ahoy-shared-folder"},
		{`scripts/abcd.sh ahoy shared-folder /Users/Shared --person someone`, sharedFolderAnyEntryID},
		{`nohup bin/abcd-darwin-arm64 implement --json stop --lift`, implementStopAnyEntryID},
		{`scripts/abcd.sh implement --lift=true stop`, implementStopAnyEntryID},
		{`bin/abcd-darwin-arm64 ahoy --person me shared-folder /x`, sharedFolderAnyEntryID},
		{`rm -r /Users/Shared/abcd-stop-501`, stopStoreEntryID},
		{`rm -r /Users/Shared/ABCD-STOP-501`, stopStoreEntryID},
		{`rm /Users/Shared/abcd-stop/abcd-stop.json`, stopStoreEntryID},
		{`rm -r /Users/Shared/abcd-stop`, stopStoreEntryID},
		{`cd ~/.abcd.noindex && mv stop.json x`, stopStoreEntryID},
		{`rm -f ~/.abcd.noindex/stop.json`, stopStoreEntryID},
		{`mv /Users/Shared/abcd-stop/abcd-stop.json /tmp/x`, stopStoreEntryID},
		{`echo '{}' > /Users/Shared/abcd-stop/abcd-stop.json`, stopStoreEntryID},
		{`echo '{}' > ~/.abcd.noindex/shared-folder.json`, stopStoreEntryID},
		{`rm "$F" # abcd-stop.json`, stopStoreEntryID},
		{`rm ~/.abcd.noindex/stop.*`, stopStoreEntryID},
		{`rm ~/.abcd.noindex/shared-*`, stopStoreEntryID},
		{`cd ~/.abcd.noindex && rm -- *.json`, stopStoreEntryID},
		{`rm -rf ~/.abcd.noindex`, stopStoreEntryID},
		{`rm -rf ~/.abcd.noindex/`, stopStoreEntryID},
		{`mv ~/.abcd.noindex ~/.old`, stopStoreEntryID},
		{`find ~/.abcd.noindex -name stop.json -delete`, stopStoreEntryID},
		{`mv /tmp/x ~/.abcd.noindex/stop.json`, stopStoreEntryID},
		{`cd "$D" && rm stop.json`, stopStoreEntryID},
		{`rm /Users/Shared/abcd-st*/abcd-stop.json`, stopStoreEntryID},
		{`find /Users/Shared -name 'abcd-stop*' -delete`, stopStoreEntryID},
		{`rm ~/.abcd.noindex/STOP.JSON`, stopStoreEntryID},
		{`echo x > ~/.ABCD.NOINDEX/SHARED-FOLDER.JSON`, stopStoreEntryID},
		{`rm /Users/Shared/ABCD-STOP/ABCD-STOP.JSON`, stopStoreEntryID},
		{`mv /Users/Shared/Abcd-Stop /tmp/x`, stopStoreEntryID},
		{`rm -rf ~/.ABCD.NOINDEX`, stopStoreEntryID},
		{`rm ~/.abcd.noindex/STOP.*`, stopStoreEntryID},
		{`rm ~/.ABCD.NOINDEX/Shared-*`, stopStoreEntryID},
		{"rm ~/.abcd.noindex/\u017ftop.json", stopStoreEntryID},
		{"echo x > ~/.abcd.noindex/\u017fhared-folder.json", stopStoreEntryID},
		{"rm /Users/Shared/abcd-\u017ftop/abcd-\u017ftop.json", stopStoreEntryID},
		{"rm ~/.abcd.noindex/\ufb06op.json", stopStoreEntryID},
	} {
		d, err := reg.Check(c.cmd)
		if err != nil || d.Verdict != VerdictBlock || d.EntryID != c.entry {
			t.Errorf("Check(%q) = %q via %q, %v; want a block via %q", c.cmd, d.Verdict, d.EntryID, err, c.entry)
		}
	}
	for _, cmd := range []string{
		`scripts/abcd.sh implement step --run run-1`,
		`scripts/abcd.sh implement resume`,
		`scripts/abcd.sh capture "an agent ran implement stop"`,
		`git log --grep implement --grep stop`,
		`cat /Users/Shared/abcd-stop/abcd-stop.json`,
		`rm build/stop.json`,
		`ls /Users/Shared/abcd-digest`,
		`cd "$D" && rm build/stop.json`,
		`rm -rf ~/.abcd.noindex/review-x/copy`,
		`rm ~/.abcd.noindex/review-*`,
		`rm ~/.abcd.noindex/review-x/*`,
		`mkdir -p ~/.abcd.noindex`,
		`printf x >> ~/.abcd.noindex/trusted-roots`,
		`find ~/.abcd.noindex/worktrees -name x -delete`,
	} {
		if d, err := reg.Check(cmd); err != nil || d.Verdict == VerdictBlock {
			t.Errorf("Check(%q) = %+v, %v; want it allowed", cmd, d, err)
		}
	}
}

// TestTheAllowStoreFoldsCaseToo: the sibling store's name is matched as a volume
// that folds case would resolve it.
func TestTheAllowStoreFoldsCaseToo(t *testing.T) {
	home := t.TempDir()
	dir := t.TempDir()
	r, err := Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	f := NewFiles(dir)
	f.home = home
	r = r.ReadingFrom(f)
	store := abcdhome.Path(home, "GUARD-ALLOW", testRootSHA)
	for _, line := range []string{
		"echo '{}' > " + store + "/allow-x.json",
		"cp /tmp/a.json " + store + "/allow-x.json",
		"cat a > ~/.ABCD.NOINDEX/Guard-Allow/x.json",
	} {
		d, err := r.Check(line)
		if err != nil || d.Verdict != VerdictBlock || d.EntryID != allowStoreEntryID {
			t.Errorf("Check(%q) = %q via %q, %v", line, d.Verdict, d.EntryID, err)
		}
	}
}

// TestARemovalThatCouldReachTheStopIsRefused: a glob removal, or a removal of the
// folder that holds it, is read against what stands there, whether or not the line
// names the stop. A removal elsewhere, or a glob that cannot match, is not refused.
func TestARemovalThatCouldReachTheStopIsRefused(t *testing.T) {
	folder := t.TempDir()
	if err := os.Mkdir(filepath.Join(folder, runstop.StopDirName+"-501"), 0o755); err != nil {
		t.Fatal(err)
	}
	elsewhere := t.TempDir()
	old := runstop.WellKnownFolder
	runstop.WellKnownFolder = func() string { return folder }
	t.Cleanup(func() { runstop.WellKnownFolder = old })
	files := NewFiles(t.TempDir())
	files.home = t.TempDir()
	reg := Defaults().ReadingFrom(files)
	for _, cmd := range []string{
		"rm -rf " + folder + "/abcd-s*",
		"rm -rf " + folder + "/*",
		"rm -rf " + folder + "/a*1",
		"find " + folder + " -name 'abcd-s*' -delete",
		"rm -rf " + folder,
	} {
		if d, err := reg.Check(cmd); err != nil || d.Verdict != VerdictBlock || d.EntryID != stopStoreEntryID {
			t.Errorf("Check(%q) = %q via %q, %v; want a block via %q", cmd, d.Verdict, d.EntryID, err, stopStoreEntryID)
		}
	}
	for _, cmd := range []string{
		"rm -rf " + elsewhere + "/*",
		"rm -rf " + folder + "/other*",
		"find " + elsewhere + " -name 'x*' -delete",
		"ls " + folder + "/*",
		`cd "$D" && rm -rf *`,
		`cd "$D" && rm *.json`,
	} {
		if d, err := reg.Check(cmd); err != nil || d.EntryID == stopStoreEntryID {
			t.Errorf("Check(%q) = %q via %q, %v; want it not refused for the stop", cmd, d.Verdict, d.EntryID, err)
		}
	}
}
