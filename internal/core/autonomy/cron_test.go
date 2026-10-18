package autonomy

import (
	"testing"
	"time"
)

// cronNext is the next match of expr strictly after from, searched to until,
// read in loc.
func cronNext(t *testing.T, expr string, from, until time.Time, loc *time.Location) (time.Time, bool) {
	t.Helper()
	c, err := ParseCron(expr)
	if err != nil {
		t.Fatalf("ParseCron(%q): %v", expr, err)
	}
	return c.Next(from, until, loc)
}

// matchesIn lists the minutes of the day (as "HH:MM") on which expr matches,
// stepping from the start of day to its end in loc.
func matchDays(t *testing.T, expr string, from time.Time, days int, loc *time.Location) []string {
	t.Helper()
	c, err := ParseCron(expr)
	if err != nil {
		t.Fatalf("ParseCron(%q): %v", expr, err)
	}
	var got []string
	end := from.AddDate(0, 0, days)
	for cur := from; ; {
		n, ok := c.Next(cur, end, loc)
		if !ok {
			break
		}
		got = append(got, n.In(loc).Format("2006-01-02 Mon 15:04"))
		cur = n
	}
	return got
}

// TestCronNextFire: lists, ranges, steps and both day fields give the next
// fire time; an expression it cannot parse never qualifies.
func TestCronNextFire(t *testing.T) {
	loc := london(t)
	now := at(t, "2026-10-18 21:07") // a Sunday
	for _, c := range []struct {
		expr string
		want string
	}{
		{"*/10 * * * *", "2026-10-18 21:10"},
		{"0 9 * * 1-5", "2026-10-19 09:00"},
		{"15,45 * * * *", "2026-10-18 21:15"},
		{"0 22-23 * * *", "2026-10-18 22:00"},
		{"0 0 1 * *", "2026-11-01 00:00"},
		{"30 6 * 11 *", "2026-11-01 06:30"},
		{"0 7 19 * 3", "2026-10-19 07:00"}, // the 19th (a Monday) or a Wednesday: the 19th first
		{"0 7 25 * 3", "2026-10-21 07:00"}, // the 25th or a Wednesday: Wednesday the 21st first
		{"8 21 * * *", "2026-10-18 21:08"},
		{"7 21 * * *", "2026-10-19 21:07"}, // strictly after now
	} {
		got, ok := cronNext(t, c.expr, now, now.AddDate(0, 1, 0), loc)
		if !ok {
			t.Errorf("%q: no match", c.expr)
			continue
		}
		if s := got.In(loc).Format("2006-01-02 15:04"); s != c.want {
			t.Errorf("%q: next %s, want %s", c.expr, s, c.want)
		}
	}
	for _, bad := range []string{"", "* * * *", "* * * * * *", "60 * * * *", "* 24 * * *", "* * 0 * *", "* * * 13 *",
		"* * * * 8", "a * * * *", "*/0 * * * *", "5-1 * * * *", "1,,2 * * * *", "0 9 * * MON", "@daily", "-1 * * * *"} {
		if _, err := ParseCron(bad); err == nil {
			t.Errorf("ParseCron(%q) parsed; want refused", bad)
		}
		if _, _, ok := LatestFire(SessionCron{Schedule: bad, Recurring: true}, now, now.Add(time.Hour), loc); ok {
			t.Errorf("an unparsable %q gave a fire time", bad)
		}
	}
}

// TestCronDayOfWeekSeven: 7 matches Sunday as 0 does.
func TestCronDayOfWeekSeven(t *testing.T) {
	loc := london(t)
	from := at(t, "2026-10-12 00:00") // a Monday
	for _, expr := range []string{"0 9 * * 7", "0 9 * * 0", "0 9 * * 6-7"} {
		got := matchDays(t, expr, from, 7, loc)
		found := false
		for _, g := range got {
			if g == "2026-10-18 Sun 09:00" {
				found = true
			}
		}
		if !found {
			t.Errorf("%q does not match Sunday: %v", expr, got)
		}
	}
	if got := matchDays(t, "0 9 * * 7", from, 7, loc); len(got) != 1 {
		t.Errorf("0 9 * * 7 matched %v, want Sunday alone", got)
	}
}

// TestCronRangeStep: 10-50/20 matches minutes 10, 30 and 50.
func TestCronRangeStep(t *testing.T) {
	loc := london(t)
	from := at(t, "2026-10-18 20:59")
	got := matchDays(t, "10-50/20 21 18 10 *", from, 1, loc)
	want := []string{"2026-10-18 Sun 21:10", "2026-10-18 Sun 21:30", "2026-10-18 Sun 21:50"}
	if len(got) != len(want) {
		t.Fatalf("10-50/20 matched %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("10-50/20 matched %v, want %v", got, want)
		}
	}
}

// TestCronStarStepInDayField: `0 9 */2 * 1` matches only days both
// odd-numbered and Monday, since a field written from `*` counts as
// unrestricted; `0 9 1 * 1` matches either.
func TestCronStarStepInDayField(t *testing.T) {
	loc := london(t)
	from := at(t, "2026-10-31 23:00")
	got := matchDays(t, "0 9 */2 * 1", from, 30, loc)
	for _, g := range got {
		d, _ := time.ParseInLocation("2006-01-02 Mon 15:04", g, loc)
		if d.Weekday() != time.Monday || d.Day()%2 != 1 {
			t.Errorf("0 9 */2 * 1 matched %s, which is not an odd-numbered Monday", g)
		}
	}
	// November 2026: Mondays 2, 9, 16, 23, 30; the odd ones are the 9th and 23rd.
	if len(got) != 2 || got[0] != "2026-11-09 Mon 09:00" || got[1] != "2026-11-23 Mon 09:00" {
		t.Errorf("0 9 */2 * 1 matched %v, want the 9th and the 23rd", got)
	}
	either := matchDays(t, "0 9 1 * 1", from, 30, loc)
	// The 1st (a Sunday) and every Monday: 1, 2, 9, 16, 23, 30.
	if len(either) != 6 || either[0] != "2026-11-01 Sun 09:00" {
		t.Errorf("0 9 1 * 1 matched %v, want the 1st and every Monday", either)
	}
}

// TestCronDaylightSaving: in Europe/London, `30 1 * * *` has no match on the
// spring-forward day and matches once, at its first occurrence, on the
// fall-back day.
func TestCronDaylightSaving(t *testing.T) {
	loc := london(t)
	spring := time.Date(2026, 3, 29, 0, 0, 0, 0, loc)
	if got, ok := cronNext(t, "30 1 * * *", spring, spring.Add(23*time.Hour), loc); ok {
		t.Errorf("spring-forward day: 30 1 * * * matched at %s, a minute that never happens", got)
	}
	fall := time.Date(2026, 10, 25, 0, 0, 0, 0, loc)
	end := fall.Add(24 * time.Hour)
	first, ok := cronNext(t, "30 1 * * *", fall, end, loc)
	if !ok {
		t.Fatal("fall-back day: no match")
	}
	if _, off := first.In(loc).Zone(); off != 3600 {
		t.Errorf("fall-back day: matched %s, want the first occurrence (BST)", first.In(loc).Format(time.RFC3339))
	}
	if again, ok := cronNext(t, "30 1 * * *", first, end, loc); ok {
		t.Errorf("fall-back day: matched again at %s; the repeated minute matches once", again.In(loc).Format(time.RFC3339))
	}
}
