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
