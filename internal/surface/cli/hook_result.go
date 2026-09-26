package cli

import (
	"encoding/json"

	"github.com/spf13/cobra"
)

// hook_result.go — the machine-readable result channel of the staging hooks
// (iss-2608261550596333).
//
// session-end and subagent-stop exit 0 on every path, because a missed
// capture is permanent and a wedged session is worse: SessionEnd ignores the
// exit code and SubagentStop reads exit 2 as BLOCKING. So the exit code cannot
// say whether the transcript was kept, and the human stderr line is prose, not
// a contract; a programmatic caller that string-matched it shipped a silent
// permanent-loss bug. Under --json each hook writes exactly one JSON line to
// stdout, on every path, and still exits 0. Without --json nothing is written
// to stdout, the shape the host invokes. The line carries no top-level "abcd"
// key: that discriminator belongs to the refusal envelope alone.

// Outcomes a staging hook reports. Captured is true for the first three: the
// transcript is staged, and the next session redacts and stores it.
const (
	hookOutcomeStaged        = "staged"
	hookOutcomeRestaged      = "restaged"
	hookOutcomeAlreadyStaged = "already_staged"
	hookOutcomeNotCaptured   = "not_captured"
)

// hookStageResult is the one JSON line a staging hook writes under --json.
type hookStageResult struct {
	Hook          string `json:"hook"`
	Outcome       string `json:"outcome"`
	Captured      bool   `json:"captured"`
	SessionID     string `json:"session_id,omitempty"`
	AgentID       string `json:"agent_id,omitempty"`
	Bytes         int64  `json:"bytes,omitempty"`
	ReplacedBytes int64  `json:"replaced_bytes,omitempty"`
	Reason        string `json:"reason,omitempty"`
}

// emitHookResult writes r as one line on stdout when the caller asked for
// --json, and nothing otherwise. An encode failure is swallowed: the hook's
// exit-0 contract outranks its report.
func emitHookResult(cmd *cobra.Command, r hookStageResult) {
	asJSON, _ := cmd.Flags().GetBool("json")
	if !asJSON {
		return
	}
	_ = json.NewEncoder(cmd.OutOrStdout()).Encode(r)
}
