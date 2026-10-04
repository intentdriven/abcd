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

// TestWordEndingInSKIsNotAPlainKey — from the re-check of
// iss-2610040202190813. The glued sweep drops the plain rule's leading \b, so
// any word ending in "sk" followed by '-' and 32 or more alphanumerics read as
// a glued key: a task id carrying a commit sha, a disk serial, a path segment,
// a lowercase word glued straight onto the run. Each was reported, masked by
// Redact and rewritten by ScrubOutbound. A word before "sk-" is never the '_',
// digit or uppercase glue a key is pasted behind, so the plain rule skips a
// match that ends a word; the same body glued behind "my_", "9" or "X" is still
// reported.
func TestWordEndingInSKIsNotAPlainKey(t *testing.T) {
	sha := testsecret.SyntheticHex(70, 40)
	body := testsecret.Synthetic(71, 40)
	for _, line := range []string{
		"see task-" + sha + " for the run",
		"the array lists disk-" + strings.Repeat("0123456789", 4)[:32] + " as spare",
		"open https://example.com/board/desk-" + testsecret.SyntheticHex(72, 40) + "/view",
		"a table cell Samaritaansk-" + body + " sits here",
	} {
		fs := scanLine(line)
		if got := tokenFindings(fs); len(got) != 0 {
			t.Errorf("%q: a word ending in sk read as a key: %+v", line, got)
		}
		if out, _ := Redact(line, fs); out != line {
			t.Errorf("%q: Redact rewrote ordinary text to %q", line, out)
		}
		got, _, err := ScrubOutbound(t.TempDir(), line+"\n", "pr-body")
		if err != nil || got != line+"\n" {
			t.Errorf("%q: ScrubOutbound returned %q, %v; want the text unchanged", line, got, err)
		}
	}
	for _, glue := range []string{"my_", "9", "X"} {
		line := "set " + glue + "sk-" + body + " now"
		if !hasKind(scanLine(line), "token:sk_generic") {
			t.Errorf("%q glued: the plain key is no longer reported", glue)
		}
	}
}

// TestKeyGluedBehindATokenEndingInLowercaseIsMasked: a plain key glued straight
// after another token is recovered at their junction, where the byte before it
// is the other token's last character. A skip keyed on that one byte being a
// lowercase letter dropped the key and left its body raw beside the masked
// neighbour; the word skip reads the whole run before the match, and a token's
// run is never word-shaped.
func TestKeyGluedBehindATokenEndingInLowercaseIsMasked(t *testing.T) {
	body := testsecret.Synthetic(73, 40)
	for _, n := range []int{35, 36} {
		line := "v gh" + "p_" + testsecret.Synthetic(74, n) + "a" + "sk-" + body + " end"
		fs := scanLine(line)
		if !hasKind(fs, "token:sk_generic") {
			t.Errorf("%d-char neighbour: the abutting plain key was dropped: %+v", n, fs)
		}
		if out, _ := Redact(line, fs); strings.Contains(out, body[4:20]) {
			t.Errorf("%d-char neighbour: the abutting plain key survived redaction: %q", n, out)
		}
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

// TestPlainSKKeyCanBeRaisedToHardFail: the opt-in internal/README.md states.
// A repository names the bundled rule in .abcd/config/pii.json and its plain
// keys then hard-fail; the regex and the kind stay the bundled ones.
func TestPlainSKKeyCanBeRaisedToHardFail(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, ".abcd/config/pii.json", `{"patterns": {"sk_key_generic": {"severity": "hard_fail"}}}`)
	sc, err := New(root)
	if err != nil {
		t.Fatal(err)
	}
	if bad, why := sc.Unavailable(); bad {
		t.Fatalf("the override degraded the scanner: %s", why)
	}
	fs := sc.ScanText(plainSKKey(53, 40), "f")
	if len(fs) != 1 || fs[0].Kind != "token:sk_generic" || fs[0].Severity != SeverityHardFail {
		t.Fatalf("want one token:sk_generic at hard_fail, got %+v", fs)
	}
}
