package cli

import (
	"fmt"
	"io"
	"strings"

	"github.com/intentdriven/abcd/internal/core/ahoy"
	"github.com/intentdriven/abcd/internal/core/mode"
	"github.com/intentdriven/abcd/internal/termsafe"
	"github.com/spf13/cobra"
)

// The question gate and its reset: the two moments the status-line badge must
// change, seen by the two hooks that already run (itd-2609212130146198).
//
// The guard hook, on the host's question tool, refuses a question while the
// mode reads managed — the agent has not said whom it is asking, so the badge
// would read "nobody is waiting" while somebody is — and marks an admitted
// question open. The prompt hook, on the next human message, resets the mode to
// managed when a question is marked open, because that message is the answer.
//
// This file is the whole of the feature on the front-door side. The guard's
// own hook calls questionGate once, before its shell-command path, and touches
// nothing of the shell guard's tokenizer or decision.

// questionTools are the host tool names that put a question to the human. The
// hook manifest's PreToolUse matcher names exactly these beside the shell tool
// (TestGuardHookIsInstalledForBashCalls holds the two together).
var questionTools = []string{"AskUserQuestion"}

// isQuestionTool reports whether a hook payload's tool is a question tool.
func isQuestionTool(name string) bool {
	for _, q := range questionTools {
		if strings.EqualFold(name, q) {
			return true
		}
	}
	return false
}

// questionRefusal is the one line the host replays to the agent when it asks
// while the mode reads managed. It names the two settings and the verb, and
// reminds the agent that choosing is its job.
const questionRefusal = "Blocked by the abcd guard (question tool): the mode reads managed, so the status line says nobody is waiting. " +
	"Before asking, say whom the question is for: run `abcd mode product-thinker` if it is for the product thinker, " +
	"or `abcd mode facilitator` if it is for the technical facilitator, then ask again. " +
	"The mode resets to managed on the next human message."

// questionGate is the guard hook's answer for a question-tool call.
//
// Where the badge does not show — no checkout, a repository abcd does not
// manage, or one without the local tier the verb writes to — the question is
// none of the gate's business and runs silently: refusing there would ask the
// agent to set a mode it cannot set. Where it shows, a managed mode refuses with
// the host's blocking status, and a mode that names somebody admits the
// question and marks it open for the reset. A store or marker the gate cannot
// read or write is not a decision: the question runs and the gate says so on the
// loud, non-blocking status, the guard's fail-open-loud contract.
func questionGate(cmd *cobra.Command, cwd string) error {
	stderr := cmd.ErrOrStderr()
	root, err := mode.Root(cwd)
	if err != nil || !ahoy.Managed(root) || !mode.HasTier(root) {
		return nil
	}
	st, err := mode.ReadAt(root)
	if err != nil {
		return questionFailOpen(stderr, "the mode store could not be read (%s)", err)
	}
	if st == mode.Managed {
		fmt.Fprintln(stderr, questionRefusal)
		return &exitError{Code: 2}
	}
	if err := mode.MarkQuestionOpen(root, st); err != nil {
		return questionFailOpen(stderr, "the question could not be marked open, so the mode will not reset on the answer (%s)", err)
	}
	return nil
}

// questionFailOpen is the gate's failOpen: exit 1, which lets the question run
// and keeps the warning in front of a human.
func questionFailOpen(w io.Writer, format string, err error) error {
	fmt.Fprintf(w, "abcd guard: NOT CHECKED — "+format+". The question runs UNGATED.\n",
		termsafe.Sanitize(scrubPaths(err)))
	return &exitError{Code: 1}
}

// resetModeOnAnswer is the prompt hook's half: when a question is marked open,
// this human message is its answer, so the mode goes back to managed, the
// marker is cleared, and one line on stderr says so. Nothing goes to stdout,
// which the host injects into the session's context. Every failure is named
// and never stops the prompt.
func resetModeOnAnswer(w io.Writer, cwd string) {
	root, err := mode.Root(cwd)
	if err != nil {
		return
	}
	reset, err := mode.ResetOnAnswer(root)
	switch {
	case err != nil:
		fmt.Fprintf(w, "abcd mode: the question marker could not be cleared, so the mode was not reset (%s)\n",
			termsafe.Sanitize(scrubPaths(err)))
	case reset:
		fmt.Fprintln(w, "abcd mode: the question was answered, so the mode is reset to managed")
	}
}
