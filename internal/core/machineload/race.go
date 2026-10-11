package machineload

import (
	"sort"
	"strconv"
	"strings"
)

// The waiter (`abcd implement load --wait`, iss-2610111826193257) holds its gate
// shut while another `go test -race` run is live. It judges that from the
// process table and each candidate's argv, never from a pattern over a command
// line: on 2026-10-11 a hand-written waiter, `pgrep -f "go test -race"`, matched
// its own shell, whose command line carried that very string, and so never
// opened. Here a race run is a process the kernel records as executing go
// (Proc.Name) whose go command is `test` with the -race flag, and the waiter's
// own process and every ancestor of it are never candidates.

// GoProgram is the base name of the go command's executable, as the process
// table names it (Proc.Name, the kernel's record of what was executed, never a
// word of a command line).
const GoProgram = "go"

// maxLineage bounds every walk up the parent chain, so a torn snapshot whose
// parent links form a cycle ends.
const maxLineage = 4096

// isRaceTest reports whether args, a go command's arguments after the program
// name, start a test run with the race detector: the go command's own leading
// `-C dir` is skipped, the first argument that is not a flag must be `test`,
// and -race (or --race, or -race=<true>) must be among the arguments after it
// and before `-args`, which hands everything after it to the test binary.
// `go run ./cmd/abcd implement load --wait -- go test -race` is therefore not a
// race run: its subcommand is `run`, and `test -race` are its program's words.
func isRaceTest(args []string) bool {
	i := 0
	for i < len(args) {
		a := args[i]
		if a == "-C" || a == "--C" {
			i += 2
			continue
		}
		if strings.HasPrefix(a, "-C=") || strings.HasPrefix(a, "--C=") {
			i++
			continue
		}
		break
	}
	if i >= len(args) || args[i] != "test" {
		return false
	}
	for _, a := range args[i+1:] {
		if a == "-args" || a == "--args" {
			return false
		}
		flag := strings.TrimPrefix(strings.TrimPrefix(a, "-"), "-")
		if flag == a {
			continue
		}
		if flag == "race" {
			return true
		}
		if v, ok := strings.CutPrefix(flag, "race="); ok {
			if b, err := strconv.ParseBool(v); err == nil && b {
				return true
			}
		}
	}
	return false
}

// ancestry returns pid and every ancestor of it the process table holds.
func lineage(procs []Proc, pid int) map[int]bool {
	byPID := make(map[int]Proc, len(procs))
	for _, p := range procs {
		byPID[p.PID] = p
	}
	out := map[int]bool{}
	for hops := 0; pid > 0 && !out[pid] && hops < maxLineage; hops++ {
		out[pid] = true
		p, ok := byPID[pid]
		if !ok {
			break
		}
		pid = p.PPID
	}
	return out
}

// Descends reports whether pid is a descendant of ancestor (a process is not
// its own descendant).
func Descends(procs []Proc, pid, ancestor int) bool {
	if pid == ancestor {
		return false
	}
	return lineage(procs, pid)[ancestor]
}

// RaceCandidates returns, in pid order, the processes a race run could be: those
// executing go, outside self's process and its ancestors. Only these are asked
// for their arguments.
func RaceCandidates(snap Snapshot, self Self) []int {
	chain := lineage(snap.Procs, self.PID)
	var out []int
	for _, p := range snap.Procs {
		if p.Name == GoProgram && !chain[p.PID] {
			out = append(out, p.PID)
		}
	}
	sort.Ints(out)
	return out
}

// RaceRuns returns, in pid order, the candidates whose arguments (args, keyed by
// pid, the program name dropped) start a race run. A candidate with no
// arguments read (gone by the time they were asked for) is not one.
func RaceRuns(snap Snapshot, self Self, args map[int][]string) []int {
	var out []int
	for _, pid := range RaceCandidates(snap, self) {
		if a, ok := args[pid]; ok && isRaceTest(a) {
			out = append(out, pid)
		}
	}
	return out
}
