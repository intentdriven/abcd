package scanner

import (
	"strings"
	"testing"
)

// network_junction_test.go — iss-195: every IPv4 and IPv6 match is
// variable-length, so each one entered stolenJunctions' backward search, and
// the combined junction probe offered every hex run inside a colon-hex
// address as the start of another address. That is both a cost (a dense
// colon-hex line handed the probes tens of thousands of bytes per byte of
// line) and a correctness defect: one compressed IPv6 address was reported
// as itself and as each of its own suffixes, a duplicate hard_fail per suffix.
// A network token does not abut another network token with no separator, so
// behind a network match only a SECRET token is sought; a secret that a
// network match over-ran is still recovered.

func networkDenseLines() map[string]string {
	var fp, mac strings.Builder
	for i := 0; i < 50; i++ {
		fp.WriteString("ab:cd:ef:01:23:45:67:89:ab:cd:ef:01:23:45:67:89 ")
		mac.WriteString("00:00:5e:00:53:0a ")
	}
	return map[string]string{"fingerprints": fp.String(), "macs": mac.String()}
}

func matchesOf(line string) []patMatch {
	pats := DefaultPatterns()
	probes := make([]matcher, len(pats))
	for i, cp := range pats {
		probes[i] = adjacencyProbe(cp.Re)
	}
	return scanAllPatterns(pats, probes, newJunctionSet(pats), line)
}

func TestOneCompressedIPv6IsOneToken(t *testing.T) {
	pats := DefaultPatterns()
	line := `"2001:db8:a9fe::",`
	var got []string
	for _, m := range matchesOf(line) {
		got = append(got, pats[m.patIdx].Name+"="+line[m.start:m.end])
	}
	if len(got) != 1 || got[0] != "net_ipv6=2001:db8:a9fe::" {
		t.Errorf("one compressed IPv6 address produced %d tokens, want exactly the address: %v", len(got), got)
	}
}

func TestASecretAbuttingANetworkTokenIsStillRecovered(t *testing.T) {
	pats := DefaultPatterns()
	key := "AKIA" + strings.Repeat("Q7", 8)
	line := "fe80::1" + key
	for _, m := range matchesOf(line) {
		if pats[m.patIdx].Name == "aws_access_key" && line[m.start:m.end] == key {
			return
		}
	}
	t.Errorf("the key the IPv6 match over-ran was not recovered: %+v", matchesOf(line))
}

// The work bound: on a line of nothing but network tokens the probes are handed
// a bounded multiple of the line. Before the fix the fingerprints line handed
// them over 50,000 bytes per byte of line.
func TestNetworkDenseLineProbeWorkIsBounded(t *testing.T) {
	const bar = 2000
	for name, line := range networkDenseLines() {
		work := probeWork(line)
		t.Logf("%s: %d-byte line, %d bytes probed (%.0fx)", name, len(line), work, float64(work)/float64(len(line)))
		if work > bar*len(line) {
			t.Errorf("%s: the probes were handed %d bytes for a %d-byte line (%dx), want at most %dx",
				name, work, len(line), work/len(line), bar)
		}
	}
}

func BenchmarkScanTextNetworkDense(b *testing.B) {
	lines := networkDenseLines()
	text := strings.Repeat(lines["fingerprints"]+"\n", 10)
	pats := DefaultPatterns()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ScanText(text, Identity{}, pats, nil, "b")
	}
}
