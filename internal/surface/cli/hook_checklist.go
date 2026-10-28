package cli

import (
	"io"

	"github.com/intentdriven/abcd/internal/core/checklist"
	"github.com/spf13/cobra"
)

// hook_checklist.go — the checklist re-injected at every session start
// (spc-2610230805429429 step 3, "Re-injection"). SessionStart's hooks entry
// has no matcher, so `abcd hook session-start` runs for every source:
// startup, resume, clear and compact. Each of them ends its one output with
// the checklist's render.
//
// The host caps a hook's whole output at 10,000 characters and swaps a longer
// one for a preview, so the checklist shares that cap with everything else
// the hook prints (the away block, the update notices, the inbox greeting and
// the notice count). The hook composes all of that first; the checklist gets
// what it leaves, less a margin, and is the one part that yields.

const (
	// sessionStartOutputCap is the host's cap on a hook's whole output, in
	// characters (the hooks reference, read 2026-10-22). The hook measures in
	// bytes, which never undercounts characters.
	sessionStartOutputCap = 10_000
	// sessionStartMargin is kept free under the cap (the spec's margin).
	sessionStartMargin = 500
	// checklistRenderBudget is the checklist's budget when nothing else is
	// printed: the one `abcd checklist render` uses, so the verb shows what a
	// session start with nothing else to say would inject.
	checklistRenderBudget = sessionStartOutputCap - sessionStartMargin
)

// emitSessionStart writes the session-start hook's one output: composed, the
// text the hook gathered, then the checklist within what composed leaves of
// the cap. The rest is never cut; a checklist that cannot be read becomes the
// reader's one line, never an error or a non-zero exit.
func emitSessionStart(cmd *cobra.Command, composed, cwd string) {
	text := composed + sessionStartChecklist(cwd, checklistRenderBudget-len(composed))
	if text != "" {
		io.WriteString(cmd.OutOrStdout(), text)
	}
}

// sessionStartChecklist is the checklist's render for the managed checkout at
// cwd within budget bytes: nothing outside a managed checkout, nothing for a
// working copy that has never had a checklist (the intent re-injects "a
// session with a checklist"), and otherwise checklist.Render of what the
// reader returns, which carries its own line when it fell back or found
// nothing usable. It reads no network and writes nothing.
func sessionStartChecklist(cwd string, budget int) (text string) {
	// A start is never failed for the checklist: a panic in the read or the
	// render is the line, not a crash of the hook.
	defer func() {
		if recover() != nil {
			text = checklist.Render(checklist.Result{Source: checklist.FromNothing,
				Reason: "No checklist to show: it could not be read at this start; `abcd checklist` says why."}, budget)
		}
	}()
	root, ok := managedRoot(cwd)
	if !ok || !checklistFilesExist(root) {
		return ""
	}
	return checklist.Render(checklist.Read(root), budget)
}
