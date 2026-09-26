package cli

import (
	"io"

	"github.com/spf13/cobra"
)

// moved.go — itd-2609212130136102: a spelling that moved stays one release as a
// stub that answers with its successor and exits non-zero, so muscle memory and
// scripts fail loudly rather than silently, and the release after removes it.
//
// Two shapes of move exist. A spelling that moved WHOLE — `ahoy dry-run`,
// `version`, `docs lint`, `site check` — becomes a deprecated command: cobra's
// Deprecated field keeps it out of every command list, the help counts and the
// completion, and cobra prints the deprecation on the way in. A spelling whose
// BARE form moved while its sub-verbs did not — `identity`, whose `init` and
// `render` stay, and `ahoy remote`, whose `apply` stays — keeps its place, and
// only its bare invocation answers with the successor.
//
// Either way the command carries annotationMovedTo, which the surface snapshot
// records as `moved_to` and the brief's appendix and sub-verb tables read, so
// the move is written down once.

// annotationMovedTo is the successor of a moved spelling: the invocation that
// does what it did.
const annotationMovedTo = "abcd.moved_to"

// movedTo is the successor cmd records, or "" when it did not move.
func movedTo(cmd *cobra.Command) string {
	return cmd.Annotations[annotationMovedTo]
}

// movedRefusal is the answer a moved spelling gives: exit 2, naming the
// invocation to run instead. It is an error, so --json renders it as the one
// refusal envelope on stdout and the text render prints it on stderr.
func movedRefusal(old, successor string) error {
	return &exitError{Code: 2, Msg: "`" + old + "` moved to `" + successor + "`; run `" + successor +
		"` instead (the old spelling is removed in the next release)"}
}

// markMoved records successor on cmd and makes its bare invocation the stub's
// answer. It is the whole of the move for a command whose sub-verbs stay.
func markMoved(cmd *cobra.Command, successor string) {
	annotate(cmd, annotationMovedTo, successor)
	cmd.RunE = func(c *cobra.Command, _ []string) error {
		return movedRefusal(c.CommandPath(), successor)
	}
}

// movedStub builds the stub a spelling that moved whole leaves behind. It takes
// any argument and any flag the old spelling took, so a script passing them
// meets the answer rather than a usage error, and it runs nothing.
//
// The deprecation notice cobra prints goes to the command's output stream,
// which Run binds to stdout; that stream is re-routed to stderr here, so a
// `--json` caller reads exactly one refusal on stdout and nothing else.
func movedStub(use, successor string) *cobra.Command {
	cmd := &cobra.Command{
		Use:                use,
		Args:               cobra.ArbitraryArgs,
		Deprecated:         "run `" + successor + "` instead",
		FParseErrWhitelist: cobra.FParseErrWhitelist{UnknownFlags: true},
	}
	markMoved(cmd, successor)
	cmd.SetOut(stderrOf{cmd})
	return cmd
}

// stderrOf writes to the error stream of the tree cmd belongs to, resolved at
// write time because a stub is built before Run binds the streams.
type stderrOf struct{ cmd *cobra.Command }

func (s stderrOf) Write(p []byte) (int, error) {
	var w io.Writer = s.cmd.Root().ErrOrStderr()
	return w.Write(p)
}
