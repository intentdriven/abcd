package scanner

import (
	"strings"
	"testing"
)

// goSourceLANHostReported runs the net_lan_hostname pattern over one line the
// way the privacy lint rule reads a line of a tracked .go file: the pattern's
// own Skip and SkipAt, then GoSourceSkip. It reports whether any match
// survives, which is what the rule would flag.
func goSourceLANHostReported(t *testing.T, line string) bool {
	t.Helper()
	for _, p := range NetworkPatterns() {
		if p.Kind != kindNetLANHost {
			continue
		}
		for _, loc := range p.Re.FindAllStringIndex(line, -1) {
			if p.Skip != nil && p.Skip(line[loc[0]:loc[1]]) {
				continue
			}
			if p.SkipAt != nil && p.SkipAt(line, loc[0], loc[1]) {
				continue
			}
			if GoSourceSkip(p.Kind, line, loc[0], loc[1]) {
				continue
			}
			return true
		}
		return false
	}
	t.Fatal("no net_lan_hostname pattern in NetworkPatterns")
	return false
}

// TestCallArgumentSelectorIsNotAHost pins iss-2610020840196926 on the Go-only
// path: in a line of Go source, a selector passed as a call's argument — the
// field `local` of a value, closing the argument list or separating it from
// the next — is code, not a LAN host. selectorExpression read a selector only
// where '(' '[' '=' or '{' followed it, so abcd lint reported the first line
// below in this repository's own tree. GoSourceSkip is what the privacy rule
// consults for a .go file; no path-less scanner entry point consults it.
//
// The selectors here are literals on purpose: they are what the exemption
// spares in a .go file, so this file is itself part of the proof that the tree
// scans clean.
func TestCallArgumentSelectorIsNotAHost(t *testing.T) {
	for _, line := range []string{
		"\tcase deferredOK && launch.CoreGreater(deferred, anchor.local):", // internal/core/capture/eligible.go
		"\treturn f(anchor.local)",
		"f(x.local, y)",
		"f(a, x.local, b)",
		"f(a,x.lan)",
		"use(cfg.box.local)",
		"fmt.Println(deferred, cfg.anchor.local)",
		"if ok := check(g(h), anchor.local); ok {",
		"want(t, got.local, exp.local)",
	} {
		if goSourceLANHostReported(t, line) {
			t.Errorf("Go selector passed as a call argument flagged as a LAN host: %q", line)
		}
	}
}

// TestGoSourceSkipIsLANHostOnly pins the kind gate: the exemption is for the
// net_lan_hostname pattern alone, so a caller handing GoSourceSkip another
// kind's match at the same span is told nothing is spared.
func TestGoSourceSkipIsLANHostOnly(t *testing.T) {
	line := "f(" + host("anchor", "local") + ")"
	start, end := 2, len(line)-1
	if !GoSourceSkip(kindNetLANHost, line, start, end) {
		t.Fatalf("the Go selector shape was not spared for %s: %q", kindNetLANHost, line)
	}
	if GoSourceSkip(kindNetDeviceHost, line, start, end) {
		t.Errorf("GoSourceSkip spared a %s match", kindNetDeviceHost)
	}
}

// TestHostClosingAParenthesisStillFlags is the other half: even in a .go
// file, the exemption reaches a code-shaped selector in a call's argument list
// and nothing else. A host in prose parentheses, in a list, in a URL, in a
// config value, quoted, hyphenated, in capitals, behind a Go comment marker or
// behind a space before the parenthesis still reports. The hosts are
// assembled, as in network_test.go, so this file carries no literal one.
func TestHostClosingAParenthesisStillFlags(t *testing.T) {
	printer := host("printer", "local")
	nas := host("nas", "lan")
	for _, line := range []string{
		// Prose: a markdown sentence's parenthesis and a list's comma.
		"The printer (" + printer + ") is on the LAN.",
		"the hosts (" + printer + ", " + nas + ") answer",
		"reach it at " + printer + ", then mount " + nas + ".",
		"see the box at " + printer + "), then",
		"ping (" + printer + ")",
		// URLs and a markdown link target.
		"see http://" + printer + ")",
		"[the printer](http://" + printer + ")",
		"[the printer](" + printer + ")",
		// Config values.
		"hosts = [" + printer + ", " + nas + "]",
		"host: " + printer + ",",
		"HOSTS=(" + printer + ")",
		// Inside a call's parentheses but not an argument of its own: the tail of
		// a path or of a user@host.
		"mount(/srv/" + printer + ")",
		"ssh(deploy@" + printer + ", now)",
		// Quoted, in Go source.
		`conn, err := net.Dial("tcp", "` + printer + `")`,
		// A call-shaped mention behind a Go comment marker is prose about a host.
		"// dial(" + printer + ") first",
		"\tx := f(y) // then dial(" + printer + ")",
		"/* dial(" + printer + ") */",
		// Not a Go selector: a hyphen, or a label that starts with a digit.
		"dial(" + host(dash("corp", "nas"), "local") + ")",
		"dial(" + host("3d", "printer", "local") + ")",
		// The exemption reads lower-case labels only: an upper-case suffix is a
		// host written in capitals, an upper-case label before the suffix is too
		// (every label is checked, not only the field), and an exported field
		// (mixed case) is mixedCaseSelector's to judge.
		"dial(" + host("printer", "LOCAL") + ")",
		"ssh(" + host("NAS", "local") + ")",
		"fmt.Println(deferred, " + host("cfg", "Anchor", "local") + ")",
	} {
		if !goSourceLANHostReported(t, line) {
			t.Errorf("LAN host closing a parenthesis or list was not reported: %q", line)
		}
	}
}

// TestCallArgumentSelectorReadsABoundedPrefix pins the direction the bound
// fails in: a selector whose line prefix runs past what the helper reads is
// reported, because the helper cannot see a comment marker or the call's
// opening parenthesis beyond it — an over-report on a line no Go source
// carries, never a leak.
func TestCallArgumentSelectorReadsABoundedPrefix(t *testing.T) {
	// Assembled: the long line is reported by design, so its literal would be
	// a finding in this file too.
	sel := host("anchor", "local")
	long := "f(" + strings.Repeat("a, ", maxCallArgumentPrefix/3+1) + sel + ")"
	if !goSourceLANHostReported(t, long) {
		t.Errorf("a selector behind a prefix longer than maxCallArgumentPrefix (%d) was spared", maxCallArgumentPrefix)
	}
	short := "f(" + strings.Repeat("a, ", 4) + sel + ")"
	if goSourceLANHostReported(t, short) {
		t.Errorf("a selector behind a short prefix was reported")
	}
}

// TestPathLessScannerStillRedactsCallShapedHosts pins the narrowing that
// sec-scannerSel asked for (iss-2610020840196926): a path-less write path — the
// history transcript store, memory, capture, intent, decide, reflect and the
// release scan all call Scanner.ScanText with a logical name, never a Go file —
// cannot tell a Go selector from a real host written as a call's argument, so
// it keeps the base behaviour and reports and redacts every shape below. The
// call-argument exemption belongs to a caller that knows the file is Go.
func TestPathLessScannerStillRedactsCallShapedHosts(t *testing.T) {
	sc, err := New(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	ex := host("example", "local")
	nas := host("nas", "local")
	pi := host("raspberrypi", "local")
	hub := host("hub", "lan")
	fritz := host("fritz", "box")
	some := host("somehost", "local")
	macbook := host("alicesmacbook", "local")
	upper := host("NAS", "local")
	anchor := host("anchor", "local")
	for _, c := range []struct {
		name, line string
		hosts      []string
	}{
		{"fmt-wrapped dial error", "dial(" + ex + "): connection refused", []string{ex}},
		{"getaddrinfo error", "error: getaddrinfo(" + ex + ") failed", []string{ex}},
		{"mermaid node", "    nas(" + ex + ") --> router", []string{ex}},
		{"mermaid chain", "    pi(" + pi + ") --> hub(" + hub + ")", []string{pi, hub}},
		{"prose without a space", "the NAS(" + nas + ") holds the backups", []string{nas}},
		{"router call shape", "router(" + fritz + ")", []string{fritz}},
		{"ssh call shape", "ssh(" + some + ")", []string{some}},
		{"ssh call shape, a person's machine", "ssh(" + macbook + ")", []string{macbook}},
		{"ssh call shape, upper-case label", "ssh(" + upper + ")", []string{upper}},
		{"json transcript record", `{"type":"tool_result","content":"dial(` + ex + `): connection refused"}`, []string{ex}},
		{"json transcript record, mermaid", `{"type":"tool_result","content":"nas(` + ex + `) --> router"}`, []string{ex}},
		// The eligible.go shape itself: without a file type the store cannot
		// know it is Go, so it is redacted like any other call-shaped host.
		{"go selector seen without a file type", "\tcase deferredOK && launch.CoreGreater(deferred, " + anchor + "):", []string{anchor}},
	} {
		t.Run(c.name, func(t *testing.T) {
			got := sc.ScanText(c.line, "transcript")
			if !hasKind(got, kindNetLANHost) {
				t.Fatalf("path-less scan did not report the LAN host in %q: %+v", c.line, got)
			}
			out, _ := Redact(c.line, got)
			for _, h := range c.hosts {
				if strings.Contains(out, h) {
					t.Errorf("host %q survived redaction: %q", h, out)
				}
			}
		})
	}
}
