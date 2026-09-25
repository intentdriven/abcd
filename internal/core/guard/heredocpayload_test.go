package guard

import (
	"strings"
	"testing"
)

// TestHereDocumentPayloadIsRead — review5-guard finding 3. `sh -c "$(cat
// <<'EOF' … EOF)"` and `eval "$(cat <<EOF … EOF)"` hand the shell the
// document's text verbatim, but a substitution's output is unknown to the
// tokenizer, so the payload read as uninspectable and the verdict was the
// family's WARN, which runs. Where the output is fixed — `cat` with nothing
// but the here-document, whose body is literal (a quoted delimiter, or one
// with no expansion in it) — the document's text is ALSO read as the payload.
// The unknown reading stays, so no verdict weakens: a document that reads
// clean keeps the warn it had.
func TestHereDocumentPayloadIsRead(t *testing.T) {
	const push = "git push --force origin main"
	runVerdictCases(t, []verdictCase{
		{"sh -c \"$(cat <<'EOF'\n" + push + "\nEOF\n)\"", VerdictBlock, "git-push-force"},
		{"sh -c \"$(cat <<EOF\n" + push + "\nEOF\n)\"", VerdictBlock, "git-push-force"},
		{"bash -c \"$(cat <<\"EOF\"\n" + push + "\nEOF\n)\"", VerdictBlock, "git-push-force"},
		{"eval \"$(cat <<'EOF'\n" + push + "\nEOF\n)\"", VerdictBlock, "git-push-force"},
		{"eval \"$(cat <<EOF\n" + push + "\nEOF\n)\"", VerdictBlock, "git-push-force"},
		{"bash -c \"$(cat <<-'EOF'\n\t" + push + "\n\tEOF\n)\"", VerdictBlock, "git-push-force"},
		{"su -c \"$(cat <<'EOF'\n" + push + "\nEOF\n)\"", VerdictBlock, "git-push-force"},
		{"sh -c \"$(cat <<'EOF'\necho start\n" + push + "\nEOF\n)\"", VerdictBlock, "git-push-force"},
		{"sh -c \"$(cat <<'EOF'\ncd s && rm -rf *\nEOF\n)\"", VerdictBlock, "rm-rf-after-cd-chain"},
		{"sh -c \"$(\ncat <<'EOF'\n" + push + "\nEOF\n)\" name", VerdictBlock, "git-push-force"},
		{"x=$(sh -c \"$(cat <<'EOF'\n" + push + "\nEOF\n)\")", VerdictBlock, "git-push-force"},
		{"sh -c \"$(cat <<'EOF'\nsh -c \"$(cat <<'IN'\n" + push + "\nIN\n)\"\nEOF\n)\"", VerdictBlock, "git-push-force"},

		// A document that reads clean keeps the warn the unknown output earns.
		{"sh -c \"$(cat <<'EOF'\necho hello\nEOF\n)\"", VerdictWarn, ""},
		// An output that is not the body's text stays unknown: an expansion in
		// an unquoted body, a command after cat, a word the output only joins.
		{"sh -c \"$(cat <<EOF\necho $HOME\nEOF\n)\"", VerdictWarn, ""},
		{"sh -c \"$(cat <<'EOF' | tr a b\necho hello\nEOF\n)\"", VerdictWarn, ""},
		// The document's text is data where no shell runs it.
		{"echo \"$(cat <<'EOF'\n" + push + "\nEOF\n)\"", VerdictAllow, ""},
		{"git commit -m \"$(cat <<'EOF'\nfix: it's done\n\n" + push + " is not run\nEOF\n)\"", VerdictAllow, ""},
	})
}

// TestHereDocumentPayloadStaysLinear pins the cost of reading a document as a
// payload: each document is scanned once for its literal output, and each
// payload read once however many readings return it.
func TestHereDocumentPayloadStaysLinear(t *testing.T) {
	shapes := map[string]func(int) string{
		"many document payloads": func(n int) string {
			return strings.Repeat("sh -c \"$(cat <<'EOF'\necho a b c\nEOF\n)\"\n", n)
		},
		"one long document payload": func(n int) string {
			return "sh -c \"$(cat <<'EOF'\n" + strings.Repeat("echo a b c; ls -l x\n", n) + "EOF\n)\""
		},
		"documents that are not payloads": func(n int) string {
			return strings.Repeat("echo \"$(cat <<EOF\ntext here\nEOF\n)\"\n", n)
		},
	}
	for name, build := range shapes {
		build := build
		t.Run(name, func(t *testing.T) {
			assertWorkGrowth(t, build, 1<<8, "each document is read once for its output and once as a payload")
		})
	}
}

// TestNestedHereDocumentPayloadIsRead — review6-guard finding 1. A document
// whose own text is a command-position `$(cat <<'F' … F)` hands the shell that
// substitution, and the shell runs its output: unquoted, the output is split
// into words on blanks and newlines and the first word is the command. The
// payload re-read saw only a bare substitution there and warned, which runs.
// That fixed output is ALSO read now, as the words bash splits it into, at
// every payload layer the guard follows; past maxPayloadDepth the layer is
// refused, not read, so a nesting too deep to follow blocks. Each shape was
// run under bash 3.2 and 5.3 with a neutral word in place of the hazard.
func TestNestedHereDocumentPayloadIsRead(t *testing.T) {
	const push = "git push --force origin main"
	inner := func(body string) string { return "$(cat <<'F'\n" + body + "\nF\n)" }
	runVerdictCases(t, []verdictCase{
		// Two levels: the review's shapes.
		{"sh -c \"$(cat <<'E'\n" + inner(push) + "\nE\n)\"", VerdictBlock, "git-push-force"},
		{"eval \"$(cat <<'E'\n" + inner(push) + "\nE\n)\"", VerdictBlock, "git-push-force"},
		{"bash -c \"$(cat <<'E'\n" + inner(push) + "\nE\n)\"", VerdictBlock, "git-push-force"},
		{"sh -c \"$(cat <<'E'\n$(cat <<F\n" + push + "\nF\n)\nE\n)\"", VerdictBlock, "git-push-force"},
		// The output is split on newlines too: one command, not five.
		{"sh -c \"$(cat <<'E'\n" + inner("git\npush\n--force\norigin\nmain") + "\nE\n)\"", VerdictBlock, "git-push-force"},
		// An empty output vanishes and the words after it are the command.
		{"sh -c \"$(cat <<'E'\n$(cat <<'F'\nF\n) " + push + "\nE\n)\"", VerdictBlock, "git-push-force"},
		// Its words are operands where the substitution is one.
		{"sh -c \"$(cat <<'E'\ngit " + inner("push --force origin main") + "\nE\n)\"", VerdictBlock, "git-push-force"},
		// The same substitution with no wrapper round it.
		{inner(push), VerdictBlock, "git-push-force"},
		{"cd s && " + inner("rm -rf *"), VerdictBlock, "rm-rf-after-cd-chain"},
		// Three levels: a document in a double-quoted document in a document.
		{"sh -c \"$(cat <<'E'\nsh -c \"$(cat <<'G'\n" + inner(push) + "\nG\n)\"\nE\n)\"", VerdictBlock, "git-push-force"},
		{"bash -c \"$(cat <<'E'\neval \"$(cat <<'G'\n" + inner(push) + "\nG\n)\"\nE\n)\"", VerdictBlock, "git-push-force"},
		// Past the bound the layer is refused, however clean its text.
		{"sh -c \"$(cat <<'E'\nsh -c \"$(cat <<'G'\nsh -c \"$(cat <<'H'\n" + inner("echo hello") + "\nH\n)\"\nG\n)\"\nE\n)\"", VerdictBlock, syntheticEntryID},
		{"sh -c \"$(cat <<'E'\nsh -c \"$(cat <<'G'\neval " + inner("echo hello") + "\nG\n)\"\nE\n)\"", VerdictBlock, syntheticEntryID},

		// A clean output keeps the warn the unknown reading earns.
		{"sh -c \"$(cat <<'E'\n" + inner("echo hello") + "\nE\n)\"", VerdictWarn, ""},
		// Words are not re-read as commands: bash runs `echo` with the rest as
		// its operands, and a `$(` inside the output is a word.
		{"sh -c \"$(cat <<'E'\n" + inner("echo "+push) + "\nE\n)\"", VerdictWarn, ""},
		{"sh -c \"$(cat <<'E'\n" + inner("$(cat <<'G'\n"+push+"\nG\n)") + "\nE\n)\"", VerdictWarn, ""},
		// Data where no shell runs it.
		{"echo \"$(cat <<'E'\n" + inner(push) + "\nE\n)\"", VerdictAllow, ""},
	})
}

// TestNestedHereDocumentPayloadStaysLinear pins the cost of reading a
// command-position document's words: each output is split once, and each
// layer is read once, whatever the nesting.
func TestNestedHereDocumentPayloadStaysLinear(t *testing.T) {
	shapes := map[string]struct {
		build func(int) string
		base  int
	}{
		"many nested document payloads": {func(n int) string {
			return strings.Repeat("sh -c \"$(cat <<'E'\n$(cat <<'F'\necho a b c\nF\n)\nE\n)\"\n", n)
		}, 1 << 8},
		// One document's output is one command of thousands of words, and a
		// command that long pays the bounded operand enumeration's constant
		// floor whether a document spells it or the line does (about 370,000
		// units: 75 a byte at 5 KB, 22 at 20 KB, for the plain command). The
		// base is set where the floor no longer dominates, so the bars see
		// the reading's own linear cost.
		"one long nested document": {func(n int) string {
			return "sh -c \"$(cat <<'E'\n$(cat <<'F'\n" + strings.Repeat("echo a b c; ls -l x\n", n) + "F\n)\nE\n)\""
		}, 1 << 10},
		"many three-level documents": {func(n int) string {
			return strings.Repeat("sh -c \"$(cat <<'E'\nsh -c \"$(cat <<'G'\n$(cat <<'F'\necho a b c\nF\n)\nG\n)\"\nE\n)\"\n", n)
		}, 1 << 8},
		"many command-position documents": {func(n int) string {
			return strings.Repeat("$(cat <<'F'\necho a b c\nF\n)\n", n)
		}, 1 << 8},
		// Each substitution holds a document and then the next one, so each
		// is offered for reading with every byte after it in its text.
		"documents nested in substitutions": {func(n int) string {
			return strings.Repeat("$(cat <<'F'\necho a b c\nF\n", n) + strings.Repeat(")", n)
		}, 1 << 6},
	}
	for name, shape := range shapes {
		shape := shape
		t.Run(name, func(t *testing.T) {
			assertWorkGrowth(t, shape.build, shape.base, "each output is split once and each layer read once")
		})
	}
}
