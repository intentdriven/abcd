package machineload

import (
	"fmt"
	"math"
	"strconv"
	"strings"
)

// The machine's settings file, ~/.abcd.noindex/load-limits, is line-oriented like its
// siblings in the caller's machine tier (trusted-roots, local-transcript-roots):
// `#` starts a comment, blank lines are ignored, and every other line is
// `<key><space or tab><value>`. This file holds the format; the guarded read of
// the file itself (a regular file the caller owns, writable by nobody else) is
// the caller's, because it goes through the module's shared declaration read and
// this package imports only the standard library.

// LimitsFileName is the settings file's name under ~/.abcd.noindex/.
const LimitsFileName = "load-limits"

// The three keys, and their bounds.
const (
	KeyStrayMinutes = "stray-minutes"
	KeyExtremeLoad  = "extreme-load"
	KeyWaitLoad     = "wait-load"

	// DefaultStrayMinutes is how long a process may run at nearly all the CPU it
	// could get before it reads as a stray.
	DefaultStrayMinutes = 30
	// DefaultExtremeFactor times the online core count is the extreme load.
	DefaultExtremeFactor = 4
	// DefaultWaitFactor times the online core count is the waiter's limit: the
	// waiter opens only while the one-minute load is strictly below twice the
	// core count. Once was too low: the baseline load of a busy multi-session
	// machine already sits around one core-count, so a limit of the cores kept
	// every waiter shut for hours (between 20 and 100 on 16 cores, 2026-10-11)
	// and sent sessions back to hand-written loops. Serialising the work is the
	// job of the race-run condition and the hold, not of the load number. One
	// race run adds about a core-count of load, so admitting below twice the
	// cores bounds the peak near three times them, under the extreme limit's
	// DefaultExtremeFactor of four.
	DefaultWaitFactor = 2

	maxStrayMinutes = 10080 // one week
	maxExtremeLoad  = 100000
	// maxKeyEcho caps an offending key where a fault echoes it.
	maxKeyEcho = 32
)

// Limits are the two limits the classifier applies, and the waiter's.
type Limits struct {
	// StrayMinutes: a process older than this at nearly all the CPU it could get
	// is a stray.
	StrayMinutes int
	// ExtremeLoad: a one-minute load average strictly above this is extreme. It is
	// an absolute load, in the online-core unit the LOAD rule's cap uses.
	ExtremeLoad float64
	// StrayFromFile and ExtremeFromFile say which limits the settings file set.
	StrayFromFile   bool
	ExtremeFromFile bool
	// WaitLoad: the waiter (`implement load --wait`) opens only while the
	// one-minute load average is strictly below this.
	WaitLoad     float64
	WaitFromFile bool
}

// DefaultLimits derives the defaults from the online core count: 30 minutes,
// four times the cores, and twice the cores.
func DefaultLimits(cores int) Limits {
	return Limits{StrayMinutes: DefaultStrayMinutes, ExtremeLoad: float64(DefaultExtremeFactor * cores),
		WaitLoad: float64(DefaultWaitFactor * cores)}
}

// LimitsFault is why a settings file is unusable. It names the line (zero when
// the fault is the file's, not a line's) and the fault class, and never a value:
// a value is whatever the caller typed, and the report is printed and logged.
type LimitsFault struct {
	Line  int
	Class string
}

func (f *LimitsFault) Error() string {
	if f.Line > 0 {
		return fmt.Sprintf("line %d: %s", f.Line, f.Class)
	}
	return f.Class
}

// ParseLimits reads a settings file's bytes. Any fault makes the whole file
// unusable: it returns the defaults for BOTH limits, never a mix of the file's
// and the defaults, with the fault.
func ParseLimits(data []byte, cores int) (Limits, *LimitsFault) {
	def := DefaultLimits(cores)
	l := def
	seen := map[string]bool{}
	for i, raw := range strings.Split(string(data), "\n") {
		n := i + 1
		line := strings.TrimSpace(raw)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		fields := strings.Fields(line)
		key := fields[0]
		if key != KeyStrayMinutes && key != KeyExtremeLoad && key != KeyWaitLoad {
			return def, &LimitsFault{Line: n, Class: fmt.Sprintf("unknown key %q (known: %s, %s, %s)", safeKey(key), KeyStrayMinutes, KeyExtremeLoad, KeyWaitLoad)}
		}
		if seen[key] {
			return def, &LimitsFault{Line: n, Class: key + " is given twice"}
		}
		seen[key] = true
		switch {
		case len(fields) == 1:
			return def, &LimitsFault{Line: n, Class: key + " has no value"}
		case len(fields) > 2:
			return def, &LimitsFault{Line: n, Class: key + " has more than one value"}
		}
		value := fields[1]
		switch key {
		case KeyStrayMinutes:
			v, err := strconv.Atoi(value)
			if err != nil {
				return def, &LimitsFault{Line: n, Class: "the value of " + key + " is not a whole number"}
			}
			if v < 1 || v > maxStrayMinutes {
				return def, &LimitsFault{Line: n, Class: fmt.Sprintf("%s is out of range (1 to %d)", key, maxStrayMinutes)}
			}
			l.StrayMinutes, l.StrayFromFile = v, true
		case KeyExtremeLoad, KeyWaitLoad:
			v, err := strconv.ParseFloat(value, 64)
			if err != nil || math.IsNaN(v) || math.IsInf(v, 0) {
				return def, &LimitsFault{Line: n, Class: "the value of " + key + " is not a number"}
			}
			if v <= 0 || v > maxExtremeLoad {
				return def, &LimitsFault{Line: n, Class: fmt.Sprintf("%s is out of range (above 0, at most %d)", key, maxExtremeLoad)}
			}
			if key == KeyWaitLoad {
				l.WaitLoad, l.WaitFromFile = v, true
			} else {
				l.ExtremeLoad, l.ExtremeFromFile = v, true
			}
		}
	}
	return l, nil
}

// safeKey makes an offending key printable: printable ASCII survives, anything
// else becomes '?', and it is capped at maxKeyEcho bytes.
func safeKey(s string) string {
	b := make([]byte, 0, maxKeyEcho)
	for i := 0; i < len(s) && len(b) < maxKeyEcho; i++ {
		c := s[i]
		if c < 0x20 || c > 0x7e {
			c = '?'
		}
		b = append(b, c)
	}
	return string(b)
}
