package cli

import (
	"errors"
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/intentdriven/abcd/internal/core/implement"
	"github.com/intentdriven/abcd/internal/fsutil"
	"github.com/intentdriven/abcd/internal/gitutil"
	"github.com/intentdriven/abcd/internal/termsafe"
	"github.com/spf13/cobra"
)

// implementStore names what the implement verbs address, for the checkout-root
// refusal.
const implementStore = "the run state"

// newImplementCommand builds the `implement` family — the front door onto
// internal/core/implement, the run machinery an autonomous run calls
// (itd-2609221656373558). It is the family the implement loop extends
// (itd-2609201916151817 decision 8: `build` is what a person types, `implement`
// the machinery a driving session calls), so its sub-verbs are the run's own
// steps: join and leave, the window's mode, the claim and its release, the bounds
// check, the log, and the comparison. Bare `abcd implement` is a read-only render
// of the run state.
func newImplementCommand(asJSON *bool) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "implement",
		Short: "Share one autonomous run between sessions: join, claim a record, check the bounds, log, and compare the division modes",
		Long: "The run machinery an autonomous run calls. Every piece lives in the machine-scoped run\n" +
			"state, `~/.abcd/runs/<root-sha>/`, keyed on the repository's root commit, so sessions\n" +
			"in different worktrees of one repository share one run and no repository file.\n\n" +
			"Bare `abcd implement` is read-only: the sessions that have joined, the claims and\n" +
			"whether each lease still holds, and the window's division mode. It creates nothing.\n\n" +
			"A session joins (`join`), which writes its record and a session_open line; nothing\n" +
			"signals any other session. The first session opens each window with its mode\n" +
			"(`mode`). A session claims a record before opening its lane (`claim`): one file per\n" +
			"record, taken by an exclusive create, so of two sessions reaching for one record\n" +
			"exactly one holds it; the claim is a lease, and a lapsed lease is claimable again.\n" +
			"The second session is bounded: one lane at a time, never the release, never a lane\n" +
			"that touches the reading corpus, no lane in a split-roles window (`check` asks before\n" +
			"a step that is not a claim). `log` appends the run's other events, and `report`\n" +
			"derives the comparison of the modes from the log.\n\n" +
			"Exit 2 on a refusal (an unrecognised input, a session that has not joined, a bound\n" +
			"the session's role does not permit), exit 3 on contention (the record is claimed by\n" +
			"another session, or the run state is locked): back off and take other work.",
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			run, err := implementRun(runPeek, "")
			if err != nil {
				return implementRefusal("", err)
			}
			st, err := implementStatus(run)
			if err != nil {
				return implementRefusal("", err)
			}
			return render(cmd.OutOrStdout(), *asJSON, st, func(w io.Writer) {
				fmt.Fprintf(w, "run state: %s\n", st.Dir)
				if st.Window != nil {
					fmt.Fprintf(w, "  mode:     %s (since %s, set by %s)\n", termsafe.Sanitize(string(st.Window.Mode)),
						st.Window.Since.Format(time.RFC3339), termsafe.Sanitize(st.Window.SetBy))
				} else {
					fmt.Fprintln(w, "  mode:     no window opened")
				}
				fmt.Fprintf(w, "  sessions: %d\n", len(st.Sessions))
				for _, s := range st.Sessions {
					fmt.Fprintf(w, "    %s  %s\n", termsafe.Sanitize(s.Session), s.Role)
				}
				fmt.Fprintf(w, "  claims:   %d\n", len(st.Claims))
				for _, c := range st.Claims {
					state := "live"
					if !c.Live {
						state = "lapsed"
					}
					fmt.Fprintf(w, "    %s  %s  lane %s  until %s  (%s)\n", c.Record, termsafe.Sanitize(c.Session),
						termsafe.Sanitize(c.Lane), c.ExpiresAt.Format(time.RFC3339), state)
				}
			})
		},
	}
	cmd.AddCommand(
		newImplementJoinCommand(asJSON),
		newImplementLeaveCommand(asJSON),
		newImplementModeCommand(asJSON),
		newImplementClaimCommand(asJSON),
		newImplementReleaseCommand(asJSON),
		newImplementCheckCommand(asJSON),
		newImplementLogCommand(asJSON),
		newImplementReportCommand(asJSON),
	)
	return cmd
}

// implementStatusOutput is the bare render's --json shape.
type implementStatusOutput struct {
	Dir      string                 `json:"dir"`
	Window   *implement.WindowState `json:"window"`
	Sessions []implement.Session    `json:"sessions"`
	Claims   []implement.ClaimState `json:"claims"`
}

// implementStatus reads the run state for the bare render.
func implementStatus(run *implement.Run) (implementStatusOutput, error) {
	out := implementStatusOutput{Dir: fsutil.RedactHome(run.Dir)}
	var err error
	if out.Sessions, err = run.Sessions(); err != nil {
		return out, err
	}
	if out.Claims, err = run.Claims(); err != nil {
		return out, err
	}
	w, ok, err := run.CurrentMode()
	if err != nil {
		return out, err
	}
	if ok {
		out.Window = &w
	}
	return out, nil
}

// runAccess is how a sub-verb opens the run.
type runAccess int

const (
	// runPeek opens the run without creating anything (the read-only renders).
	runPeek runAccess = iota
	// runJoin opens it creating what is missing: join is the one writer that may
	// bring a run into existence.
	runJoin
	// runJoined opens an existing run for a session that has joined it, and
	// refuses before creating anything when there is no run.
	runJoined
)

// implementRun resolves the run for the checkout the caller stands in: the
// checkout root, its root commit, and the run keyed on it, opened as access
// says. session is the acting session, for runJoined.
func implementRun(access runAccess, session string) (*implement.Run, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return nil, err
	}
	root, err := gitutil.CheckoutRoot(cwd, implementStore)
	if err != nil {
		return nil, err
	}
	sha := gitutil.RootCommit(root)
	switch access {
	case runJoin:
		return implement.Open(sha)
	case runJoined:
		return implement.OpenJoined(sha, session)
	}
	return implement.Peek(sha)
}

// implementRefusal maps the core's outcome classes to exit codes: a refusal
// (including no checkout to key a run on) exits 2, contention exits 3, anything
// else is an I/O failure at exit 1. The message names the sub-verb.
func implementRefusal(sub string, err error) error {
	prefix := "abcd implement"
	if sub != "" {
		prefix += " " + sub
	}
	switch {
	case errors.Is(err, implement.ErrContention):
		return &exitError{Code: 3, Msg: prefix + ": " + err.Error()}
	case errors.Is(err, implement.ErrRefused), errors.Is(err, gitutil.ErrNoCheckoutRoot):
		// A bound's refusal is itself logged, so the suffix says the act was not
		// taken rather than that nothing was written.
		return &exitError{Code: 2, Msg: prefix + ": " + err.Error() + " (not taken)"}
	}
	return fmt.Errorf("%s: %w", prefix, err)
}

// needSession refuses a writer invoked without --session. It is checked by
// hand rather than by marking the flag required, because a required flag is a
// surface the snapshot treats as a break; the refusal is the same exit 2.
func needSession(sub, session string) error {
	if session == "" {
		return &exitError{Code: 2, Msg: "abcd implement " + sub + ": --session is required: every write acts for a joined session (nothing written)"}
	}
	return nil
}

// withRun runs fn against the writable run, mapping its error. Only join may
// create the run: every other writer acts for a session that has joined, so a
// refused writer (an unjoined session, a run nobody has started) creates
// nothing — no run directory, no lock, no log.
func withRun(sub, session string, fn func(*implement.Run) error) error {
	if err := needSession(sub, session); err != nil {
		return err
	}
	access := runJoined
	if sub == "join" {
		access = runJoin
	}
	run, err := implementRun(access, session)
	if err != nil {
		return implementRefusal(sub, err)
	}
	if err := fn(run); err != nil {
		return implementRefusal(sub, err)
	}
	return nil
}

func newImplementJoinCommand(asJSON *bool) *cobra.Command {
	var session, role, model, reason string
	cmd := &cobra.Command{
		Use:   "join --session <id> --role first|second",
		Short: "Join the run: record the session and its role, and log its session_open",
		Long: "Record this session in the run state with its role and log a session_open line.\n" +
			"Nothing signals any other session: the first learns of a second only by reading the\n" +
			"run state. Joining again with the same role is a resume and is logged as one; asking\n" +
			"for the other role is refused. The role is the session's own statement, recorded\n" +
			"here and read by every bound — never taken from the environment.",
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			r, err := implement.ParseRole(role)
			if err != nil {
				return implementRefusal("join", err)
			}
			return withRun("join", session, func(run *implement.Run) error {
				res, err := run.Join(session, r, model, reason)
				if err != nil {
					return err
				}
				return render(cmd.OutOrStdout(), *asJSON, res, func(w io.Writer) {
					verb := "joined"
					if res.Rejoined {
						verb = "rejoined"
					}
					fmt.Fprintf(w, "%s %s as the %s session\n", verb, res.Session.Session, res.Session.Role)
				})
			})
		},
	}
	cmd.Flags().StringVar(&session, "session", "", "this session's id (letters, digits, '.', '_', '-')")
	cmd.Flags().StringVar(&role, "role", "", "first | second")
	cmd.Flags().StringVar(&model, "model", "", "the model this session runs, recorded on the session_open line")
	cmd.Flags().StringVar(&reason, "reason", "", "why the session opens (run start, window, resume), recorded on the line")
	return cmd
}

func newImplementLeaveCommand(asJSON *bool) *cobra.Command {
	var session, reason string
	cmd := &cobra.Command{
		Use:   "leave --session <id>",
		Short: "Leave the run: release every claim the session holds and log its session_close",
		Long: "Release every claim this session holds (each logged as claim_released), log a\n" +
			"session_close line with the reason, and remove the session's record. A session\n" +
			"that stops without leaving strands nothing: its claims lapse with their leases.",
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return withRun("leave", session, func(run *implement.Run) error {
				res, err := run.Leave(session, reason)
				if err != nil {
					return err
				}
				return render(cmd.OutOrStdout(), *asJSON, res, func(w io.Writer) {
					fmt.Fprintf(w, "%s left the run; %d claim(s) released\n", res.Session.Session, len(res.Released))
				})
			})
		},
	}
	cmd.Flags().StringVar(&session, "session", "", "this session's id")
	cmd.Flags().StringVar(&reason, "reason", "", "why the session closes (window, stop condition, crash recovery)")
	return cmd
}

func newImplementModeCommand(asJSON *bool) *cobra.Command {
	var session string
	var window int
	cmd := &cobra.Command{
		Use:       "mode <single|claim|batch|split-roles> --session <id>",
		Short:     "Open a window: log its division mode (the first session's call)",
		ValidArgs: modeWords(),
		Long: "Log a window_mode line naming how this window divides the work: `single` (one\n" +
			"session), `claim` (a session claims a record before opening its lane), `batch` (the\n" +
			"run file assigns whole batches per session), or `split-roles` (the first session\n" +
			"builds; the second reviews, audits and lands). Only the first session sets it. The\n" +
			"mode in force is the log's last window_mode line, whoever wrote it.",
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			m, err := implement.ParseMode(args[0])
			if err != nil {
				return implementRefusal("mode", err)
			}
			return withRun("mode", session, func(run *implement.Run) error {
				st, err := run.SetMode(session, m, window)
				if err != nil {
					return err
				}
				return render(cmd.OutOrStdout(), *asJSON, st, func(w io.Writer) {
					fmt.Fprintf(w, "window opened in %s mode\n", st.Mode)
				})
			})
		},
	}
	cmd.Flags().StringVar(&session, "session", "", "this session's id (a first session)")
	cmd.Flags().IntVar(&window, "window", 0, "the window's number, recorded on the line")
	return cmd
}

// modeWords is the closed mode vocabulary as strings, for shell completion.
func modeWords() []string {
	var out []string
	for _, m := range implement.Modes() {
		out = append(out, string(m))
	}
	return out
}

func newImplementClaimCommand(asJSON *bool) *cobra.Command {
	var session, lane string
	var lease time.Duration
	var paths []string
	cmd := &cobra.Command{
		Use:   "claim <record> --session <id> --lane <lane>",
		Short: "Claim a record before opening its lane; exactly one session holds it",
		Long: "Take a record for this session: one claim file per record in the run state, created\n" +
			"exclusively, so of two sessions reaching for one record exactly one holds it. The\n" +
			"claim is a lease (--lease, default 2h, 1m to 24h). Claiming a record this session\n" +
			"already holds renews the lease. A claim whose lease has passed is claimable again,\n" +
			"and the lapse is logged as claim_lapsed. A record another session holds is refused\n" +
			"at exit 3 and logged as claim_denied naming the holder; the second session also\n" +
			"logs a backoff.\n\n" +
			"The second session is refused (exit 2, logged as a refusal) when it already holds\n" +
			"a live claim, when the window is split-roles, or when a --path it declares is in the\n" +
			"reading corpus.",
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return withRun("claim", session, func(run *implement.Run) error {
				res, err := run.Claim(implement.ClaimRequest{Session: session, Record: args[0], Lane: lane, Lease: lease, Paths: paths})
				if err != nil {
					return err
				}
				return render(cmd.OutOrStdout(), *asJSON, res, func(w io.Writer) {
					verb := "claimed"
					if res.Renewed {
						verb = "renewed"
					}
					fmt.Fprintf(w, "%s %s for lane %s until %s\n", verb, res.Claim.Record, res.Claim.Lane,
						res.Claim.ExpiresAt.Format(time.RFC3339))
					if res.Lapsed != nil {
						fmt.Fprintf(w, "  (replaced session %s's lapsed claim)\n", termsafe.Sanitize(res.Lapsed.Session))
					}
				})
			})
		},
	}
	cmd.Flags().StringVar(&session, "session", "", "this session's id")
	cmd.Flags().StringVar(&lane, "lane", "", "the lane the claim is for")
	cmd.Flags().DurationVar(&lease, "lease", implement.DefaultLease, "how long the claim holds before it lapses (1m to 24h)")
	cmd.Flags().StringArrayVar(&paths, "path", nil, "a repository-relative file the lane will touch (repeatable); checked against the reading corpus")
	return cmd
}

func newImplementReleaseCommand(asJSON *bool) *cobra.Command {
	var session string
	cmd := &cobra.Command{
		Use:   "release <record> --session <id>",
		Short: "Release this session's claim on a record",
		Long: "Remove this session's claim on a record and log claim_released. Only the holder\n" +
			"releases a claim; another session's claim lapses with its lease instead.",
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return withRun("release", session, func(run *implement.Run) error {
				c, err := run.Release(session, args[0])
				if err != nil {
					return err
				}
				return render(cmd.OutOrStdout(), *asJSON, c, func(w io.Writer) {
					fmt.Fprintf(w, "released %s\n", c.Record)
				})
			})
		},
	}
	cmd.Flags().StringVar(&session, "session", "", "this session's id")
	return cmd
}

func newImplementCheckCommand(asJSON *bool) *cobra.Command {
	var session string
	var paths []string
	cmd := &cobra.Command{
		Use:       "check <lane|release|review|audit|land> --session <id>",
		Short:     "Ask whether this session may take a step; the second session's bounds refuse",
		ValidArgs: stepWords(),
		Long: "Say whether this session may take a step, before it takes it. The first session may\n" +
			"take every step. The second is refused the release step always, a lane in a\n" +
			"split-roles window, and a lane whose --path reaches the reading corpus; review,\n" +
			"audit and land are open to it. A refusal exits 2 and is logged; an allowed step\n" +
			"writes nothing.",
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			st, err := implement.ParseStep(args[0])
			if err != nil {
				return implementRefusal("check", err)
			}
			return withRun("check", session, func(run *implement.Run) error {
				v, err := run.Check(session, st, paths)
				if err != nil {
					return err
				}
				return render(cmd.OutOrStdout(), *asJSON, v, func(w io.Writer) {
					fmt.Fprintf(w, "%s may take the %s step\n", v.Session, v.Step)
				})
			})
		},
	}
	cmd.Flags().StringVar(&session, "session", "", "this session's id")
	cmd.Flags().StringArrayVar(&paths, "path", nil, "a repository-relative file the step touches (repeatable)")
	return cmd
}

// stepWords is the closed step vocabulary as strings, for shell completion.
func stepWords() []string {
	var out []string
	for _, s := range implement.Steps() {
		out = append(out, string(s))
	}
	return out
}

// fieldArgRe is one --field operand: key=value.
var fieldArgRe = regexp.MustCompile(`^([^=]+)=(.*)$`)

func newImplementLogCommand(asJSON *bool) *cobra.Command {
	var session string
	var fields []string
	cmd := &cobra.Command{
		Use:       "log <event> --session <id> [--field key=value ...]",
		Short:     "Append one of the run's events to the run log",
		ValidArgs: implement.LoggableEvents(),
		Long: "Append one event line to today's run log (`~/.abcd/runs/<root-sha>/<UTC date>.jsonl`)\n" +
			"in a single append, so two sessions writing at once each land whole lines. The line\n" +
			"carries ts, session and event, then each --field. A value that reads as a number or\n" +
			"a boolean is written as one when it reads back as the same text, so `sha=0123456`\n" +
			"stays a string. The events: " + strings.Join(implement.LoggableEvents(), ", ") + ".\n" +
			"The claim, window and session events are written by their own sub-verbs and are\n" +
			"refused here, so the log cannot record a claim the run state does not hold.",
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			kv := map[string]string{}
			for _, f := range fields {
				m := fieldArgRe.FindStringSubmatch(f)
				if m == nil {
					return &exitError{Code: 2, Msg: fmt.Sprintf("abcd implement log: --field %q is not key=value (nothing written)", f)}
				}
				if _, dup := kv[m[1]]; dup {
					return &exitError{Code: 2, Msg: fmt.Sprintf("abcd implement log: --field %s is given twice (nothing written)", m[1])}
				}
				kv[m[1]] = m[2]
			}
			return withRun("log", session, func(run *implement.Run) error {
				e, err := run.Log(session, args[0], kv)
				if err != nil {
					return err
				}
				return render(cmd.OutOrStdout(), *asJSON, e, func(w io.Writer) {
					fmt.Fprintf(w, "logged %s at %s\n", e.Event, e.TS.Format(time.RFC3339))
				})
			})
		},
	}
	cmd.Flags().StringVar(&session, "session", "", "this session's id")
	cmd.Flags().StringArrayVar(&fields, "field", nil, "an event field as key=value (repeatable)")
	return cmd
}

func newImplementReportCommand(asJSON *bool) *cobra.Command {
	var date, logPath string
	cmd := &cobra.Command{
		Use:   "report [--date YYYY-MM-DD | --log <file>]",
		Short: "Derive the comparison of the division modes from the run log (read-only)",
		Long: "Derive, per division mode, the figures the run's report compares: windows, wall\n" +
			"clock, lanes opened and landed (a lane_close whose outcome is merged or landed),\n" +
			"the second session's lanes landed, collisions (claim_denied), lapsed claims,\n" +
			"backoffs and the minutes backed off, agent minutes (agent_end's minutes, wall_minutes\n" +
			"or wall_min), ceiling wait and refusals, per session within each mode. Each event\n" +
			"belongs to the window open when it happened; each session's context lines are totalled\n" +
			"across the run, with the last used_pct seen. `leader` is the mode with the most lanes landed per wall-clock hour —\n" +
			"a figure, not a verdict. Lines the reader cannot use are listed, never dropped\n" +
			"silently.\n\n" +
			"By default the run's whole log is read, every day of it; --date reads one day, and\n" +
			"--log reads one log file named directly. Reads only; creates nothing.",
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			rep, err := implementReport(date, logPath)
			if err != nil {
				return implementRefusal("report", err)
			}
			return render(cmd.OutOrStdout(), *asJSON, rep, func(w io.Writer) {
				fmt.Fprintf(w, "%d events", rep.Events)
				if len(rep.Unparsed) > 0 {
					fmt.Fprintf(w, ", %d line(s) unreadable", len(rep.Unparsed))
				}
				fmt.Fprintln(w)
				fmt.Fprintf(w, "%-12s %7s %8s %6s %6s %6s %10s %9s %8s %8s\n",
					"mode", "windows", "wall min", "opened", "landed", "B land", "collisions", "backoff m", "agent m", "ceiling")
				for _, m := range rep.Modes {
					fmt.Fprintf(w, "%-12s %7d %8.1f %6d %6d %6d %10d %9.1f %8.1f %8.1f\n",
						termsafe.Sanitize(m.Mode), m.Windows, m.WallMinutes, m.LanesOpened, m.LanesLanded,
						m.SecondLanesLanded, m.Collisions, m.BackoffMinutes, m.AgentMinutes, m.CeilingWaitMinutes)
				}
				for _, c := range rep.Context {
					fmt.Fprintf(w, "context: %s  %d measurement(s), last %.0f%% used at %s\n", termsafe.Sanitize(c.Session),
						c.Events, c.LastUsedPct, c.LastAt.Format(time.RFC3339))
				}
				if rep.Leader != "" {
					fmt.Fprintf(w, "most lanes landed per wall-clock hour: %s\n", termsafe.Sanitize(rep.Leader))
				} else {
					fmt.Fprintln(w, "most lanes landed per wall-clock hour: none (nothing landed, or a tie)")
				}
			})
		},
	}
	cmd.Flags().StringVar(&date, "date", "", "read one day's log (YYYY-MM-DD, UTC)")
	cmd.Flags().StringVar(&logPath, "log", "", "read this log file instead of the run's own")
	return cmd
}

// maxReportLogBytes caps a log read through --log.
const maxReportLogBytes = 64 << 20

// implementReport reads the log the flags name and derives the comparison.
func implementReport(date, logPath string) (implement.Report, error) {
	if date != "" && logPath != "" {
		return implement.Report{}, fmt.Errorf("%w: --date and --log name two different logs; give one", implement.ErrRefused)
	}
	if logPath != "" {
		data, err := fsutil.ReadGuarded(logPath, maxReportLogBytes)
		if err != nil {
			return implement.Report{}, fmt.Errorf("%w: cannot read the log: %v", implement.ErrRefused, err)
		}
		events, bad := implement.ParseLog(fsutil.RedactHome(logPath), data)
		return implement.Compare(events, bad), nil
	}
	run, err := implementRun(runPeek, "")
	if err != nil {
		return implement.Report{}, err
	}
	events, bad, err := run.ReadLog()
	if err != nil {
		return implement.Report{}, err
	}
	if date != "" {
		day, err := time.Parse(time.DateOnly, date)
		if err != nil {
			return implement.Report{}, fmt.Errorf("%w: --date %q is not YYYY-MM-DD", implement.ErrRefused, date)
		}
		name := day.Format(time.DateOnly) + ".jsonl"
		kept := events[:0]
		for _, e := range events {
			if e.TS.Format(time.DateOnly) == day.Format(time.DateOnly) {
				kept = append(kept, e)
			}
		}
		events = kept
		var keptBad []implement.Unparsed
		for _, u := range bad {
			if u.File == name {
				keptBad = append(keptBad, u)
			}
		}
		bad = keptBad
	}
	return implement.Compare(events, bad), nil
}
