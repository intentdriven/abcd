package cli

// hook_session_name.go — the hooks' half of session naming
// (itd-2610111707085098, spc-2610152057143798). internal/core/sessionname
// decides the name and keeps the machine's register; this file reads what the
// hook payload says about the session, prints the core's notes on stderr, and
// shapes the prompt hook's stdout into the one object that sets a title.
//
// Session start reserves a name and sets no title, so its stdout is byte for
// byte what it was before naming: the host ignores a prompt-hook title equal
// to the one a session already has, so a title set at start never reached the
// session list (the intent's decision 9). The first prompt sets it. The host
// takes a title from a hook only as a JSON object on stdout, so on that one
// prompt the router's plain stdout becomes that object's additionalContext,
// which the host injects exactly as it injects plain stdout. Every other
// prompt leaves stdout as it was: the plain form is the default and the object
// is the exception.

import (
	"bytes"
	"encoding/json"
	"io"

	"github.com/spf13/cobra"

	"github.com/intentdriven/abcd/internal/core/rules"
	"github.com/intentdriven/abcd/internal/core/sessionname"
)

// sessionNameMachine is the machine naming reads (the home, the process table,
// the hook's parent) and whether naming runs at all. Production always names;
// the package's TestMain switches naming off, so no test that did not ask for
// it reads this machine's process table or writes a register into a home, and
// a test that does ask swaps in a fixture machine.
var sessionNameMachine = func() (sessionname.Machine, bool) { return sessionname.Live(), true }

// startSessionName is the session-start hook's naming: it reserves the name
// the first prompt sets, and writes nothing to stdout. Its notes go to
// stderr, masked, one line each; it never fails the hook.
func startSessionName(cmd *cobra.Command, in hookInput, cwd string) {
	m, ok := sessionNameMachine()
	if !ok {
		return
	}
	req := sessionname.Request{SessionID: in.SessionID, Source: in.Source, Title: in.SessionTitle, AgentID: in.AgentID}
	// The persona list is read from the repository the rules loader reads,
	// bounded by the same ownership gate, and only when the payload is one
	// naming acts on: the resolution asks git.
	if sessionname.Acts(req) {
		req.Root = rules.ResolveRoot(cwd)
	}
	for _, n := range sessionname.Start(m, req).Notes {
		diagnosticLine(cmd.ErrOrStderr(), "%s", n)
	}
}

// resendSessionTitle is the prompt hook's naming: the title owed on this
// prompt, or "". Its notes go to stderr; it never fails the hook.
func resendSessionTitle(cmd *cobra.Command, in hookInput) string {
	m, ok := sessionNameMachine()
	if !ok {
		return ""
	}
	out := sessionname.Resend(m, in.SessionID, in.SessionTitle)
	for _, n := range out.Notes {
		diagnosticLine(cmd.ErrOrStderr(), "%s", n)
	}
	return out.Name
}

// titledOutput is the hook output object that sets a session's title. The
// field order is the spec's.
type titledOutput struct {
	HookSpecificOutput titledFields `json:"hookSpecificOutput"`
}

type titledFields struct {
	HookEventName     string `json:"hookEventName"`
	AdditionalContext string `json:"additionalContext"`
	SessionTitle      string `json:"sessionTitle"`
}

// emitTitled writes the one object that sets title from the prompt hook,
// carrying context, the text the hook would otherwise have printed. A write
// that fails is named on stderr and the hook still exits 0, as every hook verb
// does.
func emitTitled(cmd *cobra.Command, context, title string) {
	var b bytes.Buffer
	enc := json.NewEncoder(&b)
	// The context is the hook's own text, and the host decodes it whole; HTML
	// escaping would only make the object harder for a person to read.
	enc.SetEscapeHTML(false)
	if err := enc.Encode(titledOutput{HookSpecificOutput: titledFields{
		HookEventName: "UserPromptSubmit", AdditionalContext: context, SessionTitle: title,
	}}); err != nil {
		diagnosticLine(cmd.ErrOrStderr(), "abcd session name: the title was not written (%s)", err.Error())
		return
	}
	if _, err := io.Copy(cmd.OutOrStdout(), &b); err != nil {
		diagnosticLine(cmd.ErrOrStderr(), "abcd session name: the title was not written (%s)", err.Error())
	}
}
