package cli

import (
	"encoding/json"
	"io"
	"strings"

	"github.com/intentdriven/abcd/internal/termsafe"
)

// The guard hook's answer to the host, on stdout (iss-2610070637562567).
//
// A block is the host's PreToolUse deny: exit 0 and one JSON object whose
// hookSpecificOutput carries permissionDecision "deny" and the reason. The
// host shows the reason to the person on one error line, without echoing the
// hook's command, and hands it to the agent. A note is the same object
// carrying additionalContext alone: the call runs with no error line, and the
// agent reads the note after the tool returns. A note never sets a permission
// decision, which would bypass the host's own permission flow. A warn or a
// fail-open goes to stderr with exit 1, and an allow prints nothing on stdout.

// preToolUseOutput is the host's hook output object. Each field is omitted
// when empty, so a deny and a note each carry their own members alone.
type preToolUseOutput struct {
	HookSpecificOutput preToolUseDecision `json:"hookSpecificOutput"`
}

type preToolUseDecision struct {
	HookEventName            string `json:"hookEventName"`
	PermissionDecision       string `json:"permissionDecision,omitempty"`
	PermissionDecisionReason string `json:"permissionDecisionReason,omitempty"`
	AdditionalContext        string `json:"additionalContext,omitempty"`
}

// hookDeny is a guard hook decision to refuse the call. The hook's RunE
// returns it from wherever the decision is made, and newGuardHookCommand
// turns it into the one deny object on stdout, with every diagnostic the
// invocation wrote before it folded into the reason.
type hookDeny struct{ reason string }

func (d *hookDeny) Error() string { return d.reason }

// denyCall is the refusal of the call, for the reason given: the text the
// person and the agent read, one or more lines.
func denyCall(reason string) error { return &hookDeny{reason: reason} }

// writeHookDeny writes the host's deny for reason to w: one JSON object on one
// line. Each line of the reason is sanitised, and the lines are joined with a
// newline, blank ones dropped.
func writeHookDeny(w io.Writer, reason string) error {
	return writeHookOutput(w, preToolUseDecision{
		PermissionDecision:       "deny",
		PermissionDecisionReason: hookText(reason),
	})
}

// writeHookNote writes a note for the agent to w, the call admitted: one JSON
// object carrying additionalContext and no permission decision.
func writeHookNote(w io.Writer, note string) error {
	return writeHookOutput(w, preToolUseDecision{AdditionalContext: hookText(note)})
}

func writeHookOutput(w io.Writer, d preToolUseDecision) error {
	d.HookEventName = "PreToolUse"
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	return enc.Encode(preToolUseOutput{HookSpecificOutput: d})
}

// hookText is s as the host shows it: each line sanitised, blank lines
// dropped, the rest joined with a newline.
func hookText(s string) string {
	var out []string
	for _, l := range strings.Split(s, "\n") {
		if l = termsafe.Sanitize(l); strings.TrimSpace(l) != "" {
			out = append(out, l)
		}
	}
	return strings.Join(out, "\n")
}
