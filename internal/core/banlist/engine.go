package banlist

import (
	"errors"
	"io"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"sync"

	"github.com/intentdriven/abcd/internal/core/tools"
)

// patternFault classifies a PRIVATE pattern against the engine that actually
// enforces it — the committed hook's `grep -iE`, a POSIX extended regular
// expression matcher — and not against Go's RE2. The two languages overlap but do
// not agree, and every disagreement is a silent protection failure in one
// direction or the other: an RE2-only construct (`\d`, `(?i)`, `(?:…)`) is
// accepted by grep as something ELSE and matches nothing, so the entry is inert
// while every surface reports it healthy; an ERE-invalid construct (`[a-z-.]`) is
// refused by grep, so the guard's fail-safe branch blocks EVERY commit while the
// same surface still reports the store healthy. Checking against RE2 detects
// neither.
type patternFault int

const (
	// faultNone: the pattern is usable by the enforcement engine.
	faultNone patternFault = iota
	// faultEmpty: an empty pattern matches everything or nothing depending on the
	// engine; either way it is never what the author meant.
	faultEmpty
	// faultLineBreak: a CR or LF would write extra lines into a line-oriented
	// store — one refused pattern could add entries nobody asked for.
	faultLineBreak
	// faultPerlish: a Perl-style escape or a `(?…)` group. grep ACCEPTS these and
	// reads them as something else, so the entry matches nothing.
	faultPerlish
	// faultRefused: the engine refuses the expression outright.
	faultRefused
)

// perlishRe screens for the Perl-style escapes POSIX ERE does not define: a
// backslash followed by an ASCII alphanumeric (`\d \w \s \b \1` …). It is
// deliberately conservative — it also catches the harmless `\\d` (an escaped
// backslash followed by a literal d) — because a false refusal costs one rewrite
// of a pattern while a false acceptance costs a protection layer that looks live.
var perlishRe = regexp.MustCompile(`\\[0-9A-Za-z]`)

// checkPattern classifies pattern, and reports whether the verdict came from the
// real engine (false means grep could not be executed and the weaker RE2 fallback
// was used, which a refusal message must say). The pattern itself is never
// returned, logged, or placed in argv: on the private layer it is the secret.
func checkPattern(pattern string) (fault patternFault, byEngine bool) {
	switch {
	case pattern == "":
		return faultEmpty, false
	case strings.ContainsAny(pattern, "\r\n"):
		return faultLineBreak, false
	}
	// The perlish SCREEN no longer short-circuits: whether a Perl-style construct is
	// inert is decided by the real engine, not asserted statically. GNU grep
	// IMPLEMENTS `\b`, `\w`, `\s` (so `widgetworks\b` matches — exit 0), and reads
	// `\d`/`(?…)` as something else; classifying all of them as "matches nothing"
	// before ever asking grep was a false claim. So probe grep FIRST — a construct it
	// refuses is faultRefused, one it accepts but that is written in a non-portable
	// Perl style is faultPerlish (a portability caveat, not "matches nothing").
	perlish := perlishRe.MatchString(pattern) || strings.Contains(pattern, "(?")
	if accepted, available := grepAccepts(pattern); available {
		switch {
		case !accepted:
			return faultRefused, true
		case perlish:
			return faultPerlish, true
		default:
			return faultNone, true
		}
	}
	// No grep on this machine: fall back to an RE2 compile, which proves less (the
	// engines disagree) but still rejects gross nonsense. The compile error is
	// DISCARDED — Go's regexp errors quote the expression. The static perlish screen
	// stands in for the probe here, since grep's own verdict cannot be had.
	if _, err := regexp.Compile(pattern); err != nil {
		return faultRefused, false
	}
	if perlish {
		return faultPerlish, false
	}
	return faultNone, false
}

// grepBinOnce caches the engine lookup: a list of N entries would otherwise pay N
// PATH searches.
var (
	grepBinOnce sync.Once
	grepBin     string
)

// lookGrep finds the engine once, through the PATH lookup every program abcd
// runs by name goes through (tools.LookPath): a grep inside the working tree
// abcd runs from is no engine.
func lookGrep() {
	grepBinOnce.Do(func() { grepBin, _ = tools.LookPath("grep", "") })
}

// grepAccepts asks the enforcement engine itself whether pattern is a usable
// expression, by matching it against an empty input. The pattern travels on
// STDIN via `-f -`, never in argv: an argument is world-readable in /proc/<pid>/cmdline
// while the process lives, and is captured verbatim by process auditing.
//
// Exit status is grep's documented contract: 0 a match, 1 no match, ≥2 an error —
// and on an empty input only the pattern can be at fault, so 1 means usable and ≥2
// means refused. Both streams are discarded: grep's diagnostic quotes the
// expression.
func grepAccepts(pattern string) (accepted, available bool) {
	lookGrep()
	if grepBin == "" {
		return false, false
	}
	cmd := exec.Command(grepBin, "-iqE", "-f", "-", "--", os.DevNull)
	// LC_ALL=C so the verdict does not depend on the caller's locale, matching the
	// hook, which sets it for the same reason.
	cmd.Env = append(os.Environ(), "LC_ALL=C")
	cmd.Stdin = strings.NewReader(pattern + "\n")
	cmd.Stdout, cmd.Stderr = io.Discard, io.Discard
	err := cmd.Run()
	if err == nil {
		return true, true
	}
	var exit *exec.ExitError
	if errors.As(err, &exit) {
		return exit.ExitCode() == 1, true
	}
	// Could not be executed at all (permissions, a broken PATH entry): report the
	// engine unavailable so the caller falls back rather than reading a failure to
	// ASK as a failure to PARSE.
	return false, false
}

// patternRefusal renders a fault as a message that names the construct class and
// never the pattern. It is the one place a private-pattern refusal is worded, so
// no caller can accidentally interpolate the value.
func patternRefusal(fault patternFault, byEngine bool) string {
	switch fault {
	case faultEmpty:
		return "empty"
	case faultLineBreak:
		return "contains a carriage return or newline, which a line-oriented store cannot hold"
	case faultPerlish:
		return "uses a Perl-style escape (`\\d`, `\\b`, `\\w`, …) or a `(?…)` group; POSIX ERE does not define these and grep " +
			"implementations diverge on them — some read a given escape literally, so the pattern may not be portable across grep " +
			"implementations and may match differently than written; write it in plain POSIX ERE (matching is already " +
			"case-insensitive, so `(?i)` is never needed)"
	case faultRefused:
		if byEngine {
			return "the guard's POSIX grep refuses it as an extended regular expression"
		}
		return "not a usable regular expression (checked against Go's engine only: grep could not be run, so the guard's own reading is unverified)"
	}
	return ""
}
