package matrix

// udiff.go computes the normalised diff a matrix reviewer is handed
// (adr-2610161733400551): one file's unified diff, from the two sides' bytes
// alone. It runs no git, so nothing of the copy's repository (a commit, its
// trailer, an index line, a configured diff driver or text filter) can enter
// it, and it is the same for every copy: CRLF line endings are read as LF,
// every hunk carries three lines of context, and no line names a time, a mode
// or an object.

import (
	"strconv"
	"strings"
)

// diffContext is the lines of context around each change.
const diffContext = 3

// maxDiffEdits bounds the edit search: a change needing more than this many
// lines removed and added between the two sides' common head and tail is
// shown whole, every old line removed and every new one added, rather than
// searched for at a cost that grows with its square.
const maxDiffEdits = 1000

// opKind is one line's fate in an edit script.
type opKind byte

const (
	opKeep opKind = iota
	opDel
	opAdd
)

// diffOp is one line of an edit script: kept (in both sides), deleted (from
// the old) or added (in the new), with its index in the side it is read from.
type diffOp struct {
	kind opKind
	a, b int
}

// unifiedDiff is the unified diff of the file at path from old to new, ""
// when the two sides are the same text once line endings are normalised. A
// nil old is a file added and a nil new a file deleted.
func unifiedDiff(path string, old, new []byte) string {
	a, b := diffLines(old), diffLines(new)
	ops := editScript(a, b)
	hunks := hunksOf(ops)
	if len(hunks) == 0 {
		return ""
	}
	var w strings.Builder
	from, to := "a/"+path, "b/"+path
	if old == nil {
		from = "/dev/null"
	}
	if new == nil {
		to = "/dev/null"
	}
	w.WriteString("--- " + from + "\n+++ " + to + "\n")
	for _, h := range hunks {
		writeHunk(&w, ops[h[0]:h[1]], a, b)
	}
	return w.String()
}

// diffLines splits data into lines, each keeping its "\n", with CRLF read as
// LF. A last line with no newline is kept without one, so a dropped final
// newline is a changed line.
func diffLines(data []byte) []string {
	if len(data) == 0 {
		return nil
	}
	s := strings.ReplaceAll(string(data), "\r\n", "\n")
	lines := strings.SplitAfter(s, "\n")
	if lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}
	return lines
}

// editScript is a shortest edit script from a to b: their common head and
// tail kept, and the middle searched by Myers' algorithm within maxDiffEdits,
// or else replaced whole.
func editScript(a, b []string) []diffOp {
	head := 0
	for head < len(a) && head < len(b) && a[head] == b[head] {
		head++
	}
	tail := 0
	for tail < len(a)-head && tail < len(b)-head && a[len(a)-1-tail] == b[len(b)-1-tail] {
		tail++
	}
	ops := make([]diffOp, 0, len(a)+len(b))
	for i := 0; i < head; i++ {
		ops = append(ops, diffOp{opKeep, i, i})
	}
	for _, op := range middleScript(a[head:len(a)-tail], b[head:len(b)-tail]) {
		op.a += head
		op.b += head
		ops = append(ops, op)
	}
	for i := 0; i < tail; i++ {
		ops = append(ops, diffOp{opKeep, len(a) - tail + i, len(b) - tail + i})
	}
	return ops
}

// middleScript is Myers' O(ND) search over a and b, keeping each round's
// frontier to walk the path back; past maxDiffEdits rounds it gives up and
// replaces a by b whole.
func middleScript(a, b []string) []diffOp {
	n, m := len(a), len(b)
	whole := func() []diffOp {
		ops := make([]diffOp, 0, n+m)
		for i := 0; i < n; i++ {
			ops = append(ops, diffOp{opDel, i, 0})
		}
		for j := 0; j < m; j++ {
			ops = append(ops, diffOp{opAdd, 0, j})
		}
		return ops
	}
	if n == 0 || m == 0 || n-m > maxDiffEdits || m-n > maxDiffEdits {
		return whole()
	}
	limit := min(n+m, maxDiffEdits)
	off := limit + 1
	v := make([]int, 2*limit+3)
	// trace[d] is the frontier before round d, over diagonals -d..d.
	var trace [][]int
	for d := 0; d <= limit; d++ {
		trace = append(trace, append([]int(nil), v[off-d:off+d+1]...))
		for k := -d; k <= d; k += 2 {
			var x int
			if k == -d || k != d && v[off+k-1] < v[off+k+1] {
				x = v[off+k+1]
			} else {
				x = v[off+k-1] + 1
			}
			y := x - k
			for x < n && y < m && a[x] == b[y] {
				x++
				y++
			}
			v[off+k] = x
			if x >= n && y >= m {
				return backtrack(trace, n, m)
			}
		}
	}
	return whole()
}

// backtrack walks Myers' frontiers from (n, m) back to the origin, building
// the edit script in order.
func backtrack(trace [][]int, n, m int) []diffOp {
	var rev []diffOp
	x, y := n, m
	for d := len(trace) - 1; d > 0; d-- {
		v := trace[d]
		at := func(k int) int { return v[k+d] }
		k := x - y
		var pk int
		if k == -d || k != d && at(k-1) < at(k+1) {
			pk = k + 1
		} else {
			pk = k - 1
		}
		px := at(pk)
		py := px - pk
		for x > px && y > py {
			x--
			y--
			rev = append(rev, diffOp{opKeep, x, y})
		}
		if x == px {
			y--
			rev = append(rev, diffOp{opAdd, x, y})
		} else {
			x--
			rev = append(rev, diffOp{opDel, x, y})
		}
	}
	for x > 0 && y > 0 {
		x--
		y--
		rev = append(rev, diffOp{opKeep, x, y})
	}
	for i, j := 0, len(rev)-1; i < j; i, j = i+1, j-1 {
		rev[i], rev[j] = rev[j], rev[i]
	}
	return rev
}

// hunksOf groups an edit script's changes into hunks, each [start, end) over
// ops with diffContext kept lines either side, merging two changes whose
// context would touch.
func hunksOf(ops []diffOp) [][2]int {
	var hunks [][2]int
	for i := 0; i < len(ops); {
		if ops[i].kind == opKeep {
			i++
			continue
		}
		start := max(0, i-diffContext)
		end := i
		for end < len(ops) {
			if ops[end].kind != opKeep {
				end++
				continue
			}
			run := end
			for run < len(ops) && ops[run].kind == opKeep {
				run++
			}
			if run == len(ops) || run-end > 2*diffContext {
				end = min(len(ops), end+diffContext)
				break
			}
			end = run
		}
		if n := len(hunks); n > 0 && start <= hunks[n-1][1] {
			hunks[n-1][1] = end
		} else {
			hunks = append(hunks, [2]int{start, end})
		}
		i = end
	}
	return hunks
}

// writeHunk writes one hunk's header and lines.
func writeHunk(w *strings.Builder, ops []diffOp, a, b []string) {
	aStart, bStart := -1, -1
	aLen, bLen := 0, 0
	for _, op := range ops {
		switch op.kind {
		case opKeep:
			aStart, bStart = firstOf(aStart, op.a), firstOf(bStart, op.b)
			aLen++
			bLen++
		case opDel:
			aStart = firstOf(aStart, op.a)
			aLen++
		case opAdd:
			bStart = firstOf(bStart, op.b)
			bLen++
		}
	}
	w.WriteString("@@ -" + hunkRange(aStart, aLen, ops, true) + " +" + hunkRange(bStart, bLen, ops, false) + " @@\n")
	for _, op := range ops {
		switch op.kind {
		case opKeep:
			writeLine(w, ' ', a[op.a])
		case opDel:
			writeLine(w, '-', a[op.a])
		case opAdd:
			writeLine(w, '+', b[op.b])
		}
	}
}

func firstOf(cur, i int) int {
	if cur < 0 {
		return i
	}
	return cur
}

// hunkRange is a hunk side's range as the unified format writes it: its first
// line, counted from 1, and its length where that is not 1. A side with no
// line names the line before the hunk, 0 at the file's head.
func hunkRange(start, n int, ops []diffOp, old bool) string {
	if n == 0 {
		// The line before: the side's index at the hunk's first op.
		at := 0
		if len(ops) > 0 {
			if old {
				at = ops[0].a
			} else {
				at = ops[0].b
			}
		}
		return strconv.Itoa(at) + ",0"
	}
	if n == 1 {
		return strconv.Itoa(start + 1)
	}
	return strconv.Itoa(start+1) + "," + strconv.Itoa(n)
}

// writeLine writes one line under its mark, marking a line with no final
// newline as the unified format does.
func writeLine(w *strings.Builder, mark byte, line string) {
	w.WriteByte(mark)
	w.WriteString(line)
	if !strings.HasSuffix(line, "\n") {
		w.WriteString("\n\\ No newline at end of file\n")
	}
}
