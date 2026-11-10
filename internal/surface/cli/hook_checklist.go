package cli

import (
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/intentdriven/abcd/internal/core/autonomy"
	"github.com/intentdriven/abcd/internal/core/checklist"
	"github.com/intentdriven/abcd/internal/core/rules"
	"github.com/intentdriven/abcd/internal/fsutil"
	"github.com/intentdriven/abcd/internal/termsafe"
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
	// characters. The hook measures in bytes, which never undercounts characters.
	sessionStartOutputCap = rules.HostOutputCap
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
	if stageActive() {
		return ""
	}
	root, ok := managedRoot(cwd)
	if !ok || !checklistFilesExist(root) {
		return ""
	}
	return checklist.Render(checklist.Read(root), budget)
}

// recordCompaction is the PreCompact event sidecar (spc-2610230805429429
// step 6): the compaction's time and trigger are appended to the managed
// checkout's checklist.events.jsonl. It never blocks the compaction: the
// hook's exit and its stdout are the reset's alone, and a compaction that
// could not be recorded is one line on stderr saying so. Outside a managed
// checkout, and on a walled stage, it records nothing and says nothing, as the
// other checklist hooks stand aside there.
func recordCompaction(w io.Writer, cwd, trigger string) {
	defer func() {
		if recover() != nil {
			fmt.Fprintln(w, "abcd checklist: the compaction was not recorded in "+checklist.EventsRelPath+" (the record failed); the compaction goes ahead")
		}
	}()
	if stageActive() {
		return
	}
	if cwd == "" {
		if wd, err := os.Getwd(); err == nil {
			cwd = wd
		}
	}
	root, ok := managedRoot(cwd)
	if !ok {
		return
	}
	if err := checklist.RecordCompaction(root, trigger, time.Now()); err != nil {
		fmt.Fprintf(w, "abcd checklist: the compaction was not recorded in %s (%s); the compaction goes ahead\n",
			checklist.EventsRelPath, termsafe.Sanitize(fsutil.RedactHome(strings.ReplaceAll(err.Error(), root, "<checkout>"))))
	}
}

// checklistPromptNote is the behind note a present session's last stop left
// for this prompt (spc-2610230805429429 step 4): the stop hook's message
// reaches the person, not the model, so the next prompt carries the same note
// into the session's context, once, outside the rules loader's dedup ledger.
// It reads the session's state in the working copy the session acts from, and
// nothing else: no owed reading, no network. Outside a managed checkout, on a
// drive's stage, or with a session id the store cannot key, it adds nothing;
// a state that cannot be read adds nothing and is named on stderr.
func checklistPromptNote(stderr io.Writer, cwd, sessionID string) string {
	if stageActive() || !autonomy.ValidSessionID(sessionID) {
		return ""
	}
	if _, ok := managedRoot(cwd); !ok {
		return ""
	}
	copyRoot, _, err := autonomy.ResolveCopy(cwd, sessionID, autonomy.Options{})
	if err != nil {
		return ""
	}
	note, err := autonomy.TakeChecklistNote(copyRoot, sessionID)
	if err != nil {
		diagnosticLine(stderr, "abcd checklist: the behind note could not be read (%s)", fsutil.RedactHome(err.Error()))
		return ""
	}
	if note == "" {
		return ""
	}
	return "## CHECKLIST\n\n" + note + "\n"
}
