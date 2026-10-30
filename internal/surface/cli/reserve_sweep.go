package cli

import (
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/intentdriven/abcd/internal/core/reservesweep"
	"github.com/intentdriven/abcd/internal/fsutil"
	"github.com/intentdriven/abcd/internal/gitutil"
	"github.com/intentdriven/abcd/internal/termsafe"
	"github.com/spf13/cobra"
)

// reserveSweepNow is the sweep's clock; nil means time.Now. Tests set it.
var reserveSweepNow func() time.Time

// sweepFlags are `reserve --sweep`'s flags (iss-2610142200389875).
type sweepFlags struct {
	on        bool
	apply     bool
	olderThan int
}

// addSweepFlags registers the sweep's flags on the reserve verb.
func addSweepFlags(cmd *cobra.Command) *sweepFlags {
	s := &sweepFlags{}
	cmd.Flags().BoolVar(&s.on, "sweep", false, "list the spent reservation branches on origin, which --apply deletes; a dry run without it")
	cmd.Flags().BoolVar(&s.apply, "apply", false, "with --sweep, delete the spent branches, each only while its tip is the one judged")
	cmd.Flags().IntVar(&s.olderThan, "older-than", int(reservesweep.DefaultAge/(24*time.Hour)), "with --sweep, the days a tip other than a release must have been untouched past its lease (at least 1)")
	return s
}

// args refuses what the sweep does not take: a record id, a reservation
// act's flags, and --apply or --older-than without --sweep.
func (s *sweepFlags) args(cmd *cobra.Command, args []string) error {
	if !s.on {
		for _, f := range []string{"apply", "older-than"} {
			if cmd.Flags().Changed(f) {
				return &exitError{Code: 2, Msg: "abcd reserve: --" + f + " is for --sweep; nothing was pushed"}
			}
		}
		return nil
	}
	var with []string
	if len(args) > 0 {
		with = append(with, "record id")
	}
	for _, f := range []string{"footprint", "take-over", "lease", "release", "holder"} {
		if cmd.Flags().Changed(f) {
			with = append(with, "--"+f)
		}
	}
	if len(with) > 0 {
		return &exitError{Code: 2, Msg: "abcd reserve: --sweep takes no " + strings.Join(with, ", ") + "; nothing was deleted"}
	}
	if s.olderThan < 1 {
		return &exitError{Code: 2, Msg: fmt.Sprintf("abcd reserve: --older-than takes a whole number of days, at least 1, not %d; nothing was deleted", s.olderThan)}
	}
	return nil
}

// sweepOutput is the sweep's --json document.
type sweepOutput struct {
	Mode          string        `json:"mode"`
	AsOf          string        `json:"as_of"`
	OlderThanDays int           `json:"older_than_days"`
	DefaultUnread string        `json:"default_branch_unread,omitempty"`
	Spent         int           `json:"spent"`
	Kept          int           `json:"kept"`
	Deleted       int           `json:"deleted"`
	Branches      []sweepBranch `json:"branches"`
}

// sweepBranch is one branch in the document.
type sweepBranch struct {
	Record    string `json:"record"`
	Branch    string `json:"branch"`
	Commit    string `json:"commit"`
	Kind      string `json:"kind,omitempty"`
	Spent     bool   `json:"spent"`
	Reason    string `json:"reason"`
	Why       string `json:"why,omitempty"`
	SettledAt string `json:"settled_at,omitempty"`
	Outcome   string `json:"outcome,omitempty"`
	Error     string `json:"error,omitempty"`
}

// runReserveSweep is `reserve --sweep`: a dry run, or with --apply the
// deletions. It exits 2 when the sweep could not run or a deletion failed
// for a reason other than the branch having moved since it was judged.
func runReserveSweep(w io.Writer, asJSON bool, s *sweepFlags) error {
	outcome := "nothing was deleted"
	cwd, err := os.Getwd()
	if err != nil {
		return err
	}
	root, err := gitutil.CheckoutRoot(cwd, reserveStore)
	if err != nil {
		return &exitError{Code: 2, Msg: "abcd reserve: " + err.Error() + "; " + outcome}
	}
	rep, err := reservesweep.Sweep(root, reservesweep.Options{Now: reserveSweepNow,
		Age: time.Duration(s.olderThan) * 24 * time.Hour, Apply: s.apply})
	if err != nil {
		msg := termsafe.Sanitize(fsutil.RedactHome(err.Error()))
		if reservesweep.IsNoOrigin(err) {
			msg += "; " + outcome
		}
		return &exitError{Code: 2, Msg: "abcd reserve: " + msg}
	}
	out := sweepView(rep, s.olderThan)
	if err := render(w, asJSON, out, func(w io.Writer) { renderReserveSweep(w, out) }); err != nil {
		return err
	}
	if rep.Failed() {
		return &exitError{Code: 2, Msg: "abcd reserve: origin refused a deletion of a spent reservation branch; the others are as reported"}
	}
	return nil
}

// sweepView is a report as the surface shows it.
func sweepView(rep reservesweep.Report, days int) sweepOutput {
	out := sweepOutput{Mode: "dry_run", AsOf: rfc3339(rep.AsOf), OlderThanDays: days,
		DefaultUnread: termsafe.Sanitize(fsutil.RedactHome(rep.DefaultUnread)), Branches: []sweepBranch{}}
	if rep.Applied {
		out.Mode = "apply"
	}
	out.Spent, out.Kept, out.Deleted = rep.Counts()
	for _, b := range rep.Branches {
		sb := sweepBranch{Record: b.Record, Branch: strings.TrimPrefix(b.Ref, "refs/heads/"), Commit: b.Commit,
			Kind: string(b.Kind), Spent: b.Spent, Reason: b.Reason, Why: b.Why, Outcome: b.Outcome,
			Error: termsafe.Sanitize(fsutil.RedactHome(b.Error))}
		if !b.SettledAt.IsZero() {
			sb.SettledAt = rfc3339(b.SettledAt)
		}
		out.Branches = append(out.Branches, sb)
	}
	return out
}

// sweepReasons says each reason in words.
var sweepReasons = map[string]string{
	reservesweep.SpentTerminal:   "the record is finished on origin's default branch",
	reservesweep.SpentReleased:   "released by a plain tombstone, no hand-back on the branch",
	reservesweep.SpentLapsed:     "its lease lapsed over a plain release, no hand-back on the branch",
	reservesweep.SpentUnreleased: "its lease lapsed and it was never released, no hand-back on the branch",
	reservesweep.KeptRecent:      "touched or held too recently",
	reservesweep.KeptHandBack:    "a hand-back stands on it and the record is open",
	reservesweep.KeptUnknown:     "it carries a commit the sweep cannot classify",
	reservesweep.KeptHistory:     "its history could not be read to its root",
	reservesweep.KeptUnreadable:  "its tip is unreadable",
}

// renderReserveSweep is the text form: a heading, one line per branch, the totals.
func renderReserveSweep(w io.Writer, out sweepOutput) {
	mode := "dry run"
	if out.Mode == "apply" {
		mode = "applied"
	}
	fmt.Fprintf(w, "abcd reserve --sweep (%s) as of %s: spent = released (a hand-back too once the record is finished), or untouched %d day(s) past its lease\n", mode, out.AsOf, out.OlderThanDays)
	if out.DefaultUnread != "" {
		fmt.Fprintf(w, "  not checked: which records are finished on origin's default branch (%s)\n", out.DefaultUnread)
	}
	for _, b := range out.Branches {
		verb := "keep  "
		if b.Spent {
			switch b.Outcome {
			case reservesweep.Deleted:
				verb = "deleted"
			case reservesweep.Moved:
				verb = "moved, kept"
			case reservesweep.Failed:
				verb = "FAILED"
			default:
				verb = "delete"
			}
		}
		line := fmt.Sprintf("  %-7s %s %s: %s", verb, b.Branch, shortCommit(b.Commit), sweepReasons[b.Reason])
		if b.Why != "" {
			line += " (" + b.Why + ")"
		}
		if b.SettledAt != "" && !b.Spent {
			line += "; settled " + b.SettledAt
		}
		if b.Error != "" {
			line += ": " + b.Error
		}
		fmt.Fprintln(w, line)
	}
	if out.Mode == "apply" {
		fmt.Fprintf(w, "%d of %d branch(es) spent; %d deleted\n", out.Spent, out.Spent+out.Kept, out.Deleted)
		return
	}
	fmt.Fprintf(w, "%d of %d branch(es) spent; nothing was deleted (--apply deletes them)\n", out.Spent, out.Spent+out.Kept)
}
