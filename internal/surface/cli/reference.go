package cli

//go:generate go run ../../../cmd/abcd-gen-cli-ref

import (
	"fmt"
	"regexp"
	"sort"
	"strings"

	"github.com/spf13/cobra"
)

// blankRuns collapses any run of three or more newlines to a single blank line,
// so section spacing is uniform regardless of which optional blocks a command
// emits. Applied once to the whole page — the collapse is deterministic.
var blankRuns = regexp.MustCompile(`\n{3,}`)

// ReferencePagePath is the committed CLI reference page, relative to the repo
// root. The generator writes it and the drift test (reference_test.go) diffs the
// freshly-walked tree against it, so the two agree on one location.
const ReferencePagePath = "docs/reference/cli/commands.md"

// referenceIntro is the preamble of the generated reference page. It is part of
// the generated artefact — the drift test compares the whole file — so it lives
// here beside the walker rather than being appended to the committed page by hand.
const referenceIntro = "# CLI command reference\n\n" +
	"This page is generated from the abcd command tree by `GenerateReference` in\n" +
	"`internal/surface/cli`. It is a derived artefact: do not edit it by hand. A\n" +
	"drift test regenerates the tree and fails the build whenever this page and the\n" +
	"tree disagree, so the reference can never silently go stale. Regenerate it with\n" +
	"`go generate ./internal/surface/cli`.\n\n" +
	"Every user-facing command is listed with its usage line, summary, and flags;\n" +
	"the operator-internal hook entrypoints are omitted.\n"

// GenerateReference walks the abcd command tree and renders it as a single,
// deterministic Markdown reference page — the source of truth for
// docs/reference/cli/commands.md. Hidden commands (the operator-internal `hook`
// subtree) are omitted, and children are emitted in a stable alphabetical order,
// so the output depends only on the command tree — never on registration order
// or the clock. That determinism is what lets a `go test` diff detect drift.
func GenerateReference() string {
	var b strings.Builder
	b.WriteString(referenceIntro)
	root := NewRootCommand()
	// Cobra attaches its own `completion` and `help` commands only when the
	// root executes, so a tree walked straight from NewRootCommand omits two
	// commands the binary answers to, and the page's "every user-facing
	// command" would be false (iss-304). Attach them here exactly as Execute
	// does.
	root.InitDefaultHelpCmd()
	root.InitDefaultCompletionCmd()
	writeCommandRef(&b, root)
	// Uniform spacing and a single trailing newline, so the page is stable no
	// matter which optional blocks each command emitted.
	return strings.TrimRight(blankRuns.ReplaceAllString(b.String(), "\n\n"), "\n") + "\n"
}

// writeCommandRef emits one command's section and recurses into its non-hidden
// children in alphabetical order. Heading depth tracks the command's depth in the
// tree (capped at Markdown's h6), so the page mirrors the command hierarchy.
func writeCommandRef(b *strings.Builder, cmd *cobra.Command) {
	if cmd.Hidden {
		return
	}

	level := strings.Count(cmd.CommandPath(), " ") + 2
	if level > 6 {
		level = 6
	}
	fmt.Fprintf(b, "\n%s `%s`\n\n", strings.Repeat("#", level), cmd.CommandPath())

	if short := strings.TrimSpace(cmd.Short); short != "" {
		fmt.Fprintf(b, "%s\n\n", short)
	}
	fmt.Fprintf(b, "**Usage:** `%s`\n\n", cmd.UseLine())

	if long := strings.TrimSpace(cmd.Long); long != "" && long != strings.TrimSpace(cmd.Short) {
		fmt.Fprintf(b, "%s\n\n", long)
	}
	if flags := strings.TrimRight(cmd.LocalFlags().FlagUsages(), "\n"); flags != "" {
		b.WriteString("**Flags:**\n\n```\n")
		b.WriteString(flags)
		b.WriteString("\n```\n\n")
	}
	if example := strings.TrimSpace(cmd.Example); example != "" {
		b.WriteString("**Example:**\n\n```\n")
		b.WriteString(example)
		b.WriteString("\n```\n\n")
	}

	children := cmd.Commands()
	sort.Slice(children, func(i, j int) bool { return children[i].Name() < children[j].Name() })
	for _, child := range children {
		writeCommandRef(b, child)
	}
}
