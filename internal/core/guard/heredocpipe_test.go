package guard

import (
	"strings"
	"testing"
)

// TestAHereDocBodyReachesTheCommandsItsOwnerPipesInto — review-drainG3
// finding 2 (iss-2609270036253187). What an unquoted here-document's
// substitutions print is its command's standard input, and that command's
// output is what the pipe after it hands on: `cat <<EOF | xargs kill` over
// `$(pgrep make)` is the here-string twin `cat <<< "$(pgrep make)" | xargs
// kill`. The body is read at the newline, after the whole pipeline, so the
// search lay past the pipe window of every command downstream of the owner;
// each of those is now handed it too.
func TestAHereDocBodyReachesTheCommandsItsOwnerPipesInto(t *testing.T) {
	runVerdictCases(t, []verdictCase{
		{"cat <<EOF | xargs kill\n$(pgrep make)\nEOF", VerdictBlock, "kill-by-search"},
		{"cat <<EOF | tee log | xargs kill\n$(pgrep make)\nEOF", VerdictBlock, "kill-by-search"},
		{"cat <<EOF | sh -c 'xargs kill'\n$(pgrep make)\nEOF", VerdictBlock, "kill-by-search"},
		{"{ cat <<EOF; } | xargs kill\n$(pgrep make)\nEOF", VerdictBlock, "kill-by-search"},
		{"cat <<EOF | xargs kill\n`pgrep make`\nEOF", VerdictBlock, "kill-by-search"},
		{"cat <<EOF | echo \"$(xargs kill)\"\n$(pgrep make)\nEOF", VerdictBlock, "kill-by-search"},
		{"cat <<A <<B | xargs kill\nx\nA\n$(pgrep make)\nB", VerdictBlock, "kill-by-search"},

		// The quoted document is text; a command that is not downstream of
		// the owner reads nothing from its body.
		{"cat <<EOF | xargs kill\n4242\nEOF", VerdictAllow, ""},
		{"cat <<'EOF' | xargs kill\n$(pgrep make)\nEOF", VerdictAllow, ""},
		{"cat <<EOF | wc -l; echo 4242 | xargs kill\n$(pgrep make)\nEOF", VerdictAllow, ""},
		{"cat <<EOF; xargs kill < pidfile\n$(pgrep make)\nEOF", VerdictAllow, ""},
		{"echo 4242 | xargs kill; cat <<EOF | wc -l\n$(pgrep make)\nEOF", VerdictAllow, ""},
	})
}

// TestAHereDocBodyFeedStaysLinear pins the cost of handing a body to the
// commands downstream of its owner: a pipeline of n owners each with a body
// is read in work linear in its length, not once per owner per command.
func TestAHereDocBodyFeedStaysLinear(t *testing.T) {
	shapes := map[string]func(int) string{
		"a pipeline of here-document owners": func(n int) string {
			return strings.Repeat("cat <<E | ", n) + "xargs kill\n" + strings.Repeat("$(echo 1)\nE\n", n)
		},
		"owners in a group piped on": func(n int) string {
			return "{ " + strings.Repeat("cat <<E; ", n) + "} | xargs kill\n" + strings.Repeat("$(echo 1)\nE\n", n)
		},
	}
	for name, build := range shapes {
		build := build
		t.Run(name, func(t *testing.T) {
			assertWorkGrowth(t, build, 1<<7, "a here-document body is handed on once per downstream command")
		})
	}
}
