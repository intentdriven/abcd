package cli

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"

	"github.com/intentdriven/abcd/internal/core/ahoy"
	"github.com/intentdriven/abcd/internal/core/statusline"
	"github.com/intentdriven/abcd/internal/gitutil"
	"github.com/intentdriven/abcd/internal/termsafe"
	"github.com/spf13/cobra"
)

// statuslineStore is the noun the checkout-root refusal is phrased with. The
// verb never surfaces that refusal — no checkout is simply "not managed" —
// but the resolver's contract asks for one.
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
// It never prompts, never reads a terminal, never touches the network. A
// payload that does not parse, or is over the cap, is named on stderr and the
// row still renders without its payload elements: a status surface that goes
// blank on a bad payload hides the parked stop the badge exists to show.
func newStatuslineCommand(asJSON *bool) *cobra.Command {
	return &cobra.Command{
		Use: "statusline",
		Long: "Render abcd's row for the host harness's status line.\n\n" +
			"The harness runs this on every status refresh, with its JSON status\n" +
			"payload on stdin, and shows what it prints. In a checkout abcd manages the\n" +
			"row is abcd's own: the presence badge first — `abcd`, `waiting: facilitator`\n" +
			"or `waiting: product thinker`, from the state `abcd mode` stores — then the\n" +
			"repository name, the branch, the model, the context percentage, the\n" +
			"five-hour and seven-day usage percentages, and the record's counts of\n" +
			"intents not yet shipped and open issues. Each element after the badge is\n" +
			"switchable in `~/.abcd/statusline.json`; a payload field the harness did not\n" +
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

			raw, payload := readStatusPayload(cmd, note)

			cwd := payload.Cwd
			if cwd == "" {
				wd, err := os.Getwd()
				if err != nil {
					return err
				}
				cwd = wd
			}

			set, notes, err := statusline.Load()
			if err != nil {
				// A setting that IS the caller's word and is malformed: named, and
				// the bundled defaults render rather than the row going blank.
				note("%s; the bundled defaults render", termsafe.Sanitize(err.Error()))
				set = statusline.Defaults()
			}
			for _, n := range notes {
				note("%s", termsafe.Sanitize(n))
			}

			root, rootErr := gitutil.CheckoutRoot(cwd, statuslineStore)
			managed := rootErr == nil && ahoy.Managed(root)
			if !managed || set.Disabled {
				return runPreviousStatusCommand(cmd, set.PreviousCommand, raw, note)
			}

			res, err := statusline.Compose(root, payload, set)
			if err != nil {
				return fmt.Errorf("abcd statusline: %w", err)
			}
			for _, n := range res.Notes {
				note("%s", termsafe.Sanitize(n))
			}
			return render(cmd.OutOrStdout(), *asJSON, res.Row, func(w io.Writer) {
				fmt.Fprintln(w, res.Row.String())
			})
		},
	}
}

// readStatusPayload reads the harness payload under the hook cap and parses
// it. It returns the raw bytes — the previous command gets the SAME stdin —
// and the parsed payload, which is the zero Payload wherever the read or the
// parse failed. Nothing here is fatal: empty stdin is the ordinary case for a
// human running the verb by hand and for the smoke harness, so it is silent;
// an over-cap or unparseable payload is named through note and the row
// renders without its payload elements. The cap is read one byte past so an
// over-cap payload is discarded whole rather than truncated into a severed
// prefix the parser would misblame as malformed (readHookInput's reason).
func readStatusPayload(cmd *cobra.Command, note func(string, ...any)) ([]byte, statusline.Payload) {
	raw, err := io.ReadAll(io.LimitReader(cmd.InOrStdin(), maxHookStdinBytes+1))
	if err != nil {
		note("unreadable stdin (%s); rendering without the payload", termsafe.Sanitize(err.Error()))
		return nil, statusline.Payload{}
	}
	if len(raw) > maxHookStdinBytes {
		note("payload is over the %d-byte cap; it was discarded unparsed", maxHookStdinBytes)
		return nil, statusline.Payload{}
	}
	if len(bytes.TrimSpace(raw)) == 0 {
		return raw, statusline.Payload{}
	}
	p, err := statusline.ParsePayload(raw)
	if err != nil {
		note("%s; rendering without the payload", termsafe.Sanitize(err.Error()))
		return raw, statusline.Payload{}
	}
	return raw, p
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
	sh := exec.Command("sh", "-c", previous)
	sh.Env = append(os.Environ(), statuslineFallbackEnv+"=1")
	sh.Stdin = bytes.NewReader(stdin)
	sh.Stdout = cmd.OutOrStdout()
	sh.Stderr = cmd.ErrOrStderr()
	err := sh.Run()
	var exit *exec.ExitError
	switch {
	case err == nil:
		return nil
	case errors.As(err, &exit):
		// Its exit code is ours, and it has already said whatever it had to say.
		return &exitError{Code: exit.ExitCode(), Msg: ""}
	default:
		return fmt.Errorf("abcd statusline: the previous status command could not be run: %s", termsafe.Sanitize(err.Error()))
	}
}
