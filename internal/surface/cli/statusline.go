package cli

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"syscall"
	"time"

	"github.com/intentdriven/abcd/internal/abcdhome"
	"github.com/intentdriven/abcd/internal/core/ahoy"
	"github.com/intentdriven/abcd/internal/core/statusline"
	"github.com/intentdriven/abcd/internal/termsafe"
	"github.com/spf13/cobra"
)

// statuslineStore is the noun the checkout-root refusal is phrased with, for
// the board's resolution of the same checkout (board.go). The status verb
// resolves through statusline.CheckoutRoot, which is bounded and phrases no
// refusal: no checkout is simply "not managed".
const statuslineStore = "the status line"

// statuslineFallbackEnv marks the environment of the previous status command
// while this verb runs it. A run that finds it already set IS that previous
// command, or something it ran: the recorded command reaches abcd's own
// status verb — a hand-wired `abcd statusline`, a path detection did not
// recognise as abcd's — and running the previous command again from here
// would run this verb again, without end (871 nested processes in four
// seconds, measured 2026-09-15). Install refuses to record such a command
// (ahoy's statusVerbRe); this is the guard for a setting written by hand or
// by an older abcd, and it fails closed: the recursion is refused at depth
// one, whatever the previous command would have printed.
const statuslineFallbackEnv = "ABCD_STATUSLINE_FALLBACK"

// The status line's time bounds (iss-2610050556383525). The harness waits on
// this verb on every refresh of every session, so nothing it waits for may be
// unbounded: a hung git, a stalled filesystem or a stdin the harness never
// closes would otherwise freeze the person's status line for as long as the
// hang lasts.
//
// statuslineBudget is the hard ceiling on abcd's OWN work — reading the
// payload and the setting, resolving the checkout, deciding whether it is
// managed, composing the row. Half a second: the work is a few small file
// reads, a handful of directory listings and a few git questions, measured in
// tens of milliseconds, so the ceiling is never the cost of an ordinary
// refresh, only the cap on an extraordinary one. It is enforced by a
// watchdog: the work runs in a goroutine that never touches the streams, and
// at the ceiling the verb stops waiting for it, prints nothing and exits 0.
// What the goroutine was blocked in ends with the process.
//
// statuslineGitBudget is the share of that ceiling git may take, spent across
// all of the row's git questions (statusline.CheckoutRoot, the root commit
// ahoy.ManagedContext asks for when a checkout carries no marker block, and
// the branch). A git still running at it is KILLED rather than abandoned, and
// it ends early enough that the rest of the row still renders inside the
// ceiling: a slow branch costs the row its branch, not the row.
//
// previousCommandBudget bounds the person's own previous status command, run
// outside managed checkouts. It is the person's command, not abcd's work, so
// it gets its own and far more generous bound: five seconds lets a slow but
// working command (a package runner's first start) keep its line, and still
// ends a hang. At it the command's whole process group is killed, the output
// it already printed stands, and the verb exits 0. It is a variable only so
// a test can shorten it.
const (
	statuslineBudget    = 500 * time.Millisecond
	statuslineGitBudget = 300 * time.Millisecond
)

var previousCommandBudget = 5 * time.Second

// previousCommandWaitDelay is how long the verb waits, after killing the
// previous command's group, for the pipes it shared to close.
const previousCommandWaitDelay = 100 * time.Millisecond

// newStatuslineCommand builds the `statusline` verb — the harness-facing shell
// around internal/core/statusline's one render (spc-70). The harness runs it
// on every status refresh with its JSON status payload on stdin, and shows
// whatever it prints on the status line.
//
// In a managed checkout it prints abcd's row: the badge first, then the
// repository, the branch, and the payload and record elements the setting
// leaves on. Anywhere else, or with the setting's off switch thrown, it runs
// the user's PREVIOUS status command — recorded at install — with the same
// stdin, and its output and exit code are this verb's (ac-5, ac-6). That is
// how one harness-wide setting yields abcd's line in managed repositories and
// the user's own line everywhere else. With nothing recorded it prints
// nothing and exits 0.
//
// It never prompts, never reads a terminal, never touches the network, and
// writes nothing (TestStatuslineWritesNothing and
// TestStatusVerbReachesNoWritePrimitive hold it). Every step it waits on is
// bounded (statuslineBudget and its siblings above): at a bound it prints
// what it has — a row without the element that did not answer, the previous
// command's output so far, or nothing — says why on stderr, and exits 0. A payload that does not parse, or is over the cap, is
// named on stderr and the row still renders without its payload elements: a
// status surface that goes blank on a bad payload hides the parked stop the
// badge exists to show.
func newStatuslineCommand(asJSON *bool) *cobra.Command {
	return &cobra.Command{
		Use: "statusline",
		Long: "Render abcd's row for the host harness's status line.\n\n" +
			"The harness runs this on every status refresh, with its JSON status\n" +
			"payload on stdin, and shows what it prints. In a checkout abcd manages the\n" +
			"row is abcd's own: the presence badge first — `abcd-managed`, `waiting on\n" +
			"the technical facilitator` or `waiting on the product thinker`, from the state\n" +
			"`abcd mode` stores, its colour ending at the badge — then the\n" +
			"repository name, the branch, the model, the context percentage, the\n" +
			"five-hour and seven-day usage percentages, and the record's counts of\n" +
			"intents not yet shipped and open issues. Each element after the badge is\n" +
			"switchable in `" + abcdhome.Display("statusline.json") + "`; a payload field the harness did not\n" +
			"supply drops its element with no placeholder.\n\n" +
			"Outside a managed checkout, or with `disabled` set in the user-level\n" +
			"setting, it runs the status command recorded there at install time with the\n" +
			"same stdin and passes its output and exit code through unchanged, so the\n" +
			"user's own line is untouched everywhere abcd does not manage. With none\n" +
			"recorded it prints nothing and exits 0.\n\n" +
			"The checkout is resolved from the payload's `cwd` (falling back to the\n" +
			"working directory). Empty stdin is an empty payload. Nothing here prompts,\n" +
			"reads a terminal, or touches the network. With --json the row is emitted as\n" +
			"its ordered elements, each with a key, a rendered and a plain form.",
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			stderr := cmd.ErrOrStderr()
			note := func(format string, a ...any) {
				fmt.Fprintf(stderr, "abcd statusline: "+format+"\n", a...)
			}

			ctx, cancel := context.WithTimeout(context.Background(), statuslineBudget)
			defer cancel()
			gitCtx, cancelGit := context.WithTimeout(ctx, statuslineGitBudget)
			defer cancelGit()

			// The watchdog. The decision is made in a goroutine that returns a
			// value and never writes a stream, so a decision abandoned at the
			// ceiling cannot print over the verb's answer afterwards; the
			// channel is buffered so the goroutine never blocks on a send
			// nobody is waiting for.
			done := make(chan statusDecision, 1)
			stdin := cmd.InOrStdin()
			go func() { done <- decideStatus(gitCtx, stdin) }()
			var d statusDecision
			select {
			case d = <-done:
			case <-ctx.Done():
				note("abcd's own work did not finish within %s, so the status line is empty for this refresh", statuslineBudget)
				return nil
			}

			for _, n := range d.notes {
				note("%s", termsafe.Sanitize(n))
			}
			switch {
			case d.err != nil:
				return d.err
			case d.blank:
				return nil
			case d.previous:
				return runPreviousStatusCommand(cmd, d.set.PreviousCommand, d.raw, note)
			}
			return render(cmd.OutOrStdout(), *asJSON, d.row, func(w io.Writer) {
				fmt.Fprintln(w, d.row.String())
			})
		},
	}
}

// statusDecision is what one refresh decided, before anything is printed:
// abcd's row, or the previous command, or nothing, and the notes to put on
// stderr on the way.
type statusDecision struct {
	// raw is the payload exactly as read: the previous command gets the SAME
	// stdin.
	raw []byte
	set statusline.Settings
	// notes are already phrased; the caller prefixes and sanitises them.
	notes []string
	// previous: run the person's previous status command.
	previous bool
	// blank: print nothing. Git did not answer for the checkout in time, so
	// whether abcd manages it is unknown, and neither abcd's row nor the
	// previous command can be vouched for.
	blank bool
	row   statusline.Row
	err   error
}

// decideStatus is the whole of abcd's own work for one refresh: read the
// payload, load the setting, resolve the checkout, decide whether it is
// managed, and compose the row. It runs under the watchdog, so it writes no
// stream; everything it would say is returned as notes. Git is bounded by
// gitCtx.
func decideStatus(gitCtx context.Context, stdin io.Reader) statusDecision {
	var d statusDecision
	raw, payload, notes := readStatusPayload(stdin)
	d.raw = raw
	d.notes = append(d.notes, notes...)

	cwd := payload.Cwd
	if cwd == "" {
		wd, err := os.Getwd()
		if err != nil {
			d.err = err
			return d
		}
		cwd = wd
	}

	set, setNotes, err := statusline.Load()
	if err != nil {
		// A setting that IS the caller's word and is malformed: named, and
		// the bundled defaults render rather than the row going blank.
		d.notes = append(d.notes, err.Error()+"; the bundled defaults render")
		set = statusline.Defaults()
	}
	d.notes = append(d.notes, setNotes...)
	d.set = set

	root, rootErr := statusline.CheckoutRoot(gitCtx, cwd)
	if rootErr != nil && !errors.Is(rootErr, statusline.ErrNoCheckout) {
		d.notes = append(d.notes, fmt.Sprintf("git did not name the checkout within %s, so the status line is empty for this refresh", statuslineGitBudget))
		d.blank = true
		return d
	}
	managed := false
	if rootErr == nil {
		var mErr error
		if managed, mErr = ahoy.ManagedContext(gitCtx, root); mErr != nil {
			d.notes = append(d.notes, fmt.Sprintf("git did not say within %s whether abcd manages this checkout, so the status line is empty for this refresh", statuslineGitBudget))
			d.blank = true
			return d
		}
	}
	if !managed || set.Disabled {
		d.previous = true
		return d
	}

	res, err := statusline.ComposeContext(gitCtx, root, payload, set)
	if err != nil {
		d.err = fmt.Errorf("abcd statusline: %w", err)
		return d
	}
	d.notes = append(d.notes, res.Notes...)
	d.row = res.Row
	return d
}

// readStatusPayload reads the harness payload under the hook cap and parses
// it. It returns the raw bytes — the previous command gets the SAME stdin —
// the parsed payload, which is the zero Payload wherever the read or the parse
// failed, and the notes it had to make. Nothing here is fatal: empty stdin is
// the ordinary case for a human running the verb by hand and for the smoke
// harness, so it is silent; an over-cap or unparseable payload is named and
// the row renders without its payload elements. The cap is read one byte past
// so an over-cap payload is discarded whole rather than truncated into a
// severed prefix the parser would misblame as malformed (readHookInput's
// reason). It writes no stream: it runs under the watchdog.
func readStatusPayload(stdin io.Reader) ([]byte, statusline.Payload, []string) {
	raw, err := io.ReadAll(io.LimitReader(stdin, maxHookStdinBytes+1))
	if err != nil {
		return nil, statusline.Payload{}, []string{fmt.Sprintf("unreadable stdin (%s); rendering without the payload", err.Error())}
	}
	if len(raw) > maxHookStdinBytes {
		return nil, statusline.Payload{}, []string{fmt.Sprintf("payload is over the %d-byte cap; it was discarded unparsed", maxHookStdinBytes)}
	}
	if len(bytes.TrimSpace(raw)) == 0 {
		return raw, statusline.Payload{}, nil
	}
	p, err := statusline.ParsePayload(raw)
	if err != nil {
		return raw, statusline.Payload{}, []string{err.Error() + "; rendering without the payload"}
	}
	return raw, p, nil
}

// runPreviousStatusCommand is the ac-5/ac-6 fallback: the user's own status
// command, recorded at install from the harness's configuration, runs through
// `sh -c` exactly as the harness would have run it, with the payload abcd
// received on its stdin, and its streams and exit code are passed through
// unchanged. The command is the caller's own word — the setting is a
// trust-boundary read that refuses a file the caller does not own or that
// others can write — and it is the command the harness was running before
// abcd took the row, so running it is restoring the user's line, not
// executing configuration abcd chose. With nothing recorded there is nothing
// to restore: print nothing, exit 0.
//
// The one thing it will not do is run itself. The child's environment carries
// statuslineFallbackEnv, and a run that finds the marker already set is a
// previous command that reached abcd's own status verb: it prints nothing,
// says so once through note, and exits 0 — the status line goes blank rather
// than the machine forking, and the note names what to fix.
//
// It runs under previousCommandBudget. At the bound its process group is
// killed, the output it already printed stands, one note says so, and the
// exit is 0: the line the person sees is what their command managed to say,
// never a frozen status line and never abcd's error in its place.
func runPreviousStatusCommand(cmd *cobra.Command, previous string, stdin []byte, note func(string, ...any)) error {
	if previous == "" {
		return nil
	}
	if os.Getenv(statuslineFallbackEnv) != "" {
		note("the previous status command recorded in %s reaches abcd's own status verb, so abcd refused to recurse; "+
			"remove it from the setting (or from the harness setting it was recorded from) to restore the line",
			statusline.SettingsDisplay)
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), previousCommandBudget)
	defer cancel()
	sh := exec.CommandContext(ctx, "sh", "-c", previous)
	sh.Env = append(os.Environ(), statuslineFallbackEnv+"=1")
	sh.Stdin = bytes.NewReader(stdin)
	sh.Stdout = cmd.OutOrStdout()
	sh.Stderr = cmd.ErrOrStderr()
	// The command leads its own process group, and the bound kills the group
	// through the pid this handle holds — never a pattern, never another
	// process — so a pipeline or a background child it started ends with it
	// rather than outliving the refresh (internal/core/tools' runArgv, the
	// same shape).
	sh.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	sh.Cancel = func() error { return syscall.Kill(-sh.Process.Pid, syscall.SIGKILL) }
	sh.WaitDelay = previousCommandWaitDelay
	err := sh.Run()
	var exit *exec.ExitError
	switch {
	case err == nil:
		return nil
	case errors.Is(ctx.Err(), context.DeadlineExceeded):
		// Whatever it printed before the bound has already reached the
		// harness and stands; the line is that, not an error.
		note("the previous status command recorded in %s did not finish within %s and was stopped; its output so far stands",
			statusline.SettingsDisplay, previousCommandBudget)
		return nil
	case errors.As(err, &exit):
		// Its exit code is ours, and it has already said whatever it had to say.
		return &exitError{Code: exit.ExitCode(), Msg: ""}
	default:
		return fmt.Errorf("abcd statusline: the previous status command could not be run: %s", termsafe.Sanitize(err.Error()))
	}
}
