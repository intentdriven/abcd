package scanner

import (
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/testsecret"
)

// Every sample in this file is assembled at runtime (testsecret), never
// written as a literal: a committed key-shaped string trips the full-history
// secret scan and the forge's push protection on the commit that adds it
// (principle secret-shaped-fixtures-at-runtime).

// legacyOpenAIMarker is the base64 of "OpenAI" a legacy OpenAI key carries
// between its two 20-character halves, split so the source holds no run that
// reads as one.
const legacyOpenAIMarker = "T3Blbk" + "FJ"

func openRouterKey(seed uint64) string {
	return "sk-or-v1-" + testsecret.SyntheticHex(seed, 64)
}

func legacyOpenAIKey(seed uint64) string {
	return "sk-" + testsecret.Synthetic(seed, 20) + legacyOpenAIMarker + testsecret.Synthetic(seed+1, 20)
}

func plainSKKey(seed uint64, n int) string {
	return "sk-" + testsecret.Synthetic(seed, n)
}

// tokenFindings are the findings of a token: kind, the class a key belongs to.
func tokenFindings(fs []Finding) []Finding {
	var out []Finding
	for _, f := range fs {
		if IsTokenKind(f.Kind) {
			out = append(out, f)
		}
	}
	return out
}

// TestSKKeysReportedOnceWithTheMostSpecificKind (iss-2610040202190813): an
// OpenRouter key, a legacy OpenAI key and a plain sk- key are each reported
// exactly once, by the kind that names it most specifically, at that kind's
// severity; the sk- keys the set already knew keep their one report and are
// never doubled by the plain rule.
func TestSKKeysReportedOnceWithTheMostSpecificKind(t *testing.T) {
	cases := []struct {
		name, key, kind string
		sev             Severity
	}{
		{"openrouter", openRouterKey(1), "token:openrouter", SeverityHardFail},
		{"legacy openai", legacyOpenAIKey(2), "token:openai_legacy", SeverityHardFail},
		{"plain at the minimum", plainSKKey(4, 32), "token:sk_generic", SeverityWarn},
		{"plain, long", plainSKKey(5, 48), "token:sk_generic", SeverityWarn},
		{"plain, hex", "sk-" + testsecret.SyntheticHex(6, 32), "token:sk_generic", SeverityWarn},
		{"anthropic", "sk-ant-" + testsecret.Synthetic(7, 48), "token:anthropic", SeverityHardFail},
		{"openai project", "sk-proj-" + testsecret.Synthetic(8, 48), "token:openai_project", SeverityHardFail},
		{"openai service account", "sk-svcacct-" + testsecret.Synthetic(9, 48), "token:openai_svcacct", SeverityHardFail},
		// iss-2610040452174484: the third prefix gitleaks' openai-api-key rule names.
		{"openai admin", "sk-admin-" + testsecret.Synthetic(3, 48), "token:openai_admin", SeverityHardFail},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			for _, line := range []string{c.key, `OPENAI_API_KEY="` + c.key + `"`, "key " + c.key + " here"} {
				got := tokenFindings(scanLine(line))
				if len(got) != 1 {
					t.Fatalf("%s: want exactly one token finding, got %d: %+v", c.name, len(got), got)
				}
				if got[0].Kind != c.kind || got[0].Severity != c.sev {
					t.Fatalf("%s: reported as %s/%s, want %s/%s", c.name, got[0].Kind, got[0].Severity, c.kind, c.sev)
				}
				if got[0].Matched != c.key {
					t.Fatalf("%s: matched %d bytes of a %d-byte key", c.name, len(got[0].Matched), len(c.key))
				}
			}
		})
	}
}

// TestSKKeyNearMissesStayQuiet: what is not one of the shapes is not taken
// for it — a model id opening on sk-, a plain tail one short of the minimum,
// an OpenRouter body in upper-case hex, and a legacy key whose first half is
// one short.
func TestSKKeyNearMissesStayQuiet(t *testing.T) {
	quiet := map[string]string{
		"a model id":             "sk-tuned/7b",
		"a model id with a tail": "sk-tuned-" + testsecret.Synthetic(10, 20) + "/7b",
		"a 31-character tail":    plainSKKey(11, 31),
		"upper-case OpenRouter":  "sk-or-v1-" + strings.ToUpper(testsecret.SyntheticHex(12, 64)),
		"a short OpenRouter":     "sk-or-v1-" + testsecret.SyntheticHex(13, 63),
	}
	for name, line := range quiet {
		if got := tokenFindings(scanLine(line)); len(got) != 0 {
			t.Errorf("%s (%d bytes) taken for a key: %+v", name, len(line), got)
		}
	}
	short := "sk-" + testsecret.Synthetic(14, 19) + legacyOpenAIMarker + testsecret.Synthetic(15, 20)
	if hasKind(scanLine(short), "token:openai_legacy") {
		t.Errorf("a legacy shape with a 19-character first half taken for a legacy key")
	}
}

// TestSKKeysSurviveNoSuffix: each new key is still caught with an underscore
// or a letter glued after it, the trailing-boundary gap the set's other
// token rules close (trailing_boundary_test.go).
func TestSKKeysSurviveNoSuffix(t *testing.T) {
	for kind, key := range map[string]string{
		"token:openrouter":    openRouterKey(20),
		"token:openai_legacy": legacyOpenAIKey(21),
		"token:sk_generic":    plainSKKey(23, 40),
		"token:openai_admin":  "sk-admin-" + testsecret.Synthetic(24, 39) + "-",
	} {
		for _, suffix := range []string{"_old", "_", "-x"} {
			if !hasKind(scanLine(key+suffix), kind) {
				t.Errorf("%s survived the suffix %q", kind, suffix)
			}
		}
	}
}

// TestPlainSKKeyIsRedactedThoughItOnlyWarns: the plain rule warns rather
// than hard-fails, so a hash-like committed string cannot fail a gate on it,
// yet every store-before-commit redactor masks every finding whatever its
// severity, so a pasted plain key never reaches a record raw.
func TestPlainSKKeyIsRedactedThoughItOnlyWarns(t *testing.T) {
	key := plainSKKey(30, 48)
	line := "export OPENAI_API_KEY=" + key
	fs := scanLine(line)
	if !hasKind(fs, "token:sk_generic") {
		t.Fatalf("plain key not found: %+v", fs)
	}
	out, n := Redact(line, fs)
	if n == 0 || strings.Contains(out, key) || strings.Contains(out, key[3:20]) {
		t.Fatalf("plain key survived redaction: %q", out)
	}
	if got := tokenFindings(scanLine(out)); len(got) != 0 {
		t.Fatalf("the redacted line still carries a key: %+v", got)
	}
}

// TestLegacyShapedRunIsMaskedToItsEnd: a run that opens like a legacy OpenAI
// key and goes on past its 48 characters is not skipped by the plain rule,
// whose span is what masks the bytes after the legacy rule's 51.
func TestLegacyShapedRunIsMaskedToItsEnd(t *testing.T) {
	tail := testsecret.Synthetic(41, 8)
	line := "key " + legacyOpenAIKey(40) + tail + " end"
	out, _ := Redact(line, scanLine(line))
	if strings.Contains(out, tail[:6]) {
		t.Fatalf("the run's tail survived redaction: %q", out)
	}
}

// TestGluedPlainSKKeyIsFoundAndMasked — the glued half of the security review
// of iss-2610040202190813. The glued sweep took hard_fail rules only, so a plain
// sk- key, which only warns, was invisible behind a word character: "my_",
// "X" and "9" glued before one each gave zero findings, and every redactor let
// it through raw. A token is a secret whatever its severity (IsTokenKind), so
// the sweep reads the plain rule too, and the finding keeps its warn severity.
func TestGluedPlainSKKeyIsFoundAndMasked(t *testing.T) {
	key := plainSKKey(50, 40)
	for _, glue := range []string{"my_", "X", "9"} {
		line := "set " + glue + key + " now"
		fs := scanLine(line)
		var got []Finding
		for _, f := range fs {
			if f.Kind == "token:sk_generic" {
				got = append(got, f)
			}
		}
		if len(got) != 1 {
			t.Fatalf("%q glued: want one token:sk_generic finding, got %+v", glue, fs)
		}
		if got[0].Severity != SeverityWarn || got[0].Matched != key {
			t.Errorf("%q glued: reported as %s over %d bytes, want warn over the %d-byte key", glue, got[0].Severity, len(got[0].Matched), len(key))
		}
		out, _ := Redact(line, fs)
		if strings.Contains(out, key[3:12]) {
			t.Errorf("%q glued: the key survived redaction: %q", glue, out)
		}
		if !strings.HasPrefix(out, "set "+glue) {
			t.Errorf("%q glued: redaction lost the text before the key: %q", glue, out)
		}
	}
}

// TestGluedPlainSKKeyNeverLeavesRaw: the two doors that hand redacted text
// back out — a refusal echoed to the terminal and an artefact a routine posts —
// return a glued plain key masked. ScrubOutbound returned it intact with a nil
// error before the sweep read the plain rule.
func TestGluedPlainSKKeyNeverLeavesRaw(t *testing.T) {
	key := plainSKKey(51, 44)
	body := key[3:12]
	repo := t.TempDir()
	if got := RedactRefusal(repo, `json: unknown field "notes_`+key+`"`); strings.Contains(got, body) {
		t.Errorf("RedactRefusal echoed the glued plain key: %q", got)
	}
	got, _, err := ScrubOutbound(repo, "set my_"+key+" before the run\n", "pr-body")
	if err != nil {
		t.Fatalf("ScrubOutbound refused text it can mask: %v", err)
	}
	if strings.Contains(got, body) {
		t.Errorf("ScrubOutbound handed back the glued plain key: %q", got)
	}
}

// TestEveryWriteBackstopTreatsAWarnTokenAsASecret: the stage-two backstops a
// committed store, a returned refusal and an outbound artefact share refuse a
// surviving token whatever its severity. A warn span of no secret class still
// passes each of them, so the widening is the token class and nothing else.
func TestEveryWriteBackstopTreatsAWarnTokenAsASecret(t *testing.T) {
	warnToken := Finding{Kind: "token:sk_generic", Severity: SeverityWarn}
	warnOther := Finding{Kind: "generic:warn_only", Severity: SeverityWarn}
	if !IsTokenKind(warnToken.Kind) || IsTokenKind(warnOther.Kind) || IsTokenKind(kindNetLANHost) {
		t.Fatalf("IsTokenKind misjudges the token class")
	}
	if got := BlockingResidual([]Finding{warnToken, warnOther}); len(got) != 1 || got[0].Kind != warnToken.Kind {
		t.Errorf("BlockingResidual kept %+v, want the warn token alone", got)
	}
	if !hasSecret([]Finding{warnToken}) || hasSecret([]Finding{warnOther}) {
		t.Errorf("RedactRefusal's backstop: warn token secret=%v, warn other secret=%v; want true, false",
			hasSecret([]Finding{warnToken}), hasSecret([]Finding{warnOther}))
	}
	if !blocksOutbound(warnToken) || blocksOutbound(warnOther) {
		t.Errorf("ScrubOutbound's stage three: warn token blocks=%v, warn other blocks=%v; want true, false",
			blocksOutbound(warnToken), blocksOutbound(warnOther))
	}
}

// TestWarnTokenNeverCountsAsAHardFail: the launch scan's hard-fail count reads
// the severity alone, so a plain key in a bundle — glued in a text file, or in
// the bytes of a binary one — is reported at warn and never fails the launch.
func TestWarnTokenNeverCountsAsAHardFail(t *testing.T) {
	root := t.TempDir()
	key := plainSKKey(52, 40)
	text := writeFile(t, root, "notes.md", "the value my_"+key+" sits here\n")
	bin := writeFile(t, root, "pic.png", "\x89PNG\x00\x00 "+key+" \x00")
	sc, err := New(root)
	if err != nil {
		t.Fatal(err)
	}
	res, err := sc.ScanBundle([]BundleFile{{LogicalPath: "notes.md", ResolvedPath: text}, {LogicalPath: "pic.png", ResolvedPath: bin}})
	if err != nil {
		t.Fatal(err)
	}
	if res.HardFails != 0 {
		t.Errorf("a warn token counted as %d hard-fail(s): %+v", res.HardFails, res.Findings)
	}
	seen := map[string]bool{}
	for _, f := range res.Findings {
		if f.Kind == "token:sk_generic" {
			seen[f.File] = true
		}
	}
	if !seen["notes.md"] || !seen["pic.png"] {
		t.Errorf("the plain key was not reported in both files (%v): %+v", seen, res.Findings)
	}
}

// TestLongerOpenRouterBodyIsMaskedToItsEnd — from the security review of
// iss-2610040202190813. A rule taking exactly 64 hex reports a longer body over
// its first 64 and leaves the rest raw after Redact, so the body is a floor;
// the 63-hex near miss stays quiet (TestSKKeyNearMissesStayQuiet).
func TestLongerOpenRouterBodyIsMaskedToItsEnd(t *testing.T) {
	key := "sk-or-v1-" + testsecret.SyntheticHex(54, 72)
	line := "OPENROUTER_API_KEY=" + key + " end"
	fs := scanLine(line)
	got := tokenFindings(fs)
	if len(got) != 1 || got[0].Kind != "token:openrouter" || got[0].Matched != key {
		t.Fatalf("want one token:openrouter over the whole %d-byte key, got %+v", len(key), got)
	}
	out, _ := Redact(line, fs)
	if tail := key[len(key)-8 : len(key)-2]; strings.Contains(out, tail) {
		t.Errorf("the body's tail survived redaction: %q", out)
	}
}
