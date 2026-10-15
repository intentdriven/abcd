package cli

import (
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/intentdriven/abcd/internal/core/claim"
	"github.com/intentdriven/abcd/internal/fsutil"
	"github.com/intentdriven/abcd/internal/gitutil"
	"github.com/intentdriven/abcd/internal/termsafe"
	"github.com/spf13/cobra"
)

// reserveStore is the noun the checkout resolution names in its refusal.
const reserveStore = "the reservation"

// reserveOutput is the --json document of one reservation act: what was done,
// to which record, for whom, from which copy (this or another, never a copy's
// key), over which files, until when, and what it replaced.
type reserveOutput struct {
	Act              claim.Act        `json:"act"`
	Record           string           `json:"record"`
	Holder           string           `json:"holder"`
	Copy             string           `json:"copy"`
	Footprint        []string         `json:"footprint"`
	ClaimedAt        string           `json:"claimed_at"`
	EffectiveExpiry  string           `json:"effective_expiry"`
	Commit           string           `json:"commit"`
	Replaced         *reserveReplaced `json:"replaced,omitempty"`
	DerivedUnchecked string           `json:"derived_unchecked,omitempty"`
}

// reserveReplaced is the reservation a take-over replaced.
type reserveReplaced struct {
	Holder          string `json:"holder,omitempty"`
	Copy            string `json:"copy,omitempty"`
	Commit          string `json:"commit"`
	EffectiveExpiry string `json:"effective_expiry"`
	Unreadable      string `json:"unreadable,omitempty"`
}

// newReserveCommand builds the `reserve` verb — the front door onto
// internal/core/claim (itd-2609091034175565, spc-2610031753359849): before an
// agent starts on an issue or an intent, it reserves the record on the shared
// remote as a branch nobody merges, so a second person, a second computer or a
// hosted session sees it before starting the same item. The verb is an
// agent's (the product thinker's ruling of 2026-10-12): a person sees
// reservations through `abcd peers`, never by typing this.
func newReserveCommand(asJSON *bool) *cobra.Command {
	var (
		footprint []string
		takeOver  bool
		release   bool
		leaseFor  time.Duration
	)
	cmd := &cobra.Command{
		Use: "reserve <record-id>",
		Long: "abcd reserve <record-id> --footprint <path> [--footprint <path>…] [--take-over] [--lease <duration>]\n" +
			"abcd reserve <record-id> --release\n\n" +
			"Reserve an issue or an intent on the shared remote before starting work on it, so\n" +
			"every other participant sees it is taken. The reservation is the branch\n" +
			"abcd-reserved/<record-id> on the remote named origin: a chain of empty-tree commits\n" +
			"naming the record, the holder (your GitHub username), this working copy, the files\n" +
			"the work will change (--footprint, once per repository-relative path) and the\n" +
			"lease (--lease, default 2h, from 1m to 24h).\n\n" +
			"Every act fetches the reservations from origin, judges the tip, writes one\n" +
			"commit and makes one plain fast-forward push, with the repository's hooks\n" +
			"running: a reservation, a renewal by the same person and copy, a take-over of\n" +
			"a reservation lapsed past its expiry by five minutes, or a release (--release),\n" +
			"which pushes a tombstone. --take-over moves your own live reservation from\n" +
			"another copy to this one; it never moves another person's. A record already\n" +
			"resolved, won't-fixed, shipped or superseded on origin's default branch counts\n" +
			"as released whatever its reservation says.\n\n" +
			"The holder is read from user.email when it is a GitHub no-reply address, or from\n" +
			"reservation.github_username in the machine configuration; the repository's\n" +
			"own configuration is never read for it. The checkout must have its local tier\n" +
			"(.abcd/.work.local/), where this copy's key is kept. `abcd peers` lists every\n" +
			"reservation.\n\n" +
			"Exit 0 when the act reached origin (or a release was already made); exit 3 when\n" +
			"the record is held by someone else or by your other copy, so back off; exit 2\n" +
			"on every other refusal and failure. Nothing is pushed on a refusal, and a failed\n" +
			"push leaves no reservation and nothing in the checkout.",
		Args: func(_ *cobra.Command, args []string) error {
			switch len(args) {
			case 0:
				return &exitError{Code: 2, Msg: "abcd reserve: name the issue or intent to reserve (abcd reserve <record-id> --footprint <path>); `abcd peers` lists the reservations; nothing was pushed"}
			case 1:
				return nil
			}
			return &exitError{Code: 2, Msg: "abcd reserve: takes one record id, not " + fmt.Sprint(len(args)) + "; nothing was pushed"}
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			if release {
				var with []string
				for _, f := range []string{"footprint", "take-over", "lease"} {
					if cmd.Flags().Changed(f) {
						with = append(with, "--"+f)
					}
				}
				if len(with) > 0 {
					return &exitError{Code: 2, Msg: "abcd reserve: --release takes no " + strings.Join(with, ", ") + "; nothing was pushed"}
				}
			}
			cwd, err := os.Getwd()
			if err != nil {
				return err
			}
			root, err := gitutil.CheckoutRoot(cwd, reserveStore)
			if err != nil {
				return &exitError{Code: 2, Msg: "abcd reserve: " + err.Error() + "; nothing was pushed"}
			}
			res, err := claim.Reserve(root, claim.Request{Record: args[0], Footprint: footprint, Lease: leaseFor,
				TakeOver: takeOver, Release: release}, claim.Options{})
			if err != nil {
				return reserveRefusal(err)
			}
			out := reserveView(res)
			return render(cmd.OutOrStdout(), *asJSON, out, func(w io.Writer) { renderReserve(w, out) })
		},
	}
	cmd.Flags().StringArrayVar(&footprint, "footprint", nil, "a repository-relative path the work will change; repeat once per path (never split on commas)")
	cmd.Flags().BoolVar(&takeOver, "take-over", false, "move your own live reservation from another copy to this one")
	cmd.Flags().DurationVar(&leaseFor, "lease", 0, "how long the reservation holds, from 1m to 24h (default 2h)")
	cmd.Flags().BoolVar(&release, "release", false, "release your reservation from this copy by pushing a tombstone")
	return cmd
}

// reserveRefusal maps the core's two refusal classes to their exit codes: the
// back-off (the record is held) exits 3, as `implement claim`'s contention
// does, and every other refusal exits 2. Each message already says nothing
// was pushed or no reservation was made, and quotes git's reason with the home
// directory redacted.
func reserveRefusal(err error) error {
	msg := "abcd reserve: " + termsafe.Sanitize(fsutil.RedactHome(err.Error()))
	switch {
	case errors.Is(err, claim.ErrBackOff):
		return &exitError{Code: 3, Msg: msg}
	case errors.Is(err, claim.ErrRefused):
		return &exitError{Code: 2, Msg: msg}
	}
	return &exitError{Code: 2, Msg: msg + "; no reservation was made"}
}

// reserveView is a result as a surface shows it: times in RFC 3339 UTC, every
// value a writer chose sanitised for the terminal.
func reserveView(res claim.Result) reserveOutput {
	out := reserveOutput{Act: res.Act, Record: res.Record, Holder: res.Holder, Copy: res.Copy,
		Footprint: make([]string, 0, len(res.Footprint)), ClaimedAt: rfc3339(res.ClaimedAt),
		EffectiveExpiry: rfc3339(res.ExpiresAt), Commit: res.Commit,
		DerivedUnchecked: termsafe.Sanitize(fsutil.RedactHome(res.DerivedUnchecked))}
	for _, f := range res.Footprint {
		out.Footprint = append(out.Footprint, termsafe.Sanitize(f))
	}
	if r := res.Replaced; r != nil {
		out.Replaced = &reserveReplaced{Holder: r.Holder, Copy: r.Copy, Commit: r.Commit,
			EffectiveExpiry: rfc3339(r.EffectiveExpiry), Unreadable: r.Unreadable}
	}
	return out
}

func rfc3339(t time.Time) string { return t.UTC().Truncate(time.Second).Format(time.RFC3339) }

// renderReserve is the text form: one line saying what was done, then the
// footprint, then what a take-over replaced and a skipped check, each when
// there is one.
func renderReserve(w io.Writer, out reserveOutput) {
	switch out.Act {
	case claim.ActReserved:
		fmt.Fprintf(w, "abcd reserve: reserved %s for %s from this copy until %s\n", out.Record, out.Holder, out.EffectiveExpiry)
	case claim.ActRenewed:
		fmt.Fprintf(w, "abcd reserve: renewed %s for %s from this copy until %s\n", out.Record, out.Holder, out.EffectiveExpiry)
	case claim.ActTakenOver:
		fmt.Fprintf(w, "abcd reserve: took over %s for %s from this copy until %s\n", out.Record, out.Holder, out.EffectiveExpiry)
	case claim.ActReleased:
		fmt.Fprintf(w, "abcd reserve: released %s\n", out.Record)
	case claim.ActAlreadyReleased:
		fmt.Fprintf(w, "abcd reserve: %s was already released; nothing was pushed\n", out.Record)
	}
	if len(out.Footprint) > 0 && out.Act != claim.ActAlreadyReleased {
		quoted := make([]string, len(out.Footprint))
		for i, f := range out.Footprint {
			quoted[i] = fmt.Sprintf("%q", f)
		}
		fmt.Fprintf(w, "  footprint: %s\n", strings.Join(quoted, ", "))
	}
	if r := out.Replaced; r != nil {
		switch {
		case r.Unreadable != "":
			fmt.Fprintf(w, "  replaced: the unreadable reservation %s (%s), held until %s\n", shortCommit(r.Commit), r.Unreadable, r.EffectiveExpiry)
		case r.Copy == claim.CopyAnother:
			fmt.Fprintf(w, "  replaced: your copy on another computer, which held it until %s\n", r.EffectiveExpiry)
		default:
			fmt.Fprintf(w, "  replaced: %s's reservation, which lapsed at %s\n", r.Holder, r.EffectiveExpiry)
		}
	}
	if out.DerivedUnchecked != "" {
		fmt.Fprintf(w, "  not checked: whether %s is already finished on origin's default branch (%s)\n", out.Record, out.DerivedUnchecked)
	}
}

func shortCommit(sha string) string {
	if len(sha) > 12 {
		return sha[:12]
	}
	return sha
}
