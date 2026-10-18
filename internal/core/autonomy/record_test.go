package autonomy

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"
)

// london is the zone the clock-time cases are read in: a named zone with a
// daylight-saving change, so a case that silently used UTC would show.
func london(t *testing.T) *time.Location {
	t.Helper()
	loc, err := time.LoadLocation("Europe/London")
	if err != nil {
		t.Skipf("no zoneinfo for Europe/London: %v", err)
	}
	return loc
}

func at(t *testing.T, s string) time.Time {
	t.Helper()
	v, err := time.ParseInLocation("2006-01-02 15:04", s, london(t))
	if err != nil {
		t.Fatal(err)
	}
	return v
}

var alice = Who{Name: "alice", Copy: "/work/abcd"}

// TestUntilClockTimeAndTimestamp: --until takes a clock time, read as its next
// occurrence in local time, or a full timestamp; a time already past, and one
// that is neither, are refused.
func TestUntilClockTimeAndTimestamp(t *testing.T) {
	now := at(t, "2026-10-18 21:00")
	for _, c := range []struct {
		in   string
		want time.Time
	}{
		{"09:00", at(t, "2026-10-19 09:00")},
		{"9:05", at(t, "2026-10-19 09:05")},
		{"23:30", at(t, "2026-10-18 23:30")},
		{"2026-10-20T09:00:00+01:00", at(t, "2026-10-20 09:00")},
		{"2026-10-20T08:00:00Z", at(t, "2026-10-20 09:00")},
		{"2026-10-20T09:00", at(t, "2026-10-20 09:00")},
		{"2026-10-20 09:00", at(t, "2026-10-20 09:00")},
	} {
		got, err := parseUntil(c.in, now)
		if err != nil {
			t.Errorf("parseUntil(%q): %v", c.in, err)
			continue
		}
		if !got.Equal(c.want) {
			t.Errorf("parseUntil(%q) = %s, want %s", c.in, got, c.want)
		}
	}
	for _, in := range []string{"", "tomorrow", "25:00", "09:60", "2026-13-01T09:00"} {
		if _, err := parseUntil(in, now); !errors.Is(err, ErrUntilUnreadable) {
			t.Errorf("parseUntil(%q) err = %v, want ErrUntilUnreadable", in, err)
		}
	}
	for _, in := range []string{"2026-10-18T20:00", "2026-10-18 21:00"} {
		if _, err := parseUntil(in, now); !errors.Is(err, ErrUntilPast) {
			t.Errorf("parseUntil(%q) err = %v, want ErrUntilPast", in, err)
		}
	}
}

// TestStaleRecordCapSevenDays: an open-ended record reads as ended seven days
// after it was set, an --until past that is refused, and a stored end past
// the cap (a hand edit) is read as the cap.
func TestStaleRecordCapSevenDays(t *testing.T) {
	now := at(t, "2026-10-18 21:00")
	r, err := begin(nil, alice, now, time.Time{}, false)
	if err != nil {
		t.Fatal(err)
	}
	if !r.OpenEnded {
		t.Error("a record set without --until is not marked open-ended")
	}
	if want := now.Add(MaxWindow); !r.Ends().Equal(want) {
		t.Errorf("open-ended record ends %s, want %s", r.Ends(), want)
	}
	if !r.Holds(now.Add(MaxWindow - time.Second)) {
		t.Error("record does not hold one second before its seven days end")
	}
	if r.Holds(now.Add(MaxWindow)) {
		t.Error("record still holds seven days on")
	}
	if _, err := begin(nil, alice, now, now.Add(MaxWindow+time.Minute), false); !errors.Is(err, ErrUntilBeyondCap) {
		t.Errorf("--until past seven days: err = %v, want ErrUntilBeyondCap", err)
	}
	if _, err := begin(nil, alice, now, now.Add(MaxWindow), false); err != nil {
		t.Errorf("--until at exactly seven days refused: %v", err)
	}
	edited := *r
	edited.OpenEnded = false
	edited.Until = now.Add(30 * 24 * time.Hour)
	if !edited.Ends().Equal(now.Add(MaxWindow)) {
		t.Errorf("a stored end thirty days on reads as %s, want the cap", edited.Ends())
	}
}

// TestRecordPastEndReadsEnded: a record holds while now is before its end, and
// past it reads as ended, lapsed until it is closed.
func TestRecordPastEndReadsEnded(t *testing.T) {
	now := at(t, "2026-10-18 21:00")
	r, err := begin(nil, alice, now, now.Add(time.Hour), false)
	if err != nil {
		t.Fatal(err)
	}
	if !r.Holds(now.Add(59 * time.Minute)) {
		t.Error("record does not hold before its end")
	}
	if r.Holds(now.Add(time.Hour)) {
		t.Error("record holds at its end")
	}
	if !r.Lapsed(now.Add(2 * time.Hour)) {
		t.Error("an unclosed record past its end does not read as lapsed")
	}
	r.Close(alice, now.Add(3*time.Hour))
	if r.Lapsed(now.Add(4*time.Hour)) || r.Holds(now.Add(30*time.Minute)) {
		t.Error("a closed record still reads as lapsed or holding")
	}
	var nilRec *Record
	if nilRec.Holds(now) || nilRec.Lapsed(now) {
		t.Error("no record reads as away")
	}
}

// TestQueueAppends: each queued step is kept in order with where and when it
// was queued; an empty step or one past the length bound is refused.
func TestQueueAppends(t *testing.T) {
	now := at(t, "2026-10-18 21:00")
	r, err := begin(nil, alice, now, time.Time{}, false)
	if err != nil {
		t.Fatal(err)
	}
	if err := r.AddStep("approve the embargo close", "/work/a", now.Add(time.Minute)); err != nil {
		t.Fatal(err)
	}
	if err := r.AddStep("  push feat/x  ", "/work/b", now.Add(2*time.Minute)); err != nil {
		t.Fatal(err)
	}
	if len(r.Queue) != 2 || r.Queue[0].Step != "approve the embargo close" || r.Queue[1].Step != "push feat/x" {
		t.Fatalf("queue = %+v", r.Queue)
	}
	if r.Queue[1].From != "/work/b" || !r.Queue[1].At.Equal(now.Add(2*time.Minute)) {
		t.Errorf("queued step lost its copy or time: %+v", r.Queue[1])
	}
	if err := r.AddStep("   ", "/work/a", now); !errors.Is(err, ErrStepEmpty) {
		t.Errorf("empty step: err = %v", err)
	}
	if err := r.AddStep(strings.Repeat("x", MaxStepLen+1), "/work/a", now); !errors.Is(err, ErrStepTooLong) {
		t.Errorf("over-long step: err = %v", err)
	}
	if len(r.Queue) != 2 {
		t.Errorf("a refused step was kept: %d steps", len(r.Queue))
	}

	// Setting the mode again keeps what is still unread.
	again, err := begin(r, alice, now.Add(time.Hour), time.Time{}, true)
	if err != nil {
		t.Fatal(err)
	}
	if len(again.Queue) != 2 {
		t.Errorf("setting the mode again dropped the unread queue: %+v", again.Queue)
	}
	r.Close(alice, now.Add(2*time.Hour))
	fresh, err := begin(r, alice, now.Add(3*time.Hour), time.Time{}, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(fresh.Queue) != 0 {
		t.Errorf("a closed record's queue, already printed, was carried into a new window: %+v", fresh.Queue)
	}
}

// TestRepeatedStopCollapses: a stop identical to the last recorded one adds to
// its count and last time instead of a new entry; a different stop, or the
// first one again after a different one, is its own entry.
func TestRepeatedStopCollapses(t *testing.T) {
	now := at(t, "2026-10-18 21:00")
	r, err := begin(nil, alice, now, time.Time{}, false)
	if err != nil {
		t.Fatal(err)
	}
	a := Stop{Kind: StopWorkOwed, Detail: "feat/x is behind main", PossiblyForcedByCap: true}
	b := Stop{Kind: StopAPIError, ErrorType: "rate_limit"}
	r.AddStop(a, now.Add(10*time.Minute))
	r.AddStop(a, now.Add(20*time.Minute))
	if len(r.Stops) != 1 {
		t.Fatalf("an identical stop made a new entry: %+v", r.Stops)
	}
	if s := r.Stops[0]; s.Count != 2 || !s.First.Equal(now.Add(10*time.Minute)) || !s.Last.Equal(now.Add(20*time.Minute)) {
		t.Errorf("collapsed stop = %+v, want count 2 with first and last times", s)
	}
	r.AddStop(b, now.Add(30*time.Minute))
	r.AddStop(a, now.Add(40*time.Minute))
	// An identical stop anywhere in the list collapses into its entry, not
	// only the one just before it (the diff review's major 3).
	if len(r.Stops) != 2 || r.Stops[0].Count != 3 || !r.Stops[0].Last.Equal(now.Add(40*time.Minute)) {
		t.Errorf("stops = %+v, want two entries, the first counted three times", r.Stops)
	}
	if r.Stops[1].Count != 1 {
		t.Errorf("a first stop's count = %d, want 1", r.Stops[1].Count)
	}
}

// TestHoldPushesRecorded: --hold-pushes is kept in the record and in its JSON,
// shown in the record's terms, and changes nothing else: the terms say the
// mode grants no rights whether or not pushes are held.
func TestHoldPushesRecorded(t *testing.T) {
	now := at(t, "2026-10-18 21:00")
	held, err := begin(nil, alice, now, at(t, "2026-10-19 09:00"), true)
	if err != nil {
		t.Fatal(err)
	}
	plain, err := begin(nil, alice, now, at(t, "2026-10-19 09:00"), false)
	if err != nil {
		t.Fatal(err)
	}
	if !held.HoldPushes || plain.HoldPushes {
		t.Fatalf("hold_pushes: held=%v plain=%v", held.HoldPushes, plain.HoldPushes)
	}
	raw, err := json.Marshal(held)
	if err != nil {
		t.Fatal(err)
	}
	var back Record
	if err := json.Unmarshal(raw, &back); err != nil {
		t.Fatal(err)
	}
	if !back.HoldPushes || !strings.Contains(string(raw), `"hold_pushes":true`) {
		t.Errorf("hold_pushes lost in the round trip: %s", raw)
	}
	heldTerms := strings.Join(held.Terms(), "\n")
	plainTerms := strings.Join(plain.Terms(), "\n")
	if !strings.Contains(heldTerms, "pushes wait") {
		t.Errorf("held terms do not say pushes wait:\n%s", heldTerms)
	}
	if strings.Contains(plainTerms, "pushes wait") {
		t.Errorf("terms without --hold-pushes say pushes wait:\n%s", plainTerms)
	}
	for _, terms := range []string{heldTerms, plainTerms} {
		if !strings.Contains(terms, "grants no rights") || !strings.Contains(terms, "standing instruction") {
			t.Errorf("terms do not say the mode grants no rights and the standing instruction decides:\n%s", terms)
		}
		if !strings.Contains(terms, "abcd mode back") {
			t.Errorf("terms do not name `abcd mode back`:\n%s", terms)
		}
	}
	if held.SetBy != plain.SetBy || !held.Ends().Equal(plain.Ends()) {
		t.Error("--hold-pushes changed something other than itself")
	}
}

// TestBackLinesOnePerItem: the lines `abcd mode back` prints are one per
// queued step, per handed-back item and per recorded stop, the cap label and
// the count included.
func TestBackLinesOnePerItem(t *testing.T) {
	now := at(t, "2026-10-18 21:00")
	r, err := begin(nil, alice, now, time.Time{}, false)
	if err != nil {
		t.Fatal(err)
	}
	_ = r.AddStep("approve the embargo close", "/work/a", now)
	_ = r.AddStep("push feat/y", "/work/a", now)
	if !r.AddHandBack("feat/y", "0123456789abcdef", now) || r.AddHandBack("feat/y", "0123456789abcdef", now) {
		t.Error("a hand-back is not recorded exactly once per branch and head")
	}
	r.AddStop(Stop{Kind: StopWorkOwed, Detail: "feat/x is behind main", PossiblyForcedByCap: true}, now)
	r.AddStop(Stop{Kind: StopWorkOwed, Detail: "feat/x is behind main", PossiblyForcedByCap: true}, now.Add(time.Hour))
	r.AddStop(Stop{Kind: StopAPIError, ErrorType: "rate_limit"}, now)
	lines := r.Lines()
	if len(lines) != 5 {
		t.Fatalf("lines = %q, want five", lines)
	}
	all := strings.Join(lines, "\n")
	for _, want := range []string{"approve the embargo close", "push feat/y", "feat/y", "possibly forced by the cap", "2 times", "rate_limit"} {
		if !strings.Contains(all, want) {
			t.Errorf("lines miss %q:\n%s", want, all)
		}
	}
	for _, l := range lines {
		if strings.Contains(l, "\n") {
			t.Errorf("a line spans lines: %q", l)
		}
	}
}

// TestOwedShape: the owed set's next item follows the fixed order whatever
// order the items arrive in, its fingerprint ignores order and reasons, and
// the capped fingerprint reads the capped items alone.
func TestOwedShape(t *testing.T) {
	kinds := owedKinds()
	want := []OwedKind{OwedRenewal, OwedFailing, OwedBehind, OwedStacked, OwedUnpushed, OwedUncarried}
	if len(kinds) != len(want) {
		t.Fatalf("kinds = %v", kinds)
	}
	for i := range want {
		if kinds[i] != want[i] {
			t.Fatalf("kinds = %v, want %v", kinds, want)
		}
	}
	a := Owed{Items: []OwedItem{
		{Kind: OwedUnpushed, Capped: true, Branch: "feat/z", Head: "c3"},
		{Kind: OwedBehind, Capped: true, Branch: "feat/x", PR: 7, Head: "a1"},
		{Kind: OwedRenewal, Record: "iss-5"},
	}}
	b := Owed{Items: []OwedItem{a.Items[2], a.Items[0], a.Items[1]}}
	b.Items[0].Reason = "renewal due at 03:00"
	if next, ok := a.Next(); !ok || next.Kind != OwedRenewal {
		t.Errorf("next = %+v, want the renewal", next)
	}
	if a.Fingerprint() != b.Fingerprint() {
		t.Error("the fingerprint depends on order or on a reason")
	}
	if !a.HasCapped() {
		t.Error("HasCapped is false with capped items owed")
	}
	c := Owed{Items: a.Items[:2]}
	if a.CappedFingerprint() != c.CappedFingerprint() {
		t.Error("a renewal coming or going changed the capped fingerprint")
	}
	if a.Fingerprint() == c.Fingerprint() {
		t.Error("the full fingerprint ignored the renewal")
	}
	moved := Owed{Items: []OwedItem{a.Items[0], {Kind: OwedBehind, Capped: true, Branch: "feat/x", PR: 7, Head: "a2"}}}
	if moved.CappedFingerprint() == c.CappedFingerprint() {
		t.Error("a new head on a capped item left the capped fingerprint unchanged")
	}
	var empty Owed
	if !empty.Empty() || empty.HasCapped() {
		t.Error("an empty owed set reads as holding work")
	}
	if _, ok := empty.Next(); ok {
		t.Error("an empty owed set has a next item")
	}
}
