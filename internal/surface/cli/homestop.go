package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/intentdriven/abcd/internal/abcdhome"
)

// homestop.go — the stop while the old home stands (itd-2610030720038073
// decision 6, spc-2610031309233367 "The stop").
//
// abcd's home is ~/.abcd.noindex. While an old ~/.abcd stands, abcd moves
// nothing and reads nothing from either folder: every command, every hook and
// the status line write nothing and name the folder and the one rename command,
// until the person renames it. The check runs in the binary's one front door,
// before the command tree executes, so no verb's own code runs while stopped,
// and nothing the verb would have created (a fresh ~/.abcd.noindex beside the
// old folder above all) can be. It reads the two names with Lstat
// (abcdhome.Check) and, for `guard hook` alone, the hook payload on stdin; it
// writes only to stdout and stderr.
//
// Each command is answered in its own form, because each is read by a
// different reader: a person or a script reads an ordinary verb's refusal; the
// host adds a prompt or session-start hook's stdout to the agent's context, so
// the agent can tell the person; a staging hook's result line is the machine
// contract its callers parse; the shell guard's blocking status is what keeps a
// command from running unchecked; and the status line is one short row.

// homeStop is the stop in force for the home HOME names, or nil. An empty or
// relative HOME is no stop: the refusals for that shape stand where they are.
func homeStop() *abcdhome.Stop {
	return abcdhome.Check(os.Getenv("HOME"))
}

// renderHomeStop answers args with the stop in the form of the command they
// name, and returns the exit status. It is called only when a stop is in
// force, which is also why root.Find's side effect on the tree (findByPath's
// note) costs nothing here: the tree never executes.
func renderHomeStop(root *cobra.Command, args []string, stop *abcdhome.Stop, stdin io.Reader, stdout, stderr io.Writer) int {
	path := ""
	if cmd, _, err := root.Find(args); err == nil && cmd != nil {
		path = cmd.CommandPath()
	}
	asJSON := jsonRequestedIn(args)
	switch strings.TrimPrefix(path, root.Name()+" ") {
	case "hook prompt-router":
		// The host adds a prompt hook's stdout to the agent's context, so the
		// agent reads the line and tells the person. Exit 0: a hook never
		// wedges a session.
		if asJSON {
			_ = render(stdout, true, routerView{Text: stop.Line + "\n", Injected: []string{}, Error: stop.Line}, nil)
			return 0
		}
		fmt.Fprintln(stdout, stop.Line)
		return 0
	case "hook session-start":
		fmt.Fprintln(stdout, stop.Line)
		return 0
	case "hook prompt-router-reset", "hook session-end", "hook subagent-stop":
		// The staging hooks' uncaptured result (hook_result.go), the line as
		// its reason, and exit 0 on every path as their contract says.
		hook := strings.TrimPrefix(path, root.Name()+" hook ")
		fmt.Fprintln(stderr, "abcd:", stop.Line)
		if asJSON {
			_ = json.NewEncoder(stdout).Encode(hookStageResult{Hook: hook, Outcome: hookOutcomeNotCaptured, Reason: stop.Line})
		}
		return 0
	case "guard hook":
		if admitsRename(stop, stdin) {
			return 0
		}
		// The host's blocking status: nothing runs unguarded while abcd cannot
		// read its own state (the spec's open question 3).
		fmt.Fprintln(stderr, "abcd:", stop.Line)
		return 2
	case "statusline":
		fmt.Fprintln(stdout, stop.Short)
		return 0
	}
	if asJSON {
		enc := json.NewEncoder(stdout)
		enc.SetIndent("", "  ")
		_ = enc.Encode(newErrorEnvelope(stop.Line, 1))
		return 1
	}
	fmt.Fprintln(stderr, "abcd:", stop.Line)
	return 1
}

// admitsRename reports whether the guard hook's payload is the one command the
// stop names, run through the shell tool: the person's own act, which the guard
// lets through so their session can make it. Only the old folder's stop admits
// it. With both folders standing, the same command would move the old folder
// INTO the new one, so it is refused with the rest and the line says what to
// do instead. Anything unreadable is not the rename.
func admitsRename(stop *abcdhome.Stop, stdin io.Reader) bool {
	if stop.Both || stdin == nil {
		return false
	}
	raw, err := io.ReadAll(io.LimitReader(stdin, maxHookStdinBytes+1))
	if err != nil || len(raw) > maxHookStdinBytes {
		return false
	}
	var in guardHookInput
	if json.Unmarshal(raw, &in) != nil {
		return false
	}
	// Only the white space a shell itself drops around a command is trimmed:
	// strings.TrimSpace would also strip Unicode spaces (U+00A0, U+3000), which
	// a shell keeps, so `mv` would rename the folder to a name with an
	// invisible character in it and the stop would lift.
	return strings.EqualFold(in.ToolName, "Bash") && strings.Trim(in.ToolInput.Command, " \t\r\n") == abcdhome.RenameCommand
}
