package claim

import (
	"fmt"
	"sort"
	"strconv"

	"github.com/intentdriven/abcd/internal/termsafe"
)

// The cap counts a person's live reservations across every session, copy and
// account that pushes as their GitHub login, while each machine reads its own
// reservation.max_live (maxlive.go). A machine's sessions can therefore hold
// another account on the same login at a ceiling nobody here can read
// (iss-2610231004140527). The ceiling view says how near the reader stands to
// this machine's, and from which copies the holds come.

// SiblingLive is how many live reservations one sibling worktree holds.
type SiblingLive struct {
	Name string `json:"name"`
	Live int    `json:"live"`
}

// Ceiling is the reader's live reservations against this machine's cap,
// counted by the copy each was made from.
type Ceiling struct {
	Holder string `json:"holder"`
	Live   int    `json:"live"`
	Limit  int    `json:"limit"`
	// ThisCopy, Siblings and Elsewhere split Live: this checkout's copy, each
	// other worktree of the repository on this machine (by name, sorted), and
	// every copy not found among them: another clone on this machine or
	// another, a removed worktree, or another account pushing as the login.
	ThisCopy  int           `json:"this_copy"`
	Siblings  []SiblingLive `json:"siblings"`
	Elsewhere int           `json:"elsewhere"`
	// Warn is true from four fifths of the limit.
	Warn bool `json:"warn"`
	// Note says why the machine's configuration was read past for the
	// limit, or why the worktrees could not be listed; "" when neither.
	Note string `json:"note,omitempty"`
}

// CeilingFor is the reader's ceiling in the view s, naming its siblings
// (NameSiblings) first: false when the reader's GitHub login cannot be named
// without the network, or this machine's reservation.max_live is one an act
// would refuse.
func CeilingFor(root string, s *Snapshot, o Options) (Ceiling, bool) {
	holder := readerHolder(root, o)
	if holder == "" {
		return Ceiling{}, false
	}
	limit, note, err := maxLive(o.home())
	if err != nil {
		return Ceiling{}, false
	}
	s.NameSiblings(root, holder)
	note = joinNote(note, s.SiblingsUnread)
	c := Ceiling{Holder: holder, Limit: limit, Siblings: []SiblingLive{}, Note: note}
	bySibling := map[string]int{}
	for _, r := range s.Reservations {
		if r.State != StateLive || isRelease(r.Record) || !sameHolder(r.Holder, holder) {
			continue
		}
		c.Live++
		switch {
		case r.Copy == CopyThis:
			c.ThisCopy++
		case r.Sibling != "":
			bySibling[r.Sibling]++
		default:
			c.Elsewhere++
		}
	}
	for name, n := range bySibling {
		c.Siblings = append(c.Siblings, SiblingLive{Name: name, Live: n})
	}
	sort.Slice(c.Siblings, func(i, j int) bool { return c.Siblings[i].Name < c.Siblings[j].Name })
	c.Warn = c.Live*5 >= c.Limit*4
	return c, true
}

// Warning is the ceiling's warning, "" below four fifths of the limit.
func (c Ceiling) Warning() string {
	if !c.Warn {
		return ""
	}
	where := "near"
	if c.Live >= c.Limit {
		where = "at"
	}
	return fmt.Sprintf("%s holds %d of the %d live reservations this machine's %s allows, %s the cap; "+
		"every session and account on the same login counts toward the same total, "+
		"so another account's own ceiling, which is not readable here, may already be reached",
		c.Holder, c.Live, c.Limit, MaxLiveKey, where)
}

// joinNote is a and b joined by "; ", either alone when the other is "".
func joinNote(a, b string) string {
	switch {
	case a == "":
		return b
	case b == "":
		return a
	}
	return a + "; " + b
}

// copyPlace names the copy a reservation was made from, as the cap's refusal
// names it: this copy, a sibling worktree here (its name quoted and
// terminal-safe, as abcd peers shows it), or elsewhere.
func copyPlace(r Reservation) string {
	switch {
	case r.Copy == CopyThis:
		return "this copy"
	case r.Sibling != "":
		return strconv.Quote(termsafe.Sanitize(r.Sibling)) + ", on this machine"
	}
	return "elsewhere"
}
