package guard

import (
	"strings"
	"testing"
)

// TestBacktickTextIsReadAfterItsBackslashPrePass — review7-guard finding 1
// (iss-2609260115287911). Between backticks bash removes a backslash before a
// `$`, a backtick or a backslash BEFORE it parses the command, and directly
// inside double quotes a backslash before a `"` as well. So an escaped
// `$( … )` or an escaped backtick pair in a backtick's text is a live
// substitution, in an unquoted here-document body there as much as in a word.
// The guard followed the text verbatim and skipped both as escaped. Each shape
// was run under bash 3.2 and 5.3 with a neutral word in place of the hazard.
func TestBacktickTextIsReadAfterItsBackslashPrePass(t *testing.T) {
	const push = "git push --force origin main"
	const bt = "`"
	runVerdictCases(t, []verdictCase{
		// An unquoted document inside backticks: its escaped substitutions run.
		{"echo " + bt + "cat <<F\n\\$(" + push + ")\nF\n" + bt, VerdictBlock, "git-push-force"},
		{"echo " + bt + "cat <<-F\n\\$(" + push + ")\nF\n" + bt, VerdictBlock, "git-push-force"},
		{"x=" + bt + "cat <<F\n\\$(" + push + ")\nF\n" + bt, VerdictBlock, "git-push-force"},
		{"echo " + bt + "cat <<F\n\\" + bt + push + "\\" + bt + "\nF\n" + bt, VerdictBlock, "git-push-force"},
		{"echo \"" + bt + "cat <<F\n\\" + bt + push + "\\" + bt + "\nF\n" + bt + "\"", VerdictBlock, "git-push-force"},
		{"echo \"" + bt + "cat <<F\n\\$(" + push + ")\nF\n" + bt + "\"", VerdictBlock, "git-push-force"},
		{"echo $((1 + " + bt + "cat <<F\n\\$(" + push + ")\nF\n" + bt + "))", VerdictBlock, "git-push-force"},
		// An escaped backtick pair nested in a backtick is a substitution.
		{"echo " + bt + "echo \\" + bt + push + "\\" + bt + bt, VerdictBlock, "git-push-force"},
		{"echo \"" + bt + "echo \\" + bt + push + "\\" + bt + bt + "\"", VerdictBlock, "git-push-force"},
		{"echo $(echo " + bt + "echo \\" + bt + push + "\\" + bt + bt + ")", VerdictBlock, "git-push-force"},
		{"cat <<F\n" + bt + "echo \\" + bt + push + "\\" + bt + bt + "\nF", VerdictBlock, "git-push-force"},
		// Directly inside double quotes an escaped `"` is a quote again.
		{"echo \"" + bt + "sh -c \\\"" + push + "\\\"" + bt + "\"", VerdictBlock, "git-push-force"},

		// Outside backticks the escape stands: bash prints the text.
		{"echo $(cat <<F\n\\$(" + push + ")\nF\n)", VerdictAllow, ""},
		// A quoted delimiter's body is literal after the pre-pass too.
		{"echo " + bt + "cat <<'F'\n\\$(" + push + ")\nF\n" + bt, VerdictAllow, ""},
		// An escaped backslash before an escaped backtick leaves one escape.
		{"echo " + bt + "echo \\\\\\" + bt + push + "\\\\\\" + bt + bt, VerdictAllow, ""},
	})
}

// TestBacktickPrePassStaysLinear pins the cost of the pre-pass: a backtick's
// text is scanned for its close once and read once more after the pre-pass.
func TestBacktickPrePassStaysLinear(t *testing.T) {
	shapes := map[string]func(int) string{
		"many escaped backtick texts": func(n int) string {
			return strings.Repeat("echo `echo \\$HOME a b c`\n", n)
		},
		"one long escaped backtick text": func(n int) string {
			return "echo `" + strings.Repeat("echo \\$HOME a b c; ", n) + "`"
		},
		"documents inside escaped backtick texts": func(n int) string {
			return strings.Repeat("echo `cat <<F\n\\$HOME text\nF\n`\n", n)
		},
	}
	for name, build := range shapes {
		build := build
		t.Run(name, func(t *testing.T) {
			assertWorkGrowth(t, build, 1<<8, "each backtick text is scanned for its close and read once after the pre-pass")
		})
	}
}
