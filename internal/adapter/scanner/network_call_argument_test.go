package scanner

import (
	"strings"
	"testing"
)

// TestCallArgumentSelectorIsNotAHost pins iss-2610020840196926: a Go selector
// passed as a call's argument — the field `local` of a value, closing the
// argument list or separating it from the next — is code, not a LAN host.
// selectorExpression read a selector only where '(' '[' '=' or '{' followed
// it, so abcd lint reported the first line below in this repository's own
// tree, and Stage-1 redaction would have rewritten it in a stored transcript.
//
// The selectors here are literals on purpose: they are what the exemption
// spares, so this file is itself part of the proof that the tree scans clean.
func TestCallArgumentSelectorIsNotAHost(t *testing.T) {
	for _, line := range []string{
		"\tcase deferredOK && launch.CoreGreater(deferred, anchor.local):", // internal/core/capture/eligible.go
		"\treturn f(anchor.local)",
		"f(x.local, y)",
		"f(a, x.local, b)",
		"f(a,x.lan)",
		"use(cfg.box.local)",
		"fmt.Println(deferred, cfg.Anchor.local)",
		"if ok := check(g(h), anchor.local); ok {",
		"want(t, got.local, exp.local)",
	} {
		if got := scanNet(line); hasKind(got, kindNetLANHost) {
			t.Errorf("Go selector passed as a call argument flagged as a LAN host: %q: %+v", line, got)
		}
	}
}

// TestHostClosingAParenthesisStillFlags is the other half: the widened
// exemption reaches a code-shaped selector in a call's argument list and
// nothing else. A host in prose parentheses, in a list, in a URL, in a config
// value, quoted, hyphenated, behind a comment marker or behind a space before
// the parenthesis still reports. The hosts are assembled, as in network_test.go,
// so this file carries no literal one.
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
		// A call-shaped mention behind a comment marker is prose about a host.
		"// dial(" + printer + ") first",
		"\tx := f(y) // then dial(" + printer + ")",
		"/* dial(" + printer + ") */",
		// Not a Go selector: a hyphen, or a label that starts with a digit.
		"dial(" + host(dash("corp", "nas"), "local") + ")",
		"dial(" + host("3d", "printer", "local") + ")",
		// The exemption reads a lower-case field only: an upper-case suffix is a
		// host written in capitals, and an exported field (mixed case) is
		// mixedCaseSelector's to judge.
		"dial(" + host("printer", "LOCAL") + ")",
	} {
		if got := scanNet(line); !hasKind(got, kindNetLANHost) {
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
	if got := scanNet(long); !hasKind(got, kindNetLANHost) {
		t.Errorf("a selector behind a prefix longer than maxCallArgumentPrefix (%d) was spared", maxCallArgumentPrefix)
	}
	short := "f(" + strings.Repeat("a, ", 4) + sel + ")"
	if got := scanNet(short); hasKind(got, kindNetLANHost) {
		t.Errorf("a selector behind a short prefix was reported: %+v", got)
	}
}
