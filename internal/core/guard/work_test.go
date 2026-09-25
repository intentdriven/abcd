package guard

import "testing"

// linearWorkBar is the most the guard's counted work may grow when its input
// grows fourfold. Linear work grows 4x; the bar leaves 1.5x of room over that,
// and each regression the bounds exist for grew 16x or more (a per-start walk
// of the whole line, a per-start re-tokenize of a payload, a per-segment budget
// over an unbounded segment count, a per-brace look-ahead to the end of the
// line), so the bar separates the two classes with room on both sides.
const linearWorkBar = 6.0

// workPerByteBar is the absolute bound beside the growth bar: at most this many
// units of work per byte of input. A ratio sees a cost CLASS but not a
// constant, and the speculation bounds overlap, so dropping any one of them
// leaves the cost linear with a constant tens of times larger — the 14.2s
// regression was exactly that, 64 starts each walking the whole line. The
// bounded shapes measure one to five units per byte; dropping one bound
// measures well over a hundred.
const workPerByteBar = 20.0

// checkWork runs one check over line against the bundled registry and returns
// the work the guard counted doing it (tally in work.go): bytes tokenized, bytes
// the brace look-ahead and the closing scans read, and tokens each pattern match
// walked. It reads past Check's length cap (maxCommandBytes), because the cost
// CLASS is a property of the reading, and a cap in front of a quadratic reading
// only hides it until the cap moves.
func checkWork(t *testing.T, line string) (Decision, int) {
	t.Helper()
	n := 0
	workTally = &n
	defer func() { workTally = nil }()
	d, err := Defaults().check(line)
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	return d, n
}

// assertWorkGrowth is the cost guard's shape in place of a stopwatch
// (iss-2609240046582859): build the shape at base and at four times it, count
// the guard's work at each size, and fail when it grows faster than linear. A
// wall-clock ceiling measures the machine running the test, and a loaded gate
// run trips it with nothing wrong; what the bound protects is a cost CLASS, and
// a class is a count, the same on an idle machine and a loaded one.
func assertWorkGrowth(t *testing.T, build func(int) string, base int, why string) (small, large Decision) {
	t.Helper()
	if raceEnabled {
		t.Skip("a deterministic count gains nothing under -race; the uninstrumented run asserts it")
	}
	sLine, lLine := build(base), build(4*base)
	if len(lLine) < 3*len(sLine) {
		t.Fatalf("the shape does not scale with its parameter: %d bytes at base, %d at four times it", len(sLine), len(lLine))
	}
	small, lo := checkWork(t, sLine)
	large, hi := checkWork(t, lLine)
	if lo == 0 {
		t.Fatalf("the %d-byte shape counted no work; it pins nothing", len(sLine))
	}
	growth := float64(hi) / float64(lo)
	t.Logf("%d -> %d bytes of input; %d -> %d units of work; growth %.2fx (bar %.1fx)", len(sLine), len(lLine), lo, hi, growth, linearWorkBar)
	if perByte := float64(hi) / float64(len(lLine)); perByte > workPerByteBar {
		t.Errorf("the guard did %.1f units of work per byte of a %d-byte input, want at most %.0f: %s",
			perByte, len(lLine), workPerByteBar, why)
	}
	if growth > linearWorkBar {
		t.Errorf("quadrupling the input multiplied the guard's work by %.2fx (%d -> %d), want at most %.1fx: %s",
			growth, lo, hi, linearWorkBar, why)
	}
	return small, large
}
