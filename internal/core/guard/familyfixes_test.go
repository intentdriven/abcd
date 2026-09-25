package guard

import "testing"

// The guard-bypass family: four "incomplete set at a trust boundary" defects,
// each a sibling form the matcher/tokenizer/launcher tables missed while the
// canonical spelling was covered. Every case here ALLOWED (silently) on pre-fix
// code and must now reach the same verdict as the spelling the guard already
// blocks.

// TestBacktickCommandSubstitutionBlocks — gh-312. A backtick-quoted cmd runs it exactly as
// `$(cmd)` does, but the tokenizer had no backtick case, so the hazard never
// reached command position while the `$( … )` spelling blocked.
func TestBacktickCommandSubstitutionBlocks(t *testing.T) {
	cases := []struct {
		name    string
		command string
		want    Verdict
	}{
		{"bare backtick gh delete", "`gh repo delete owner/repo`", VerdictBlock},
		{"echo-wrapped backtick gh delete", "echo `gh repo delete owner/repo`", VerdictBlock},
		{"assignment backtick force push", "x=`git push --force origin main`", VerdictBlock},
		{"cd-chained backtick rm", "cd scratch && `rm -rf *`", VerdictBlock},
		// Parity: the `$( … )` spelling that already blocked must keep blocking.
		{"dollar-paren still blocks", "echo $(gh repo delete owner/repo)", VerdictBlock},
		// A backtick inside single quotes is literal in the shell, so it stays a
		// quoted argument — never command position, never a hazard.
		{"single-quoted backtick is inert", "echo '`gh repo delete owner/repo`'", VerdictAllow},
		{"benign backtick is not invented into a hazard", "echo `git status`", VerdictAllow},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := guardVerdict(t, tc.command).Verdict; got != tc.want {
				t.Errorf("Check(%q).Verdict = %q, want %q", tc.command, got, tc.want)
			}
		})
	}
}

// TestCommandNameCaseInsensitive — gh-315. On a case-insensitive filesystem
// (macOS's default, a first-class CI target) `GIT`/`RM`/`GH`/`SUDO` resolve to
// the real binary and execute the hazard, but the byte-exact `cmd != p.Command`
// compare (and the case-sensitive `wrappers` lookup) let them pass.
func TestCommandNameCaseInsensitive(t *testing.T) {
	cases := []struct {
		name    string
		command string
		want    Verdict
	}{
		{"upper GIT push force", "GIT push --force origin main", VerdictBlock},
		{"mixed Git push force", "Git push --force origin main", VerdictBlock},
		{"upper RM after cd", "cd scratch && RM -rf *", VerdictBlock},
		{"upper GH repo delete", "GH repo delete owner/repo --yes", VerdictBlock},
		{"upper SUDO wrapper still shows the command", "SUDO git push --force origin main", VerdictBlock},
		{"upper BASH -c descends", `BASH -c "git push --force origin main"`, VerdictBlock},
		{"upper ENV -S descends", "ENV -S 'gh repo delete owner/repo'", VerdictBlock},
		{"upper SU -c descends", "SU -c 'git push --force origin main'", VerdictBlock},
		// The fold is scoped to the command NAME. Arguments, subcommands and flags
		// stay case-sensitive — git/gh/rm parse those case-sensitively, so a
		// case-varied subcommand does NOT run the hazard and must not be blocked.
		{"uppercase subcommand does not match", "git PUSH --force origin main", VerdictAllow},
		{"uppercase flag does not match", "git push --FORCE origin main", VerdictAllow},
		{"benign lowercase is unaffected", "git status", VerdictAllow},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := guardVerdict(t, tc.command).Verdict; got != tc.want {
				t.Errorf("Check(%q).Verdict = %q, want %q", tc.command, got, tc.want)
			}
		})
	}
}

// TestSingleStringLaunchersDescend — gh-354. `watch '<cmd>'` and GNU `parallel
// '<cmd>' ::: args` hand a single quoted operand to `sh -c`, but they were in no
// launcher table, so the quoted payload stayed one opaque token that defeated
// even the Tier-2 fail-safe. The UNQUOTED, multi-token form already warns via
// Tier 2 (pinned in the adversarial corpus) and must stay a warn.
func TestSingleStringLaunchersDescend(t *testing.T) {
	cases := []struct {
		name    string
		command string
		want    Verdict
	}{
		{"watch quoted force push", "watch 'git push --force origin main'", VerdictBlock},
		{"watch with separate interval flag", "watch -n 5 'git push --force origin main'", VerdictBlock},
		{"watch with glued interval flag", "watch -n1 'gh repo delete owner/repo'", VerdictBlock},
		{"parallel quoted force push", "parallel 'git push --force origin main' ::: a b", VerdictBlock},
		{"sudo watch still descends", "sudo watch 'git push --force origin main'", VerdictBlock},
		// Benign quoted payloads must not be invented into hazards.
		{"watch benign is allowed", "watch 'git status'", VerdictAllow},
		{"watch version flag is allowed", "watch --version", VerdictAllow},
		// The unquoted, multi-token form is the Tier-2 warn the corpus pins; the
		// fix must not silence it and must not upgrade it to a block.
		{"unquoted watch stays a warn", "watch -n1 git push --force origin main", VerdictWarn},
		{"unquoted parallel stays a warn", "parallel git push --force ::: a b", VerdictWarn},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := guardVerdict(t, tc.command).Verdict; got != tc.want {
				t.Errorf("Check(%q).Verdict = %q, want %q", tc.command, got, tc.want)
			}
		})
	}
}
