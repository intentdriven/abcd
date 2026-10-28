package cli

// The PermissionRequest hook (iss-2610171110154902): the host is about to show
// the person an approval prompt for a shell command. While abcd's build or
// drain runs for the session's checkout nobody is at the terminal to answer
// it, and the run would wait until someone came back; so abcd refuses the
// prompt and tells the agent how to rewrite the command. It never grants one:
// an approval granted to whatever the agent retries would let any command past
// every check by being sent twice. In every other session it says nothing and
// the host's prompt shows as it always has.
//
// Fail-open like every hook verb: an unreadable payload, a tool that is not
// the shell, a checkout that cannot be read and an error of any kind print
// nothing on stdout and exit 0, which is the host's "no decision", and its
// prompt shows.

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/intentdriven/abcd/internal/core/implement/loop"
	"github.com/intentdriven/abcd/internal/termsafe"
)

// unattendedRefusalNotice is the one line build and drain print as they
// start: the person is told once that the prompts are refused while it runs.
const unattendedRefusalNotice = "While this runs, the host's approval prompts for shell commands in this checkout and the lanes it runs " +
	"are refused, and the agent is told to rewrite the command into plain steps abcd can check."

// noticeUnattended tells the person, once, as a build or drain starts, that
// the host's approval prompts are refused while it runs. It goes to stderr
// whatever the output mode, as the drain's loosened-floor warning does, so a
// reader of --json sees it and the JSON on stdout stays whole.
func noticeUnattended(cmd *cobra.Command, prefix string) {
	fmt.Fprintln(cmd.ErrOrStderr(), prefix+": "+unattendedRefusalNotice)
}

// permissionRequestOutput is the host's PermissionRequest hook output. The
// decision carries no interrupt, so the agent carries on with the message.
type permissionRequestOutput struct {
	HookSpecificOutput permissionRequestDecision `json:"hookSpecificOutput"`
}

type permissionRequestDecision struct {
	HookEventName string                  `json:"hookEventName"`
	Decision      permissionRequestBehave `json:"decision"`
}

type permissionRequestBehave struct {
	Behavior string `json:"behavior"`
	Message  string `json:"message"`
}

// writePermissionDeny writes the host's refusal of an approval prompt to w:
// one JSON object on one line, the message sanitised as a guard deny's reason
// is. There is no allow writer: abcd never grants a prompt.
func writePermissionDeny(w io.Writer, message string) error {
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	return enc.Encode(permissionRequestOutput{HookSpecificOutput: permissionRequestDecision{
		HookEventName: "PermissionRequest",
		Decision:      permissionRequestBehave{Behavior: "deny", Message: hookText(message)},
	}})
}

// unattendedRefusal is the message the agent is told when its prompt is
// refused for u.
func unattendedRefusal(u loop.Unattended) string {
	what := "a drain"
	switch {
	case u.Kind == loop.UnattendedBuild && u.Key != "":
		what = fmt.Sprintf("a build of %s (run %s)", u.Key, u.RunID)
	case u.Kind == loop.UnattendedBuild:
		// The core passes on only a key that is a record id.
		what = fmt.Sprintf("a build (run %s)", u.RunID)
	}
	return "abcd refused this approval prompt: " + what + " is running in this checkout, with nobody at the terminal to approve it. " +
		"Rewrite the command so it needs no approval: run each step as its own plain command, and put a script's steps on the " +
		"command line rather than in a file, a heredoc or a `bash -c` string. If the step cannot be done that way, skip it and " +
		"say why in your report. The same command sent again is refused again. If the person is at the terminal and no run is " +
		"going any more, tell them: the refusal ends two hours after the run last wrote its state."
}

func newPermissionRequestCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "permission-request",
		Short: "PermissionRequest on the shell tool: refuse the approval prompt while a build or drain runs",
		Long: `Reads the host's PermissionRequest payload on stdin. While abcd's build or
drain is running for the session's checkout, or the session is one of a
running build's lanes, stdout is the host's refusal of the prompt, one line:

  {"hookSpecificOutput": {"hookEventName": "PermissionRequest",
   "decision": {"behavior": "deny", "message": ...}}}

and the message tells the agent to rewrite the command into plain steps
abcd can check, or skip the step and say why. A build counts while its run
is unfinished, no lane of it is handed back or held (the person's by then),
and its state was written within the reservation lease (two hours); a drain while it has not stopped and was written within the same
window. The prompt is refused every time it comes back, and abcd never
grants one.

In any other session, for a tool that is not the shell, and on an
unreadable payload or any error, stdout is empty and the host's prompt
shows as usual. Every outcome exits 0.`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			in, err := readHookInput(cmd)
			if err != nil {
				fmt.Fprintf(cmd.ErrOrStderr(), "abcd permission: unreadable hook payload (%s); the host's prompt shows\n", termsafe.Sanitize(err.Error()))
				return nil
			}
			if !strings.EqualFold(in.ToolName, "Bash") {
				return nil
			}
			cwd := in.Cwd
			if cwd == "" {
				if wd, err := os.Getwd(); err == nil {
					cwd = wd
				}
			}
			u, on, err := loop.RunningUnattended(rulesRoot(cwd, cmd.ErrOrStderr()), time.Now())
			if err != nil {
				fmt.Fprintf(cmd.ErrOrStderr(), "abcd permission: the run state could not be read (%s); the host's prompt shows\n", termsafe.Sanitize(err.Error()))
				return nil
			}
			if !on {
				return nil
			}
			if err := writePermissionDeny(cmd.OutOrStdout(), unattendedRefusal(u)); err != nil {
				fmt.Fprintf(cmd.ErrOrStderr(), "abcd permission: the refusal could not be written (%s)\n", termsafe.Sanitize(err.Error()))
			}
			return nil
		},
	}
}
