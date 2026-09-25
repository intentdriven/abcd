package scanner

// meter.go — the per-line scan's cost seam (iss-2609240203462704).
//
// The package's count-based cost guards used to reach scanAllPatterns alone:
// probeWork tallies the bytes the adjacency probes are handed, and nothing
// counted the rest of a line's scan — the Skip and SkipAt callbacks, the
// identity matchers over the raw and the decoded line, the percent-decode
// passes, or each pattern's own FindAllStringIndex. A per-match helper that
// walked the whole line therefore passed every guard, and one did: the
// generic-account position check folded the line prefix for every match, so
// a line dense in a generic login cost the square of its length.
//
// scanMeter is where each of those stages says what it read. A stage charges
// the bytes it hands a regexp, and a helper that reads past its match — a
// walk back to a token start, a search forward for a closing delimiter, a scan
// of a span list — charges the bytes or entries it actually visited. It is a
// no-op in production; TestScanLineWorkIsLinear swaps in a tally and asserts
// that quadrupling a line at most quadruples every stage's charge, which is
// the cost CLASS the guards protect, measured the same on an idle machine and
// a loaded one.
//
// The swap is test-only and single-goroutine: the guard reading it is a
// top-level test that skips under -race (a deterministic count gains nothing
// there), and this package declares no parallel tests, so no scan runs while
// the tally is installed except the one the guard makes.

// The stages a line's scan charges, one per place the per-line work can grow.
const (
	// stagePattern is each pattern's own FindAllStringIndex over the line.
	stagePattern = "pattern"
	// stageAdjacency is every string the adjacency and junction probes are
	// handed (the quantity probeWork has always counted).
	stageAdjacency = "adjacency"
	// stageSkip is each Skip callback, charged the match it is handed.
	stageSkip = "skip"
	// stageSkipAt is what each SkipAt callback reads of the line around its
	// match, charged by the context helpers themselves.
	stageSkipAt = "skip_at"
	// stageIdentity is each identity matcher's pass over the line and what the
	// per-match position checks read around each match.
	stageIdentity = "identity"
	// stagePercent is each percent-decode pass over the line.
	stagePercent = "percent"
)

// costMeter records per-stage scan work. The zero value charges nothing.
type costMeter struct {
	tally func(stage string, n int)
}

// charge records n units of work against stage when a tally is installed.
func (m *costMeter) charge(stage string, n int) {
	if m.tally != nil {
		m.tally(stage, n)
	}
}

// scanMeter is the package's one meter; see the file comment.
var scanMeter costMeter
