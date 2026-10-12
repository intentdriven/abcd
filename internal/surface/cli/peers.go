package cli

import (
	"errors"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
	"time"

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

	// The reservations on the shared remote (itd-2609091034175565): every one
	// as fetched just now, or as last fetched when the fetch failed.
	Reservations     []reservationRow `json:"reservations"`
	LiveByHolder     map[string]int   `json:"live_by_holder"`
	View             reservationView  `json:"view"`
	FetchError       string           `json:"fetch_error,omitempty"`
	DerivedUnchecked string           `json:"derived_unchecked,omitempty"`
}

// reservationRow is one reservation as the listing shows it. Every value a
// writer chose is terminal-safe here and quoted in the text form.
type reservationRow struct {
	Record          string   `json:"record"`
	Holder          string   `json:"holder,omitempty"`
	Copy            string   `json:"copy,omitempty"`
	Footprint       []string `json:"footprint"`
	ClaimedAt       string   `json:"claimed_at,omitempty"`
	EffectiveExpiry string   `json:"effective_expiry"`
	State           string   `json:"state"`
	ReleasedBy      string   `json:"released_by,omitempty"`
	Unreadable      string   `json:"unreadable,omitempty"`
	HeldFromFetch   bool     `json:"held_from_fetch,omitempty"`
	Commit          string   `json:"commit"`
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
			"hold), and each person's count of live reservations. The fetch is the only\n" +
			"network act; it updates the remote-tracking reservation branches and stamps\n" +
			"when it ran in the local tier, and nothing else is written. A checkout with no\n" +
			"remote named origin lists no reservations. When the fetch fails, the\n" +
			"reservations are listed as last fetched, with that view's age, and the failure\n" +
			"is named on stderr and in fetch_error.\n\n" +
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
			out := peersView(rep)
			failed := addReservations(root, &out, time.Now())
			if err := render(cmd.OutOrStdout(), *asJSON, out, func(w io.Writer) { renderPeers(w, out) }); err != nil {
				return err
			}
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
// directory name rather than an absolute local path (iss-2609281329007423).
func peersView(rep peers.Report) peersOutput {
	out := peersOutput{Sources: rep.Sources, DefaultRef: rep.DefaultRef, Live: rep.Live(), IDs: rep.IDCount(),
		Peers: make([]peers.Peer, 0, len(rep.Peers)), Skipped: make([]peers.Skipped, 0, len(rep.Skipped))}
	for _, p := range rep.Peers {
		p.NotRead = fsutil.DisplayPathsIn(p.NotRead, p.Path)
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
	renderReservations(w, out)
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
	for _, reason := range []string{peers.SkipMerged, peers.SkipGone} {
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

// boardPeers composes the board's peers member for the checkout containing
// cwd, or nil: outside a checkout, when the reader fails (said on stderr), and
// when no peer holds anything that differs here.
func boardPeers(cwd string, stderr io.Writer) *boardPeersLine {
	root, err := gitutil.CheckoutRoot(cwd, peersStore)
	if err != nil {
		return nil
	}
	rep, err := peers.Scan(root)
	if err != nil {
		// The reader's error can name a record folder by its absolute path,
		// so the checkout is named by the board's display rule here too
		// (iss-2609281613094952).
		fmt.Fprintf(stderr, "abcd: the peers line is omitted — %s\n", termsafe.Sanitize(fsutil.DisplayPathsIn(err.Error(), root)))
		return nil
	}
	if rep.IDCount() == 0 {
		return nil
	}
	return &boardPeersLine{Live: rep.Live(), IDs: rep.IDCount()}
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
	out.Reservations, out.LiveByHolder = []reservationRow{}, map[string]int{}
	gap, err := claim.Fetch(root, claim.Options{Now: func() time.Time { return now }})
	if errors.Is(err, claim.ErrNoOrigin) {
		return ""
	}
	out.View.Origin = true
	if err != nil {
		out.FetchError = termsafe.Sanitize(fsutil.RedactHome(err.Error()))
		failed = out.FetchError
	}
	out.DerivedUnchecked = termsafe.Sanitize(fsutil.RedactHome(gap))
	snap, verr := claim.View(root, now)
	if verr != nil {
		msg := termsafe.Sanitize(fsutil.RedactHome(verr.Error()))
		if failed != "" {
			return failed + "; " + msg
		}
		return msg
	}
	if !snap.FetchedAt.IsZero() {
		out.View.FetchedAt = rfc3339(snap.FetchedAt)
		age := int64(now.Sub(snap.FetchedAt).Seconds())
		out.View.AgeSeconds = &age
	}
	for _, r := range snap.Reservations {
		row := reservationRow{Record: r.Record, Holder: r.Holder, Copy: r.Copy, Footprint: make([]string, 0, len(r.Footprint)),
			EffectiveExpiry: rfc3339(r.EffectiveExpiry), State: r.State, ReleasedBy: r.ReleasedBy,
			Unreadable: r.Unreadable, HeldFromFetch: r.HeldFromFetch, Commit: r.Commit}
		if !r.ClaimedAt.IsZero() {
			row.ClaimedAt = rfc3339(r.ClaimedAt)
		}
		for _, f := range r.Footprint {
			row.Footprint = append(row.Footprint, termsafe.Sanitize(f))
		}
		out.Reservations = append(out.Reservations, row)
	}
	out.LiveByHolder = snap.LiveByHolder()
	return failed
}

// renderReservations is the reservations block of the text form: a heading
// that dates the view, one line per reservation, and the live counts.
func renderReservations(w io.Writer, out peersOutput) {
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
		}
		if r.Copy == claim.CopyThis {
			line += "  (this copy)"
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
	if out.DerivedUnchecked != "" {
		fmt.Fprintf(w, "  not checked: which records are finished on origin's default branch (%s)\n", out.DerivedUnchecked)
	}
}
