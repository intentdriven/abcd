package guard

import (
	"strings"
	"testing"
)

// TestGluedFixedOutputIsReadAsTheJoinedWords — review7-guard finding 3
// (iss-2609260115380561). A fixed `$(cat <<'F' … F)` output was read as its
// words only where the word was wholly that substitution; text glued to it, or
// a second such substitution, left the word unknown, and a lone unknown word
// in command position allows. bash joins the output's first and last words to
// the text beside them and runs the result, so the joined words are read:
// unquoted, split on blanks and newlines where the output stood; quoted, one
// word. Each shape was run under bash 3.2 and 5.3 with a neutral word.
func TestGluedFixedOutputIsReadAsTheJoinedWords(t *testing.T) {
	const push = "git push --force origin main"
	doc := func(body string) string { return "$(cat <<'F'\n" + body + "\nF\n)" }
	runVerdictCases(t, []verdictCase{
		{doc(push) + "x", VerdictBlock, "git-push-force"},
		{doc("git") + doc(" push --force origin main"), VerdictBlock, "git-push-force"},
		{doc("git push ") + "--force origin main", VerdictBlock, "git-push-force"},
		{"sudo " + doc("git push --force origin") + "x", VerdictBlock, "git-push-force"},
		{"`cat <<'F'\n" + push + "\nF\n`x", VerdictBlock, "git-push-force"},
		{doc(push) + "$(true)", VerdictBlock, "git-push-force"},
		// Quoted, the output and the text beside it are one word: a payload.
		{"sh -c \"" + doc(push) + "\"x", VerdictBlock, "git-push-force"},
		{"sh -c \"" + doc("git push") + " --force origin main\"", VerdictBlock, "git-push-force"},
		{"sh -c \"" + doc("git push") + doc(" --force origin main") + "\"", VerdictBlock, "git-push-force"},

		// Glued text that makes another word keeps the verdict bash earns.
		{"x" + doc(push), VerdictAllow, ""},
		{doc("echo hi") + "x", VerdictAllow, ""},
		{"echo \"" + doc(push) + "\"x", VerdictAllow, ""},
	})
}

// TestGluedFixedOutputStaysLinear pins the cost of the joined reading: each
// output is copied once into the words its word makes.
func TestGluedFixedOutputStaysLinear(t *testing.T) {
	shapes := map[string]func(int) string{
		"many glued outputs": func(n int) string {
			return strings.Repeat("echo $(cat <<'F'\na b c\nF\n)x$(cat <<'F'\nd e\nF\n)\n", n)
		},
		"one word of many glued outputs": func(n int) string {
			return "echo " + strings.Repeat("$(cat <<'F'\na b c\nF\n)x", n)
		},
		"one long glued output": func(n int) string {
			return "echo $(cat <<'F'\n" + strings.Repeat("abcdefgh ijklmnop qrstuvwx ", n) + "\nF\n)x"
		},
	}
	for name, build := range shapes {
		build := build
		t.Run(name, func(t *testing.T) {
			assertWorkGrowth(t, build, 1<<8, "each glued output is joined once")
		})
	}
}
