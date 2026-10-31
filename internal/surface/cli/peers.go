package cli

import (
	"errors"
	"fmt"
	"io"
	"os"
	"slices"
	"sort"
	"strings"
	"time"

	"github.com/intentdriven/abcd/internal/abcdhome"
	"github.com/intentdriven/abcd/internal/core/claim"
	"github.com/intentdriven/abcd/internal/core/peers"
	"github.com/intentdriven/abcd/internal/fsutil"
	"github.com/intentdriven/abcd/internal/gitutil"
	"github.com/intentdriven/abcd/internal/termsafe"
	"github.com/spf13/cobra"
)

// peersStore is the noun the checkout resolution names in its refusal.
const peersStore = "the peer listing"

// peersOutput is the --json envelope of the peers verb: the core's report with
// every path home-redacted, plus the two counts the board line carries, so a
// consumer reads them without recounting.
type peersOutput struct {
	Sources    []peers.Source  `json:"sources"`
	DefaultRef string          `json:"default_ref,omitempty"`
	Live       int             `json:"live"`
	IDs        int             `json:"ids"`
	Peers      []peers.Peer    `json:"peers"`
	Skipped    []peers.Skipped `json:"skipped"`

	reservationsBlock
}

// reservationsBlock is the reservations as a listing shows them, `abcd
// peers`'s and `abcd status`'s alike: its fields sit in the listing's own
// JSON object.
type reservationsBlock struct {
	// The reservations on the shared remote (itd-2609091034175565): every one
	// as fetched just now, or as last fetched when the fetch failed.
	Reservations []reservationRow `json:"reservations"`
	LiveByHolder map[string]int   `json:"live_by_holder"`
	// ReleaseHolds is every release being cut, which holds every other
	// session's merge (itd-2609151838312703 C7, C8).
	ReleaseHolds []holdRow `json:"release_holds"`
	// LapsedReleases is every release whose cut stopped renewing its
	// reservation with no tag on origin: it holds no merge, and is named so
	// that no session waits on a holder that is gone (iss-2610241104098581).
	LapsedReleases []lapsedRow `json:"lapsed_releases"`
	// HandBacksByHolder is each holder's hand-back tombstones of the last day,
	// newest first and bounded (spc-2610161044461448 step 18).
	HandBacksByHolder []handBackHolder `json:"hand_backs_by_holder"`
	// ReleaseWithdrawals is each release whose tip records withdrawn pull
	// requests worth naming (iss-2610222014550692): those it holds, those
	// still to re-arm (this person's, due on their next act, or another
	// person's, stranded for them) and those skipped for their owners.
	ReleaseWithdrawals []claim.ReleaseWithdrawals `json:"release_withdrawals,omitempty"`
	// LostWithdrawals is each recent release whose branch history records
	// withdrawals a later commit dropped (an older abcd's renewal), which
	// nothing re-arms (dcn-2610230328242929).
	LostWithdrawals []claim.LostWithdrawals `json:"lost_withdrawals,omitempty"`
	// Settled is each ended release whose withdrawals of this person's the
	// listing re-armed or skipped after its fetch (`abcd peers` only).
	Settled []claim.Settle `json:"settled,omitempty"`
	// Ceiling is the reader's live reservations against this machine's cap,
	// by copy, with its warning (iss-2610231004140527); absent when the
	// reader's login cannot be named or the cap cannot be read.
	Ceiling          *ceilingRow     `json:"ceiling,omitempty"`
	View             reservationView `json:"view"`
	FetchError       string          `json:"fetch_error,omitempty"`
	DerivedUnchecked string          `json:"derived_unchecked,omitempty"`
	// HolderIDsRefused is the class of the refusal of the declared ids'
	// file when the guarded read refused it, so the listing read none of its
	// ids as the reader's (claim.HolderIDsRefusedLine); stderr says so too.
	HolderIDsRefused string `json:"holder_ids_refused,omitempty"`
	// HolderIDRefused is the class of the refusal of the reader's holder id
	// (ABCD_HOLDER_ID set to no id, or the kept file refused), so the
	// listing read no id as the reader's (claim.HolderIDRefusedLine); stderr
	// says so too.
	HolderIDRefused string `json:"holder_id_refused,omitempty"`
}

// noteHolderIDsRefused writes the one line saying the declared ids' file
// was refused as class, so no declared id was read, to errw under verb; it
// writes nothing for no class.
func noteHolderIDsRefused(errw io.Writer, verb, class string) {
	if l := claim.HolderIDsRefusedLine(class); l != "" {
		fmt.Fprintln(errw, verb+": "+l)
	}
}

// noteHolderIDRefused writes the one line saying the reader's holder id was
// refused as class, so no id was read, to errw under verb; it writes nothing
// for no class.
func noteHolderIDRefused(errw io.Writer, verb, class string) {
	if l := claim.HolderIDRefusedLine(class); l != "" {
		fmt.Fprintln(errw, verb+": "+l)
	}
}

// reservationRow is one reservation as the listing shows it. Every value a
// writer chose is terminal-safe here and quoted in the text form.
type reservationRow struct {
	Record string `json:"record"`
	Holder string `json:"holder,omitempty"`
	Copy   string `json:"copy,omitempty"`
	// Sibling is the other worktree of this repository, on this machine,
	// whose copy made the reservation: its folder's name, with parent folders
	// added only where two would otherwise read alike, never its key.
	Sibling         string   `json:"sibling,omitempty"`
	Footprint       []string `json:"footprint"`
	ClaimedAt       string   `json:"claimed_at,omitempty"`
	EffectiveExpiry string   `json:"effective_expiry"`
	State           string   `json:"state"`
	ReleasedBy      string   `json:"released_by,omitempty"`
	Unreadable      string   `json:"unreadable,omitempty"`
	HeldFromFetch   bool     `json:"held_from_fetch,omitempty"`
	// Stage is the drain lane's stage a live reservation's renewal names, one
	// of claim.LaneStages, absent when it names none or is not live.
	Stage  string `json:"stage,omitempty"`
	Commit string `json:"commit"`
}

// handBackHolder is one holder's hand-backs of the last day, as the listing
// shows them: the newest claim.HandBacksShown, newest first, and how many more
// the day holds.
type handBackHolder struct {
	Holder    string        `json:"holder"`
	HandBacks []handBackRow `json:"hand_backs"`
	More      int           `json:"more"`
}

// handBackRow is one hand-back tombstone: its record, its kind (one of
// claim.HandBackKinds) and its Abcd-Released-At.
type handBackRow struct {
	Record     string `json:"record"`
	Kind       string `json:"kind"`
	ReleasedAt string `json:"released_at"`
}

// holdRow is one release being cut, as the listing shows it: its release and
// subject, its holder (absent when the reservation is unreadable, with the
// reason), whose copy it is when it is this one's, until when it holds,
// whether it is this copy's own cut (own) or dated ahead with no first
// sighting kept here (unsighted), neither of which the merge hold counts
// (claim.MergeHolds), and the line it is shown by.
type holdRow struct {
	Release         string `json:"release"`
	Subject         string `json:"subject"`
	Holder          string `json:"holder,omitempty"`
	Copy            string `json:"copy,omitempty"`
	EffectiveExpiry string `json:"effective_expiry"`
	Unreadable      string `json:"unreadable,omitempty"`
	Own             bool   `json:"own"`
	Unsighted       bool   `json:"unsighted"`
	Text            string `json:"text"`
}

// lapsedRow is one lapsed release cut as the listing shows it (claim.Lapsed).
type lapsedRow struct {
	Release   string `json:"release"`
	Subject   string `json:"subject"`
	Holder    string `json:"holder"`
	Copy      string `json:"copy,omitempty"`
	RenewedAt string `json:"renewed_at"`
	LapsedAt  string `json:"lapsed_at"`
	Text      string `json:"text"`
}

// ceilingRow is the reader's ceiling as the listing shows it.
type ceilingRow struct {
	claim.Ceiling
	Warning string `json:"warning,omitempty"`
}

// reservationView dates the listing: when abcd last fetched the reservations
// here and how long ago, both absent when nothing here stamped a fetch, and
// whether the checkout has a remote named origin to fetch them from.
type reservationView struct {
	Origin     bool   `json:"origin"`
	FetchedAt  string `json:"fetched_at,omitempty"`
	AgeSeconds *int64 `json:"age_seconds"`
}

// newPeersCommand builds the `peers` verb — the front door onto
// internal/core/peers (itd-2609091416295622, spc-2609202056480020): what the
// sibling worktrees and the local branches of this checkout hold that this tree
// does not, before a session captures, fixes or files anything.
func newPeersCommand(asJSON *bool) *cobra.Command {
	return &cobra.Command{
		Use: "peers",
		Long: "List what this checkout's peers hold that it does not, before capturing, fixing or\n" +
			"filing anything. A peer is a linked worktree sharing this repository's git\n" +
			"common dir, read off its disk so an uncommitted capture is seen, or a local\n" +
			"branch no worktree has checked out, read from the object store.\n\n" +
			"Per peer, three kinds of row: an issue open there and absent here; an issue\n" +
			"open here and resolved or won't-fixed there; an intent drafted there and\n" +
			"absent here. A peer whose worktree directory is gone, or whose branch is\n" +
			"merged into the default branch (a worktree only when its record folders are\n" +
			"also clean), is skipped and counted. A peer git refuses to answer for, one\n" +
			"whose common dir is another repository's, or one whose ledger holds an id in\n" +
			"two status folders is named with the reason and not read; a gone or refused\n" +
			"worktree's branch is then read from the object store instead.\n\n" +
			"Then the reservations on the shared remote: one fetch from origin of every\n" +
			"abcd-reserved/ branch, with pruning, and of the default branch, then every\n" +
			"reservation with its record, holder, footprint, effective expiry and state\n" +
			"(live, expired, released, or unreadable with its reason and the end of its\n" +
			"hold), a live drain lane's stage when its renewal names one, the worktree of\n" +
			"this repository on this machine that made one (by its folder's name, with\n" +
			"parent folders added only where two would otherwise read alike),\n" +
			"each person's count of live reservations, the reader's own counted by copy\n" +
			"(this copy, each such worktree, elsewhere) against this machine's\n" +
			"reservation.max_live with a warning from four fifths of it, since every\n" +
			"session and account on the same login counts toward the same total, and a\n" +
			"line for each release being cut (a live\n" +
			"rel-vX.Y.Z reservation, which holds every other session's merge until its\n" +
			"tag is on origin). Then each holder's hand-backs of the last day, from the\n" +
			"same view: the record and kind of each released tip marking one, newest\n" +
			"first, the newest ten shown and the rest counted; one dated more than five\n" +
			"minutes ahead of this computer's clock is not shown. Then each release's\n" +
			"withdrawn pull requests (release_withdrawals): those a live release holds or\n" +
			"admitted, a lease that lapsed before the release ended (ended by its own\n" +
			"person's next act five minutes after), those an ended one has still to re-arm\n" +
			"(yours, or another person's, named\n" +
			"with whose), and those its tombstone skipped for their owners; and those a\n" +
			"later commit on a recent release's branch dropped (lost_withdrawals). The fetch\n" +
			"reads origin's release tags with the reservations; it updates the\n" +
			"remote-tracking reservation branches and the release tags abcd keeps under\n" +
			"refs/abcd/release-tags/, stamps when it ran in the local tier, with when it\n" +
			"first fetched each reservation dated ahead of this computer's clock or\n" +
			"undated, and keeps there each verdict of the forge's record of pushes that\n" +
			"never changes. One more act follows it, only when a release has ended (its tag on\n" +
			"origin, or a tombstone its holder pushed) with pull requests you withdrew for\n" +
			"it not yet re-armed: a tombstone is pushed first, then each one the forge's\n" +
			"record of pushes says you recorded is re-armed through gh in the form you\n" +
			"recorded, pinned to its withdrawn head, or skipped when its head\n" +
			"moved, it was closed or merged, or it can only be merged by hand, and a second\n" +
			"tombstone records every state (settled). Each pull request re-armed, found\n" +
			"armed or skipped for its owner then gets one fixed comment, and a comment\n" +
			"that fails is named in settled.failures. Nothing else is written or posted. The release-tag refs are ordinary\n" +
			"refs of this checkout, so git log --all lists the commits they name and\n" +
			"git push --mirror would push them. A checkout with no remote named origin\n" +
			"lists no reservations.\n" +
			"When the fetch fails, the reservations are listed as last fetched, with that\n" +
			"view's age, and the failure is named on stderr and in fetch_error.\n\n" +
			"When " + abcdhome.Display("holder-ids") + " is there and its guarded read refuses it, no\n" +
			"declared id is read as yours, and one line on stderr names the class of the\n" +
			"refusal, never the file's contents (holder_ids_refused in --json).\n\n" +
			"When your holder id is set or kept but refused (ABCD_HOLDER_ID set to\n" +
			"anything but one holder id, or " + abcdhome.Display("holder-id") + " refused by its\n" +
			"guarded read), no hold or withdrawal under it is read as yours, and one line\n" +
			"on stderr names the class of the refusal, never the value or the file's\n" +
			"contents (holder_id_refused in --json).\n\n" +
			"A worktree is named home-relative (~/...), or by its directory name when it\n" +
			"sits outside HOME, on every stream. Exit 0 whatever the peers hold; exit 1 when\n" +
			"the reservations could not be fetched; exit 2 outside a git checkout.",
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			cwd, err := os.Getwd()
			if err != nil {
				return err
			}
			root, err := gitutil.CheckoutRoot(cwd, peersStore)
			if err != nil {
				return &exitError{Code: 2, Msg: "abcd peers: " + err.Error() + " (nothing read)"}
			}
			rep, err := peers.Read(root)
			if err != nil {
				return fmt.Errorf("abcd peers: %w", err)
			}
			noteMachineHome(cmd.ErrOrStderr(), "abcd peers")
			out := peersView(root, rep)
			failed := addReservations(root, &out, time.Now())
			if err := render(cmd.OutOrStdout(), *asJSON, out, func(w io.Writer) { renderPeers(w, out) }); err != nil {
				return err
			}
			noteHolderIDsRefused(cmd.ErrOrStderr(), "abcd peers", out.HolderIDsRefused)
			noteHolderIDRefused(cmd.ErrOrStderr(), "abcd peers", out.HolderIDRefused)
			if failed != "" {
				// The last view is rendered; the failure is named and exits 1.
				fmt.Fprintln(cmd.ErrOrStderr(), "abcd peers: "+failed)
				return &exitError{Code: 1}
			}
			return nil
		},
	}
}

// peersView is the report as a surface shows it: paths and reasons redacted,
// each worktree named by fsutil.DisplayPath, so one outside HOME is its
// directory name rather than an absolute local path, and each not-read reason
// by peers.Peer.NotReadShown, the form the drain and the build print, so a
// path the reason names outside HOME is no absolute local path either
// (iss-2609281329007423).
func peersView(root string, rep peers.Report) peersOutput {
	out := peersOutput{Sources: rep.Sources, DefaultRef: rep.DefaultRef, Live: rep.Live(), IDs: rep.IDCount(),
		Peers: make([]peers.Peer, 0, len(rep.Peers)), Skipped: make([]peers.Skipped, 0, len(rep.Skipped))}
	for _, p := range rep.Peers {
		p.NotRead = p.NotReadShown(root)
		p.Path = fsutil.DisplayPath(p.Path)
		out.Peers = append(out.Peers, p)
	}
	for _, s := range rep.Skipped {
		s.Path = fsutil.DisplayPath(s.Path)
		out.Skipped = append(out.Skipped, s)
	}
	return out
}

// renderPeers is the text form: one summary line, then a block only for a peer
// with something to say, so a session reading it before every pick reads a
// line or two, not one per worktree.
func renderPeers(w io.Writer, out peersOutput) {
	renderWorktreePeers(w, out)
	renderReservations(w, out.reservationsBlock)
}

func renderWorktreePeers(w io.Writer, out peersOutput) {
	if out.Live == 0 && len(out.Skipped) == 0 {
		fmt.Fprintln(w, "abcd peers — no peers")
		return
	}
	fmt.Fprintf(w, "abcd peers — %s, %s differing here%s\n",
		countOf(out.Live, "live peer"), countOf(out.IDs, "record"), skippedNote(out.Skipped))
	for _, p := range out.Peers {
		if p.NotRead == "" && len(p.Rows) == 0 {
			continue
		}
		fmt.Fprintf(w, "%s\n", termsafe.Sanitize(peerLabel(p)))
		if p.NotRead != "" {
			fmt.Fprintf(w, "  not read: %s\n", termsafe.Sanitize(p.NotRead))
			continue
		}
		for _, r := range p.Rows {
			line := fmt.Sprintf("  %s  %s", r.ID, rowPhrase(r))
			if r.Title != "" {
				line += " — " + r.Title
			}
			fmt.Fprintln(w, termsafe.Sanitize(line))
		}
	}
}

// peerLabel names a peer: its branch (or "detached"), its path, its source.
func peerLabel(p peers.Peer) string {
	name := p.Branch
	if name == "" {
		name = "(detached)"
	}
	if p.Path != "" {
		return name + "  " + p.Path + "  (" + string(p.Source) + ")"
	}
	return name + "  (" + string(p.Source) + ")"
}

func rowPhrase(r peers.Row) string {
	switch r.Kind {
	case peers.KindOpenThere:
		return "open there, absent here"
	case peers.KindTerminalThere:
		return r.Folder + " there, open here"
	default:
		return "drafted there, absent here"
	}
}

func skippedNote(sk []peers.Skipped) string {
	if len(sk) == 0 {
		return ""
	}
	counts := map[string]int{}
	for _, s := range sk {
		counts[s.Reason]++
	}
	var parts []string
	for _, reason := range []string{peers.SkipMerged, peers.SkipGone, peers.SkipEmbargo} {
		if n := counts[reason]; n > 0 {
			parts = append(parts, fmt.Sprintf("%d %s", n, reason))
		}
	}
	return fmt.Sprintf("; %d skipped (%s)", len(sk), strings.Join(parts, ", "))
}

func countOf(n int, noun string) string {
	if n == 1 {
		return "1 " + noun
	}
	return fmt.Sprintf("%d %ss", n, noun)
}

// boardPeersLine is the board's peers member: present only when some live peer
// holds a record that differs here.
type boardPeersLine struct {
	Live int `json:"live"`
	IDs  int `json:"ids"`
}

// boardScan runs the board's one peers scan for the checkout containing cwd
// and composes the two members it feeds: the peers member and the worktrees
// member (spc-2609301811532881), so the store's line costs no second scan.
// Each is nil outside a checkout and when the reader fails (said on stderr);
// the peers member is nil when no peer holds anything that differs here.
func boardScan(cwd string, stderr io.Writer) (*boardPeersLine, *boardWorktreesLine) {
	root, err := gitutil.CheckoutRoot(cwd, peersStore)
	if err != nil {
		return nil, nil
	}
	rep, err := peers.Scan(root)
	if err != nil {
		// The reader's error can name a record folder by its absolute path,
		// so the checkout is named by the board's display rule here too
		// (iss-2609281613094952).
		fmt.Fprintf(stderr, "abcd: the peers line is omitted — %s\n", termsafe.Sanitize(fsutil.DisplayPathsIn(err.Error(), root)))
		return nil, nil
	}
	wt := boardWorktrees(root, rep, stderr)
	if rep.IDCount() == 0 {
		return nil, wt
	}
	return &boardPeersLine{Live: rep.Live(), IDs: rep.IDCount()}, wt
}

// peerHeldError is the not-found refusal a peer answers: the record is not in
// this checkout, and a peer holds it. It unwraps to the local not-found error,
// so a caller testing for that class still finds it.
type peerHeldError struct {
	msg string
	err error
}

func (e *peerHeldError) Error() string { return e.msg }
func (e *peerHeldError) Unwrap() error { return e.err }

// peerHeldRefusal is consulted on a verb's not-found path, and only there, so a
// healthy lookup pays nothing: when this checkout does not hold id and a live
// peer does, it returns the refusal naming each holder's branch, path and
// folder; otherwise it returns err unchanged. The prefix is the verb's own.
func peerHeldRefusal(cwd, prefix, id string, err error) error {
	root, rerr := gitutil.CheckoutRoot(cwd, peersStore)
	if rerr != nil {
		return err
	}
	rep, rerr := peers.Scan(root)
	if rerr != nil || rep.HeldHere(id) {
		return err
	}
	// Every location, a peer's untouched older copy of the record
	// included (peers.Report.Holders leaves those out): the question here is
	// where the record is, not who holds it.
	locs := rep.Locate(id)
	if len(locs) == 0 {
		return err
	}
	// A peer's branch name and path are another checkout's bytes, and git admits
	// a C1 control or a bidi override in either; this refusal reaches stderr and
	// the --json envelope through scrubPaths alone, so it is sanitised here. The
	// folder needs nothing: the scan admits only its own fixed folder names.
	var holders []string
	for _, l := range locs {
		h := "branch " + termsafe.Sanitize(l.Branch)
		if l.Branch == "" {
			h = "a detached worktree"
		}
		if l.Path != "" {
			h += " at " + termsafe.Sanitize(fsutil.DisplayPath(l.Path))
		} else {
			h += " (checked out nowhere)"
		}
		holders = append(holders, h+" holds it in "+l.Folder+"/")
	}
	msg := fmt.Sprintf("%s%s is not in this checkout, and a peer holds it: %s. Work on it where it is, or bring that branch's record here; `abcd peers` shows the whole picture (nothing written)",
		prefix, id, strings.Join(holders, "; "))
	var ee *exitError
	if errors.As(err, &ee) {
		return &exitError{Code: ee.Code, Msg: msg}
	}
	return &peerHeldError{msg: msg, err: err}
}

// addReservations fetches the reservations from origin and adds them, as
// judged now, to out. It returns the failure to name when the fetch failed
// (the view is then the last one fetched) or the view could not be read, and
// "" otherwise. A checkout with no remote named origin lists none.
func addReservations(root string, out *peersOutput, now time.Time) (failed string) {
	o := claim.Options{Now: func() time.Time { return now }}
	gap, err := claim.Fetch(root, o)
	if err == nil {
		// The fallback re-arm (dcn-2610222219297651): a copy of the person
		// whose release has ended re-arms what it withdrew, once, and
		// records it; another person's copy only names it below. The
		// listing then reads the reservations as they now stand.
		if settled, serr := claim.SettleEndedReleases(root, o); serr == nil && len(settled) > 0 {
			for i := range settled {
				settled[i].Failed = termsafe.Sanitize(fsutil.RedactHome(settled[i].Failed))
				for j := range settled[i].Failures {
					settled[i].Failures[j].Reason = termsafe.Sanitize(fsutil.RedactHome(settled[i].Failures[j].Reason))
				}
			}
			out.Settled = settled
			if g, ferr := claim.Fetch(root, o); ferr == nil {
				gap = g
			}
		}
	}
	return out.fill(root, now, gap, err)
}

// fill adds to b the reservations as last fetched, judged now, after a fetch
// that answered gap and err. It returns the failure to name, as
// addReservations does.
func (b *reservationsBlock) fill(root string, now time.Time, gap string, err error) (failed string) {
	failed, ok := b.fillView(root, now, gap, err)
	if !ok {
		return failed
	}
	// The holds are judged by the merge hold's own rule, so the listing counts
	// a release exactly as the land fence and `abcd land` do.
	holds, herr := claim.JudgedHolds(root, now, claim.Options{Now: func() time.Time { return now }})
	return b.fillHolds(failed, holds, herr)
}

// fillView is fill's reservations alone: ok is false when there are none to
// list (no origin) or they could not be read, and failed then says why.
func (b *reservationsBlock) fillView(root string, now time.Time, gap string, err error) (failed string, ok bool) {
	out := b
	out.Reservations, out.LiveByHolder, out.ReleaseHolds = []reservationRow{}, map[string]int{}, []holdRow{}
	out.LapsedReleases = []lapsedRow{}
	out.HandBacksByHolder = []handBackHolder{}
	if errors.Is(err, claim.ErrNoOrigin) {
		return "", false
	}
	out.View.Origin = true
	reader := claim.ReaderSelf(root, claim.Options{Now: func() time.Time { return now }})
	out.HolderIDsRefused, out.HolderIDRefused = reader.HolderIDsRefused, reader.HolderIDRefused
	if err != nil {
		out.FetchError = termsafe.Sanitize(fsutil.RedactHome(err.Error()))
		failed = out.FetchError
	}
	out.DerivedUnchecked = termsafe.Sanitize(fsutil.RedactHome(gap))
	snap, verr := claim.View(root, now)
	if verr != nil {
		return joinFailed(failed, termsafe.Sanitize(fsutil.RedactHome(verr.Error()))), false
	}
	if !snap.FetchedAt.IsZero() {
		out.View.FetchedAt = rfc3339(snap.FetchedAt)
		age := int64(now.Sub(snap.FetchedAt).Seconds())
		out.View.AgeSeconds = &age
	}
	// The ceiling names the reader's sibling worktrees in the view, so it is
	// read before the rows that show them.
	if c, ok := claim.CeilingFor(root, &snap, claim.Options{Now: func() time.Time { return now }}); ok {
		c.Holder = termsafe.Sanitize(c.Holder)
		for i := range c.Siblings {
			c.Siblings[i].Name = termsafe.Sanitize(c.Siblings[i].Name)
		}
		c.Note = termsafe.Sanitize(fsutil.RedactHome(c.Note))
		out.Ceiling = &ceilingRow{Ceiling: c, Warning: c.Warning()}
	}
	for _, r := range snap.Reservations {
		row := reservationRow{Record: r.Record, Holder: r.Holder, Copy: r.Copy, Footprint: make([]string, 0, len(r.Footprint)),
			EffectiveExpiry: rfc3339(r.EffectiveExpiry), State: r.State, ReleasedBy: r.ReleasedBy,
			Unreadable: r.Unreadable, HeldFromFetch: r.HeldFromFetch, Commit: r.Commit, Sibling: termsafe.Sanitize(r.Sibling)}
		if !r.ClaimedAt.IsZero() {
			row.ClaimedAt = rfc3339(r.ClaimedAt)
		}
		for _, f := range r.Footprint {
			row.Footprint = append(row.Footprint, termsafe.Sanitize(f))
		}
		if r.State == claim.StateLive && claim.ValidLaneStage(r.Stage) {
			// The stage is where a live lane stands, so a lapsed one shows
			// none, as in the text; and it is from a closed vocabulary the
			// reader has checked, so only a value inside it is ever shown.
			row.Stage = r.Stage
		}
		out.Reservations = append(out.Reservations, row)
	}
	out.LiveByHolder = snap.LiveByHolder()
	out.ReleaseWithdrawals = snap.ReleaseWithdrawals(root, reader, now)
	out.LostWithdrawals = claim.LostReleaseWithdrawals(root, now)
	for _, l := range snap.LapsedFor(now, reader) {
		out.LapsedReleases = append(out.LapsedReleases, lapsedRow{Release: l.Release, Subject: l.Subject,
			Holder: termsafe.Sanitize(l.Holder), Copy: l.Copy, RenewedAt: rfc3339(l.RenewedAt), LapsedAt: rfc3339(l.LapsedAt),
			Text: termsafe.Sanitize(l.Text())})
	}
	// The hand-backs are read from the same view, as last fetched: no act of
	// their own touches the network.
	for _, h := range snap.RecentHandBacks(now) {
		hh := handBackHolder{Holder: termsafe.Sanitize(h.Holder), HandBacks: make([]handBackRow, 0, len(h.HandBacks)), More: h.More}
		for _, hb := range h.HandBacks {
			if !slices.Contains(claim.HandBackKinds, hb.Kind) {
				// The reader admits no other kind; one is never shown.
				continue
			}
			hh.HandBacks = append(hh.HandBacks, handBackRow{Record: termsafe.Sanitize(hb.Record), Kind: hb.Kind, ReleasedAt: rfc3339(hb.ReleasedAt)})
		}
		out.HandBacksByHolder = append(out.HandBacksByHolder, hh)
	}
	return failed, true
}

// fillHolds adds the release holds, read once by whoever calls it (the
// listing's own read, or `abcd status`'s ReadStatus), to the block; herr is
// why they could not be read, joined to failed.
func (b *reservationsBlock) fillHolds(failed string, holds []claim.Hold, herr error) string {
	if herr != nil {
		return joinFailed(failed, termsafe.Sanitize(fsutil.RedactHome(herr.Error())))
	}
	for _, h := range holds {
		b.ReleaseHolds = append(b.ReleaseHolds, holdRow{Release: h.Release, Subject: h.Subject, Holder: h.Holder,
			Copy: h.Copy, EffectiveExpiry: rfc3339(h.EffectiveExpiry), Unreadable: h.Unreadable, Own: h.Own,
			Unsighted: h.Unsighted, Text: termsafe.Sanitize(h.Line())})
	}
	return failed
}

// joinFailed is failed with msg after it, "; " between them.
func joinFailed(failed, msg string) string {
	if failed != "" {
		return failed + "; " + msg
	}
	return msg
}

// renderReservations is the reservations block of the text form: a heading
// that dates the view, one line per reservation, and the live counts.
func renderReservations(w io.Writer, out reservationsBlock) {
	if !out.View.Origin {
		fmt.Fprintln(w, "reservations — this checkout has no remote named origin")
		return
	}
	switch {
	case out.View.AgeSeconds == nil:
		fmt.Fprintln(w, "reservations — nothing has been fetched here; the age of this view is unknown")
	case out.FetchError != "":
		fmt.Fprintf(w, "reservations — as of the fetch %s ago (%s); this fetch failed\n", ageText(*out.View.AgeSeconds), out.View.FetchedAt)
	default:
		fmt.Fprintf(w, "reservations — as of the fetch %s ago (%s)\n", ageText(*out.View.AgeSeconds), out.View.FetchedAt)
	}
	if len(out.Reservations) == 0 {
		fmt.Fprintln(w, "  none")
	}
	for _, r := range out.Reservations {
		var line string
		switch r.State {
		case claim.StateUnreadable:
			line = fmt.Sprintf("  %s  unreadable (%s)  held until %s", r.Record, r.Unreadable, r.EffectiveExpiry)
			if r.HeldFromFetch {
				line += ", counted from when this copy first fetched it"
			}
		case claim.StateReleased:
			line = fmt.Sprintf("  %s  released (%s)  %s", r.Record, strings.ReplaceAll(r.ReleasedBy, "-", " "), r.Holder)
		default:
			line = fmt.Sprintf("  %s  %s  %s  until %s", r.Record, r.State, r.Holder, r.EffectiveExpiry)
			if r.State == claim.StateLive && r.Stage != "" {
				line += "  stage " + r.Stage
			}
		}
		switch {
		case r.Copy == claim.CopyThis:
			line += "  (this copy)"
		case r.Sibling != "":
			line += fmt.Sprintf("  (from the worktree %q on this machine)", r.Sibling)
		}
		if len(r.Footprint) > 0 {
			quoted := make([]string, len(r.Footprint))
			for i, f := range r.Footprint {
				quoted[i] = fmt.Sprintf("%q", f)
			}
			line += "  footprint " + strings.Join(quoted, ", ")
		}
		fmt.Fprintln(w, line)
	}
	for _, h := range out.ReleaseHolds {
		fmt.Fprintln(w, "  "+h.Text)
	}
	for _, st := range out.Settled {
		for _, wd := range st.Settled {
			// A skip is named below, from the tombstone that records it.
			if wd.State == claim.WithdrawnRearmed || wd.State == claim.WithdrawnFoundArmed {
				fmt.Fprintln(w, "  "+claim.SettledLine(st.Subject, wd))
			}
		}
		for _, f := range st.Failures {
			fmt.Fprintln(w, "  "+f.Line())
		}
		if st.Failed != "" {
			fmt.Fprintln(w, "  "+st.Failed)
		}
		for _, line := range st.Notes() {
			fmt.Fprintln(w, "  "+line)
		}
	}
	for _, rw := range out.ReleaseWithdrawals {
		for _, line := range rw.Lines() {
			fmt.Fprintln(w, "  "+line)
		}
	}
	for _, l := range out.LostWithdrawals {
		fmt.Fprintln(w, "  "+l.Line())
	}
	for _, l := range out.LapsedReleases {
		fmt.Fprintln(w, "  "+l.Text)
	}
	if len(out.LiveByHolder) > 0 {
		names := make([]string, 0, len(out.LiveByHolder))
		for n := range out.LiveByHolder {
			names = append(names, n)
		}
		sort.Strings(names)
		parts := make([]string, len(names))
		for i, n := range names {
			parts[i] = fmt.Sprintf("%s %d", n, out.LiveByHolder[n])
		}
		fmt.Fprintf(w, "  live: %s\n", strings.Join(parts, ", "))
	}
	if c := out.Ceiling; c != nil {
		parts := []string{fmt.Sprintf("this copy %d", c.ThisCopy)}
		for _, s := range c.Siblings {
			parts = append(parts, fmt.Sprintf("%q %d", s.Name, s.Live))
		}
		parts = append(parts, fmt.Sprintf("elsewhere %d", c.Elsewhere))
		fmt.Fprintf(w, "  your holds: %s %d of %d (%s)\n", c.Holder, c.Live, c.Limit, strings.Join(parts, ", "))
		if c.Warning != "" {
			fmt.Fprintf(w, "  warning: %s\n", c.Warning)
		}
		// Why the ceiling is the default, or why siblings count as
		// elsewhere: --json carries it as the ceiling's note, and the text
		// says it too, so a default is never read as a limit someone set.
		if c.Note != "" {
			fmt.Fprintf(w, "  note: %s\n", c.Note)
		}
	}
	for _, h := range out.HandBacksByHolder {
		fmt.Fprintf(w, "  handed back by %s in the last day:\n", h.Holder)
		for _, hb := range h.HandBacks {
			fmt.Fprintf(w, "    %s  %s  at %s\n", hb.Record, hb.Kind, hb.ReleasedAt)
		}
		if h.More > 0 {
			fmt.Fprintf(w, "    and %d more\n", h.More)
		}
	}
	if out.DerivedUnchecked != "" {
		fmt.Fprintf(w, "  not checked: which records are finished on origin's default branch (%s)\n", out.DerivedUnchecked)
	}
}
