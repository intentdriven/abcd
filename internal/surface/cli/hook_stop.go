package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"regexp"
	"strconv"
	"time"

	"github.com/intentdriven/abcd/internal/core/autonomy"
	"github.com/intentdriven/abcd/internal/fsutil"
	"github.com/intentdriven/abcd/internal/termsafe"
	"github.com/spf13/cobra"
)

// hook_stop.go — `abcd hook stop` and `abcd hook stop-failure`, the front
// doors onto the away mode's stop check (spc-2610180914469489 step 5).
//
// Every path is fail-open and loud, the guard's contract: a malformed
// payload, an unreadable record or session state, an owed set that could not
// be read, or the away store's lock not had within its two seconds allows the
// stop and says so on stderr. A stop is never blocked on abcd's own fault.

// stopOwedReader is the seam onto the owed reader (spc-2610180914469489 step
// 4): autonomy.ReadOwed, asked with the session's working copy (its start
// copy, from the session-start note), the away record (not holding, the
// reader reads disk only and never calls the forge), the home the away store
// lives under, and the clock. While away its forge listing runs under its own
// ten-second deadline. A test replaces it with constructed sets.
var stopOwedReader = autonomy.ReadOwed

// stopClock is the seam onto the hook's clock and the zone it reads the
// session's task schedules in: the machine's own, time.Now and nil (the
// local zone, as the host reads its tasks). A test pins both, so a wake it
// asserts reads the same minutes whatever the machine's zone and whatever
// hour the test runs in, a daylight-saving change's included
// (iss-2610250107576049).
var stopClock = func() (time.Time, *time.Location) { return time.Now(), nil }

// stopReadBudget bounds the owed read, counted from the hook's start (the
// checkout's root and root commit included), its forge listing's own
// ten-second deadline inside it. Judging, the session's state and at most a
// few store writes under their two-second lock follow, so the hook ends
// within about 20 seconds, well inside the host's 30-second timeout. A read
// the budget cuts short is incomplete, and the stop is allowed, naming what
// was not read (spc-2610180914469489, "Diff review round 1", major 6).
const stopReadBudget = 14 * time.Second

// stopBlockCapEnv names the host's setting that raises its continuation cap.
const stopBlockCapEnv = "CLAUDE_CODE_STOP_HOOK_BLOCK_CAP"

// stopPayload is the Stop and StopFailure payload, the fields the stop check
// reads.
type stopPayload struct {
	SessionID            string                    `json:"session_id"`
	Cwd                  string                    `json:"cwd"`
	StopHookActive       bool                      `json:"stop_hook_active"`
	LastAssistantMessage string                    `json:"last_assistant_message"`
	BackgroundTasks      []autonomy.BackgroundTask `json:"background_tasks"`
	SessionCrons         []autonomy.SessionCron    `json:"session_crons"`
	// Error is StopFailure's error type (rate_limit, overloaded, …).
	Error string `json:"error"`
}

func readStopPayload(cmd *cobra.Command) (stopPayload, error) {
	raw, err := io.ReadAll(io.LimitReader(cmd.InOrStdin(), maxHookStdinBytes+1))
	if err != nil {
		return stopPayload{}, err
	}
	if len(raw) > maxHookStdinBytes {
		return stopPayload{}, fmt.Errorf("payload is over the %d-byte cap; it was discarded unparsed", maxHookStdinBytes)
	}
	var p stopPayload
	if err := json.Unmarshal(raw, &p); err != nil {
		return stopPayload{}, err
	}
	if p.Cwd == "" {
		if wd, err := os.Getwd(); err == nil {
			p.Cwd = wd
		}
	}
	return p, nil
}

// stopAllow names on stderr why the stop is allowed, and allows it.
func stopAllow(cmd *cobra.Command, format string, a ...any) error {
	diagnosticLine(cmd.ErrOrStderr(), "abcd stop: "+format+"; the stop is allowed", a...)
	return nil
}

func newHookStopCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "stop",
		Short: "Stop: keep a session working while the person is away and work is owed",
		Long: `The Stop hook's front door onto the away mode's stop check.

While the away record holds (abcd mode autonomous), a turn's end is blocked
while work is owed, naming the next item, and is allowed only with a
session cron whose latest fire time, the host's lateness included, falls
before the deadline: within half an hour while work is owed (an hour when
none is) and before the session's reservations need renewing. A wake
releases an unchanged set of work the session can do itself at most twice
in a row, and a last message ending on a question is blocked once. Plain
stdout is then the host's non-error Stop feedback,
{"hookSpecificOutput":{"hookEventName":"Stop","additionalContext":...}},
which continues the session as a block does; the block passing the
host's cap of consecutive continuations (eight, or
CLAUDE_CODE_STOP_HOOK_BLOCK_CAP) is recorded in the away record as a stop
possibly forced by the cap. While the person is present the check reads disk
only and, with work owed, allows the stop with a systemMessage naming it.
The owed work is read within 14 seconds of the hook's start, so the hook
ends well inside the host's 30-second timeout; a reading cut short allows
the stop with a systemMessage naming what was not read, recorded in the away
record while the person is away.

The session's state is kept per session id in its start copy's local tier,
.abcd/.work.local/autonomy/sessions/<session-id>.json. A malformed payload,
an unreadable record or a lock not had allows the stop, naming why on
stderr; every outcome exits 0. Outside a checkout abcd manages (no .abcd/
at its root, nor at its main worktree's) it reads nothing and prints
nothing.`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) (ret error) {
			// A panic allows the stop: exit 2 is the host's block on this
			// event, and a stop is never blocked on abcd's own fault.
			defer func() {
				if r := recover(); r != nil {
					ret = stopAllow(cmd, "the stop check failed (%s)", termsafe.CleanProseLine(fsutil.RedactHome(fmt.Sprint(r)), 256))
				}
			}()
			ctx, cancel := context.WithTimeout(context.Background(), stopReadBudget)
			defer cancel()
			p, err := readStopPayload(cmd)
			if err != nil {
				return stopAllow(cmd, "unreadable Stop payload (%s)", termsafe.Sanitize(err.Error()))
			}
			root, ok := managedRoot(p.Cwd)
			if !ok {
				return nil // not a checkout abcd manages: nothing to check, and no git spent
			}
			copyRoot, st, err := autonomy.ResolveCopy(root, p.SessionID, autonomy.Options{})
			if err != nil {
				return nil // not a checkout abcd can key a store on: nothing to check
			}
			now, loc := stopClock()
			rec, away, err := st.Away(now)
			if err != nil {
				return stopAllow(cmd, "%s; read as not away", fsutil.RedactHome(err.Error()))
			}
			if away && !autonomy.ValidSessionID(p.SessionID) {
				return stopAllow(cmd, "the session id %q is not [A-Za-z0-9_-]{1,128}", termsafe.CleanProseLine(p.SessionID, 64))
			}
			home, _ := os.UserHomeDir()
			owed, err := stopOwedReader(ctx, autonomy.ReadOptions{Root: copyRoot, Record: rec, Home: home, Now: func() time.Time { return now }})
			if err != nil {
				return stopAllow(cmd, "owed work could not be read (%s)", fsutil.RedactHome(err.Error()))
			}
			in := autonomy.StopInput{
				SessionID: p.SessionID, StopHookActive: p.StopHookActive, LastAssistantMessage: p.LastAssistantMessage,
				SessionCrons: p.SessionCrons, BackgroundTasks: p.BackgroundTasks, Cap: stopBlockCap(cmd.ErrOrStderr()),
				Location: loc,
			}
			var prev autonomy.SessionState
			if away {
				if prev, err = autonomy.LoadSession(copyRoot, p.SessionID); err != nil {
					return stopAllow(cmd, "the session's state could not be read (%s)", fsutil.RedactHome(err.Error()))
				}
			}
			v := autonomy.Judge(rec, owed, in, prev, now)
			if v.Save {
				if err := autonomy.SaveSession(copyRoot, p.SessionID, v.State); err != nil {
					return stopAllow(cmd, "the session's state could not be saved (%s)", fsutil.RedactHome(err.Error()))
				}
			}
			for _, h := range v.HandBacks {
				if _, err := st.RecordHandBack(h.Branch, h.Head, now); err != nil {
					return stopAllow(cmd, "the hand-back of %s was not recorded (%s)", termsafe.CleanProseLine(h.Branch, 128), stopStoreWords(err))
				}
			}
			if v.Stop != nil {
				if _, err := st.RecordStop(*v.Stop, now); err != nil {
					return stopAllow(cmd, "the stop was not recorded (%s)", stopStoreWords(err))
				}
			}
			return emitStop(cmd, v)
		},
	}
}

// stopStoreWords names an away-store failure, the lock's included.
func stopStoreWords(err error) string {
	if errors.Is(err, autonomy.ErrLockBusy) {
		return "the away store's lock was not had in time: " + fsutil.RedactHome(err.Error())
	}
	return fsutil.RedactHome(err.Error())
}

// stopBlockCap is the host's continuation cap as the environment sets it;
// 0 (the default of eight) when it is unset or not a positive number.
func stopBlockCap(stderr io.Writer) int {
	raw := os.Getenv(stopBlockCapEnv)
	if raw == "" {
		return 0
	}
	n, err := strconv.Atoi(raw)
	if err != nil || n <= 0 {
		diagnosticLine(stderr, "abcd stop: %s is set to %s, which is not a positive number; counting to the host's default of %d",
			stopBlockCapEnv, termsafe.DescribeRefused(raw), autonomy.DefaultBlockCap)
		return 0
	}
	return n
}

// stopBlockOutput is a block written as the host's non-error feedback for a
// Stop hook: hookSpecificOutput.additionalContext keeps the session going
// under the same loop protections as {"decision":"block"} (stop_hook_active
// and the continuation cap), and the host labels it "Stop hook feedback"
// rather than showing a hook error, its documented form for a hook working as
// designed (code.claude.com/docs/en/hooks, "Stop decision control").
type stopBlockOutput struct {
	HookSpecificOutput stopBlockFields `json:"hookSpecificOutput"`
}

type stopBlockFields struct {
	HookEventName     string `json:"hookEventName"`
	AdditionalContext string `json:"additionalContext"`
}

type stopMessageOutput struct {
	SystemMessage string `json:"systemMessage"`
}

// emitStop writes the host's answer: the block object, the person's line, or
// nothing.
func emitStop(cmd *cobra.Command, v autonomy.Verdict) error {
	var obj any
	switch {
	case v.Block:
		obj = stopBlockOutput{HookSpecificOutput: stopBlockFields{HookEventName: "Stop", AdditionalContext: v.Reason}}
	case v.SystemMessage != "":
		obj = stopMessageOutput{SystemMessage: termsafe.CleanProseLine(v.SystemMessage, 1024)}
	default:
		return nil
	}
	var b bytes.Buffer
	enc := json.NewEncoder(&b)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(obj); err != nil {
		return stopAllow(cmd, "the answer could not be written (%s)", err.Error())
	}
	if _, err := io.Copy(cmd.OutOrStdout(), &b); err != nil {
		diagnosticLine(cmd.ErrOrStderr(), "abcd stop: the answer could not be written (%s)", err.Error())
	}
	return nil
}

// stopErrorRe is the shape of the host's error types; anything else is
// recorded as unknown.
var stopErrorRe = regexp.MustCompile(`^[a-z][a-z0-9_]{0,63}$`)

func newHookStopFailureCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "stop-failure",
		Short: "StopFailure: record a turn the host ended on an API error while the person is away",
		Long: `The StopFailure hook's front door. While the away record holds, a turn the
host ended on an API error is recorded in it as "stopped on an API error"
with the error type, under the store's lock; abcd mode back prints it. The
host ignores this hook's output and it cannot continue the session. While
the person is present, or outside a checkout abcd manages, nothing is
written. Every outcome exits 0.`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) (ret error) {
			defer func() {
				if r := recover(); r != nil {
					diagnosticLine(cmd.ErrOrStderr(), "abcd stop: the API-error stop was not recorded (%s)", termsafe.CleanProseLine(fsutil.RedactHome(fmt.Sprint(r)), 256))
					ret = nil
				}
			}()
			p, err := readStopPayload(cmd)
			if err != nil {
				diagnosticLine(cmd.ErrOrStderr(), "abcd stop: unreadable StopFailure payload (%s); nothing recorded", termsafe.Sanitize(err.Error()))
				return nil
			}
			root, ok := managedRoot(p.Cwd)
			if !ok {
				return nil
			}
			_, st, err := autonomy.ResolveCopy(root, p.SessionID, autonomy.Options{})
			if err != nil {
				return nil
			}
			now, _ := stopClock()
			_, away, err := st.Away(now)
			if err != nil {
				diagnosticLine(cmd.ErrOrStderr(), "abcd stop: %s; read as not away, nothing recorded", fsutil.RedactHome(err.Error()))
				return nil
			}
			if !away {
				return nil
			}
			kind := p.Error
			if !stopErrorRe.MatchString(kind) {
				kind = "unknown"
			}
			if _, err := st.RecordStop(autonomy.Stop{Kind: autonomy.StopAPIError, ErrorType: kind}, now); err != nil {
				diagnosticLine(cmd.ErrOrStderr(), "abcd stop: the API-error stop was not recorded (%s)", stopStoreWords(err))
			}
			return nil
		},
	}
}
