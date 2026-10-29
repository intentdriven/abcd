package cli

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/intentdriven/abcd/internal/core/ahoy"
	"github.com/intentdriven/abcd/internal/core/history"
	"github.com/intentdriven/abcd/internal/fsutil"
	"github.com/intentdriven/abcd/internal/termsafe"
)

// runHookSessionEnd is `abcd hook session-end`: stage the raw transcript by
// session id, with no repository lookup (iss-2610241519088123).
//
// The host gives a SessionEnd hook 1.5 seconds in all, and a timeout a
// plugin's hooks.json sets does not raise it (the host's hooks reference). On
// a loaded machine the binary's own start takes a third of that, and finding
// the repository took a detection and a root-commit lookup through git on top,
// retried when git failed once, so the host cancelled the hook before it
// staged anything and the transcript was lost unseen. So this runs no git at
// all: it reads the transcript and writes it into the machine-level pending
// directory with the payload's cwd beside it. The next session start finds
// the store and files it there (placePendingNotices), and the drain redacts
// and stores it.
func runHookSessionEnd(cmd *cobra.Command, _ []string) error {
	// Diagnostics go to stderr, out of band; stdout stays empty, since a
	// SessionEnd hook's stdout is not a place to speak to the model — unless
	// the caller asked for --json, when it carries one result line on every
	// path (hook_result.go, iss-2608261550596333). The result names the
	// session it lost, as subagent-stop's does (iss-2609260221577624): in is
	// read below, and warn names whatever of it was parsed — nothing, for a
	// payload that did not parse.
	var in hookInput
	warn := func(format string, a ...any) error {
		msg := diagnosticLine(cmd.ErrOrStderr(), "abcd history: "+format, a...)
		emitHookResult(cmd, hookStageResult{Hook: "session-end", Outcome: hookOutcomeNotCaptured,
			SessionID: termsafe.Sanitize(in.SessionID), Reason: strings.TrimPrefix(msg, "abcd history: ")})
		return nil // never non-zero: a SessionEnd hook must not wedge the session
	}

	parsed, err := readHookInput(cmd)
	if err != nil {
		return warn("unreadable SessionEnd payload (%v); capturing nothing", err)
	}
	in = parsed
	// The session no longer works in its checkout: one file remove, so it
	// fits the hook's short budget (hook_occupancy.go).
	forgetOccupancy(in)
	if in.TranscriptPath == "" {
		return warn("SessionEnd payload carries no transcript_path; capturing nothing")
	}
	cwd := in.Cwd
	if cwd == "" {
		if wd, err := os.Getwd(); err == nil {
			cwd = wd
		}
	}
	raw, err := readTranscript(in.TranscriptPath)
	if err != nil {
		return warn("%v; capturing nothing", err)
	}
	res, err := history.StagePending(history.PendingMeta{SessionID: in.SessionID, Cwd: cwd, SourcePath: in.TranscriptPath}, raw)
	if err != nil {
		// The marker is where the person sees the loss: this hook exits 0,
		// and its stderr reaches only the host's debug log. The next session
		// start reports it.
		if nerr := history.NotePendingMiss("session "+in.SessionID, err.Error()); nerr != nil {
			diagnosticLine(cmd.ErrOrStderr(), "abcd history: the missed stage could not be recorded either (%v)", nerr)
		}
		return warn("staging failed (%v); this session was not captured, and the next session start reports it", err)
	}
	staged := hookStageResult{Hook: "session-end", Captured: true, SessionID: res.Staged.SessionID, Bytes: res.Staged.Bytes}
	switch {
	case !res.Wrote:
		fmt.Fprintf(cmd.ErrOrStderr(), "abcd history: session %s already staged with identical bytes (no-op)\n", res.Staged.SessionID)
		staged.Outcome = hookOutcomeAlreadyStaged
	case res.Replaced:
		// Different bytes for an already-staged session: the later snapshot
		// replaced the earlier one, and this says so rather than reporting a
		// no-op that would hide a replaced transcript.
		fmt.Fprintf(cmd.ErrOrStderr(), "abcd history: re-staged %s (%d bytes), replacing %d stale bytes; the next session files, redacts and stores it\n",
			res.Staged.SessionID, res.Staged.Bytes, res.ReplacedBytes)
		staged.Outcome, staged.ReplacedBytes = hookOutcomeRestaged, res.ReplacedBytes
	default:
		fmt.Fprintf(cmd.ErrOrStderr(), "abcd history: staged %s (%d bytes); the next session files, redacts and stores it\n",
			res.Staged.SessionID, res.Staged.Bytes)
		staged.Outcome = hookOutcomeStaged
	}
	emitHookResult(cmd, staged)
	return nil
}

// placePendingNotices files every transcript the session-end hook left pending
// in its store's staging lane, before this start's drain runs, and returns the
// notices for what it could not place and for any session-end stage that
// failed outright. Silent when everything was placed: placing is not news,
// storing is, and the drain reports that.
func placePendingNotices() []string {
	var notices []string
	if miss, ok, err := history.TakePendingMiss(); err != nil {
		notices = append(notices, fmt.Sprintf("abcd history: the record of session transcripts missed at session end could not be read (%s).",
			termsafe.Sanitize(fsutil.RedactHome(err.Error()))))
	} else if ok {
		notices = append(notices, fmt.Sprintf("abcd history: %d session transcript(s) could not be staged at session end and are lost; the last was %s (%s).",
			miss.Count, termsafe.Sanitize(miss.LastWho), termsafe.Sanitize(fsutil.RedactHome(miss.LastReason))))
	}
	res, err := history.PlacePending(resolveCwdRepo)
	if err != nil {
		return append(notices, fmt.Sprintf("abcd history: transcripts staged at session end could not be listed (%s); they wait for the next start.",
			termsafe.Sanitize(fsutil.RedactHome(err.Error()))))
	}
	for _, d := range res.Dropped {
		notices = append(notices, fmt.Sprintf("abcd history: session %s ended in %s, which is in no repository with a commit, so its transcript was not captured and its raw copy was deleted.",
			termsafe.Sanitize(d.SessionID), termsafe.Sanitize(fsutil.RedactHome(d.Cwd))))
	}
	for _, u := range res.Unplaced {
		notices = append(notices, fmt.Sprintf("abcd history: the transcript of session %s, which ended in %s, was not filed in any store: %s. It stays UNREDACTED at %s until a session start can place it, or `abcd history discard --pending <file> --yes` deletes it.",
			termsafe.Sanitize(u.SessionID), termsafe.Sanitize(fsutil.RedactHome(u.Cwd)),
			termsafe.Sanitize(fsutil.RedactHome(u.Reason)), termsafe.Sanitize(fsutil.RedactHome(u.Path))))
	}
	return notices
}

// resolveCwdRepo is placement's RepoResolver: the repository a pending
// transcript's cwd belongs to, found the way every hook finds it.
func resolveCwdRepo(cwd string) (string, string, error) {
	if _, err := os.Stat(cwd); err != nil {
		return "", "", fmt.Errorf("its working directory is gone (%v)", err)
	}
	det, err := ahoy.Detect(cwd)
	if err != nil {
		return "", "", err
	}
	if det.RootSHA == "" {
		if rerr := det.RepoIdentity.RootErr(); rerr != nil {
			return "", "", fmt.Errorf("git gave no root commit for its working directory (%v)", rerr)
		}
		// git answered: no repository here, or one with no commit yet.
		return "", "", fmt.Errorf("its working directory is %w", history.ErrNoRepository)
	}
	return captureRoot(cwd), det.RootSHA, nil
}
