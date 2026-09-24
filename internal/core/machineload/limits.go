package machineload

import (
	"fmt"
	"math"
	"strconv"
	"strings"
)

// The machine's settings file, ~/.abcd/load-limits, is line-oriented like its
// siblings in the caller's machine tier (trusted-roots, local-transcript-roots):
// `#` starts a comment, blank lines are ignored, and every other line is
// `<key><space or tab><value>`. This file holds the format; the guarded read of
// the file itself (a regular file the caller owns, writable by nobody else) is
// the caller's, because it goes through the module's shared declaration read and
// this package imports only the standard library.

// LimitsFileName is the settings file's name under ~/.abcd/.
const LimitsFileName = "load-limits"

// The two keys, and their bounds.
const (
	KeyStrayMinutes = "stray-minutes"
	KeyExtremeLoad  = "extreme-load"

	// DefaultStrayMinutes is how long a process may run at a near-full core
	// before it reads as a stray.
	DefaultStrayMinutes = 30
	// DefaultExtremeFactor times the online core count is the extreme load.
	DefaultExtremeFactor = 4

	maxStrayMinutes = 10080 // one week
	maxExtremeLoad  = 100000
	// maxKeyEcho caps an offending key where a fault echoes it.
	maxKeyEcho = 32
)

// Limits are the two limits the classifier applies.
type Limits struct {
	// StrayMinutes: a process older than this at a near-full core is a stray.
	StrayMinutes int
	// ExtremeLoad: a one-minute load average strictly above this is extreme. It is
	// an absolute load, in the online-core unit the LOAD rule's cap uses.
	ExtremeLoad float64
	// StrayFromFile and ExtremeFromFile say which limits the settings file set.
	StrayFromFile   bool
	ExtremeFromFile bool
}

// DefaultLimits derives the defaults from the online core count: 30 minutes, and
// four times the cores.
func DefaultLimits(cores int) Limits {
	return Limits{StrayMinutes: DefaultStrayMinutes, ExtremeLoad: float64(DefaultExtremeFactor * cores)}
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
		if key != KeyStrayMinutes && key != KeyExtremeLoad {
			return def, &LimitsFault{Line: n, Class: fmt.Sprintf("unknown key %q (known: %s, %s)", safeKey(key), KeyStrayMinutes, KeyExtremeLoad)}
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
		case KeyExtremeLoad:
			v, err := strconv.ParseFloat(value, 64)
			if err != nil || math.IsNaN(v) || math.IsInf(v, 0) {
				return def, &LimitsFault{Line: n, Class: "the value of " + key + " is not a number"}
			}
			if v <= 0 || v > maxExtremeLoad {
				return def, &LimitsFault{Line: n, Class: fmt.Sprintf("%s is out of range (above 0, at most %d)", key, maxExtremeLoad)}
			}
			l.ExtremeLoad, l.ExtremeFromFile = v, true
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
