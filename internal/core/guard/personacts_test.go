package guard

import (
	"errors"
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

// TestNamingAReleaseTagIsThePersonsAct: `abcd update` with a release tag as its
// operand is refused through the binary the registry names and under any program
// whose name carries abcd's: the source-checkout wrapper, a cross-compiled build,
// a plugin root's copy, a program the line assigns, a wrapper or a launcher
// stepped through. The verb with no tag, its flags, its news sub-verb and the
// update of another program are left alone.
func TestNamingAReleaseTagIsThePersonsAct(t *testing.T) {
	const entry, anyProgram = "abcd-update-tag", "abcd-update-tag-any-program"
	reg := Defaults()
	for _, c := range []struct{ cmd, entry string }{
		{`abcd update v0.7.0`, entry},
		{`abcd update v0.7.0 --json`, entry},
		{`abcd update --json v0.7.0`, entry},
		{`abcd update --yes v0.7.0`, entry},
		{`abcd update --yes=true v0.7.0`, entry},
		{`abcd update --check=false v0.7.0`, entry},
		{`abcd update -- v0.7.0`, entry},
		{`abcd --json update v0.7.0`, entry},
		{`abcd update v0.7.0-rc.1`, entry},
		{`abcd update v1`, entry},
		{`abcd update 0.7.0`, entry},
		{`abcd update V0.7.0`, entry},
		{`abcd update "v0.7.0"`, entry},
		{`abcd update v0.7.'0'`, entry},
		{`abcd update $'v0.7.0'`, entry},
		{`abcd update v0.{7,8}.0`, entry},
		{`abcd update v0.?.0`, entry},
		{`abcd update "$TAG"`, entry},
		{`abcd update v"$N"`, entry},
		{`abcd update "v0.${N}.0"`, entry},
		{`abcd update $(cat tag.txt)`, entry},
		{`T=v0.7.0; abcd update "$T"`, entry},
		{`abcd "$V" v0.7.0`, entry},
		{`abcd $(echo update) v0.7.0`, entry},
		{"abcd update \\\n  v0.7.0", entry},
		{`npx abcd update v0.7.0`, entry},
		{`f() { abcd update "$1"; }; f v0.7.0`, entry},
		{`/usr/local/bin/abcd update v0.7.0`, entry},
		{`"${CLAUDE_PLUGIN_ROOT}/abcd" update v0.7.0 --json`, entry},
		{`~/.claude/plugins/cache/abcd/abcd/0.31.0/abcd update v0.7.0`, entry},
		{`ABCD update v0.7.0`, entry},
		{`go run ./cmd/abcd update v0.7.0`, entry},
		{`go run -C /src/abcd ./cmd/abcd update v0.7.0 --yes`, entry},
		{`go run github.com/intentdriven/abcd/cmd/abcd@v0.15.1 update v0.7.0`, entry},
		{`env FOO=1 abcd update v0.7.0`, entry},
		{`FOO=1 abcd update v0.7.0`, entry},
		{`nohup abcd update v0.7.0`, entry},
		{`timeout 60 abcd update v0.7.0`, entry},
		{`nice -n 5 abcd update v0.7.0`, entry},
		{`command abcd update v0.7.0`, entry},
		{`sudo abcd update v0.7.0`, entry},
		{`xargs abcd update v0.7.0`, entry},
		{`sh -c 'abcd update v0.7.0'`, entry},
		{`bash -lc "cd x && abcd update v0.7.0"`, entry},
		{`eval 'abcd update v0.7.0'`, entry},
		{`(abcd update v0.7.0)`, entry},
		{`echo x && abcd update v0.7.0 | tail -3`, entry},

		{`scripts/abcd.sh update v0.7.0`, anyProgram},
		{`./scripts/abcd.sh update v0.7.0 --json`, anyProgram},
		{`scripts/abcd.sh --json update v0.7.0`, anyProgram},
		{`scripts/abcd.sh update --yes v0.7.0`, anyProgram},
		{`scripts/abcd.sh update --yes --json v0.7.0`, anyProgram},
		{`scripts/abcd.sh update --check=false v0.7.0`, anyProgram},
		{`scripts/abcd.sh update -- v0.7.0`, anyProgram},
		{`scripts/abcd.sh update 0.7.0`, anyProgram},
		{`scripts/abcd.sh update v0.?.0`, anyProgram},
		{`scripts/abcd.sh update "$TAG"`, anyProgram},
		{`scripts/abcd.sh update v"$N"`, anyProgram},
		{`scripts/abcd.sh update "$F" v0.7.0`, anyProgram},
		{`scripts/abcd.sh update --"$F" v0.7.0`, anyProgram},
		{`scripts/abcd.sh "$F" update v0.7.0`, anyProgram},
		{`scripts/abcd.sh "$V" v0.7.0`, anyProgram},
		{`scripts/abcd.sh up"$X" v0.7.0`, anyProgram},
		{`scripts/abcd.sh $(echo update) v0.7.0`, anyProgram},
		{`T=v0.7.0; scripts/abcd.sh update "$T"`, anyProgram},
		{`V=update; scripts/abcd.sh "$V" v0.7.0`, anyProgram},
		{`A='update v0.7.0'; scripts/abcd.sh $A`, anyProgram},
		{`bash scripts/abcd.sh update v0.7.0`, anyProgram},
		{`bash -- scripts/abcd.sh update v0.7.0`, anyProgram},
		{`bash -x scripts/abcd.sh update "$TAG"`, anyProgram},
		{`source scripts/abcd.sh update v0.7.0`, anyProgram},
		{`. scripts/abcd.sh update v0.7.0`, anyProgram},
		{`script -q /dev/null scripts/abcd.sh update v0.7.0`, anyProgram},
		{`caffeinate -i scripts/abcd.sh update v0.7.0`, anyProgram},
		{`arch -arm64 bin/abcd-darwin-arm64 update v0.7.0`, anyProgram},
		{`bin/abcd-darwin-arm64 update v0.7.0`, anyProgram},
		{`./bin/abcd-linux-amd64 update --yes v0.7.0`, anyProgram},
		{`nohup bin/abcd-darwin-arm64 update v0.7.0`, anyProgram},
		{`nice -n 5 scripts/abcd.sh update v0.7.0`, anyProgram},
		{`env FOO=1 scripts/abcd.sh update v0.7.0`, anyProgram},
		{`sh -c 'scripts/abcd.sh update v0.7.0'`, anyProgram},
		{`cd cmd/abcd && go run . update v0.7.0`, anyProgram},
		{`go run -C cmd/abcd . update v0.7.0`, anyProgram},
		{`"$B" update v0.7.0`, anyProgram},
		{`B=scripts/abcd.sh; "$B" update v0.7.0`, anyProgram},
	} {
		d, err := reg.Check(c.cmd)
		// A program a variable names is refused as unknown first.
		varProgram := strings.Contains(c.cmd, `"$B"`) && d.EntryID == unknownProgramEntryID
		if err != nil || d.Verdict != VerdictBlock || (d.EntryID != c.entry && !varProgram) {
			t.Errorf("Check(%q) = %q via %q, %v; want a block via %q", c.cmd, d.Verdict, d.EntryID, err, c.entry)
		}
	}
	for _, program := range []string{
		`abcd`, `scripts/abcd.sh`, `bash scripts/abcd.sh`, `go run ./cmd/abcd`, `bin/abcd-darwin-arm64`,
		`"${CLAUDE_PLUGIN_ROOT}/abcd"`, `nice -n 5 abcd`, `nohup scripts/abcd.sh`,
	} {
		for _, args := range []string{
			`update`,
			`update --yes`,
			`update --yes --json`,
			`update --json`,
			`--json update`,
			`update --check`,
			`update --check --json`,
			`update --help`,
			`update -h`,
			`update news`,
			`update news --json`,
			`update news --full`,
			`update news --seen`,
			`update news --answer yes`,
			`update news --answer not-now`,
			`update news --answer="$A"`,
			`update news --answer "$A"`,
			`update news v0.7.0`,
			`update latest`,
			`update ./notes.txt`,
			`help update`,
			`explain update`,
			`docs update v0.7.0`,
			`capture "an agent ran abcd update v0.7.0"`,
			`announce "$ID"`,
			`reflect write v0.7.0`,
		} {
			cmd := program + " " + args
			if d, err := reg.Check(cmd); err != nil || d.Verdict == VerdictBlock {
				t.Errorf("Check(%q) = %+v, %v; want it allowed", cmd, d, err)
			}
		}
	}
	for _, cmd := range []string{
		`npm update v1.2.3`,
		`brew update`,
		`git submodule update v1`,
		`go run . update v1.2.3`,
		`go run ./cmd/other update v1.2.3`,
		`./deploy.sh update v1.2.3`,
		`bash deploy.sh update v1.2.3`,
		`git commit -m 'abcd update v0.7.0'`,
		`grep -rn 'abcd update v0.7.0' commands`,
		`echo abcd update v0.7.0`,
		`cat scripts/abcd.sh update v0.7.0`,
		`git log --grep update v0.7.0`,
		`"$PM" update lodash`,
	} {
		if d, err := reg.Check(cmd); err != nil || d.Verdict == VerdictBlock {
			t.Errorf("Check(%q) = %+v, %v; want it allowed", cmd, d, err)
		}
	}
}

// TestNamingAReleaseTagIsRefusedInEveryMode: the mode decides who merges, never
// who names a release.
func TestNamingAReleaseTagIsRefusedInEveryMode(t *testing.T) {
	const anyProgram = "abcd-update-tag-any-program"
	for _, human := range []bool{true, false} {
		reads := 0
		reg := Defaults().WithMergeMode(modeOf(human, &reads))
		for cmd, id := range map[string]string{
			`abcd update v0.7.0`:            "abcd-update-tag",
			`scripts/abcd.sh update v0.7.0`: anyProgram,
		} {
			if d, err := reg.Check(cmd); err != nil || d.Verdict != VerdictBlock || d.EntryID != id {
				t.Errorf("human %v: Check(%q) = %q via %q, %v; want a block via %q", human, cmd, d.Verdict, d.EntryID, err, id)
			}
		}
	}
}

// homeDir and upperHomeDir spell abcd's home folder as a line does, the second as a
// volume that folds case resolves it.
var (
	homeDir      = abcdhome.Display()
	upperHomeDir = "~/" + strings.ToUpper(abcdhome.Rel())
)

// TestStopStorePathNamesWhatTheFileToolsMayNotWrite: the file tools are held to
// the same names a shell line is, wherever the shared folder stands.
func TestStopStorePathNamesWhatTheFileToolsMayNotWrite(t *testing.T) {
	for p, want := range map[string]bool{
		"/Users/Shared/abcd-stop/abcd-stop.json":                             true,
		abcdhome.Path("/home/x", "stop.json"):                                true,
		abcdhome.Path("/home/x", "shared-folder.json"):                       true,
		"/home/x/" + abcdhome.Rel("sub") + "/../stop.json":                   true,
		abcdhome.Path("/home/x", "STOP.JSON"):                                true,
		"/home/x/" + strings.ToUpper(abcdhome.Rel()) + "/Shared-Folder.json": true,
		"/Users/Shared/ABCD-STOP/ABCD-STOP.JSON":                             true,
		abcdhome.Path("/home/x", "\u017ftop.json"):                           true,
		abcdhome.Path("/home/x", "\u017fhared-folder.json"):                  true,
		"/home/x/project/stop.json":                                          false,
		abcdhome.Path("/home/x", "rules.json"):                               false,
		abcdhome.Path("/home/x", "worktrees/a/stop.json"):                    false,
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
		{`scripts/abcd.sh implement --lift="$V" stop`, implementStopAnyEntryID},
		{`scripts/abcd.sh ahoy --person="$USER" shared-folder /Users/Shared`, sharedFolderAnyEntryID},
		{`scripts/abcd.sh implement "$V" stop`, implementStopAnyEntryID},
		{`scripts/abcd.sh implement --lift"$V" stop`, implementStopAnyEntryID},
		{`V='=true'; scripts/abcd.sh implement --lift"$V" stop`, implementStopAnyEntryID},
		{`scripts/abcd.sh ahoy --person"$X" shared-folder /Users/Shared`, sharedFolderAnyEntryID},
		{`scripts/abcd.sh ahoy "$F" me shared-folder /Users/Shared`, sharedFolderAnyEntryID},
		{`F=--person; scripts/abcd.sh ahoy "$F" me shared-folder /Users/Shared`, sharedFolderAnyEntryID},
		{`scripts/abcd.sh implement "$V"`, implementStopAnyEntryID},
		{`V=stop; scripts/abcd.sh implement "$V"`, implementStopAnyEntryID},
		{`scripts/abcd.sh implement st"$X"`, implementStopAnyEntryID},
		{`scripts/abcd.sh implement $(echo stop)`, implementStopAnyEntryID},
		{`scripts/abcd.sh "$V" stop`, implementStopAnyEntryID},
		{`scripts/abcd.sh $(echo implement) stop`, implementStopAnyEntryID},
		{`A='implement stop'; scripts/abcd.sh $A`, implementStopAnyEntryID},
		{`scripts/abcd.sh implement $'stop'`, implementStopAnyEntryID},
		{`bash -- scripts/abcd.sh implement stop`, implementStopAnyEntryID},
		{`sh -- scripts/abcd.sh implement stop`, implementStopAnyEntryID},
		{`zsh -- scripts/abcd.sh implement stop`, implementStopAnyEntryID},
		{`dash -- scripts/abcd.sh implement stop`, implementStopAnyEntryID},
		{`ksh -- scripts/abcd.sh implement stop`, implementStopAnyEntryID},
		{`bash -- scripts/abcd.sh ahoy shared-folder /Users/Shared`, sharedFolderAnyEntryID},
		{`bash -- renamed implement stop`, implementStopAnyEntryID},
		{`bash -x scripts/abcd.sh implement "$V"`, implementStopAnyEntryID},
		{`sh -e scripts/abcd.sh implement "$V"`, implementStopAnyEntryID},
		{`bash -o errexit scripts/abcd.sh implement "$V"`, implementStopAnyEntryID},
		{`bash "$F" scripts/abcd.sh implement "$V"`, implementStopAnyEntryID},
		{`script -q /dev/null scripts/abcd.sh implement "$V"`, implementStopAnyEntryID},
		{`source scripts/abcd.sh implement "$V"`, implementStopAnyEntryID},
		{`. scripts/abcd.sh implement "$V"`, implementStopAnyEntryID},
		{`caffeinate scripts/abcd.sh implement "$V"`, implementStopAnyEntryID},
		{`caffeinate -i scripts/abcd.sh implement "$V"`, implementStopAnyEntryID},
		{`arch -arm64 scripts/abcd.sh implement "$V"`, implementStopAnyEntryID},
		{`unbuffer scripts/abcd.sh implement "$V"`, implementStopAnyEntryID},
		{`chronic scripts/abcd.sh implement "$V"`, implementStopAnyEntryID},
		{`fish scripts/abcd.sh implement "$V"`, implementStopAnyEntryID},
		{`rbash scripts/abcd.sh implement "$V"`, implementStopAnyEntryID},
		{`ash scripts/abcd.sh implement "$V"`, implementStopAnyEntryID},
		{`mksh scripts/abcd.sh implement "$V"`, implementStopAnyEntryID},
		{`csh scripts/abcd.sh implement "$V"`, implementStopAnyEntryID},
		{`tcsh scripts/abcd.sh implement "$V"`, implementStopAnyEntryID},
		{`BASH scripts/abcd.sh implement "$V"`, implementStopAnyEntryID},
		{`Bash -- scripts/abcd.sh implement stop`, implementStopAnyEntryID},
		{`CAFFEINATE scripts/abcd.sh implement "$V"`, implementStopAnyEntryID},
		{`/bin/bas? scripts/abcd.sh implement "$V"`, implementStopAnyEntryID},
		{`bash scripts/abcd.sh implement "$V"`, implementStopAnyEntryID},
		{`bin/abcd-darwin-arm64 implement "$V"`, implementStopAnyEntryID},
		{`scripts/abcd.sh ahoy "$V"`, sharedFolderAnyEntryID},
		{`bin/abcd-darwin-arm64 ahoy --person me shared-folder /x`, sharedFolderAnyEntryID},
		{`rm -r /Users/Shared/abcd-stop-501`, stopStoreEntryID},
		{`rm -r /Users/Shared/ABCD-STOP-501`, stopStoreEntryID},
		{`rm /Users/Shared/abcd-stop/abcd-stop.json`, stopStoreEntryID},
		{`rm -r /Users/Shared/abcd-stop`, stopStoreEntryID},
		{`cd ` + homeDir + ` && mv stop.json x`, stopStoreEntryID},
		{`rm -f ` + homeDir + `/stop.json`, stopStoreEntryID},
		{`mv /Users/Shared/abcd-stop/abcd-stop.json /tmp/x`, stopStoreEntryID},
		{`echo '{}' > /Users/Shared/abcd-stop/abcd-stop.json`, stopStoreEntryID},
		{`echo '{}' > ` + homeDir + `/shared-folder.json`, stopStoreEntryID},
		{`rm "$F" # abcd-stop.json`, stopStoreEntryID},
		{`rm ` + homeDir + `/stop.*`, stopStoreEntryID},
		{`rm ` + homeDir + `/shared-*`, stopStoreEntryID},
		{`cd ` + homeDir + ` && rm -- *.json`, stopStoreEntryID},
		{`rm -rf ` + homeDir, stopStoreEntryID},
		{`rm -rf ` + homeDir + `/`, stopStoreEntryID},
		{`mv ` + homeDir + ` ~/.old`, stopStoreEntryID},
		{`find ` + homeDir + ` -name stop.json -delete`, stopStoreEntryID},
		{`mv /tmp/x ` + homeDir + `/stop.json`, stopStoreEntryID},
		{`cd "$D" && rm stop.json`, stopStoreEntryID},
		{`rm /Users/Shared/abcd-st*/abcd-stop.json`, stopStoreEntryID},
		{`find /Users/Shared -name 'abcd-stop*' -delete`, stopStoreEntryID},
		{`rm ` + homeDir + `/STOP.JSON`, stopStoreEntryID},
		{`echo x > ` + upperHomeDir + `/SHARED-FOLDER.JSON`, stopStoreEntryID},
		{`rm /Users/Shared/ABCD-STOP/ABCD-STOP.JSON`, stopStoreEntryID},
		{`mv /Users/Shared/Abcd-Stop /tmp/x`, stopStoreEntryID},
		{`rm -rf ` + upperHomeDir, stopStoreEntryID},
		{`rm ` + homeDir + `/STOP.*`, stopStoreEntryID},
		{`rm ` + upperHomeDir + `/Shared-*`, stopStoreEntryID},
		{"rm " + homeDir + "/\u017ftop.json", stopStoreEntryID},
		{"echo x > " + homeDir + "/\u017fhared-folder.json", stopStoreEntryID},
		{"rm /Users/Shared/abcd-\u017ftop/abcd-\u017ftop.json", stopStoreEntryID},
		{"rm " + homeDir + "/\ufb06op.json", stopStoreEntryID},
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
		`grep implement "$F"`,
		`grep -r implement "$F" stop`,
		`grep abcd implement "$F"`,
		`printf 'see scripts/abcd.sh, then' "$A" "$B"`,
		`grep scripts/abcd.sh implement "$F"`,
		`bash hooks/bootstrap.sh`,
		`bash hooks/bootstrap.sh "$A" "$B"`,
		`bash -- hooks/bootstrap.sh "$A"`,
		`bash -x scripts/abcd.sh capture "$T"`,
		`bash -- scripts/abcd.sh implement step`,
		`script -q /dev/null scripts/abcd.sh capture "$T"`,
		`caffeinate make test`,
		`scripts/abcd.sh capture "$T"`,
		`scripts/abcd.sh --json capture "$T"`,
		`scripts/abcd.sh implement step --run "$R"`,
		`scripts/abcd.sh implement receipt "$P"`,
		`scripts/abcd.sh implement step`,
		`scripts/abcd.sh ahoy --json`,
		`scripts/abcd.sh ahoy worktree add "$N"`,
		`cat /Users/Shared/abcd-stop/abcd-stop.json`,
		`rm build/stop.json`,
		`ls /Users/Shared/abcd-digest`,
		`cd "$D" && rm build/stop.json`,
		`rm -rf ` + homeDir + `/review-x/copy`,
		`rm ` + homeDir + `/review-*`,
		`rm ` + homeDir + `/review-x/*`,
		`mkdir -p ` + homeDir,
		`printf x >> ` + homeDir + `/trusted-roots`,
		`find ` + homeDir + `/worktrees -name x -delete`,
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
		"cat a > " + upperHomeDir + "/Guard-Allow/x.json",
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

// TestAnActsProgramIsReadByItsBasename: what a program may be is read from its
// basename, as the registry reads it: one the line fixes as another program's,
// wherever it stands, is that program, and one the line leaves unfixed, or fixes
// as abcd's, is read as abcd for every act.
func TestAnActsProgramIsReadByItsBasename(t *testing.T) {
	reg := Defaults()
	for _, c := range []struct{ cmd, entry string }{
		{`"$DIR/abcd" update v0.7.0`, "abcd-update-tag"},
		{`"$DIR/abcd" update "$TAG"`, "abcd-update-tag"},
		{`"${X}abcd" update v0.7.0`, "abcd-update-tag"},
		{`"$DIR/bin/abcd-darwin-arm64" update "$TAG"`, updateTagAnyEntryID},
		{`"$DIR/scripts/abcd.sh" update v0.7.0`, updateTagAnyEntryID},
		{`"$DIR/${B}.sh" update v0.7.0`, updateTagAnyEntryID},
		{`"$DIR/$B" update v0.7.0`, unknownProgramEntryID},
		{`"$B" update v0.7.0`, unknownProgramEntryID},
		{`"$BIN/bash" scripts/abcd.sh update v0.7.0`, updateTagAnyEntryID},
		{`"$BIN/bash" "$DIR/abcd.sh" update v0.7.0`, updateTagAnyEntryID},

		{`"$DIR/abcd" implement "$V"`, "abcd-implement-stop"},
		{`"$DIR/abcd" "$V" stop`, "abcd-implement-stop"},
		{`"$DIR/scripts/abcd.sh" implement "$V"`, implementStopAnyEntryID},
		{`"$DIR/scripts/abcd.sh" "$V" stop`, implementStopAnyEntryID},
		{`"$DIR/bin/abcd-darwin-arm64" ahoy "$V"`, sharedFolderAnyEntryID},
		{`"$DIR/${B}.sh" implement "$V"`, implementStopAnyEntryID},
		{`"$DIR/${B}.sh" "$V" stop`, implementStopAnyEntryID},
		{`"$DIR/${B}.sh" ahoy "$V"`, sharedFolderAnyEntryID},
		{`"$B" implement "$V"`, unknownProgramEntryID},
		{`"$DIR/deploy.sh" implement stop`, implementStopAnyEntryID},
		{`"$DIR/deploy.sh" ahoy shared-folder /x`, sharedFolderAnyEntryID},
		{`"$DIR/deploy.sh" land --approve`, approveAnyEntryID},
		{`"$DIR/deploy.sh" library ledger --flip 3`, ledgerFlipAnyEntryID},
	} {
		d, err := reg.Check(c.cmd)
		if err != nil || d.Verdict != VerdictBlock || d.EntryID != c.entry {
			t.Errorf("Check(%q) = %q via %q, %v; want a block via %q", c.cmd, d.Verdict, d.EntryID, err, c.entry)
		}
	}
	for _, cmd := range []string{
		`"$SCRIPT_DIR/deploy.sh" update v1.2.3`,
		`$HOME/bin/tool update v2`,
		`sudo "$HOME/bin/tool" update v2.1.0`,
		`"$(pwd)/deploy.sh" update v1.2.3`,
		`"$VENV/bin/python" -m tool update v2`,
		`"$DIR/deploy.sh" update "$TAG"`,
		`"$BIN/bash" deploy.sh update v1.2.3`,
		`sh -c "$DIR/deploy.sh update v1.2.3"`,
		`"$DIR/deploy.sh" implement "$V"`,
		`"$DIR/deploy.sh" "$V" stop`,
		`"$DIR/deploy.sh" ahoy "$V"`,
		`"$BIN/grep" implement "$F"`,
	} {
		if d, err := reg.Check(cmd); err != nil || d.Verdict == VerdictBlock {
			t.Errorf("Check(%q) = %q via %q, %v; want it not refused", cmd, d.Verdict, d.EntryID, err)
		}
	}
}

// TestAProgramTheLineFixesIsNotReadAsAbcd: a variable that names the program, set
// on the line to fixed words, where every other command of the line is one that sets
// no variable (quietCommand), is the program those words name. Where the line does
// not fix it, the update with a tag stays refused, and the refusal says what stands
// in the name's place.
func TestAProgramTheLineFixesIsNotReadAsAbcd(t *testing.T) {
	reg := Defaults()
	for _, cmd := range []string{
		`PM=npm; "$PM" update "$PKG"`,
		`PM=npm; "$PM" update v1.2.3`,
		`PM=npm && $PM update "$PKG"`,
		"PM=npm\n${PM} update \"$PKG\"",
		`PKG=lodash; PM=npm; "$PM" update "$PKG"`,
		`PM=npm; BIN=/opt/bin; "$BIN/$PM" update v1.2.3`,
		`for PM in npm yarn; do "$PM" update "$PKG"; done`,
		`for PM in npm yarn; do "$PM" update v1.2.3; done`,
		`PM=npm; [ -f package.json ] && "$PM" update v1.2.3`,
		`PM=npm; [[ -f package.json ]] && "$PM" update v1.2.3`,
		`PM=npm; printf '%s\n' "$MSG"; "$PM" update v1.2.3`,
		`for PM in npm pnpm; do echo "$PM"; "$PM" update "$PKG"; done`,
		`PM=npm; cd web && echo "in $PWD"; true; : ok; "$PM" update v1.2.3`,
		`PM=npm; if [ -n "$CI" ]; then echo ci; else echo local; fi; "$PM" update v1.2.3`,
		`PM=npm; test -d node_modules || echo none; "$PM" update v1.2.3`,
	} {
		if d, err := reg.Check(cmd); err != nil || d.Verdict == VerdictBlock {
			t.Errorf("Check(%q) = %q via %q, %v; want it not refused", cmd, d.Verdict, d.EntryID, err)
		}
	}
	const entry = "abcd-update-tag"
	for _, c := range []struct{ cmd, entry string }{
		{`"$PM" update "$PKG"`, unknownProgramEntryID},
		{`"$PM" update v1.2.3`, unknownProgramEntryID},
		{`$(which npm) update "$PKG"`, unknownProgramEntryID},
		{`PM=abcd; "$PM" update "$PKG"`, entry},
		{`for PM in npm abcd; do "$PM" update "$PKG"; done`, entry},
		{`PM=npm; PM=abcd; "$PM" update v0.7.0`, entry},
		{`A=x; A=ab; for B in cd y; do "$A$B" update v0.7.0; done`, entry},
		{`A=x; B=cd; A=ab; B=y; B=cd; "$A$B" update v0.7.0`, entry},
		{`A=ab; A=x; B=y; B=cd; "$A$B" update v0.7.0`, entry},
		{`PM=npm; PM=npm printf -v PM ./bin/x; "$PM" update v0.7.0`, unknownProgramEntryID},
		{`PM=npm; PM=npm read "$X"; "$PM" update v0.7.0`, unknownProgramEntryID},
		{`PM=npm; print -v "$X" y; "$PM" update v1.2.3`, unknownProgramEntryID},
		{`PM=npm; set -A "$X" y; "$PM" update v1.2.3`, unknownProgramEntryID},
		{`PM=npm; vared "$X"; "$PM" update v1.2.3`, unknownProgramEntryID},
		{`PM=npm; zparseopts -A "$X" a; "$PM" update v1.2.3`, unknownProgramEntryID},
		{`A=xbcd; "${A/x/a}" update v0.7.0`, unknownProgramEntryID},
		{`A=xbcd; "${A//x/a}" update v0.7.0`, unknownProgramEntryID},
		{`A=xbcd; "${A/#x/a}" update v0.7.0`, unknownProgramEntryID},
		{`A=abcdX; "${A%X}" update v0.7.0`, unknownProgramEntryID},
		{`A=Xabcd; "${A#X}" update v0.7.0`, unknownProgramEntryID},
		{`A=abX; B=cd; "${A%X}$B" update v0.7.0`, unknownProgramEntryID},
		{`A=Xbcd; "a${A#X}" update v0.7.0`, unknownProgramEntryID},
		{`REPLY=npm; read; "$REPLY" update v0.7.0`, unknownProgramEntryID},
		{`REPLY=npm; select x in a; do "$REPLY" update v0.7.0; done`, unknownProgramEntryID},
		{`OPTARG=npm; getopts a: o; "$OPTARG" update v0.7.0`, unknownProgramEntryID},
		{`MAPFILE=npm; mapfile < list.txt; "$MAPFILE" update v0.7.0`, unknownProgramEntryID},
		{`PWD=npm; cd abcd; "$PWD" update v0.7.0`, unknownProgramEntryID},
		{`PM=npm; emulate -c "$X"; "$PM" update v0.7.0`, unknownProgramEntryID},
		{`PM=npm; trap "$X" DEBUG; "$PM" update v0.7.0`, unknownProgramEntryID},
		{`PM=npm; printf -v "$X" y; "$PM" update v1.2.3`, unknownProgramEntryID},
		{`PM=npm; printf "$O" "$X" y; "$PM" update v1.2.3`, unknownProgramEntryID},
		{`PM=npm; export "$X=y"; "$PM" update v1.2.3`, unknownProgramEntryID},
		{`PM=npm; wait -p "$X"; "$PM" update v1.2.3`, unknownProgramEntryID},
		{`PM=npm; getopts a: "$X"; "$PM" update v1.2.3`, unknownProgramEntryID},
		{`PM=npm; mapfile -C "$X" a < f; "$PM" update v1.2.3`, unknownProgramEntryID},
		{`PM=abcd; PM=npm; "$PM" update v0.7.0`, entry},
		{`PM=scripts/abcd.sh; "$PM" update v0.7.0`, updateTagAnyEntryID},
		{`PM=npm "$PM" update v1.2.3`, unknownProgramEntryID},
		{`PM="$X"; "$PM" update v1.2.3`, unknownProgramEntryID},
		{`PM=np*; $PM update v1.2.3`, unknownProgramEntryID},
		{`PM=npm; PM+=x; "$PM" update v1.2.3`, unknownProgramEntryID},
		{`(PM=npm); "$PM" update v1.2.3`, unknownProgramEntryID},
		{`PM=npm & "$PM" update v1.2.3`, unknownProgramEntryID},
		{`PM=npm | "$PM" update v1.2.3`, unknownProgramEntryID},
		{`false && PM=npm; "$PM" update v1.2.3`, unknownProgramEntryID},
		{`if x; then PM=npm; fi; "$PM" update v1.2.3`, unknownProgramEntryID},
		{`X=1 || PM=npm; "$PM" update v1.2.3`, unknownProgramEntryID},
		{`X=1 < list.txt && PM=npm; "$PM" update v1.2.3`, unknownProgramEntryID},
		{`PM=npm; . ./env.sh; "$PM" update v1.2.3`, unknownProgramEntryID},
		{`PM=npm; eval "$X"; "$PM" update v1.2.3`, unknownProgramEntryID},
		{`PM=npm; "$X" y; "$PM" update v1.2.3`, unknownProgramEntryID},
		{`PM=npm; read PM; "$PM" update v1.2.3`, unknownProgramEntryID},
		{`PM=npm; read "$X"; "$PM" update v1.2.3`, unknownProgramEntryID},
		{`PM=npm; printf -v PM '%s' x; "$PM" update v1.2.3`, unknownProgramEntryID},
		{`PM=npm; declare "$X"; "$PM" update v1.2.3`, unknownProgramEntryID},
		{`PM=npm; declare -n R=PM; "$PM" update v1.2.3`, unknownProgramEntryID},
		{`PM=npm; unset PM; "$PM" update v1.2.3`, unknownProgramEntryID},
		{`PM=npm; f() { PM=y; }; f; "$PM" update v1.2.3`, unknownProgramEntryID},
		{`PM=npm; : $((PM=1)); "$PM" update v1.2.3`, unknownProgramEntryID},
		{`for PM in "$@"; do "$PM" update v1.2.3; done`, unknownProgramEntryID},
		{`for PM in bin/*; do "$PM" update v1.2.3; done`, unknownProgramEntryID},
		{`for PM in npm; do :; done; PM="$X"; "$PM" update v1.2.3`, unknownProgramEntryID},
		{`for PM in; do :; done; "$PM" update v1.2.3`, unknownProgramEntryID},
		{`PM=env; "$PM" "$X" update v1.2.3`, unknownProgramEntryID},
		{`PM=env; "$PM" abcd update v0.7.0`, entry},
		{`PM=npm; n=P; declare -n r="${n}M"; r=abcd; "$PM" update v1.2.3`, unknownProgramEntryID},
		{`PM=npm; n=P; local -n r="${n}M"; r=abcd; "$PM" update v1.2.3`, unknownProgramEntryID},
		{`PM=npm; n=P; typeset -n r="${n}M"; r=abcd; "$PM" update v1.2.3`, unknownProgramEntryID},
		{`PM=npm; declare -n r=$1; r=abcd; "$PM" update v1.2.3`, unknownProgramEntryID},
		{`PM=npm; declare -n r; r=PM; r=abcd; "$PM" update v1.2.3`, unknownProgramEntryID},
		{`PM=npm; printf -vPM abcd; "$PM" update v1.2.3`, unknownProgramEntryID},
		{`PM=npm; print -vPM abcd; "$PM" update v1.2.3`, unknownProgramEntryID},
		{`PM=npm; read -aPM < f; "$PM" update v1.2.3`, unknownProgramEntryID},
		{`PM=npm; read PM?; "$PM" update v1.2.3`, unknownProgramEntryID},
		{`PM=npm; printf -v PM? abcd; "$PM" update v1.2.3`, unknownProgramEntryID},
		{`PM=npm; zmodload zsh/pcre; pcre_compile x; pcre_match -v PM x; "$PM" update v1.2.3`, unknownProgramEntryID},
		{`PM=npm; n=PM; : ${(P)n::=abcd}; "$PM" update v1.2.3`, unknownProgramEntryID},
		{`PM=npm; n=PM; echo ${(P)n::=abcd}; "$PM" update v1.2.3`, unknownProgramEntryID},
		{`PM=npm; n=PM; zstat -A "$n" f; "$PM" update v1.2.3`, unknownProgramEntryID},
		{`PM=npm; echo *; "$PM" update v1.2.3`, unknownProgramEntryID},
		{`PM=npm; echo $(true); "$PM" update v1.2.3`, unknownProgramEntryID},
		{`PM=npm; [[ 1 -eq PM=1 ]]; "$PM" update v1.2.3`, unknownProgramEntryID},
		{`PM=npm; printf "$F" x; "$PM" update v1.2.3`, unknownProgramEntryID},
		{`PM=npm; X=$(true); "$PM" update v1.2.3`, unknownProgramEntryID},
		{`PM=npm; "$PM" ci; "$PM" update v1.2.3`, unknownProgramEntryID},
		{`$PM:s/x// update v1.2.3`, updateTagAnyEntryID},
		{`PM=xbcd; $PM:s/x/a/ update v1.2.3`, updateTagAnyEntryID},
		{`PM=xbcd; "$PM:s/x/a/" update v1.2.3`, updateTagAnyEntryID},
		{`PM=abcdX; $PM[1,4] update v1.2.3`, updateTagAnyEntryID},
	} {
		d, err := reg.Check(c.cmd)
		if err != nil || d.Verdict != VerdictBlock || d.EntryID != c.entry {
			t.Errorf("Check(%q) = %q via %q, %v; want a block via %q", c.cmd, d.Verdict, d.EntryID, err, c.entry)
		}
	}
	for _, cmd := range []string{`"$PM" update "$PKG"`, `$(which npm) update "$PKG"`} {
		d, err := reg.Check(cmd)
		if err != nil {
			t.Fatal(err)
		}
		for _, want := range []string{"variable", "substitution"} {
			if !strings.Contains(d.Reason, want) {
				t.Errorf("%q: the reason does not name a %s: %s", cmd, want, d.Reason)
			}
		}
		if strings.Contains(d.Reason, "is the output of") || strings.Contains(d.Successor, "in a variable") {
			t.Errorf("%q: the refusal calls the name a substitution's output, or sends it to a variable: %s %s", cmd, d.Reason, d.Successor)
		}
	}
}

// TestGoRunOfAFileIsTheProgramOfItsDirectory: a `go run` of a Go file is the program
// whose directory the file stands in, as a `go run` of an unnamed package is.
func TestGoRunOfAFileIsTheProgramOfItsDirectory(t *testing.T) {
	reg := Defaults()
	for _, cmd := range []string{
		`cd cmd/abcd && go run main.go update v0.7.0`,
		`cd cmd/abcd && go run ./main.go update "$TAG"`,
		`cd cmd/abcd && go run main.go flags.go update v0.7.0`,
		`cd cmd/abcd && go run *.go update v0.7.0`,
		`pushd cmd/abcd; go run -v main.go --json update v0.7.0`,
		`go run -C cmd/abcd main.go update v0.7.0`,
		`go run cmd/abcd/main.go update v0.7.0`,
		`go run ./cmd/abcd/main.go ./cmd/abcd/flags.go update v0.7.0`,
		`cat tag.txt | xargs go run cmd/abcd/main.go update`,
	} {
		d, err := reg.Check(cmd)
		if err != nil || d.Verdict != VerdictBlock || d.EntryID != updateTagAnyEntryID {
			t.Errorf("Check(%q) = %q via %q, %v; want a block via %q", cmd, d.Verdict, d.EntryID, err, updateTagAnyEntryID)
		}
	}
	for _, cmd := range []string{
		`go run main.go update v1.2.3`,
		`go run cmd/other/main.go update v1.2.3`,
		`cd cmd/other && go run main.go update v1.2.3`,
		`cd cmd/abcd && go run main.go update`,
		`cd cmd/abcd && go run main.go update --check`,
		`cd cmd/abcd && go run main.go update news --seen`,
		`cd cmd/abcd && go vet main.go && ./deploy.sh update v1.2.3`,
	} {
		if d, err := reg.Check(cmd); err != nil || d.Verdict == VerdictBlock {
			t.Errorf("Check(%q) = %q via %q, %v; want it not refused", cmd, d.Verdict, d.EntryID, err)
		}
	}
}

// TestARepositorysGuardFileCannotClaimAPersonActsID: a repository's guard file that
// declares an entry under an act's reserved id does not load, and the act stays
// refused under the registry that is left.
func TestARepositorysGuardFileCannotClaimAPersonActsID(t *testing.T) {
	for _, a := range personActs {
		dir := t.TempDir()
		if err := os.MkdirAll(filepath.Join(dir, ".abcd"), 0o755); err != nil {
			t.Fatal(err)
		}
		file := `{"schema_version": 1, "entries": {"` + a.id + `": {"tier": "warn", "pattern": {"command": "ls"},
			"why": "w", "successor": "s"}}}`
		if err := os.WriteFile(filepath.Join(dir, filepath.FromSlash(RepoRelPath)), []byte(file), 0o644); err != nil {
			t.Fatal(err)
		}
		ld := LoadRepo(dir)
		if ld.Posture != LoadRepoDropped || !errors.Is(ld.Err, ErrInvalidEntry) || !strings.Contains(ld.Err.Error(), a.id) {
			t.Errorf("%s: posture %v, %v; want the repository's file refused for the id", a.id, ld.Posture, ld.Err)
		}
		if _, has := ld.Registry.Entries[a.id]; has {
			t.Errorf("%s is an entry of the registry left in force", a.id)
		}
		if d, err := ld.Registry.Check("scripts/abcd.sh update v0.7.0"); err != nil || d.EntryID != updateTagAnyEntryID || d.Verdict != VerdictBlock {
			t.Errorf("%s: the registry left in force reads an update with a tag as %q via %q, %v", a.id, d.Verdict, d.EntryID, err)
		}
	}
}

// TestAnUnlistedCommandLeavesNoProgramFixed: a command that is not one of the few
// that set no variable, anywhere on the line, leaves a variable's program unread,
// whatever the command; the same line with a listed command in its place fixes it.
func TestAnUnlistedCommandLeavesNoProgramFixed(t *testing.T) {
	reg := Defaults()
	for _, quiet := range []string{"echo hi", "printf '%s' x", "true", "false", ":", "cd web", "test -f x", "[ -f x ]", "[[ -f x ]]", "X=1"} {
		cmd := `PM=npm; ` + quiet + `; "$PM" update v1.2.3`
		if d, err := reg.Check(cmd); err != nil || d.Verdict == VerdictBlock {
			t.Errorf("Check(%q) = %q via %q, %v; want it not refused", cmd, d.Verdict, d.EntryID, err)
		}
	}
	for _, loud := range []string{
		"ls", "cat f", "make", "npm ci", "grep x f", "print hi", "read x", "declare x=1", "export X=1",
		"local x", "typeset x", "readonly x", "set -e", "unset x", "eval true", "source f", ". f", "trap '' INT",
		"alias a=b", "zmodload zsh/pcre", "getopts a o", "mapfile a", "shift", "wait", "builtin echo hi",
		"command echo hi", "env", "let x=1", "vared x", "zstyle x y z", "select x in a; do :; done",
		"X=(a b)", "X+=a", "X=$Y", "for x in $(ls); do :; done", "echo ${x:=y}", "echo $((x=1))",
	} {
		cmd := `PM=npm; ` + loud + `; "$PM" update v1.2.3`
		d, err := reg.Check(cmd)
		if err != nil || d.Verdict != VerdictBlock {
			t.Errorf("Check(%q) = %q via %q, %v; want it refused", cmd, d.Verdict, d.EntryID, err)
		}
	}
}
