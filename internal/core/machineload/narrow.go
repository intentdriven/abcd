package machineload

import (
	"path"
	"slices"
	"sort"
	"strconv"
	"strings"
	"time"
)

// A narrow run is a go test naming a few packages outright. It takes minutes,
// and on 2026-11-04 about thirty of them queued one at a time behind whole
// preflights at a load of 12 on 16 cores until their waits timed out, with no
// test run. So a waiter whose command is one waits at a gate of its own: it
// takes no place in the first gate's queue and no slot (gateTable.waiters,
// JustOpened), and a whole run counts it through the load figure alone. Its own
// limit, load and memory still shut it (implement/load_narrow.go).
//
// Every waiter classifies every other from its arguments, itself included, so the
// argv is the one truth: a waiter is narrow only if it carries NarrowMark and its
// command is a narrow run. What the words after `--` run under is not in the argv,
// and only the waiter can read it, so a waiter that cannot honour its mark takes
// the mark off its argv before it waits (the front door's fallback). The
// classification fails closed: what it cannot read as narrow is a whole run, so the
// mistake it can make is a slower wait, never a whole run admitted past its limit.

// NarrowMark is the flag, before `--`, that asks for the narrow gate. An abcd
// built before it skips any flag it does not know when it reads a waiter's words,
// so to that abcd a marked waiter is an ordinary one, and that abcd cannot itself
// be started with the mark.
const NarrowMark = "--narrow"

// MaxNarrowPackages is the most packages a narrow run names. Three is a change's
// usual reach, its package, the caller that exercises it and the surface over
// that, and it is the most any builder's gated test named on the evening of
// 2026-11-04. go test runs packages side by side, so a run's weight grows with the
// count, and a longer list is the start of a sweep.
const MaxNarrowPackages = 3

// narrowBoolFlags and narrowValueFlags are all the flags a narrow run may carry,
// without their dashes. Each only selects or reports, so none can make a run
// heavier than the packages it names. Any other flag is a whole run, -race first:
// a race run stays where the rule keeping race stages apart can see it, whatever
// its size, and -p, -parallel, -tags, -cover, -exec and the rest either widen the
// run or are not known to be narrow.
var (
	narrowBoolFlags  = []string{"v", "short", "failfast", "json"}
	narrowValueFlags = []string{"run", "skip", "timeout", "count"}
)

// IsNarrowCommand reports whether cmd, a command a waiter gates (the words after
// `--`, its program first), is a narrow run: its program is the exact word go, as
// PATH finds it for everyone, its go command
// `test`, after `-C <dir>` at most, which a verifier's copy uses, every flag one of
// the few above (-count only as 1, since a repeat count is a soak), and at least one and at most
// MaxNarrowPackages explicit package paths, with no wildcard. Anything else is a
// whole run: a shell or a make is not read into, no package at all, `-args`, and
// a word holding whitespace, which another account's reading of the table on
// macOS would split differently.
func IsNarrowCommand(cmd []string) bool {
	if len(cmd) < 3 || cmd[0] != GoProgram {
		return false
	}
	rest := cmd[1:]
	switch {
	case rest[0] == "-C" && len(rest) > 2:
		rest = rest[2:]
	case strings.HasPrefix(rest[0], "-C="):
		rest = rest[1:]
	}
	if rest[0] != "test" {
		return false
	}
	rest = rest[1:]
	packages := 0
	for i := 0; i < len(rest); i++ {
		a := rest[i]
		if strings.ContainsAny(a, " \t\r\n") {
			return false
		}
		if !strings.HasPrefix(a, "-") {
			if !isExplicitPackage(a) {
				return false
			}
			packages++
			continue
		}
		name, value, hasValue := strings.Cut(strings.TrimPrefix(strings.TrimPrefix(a, "-"), "-"), "=")
		switch {
		case slices.Contains(narrowBoolFlags, name):
			if hasValue {
				if _, err := strconv.ParseBool(value); err != nil {
					return false
				}
			}
		case slices.Contains(narrowValueFlags, name):
			if !hasValue {
				i++
				if i >= len(rest) || strings.ContainsAny(rest[i], " \t\r\n") {
					return false
				}
				value = rest[i]
			}
			if name == "count" && value != "1" {
				return false
			}
		default:
			return false
		}
	}
	return packages >= 1 && packages <= MaxNarrowPackages
}

// isExplicitPackage reports whether word names one package outright, judged as go
// judges it (search.CleanPatterns): a local path, `.`, `..` or one starting `./` or
// `../`, or any other path that still holds a slash once cleaned, and nothing that
// expands. So a bare word is refused however it is spelt (`all`, `all/`, `x/../all`),
// and `all`, `std`, `cmd`, `work` and `tool` are never read as one package; so is a
// file list, a version, and a backslash, which go rewrites to a slash.
func isExplicitPackage(word string) bool {
	if strings.Contains(word, "...") || strings.ContainsAny(word, "*?[]{}$~\\@") || strings.HasSuffix(word, ".go") {
		return false
	}
	if word == "." || word == ".." || strings.HasPrefix(word, "./") || strings.HasPrefix(word, "../") {
		return true
	}
	return strings.Contains(path.Clean(word), "/")
}

// hasNarrowMark reports whether args carry NarrowMark before any `--`.
func hasNarrowMark(args []string) bool {
	marked := false
	for _, a := range args {
		if a == "--" {
			break
		}
		if a == NarrowMark {
			marked = true
		}
		if v, ok := strings.CutPrefix(a, NarrowMark+"="); ok {
			b, err := strconv.ParseBool(v)
			marked = err == nil && b
		}
	}
	return marked
}

// isNarrowWaiter reports whether args, abcd's arguments after the program name,
// run a first gate's waiter carrying NarrowMark whose command is a narrow run. Its
// words are a waiter's like any other, so an abcd built before the class reads it as
// one: it queues behind one that started earlier and counts one running in a slot,
// the safe direction.
func isNarrowWaiter(args []string) bool {
	wait, race := waiterFlags(args)
	return wait && !race && hasNarrowMark(args) && IsNarrowCommand(gatedCommand(args))
}

// NarrowAhead reads the narrow runs from one snapshot and the candidates'
// arguments, outside self's process and its ancestors: those running their
// commands, those queued ahead of self by the queue's order (queueOrder), and
// every one queued, each in pid order.
func NarrowAhead(snap Snapshot, self Self, args map[int][]string) (running, ahead, queued []int) {
	if !snap.HasProcs {
		return nil, nil, nil
	}
	t := readGate(snap, self, args)
	for _, pid := range t.candidates {
		if t.chain[pid] || !isAbcdName(t.byPID[pid].Name) || !isNarrowWaiter(args[pid]) {
			continue
		}
		if t.running[pid] {
			running = append(running, pid)
		} else {
			queued = append(queued, pid)
		}
	}
	ahead, _ = t.queuedAhead(queued)
	return running, ahead, queued
}

// OverdueWhole returns, in pid order, the whole-run waiters in a (self's own
// reading, so each started before self) that have been queued for at least after, with
// a whole-run slot free now, against slots (load.preflights): a waiter whose slots
// are all taken is shut by them, and no narrow run is in its way.
//
// Only a waiter that started earlier can be overdue. An older abcd counts a
// narrow waiter that started before it as ahead, so a narrow run that stood aside
// for a later start would wait on a waiter waiting on it; every wait here points
// at an earlier start, and such a chain has no cycle.
func OverdueWhole(snap Snapshot, a Ahead, slots int, after time.Duration) []int {
	if len(a.TakingSlots(slots).Busy) >= max(slots, 1) {
		return nil
	}
	var out []int
	for _, p := range snap.Procs {
		if slices.Contains(a.Queued, p.PID) && p.Age >= after {
			out = append(out, p.PID)
		}
	}
	sort.Ints(out)
	return out
}
