package cli

import (
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/intentdriven/abcd/internal/core/claim"
	"github.com/intentdriven/abcd/internal/core/embargo"
	"github.com/intentdriven/abcd/internal/core/launch"
	"github.com/intentdriven/abcd/internal/core/layered"
	"github.com/intentdriven/abcd/internal/fsutil"
	"github.com/intentdriven/abcd/internal/gitutil"
	"github.com/intentdriven/abcd/internal/termsafe"
	"github.com/spf13/cobra"
)

// The reservation acts this binary makes consult the embargo check an
// announcement does (claim.checkEmbargo): the reserve verb names it, and the
// callers that build their own options, the implement loop's build and drain
// among them, get it registered here.
func init() { claim.RegisterEmbargoCheck(reserveEmbargoCheck) }

// reserveEmbargoCheck is the announcement's embargo check, with the store
// that could not be read named when one could not, so the person is told
// which store to settle.
func reserveEmbargoCheck(root, home, text string, records []string) (bool, error) {
	held, err := announceEmbargoCheck(root, home, text, records)
	var r *embargo.RefusalError
	if err != nil && errors.As(err, &r) && r.Store != "" {
		err = fmt.Errorf("%w; the store at %s", err, r.Store)
	}
	return held, err
}

// reserveStore is the noun the checkout resolution names in its refusal.
const reserveStore = "the reservation"

// reserveForgeUser is the forge's user endpoint the verb's acts may ask whose
// account the session's token is; nil means the real call. Tests replace it so
// none touches the network.
var reserveForgeUser func(root string) (login string, id int64, err error)

// The launch check of the release hold (iss-2610222014550692) reads the
// merges armed on the forge and this copy's release reservation through the
// reservation package, which the launch package cannot import; this front
// door, which holds both, registers the reading.
func init() {
	launch.ReadArmedMerges = func(root string) (launch.ArmedMerges, error) {
		a, err := claim.ArmedMergesForCut(root, claim.Options{})
		switch {
		case errors.Is(err, claim.ErrNotOnGitHub):
			return launch.ArmedMerges{}, launch.ErrNotOnForge
		case err != nil:
			return launch.ArmedMerges{}, errors.New(termsafe.Sanitize(fsutil.RedactHome(err.Error())))
		}
		out := launch.ArmedMerges{Base: a.Base, Release: a.Release, Lapsed: a.Lapsed}
		if a.Lapsed != "" {
			out.LapsedAt = a.LapsedAt.UTC().Format(time.RFC3339)
		}
		if len(a.Admitted) > 0 {
			out.Admitted = map[int]string{}
			for _, w := range a.Admitted {
				out.Admitted[w.Number] = w.Head
			}
		}
		if len(a.Kept) > 0 {
			out.Kept = map[int]bool{}
			for _, w := range a.Kept {
				out.Kept[w.Number] = true
			}
		}
		if len(a.Unverified) > 0 {
			out.Unverified = map[int]string{}
			for _, w := range a.Unverified {
				out.Unverified[w.Number] = termsafe.Sanitize(fsutil.RedactHome(a.UnverifiedWhy))
			}
		}
		for _, l := range a.Lost {
			out.Lost = append(out.Lost, termsafe.Sanitize(l.Line()))
		}
		for _, pr := range a.Armed {
			out.Armed = append(out.Armed, launch.ArmedMerge{Number: pr.Number, Head: pr.Head, HeadRef: pr.HeadRef, CrossRepository: pr.CrossRepository,
				Queued: pr.Queued, AutoMerge: pr.AutoMerge, Method: pr.Method})
		}
		return out, nil
	}
}

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
	ForkUnchecked    string           `json:"fork_unchecked,omitempty"`
	ForkAllowed      string           `json:"fork_allowed,omitempty"`
	// HandedBack is the hand-back tombstone a reservation was made over, its
	// record unchanged on origin's default branch since; note is the
	// sentence the text form prints for it.
	HandedBack *reserveHandedBack `json:"handed_back,omitempty"`
	// HandBackKept is the hand-back a plain release kept standing, the
	// released reservation having been made over it with its record unchanged
	// since; HandBackUnpublished why a release's tombstone marks no hand-back
	// it would otherwise have marked.
	HandBackKept        *reserveHandedBack `json:"hand_back_kept,omitempty"`
	HandBackUnpublished string             `json:"hand_back_unpublished,omitempty"`
	// OwnOverlap is each path the reservation shares with the person's own
	// from another copy, which it was made over; the text form warns of each
	// on stderr. GeneratedShared is each generated file it shares with
	// another record, which the text form names on a shared: line.
	// GeneratedUnread is why no generated file was recognised.
	OwnOverlap      []reserveOwnOverlap     `json:"own_overlap,omitempty"`
	GeneratedShared []reserveGeneratedShare `json:"generated_shared,omitempty"`
	GeneratedUnread string                  `json:"generated_unread,omitempty"`
	// Withdrew is each merge a release reserved anew withdrew from origin's
	// default branch, and Withdrawals every withdrawal the commit pushed
	// records (iss-2610222014550692); ForgeFailures each pull request the
	// forge refused a withdrawal or re-arm of, and ForgeUnchecked why the
	// armed merges could not be listed at all. Admitted is the pull request
	// --admit re-armed; Stranded each withdrawal of another person's the
	// release carries; Settles each release whose withdrawals the act re-armed
	// or skipped. The text form warns of every failure on stderr.
	Withdrew       []claim.Withdrawal   `json:"withdrew,omitempty"`
	Withdrawals    []claim.Withdrawal   `json:"withdrawals,omitempty"`
	ForgeFailures  []claim.ForgeFailure `json:"forge_failures,omitempty"`
	ForgeUnchecked string               `json:"forge_unchecked,omitempty"`
	Admitted       *claim.Withdrawal    `json:"admitted,omitempty"`
	Stranded       []claim.Withdrawal   `json:"stranded,omitempty"`
	Settles        []claim.Settle       `json:"settles,omitempty"`
	// DirectoryFootprint is each footprint path naming a directory git
	// tracks, with the tracked files beneath it (iss-2610230016544142): the
	// reservation is made all the same, and the text form warns of each on
	// stderr.
	DirectoryFootprint []reserveDirectory `json:"directory_footprint,omitempty"`
	// DroppedFootprint is each path the reservation this act replaced, this
	// person's own, held and its footprint leaves out
	// (iss-2610290507512835): none is held now, and the text form warns of
	// them on stderr.
	DroppedFootprint []string `json:"dropped_footprint,omitempty"`
	note             string
	warnings         []string
}

// reserveDirectory is a footprint path naming a directory: the tracked files
// beneath it on the branch it was read from (-1 when git could not count
// them).
type reserveDirectory struct {
	Path         string `json:"path"`
	TrackedFiles int    `json:"tracked_files"`
	On           string `json:"on"`
}

// reserveOwnOverlap is a path shared with one's own reservation from another
// copy.
type reserveOwnOverlap struct {
	Path            string `json:"path"`
	Record          string `json:"record"`
	EffectiveExpiry string `json:"effective_expiry"`
	// Copy names the copy the other reservation was made from, as the
	// warning does (iss-2610290507526253), and CopyUnread why the worktrees
	// could not be listed when that left it "your other copy".
	Copy       string `json:"copy"`
	CopyUnread string `json:"copy_unread,omitempty"`
}

// reserveGeneratedShare is a generated file shared with another record's
// reservation.
type reserveGeneratedShare struct {
	Path   string `json:"path"`
	Record string `json:"record"`
}

// reserveHandedBack is the hand-back a person's own reservation was made over.
type reserveHandedBack struct {
	Kind   string `json:"kind"`
	Holder string `json:"holder"`
	At     string `json:"at"`
}

// reserveReplaced is the reservation a take-over replaced.
type reserveReplaced struct {
	Holder          string `json:"holder,omitempty"`
	Copy            string `json:"copy,omitempty"`
	YourCopy        string `json:"your_copy,omitempty"`
	Commit          string `json:"commit"`
	EffectiveExpiry string `json:"effective_expiry"`
	Unreadable      string `json:"unreadable,omitempty"`
}

// newReserveCommand builds the `reserve` verb — the front door onto
// internal/core/claim (itd-2609091034175565, spc-2610031753359849): before an
// agent starts on an issue or an intent, it reserves the record on the shared
// remote as a branch nobody merges, so a second person, a second computer or a
// hosted session sees it before starting the same item, and a session
// cutting a release reserves the release, which holds every other session's
// merge (itd-2609151838312703 C7). The verb is an
// agent's (the product thinker's ruling of 2026-10-12): a person sees
// reservations through `abcd peers`, never by typing this.
func newReserveCommand(asJSON *bool) *cobra.Command {
	var (
		footprint []string
		takeOver  bool
		release   bool
		leaseFor  time.Duration
		holder    bool
		admit     int
		sweep     *sweepFlags
	)
	cmd := &cobra.Command{
		Use: "reserve <record-id>",
		Long: "abcd reserve <record-id> --footprint <path> [--footprint <path>…] [--take-over] [--lease <duration>]\n" +
			"abcd reserve rel-vX.Y.Z --admit <pr>\n" +
			"abcd reserve <record-id> --release\n" +
			"abcd reserve --holder\n" +
			"abcd reserve --sweep [--apply] [--older-than <days>]\n\n" +
			"Reserve an issue or an intent on the shared remote before starting work on it, so\n" +
			"every other participant sees it is taken. The reservation is the branch\n" +
			"abcd-reserved/<record-id> on the remote named origin: a chain of empty-tree commits\n" +
			"naming the record, the holder (your GitHub username), this working copy, the files\n" +
			"the work will change (--footprint, once per repository-relative path) and the\n" +
			"lease (--lease, from 1m to 24h; default 2h, or 24h for a release).\n\n" +
			"A session cutting a release reserves the release instead, rel-vX.Y.Z, with no\n" +
			"footprint: while it is live every other session's land step holds its merge.\n" +
			"It holds for 24h unless --lease names less, sits outside the per-person cap,\n" +
			"and ends when the tag vX.Y.Z is on origin or its holder releases it; running\n" +
			"the same command again from the same copy renews it.\n\n" +
			"A release reserved anew (not renewed from the same copy while live) also withdraws\n" +
			"every pull request into origin's default branch that has auto-merge on or sits in\n" +
			"the merge queue, but the release pull request from this checkout's branch,\n" +
			"through gh under your own identity (refused when gh acts as another account),\n" +
			"and records each on the reservation first, with the form it was armed in and its\n" +
			"head (withdrew in --json). A withdrawal the forge refuses is named and the\n" +
			"reservation stands (forge_failures); the launch check then refuses the cut while\n" +
			"that pull request is armed. One release is cut at a time: another release live\n" +
			"backs this one off. --admit <pr>, from the copy cutting the release, records one\n" +
			"withdrawn pull request admitted, then re-arms it in its recorded form, pinned to\n" +
			"its withdrawn head. --release from that copy pushes the tombstone, then re-arms\n" +
			"every other one the same way; once the tag is on origin, any act of a copy of\n" +
			"the same person that fetches the reservations does so, and so does one five\n" +
			"minutes after the lease lapsed with no tag, for the cut's own person alone; a cut\n" +
			"still going on renews it by reserving again. One\n" +
			"whose head moved, that was closed or merged, or that can only be merged by hand\n" +
			"(no merge queue), is skipped and named for its owner; every state is recorded on\n" +
			"the tombstone (settles in --json), so no copy acts twice, a copy of another\n" +
			"person never re-arms (stranded in --json). A person's withdrawals are read only\n" +
			"from their own newest push to the release's branch, as the forge's record of\n" +
			"pushes shows it: commits others pushed above it are counted and never read\n" +
			"(not_read in settles), and when that push cannot be found or placed nothing is\n" +
			"acted on for that person (unplaced in settles).\n\n" +
			"Every act fetches the reservations from origin, judges the tip, writes one\n" +
			"commit and makes one plain fast-forward push, with the repository's hooks\n" +
			"running: a reservation, a renewal by the same person and copy, a take-over of\n" +
			"a reservation lapsed past its expiry by five minutes, or a release (--release),\n" +
			"which pushes a tombstone. --take-over moves your own live reservation from\n" +
			"another copy to this one; it never moves another person's. A record already\n" +
			"resolved, won't-fixed, shipped or superseded on origin's default branch counts\n" +
			"as released whatever its reservation says, and so does a release whose tag is\n" +
			"on origin.\n\n" +
			"A new reservation or a take-over asks GitHub whether origin is a fork, and a\n" +
			"fork is refused unless reservation.fork_origin names it; when GitHub cannot say,\n" +
			"the act goes on and warns (not checked: whether origin is a fork, fork_unchecked\n" +
			"in --json). An issue a drain handed back carries the hand-back on its tombstone:\n" +
			"your own reservation over it is made, with a note (handed_back in --json), and a\n" +
			"plain --release of it keeps the hand-back standing until a person changes the\n" +
			"record on origin's default branch (hand_back_kept in --json), so no drain takes\n" +
			"the issue, nor takes over or renews your reservation over it. A release whose\n" +
			"tombstone cannot mark the hand-back is still made, plain, and says why\n" +
			"(hand_back_unpublished in --json).\n\n" +
			"A footprint sharing a path with another person's live reservation backs off\n" +
			"(exit 3). A file that origin's default branch, as last fetched, declares\n" +
			"linguist-generated in its root .gitattributes is shared, never an overlap, and\n" +
			"is named on a shared: line for whoever lands second to regenerate after\n" +
			"merging (generated_shared in --json); a directory never is. When that\n" +
			"declaration cannot be read nothing is shared, and the act says why\n" +
			"(generated_unread in --json). Your own reservation sharing a path with yours\n" +
			"from another copy is made, and warns on stderr, naming that copy (own_overlap\n" +
			"in --json). Name files, not directories: a directory holds off every\n" +
			"reservation of a file inside it. A footprint path naming a directory git\n" +
			"tracks is reserved all the same, and warns on stderr, naming it and how many\n" +
			"tracked files it covers on origin's default branch, or HEAD\n" +
			"(directory_footprint in --json). A path named twice is written once. A reserve\n" +
			"of a record you already hold writes its footprint in place of the one held,\n" +
			"and warns on stderr, naming each held path it leaves out (dropped_footprint in\n" +
			"--json): name them again to keep holding them.\n\n" +
			"A reservation, a renewal and a take-over consult the embargo store on this\n" +
			"machine before anything is pushed, as an announcement does: a record the store\n" +
			"holds, or a footprint naming something under embargo, is refused (exit 2) with\n" +
			"nothing pushed, and so is the act when a store cannot be read, naming that\n" +
			"store. A release does not consult it. A build's and a drain's reservations\n" +
			"consult it too.\n\n" +
			"The holder is your GitHub username, from the first of these that names it:\n" +
			"user.email, when it is a GitHub no-reply address; reservation.github_username\n" +
			"in the machine configuration (the repository's own configuration is never read\n" +
			"for it); and the account this session's own GitHub token belongs to, which\n" +
			"takes one call to GitHub's user endpoint, through gh. When git's identity is\n" +
			"not a person's (unset, or a machine's), the token alone names the holder, that\n" +
			"account authors the reservation commit, and the identity git had is shown\n" +
			"nowhere. The checkout must have its local tier (.abcd/.work.local/), where this\n" +
			"copy's key is kept. `abcd peers` lists every reservation.\n\n" +
			"--holder prints the account an act from this checkout would reserve as and the\n" +
			"source that named it (git, config or token), or its --json document. It may\n" +
			"make that one call to the user endpoint in two cases: git's identity is a\n" +
			"person's but carries neither a GitHub no-reply address nor a machine key; or\n" +
			"git's identity is not a person's (unset, or a machine's), when the machine key\n" +
			"is skipped. It takes no record id and no other flag, and reserves, writes and\n" +
			"pushes nothing. It exits 0 when the account is known and 2 when not.\n\n" +
			"Exit 0 when the act reached origin (or a release was already made); exit 3 when\n" +
			"the record is held by someone else or by your other copy, so back off; exit 2\n" +
			"on every other refusal and failure. Nothing is pushed on a refusal, and a failed\n" +
			"push leaves no reservation and nothing in the checkout.\n\n" +
			"--sweep lists the spent reservation branches on origin and, with --apply, deletes\n" +
			"them, each with a lease on the exact tip it judged, so a branch re-reserved\n" +
			"meanwhile survives. A branch is spent when its tip has settled, its lease past\n" +
			"the five-minute margin included (a plain release, or a hand-back on a finished\n" +
			"record, at once, at the next sweep; any other tip only once untouched for\n" +
			"--older-than days, default 7, at least 1),\n" +
			"and either its record is resolved, won't-fixed, shipped or superseded\n" +
			"on origin's default branch, or every commit on it is a reservation or a plain\n" +
			"release, whether or not any is a release, since a lapsed lease already lets any\n" +
			"session take the record over. A hand-back on a record still open, a commit of\n" +
			"a kind the sweep cannot classify, a history it cannot read to its root, and an\n" +
			"unreadable tip keep the branch.\n" +
			"The reservations ruleset lets only its bypass actor delete these branches, so\n" +
			"the scheduled reservation-sweep workflow is what applies it; anyone else's\n" +
			"--apply is refused by origin. It takes no record id and no other flag; exit 0\n" +
			"when it ran (a branch that moved since it was judged is kept, not a failure),\n" +
			"exit 2 when it could not run or origin refused a deletion.",
		Args: func(cmd *cobra.Command, args []string) error {
			if err := sweep.args(cmd, args); err != nil || sweep.on {
				return err
			}
			if holder {
				var with []string
				if len(args) > 0 {
					with = append(with, "record id")
				}
				for _, f := range []string{"footprint", "take-over", "lease", "release", "admit"} {
					if cmd.Flags().Changed(f) {
						with = append(with, "--"+f)
					}
				}
				if len(with) > 0 {
					return &exitError{Code: 2, Msg: "abcd reserve: --holder takes no " + strings.Join(with, ", ") + "; nothing was reserved"}
				}
				return nil
			}
			switch len(args) {
			case 0:
				return &exitError{Code: 2, Msg: "abcd reserve: name the issue or intent to reserve (abcd reserve <record-id> --footprint <path>), or the release being cut (abcd reserve rel-vX.Y.Z); `abcd peers` lists the reservations; nothing was pushed"}
			case 1:
				return nil
			}
			return &exitError{Code: 2, Msg: "abcd reserve: takes one record id, not " + fmt.Sprint(len(args)) + "; nothing was pushed"}
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			if sweep.on {
				return runReserveSweep(cmd.OutOrStdout(), *asJSON, sweep)
			}
			if holder {
				return runReserveHolder(cmd.OutOrStdout(), cmd.ErrOrStderr(), *asJSON)
			}
			for _, lone := range []struct {
				flag string
				on   bool
			}{{"--release", release}, {"--admit", cmd.Flags().Changed("admit")}} {
				if !lone.on {
					continue
				}
				var with []string
				for _, f := range []string{"footprint", "take-over", "lease", "release", "admit"} {
					if "--"+f != lone.flag && cmd.Flags().Changed(f) {
						with = append(with, "--"+f)
					}
				}
				if len(with) > 0 {
					return &exitError{Code: 2, Msg: "abcd reserve: " + lone.flag + " takes no " + strings.Join(with, ", ") + "; nothing was pushed"}
				}
			}
			if cmd.Flags().Changed("admit") && admit <= 0 {
				return &exitError{Code: 2, Msg: fmt.Sprintf("abcd reserve: --admit takes a pull request number, not %d; nothing was pushed", admit)}
			}
			cwd, err := os.Getwd()
			if err != nil {
				return err
			}
			root, err := gitutil.CheckoutRoot(cwd, reserveStore)
			if err != nil {
				return &exitError{Code: 2, Msg: "abcd reserve: " + err.Error() + "; nothing was pushed"}
			}
			noteMachineHome(cmd.ErrOrStderr(), "abcd reserve")
			res, err := claim.Reserve(root, claim.Request{Record: args[0], Footprint: footprint, Lease: leaseFor,
				TakeOver: takeOver, Release: release, Admit: admit, WarnOwnOverlap: true}, claim.Options{Embargo: reserveEmbargoCheck, ForgeUser: reserveForgeUser})
			if err != nil {
				return reserveRefusal(err)
			}
			out := reserveView(res)
			if err := render(cmd.OutOrStdout(), *asJSON, out, func(w io.Writer) { renderReserve(w, out) }); err != nil {
				return err
			}
			if !*asJSON {
				for _, line := range out.warnings {
					fmt.Fprintf(cmd.ErrOrStderr(), "abcd reserve: warning: %s\n", line)
				}
			}
			return nil
		},
	}
	cmd.Flags().StringArrayVar(&footprint, "footprint", nil, "a repository-relative path the work will change; repeat once per path (never split on commas)")
	cmd.Flags().BoolVar(&takeOver, "take-over", false, "move your own live reservation from another copy to this one")
	cmd.Flags().DurationVar(&leaseFor, "lease", 0, "how long the reservation holds, from 1m to 24h (default 2h, or 24h for a release)")
	cmd.Flags().BoolVar(&release, "release", false, "release your reservation from this copy by pushing a tombstone")
	cmd.Flags().IntVar(&admit, "admit", 0, "on a release reservation, from the copy cutting it: re-arm this withdrawn pull request and record it admitted")
	cmd.Flags().BoolVar(&holder, "holder", false, "say which GitHub account an act would reserve as, and from where; reserves nothing")
	sweep = addSweepFlags(cmd)
	return cmd
}

// runReserveHolder is `reserve --holder`: the account a reservation act from
// this checkout would be made as, and the source that named it. It writes
// nothing; when git's identity names no holder it may ask the forge's user
// endpoint once, as the act itself would.
func runReserveHolder(w, errw io.Writer, asJSON bool) error {
	cwd, err := os.Getwd()
	if err != nil {
		return err
	}
	root, err := gitutil.CheckoutRoot(cwd, reserveStore)
	if err != nil {
		return &exitError{Code: 2, Msg: "abcd reserve: " + err.Error() + "; nothing was reserved"}
	}
	noteMachineHome(errw, "abcd reserve")
	h, err := claim.ResolveHolder(root, claim.Options{ForgeUser: reserveForgeUser})
	if err != nil {
		return &exitError{Code: 2, Msg: "abcd reserve: " + termsafe.Sanitize(fsutil.RedactHome(err.Error()))}
	}
	return render(w, asJSON, h, func(w io.Writer) { fmt.Fprintf(w, "would reserve as %s (from %s)\n", h.Login, h.Source) })
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
		DerivedUnchecked: termsafe.Sanitize(fsutil.RedactHome(res.DerivedUnchecked)),
		ForkUnchecked:    termsafe.Sanitize(fsutil.RedactHome(res.ForkUnchecked)),
		ForkAllowed:      termsafe.Sanitize(res.ForkAllowed)}
	for _, f := range res.Footprint {
		out.Footprint = append(out.Footprint, termsafe.Sanitize(f))
	}
	if r := res.Replaced; r != nil {
		out.Replaced = &reserveReplaced{Holder: r.Holder, Copy: r.Copy, YourCopy: termsafe.Sanitize(r.YourCopy), Commit: r.Commit,
			EffectiveExpiry: rfc3339(r.EffectiveExpiry), Unreadable: r.Unreadable}
	}
	if h := res.HandedBack; h != nil {
		out.HandedBack = &reserveHandedBack{Kind: termsafe.Sanitize(h.Kind), Holder: termsafe.Sanitize(h.Holder), At: rfc3339(h.At)}
		out.note = termsafe.Sanitize(h.Note(res.Record))
	}
	if h := res.HandBackKept; h != nil {
		out.HandBackKept = &reserveHandedBack{Kind: termsafe.Sanitize(h.Kind), Holder: termsafe.Sanitize(h.Holder), At: rfc3339(h.At)}
		out.note = termsafe.Sanitize(h.KeptNote(res.Record))
	}
	out.HandBackUnpublished = termsafe.Sanitize(fsutil.RedactHome(res.HandBackUnpublished))
	out.GeneratedUnread = termsafe.Sanitize(fsutil.RedactHome(res.GeneratedUnread))
	for _, o := range res.OwnOverlap {
		out.OwnOverlap = append(out.OwnOverlap, reserveOwnOverlap{Path: termsafe.Sanitize(o.Path), Record: termsafe.Sanitize(o.Record),
			EffectiveExpiry: rfc3339(o.Expiry), Copy: termsafe.Sanitize(o.Copy), CopyUnread: termsafe.Sanitize(o.CopyUnread)})
		out.warnings = append(out.warnings, o.Warning())
	}
	out.Withdrew, out.Withdrawals, out.Stranded = res.Withdrew, res.Withdrawals, res.Stranded
	out.Admitted, out.Settles = res.Admitted, res.Settles
	out.ForgeUnchecked = termsafe.Sanitize(fsutil.RedactHome(res.ForgeUnchecked))
	for _, f := range res.ForgeFailures {
		f.Reason = termsafe.Sanitize(fsutil.RedactHome(f.Reason))
		out.ForgeFailures = append(out.ForgeFailures, f)
		out.warnings = append(out.warnings, f.Line())
	}
	if out.ForgeUnchecked != "" {
		out.warnings = append(out.warnings, out.ForgeUnchecked)
	}
	for i, st := range out.Settles {
		st.Failed = termsafe.Sanitize(fsutil.RedactHome(st.Failed))
		for j, f := range st.Failures {
			st.Failures[j].Reason = termsafe.Sanitize(fsutil.RedactHome(f.Reason))
			out.warnings = append(out.warnings, st.Failures[j].Line())
		}
		if st.Failed != "" {
			out.warnings = append(out.warnings, st.Failed)
		}
		out.Settles[i] = st
	}
	for _, g := range res.GeneratedShared {
		out.GeneratedShared = append(out.GeneratedShared, reserveGeneratedShare{Path: termsafe.Sanitize(g.Path), Record: termsafe.Sanitize(g.Record)})
	}
	for _, d := range res.Directories {
		out.DirectoryFootprint = append(out.DirectoryFootprint, reserveDirectory{Path: termsafe.Sanitize(d.Path),
			TrackedFiles: d.Files, On: termsafe.Sanitize(d.On)})
		out.warnings = append(out.warnings, d.Warning())
	}
	for _, p := range res.Dropped {
		out.DroppedFootprint = append(out.DroppedFootprint, termsafe.Sanitize(p))
	}
	if len(res.Dropped) > 0 {
		out.warnings = append(out.warnings, claim.DroppedLine(res.Dropped))
	}
	return out
}

func rfc3339(t time.Time) string { return t.UTC().Truncate(time.Second).Format(time.RFC3339) }

// renderReserve is the text form: one line saying what was done, then the
// footprint, then what a take-over replaced, a skipped check and the hand-back
// the reservation was made over, each when there is one.
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
	case claim.ActAdmitted:
		if a := out.Admitted; a != nil {
			fmt.Fprintf(w, "abcd reserve: admitted #%d into %s, re-armed (%s) at %s; renewed until %s\n", a.Number, out.Record, a.Form, shortCommit(a.Head), out.EffectiveExpiry)
		}
	}
	if len(out.Withdrew) > 0 {
		parts := make([]string, len(out.Withdrew))
		for i, wd := range out.Withdrew {
			parts[i] = fmt.Sprintf("#%d (%s at %s)", wd.Number, wd.Form, shortCommit(wd.Head))
		}
		fmt.Fprintf(w, "  withdrew: %s; each is re-armed when the release ends, or now with --admit <pr>\n", strings.Join(parts, ", "))
	}
	for _, st := range out.Settles {
		for _, wd := range st.Settled {
			fmt.Fprintf(w, "  %s\n", claim.SettledLine(st.Subject, wd))
		}
		for _, line := range st.Notes() {
			fmt.Fprintf(w, "  note: %s\n", line)
		}
	}
	if len(out.Stranded) > 0 {
		fmt.Fprintf(w, "  note: %s\n", claim.StrandedLine(out.Record, out.Stranded))
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
			fmt.Fprintf(w, "  replaced: %s, which held it until %s\n", r.YourCopy, r.EffectiveExpiry)
		default:
			fmt.Fprintf(w, "  replaced: %s's reservation, which lapsed at %s\n", r.Holder, r.EffectiveExpiry)
		}
	}
	if out.DerivedUnchecked != "" {
		fmt.Fprintf(w, "  not checked: whether %s is already finished on origin's default branch (%s)\n", out.Record, out.DerivedUnchecked)
	}
	if out.ForkUnchecked != "" {
		fmt.Fprintf(w, "  %s\n", claim.ForkUncheckedLine(out.ForkUnchecked))
	}
	if out.ForkAllowed != "" {
		fmt.Fprintf(w, "  %s\n", claim.ForkAllowedLine(out.ForkAllowed))
	}
	if out.note != "" {
		fmt.Fprintf(w, "  note: %s\n", out.note)
	}
	if out.HandBackUnpublished != "" {
		fmt.Fprintf(w, "  no hand-back: %s\n", out.HandBackUnpublished)
	}
	for _, g := range out.GeneratedShared {
		fmt.Fprintf(w, "  shared: %s\n", claim.GeneratedShare{Path: g.Path, Record: g.Record}.Line())
	}
	if out.GeneratedUnread != "" {
		fmt.Fprintf(w, "  %s\n", claim.GeneratedUnreadLine(out.GeneratedUnread))
	}
}

func shortCommit(sha string) string {
	if len(sha) > 12 {
		return sha[:12]
	}
	return sha
}

// noteMachineHome puts on stderr, prefixed by verb, the one line saying the
// process's HOME lies inside a git working tree and so is not read as this
// machine's layer (layered.MachineRoots), the home claim's readers are given.
// It writes nothing for any other home.
func noteMachineHome(w io.Writer, verb string) {
	home, err := os.UserHomeDir()
	if err != nil {
		return
	}
	if _, note := layered.MachineRoots(home); note != "" {
		fmt.Fprintln(w, verb+": "+termsafe.Sanitize(fsutil.RedactHome(note)))
	}
}
