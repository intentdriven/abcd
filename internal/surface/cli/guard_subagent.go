package cli

// The guard's hold on a sub-agent's writes until it has read the conventions
// file (spc-2611251757228271 step 2).
//
// The host names the sub-agent a call is made inside (agent_id), and never for
// the session's own calls. The hold reads that sub-agent's transcript, the one
// its stop is judged on, with the judge its stop uses, so a reading is one thing
// in both. It is asked only of a file the host's own tools write and of a shell
// line that makes or sends a commit, so a sub-agent that never writes reads no
// store and pays nothing.
//
// What cannot be judged is never held. It leaves a fact saying so, after which
// that sub-agent is not judged again: abcd does not come back to hold one it has
// said it could not judge, and it pays for git and a transcript read once. The
// stop's line is where it is said.

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/intentdriven/abcd/internal/core/guard"
	"github.com/intentdriven/abcd/internal/core/history"
	"github.com/intentdriven/abcd/internal/core/implement/loop"
	"github.com/intentdriven/abcd/internal/core/rules"
)

// Why a sub-agent could not be judged, the closed set an unjudged fact holds.
const (
	unjudgedFacts       = "facts-unreadable"
	unjudgedGit         = "git-did-not-answer"
	unjudgedRules       = "conventions-unknown"
	unjudgedNoPath      = "no-transcript-path"
	unjudgedTranscript  = "transcript-unreadable"
	unjudgedUnsayable   = "could-not-be-told"
	unjudgedFault       = "internal-fault"
	passedRead          = "read"
	passedBriefed       = "briefed"
	subagentTranscripts = "subagents"
)

// looseString is a payload field the hold reads and nothing else does: a value
// of another JSON type reads as empty, rather than failing the whole payload
// open, which would run the command unchecked.
type looseString string

func (s *looseString) UnmarshalJSON(b []byte) error {
	var v string
	if json.Unmarshal(b, &v) != nil {
		v = ""
	}
	*s = looseString(v)
	return nil
}

// subagentWrites reports whether a call is a sub-agent's write: a file the
// host's own tools write, or a shell line that makes or sends a commit. It is
// the one predicate the hold and the stop share.
func subagentWrites(tool, command string) bool {
	return isFileWriteTool(tool) || strings.EqualFold(tool, "Bash") && len(guard.RecordActs(command)) > 0
}

// judgeSubagentTranscript judges a transcript for the conventions file named.
func judgeSubagentTranscript(raw []byte, file string) history.SubagentReading {
	return history.JudgeSubagent(raw, history.SubagentAsk{
		File: file, RunDir: loop.RunRelDir, BriefFile: loop.BriefFileName, IsWrite: subagentWrites,
	})
}

// subagentHold refuses a sub-agent's write until the sub-agent has read the
// conventions file, and answers nil for every other call. The caller has found
// that the call writes.
func subagentHold(in guardHookInput, cwd string) (err error) {
	session, agent := in.SessionID, string(in.AgentID)
	if agent == "" || stageActive() || coldReader(string(in.AgentType)) {
		return nil
	}
	if !subagentKey(session, agent) {
		return nil // nothing to keep a fact under: never judged, never held
	}
	unjudged := func(why string) error {
		_ = rules.WriteSubagentFact(session, agent, rules.SubagentUnjudged, rules.SubagentRecord{Reason: why})
		return nil
	}
	// A panic exits 2, which the host reads as a block.
	defer func() {
		if r := recover(); r != nil {
			err = unjudged(unjudgedFault)
		}
	}()
	for _, f := range []rules.SubagentFact{rules.SubagentPassed, rules.SubagentUnjudged} {
		switch _, found, err := rules.ReadSubagentFact(session, agent, f); {
		case err != nil:
			return unjudged(unjudgedFacts)
		case found:
			return nil
		}
	}
	told, found, err := rules.ReadSubagentFact(session, agent, rules.SubagentTold)
	if err != nil {
		return unjudged(unjudgedFacts)
	}
	if !found {
		if guardGitStalls(cwd) {
			if !looksManaged(cwd) {
				return nil
			}
			return unjudged(unjudgedGit)
		}
		ctx, cancel := context.WithTimeout(context.Background(), guardGitBudget)
		defer cancel()
		// What the helpers have to say is the guard's own route to say, once.
		c, cerr := subagentConventions(ctx, cwd, new(bytes.Buffer))
		switch {
		case cerr != nil:
			return unjudged(unjudgedRules)
		case !c.Applies || c.File == "":
			return nil
		}
		told = rules.SubagentRecord{File: c.File, Root: c.Root, By: rules.SubagentToldByGuard}
		if err := rules.WriteSubagentFact(session, agent, rules.SubagentTold, told); err != nil {
			return unjudged(unjudgedUnsayable)
		}
	}
	if told.File == "" {
		return nil
	}
	transcript, ok := subagentTranscriptPath(string(in.TranscriptPath), agent)
	if !ok {
		return unjudged(unjudgedNoPath)
	}
	raw, err := readTranscript(transcript)
	if err != nil {
		return unjudged(unjudgedTranscript)
	}
	switch got := judgeSubagentTranscript(raw, told.File); {
	case got.ReadFile:
		_ = rules.WriteSubagentFact(session, agent, rules.SubagentPassed, rules.SubagentRecord{Reason: passedRead})
		return nil
	case got.ReadBrief:
		_ = rules.WriteSubagentFact(session, agent, rules.SubagentPassed, rules.SubagentRecord{Reason: passedBriefed})
		return nil
	}
	_ = rules.WriteSubagentFact(session, agent, rules.SubagentHeld, rules.SubagentRecord{})
	return denyCall(subagentHoldReason(told))
}

// subagentTranscriptPath is where the host keeps a sub-agent's transcript,
// beside the session's that its payload names.
func subagentTranscriptPath(session, agent string) (string, bool) {
	if !filepath.IsAbs(session) || !strings.HasSuffix(session, harnessTranscriptSuffix) {
		return "", false
	}
	return filepath.Join(strings.TrimSuffix(session, harnessTranscriptSuffix), subagentTranscripts,
		"agent-"+agent+harnessTranscriptSuffix), true
}

// subagentHoldReason is the refusal: the file, where abcd looked for it, and
// the way out.
func subagentHoldReason(told rules.SubagentRecord) string {
	looked := ""
	if root, ok := plainPath(told.Root); ok {
		looked = fmt.Sprintf(" (abcd looked in %s)", root)
	}
	return fmt.Sprintf("Blocked by the abcd guard (unread conventions): this sub-agent has not read %[1]s, the repository's conventions, and its writes wait until it has. "+
		"Run instead: read %[1]s at the root of the worktree you work in%[2]s, then send this write again. "+
		"Any call of a tool that does not write and names the file passes: the host's reading tool, `cat %[1]s` in the shell, or a search for it. "+
		"Make the read a call of its own: one in the same shell line as the held command is not counted, and one sent in the same turn as the write may not be.",
		told.File, looked)
}
