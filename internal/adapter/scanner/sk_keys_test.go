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
		if strings.HasPrefix(f.Kind, "token:") {
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
