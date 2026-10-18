package cli

// The SubagentStop hook: the front door through which a finished sub-agent's
// transcript enters the store.
//
// It is built to the same shape as `hook session-end` — fail-closed, always
// exit 0, diagnostics on stderr, stdout empty — with one difference that is not
// cosmetic. SessionEnd's exit code is ignored by contract. SubagentStop's is
// NOT: exit code 2 is the host's BLOCKING status there, and it would stop the
// sub-agent from finishing. Exit 2 is the ONLY blocking code; a general
// non-zero exit is a visible diagnostic, not a stall.
//
// Every path here therefore returns nil, because a Go error returned from RunE
// is what would put a code this hook must never emit on the table, and no
// staging failure is worth risking the agent's completion over. That is a rule
// about THIS binary's own exits, not about the launcher's: the `hooks.json`
// wrapper's `exit 1` when no binary resolves is deliberately non-zero and
// deliberately not 2 — it is the one signal a user gets that their transcripts
// are not being captured, and silencing it to 0 would hide a real gap.
// `TestSubagentStopNeverBootstraps` pins exactly that: not 2, not 0, not 127.
//
// The hook STAGES; it does not capture. Redaction costs roughly 0.7s per MB and
// this event fires inside a live session, where a stall is felt directly. The
// next SessionStart drains the staged copy through the unchanged, fail-closed
// Capture.

import (
	"errors"
	"fmt"
	"io"
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

// subagentResolveWaits are the pauses before each further attempt to resolve
// the repository when the payload's cwd carries a .git entry but git gave no
// root commit: a git that failed once, which on a loaded machine is a fork
// refused or a git killed early, never a directory outside any repository
// (iss-2610051434156079). About three and a half seconds in all.
var subagentResolveWaits = []time.Duration{100 * time.Millisecond, 400 * time.Millisecond, time.Second, 2 * time.Second}

// subagentStageRetries bounds how many times a stage refused by lock
// contention is tried again. Each try already waits out the staging lock's own
// timeout (history's five seconds), so three more give a held lock twenty
// seconds in all, well inside the host's hook timeout; past that the
// transcript is marked as missed rather than dropped unseen.
var subagentStageRetries = 3

// newSubagentStopCommand builds `abcd hook subagent-stop`.
func newSubagentStopCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "subagent-stop",
		Short: "SubagentStop: stage a finished sub-agent's transcript for the next session to redact and store",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			// Diagnostics go to stderr, out of band; stdout stays empty. A
			// SubagentStop hook's stdout is not a place to speak to the model.
			var in hookInput // read below; warn names whatever of it was parsed
			warn := func(format string, a ...any) error {
				msg := strings.TrimPrefix(diagnosticLine(cmd.ErrOrStderr(), "abcd history: "+format, a...), "abcd history: ")
				// Under --json, the one result line every path writes
				// (hook_result.go, iss-2608261550596333).
				emitHookResult(cmd, hookStageResult{Hook: "subagent-stop", Outcome: hookOutcomeNotCaptured,
					SessionID: termsafe.Sanitize(in.SessionID), AgentID: termsafe.Sanitize(in.AgentID), Reason: msg})
				return nil // never an error: exit 2 is this event's BLOCKING code
			}

			parsed, err := readHookInput(cmd)
			if err != nil {
				return warn("unreadable SubagentStop payload (%v); staging nothing", err)
			}
			in = parsed
			repoRoot, rootSHA, via, rootErr := resolveSubagentStore(in)

			// The absent-field case comes before the store check has any
			// consequence, but after it has been attempted: the marker needs a
			// store to live in, and a harness that never delivers the field is
			// the thing the marker exists to name.
			if in.AgentTranscriptPath == "" {
				if rootSHA != "" {
					if err := history.NoteSubagentGap(repoRoot, rootSHA, in.Event); err != nil {
						diagnosticLine(cmd.ErrOrStderr(),
							"abcd history: could not record the sub-agent payload gap (%v)", err)
					}
				}
				return warn("this harness fired %s with no agent_transcript_path, so no sub-agent transcript can be captured here; `abcd history staged` reports this",
					orDefault(in.Event, "SubagentStop"))
			}
			if rootSHA == "" && rootErr != nil {
				// git did not answer for the cwd's root commit (a fork refused
				// on a loaded machine, a git that exited early): that is a git
				// that failed this once, not a directory outside any
				// repository. It is asked again, and the session note re-read
				// between tries, since a burst's siblings write it as they
				// resolve (iss-2610051434156079).
				repoRoot, rootSHA, via = retrySubagentStore(cmd.ErrOrStderr(), in, rootErr)
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
			res, err := stageWithRetry(cmd.ErrOrStderr(), repoRoot, rootSHA, meta, raw)
			if err != nil {
				// The transcript was read and is lost. The marker is what lets
				// the person see that: this hook's stderr reaches only a debug
				// log, since it must exit 0.
				if nerr := history.NoteStageMiss(repoRoot, rootSHA, "sub-agent "+in.AgentID, err.Error()); nerr != nil {
					diagnosticLine(cmd.ErrOrStderr(), "abcd history: the missed stage could not be recorded either (%v)", nerr)
				}
				return warn("staging sub-agent %s failed (%v); this transcript was not captured, and `abcd history staged` reports it", in.AgentID, err)
			}
			staged := hookStageResult{Hook: "subagent-stop", Captured: true, Outcome: hookOutcomeStaged,
				SessionID: termsafe.Sanitize(in.SessionID), AgentID: in.AgentID, Bytes: res.Staged.Bytes}
			if !res.Wrote {
				fmt.Fprintf(cmd.ErrOrStderr(), "abcd history: sub-agent %s already staged with identical bytes (no-op)\n", in.AgentID)
				staged.Outcome = hookOutcomeAlreadyStaged
				emitHookResult(cmd, staged)
				return nil
			}
			verb := "staged"
			if res.Replaced {
				verb = "re-staged"
				staged.Outcome, staged.ReplacedBytes = hookOutcomeRestaged, res.ReplacedBytes
			}
			defer emitHookResult(cmd, staged)
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
// The repository ROOT it returns is the directory the store resolves the
// per-repo opt-in against, and it is empty on the session-note route by
// construction: that route exists precisely because the sub-agent's working
// directory is gone. A store this session opted into its own checkout is
// therefore unreachable from that route, which is why the session note is
// written into the resolved lane rather than a fixed path — a lane the lookup
// cannot enumerate is one it correctly reports as not having seen the session,
// instead of staging the transcript into a second store nothing drains.
//
// rootErr is why the cwd gave no root commit when git did not answer for it
// (ahoy.RepoIdentity.RootErr), nil when the cwd's answer was definite: it is
// the signal that a second try could find what the first did not.
func resolveSubagentStore(in hookInput) (repoRoot, rootSHA, via string, rootErr error) {
	cwd := in.Cwd
	if cwd == "" {
		if wd, err := os.Getwd(); err == nil {
			cwd = wd
		}
	}
	det, err := ahoy.Detect(cwd)
	if err == nil && det.RootSHA != "" {
		root := captureRoot(cwd)
		if in.SessionID != "" {
			// Best effort: a failure here degrades a fallback, never a capture.
			_ = history.NoteSessionRepo(root, det.RootSHA, in.SessionID)
		}
		return root, det.RootSHA, "cwd", nil
	}
	if err == nil {
		rootErr = det.RepoIdentity.RootErr()
	}
	if in.SessionID != "" {
		if sha, err := history.SessionRepo(in.SessionID); err == nil {
			return "", sha, "session", nil
		}
	}
	return "", "", "", rootErr
}

// retrySubagentStore asks again, after resolveSubagentStore found no store
// because git did not answer for the cwd's root commit (first, the failure the
// first try met): git failing once is a loaded machine, not a directory
// outside any repository, and a burst's siblings write the session note as
// they resolve, so it is re-read between tries. A cwd git answered for
// definitely — none, gone, or a repository with no commits — is never retried,
// since a second git call could find nothing there; the caller does not call
// it then. A try whose answer turns definite stops the retries the same way.
// Each retry is said on stderr.
func retrySubagentStore(w io.Writer, in hookInput, first error) (repoRoot, rootSHA, via string) {
	last := first
	for i, wait := range subagentResolveWaits {
		time.Sleep(wait)
		var rootErr error
		if repoRoot, rootSHA, via, rootErr = resolveSubagentStore(in); rootSHA != "" {
			fmt.Fprintf(w, "abcd history: sub-agent %s: the repository resolved on try %d, after git first gave no root commit for the cwd\n",
				termsafe.Sanitize(in.AgentID), i+2)
			return repoRoot, rootSHA, via
		}
		if rootErr == nil {
			fmt.Fprintf(w, "abcd history: sub-agent %s: on try %d git answered that the cwd has no root commit\n",
				termsafe.Sanitize(in.AgentID), i+2)
			return "", "", ""
		}
		last = rootErr
	}
	fmt.Fprintf(w, "abcd history: sub-agent %s: git gave no root commit for the cwd in %d tries (%s)\n",
		termsafe.Sanitize(in.AgentID), len(subagentResolveWaits)+1, termsafe.Sanitize(fsutil.RedactHome(last.Error())))
	return "", "", ""
}

// stageWithRetry stages, trying again when the staging lock was held past its
// timeout, so a stage that cannot take the lock waits longer instead of losing
// the transcript (iss-2610051434156079). Every other failure returns at once:
// it would fail the same way again.
func stageWithRetry(w io.Writer, repoRoot, rootSHA string, meta history.StageMeta, raw []byte) (history.StageResult, error) {
	who := "session " + termsafe.Sanitize(meta.Lineage.SessionID)
	if meta.Lineage.AgentID != "" {
		who = "sub-agent " + termsafe.Sanitize(meta.Lineage.AgentID)
	}
	res, err := history.Stage(repoRoot, rootSHA, meta, raw)
	for try := 0; err != nil && errors.Is(err, fsutil.ErrLockContention) && try < subagentStageRetries; try++ {
		fmt.Fprintf(w, "abcd history: %s: the staging lock was held past its timeout; trying again (%d of %d)\n",
			who, try+1, subagentStageRetries)
		res, err = history.Stage(repoRoot, rootSHA, meta, raw)
	}
	return res, err
}

// retryDetect is the session-end hook's retrySubagentStore: when git gave no
// root commit for a repository-shaped cwd, detection is asked again on the
// same bounded waits, and a try that turns definite (a root, or git answering
// that there is none) ends the retries. Each retry is said on stderr
// (iss-2610051434156079).
func retryDetect(w io.Writer, cwd string, first ahoy.DetectionResult) (ahoy.DetectionResult, error) {
	det := first
	for i, wait := range subagentResolveWaits {
		time.Sleep(wait)
		next, err := ahoy.Detect(cwd)
		if err != nil {
			return next, err
		}
		det = next
		if det.RootSHA != "" {
			fmt.Fprintf(w, "abcd history: the repository resolved on try %d, after git first gave no root commit for the cwd\n", i+2)
			return det, nil
		}
		if det.RepoIdentity.RootErr() == nil {
			return det, nil
		}
	}
	return det, nil
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
	return parseHarnessAgentSidecar(strings.TrimSuffix(transcriptPath, harnessTranscriptSuffix) + harnessSidecarSuffix)
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
