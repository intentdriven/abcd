package scanner

import (
	"strings"
	"testing"
)

// iss-2610090821491948: the percent view and the JSON layers each decoded the
// raw line, and neither ever decoded the other's output. A JSON escape of '%'
// (%) or a percent encoding of a backslash (%5C) stacks the two so that
// neither view alone reads the value.
func TestStackedJSONAndPercentEncodingsAreDecoded(t *testing.T) {
	token := syntheticPAT(2610090821491948)
	tail := token[1:] // the token's first byte is 'g', 0x67
	cases := []struct {
		name, line, live, kind string
	}{
		{"json then percent", "token=" + jsonU("0025") + "67" + tail, tail, "token:github_pat"},
		{"percent then json", "token=%5Cu0067" + tail, tail, "token:github_pat"},
		{"json then percent home", "see " + jsonU("0025") + "2Fhome" + jsonU("0025") + "2Falice in the log",
			jsonU("0025") + "2Fhome" + jsonU("0025") + "2Falice", kindHomeSelf},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			text := tc.line + "\n"
			findings := ScanText(text, testIdent(), DefaultPatterns(), DefaultIdentitySeverities(), "memory")
			if !hasKind(findings, tc.kind) {
				t.Fatalf("no %s finding for the stacked spelling %q: %+v", tc.kind, tc.line, findings)
			}
			redacted, _ := Redact(text, findings)
			if strings.Contains(redacted, tc.live) {
				t.Errorf("the encoded value survived redaction:\n%s", redacted)
			}
			if residual := ScanText(redacted, testIdent(), DefaultPatterns(), DefaultIdentitySeverities(), "memory"); hasKind(residual, tc.kind) {
				t.Errorf("still detectable after redaction: %+v", residual)
			}
		})
	}
}

// jsonU spells a JSON \uXXXX escape, built at runtime so the source holds no
// escape an editor or a tool might decode on the way to disk.
func jsonU(hex string) string { return string(rune(92)) + "u" + hex }

// The controls: the plaintext token, a single percent escape of its first
// byte, and a single JSON escape of it are still found.
func TestSingleEncodingsOfATokenStillHardFail(t *testing.T) {
	token := syntheticPAT(2610090821491949)
	for _, line := range []string{
		"token=" + token,
		"token=%67" + token[1:],
		"token=" + jsonU("0067") + token[1:],
	} {
		findings := ScanText(line+"\n", testIdent(), DefaultPatterns(), DefaultIdentitySeverities(), "memory")
		if !hasKind(findings, "token:github_pat") {
			t.Errorf("no finding for %q: %+v", line, findings)
		}
	}
}

// iss-2610090821491948, review round: composing the two decoders once each
// way left a third alternation unread. A percent escape of the backslash of a
// JSON escape of the percent sign (percent, then JSON, then percent) and its
// mirror (JSON, then percent, then JSON) each need three decodes in turn.
func TestThreeStepStackedEncodingsAreDecoded(t *testing.T) {
	token := syntheticPAT(2610090821491950)
	tail := token[1:] // the token's first byte is 'g', 0x67
	for name, line := range map[string]string{
		"percent json percent":      "token=%5Cu002567" + tail,
		"json percent json":         "token=" + jsonU("0025") + "5Cu0067" + tail,
		"json percent percent":      "token=" + jsonU("0025") + "2567" + tail,
		"percent json json":         "token=%5C%5Cu0067" + tail,
		"percent percent json":      "token=%255Cu0067" + tail,
		"percent json percent json": "token=%5Cu00255Cu0067" + tail,
	} {
		t.Run(name, func(t *testing.T) {
			text := line + "\n"
			findings := ScanText(text, testIdent(), DefaultPatterns(), DefaultIdentitySeverities(), "memory")
			if !hasKind(findings, "token:github_pat") {
				t.Fatalf("no finding for the stacked spelling %q: %+v", line, findings)
			}
			redacted, _ := Redact(text, findings)
			if strings.Contains(redacted, tail) {
				t.Errorf("the encoded value survived redaction:\n%s", redacted)
			}
		})
	}
}

// The bound: five alternating decodes is past the four-layer cap, and stays
// unread, the same bounded-work trade the layer caps make.
func TestStackedEncodingsPastTheLayerCapStayRaw(t *testing.T) {
	token := syntheticPAT(2610090821491951)
	line := "token=%5Cu00255Cu002567" + token[1:] // percent, JSON, percent, JSON, percent
	if f := ScanText(line+"\n", testIdent(), DefaultPatterns(), DefaultIdentitySeverities(), "memory"); hasKind(f, "token:github_pat") {
		t.Fatalf("a fifth layer was decoded; the cap is four: %+v", f)
	}
}
