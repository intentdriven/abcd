package cli

// implement_heartbeat.go is the front door of `abcd implement heartbeat`
// (iss-2611220942554289): an autonomous run's periodic check in one command,
// where a run used to compose it by hand from a handful of shell commands each
// time. It renews the reservations the run's open pull requests stand on
// (claim.RenewHeld, answered by prLive), reports each
// tracked pull request (landing.PRStates) and the free disk against the load
// gate's floor (implement.ReadDisk), and says what changed since the last
// heartbeat, which it keeps in the checkout's local tier
// (autonomy.WriteHeartbeat).

import (
	"fmt"
	"io"
	"os"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/intentdriven/abcd/internal/abcdhome"
	"github.com/intentdriven/abcd/internal/core/autonomy"
	"github.com/intentdriven/abcd/internal/core/claim"
	"github.com/intentdriven/abcd/internal/core/implement"
	"github.com/intentdriven/abcd/internal/core/landing"
	"github.com/intentdriven/abcd/internal/fsutil"
	"github.com/intentdriven/abcd/internal/gitutil"
	"github.com/intentdriven/abcd/internal/termsafe"
)

// heartbeatSeams are the seams the surface tests replace; nil is the real
// one: the forge client (landing.GH), the reservations' fetch (claim.Fetch)
// and the disk read (implement.ReadDisk at implement.DiskFloor).
var heartbeatSeams struct {
	Forge func(dir string, args ...string) (string, error)
	Fetch func(root string, o claim.Options) (string, error)
	Disk  func(home string) implement.DiskCheck
}

// heartbeatOutput is the heartbeat's --json shape.
type heartbeatOutput struct {
	At         string                `json:"at"`
	PreviousAt string                `json:"previous_at,omitempty"`
	Changed    bool                  `json:"changed"`
	Summary    string                `json:"summary"`
	Changes    []autonomy.BeatChange `json:"changes"`
	Items      []autonomy.BeatItem   `json:"items"`
	Notices    []string              `json:"notices,omitempty"`
}

// newImplementHeartbeatCommand builds `abcd implement heartbeat [<branch>|<pr>...]`.
func newImplementHeartbeatCommand(asJSON *bool) *cobra.Command {
	return &cobra.Command{
		Use: "heartbeat [<branch>|<pr>...]",
		Long: "An autonomous run's periodic check, in one command:\n\n" +
			"  1. Reservations: fetches the reservations from origin and renews each of your own\n" +
			"     reservations of an issue or an intent held from any worktree of this checkout\n" +
			"     that is live and half through its lease, with its own term, as abcd reserve\n" +
			"     --renew renews it, and only one whose latest commit the worktree holding it\n" +
			"     pushed and whose worktree's branch has a pull request open, armed or queued.\n" +
			"     Any other (abandoned, or a lane mid-work with no pull request yet) is reported\n" +
			"     held and not renewed, with why, and lapses on its lease; a lane's own loop renews\n" +
			"     its reservations. A lapsed one is reported, never revived; one held from another\n" +
			"     clone or computer, another person's and a release being cut are left alone.\n" +
			"  2. Pull requests: reports each tracked pull request as abcd land --state does:\n" +
			"     merged, queued with its position, armed, open and not armed, removed from the\n" +
			"     merge queue with the reason, or closed. The tracked ones are your open pull\n" +
			"     requests from this repository's branches, and each the last heartbeat read that\n" +
			"     was not yet merged or closed, so a merge is reported once; or the branches and\n" +
			"     numbers (N or #N) named, alone.\n" +
			"  3. Disk: reports the free space against the load gate's disk floor, read as\n" +
			"     abcd implement load reads it.\n\n" +
			"It keeps what it read in .abcd/.work.local/heartbeat.json and prints one line when\n" +
			"nothing changed since the last heartbeat, and otherwise one line for each change: a\n" +
			"fact that is new, in another state, or gone. A renewal that only moved a lease on\n" +
			"is no change. --json carries the same: changed, summary, changes (each with was or\n" +
			"gone where it applies) and every fact read (items).\n\n" +
			"It writes only the reservation renewals and its own state file; it trims nothing.\n" +
			"A read it cannot make (the fetch, the listing, a pull request, the disk) is reported\n" +
			"as a fact, never fatal. Exit 2 on an operand that is neither a branch name nor a\n" +
			"number, or outside a checkout.",
		RunE: func(cmd *cobra.Command, args []string) error {
			const prefix = "abcd implement heartbeat"
			cwd, err := os.Getwd()
			if err != nil {
				return err
			}
			root, err := gitutil.CheckoutRoot(cwd, "the heartbeat")
			if err != nil {
				return &exitError{Code: 2, Msg: prefix + ": " + termsafe.Sanitize(err.Error())}
			}
			var asks []landing.PRAsk
			for _, a := range args {
				ask, err := landing.ParsePRAsk(a)
				if err != nil {
					return &exitError{Code: 2, Msg: prefix + ": " + err.Error() + "; nothing was renewed"}
				}
				asks = append(asks, ask)
			}
			out := heartbeat(root, asks)
			for _, n := range out.Notices {
				fmt.Fprintf(cmd.ErrOrStderr(), "%s: %s\n", prefix, n)
			}
			return render(cmd.OutOrStdout(), *asJSON, out, func(w io.Writer) {
				if !out.Changed {
					fmt.Fprintf(w, "%s: %s\n", prefix, out.Summary)
					return
				}
				for _, c := range out.Changes {
					l := c.Line
					if c.Was != "" {
						l += " (was " + c.Was + ")"
					}
					fmt.Fprintf(w, "%s: %s\n", prefix, l)
				}
			})
		},
	}
}

// heartbeat is one heartbeat in the checkout at root, asks the pull requests
// named (none: the tracked ones).
func heartbeat(root string, asks []landing.PRAsk) heartbeatOutput {
	now := time.Now().UTC().Truncate(time.Second)
	out := heartbeatOutput{At: now.Format(time.RFC3339), Changes: []autonomy.BeatChange{}}
	prev, prevAt, err := autonomy.ReadHeartbeat(root)
	if err != nil {
		out.Notices = append(out.Notices, "the last heartbeat's facts could not be read ("+beatText(err.Error())+"), so every fact counts as new")
	}
	if !prevAt.IsZero() {
		out.PreviousAt = prevAt.UTC().Format(time.RFC3339)
	}
	forge := heartbeatSeams.Forge
	if forge == nil {
		forge = landing.GH
	}
	prItems, states := heartbeatPRs(root, forge, asks, prev)
	var items []autonomy.BeatItem
	items = append(items, heartbeatReservations(root, prLive(root, forge, states))...)
	items = append(items, prItems...)
	items = append(items, heartbeatDisk())
	out.Items = items
	out.Changes = append(out.Changes, autonomy.HeartbeatChanges(prev, items)...)
	out.Changed = len(out.Changes) > 0
	out.Summary = heartbeatSummary(items, out.PreviousAt)
	if err := autonomy.WriteHeartbeat(root, now, items); err != nil {
		out.Notices = append(out.Notices, "this heartbeat's facts were not kept ("+beatText(err.Error())+"), so the next one counts every fact as new")
	}
	return out
}

// heartbeatReservations fetches the reservations and renews the person's own
// held from this checkout that live answers for, one fact each, and one forge
// fact for a fetch or a renewal pass that failed.
func heartbeatReservations(root string, live claim.LiveWork) []autonomy.BeatItem {
	var items []autonomy.BeatItem
	fetch := heartbeatSeams.Fetch
	if fetch == nil {
		fetch = claim.Fetch
	}
	if _, err := fetch(root, claim.Options{}); err != nil {
		items = append(items, autonomy.BeatItem{Kind: autonomy.BeatForge, Key: "reservations", State: "unfetched",
			Line: "reservations: not fetched from origin (" + beatText(err.Error()) + "); judged as last fetched"})
	}
	held, err := claim.RenewHeld(root, live, claim.Options{})
	if err != nil {
		return append(items, autonomy.BeatItem{Kind: autonomy.BeatForge, Key: "renewal", State: "failed",
			Line: "reservations: not renewed (" + beatText(err.Error()) + ")"})
	}
	for _, h := range held {
		from := ""
		if h.HeldFrom != "" {
			from = " from " + beatText(h.HeldFrom)
		}
		until := h.Until.UTC().Format(time.RFC3339)
		it := autonomy.BeatItem{Kind: autonomy.BeatReservation, Key: h.Record}
		switch {
		case h.Err != nil:
			it.State, it.Line = "unrenewed", h.Record+": held"+from+" until "+until+", and not renewed: "+beatText(h.Err.Error())
		case h.State == claim.StateExpired:
			it.State, it.Line = "lapsed", h.Record+": lapsed at "+until+from+"; reserve it again to keep it"
		case h.Unbacked != "":
			it.State, it.Line = "unbacked", h.Record+": held"+from+" until "+until+", and not renewed: "+beatText(h.Unbacked)+"; it lapses on its lease"
		case h.Renewed:
			it.State, it.Line = "held", h.Record+": renewed"+from+" until "+until
		default:
			it.State, it.Line = "held", h.Record+": held"+from+" until "+until
		}
		items = append(items, it)
	}
	return items
}

// heartbeatPRs reports each pull request asked, or with none asked the
// account's open ones and each prev read not yet merged or closed, with the
// states it read.
func heartbeatPRs(root string, forge func(dir string, args ...string) (string, error), asks []landing.PRAsk, prev []autonomy.BeatItem) ([]autonomy.BeatItem, []landing.PRState) {
	var items []autonomy.BeatItem
	if len(asks) == 0 {
		own, err := landing.OwnOpenPRs(root, forge)
		if err != nil {
			items = append(items, autonomy.BeatItem{Kind: autonomy.BeatForge, Key: "pull_requests", State: "unlisted",
				Line: "pull requests: your open pull requests could not be listed (" + beatText(err.Error()) + ")"})
		}
		asks = own
		for _, n := range autonomy.TrackedPRs(prev) {
			if !slices.ContainsFunc(asks, func(a landing.PRAsk) bool { return a.Number == n }) {
				asks = append(asks, landing.PRAsk{Number: n})
			}
		}
		slices.SortFunc(asks, func(a, b landing.PRAsk) int { return a.Number - b.Number })
	}
	states := landing.PRStates(root, forge, asks)
	for _, s := range states {
		items = append(items, autonomy.BeatItem{Kind: autonomy.BeatPullRequest, Key: s.Key(), State: prBeatState(s),
			Line: s.Line(), Terminal: s.Terminal()})
	}
	return items, states
}

// prLive is the heartbeat's claim.LiveWork: a branch has live work behind it
// when its newest pull request from this repository is open (one the merge
// queue removed included), armed or queued. A branch among the states the
// heartbeat already read is answered from them; any other is asked of the
// forge once.
func prLive(root string, forge func(dir string, args ...string) (string, error), read []landing.PRState) claim.LiveWork {
	known := map[string]landing.PRState{}
	for _, s := range read {
		if s.Branch != "" && s.State != landing.PRUnread {
			if k, ok := known[s.Branch]; !ok || s.Number > k.Number {
				known[s.Branch] = s
			}
		}
	}
	return func(branch string) (bool, string) {
		s, ok := known[branch]
		if !ok {
			s = landing.PRStates(root, forge, []landing.PRAsk{{Branch: branch}})[0]
			known[branch] = s
		}
		switch s.State {
		case landing.PROpen, landing.PRArmed, landing.PRQueued, landing.PRRemoved:
			return true, ""
		case landing.PRNone:
			return false, "no pull request from its branch " + branch
		case landing.PRMerged:
			return false, "its branch " + branch + "'s pull request " + s.Key() + " is merged"
		case landing.PRClosed:
			return false, "its branch " + branch + "'s pull request " + s.Key() + " is closed"
		}
		return false, "its branch " + branch + "'s pull request could not be read (" + s.Unread + ")"
	}
}

// prBeatState is the state a pull request is compared on: its state, with a
// queued one's entry state and position and a removal's time, so a move in
// the queue or a second removal is a change.
func prBeatState(s landing.PRState) string {
	switch s.State {
	case landing.PRQueued:
		return s.State + ":" + s.QueueState + ":" + strconv.Itoa(s.Position)
	case landing.PRRemoved:
		return s.State + ":" + s.RemovedAt
	}
	return s.State
}

// heartbeatDisk is the disk's fact against the load gate's floor.
func heartbeatDisk() autonomy.BeatItem {
	home, _ := abcdhome.MachineHome()
	read := heartbeatSeams.Disk
	if read == nil {
		read = func(home string) implement.DiskCheck { return implement.ReadDisk(home, implement.DiskFloor(home)) }
	}
	d := read(home)
	return autonomy.BeatItem{Kind: autonomy.BeatDisk, Key: "disk", State: d.Status, Line: "disk: " + beatText(d.Line())}
}

// heartbeatSummary is the one line a heartbeat with no change prints.
func heartbeatSummary(items []autonomy.BeatItem, prevAt string) string {
	var res, renewed, prs int
	disk := "unread"
	for _, it := range items {
		switch it.Kind {
		case autonomy.BeatReservation:
			if it.State == "held" || it.State == "unbacked" {
				res++
			}
			if strings.Contains(it.Line, ": renewed") {
				renewed++
			}
		case autonomy.BeatPullRequest:
			prs++
		case autonomy.BeatDisk:
			disk = it.State
		}
	}
	since := "the last heartbeat"
	if prevAt != "" {
		since = prevAt
	}
	return fmt.Sprintf("nothing changed since %s: %d reservations held (%d renewed now), %d pull requests tracked, disk %s",
		since, res, renewed, prs, disk)
}

// beatText is a clause made safe to print: home-redacted and sanitised.
func beatText(s string) string { return termsafe.Sanitize(fsutil.RedactHome(s)) }
