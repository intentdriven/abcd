package cli

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

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
}

// newPeersCommand builds the `peers` verb — the front door onto
// internal/core/peers (itd-2609091416295622, spc-2609202056480020): what the
// sibling worktrees and the local branches of this checkout hold that this tree
// does not, before a session captures, fixes or files anything.
func newPeersCommand(asJSON *bool) *cobra.Command {
	return &cobra.Command{
		Use:   "peers",
		Short: "List the records this checkout's sibling worktrees and local branches hold that it does not (read-only)",
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
			"two status folders is named with the reason and not read.\n\n" +
			"Strictly read-only: it writes nothing, takes no lock, and fetches nothing.\n" +
			"Home paths are redacted to ~ on every stream. Exit 0 whatever the peers\n" +
			"hold; exit 2 outside a git checkout.",
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
			return render(cmd.OutOrStdout(), *asJSON, out, func(w io.Writer) { renderPeers(w, out) })
		},
	}
}

// peersView is the report as a surface shows it: paths and reasons redacted.
func peersView(rep peers.Report) peersOutput {
	out := peersOutput{Sources: rep.Sources, DefaultRef: rep.DefaultRef, Live: rep.Live(), IDs: rep.IDCount(),
		Peers: make([]peers.Peer, 0, len(rep.Peers)), Skipped: make([]peers.Skipped, 0, len(rep.Skipped))}
	for _, p := range rep.Peers {
		p.Path = redactHomePath(p.Path)
		p.NotRead = redactHomePath(p.NotRead)
		out.Peers = append(out.Peers, p)
	}
	for _, s := range rep.Skipped {
		s.Path = redactHomePath(s.Path)
		out.Skipped = append(out.Skipped, s)
	}
	return out
}

// renderPeers is the text form: one summary line, then a block only for a peer
// with something to say, so a session reading it before every pick reads a
// line or two, not one per worktree.
func renderPeers(w io.Writer, out peersOutput) {
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

// redactHomePath replaces the home directory with ~ in either spelling: the one
// the environment names and the one git reports, which differ wherever the home
// sits behind a symlink. The resolved spelling goes first: the environment's
// spelling can be a suffix of it (/var/… inside /private/var/…), and redacting
// that first would leave the resolved prefix stranded in front of the ~, since
// fsutil.RedactRoot checks no boundary before a match (iss-2609230641546141).
func redactHomePath(s string) string {
	if home, err := os.UserHomeDir(); err == nil {
		if real, err := filepath.EvalSymlinks(home); err == nil && real != filepath.Clean(home) {
			s = fsutil.RedactRoot(s, real, "~")
		}
	}
	return fsutil.RedactHome(s)
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
	rep, err := peers.Read(root)
	if err != nil {
		fmt.Fprintf(stderr, "abcd: the peers line is omitted — %s\n", termsafe.Sanitize(redactHomePath(err.Error())))
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
	rep, rerr := peers.Read(root)
	if rerr != nil || rep.HeldHere(id) {
		return err
	}
	locs := rep.Locate(id)
	if len(locs) == 0 {
		return err
	}
	var holders []string
	for _, l := range locs {
		h := "branch " + l.Branch
		if l.Branch == "" {
			h = "a detached worktree"
		}
		if l.Path != "" {
			h += " at " + redactHomePath(l.Path)
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
