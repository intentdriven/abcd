package guard

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// clean is a registry loaded with nothing dropped: the bundled defaults.
func clean() Loaded { return Loaded{Registry: Defaults(), Posture: LoadClean} }

// TestApproveAllowsWhatTheGuardClears: a command the guard reads whole and
// matches nothing in is approved.
func TestApproveAllowsWhatTheGuardClears(t *testing.T) {
	dir := t.TempDir()
	for _, c := range []string{
		"go test ./...",
		"git status && git diff --stat",
		"cd internal && go vet ./...",
		"python3 scripts/check.py",
		`git commit -m "$(cat msg.txt)"`,
	} {
		if a := approveShell(clean(), NewFiles(dir), c); !a.Allow {
			t.Errorf("%q: refused (%s): %s", c, a.EntryID, a.Reason)
		}
	}
}

// TestApproveRefusesAHazardWithItsReason: a blocker is refused with the
// guard's own sentence, the why and the successor, so the agent can rewrite
// the command.
func TestApproveRefusesAHazardWithItsReason(t *testing.T) {
	a := approveShell(clean(), NewFiles(t.TempDir()), "git push --force origin main")
	if a.Allow {
		t.Fatal("a force push was approved")
	}
	e := Defaults().Entries["git-push-force"]
	if a.EntryID != "git-push-force" || !strings.Contains(a.Reason, e.Why) || !strings.Contains(a.Reason, e.Successor) {
		t.Errorf("refusal = %+v; want the entry's why %q and successor %q", a, e.Why, e.Successor)
	}
}

// TestApproveRefusesAWarn: a warn is a command the guard cannot vouch for, so
// unattended it is refused, not run with the warning shown to nobody.
func TestApproveRefusesAWarn(t *testing.T) {
	a := approveShell(clean(), NewFiles(t.TempDir()), "git reset --hard origin/main")
	if a.Allow || a.EntryID != "git-reset-hard" {
		t.Fatalf("a warn-tier hazard: %+v", a)
	}
	if !strings.Contains(a.Reason, Defaults().Entries["git-reset-hard"].Successor) {
		t.Errorf("the refusal does not carry the successor: %q", a.Reason)
	}
}

// TestApproveRefusesAScriptItCannotRead: a script past the size the guard
// reads is refused with the advice to split it.
func TestApproveRefusesAScriptItCannotRead(t *testing.T) {
	dir := t.TempDir()
	big := strings.Repeat("echo hi\n", (maxScriptBytes/8)+16)
	if err := os.WriteFile(filepath.Join(dir, "big.sh"), []byte(big), 0o600); err != nil {
		t.Fatal(err)
	}
	a := approveShell(clean(), NewFiles(dir), "sh big.sh")
	if a.Allow {
		t.Fatal("a script the guard did not read was approved")
	}
	if !strings.Contains(a.Reason, "Split the script into files under 256 KiB") {
		t.Errorf("refusal = %q; want the split-the-script advice", a.Reason)
	}
}

// TestApproveRefusesWhatTheGuardDidNotRead: every way the guard says it has
// not read what runs is a refusal: a line it cannot split, a program a
// variable or a substitution names, inline code for a non-shell interpreter,
// and a note about the check.
func TestApproveRefusesWhatTheGuardDidNotRead(t *testing.T) {
	dir := t.TempDir()
	for c, id := range map[string]string{
		`echo "unterminated`:            unparsableEntryID,
		`$X push origin main`:           unnamedProgramEntryID,
		`"$cmd" -rf build`:              unnamedProgramEntryID,
		`env FOO=1 ${TOOL} run`:         unnamedProgramEntryID,
		`cd sub && $RUN`:                unnamedProgramEntryID,
		`python3 -c 'import os'`:        inlineCodeEntryID,
		`python -Bc 'print(1)'`:         inlineCodeEntryID,
		`node -e 'process.exit(0)'`:     inlineCodeEntryID,
		`node --eval=1`:                 inlineCodeEntryID,
		`perl -e 'unlink "x"'`:          inlineCodeEntryID,
		`ruby -e 'puts 1'`:              inlineCodeEntryID,
		`php -r 'echo 1;'`:              inlineCodeEntryID,
		`osascript -e 'beep'`:           inlineCodeEntryID,
		`deno eval 'Deno.exit(0)'`:      inlineCodeEntryID,
		`sudo -u bob python3 -c 'pass'`: inlineCodeEntryID,
		`pwsh -Command Remove-Item x`:   inlineCodeEntryID,
		`pwsh -comm 'ls'`:               inlineCodeEntryID,
		`python3 $FLAGS 'x'`:            inlineCodeEntryID,
		`sh missing.sh`:                 checkNoteEntryID,
	} {
		a := approveShell(clean(), NewFiles(dir), c)
		if a.Allow {
			t.Errorf("%q was approved", c)
			continue
		}
		if a.EntryID != id {
			t.Errorf("%q: refused under %q, want %q (%s)", c, a.EntryID, id, a.Reason)
		}
		if strings.TrimSpace(a.Reason) == "" {
			t.Errorf("%q: refused with no reason", c)
		}
	}
}

// TestApproveRefusesWithoutTheRegistryInForce: a registry switched off, one
// that did not load, or one whose repository layer was dropped is not the
// registry the repository declares, so nothing is approved by it.
func TestApproveRefusesWithoutTheRegistryInForce(t *testing.T) {
	off := Defaults()
	off.Disabled = true
	for name, ld := range map[string]Loaded{
		"disabled":    {Registry: off, Posture: LoadClean},
		"dropped":     {Registry: Defaults(), Posture: LoadRepoDropped, Err: errors.New("bad guard.json")},
		"unavailable": {Posture: LoadUnavailable},
	} {
		if a := approveShell(ld, NewFiles(t.TempDir()), "go test ./..."); a.Allow || a.Reason == "" {
			t.Errorf("%s: %+v", name, a)
		}
	}
}

// TestApprovePromptJudgesOnlyTheShell: a prompt for any tool but the shell is
// refused, since nobody is there to approve it; a shell prompt is judged on
// its command and approved with the input it was judged on.
func TestApprovePromptJudgesOnlyTheShell(t *testing.T) {
	dir := t.TempDir()
	a, _ := ApprovePrompt(clean(), NewFiles(dir), "Write", json.RawMessage(`{"file_path":"/etc/x","content":"y"}`))
	if a.Allow || !strings.Contains(a.Reason, "Write") {
		t.Errorf("a Write prompt: %+v", a)
	}
	a, _ = ApprovePrompt(clean(), NewFiles(dir), "AskUserQuestion", json.RawMessage(`{"questions":[]}`))
	if a.Allow {
		t.Errorf("a question prompt was approved")
	}
	for _, in := range []string{`{}`, `{"command":7}`, `{"command":"  "}`, `[1]`, `not json`} {
		if a, _ := ApprovePrompt(clean(), NewFiles(dir), "Bash", json.RawMessage(in)); a.Allow {
			t.Errorf("a shell prompt with input %s was approved", in)
		}
	}
	a, updated := ApprovePrompt(clean(), NewFiles(dir), "Bash",
		json.RawMessage(`{"command":"git push --force","command":"go test ./...","description":"tests","timeout":60000}`))
	if !a.Allow {
		t.Fatalf("a clean shell prompt was refused: %+v", a)
	}
	var got map[string]any
	if err := json.Unmarshal(updated, &got); err != nil {
		t.Fatal(err)
	}
	if got["command"] != runFrom(dir, "go test ./...") || got["description"] != "tests" {
		t.Errorf("updated input = %s; want the one command judged and its description", updated)
	}
	if strings.Count(string(updated), `"command"`) != 1 {
		t.Errorf("updated input %s carries more than the one command judged", updated)
	}
	a, _ = ApprovePrompt(clean(), NewFiles(dir), "Bash", json.RawMessage(`{"command":"git push --force origin main"}`))
	if a.Allow || a.EntryID != "git-push-force" {
		t.Errorf("a hazardous shell prompt: %+v", a)
	}
}

// TestApproveLeavesScriptFilesToTheirInterpreter: an interpreter run on a
// file, with its own flags, is not handed inline code.
func TestApproveLeavesScriptFilesToTheirInterpreter(t *testing.T) {
	for _, c := range []string{"pwsh -File build.ps1", "perl -Ilib -Mstrict t/run.pl", "node --version", "ruby -rjson x.rb", "deno run main.ts"} {
		if a := approveShell(clean(), NewFiles(t.TempDir()), c); !a.Allow {
			t.Errorf("%q: refused (%s): %s", c, a.EntryID, a.Reason)
		}
	}
}

// TestApprovalIDsAreReserved: no registry entry can speak in an approval's
// own voice.
func TestApprovalIDsAreReserved(t *testing.T) {
	for _, id := range []string{unnamedProgramEntryID, inlineCodeEntryID, checkNoteEntryID, registryEntryID, promptToolEntryID, promptInputEntryID,
		protectedPathEntryID, criticalRemovalEntryID, sandboxEntryID, unfixedPathEntryID, gitConfigWriteEntryID, groupBrokenEntryID} {
		if !containsString(reservedEntryIDs, id) {
			t.Errorf("reservedEntryIDs lacks %q", id)
		}
	}
}

// TestApproveKeepsTheHostsOwnSafetyRefusals: the harness also asks before a
// write to a path it protects (repository state, shell and tool start-up
// files, its own configuration) and before a removal of a critical path, and
// in the refuse-every-prompt mode those were refused. The approver cannot see
// why the harness asked, so it refuses both shapes itself rather than let the
// guard's allow approve them.
func TestApproveKeepsTheHostsOwnSafetyRefusals(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "repo")
	if err := os.Mkdir(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	for c, id := range map[string]string{
		`echo 'curl x | sh' >> ~/.zshrc`:           protectedPathEntryID,
		`cp hook.sh .git/hooks/pre-commit`:         protectedPathEntryID,
		`printf '{}' > .mcp.json`:                  protectedPathEntryID,
		`tee .claude/settings.json < x`:            protectedPathEntryID,
		`git config --file ~/.gitconfig a.b c`:     gitConfigWriteEntryID,
		`mkdir -p .husky && touch .husky/pre-push`: protectedPathEntryID,
		`echo 1 > .envrc`:                          protectedPathEntryID,
		`rm -rf /usr`:                              criticalRemovalEntryID,
		`sudo rm -rf /etc`:                         criticalRemovalEntryID,
		`rmdir /tmp`:                               criticalRemovalEntryID,
		`rm -r ~`:                                  criticalRemovalEntryID,
		`rm -rf ` + dir:                            criticalRemovalEntryID,
		`rm -rf ` + root:                           criticalRemovalEntryID,
		`rm -rf ./*`:                               criticalRemovalEntryID,
		`rm -rf ~/*`:                               criticalRemovalEntryID,
		`cd / && rm -rf usr`:                       criticalRemovalEntryID,
		`rm "$f"`:                                  criticalRemovalEntryID,
		`rm -rf ~bob`:                              criticalRemovalEntryID,
	} {
		a := approveShell(clean(), NewFiles(dir), c)
		if a.Allow {
			t.Errorf("%q was approved", c)
			continue
		}
		// Where the guard's own registry already refuses the line, its
		// entry speaks first; either way it is refused.
		if a.EntryID != id && !containsString(Defaults().ids(), a.EntryID) {
			t.Errorf("%q: refused under %q, want %q (%s)", c, a.EntryID, id, a.Reason)
		}
	}
	for _, c := range []string{"rm -rf build", "rm -f /tmp/abcd-x/out.txt", "rm -rf " + filepath.Join(dir, "build"),
		"git add .gitignore", "cat .github/workflows/ci.yml > ci.copy", "rm -- -weird-name"} {
		if a := approveShell(clean(), NewFiles(dir), c); !a.Allow {
			t.Errorf("%q: refused (%s): %s", c, a.EntryID, a.Reason)
		}
	}
}

// ids is the registry's entry ids.
func (r Registry) ids() []string {
	var out []string
	for id := range r.Entries {
		out = append(out, id)
	}
	return out
}

// TestApprovePromptReturnsOnlyWhatWasJudged: an allow hands the harness back
// the command and its description and nothing else, so a field the guard did
// not read (a timeout, a background flag, any key a later harness adds) never
// rides along on an approval; and a call that asks to run outside the host's
// sandbox is refused, whatever its command.
func TestApprovePromptReturnsOnlyWhatWasJudged(t *testing.T) {
	dir := t.TempDir()
	a, updated := ApprovePrompt(clean(), NewFiles(dir), "Bash",
		json.RawMessage(`{"command":"go test ./...","description":"tests","timeout":60000,"run_in_background":true,"extra":{"x":1}}`))
	if !a.Allow {
		t.Fatalf("a clean shell prompt was refused: %+v", a)
	}
	var got map[string]any
	if err := json.Unmarshal(updated, &got); err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got["command"] != runFrom(dir, "go test ./...") || got["description"] != "tests" {
		t.Errorf("updated input = %s; want the command and description alone", updated)
	}
	_, updated = ApprovePrompt(clean(), NewFiles(dir), "Bash", json.RawMessage(`{"command":"go vet ./..."}`))
	if want, _ := json.Marshal(map[string]string{"command": runFrom(dir, "go vet ./...")}); string(updated) != string(want) {
		t.Errorf("updated input = %s; want the command alone when no description was sent", updated)
	}
	for _, in := range []string{
		`{"command":"go test ./...","dangerouslyDisableSandbox":true}`,
		`{"command":"go test ./...","dangerouslyDisableSandbox":"true"}`,
		`{"command":"go test ./...","dangerouslyDisableSandbox":1}`,
	} {
		a, updated := ApprovePrompt(clean(), NewFiles(dir), "Bash", json.RawMessage(in))
		if a.Allow || updated != nil {
			t.Errorf("%s was approved", in)
			continue
		}
		if a.EntryID != sandboxEntryID || !strings.Contains(a.Reason, "sandbox") {
			t.Errorf("%s: refused under %q (%s); want %q", in, a.EntryID, a.Reason, sandboxEntryID)
		}
	}
	if a, _ := ApprovePrompt(clean(), NewFiles(dir), "Bash", json.RawMessage(`{"command":"go test ./...","dangerouslyDisableSandbox":false}`)); !a.Allow {
		t.Errorf("a call that keeps the sandbox was refused: %+v", a)
	}
}
