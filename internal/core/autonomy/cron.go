package autonomy

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
)

// ErrCron is a cron expression abcd's parser does not read. Such an
// expression never qualifies as a wake.
var ErrCron = errors.New("autonomy: not a five-field cron expression abcd reads")

// Cron is a parsed five-field cron expression: minute, hour, day of month,
// month and day of week, each a set of the values it matches. It follows
// vixie cron, as the host says its scheduled tasks do: when both day fields
// are restricted either matches, and a day field written from `*` (`*/n`
// included) counts as unrestricted for that rule.
type Cron struct {
	minute, hour, dom, month, dow uint64
	domStar, dowStar              bool
	// everyFive is exactly `*/5 * * * *`, which the host runs on time.
	everyFive bool
}

type cronField struct {
	lo, hi int
}

var cronFields = [5]cronField{{0, 59}, {0, 23}, {1, 31}, {1, 12}, {0, 7}}

// ParseCron reads a five-field expression: each field a comma list of `*`,
// `*/n`, `a`, `a-b` or `a-b/n`, in numbers (no names, no `@` forms); day of
// week takes 0 or 7 for Sunday. Anything else is ErrCron.
func ParseCron(expr string) (Cron, error) {
	f := strings.Fields(expr)
	if len(f) != 5 {
		return Cron{}, fmt.Errorf("%w: %d fields, want 5", ErrCron, len(f))
	}
	var sets [5]uint64
	for i, s := range f {
		set, err := parseCronField(s, cronFields[i])
		if err != nil {
			return Cron{}, fmt.Errorf("%w: field %d: %v", ErrCron, i+1, err)
		}
		sets[i] = set
	}
	if sets[4]&(1<<7) != 0 {
		sets[4] = (sets[4] | 1) &^ (1 << 7)
	}
	return Cron{
		minute: sets[0], hour: sets[1], dom: sets[2], month: sets[3], dow: sets[4],
		domStar:   strings.HasPrefix(f[2], "*"),
		dowStar:   strings.HasPrefix(f[4], "*"),
		everyFive: strings.Join(f, " ") == "*/5 * * * *",
	}, nil
}

func parseCronField(s string, r cronField) (uint64, error) {
	var set uint64
	for _, item := range strings.Split(s, ",") {
		if item == "" {
			return 0, errors.New("an empty list item")
		}
		rng, stepText, hasStep := strings.Cut(item, "/")
		step := 1
		if hasStep {
			n, err := cronNumber(stepText)
			if err != nil || n < 1 {
				return 0, fmt.Errorf("step %q", stepText)
			}
			step = n
		}
		lo, hi := r.lo, r.hi
		switch {
		case rng == "*":
		case strings.Contains(rng, "-"):
			a, b, _ := strings.Cut(rng, "-")
			x, err1 := cronNumber(a)
			y, err2 := cronNumber(b)
			if err1 != nil || err2 != nil || x > y {
				return 0, fmt.Errorf("range %q", rng)
			}
			lo, hi = x, y
		default:
			if hasStep {
				return 0, fmt.Errorf("a step on a single value %q", item)
			}
			x, err := cronNumber(rng)
			if err != nil {
				return 0, fmt.Errorf("value %q", rng)
			}
			lo, hi = x, x
		}
		if lo < r.lo || hi > r.hi {
			return 0, fmt.Errorf("%q is outside %d-%d", item, r.lo, r.hi)
		}
		for v := lo; v <= hi; v += step {
			set |= 1 << uint(v)
		}
	}
	return set, nil
}

func cronNumber(s string) (int, error) {
	if s == "" || len(s) > 2 {
		return 0, errors.New("not a number")
	}
	for _, c := range s {
		if c < '0' || c > '9' {
			return 0, errors.New("not a number")
		}
	}
	return strconv.Atoi(s)
}

// matches reports whether the wall-clock minute w (in its own location)
// matches the expression.
func (c Cron) matches(w time.Time) bool {
	if c.minute&(1<<uint(w.Minute())) == 0 || c.hour&(1<<uint(w.Hour())) == 0 || c.month&(1<<uint(w.Month())) == 0 {
		return false
	}
	domOK := c.dom&(1<<uint(w.Day())) != 0
	dowOK := c.dow&(1<<uint(w.Weekday())) != 0
	if c.domStar || c.dowStar {
		return domOK && dowOK
	}
	return domOK || dowOK
}

// Next is the first time strictly after from, and no later than until, at
// which the expression matches, read in loc's wall-clock time (nil means the
// machine's local time, as the host reads its tasks). It steps minute by
// minute in absolute time, so a wall-clock minute a daylight-saving change
// skips never matches, and one it repeats matches only at its first
// occurrence.
func (c Cron) Next(from, until time.Time, loc *time.Location) (time.Time, bool) {
	if loc == nil {
		loc = time.Local
	}
	for t := from.Truncate(time.Minute).Add(time.Minute); !t.After(until); t = t.Add(time.Minute) {
		w := t.In(loc)
		if c.matches(w) && !repeatedWallMinute(t, loc) {
			return t, true
		}
	}
	return time.Time{}, false
}

// repeatedWallMinute reports whether t's wall-clock minute in loc already
// happened earlier, as the hour after a fall-back change repeats the one
// before it.
func repeatedWallMinute(t time.Time, loc *time.Location) bool {
	_, now := t.In(loc).Zone()
	_, before := t.Add(-24 * time.Hour).In(loc).Zone()
	d := time.Duration(before-now) * time.Second
	if d <= 0 {
		return false
	}
	return t.Add(-d).In(loc).Format("2006-01-02 15:04") == t.In(loc).Format("2006-01-02 15:04")
}
