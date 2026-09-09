package cli

// The SubagentStop hook: the front door through which a finished sub-agent's
// transcript enters the store.
//
// It is built to the same shape as `hook session-end` — fail-closed, always
// exit 0, diagnostics on stderr, stdout empty — with one difference that is not
// cosmetic. SessionEnd's exit code is ignored by contract. SubagentStop's is
// NOT: the event is BLOCKING, so a non-zero exit prevents the sub-agent from
// stopping. Every path here therefore degrades to a diagnostic and returns nil.
// The always-exit-0 rule is load-bearing, not tidiness.
//
// The hook STAGES; it does not capture. Redaction costs roughly 0.7s per MB and
// this event fires inside a live session, where a stall is felt directly. The
// next SessionStart drains the staged copy through the unchanged, fail-closed
// Capture.

import (
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/intentdriven/abcd/internal/core/ahoy"
	"github.com/intentdriven/abcd/internal/core/history"
	"github.com/intentdriven/abcd/internal/fsutil"
	"github.com/intentdriven/abcd/internal/termsafe"
)

// idScalarRe mirrors the store's own agent/session id charset. The store
// validates these too and would refuse a bad one — but its refusal would sink
// the whole stage, and losing a transcript over an unusable lineage detail is
// the wrong trade. A scalar that fails here is dropped and the record says so
// through its attribution rung.
var idScalarRe = regexp.MustCompile(`^[A-Za-z0-9._-]+$`)

// harnessTranscriptSuffix and harnessSidecarSuffix derive the harness's own
// per-agent metadata file from the transcript path the payload gives us, by
// SUBSTITUTING the extension. Never by walking a directory: a design that reads
// the harness's on-disk layout is one the harness can silently break, and the
// whole reason this event is usable is that it hands over the path itself.
const (
	harnessTranscriptSuffix = ".jsonl"
	harnessSidecarSuffix    = ".meta.json"
)

// maxHarnessSidecarBytes caps the harness sidecar read. It is a handful of
// short scalars; anything larger is not one.
const maxHarnessSidecarBytes = 64 << 10

// subagentSettleAttempts and subagentSettleInterval bound the stage-time wait
// for a transcript the harness may still be writing.
//
// Whether SubagentStop fires before the sub-agent's transcript is flushed is
// UNVERIFIED — the documentation suggests it may, the binary does not settle it,
// and measuring the rate is a separate step. This is the cheap mitigation, not
// the answer: a few short re-reads, then stage what is there and SAY the copy
// may be short. The total worst case is under a tenth of a second, and it
// happens OUTSIDE the staging lock, so a burst of simultaneous completions
// never serialises on it.
const (
	subagentSettleAttempts = 4
	subagentSettleInterval = 25 * time.Millisecond
)

// newSubagentStopCommand builds `abcd hook subagent-stop`.
func newSubagentStopCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "subagent-stop",
		Short: "SubagentStop: stage a finished sub-agent's transcript for the next session to redact and store",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			// Diagnostics go to stderr, out of band; stdout stays empty. A
			// SubagentStop hook's stdout is not a place to speak to the model.
			warn := func(format string, a ...any) error {
				fmt.Fprintf(cmd.ErrOrStderr(), "abcd history: "+format+"\n", a...)
				return nil // never non-zero: this event BLOCKS the sub-agent
			}

			in, err := readHookInput(cmd)
			if err != nil {
				return warn("unreadable SubagentStop payload (%v); staging nothing", err)
			}
			rootSHA, via := resolveSubagentStore(in)

			// The absent-field case comes before the store check has any
			// consequence, but after it has been attempted: the marker needs a
			// store to live in, and a harness that never delivers the field is
			// the thing the marker exists to name.
			if in.AgentTranscriptPath == "" {
				if rootSHA != "" {
					if err := history.NoteSubagentGap(rootSHA, in.Event); err != nil {
						fmt.Fprintf(cmd.ErrOrStderr(),
							"abcd history: could not record the sub-agent payload gap (%v)\n", err)
					}
				}
				return warn("this harness fired %s with no agent_transcript_path, so no sub-agent transcript can be captured here; `abcd history staged` reports this",
					orDefault(in.Event, "SubagentStop"))
			}
			if rootSHA == "" {
				return warn("cannot resolve the repository for this sub-agent — the payload's cwd %q did not resolve and no store has seen session %q; staging nothing",
					in.Cwd, in.SessionID)
			}
			if !idScalarRe.MatchString(in.AgentID) {
				// Without an agent id the stage has no key of its own, and it
				// would collide with the spawning session's own staged
				// transcript — overwriting the spine to save a branch.
				return warn("SubagentStop payload carries no usable agent_id (%q); staging nothing rather than filing this transcript as the session's own", in.AgentID)
			}

			raw, settled, err := readSettledTranscript(in.AgentTranscriptPath)
			if err != nil {
				return warn("%v; staging nothing", err)
			}
			if !settled {
				fmt.Fprintf(cmd.ErrOrStderr(),
					"abcd history: sub-agent %s: the transcript's last line was still incomplete after %d reads; staging what is on disk, which may be short\n",
					in.AgentID, subagentSettleAttempts)
			}

			meta := subagentStageMeta(in)
			res, err := history.Stage(rootSHA, meta, raw)
			if err != nil {
				return warn("staging sub-agent %s failed (%v); this transcript was not captured", in.AgentID, err)
			}
			if !res.Wrote {
				return warn("sub-agent %s already staged with identical bytes (no-op)", in.AgentID)
			}
			verb := "staged"
			if res.Replaced {
				verb = "re-staged"
			}
			// The agent id passed idScalarRe above, but the session id came
			// straight off the payload and this line goes to a terminal.
			fmt.Fprintf(cmd.ErrOrStderr(),
				"abcd history: %s sub-agent %s of session %s (%d bytes, repo resolved via %s, spawn %s); the next session redacts and stores it\n",
				verb, in.AgentID, termsafe.Sanitize(in.SessionID), res.Staged.Bytes, via, meta.Lineage.SpawnAttribution)
			return nil
		},
	}
}

// resolveSubagentStore answers which repository's store this sub-agent's
// transcript belongs in, and by which route.
//
// The cwd is tried first, exactly as `hook session-end` does. It is not enough
// on its own: a sub-agent given its own worktree records that worktree as its
// cwd, and the harness REMOVES the worktree when the agent stops — so the
// directory is frequently gone by the time this hook runs, and the agents that
// affects are the isolated implementation lanes, the ones whose transcripts are
// worth the most. The spawning session is the fallback: a store that has seen
// that session id knows which repository owns it.
//
// A resolution through the cwd also RECORDS the tie, so the session's later
// sub-agents can be resolved even if this one was the last to run in a real
// directory.
func resolveSubagentStore(in hookInput) (rootSHA, via string) {
	cwd := in.Cwd
	if cwd == "" {
		if wd, err := os.Getwd(); err == nil {
			cwd = wd
		}
	}
	if det, err := ahoy.Detect(cwd); err == nil && det.RootSHA != "" {
		if in.SessionID != "" {
			// Best effort: a failure here degrades a fallback, never a capture.
			_ = history.NoteSessionRepo(det.RootSHA, in.SessionID)
		}
		return det.RootSHA, "cwd"
	}
	if in.SessionID != "" {
		if sha, err := history.SessionRepo(in.SessionID); err == nil {
			return sha, "session"
		}
	}
	return "", ""
}

// subagentStageMeta assembles the lineage for one sub-agent stage, running the
// attribution ladder's first rung.
//
// Rung 1 is the harness's own per-agent sidecar, which carries the spawn depth,
// the tool call that launched the agent, and — when the spawner was itself an
// agent — the parent's id. Its ABSENCE of a parent at depth 1 is information,
// not a gap: that agent was spawned by the main thread.
//
// Rung 3, unattributed, is what a missing or unusable sidecar leaves. It is a
// recorded value rather than an empty one because "spawned by the main thread"
// and "spawn point unknown" would otherwise be the same empty fields.
//
// Rung 2 — the spawning transcript's own tool result — is resolved at
// reconstruction time, not here.
func subagentStageMeta(in hookInput) history.StageMeta {
	lineage := history.CaptureMeta{
		SessionID:        in.SessionID,
		Kind:             "native",
		AgentID:          in.AgentID,
		AgentType:        safeTextScalar(in.AgentType),
		LineageSource:    "hook",
		SpawnAttribution: "unattributed",
	}
	if side, ok := readHarnessAgentSidecar(in.AgentTranscriptPath); ok {
		lineage.SpawnAttribution = "sidecar"
		lineage.SpawnDepth = side.SpawnDepth
		lineage.SpawnToolUseID = safeTextScalar(side.ToolUseID)
		if idScalarRe.MatchString(side.ParentAgentID) {
			lineage.ParentAgentID = side.ParentAgentID
		}
		if lineage.AgentType == "" {
			lineage.AgentType = safeTextScalar(side.AgentType)
		}
	}
	return history.StageMeta{Lineage: lineage, SourcePath: in.AgentTranscriptPath}
}

// safeTextScalar drops a value that would forge a record frontmatter field. The
// store refuses a line break in a scalar, and its refusal would sink the whole
// stage; dropping one decorative field is the better trade.
func safeTextScalar(s string) string {
	if strings.ContainsAny(s, "\r\n") {
		return ""
	}
	return s
}

// harnessAgentSidecar is the subset of the harness's per-agent metadata file
// this hook reads. Unknown fields are ignored, and every field is optional: the
// file is written by the host, not by abcd, and its shape is the host's to
// change.
type harnessAgentSidecar struct {
	AgentType     string `json:"agentType"`
	ParentAgentID string `json:"parentAgentId"`
	SpawnDepth    int    `json:"spawnDepth"`
	ToolUseID     string `json:"toolUseId"`
}

// readHarnessAgentSidecar reads the sidecar beside a sub-agent transcript.
//
// The read is guarded and best-effort in every direction: a transcript path
// that is not a .jsonl, an absent file, an unreadable one, a non-regular one,
// one that does not parse, and one that carries no spawn depth all return
// false. None of them is an error — they mean the first rung did not answer,
// which the caller records as such.
func readHarnessAgentSidecar(transcriptPath string) (harnessAgentSidecar, bool) {
	if !strings.HasSuffix(transcriptPath, harnessTranscriptSuffix) {
		return harnessAgentSidecar{}, false
	}
	path := strings.TrimSuffix(transcriptPath, harnessTranscriptSuffix) + harnessSidecarSuffix
	data, err := fsutil.ReadGuarded(path, maxHarnessSidecarBytes)
	if err != nil {
		return harnessAgentSidecar{}, false
	}
	var side harnessAgentSidecar
	if err := json.Unmarshal(data, &side); err != nil {
		return harnessAgentSidecar{}, false
	}
	// A sub-agent is at depth 1 or deeper by definition. A sidecar that does not
	// say so is not one this rung can read, whatever else it contains.
	if side.SpawnDepth <= 0 {
		return harnessAgentSidecar{}, false
	}
	return side, true
}

// readSettledTranscript reads a sub-agent transcript, re-reading a bounded
// number of times while its final line is still an incomplete JSON value.
//
// The wait lives here, in the surface, and NOT inside history.Stage: Stage's
// critical section is under a staging lock whose 5s timeout was tuned for a
// single SessionEnd, and many sub-agents can finish at once. A wait inside that
// lock would turn a burst of completions into a queue and refuse the slowest
// with a contention error.
//
// The returned bool says whether the transcript ever settled. A false is not an
// error — the bytes are still staged — but it is the signal the flush-race
// measurement counts.
func readSettledTranscript(path string) ([]byte, bool, error) {
	var raw []byte
	for attempt := range subagentSettleAttempts {
		var err error
		raw, err = readTranscript(path)
		if err != nil {
			return nil, false, err
		}
		if history.TranscriptSettled(raw) {
			return raw, true, nil
		}
		if attempt < subagentSettleAttempts-1 {
			time.Sleep(subagentSettleInterval)
		}
	}
	return raw, false, nil
}
