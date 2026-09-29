package scanner

import (
	"regexp"
	"strings"
	"testing"
)

// gluedTokens builds the two credential shapes the glued-token tests plant, at
// runtime so no token-shaped literal is ever committed: a GitHub PAT whose body
// is a run of D, and an AWS access key id whose body is a run of Q. Each is
// returned with a run of its body long enough that it can only survive
// redaction if the body itself did (the seal keeps three head runes and two
// tail runes).
func gluedTokens() (pat, patBody, akia, akiaBody string) {
	pat = "gh" + "p_" + strings.Repeat("D", 36)
	akia = "AK" + "IA" + strings.Repeat("Q", 16)
	return pat, strings.Repeat("D", 6), akia, strings.Repeat("Q", 6)
}

// TestRedactRefusalSealsAWordGluedToken — iss-2609290541525428. Every bundled
// secret pattern anchors its start on a leading \b, and '_', a letter and a
// digit are word characters, so a token right behind one has no boundary and
// the scan never matched it: a key or a path spelled notes_<token>, or a token
// with a letter on each side, came back from RedactRefusal raw. Each spelling
// is sealed now, and the readable part of the text around it is kept.
func TestRedactRefusalSealsAWordGluedToken(t *testing.T) {
	pat, patBody, akia, akiaBody := gluedTokens()
	cases := []struct {
		name, text, token, body, keep string
	}{
		{"pat behind an underscore, in a key", `json: unknown field "notes_` + pat + `"`, pat, patBody, `json: unknown field "notes_`},
		{"access key behind an underscore, in a key", `json: unknown field "notes_` + akia + `"`, akia, akiaBody, `json: unknown field "notes_`},
		{"pat between two letters", "x" + pat + "y", pat, patBody, "x"},
		{"access key between two letters", "x" + akia + "y", akia, akiaBody, "x"},
		{"pat behind an underscore, in a path", "sub/notes_" + pat + ".md does not exist", pat, patBody, "sub/notes_"},
		{"access key behind a digit, in a path", "sub/v2" + akia + ".md", akia, akiaBody, "sub/v2"},
		{"pat behind a run of underscore-joined words", "a_b_c_d_" + pat, pat, patBody, "a_b_c_d_"},
		{"two glued tokens in one key", "k_" + pat + "_" + akia, akia, akiaBody, "k_"},
		{"a bounded token still seals", "notes-" + pat, pat, patBody, "notes-"},
	}
	repo := t.TempDir()
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := RedactRefusal(repo, tc.text)
			if strings.Contains(got, tc.token) || strings.Contains(got, tc.body) {
				t.Errorf("RedactRefusal echoed the glued token: %q", got)
			}
			if tc.name == "two glued tokens in one key" && strings.Contains(got, patBody) {
				t.Errorf("RedactRefusal echoed the first of two glued tokens: %q", got)
			}
			if !strings.HasPrefix(got, tc.keep) {
				t.Errorf("RedactRefusal lost the readable text before the token (want prefix %q): %q", tc.keep, got)
			}
		})
	}
}

// TestRedactRefusalLeavesOrdinaryGluedWordsAlone: dropping the leading boundary
// must not turn an ordinary snake_case key or a prefix-shaped word into a
// finding. A key the reader needs is returned as it was written.
func TestRedactRefusalLeavesOrdinaryGluedWordsAlone(t *testing.T) {
	repo := t.TempDir()
	for _, text := range []string{
		`json: unknown field "reviewer_notes_for_the_second_round"`,
		`json: unknown field "task_ghp_short"`,
		`json: unknown field "xAKIA_not_a_key"`,
		"sub/notes_2026-09-29_draft.md does not exist",
	} {
		if got := RedactRefusal(repo, text); got != text {
			t.Errorf("RedactRefusal rewrote ordinary text:\n in:  %q\n out: %q", text, got)
		}
	}
}

// TestGluedSweepFailsClosedOnAnUncompilablePattern: a configured pattern whose
// leading \b carries a quantifier has no boundary-free form. The sweep says it
// cannot vouch for the text rather than silently dropping the pattern, and the
// sweep over the bundled set always can.
func TestGluedSweepFailsClosedOnAnUncompilablePattern(t *testing.T) {
	bad := Pattern{Name: "configured", Kind: "token:configured", Severity: SeverityHardFail, Re: regexp.MustCompile(`\b*zz[0-9]{8}`)}
	if _, ok := gluedFindings("x", append(DefaultPatterns(), bad), "f"); ok {
		t.Error("the sweep vouched for the text with a pattern it could not build")
	}
	if _, ok := gluedFindings("x", DefaultPatterns(), "f"); !ok {
		t.Error("the sweep could not build the bundled pattern set")
	}
}

// TestRedactRefusalSealsAnEscapedGluedToken — RedactRefusal reads ScanText, so
// a glued token spelled with escaped bytes came back raw from it too.
func TestRedactRefusalSealsAnEscapedGluedToken(t *testing.T) {
	repo := t.TempDir()
	for _, tc := range escapedGluedLines() {
		t.Run(tc.name, func(t *testing.T) {
			got := RedactRefusal(repo, tc.line)
			if tail := tc.token[len(tc.token)-12:]; strings.Contains(got, tail) {
				t.Errorf("RedactRefusal echoed the escaped glued token: %q", got)
			}
		})
	}
}

// TestUnavailableNamesAnIncompleteGluedSweep — iss-2609290743362554. A
// configured secret pattern whose leading \b carries a quantifier loads, but
// the glued sweep cannot build its boundary-free form, so every ScanText
// consumer ran a narrower sweep and only RedactRefusal knew. The scanner now
// says so where every write-time redactor and the launch scan already look:
// Unavailable, with a reason naming the pattern. A configured pattern the
// sweep can build leaves the scanner available.
func TestUnavailableNamesAnIncompleteGluedSweep(t *testing.T) {
	for _, tc := range []struct {
		name, regex string
		degraded    bool
	}{
		{"quantified boundary", `\\b*zz[0-9]{8}`, true},
		{"plain boundary", `\\bzz[0-9]{8}\\b`, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root := t.TempDir()
			writeFile(t, root, ".abcd/config/pii.json", `{ "patterns": { "zz_custom": { "regex": "`+tc.regex+`", "severity": "hard_fail" } } }`)
			sc, err := New(root)
			if err != nil {
				t.Fatal(err)
			}
			degraded, reason := sc.Unavailable()
			if degraded != tc.degraded {
				t.Fatalf("Unavailable() = %v (%q), want %v", degraded, reason, tc.degraded)
			}
			if !tc.degraded {
				return
			}
			if !strings.Contains(reason, "zz_custom") || !strings.Contains(reason, "glued") {
				t.Errorf("the reason does not name the pattern and the sweep: %q", reason)
			}
			res, err := sc.ScanBundle(nil)
			if err != nil {
				t.Fatal(err)
			}
			if !res.Unavailable || res.UnavailableReason != reason {
				t.Errorf("ScanBundle did not surface the degraded sweep: %+v", res)
			}
		})
	}
}
