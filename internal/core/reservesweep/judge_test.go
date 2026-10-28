package reservesweep

import (
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/intentdriven/abcd/internal/core/claim"
)

// The spent-or-kept table (iss-2610142200389875): each predicate on both sides
// of its boundary, the five-minute take-over grace, the hand-back rules of the
// drain work (dcn-2610161806433529, dcn-2610161826443524) and the fail-closed
// answer for a kind the sweep cannot classify.

var t0 = time.Date(2026, 10, 1, 12, 0, 0, 0, time.UTC)

const week = 7 * 24 * time.Hour

// copyKey is a working copy's key in the claim package's form.
const copyKey = "c0ffee00c0ffee00c0ffee00c0ffee00"

// emptyTreeSHA1 is git's empty tree under SHA-1.
const emptyTreeSHA1 = "4b825dc642cb6eb9a060e54bf8d69288fbee4904"

func reservationAt(at time.Time) commitInfo {
	return commitInfo{Kind: KindReservation, CommittedAt: at, ClaimedAt: at, Parent: "p"}
}

func plainAt(at time.Time) commitInfo {
	return commitInfo{Kind: KindPlainTombstone, CommittedAt: at, ClaimedAt: at.Add(-time.Hour), ReleasedAt: at, Parent: "p"}
}

func handBackAt(at time.Time) commitInfo {
	c := plainAt(at)
	c.Kind = KindHandBack
	return c
}

// root marks the last commit of a chain as the root.
func root(chain ...commitInfo) []commitInfo {
	chain[len(chain)-1].Parent = ""
	return chain
}

func TestAPlainReleaseIsSpentAtTheNextSweepWithoutTheAge(t *testing.T) {
	f := facts{Chain: root(plainAt(t0), reservationAt(t0.Add(-time.Hour))), ChainWhole: true, EffectiveExpiry: t0}
	// The tombstone's effective expiry is its release, so the take-over margin
	// counts past it; past that, a release is spent at the next sweep, whatever
	// the age (the product thinker's ruling of 2026-10-28, iss-2610281719275461).
	settled := t0.Add(claim.TakeoverMargin)
	if v := judge(f, settled.Add(-time.Second), week); v.Spent || v.Reason != KeptRecent {
		t.Fatalf("a second inside the take-over margin = %+v; want kept as recent", v)
	}
	if v := judge(f, settled, week); !v.Spent || v.Reason != SpentReleased {
		t.Fatalf("once the margin has passed, a week short of the age = %+v; want spent as released", v)
	}
}

func TestALapsedReservationNeverReleasedStillWaitsTheAge(t *testing.T) {
	exp := t0.Add(2 * time.Hour)
	f := facts{Chain: root(reservationAt(t0)), ChainWhole: true, EffectiveExpiry: exp}
	settled := exp.Add(claim.TakeoverMargin)
	if v := judge(f, settled.Add(week-time.Second), week); v.Spent || v.Reason != KeptRecent {
		t.Fatalf("a lapse a second short of the age = %+v; want kept as recent", v)
	}
	if v := judge(f, settled.Add(week), week); !v.Spent || v.Reason != SpentUnreleased {
		t.Fatalf("a lapse at the age = %+v; want spent as lapsed and unreleased", v)
	}
}

func TestALapsedReservationOverAPlainReleaseIsSpentPastItsGraceAndTheAge(t *testing.T) {
	claimed := t0.Add(2 * time.Hour)
	exp := claimed.Add(2 * time.Hour)
	f := facts{Chain: root(reservationAt(claimed), plainAt(t0), reservationAt(t0.Add(-time.Hour))), ChainWhole: true, EffectiveExpiry: exp}
	// Lapsed a week ago, but inside the five-minute grace beyond that week.
	if v := judge(f, exp.Add(week), week); v.Spent || v.Reason != KeptRecent {
		t.Fatalf("a week past the expiry but inside the grace = %+v; want kept", v)
	}
	if v := judge(f, exp.Add(claim.TakeoverMargin+week-time.Second), week); v.Spent {
		t.Fatalf("a second before the grace and the age = %+v; want kept", v)
	}
	if v := judge(f, exp.Add(claim.TakeoverMargin+week), week); !v.Spent || v.Reason != SpentLapsed {
		t.Fatalf("at the grace and the age = %+v; want spent as lapsed", v)
	}
}

func TestALiveReservationIsNeverSpent(t *testing.T) {
	// Claimed long ago by its own date, but the claim reader judged it live
	// (a tip dated ahead is counted from the reader's now).
	f := facts{Chain: root(reservationAt(t0), plainAt(t0.Add(-time.Hour))), ChainWhole: true, EffectiveExpiry: t0.Add(30 * week)}
	for _, terminal := range []bool{false, true} {
		f.Terminal = terminal
		if v := judge(f, t0.Add(10*week), week); v.Spent {
			t.Fatalf("a reservation the claim reader holds live (terminal %v) = %+v; want kept", terminal, v)
		}
	}
}

func TestATipDatedAheadOfTheReaderIsKept(t *testing.T) {
	ahead := t0.Add(52 * week)
	f := facts{Chain: root(plainAt(ahead)), ChainWhole: true, EffectiveExpiry: ahead, Terminal: true}
	if v := judge(f, t0.Add(2*week), week); v.Spent {
		t.Fatalf("a tombstone dated a year ahead = %+v; want kept", v)
	}
}

func TestATerminalRecordIsSpentWhateverItsTipOnceOlderThanTheAge(t *testing.T) {
	exp := t0.Add(2 * time.Hour)
	for name, chain := range map[string][]commitInfo{
		"a lapsed reservation never released": root(reservationAt(t0)),
		"a hand-back":                         root(handBackAt(t0), reservationAt(t0.Add(-time.Hour))),
	} {
		f := facts{Chain: chain, ChainWhole: true, EffectiveExpiry: exp, Terminal: true}
		settled := latest(chain[0].CommittedAt, chain[0].ReleasedAt, exp.Add(claim.TakeoverMargin))
		if v := judge(f, settled.Add(week-time.Second), week); v.Spent {
			t.Fatalf("%s, terminal, a second before the age = %+v; want kept", name, v)
		}
		if v := judge(f, settled.Add(week), week); !v.Spent || v.Reason != SpentTerminal {
			t.Fatalf("%s, terminal, at the age = %+v; want spent as terminal", name, v)
		}
	}
	// A plain release on a terminal record is spent at the next sweep, as
	// on an open one: the release is the holder's own act.
	f := facts{Chain: root(plainAt(t0)), ChainWhole: true, EffectiveExpiry: t0, Terminal: true}
	settled := t0.Add(claim.TakeoverMargin)
	if v := judge(f, settled.Add(-time.Second), week); v.Spent {
		t.Fatalf("a plain release, terminal, inside the margin = %+v; want kept", v)
	}
	if v := judge(f, settled, week); !v.Spent || v.Reason != SpentTerminal {
		t.Fatalf("a plain release, terminal, past the margin = %+v; want spent as terminal", v)
	}
}

func TestATerminalRecordWhoseTipIsUnknownIsKept(t *testing.T) {
	tip := commitInfo{Kind: KindUnknown, CommittedAt: t0, Why: "it carries a trailer the sweep does not know"}
	f := facts{Chain: []commitInfo{tip}, EffectiveExpiry: t0, Terminal: true}
	if v := judge(f, t0.Add(10*week), week); v.Spent || v.Reason != KeptUnknown {
		t.Fatalf("an unknown tip on a terminal record = %+v; want kept as unknown", v)
	}
}

func TestAHandBackOnAnOpenRecordKeepsTheBranch(t *testing.T) {
	later := t0.Add(time.Hour)
	for name, chain := range map[string][]commitInfo{
		"the tip":                               root(handBackAt(t0), reservationAt(t0.Add(-time.Hour))),
		"under a plain release that dropped it": root(plainAt(later.Add(time.Hour)), reservationAt(later), handBackAt(t0), reservationAt(t0.Add(-time.Hour))),
		"under a lapsed reservation":            root(reservationAt(later), handBackAt(t0), reservationAt(t0.Add(-time.Hour))),
		"two releases down":                     root(plainAt(later.Add(3*time.Hour)), reservationAt(later.Add(2*time.Hour)), plainAt(later.Add(time.Hour)), reservationAt(later), handBackAt(t0), reservationAt(t0.Add(-time.Hour))),
	} {
		f := facts{Chain: chain, ChainWhole: true, EffectiveExpiry: chain[0].CommittedAt.Add(2 * time.Hour)}
		if v := judge(f, t0.Add(10*week), week); v.Spent || v.Reason != KeptHandBack {
			t.Fatalf("a hand-back at %s on an open record = %+v; want kept as a hand-back", name, v)
		}
	}
}

func TestAnUnknownCommitAnywhereKeepsAnOpenRecordsBranch(t *testing.T) {
	unknown := commitInfo{Kind: KindUnknown, CommittedAt: t0, Why: "it carries a trailer the sweep does not know"}
	f := facts{Chain: []commitInfo{plainAt(t0.Add(time.Hour)), unknown}, EffectiveExpiry: t0.Add(time.Hour)}
	if v := judge(f, t0.Add(10*week), week); v.Spent || v.Reason != KeptUnknown {
		t.Fatalf("an unknown commit under a plain release = %+v; want kept as unknown", v)
	}
}

// TestALapsedReservationNeverReleasedIsSpentOnceOlderThanTheAge: a lease
// that lapsed without a release protects nothing, since any session may take
// it over, so the branch is spent once its lapse is older than the age, and
// kept a second before.
func TestALapsedReservationNeverReleasedIsSpentOnceOlderThanTheAge(t *testing.T) {
	exp := t0.Add(3 * time.Hour)
	f := facts{Chain: root(reservationAt(t0.Add(time.Hour)), reservationAt(t0)), ChainWhole: true, EffectiveExpiry: exp}
	settled := exp.Add(claim.TakeoverMargin)
	if v := judge(f, settled.Add(week-time.Second), week); v.Spent || v.Reason != KeptRecent {
		t.Fatalf("a reservation never released, a second before the age = %+v; want kept as recent", v)
	}
	if v := judge(f, settled.Add(week), week); !v.Spent || v.Reason != SpentUnreleased {
		t.Fatalf("a reservation never released, at the age = %+v; want spent as lapsed unreleased", v)
	}
}

// TestALapsedReservationNeverReleasedWithUnreadHistoryIsKept: the lapse rule
// reads the whole line, as the release rule does.
func TestALapsedReservationNeverReleasedWithUnreadHistoryIsKept(t *testing.T) {
	f := facts{Chain: root(reservationAt(t0)), ChainWhole: false, EffectiveExpiry: t0.Add(time.Hour)}
	if v := judge(f, t0.Add(10*week), week); v.Spent || v.Reason != KeptHistory {
		t.Fatalf("a reservation never released, history unread = %+v; want kept as history unread", v)
	}
}

func TestAHistoryNotReadToItsRootKeepsTheBranch(t *testing.T) {
	f := facts{Chain: []commitInfo{plainAt(t0), reservationAt(t0.Add(-time.Hour))}, ChainWhole: false, EffectiveExpiry: t0}
	if v := judge(f, t0.Add(10*week), week); v.Spent || v.Reason != KeptHistory {
		t.Fatalf("a branch whose history stops short of its root = %+v; want kept", v)
	}
}

func TestAnUnreadableTipIsKept(t *testing.T) {
	f := facts{Unreadable: "its tree is not the empty tree", Terminal: true}
	if v := judge(f, t0.Add(10*week), week); v.Spent || v.Reason != KeptUnreadable {
		t.Fatalf("an unreadable tip = %+v; want kept", v)
	}
}

// commitObject is a commit object as `git cat-file commit` prints it.
func commitObject(tree, parent string, at time.Time, msg string) []byte {
	var b strings.Builder
	b.WriteString("tree " + tree + "\n")
	if parent != "" {
		b.WriteString("parent " + parent + "\n")
	}
	stamp := " <pat@example.com> " + strconv.FormatInt(at.Unix(), 10) + " +0000\n"
	b.WriteString("author Pat" + stamp)
	b.WriteString("committer Pat" + stamp)
	b.WriteString("\n" + msg)
	return []byte(b.String())
}

func rfc(t time.Time) string { return t.UTC().Format(time.RFC3339) }

// reservationText and tombstoneText are the messages internal/core/claim
// writes; handBackText the drain work's hand-back tombstone.
func reservationText(id string, claimed time.Time) string {
	return "abcd reservation: " + id + "\n\nAbcd-Reservation: " + id + "\nAbcd-Holder: octo\nAbcd-Copy: " + copyKey + "\n" +
		"Abcd-Footprint: internal/x.go\nAbcd-Claimed-At: " + rfc(claimed) + "\nAbcd-Expires-At: " + rfc(claimed.Add(2*time.Hour)) +
		"\nAssisted-by: abcd:0.0.0\n"
}

func tombstoneText(id string, claimed, released time.Time, extra string) string {
	return "abcd reservation released: " + id + "\n\nAbcd-Reservation: " + id + "\nAbcd-Holder: octo\nAbcd-Copy: " + copyKey + "\n" +
		"Abcd-Footprint: internal/x.go\nAbcd-Claimed-At: " + rfc(claimed) + "\nAbcd-Expires-At: " + rfc(released) +
		"\nAbcd-Released-At: " + rfc(released) + "\n" + extra + "Assisted-by: abcd:0.0.0\n"
}

func handBackText(id string, claimed, released time.Time) string {
	return tombstoneText(id, claimed, released, "Abcd-Handed-Back: "+claim.HandBackUserVisible+"\nAbcd-Record: 0123456789abcdef0123456789abcdef01234567\n")
}

func TestTheClassifierReadsEachKind(t *testing.T) {
	const id = "iss-5"
	const blob = "0123456789abcdef0123456789abcdef01234567"
	rel := t0.Add(time.Hour)
	for name, tc := range map[string]struct {
		msg  string
		tree string
		want Kind
	}{
		"a reservation":                           {reservationText(id, t0), emptyTreeSHA1, KindReservation},
		"a plain tombstone":                       {tombstoneText(id, t0, rel, ""), emptyTreeSHA1, KindPlainTombstone},
		"a hand-back":                             {handBackText(id, t0, rel), emptyTreeSHA1, KindHandBack},
		"a hand-back key in another case":         {tombstoneText(id, t0, rel, "abcd-handed-back: "+claim.HandBackStuck+"\nabcd-record: "+blob+"\n"), emptyTreeSHA1, KindHandBack},
		"a hand-back kind outside the vocabulary": {tombstoneText(id, t0, rel, "Abcd-Handed-Back: needs-decision\nAbcd-Record: "+blob+"\n"), emptyTreeSHA1, KindUnknown},
		"a hand-back without its record blob":     {tombstoneText(id, t0, rel, "Abcd-Handed-Back: "+claim.HandBackStuck+"\n"), emptyTreeSHA1, KindUnknown},
		"a record blob without its kind":          {tombstoneText(id, t0, rel, "Abcd-Record: "+blob+"\n"), emptyTreeSHA1, KindUnknown},
		"a holder that is no username":            {strings.Replace(tombstoneText(id, t0, rel, ""), "Abcd-Holder: octo", "Abcd-Holder: not a user", 1), emptyTreeSHA1, KindUnknown},
		"a copy in another form":                  {strings.Replace(reservationText(id, t0), "Abcd-Copy: "+copyKey, "Abcd-Copy: laptop", 1), emptyTreeSHA1, KindUnknown},
		"a footprint outside the repository":      {strings.Replace(reservationText(id, t0), "internal/x.go", "../x.go", 1), emptyTreeSHA1, KindUnknown},
		"a message past the cap":                  {tombstoneText(id, t0, rel, "") + strings.Repeat("x", claim.MaxMessageBytes), emptyTreeSHA1, KindUnknown},
		"an unknown kind of tombstone":            {tombstoneText(id, t0, rel, "Abcd-Released-Kind: handed-over\n"), emptyTreeSHA1, KindUnknown},
		"an unknown trailer on a reservation":     {strings.Replace(reservationText(id, t0), "Assisted-by", "Abcd-Lane: 3\nAssisted-by", 1), emptyTreeSHA1, KindUnknown},
		"a hand-back on a reservation":            {strings.Replace(reservationText(id, t0), "Assisted-by", "Abcd-Handed-Back: needs-decision\nAssisted-by", 1), emptyTreeSHA1, KindUnknown},
		"a second paragraph":                      {strings.Replace(tombstoneText(id, t0, rel, ""), "\n\n", "\n\nwhy it was released\n\n", 1), emptyTreeSHA1, KindUnknown},
		"another record":                          {tombstoneText("iss-6", t0, rel, ""), emptyTreeSHA1, KindUnknown},
		"a release time that does not read":       {strings.Replace(tombstoneText(id, t0, rel, ""), "Abcd-Released-At: "+rfc(rel), "Abcd-Released-At: soon", 1), emptyTreeSHA1, KindUnknown},
		"a release under a reservation's subject": {strings.Replace(tombstoneText(id, t0, rel, ""), "abcd reservation released:", "abcd reservation:", 1), emptyTreeSHA1, KindUnknown},
		"a tree that is not empty":                {tombstoneText(id, t0, rel, ""), "1111111111111111111111111111111111111111", KindUnknown},
	} {
		c := classify(id, "abc", commitObject(tc.tree, "", rel, tc.msg), emptyTreeSHA1)
		if c.Kind != tc.want {
			t.Errorf("%s = %s (%s); want %s", name, c.Kind, c.Why, tc.want)
		}
		if c.Kind == KindUnknown && c.Why == "" {
			t.Errorf("%s: unknown with no reason", name)
		}
	}
}

// TestAMergeCommitIsUnknown: abcd writes only one-parent commits on a
// reservation branch, and the sweep reads the first-parent line alone, so a
// commit with a second parent could carry a hand-back the walk never sees. It
// is unknown, and its branch is kept.
func TestAMergeCommitIsUnknown(t *testing.T) {
	const id = "iss-5"
	obj := commitObject(emptyTreeSHA1, "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", t0, reservationText(id, t0))
	obj = []byte(strings.Replace(string(obj), "\nauthor ", "\nparent bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb\nauthor ", 1))
	if c := classify(id, "abc", obj, emptyTreeSHA1); c.Kind != KindUnknown || c.Why == "" {
		t.Fatalf("a reservation with two parents = %+v; want unknown, with a reason", c)
	}
}
