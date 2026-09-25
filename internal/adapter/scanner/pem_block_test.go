package scanner

import (
	"strings"
	"testing"
)

// pem_block_test.go — GHSA-gmp7-9rvm-qcr3 / GHSA-5qr6-f78x-g2cx /
// GHSA-29jw-3jg9-qmhx: the pem_private_key pattern matched the BEGIN header
// only, so Redact masked one line and every store wrote the base64 key body
// and the END line verbatim while reporting the record redacted. Every marker
// here is assembled from halves at runtime and every body is a repeated
// letter: nothing in this file is, or scans as, a private key.

func pemFixture() (header, body1, body2, tail, end string) {
	header = "-----BEGIN " + "OPENSSH PRIVATE KEY-----"
	body1 = strings.Repeat("Q", 64)
	body2 = strings.Repeat("R", 64)
	tail = strings.Repeat("S", 12) + "="
	end = "-----END " + "OPENSSH PRIVATE KEY-----"
	return
}

// pemNarrowFixture returns the three body renderings iss-2609090932441377
// reports as lost, and the identifier the opener must keep on refusing. Each
// is assembled at run time like every other marker here, and each is a
// REPEATED pattern rather than key material — deliberately, since the rule
// under test reads the run's alphabet and not its entropy, so a fixture that
// scanned as a key would be both unsafe and unnecessary.
func pemNarrowFixture() (narrow32, tail26, narrow30, identifier string) {
	narrow32 = strings.Repeat("Qq7Z", 8)         // 32 chars, carries a digit
	tail26 = strings.Repeat("Q", 24) + "=" + "=" // 26 chars, carries the pad
	narrow30 = strings.Repeat("Rr3", 10)         // 30 chars, carries a digit
	identifier = "CertificateRotationPolicy"     // 25 chars, pure [A-Za-z]
	return
}

func redactAllN(t *testing.T, text string) (string, int) {
	t.Helper()
	findings := ScanText(text, Identity{}, DefaultPatterns(), DefaultIdentitySeverities(), "t")
	if !hasKind(findings, "token:pem_private_key") {
		t.Fatalf("the PEM header was not detected at all: %v", findings)
	}
	return Redact(text, findings)
}

func redactAll(t *testing.T, text string) string {
	t.Helper()
	out, _ := redactAllN(t, text)
	return out
}

// TestRedactPEMBlockConsumesBodyThroughEnd: a header on its own line is
// followed by the key body and the END line; none of them may survive, the
// prose on either side must, and the stage-two rescan must be clean.
func TestRedactPEMBlockConsumesBodyThroughEnd(t *testing.T) {
	header, body1, body2, tail, end := pemFixture()
	pat := "ghp_" + strings.Repeat("a", 40)
	text := strings.Join([]string{
		"before the key",
		header, body1, body2, tail, end,
		"after the key, token " + pat,
	}, "\n")
	out := redactAll(t, text)
	for _, leak := range []string{body1, body2, tail, end, pat} {
		if strings.Contains(out, leak) {
			t.Errorf("redaction left %q in place:\n%s", leak[:8], out)
		}
	}
	for _, keep := range []string{"before the key", "after the key, token "} {
		if !strings.Contains(out, keep) {
			t.Errorf("redaction lost the prose %q:\n%s", keep, out)
		}
	}
	if resid := BlockingResidual(ScanText(out, Identity{}, DefaultPatterns(), DefaultIdentitySeverities(), "t")); len(resid) != 0 {
		t.Errorf("stage-two rescan of the redacted text is not clean: %v", resid)
	}
}

// TestRedactPEMHeaderWithoutEndKeepsProse: a header with no END line must not
// swallow the record after it. A body that demonstrably opened is still taken;
// a header that is only NAMED — in a rotation note, a runbook, an issue record
// — opens no block, so nothing after it may be consumed.
//
// The shape rule alone cannot tell the two apart. "Body-shaped" accepts a
// blank line, a code fence, a setext underline, a bare number and a
// single-token list item, so a sentence mentioning the header used to delete
// the lines after it from a committed record and could leave an unbalanced
// fence behind. The consumer therefore demands positive EVIDENCE that a body
// opened before it takes anything.
func TestRedactPEMHeaderWithoutEndKeepsProse(t *testing.T) {
	header, body1, _, _, _ := pemFixture()

	t.Run("truncated block", func(t *testing.T) {
		text := strings.Join([]string{
			header, body1, "",
			"The next paragraph is prose that must stay.",
			"So must this one, with an https://example.com/link in it.",
		}, "\n")
		out := redactAll(t, text)
		if strings.Contains(out, body1) {
			t.Errorf("the body line after a truncated block survived:\n%s", out)
		}
		for _, keep := range []string{"The next paragraph is prose that must stay.", "So must this one, with an https://example.com/link in it."} {
			if !strings.Contains(out, keep) {
				t.Errorf("a truncated block swallowed the prose %q:\n%s", keep, out)
			}
		}
	})

	mention := "Rotate the key: the file still opens with " + header + " and must go."
	cases := map[string][]string{
		"blank line":       {"", "The paragraph after the blank must stay."},
		"code fence":       {"```", "rotate revoke", "```", "after the fence"},
		"setext underline": {"Rotation", "========================", "The section body must stay."},
		"bare number":      {"42", "The numbered line must stay."},
		"single-word item": {"- rotate", "- revoke", "The list must stay."},
	}
	for name, after := range cases {
		t.Run(name, func(t *testing.T) {
			want := strings.Join(after, "\n")
			out := redactAll(t, mention+"\n"+want)
			if !strings.HasSuffix(out, "\n"+want) {
				t.Errorf("a bare header mention consumed the lines after it:\ngot  %q\nwant suffix %q", out, "\n"+want)
			}
		})
	}

	// The exact reproduction: a mention, a blank line, a fenced snippet, a
	// blank line and one more prose line. Five lines went, and the closing
	// fence outlived its opener.
	t.Run("reproduction", func(t *testing.T) {
		prose := strings.Join([]string{"", "```", "rotate revoke", "```", "", "Owner: platform."}, "\n")
		out, n := redactAllN(t, mention+"\n"+prose)
		if !strings.HasSuffix(out, "\n"+prose) {
			t.Errorf("prose after a bare header mention is not byte-identical:\ngot  %q\nwant suffix %q", out, "\n"+prose)
		}
		if strings.Contains(out, "[redacted-pem-body") {
			t.Errorf("a block was collapsed where none opened:\n%s", out)
		}
		// One rewrite: the header span itself. No block, so no second count.
		if n != 1 {
			t.Errorf("Redact reports %d rewrites, want 1 (the header span alone, 0 blocks)", n)
		}
	})
}

// TestRedactPEMBlockSurvivesASealOnABodyLine: the block consumer ran on the
// lines the per-line redaction had already rewritten, and a seal writes '*'
// bytes, which are not body-shaped. So a SECOND finding on a body line — an
// AWS-key-shaped run inside the base64 is enough — ended the block at that
// line and wrote the rest of the key, and the END marker, verbatim. The
// boundaries are computed on the original lines instead.
func TestRedactPEMBlockSurvivesASealOnABodyLine(t *testing.T) {
	header, _, _, _, end := pemFixture()
	awsShaped := "AKIA" + strings.Repeat("Z", 16)
	body1 := awsShaped + strings.Repeat("Q", 40)
	body2 := strings.Repeat("R", 64)
	text := strings.Join([]string{"before the key", header, body1, body2, end, "after the key"}, "\n")
	out := redactAll(t, text)
	for _, leak := range []string{body2, end, strings.Repeat("Q", 40)} {
		if strings.Contains(out, leak) {
			t.Errorf("a seal on a body line ended the block early; %q survived:\n%s", leak[:8], out)
		}
	}
	for _, keep := range []string{"before the key", "after the key"} {
		if !strings.Contains(out, keep) {
			t.Errorf("the prose %q was lost:\n%s", keep, out)
		}
	}
}

// TestRedactPEMOneLineBodyDoesNotSurvive: header, body and END on ONE line —
// a resolve note, a JSON/K8s secret dump with literal \n escapes. Byte-span
// sealing of the header alone left every body byte after it in place.
func TestRedactPEMOneLineBodyDoesNotSurvive(t *testing.T) {
	header, body1, _, tail, end := pemFixture()
	cases := map[string]string{
		"note":        "note: " + header + " " + body1 + " " + tail + " " + end + " rotated since",
		"json":        `{"key":"` + header + `\n` + body1 + `\n` + tail + `\n` + end + `\n"}`,
		"no end":      "pasted " + header + " " + body1 + " " + tail,
		"header only": "the file starts with " + header + " and is 3 KiB",
	}
	for name, text := range cases {
		t.Run(name, func(t *testing.T) {
			out := redactAll(t, text)
			if strings.Contains(out, body1) || strings.Contains(out, tail) {
				t.Errorf("body bytes on the header line survived:\n%s", out)
			}
			if name == "note" && !strings.Contains(out, "rotated since") {
				t.Errorf("prose after the END marker was lost:\n%s", out)
			}
			if name == "header only" && !strings.Contains(out, "is 3 KiB") {
				t.Errorf("prose after a bare header was lost:\n%s", out)
			}
		})
	}
}

// TestPEMHeaderIsMaskedWhole: a private-key span keeps no head/tail
// fingerprint. On the header alone the fingerprint was harmless; once the span
// extends over body bytes, the kept tail would be two bytes of key.
func TestPEMHeaderIsMaskedWhole(t *testing.T) {
	header, body1, _, _, _ := pemFixture()
	text := header + " " + body1
	out := redactAll(t, text)
	if strings.Contains(out, "---") || strings.HasSuffix(out, "QQ") {
		t.Errorf("PEM span kept a fingerprint of the raw bytes: %q", out)
	}
}

// TestRedactPEMBodyConsumerIsBounded pins the bound pem.go states and what it
// governs. Past maxPEMBodyLines a block goes on only over lines that carry key
// material, so the tail of an oversized key is consumed with it
// (iss-2609020127210042) while a pathological run of body-SHAPED lines that
// carry none — blank lines, single short tokens — still stops at the bound,
// and the prose after either is untouched.
func TestRedactPEMBodyConsumerIsBounded(t *testing.T) {
	header, body1, _, _, end := pemFixture()
	const beyond = 3
	t.Run("key material past the bound is the block's tail", func(t *testing.T) {
		lines := []string{header}
		for i := 0; i < maxPEMBodyLines+beyond; i++ {
			lines = append(lines, body1)
		}
		lines = append(lines, end, "prose after an oversized block")
		out := redactAll(t, strings.Join(lines, "\n"))
		if !strings.Contains(out, pemBodyPlaceholder(maxPEMBodyLines+beyond+1)) {
			t.Errorf("placeholder does not report the whole block: %q", out[len(out)-200:])
		}
		if !strings.Contains(out, "prose after an oversized block") {
			t.Errorf("prose after the block was lost")
		}
		if got := strings.Count(out, body1); got != 0 {
			t.Errorf("body lines past the bound survived: %d", got)
		}
	})
	t.Run("body-shaped lines with no key material stop at the bound", func(t *testing.T) {
		lines := []string{header, body1}
		for i := 0; i < maxPEMBodyLines+beyond; i++ {
			lines = append(lines, "ok")
		}
		lines = append(lines, "prose after a pathological block")
		out := redactAll(t, strings.Join(lines, "\n"))
		if !strings.Contains(out, pemBodyPlaceholder(maxPEMBodyLines)) {
			t.Errorf("placeholder does not report the bound: %q", out[len(out)-200:])
		}
		if got := strings.Count(out, "\nok"); got != beyond+1 {
			t.Errorf("lines past the bound: got %d surviving, want %d (the bound is the contract)", got, beyond+1)
		}
	})
}

// TestRedactPEMBlockWithGutters: a block pasted with indentation, a diff
// marker, a line-number gutter and JSON quoting is still consumed through
// its END line.
func TestRedactPEMBlockWithGutters(t *testing.T) {
	header, body1, body2, tail, end := pemFixture()
	cases := map[string][]string{
		"indented":  {"    " + header, "    " + body1, "    " + body2, "    " + tail, "    " + end},
		"diff":      {"+" + header, "+" + body1, "-" + body2, "+" + tail, "+" + end},
		"numbered":  {"  1\t" + header, "  2\t" + body1, "  3\t" + body2, "  4\t" + tail, "  5\t" + end},
		"quoted":    {`  "` + header + `",`, `  "` + body1 + `",`, `  "` + body2 + `",`, `  "` + tail + `",`, `  "` + end + `"`},
		"encrypted": {header, "Proc-Type: 4,ENCRYPTED", "DEK-Info: AES-128-CBC,0123456789ABCDEF0123456789ABCDEF", "", body1, tail, end},
	}
	for name, block := range cases {
		t.Run(name, func(t *testing.T) {
			out := redactAll(t, "before\n"+strings.Join(block, "\n")+"\nafter")
			for _, leak := range []string{body1, body2, tail, end} {
				if strings.Contains(out, leak) {
					t.Errorf("%q survived:\n%s", leak[:8], out)
				}
			}
			if !strings.HasPrefix(out, "before\n") || !strings.HasSuffix(out, "\nafter") {
				t.Errorf("prose around the block was disturbed:\n%s", out)
			}
		})
	}
}

// TestPEMSameLineBodyStopsAtProse: on the header's own line the body was a
// chunk of 16+ base64 bytes followed by ANY word-runs to the first
// punctuation, so a sentence that pastes a key and then keeps talking lost its
// prose to the mask ("… and it was rotated on Tuesday" went). Every chunk the
// body claims must be long enough to be key material; a short final chunk is
// taken only where it closes the line or an END marker follows it.
func TestPEMSameLineBodyStopsAtProse(t *testing.T) {
	header, body1, _, tail, end := pemFixture()
	prose := " and it was rotated on Tuesday, fine."
	t.Run("open block", func(t *testing.T) {
		out := redactAll(t, "pasted "+header+" "+body1+prose)
		if strings.Contains(out, body1) {
			t.Errorf("the same-line body survived:\n%s", out)
		}
		if !strings.HasPrefix(out, "pasted ") || !strings.HasSuffix(out, prose) {
			t.Errorf("prose around the same-line body is not byte-identical:\ngot %q\nwant suffix %q", out, prose)
		}
	})
	// A closed block still takes its short final padding chunk: the END marker
	// after it is the evidence that the chunk belongs to the key.
	t.Run("closed block keeps taking the short tail", func(t *testing.T) {
		out := redactAll(t, "note: "+header+" "+body1+" "+tail+" "+end+prose)
		for _, leak := range []string{body1, tail, end} {
			if strings.Contains(out, leak) {
				t.Errorf("%q survived a closed one-line block:\n%s", leak[:8], out)
			}
		}
		if !strings.HasSuffix(out, prose) {
			t.Errorf("prose after the END marker is not byte-identical:\ngot %q", out)
		}
	})
}

// TestRedactPEMBlockClosesAtItsEndMarker: a REAL key block whose first body
// lines are blank or short carries its evidence deeper than the two-line
// window pemBodyEvidence looks over — a leading blank line, the short prefix
// chunks a DER body opens with, a hard-wrapped paste. The window is ONE route
// to opening a block, not the only one: an END marker reached within the bound
// over an unbroken run of body-shaped lines closes the block wherever the
// evidence sits. Without that second route the whole body and the END line
// were written verbatim into a committed record while the header alone was
// masked and the record asserted one redacted secret.
func TestRedactPEMBlockClosesAtItsEndMarker(t *testing.T) {
	header, body1, _, _, end := pemFixture()
	// Assembled from halves like every other marker here: four and four base64
	// bytes, far too short to be evidence and far too short to be key material.
	shortA, shortB := "MII"+"E", "Ag"+"EA"
	cases := map[string][]string{
		"blank first body lines":  {header, "", "", body1, end},
		"short leading chunks":    {header, shortA, shortB, body1, end},
		"blank then short chunk":  {header, "", shortA, body1, end},
		"blank then indented one": {header, "", "\t" + shortB, body1, end},
	}
	for name, block := range cases {
		t.Run(name, func(t *testing.T) {
			out := redactAll(t, "before the key\n"+strings.Join(block, "\n")+"\nafter the key")
			for _, leak := range []string{body1, end} {
				if strings.Contains(out, leak) {
					t.Errorf("the block was not closed at its END marker; %q survived:\n%s", leak[:8], out)
				}
			}
			if !strings.HasPrefix(out, "before the key\n") || !strings.HasSuffix(out, "\nafter the key") {
				t.Errorf("prose around the block was disturbed:\n%s", out)
			}
		})
	}
}

// mentionFixture is the exact shape iss-2609020127210042 sub-residual (b)
// reports: a security note that NAMES a BEGIN marker in a sentence, then a
// fenced snippet whose single line is a long CamelCase identifier. The
// identifier is 25 characters from the base64 alphabet, so the old opener
// (any run of 16+) read it as key material two lines past the header, the
// block consumer spliced the fence, the identifier and the closing fence out
// of the record, and the reader was told nothing. The orphaned fence then
// broke `abcd site build`, a make preflight gate.
func mentionFixture() string {
	header := "-----BEGIN " + "RSA PRIVATE KEY-----"
	return strings.Join([]string{
		"We rotate on the " + header + " header, per:",
		"```",
		"CertificateRotationPolicy",
		"```",
		"Owner: platform.",
	}, "\n")
}

// TestRedactPEMMentionedHeaderDoesNotOpenOnAnIdentifier is the negative half
// of the opener pair: a mention must open no block, so nothing after the
// header's own line may move. The header line itself IS rewritten — its span
// is masked whole, which is the detection working and not the defect — so the
// byte-identity claim runs from the line after it, and the fence count over
// the whole output must be unchanged (an odd count is the orphan that breaks
// the site render).
func TestRedactPEMMentionedHeaderDoesNotOpenOnAnIdentifier(t *testing.T) {
	input := mentionFixture()
	out, n := redactAllN(t, input)
	inLines, outLines := strings.Split(input, "\n"), strings.Split(out, "\n")
	if len(outLines) != len(inLines) {
		t.Fatalf("a mention consumed lines: got %d lines, want %d:\n--- got\n%s\n--- want\n%s", len(outLines), len(inLines), out, input)
	}
	for i := 1; i < len(inLines); i++ {
		if outLines[i] != inLines[i] {
			t.Errorf("line %d after a mentioned header is not byte-identical:\ngot  %q\nwant %q\n--- whole output\n%s", i+1, outLines[i], inLines[i], out)
		}
	}
	if got, want := strings.Count(out, "```"), strings.Count(input, "```"); got != want {
		t.Errorf("fence count changed: got %d, want %d (an orphaned fence breaks the site render):\n%s", got, want, out)
	}
	if strings.Contains(out, "[redacted-pem-body") {
		t.Errorf("a block was collapsed where a header was only named:\n%s", out)
	}
	// One rewrite: the header span. No block, so no second count.
	if n != 1 {
		t.Errorf("Redact reports %d rewrites, want 1 (the header span alone, 0 blocks):\n%s", n, out)
	}
}

// TestRedactPEMOpenerStillTakesEveryRealKeyShape is the positive half, and it
// is the half that matters: raising the opener's bar LOOSENS a security
// control, so every real-key rendering the three advisories bought —
// GHSA-gmp7-9rvm-qcr3, GHSA-5qr6-f78x-g2cx, GHSA-29jw-3jg9-qmhx — must still
// collapse to the placeholder with no body byte and no END marker left. The
// cases are the two routes into pemBlockEnd: a closed block (the END marker is
// its own evidence, whatever the gutter) and an open one (the opener rule is
// the only evidence there is).
func TestRedactPEMOpenerStillTakesEveryRealKeyShape(t *testing.T) {
	header, body1, body2, tail, end := pemFixture()
	narrow1, narrow2 := strings.Repeat("Q", 20), strings.Repeat("R", 20)
	narrow32, tail26, narrow30, _ := pemNarrowFixture()
	cases := map[string]struct {
		block []string
		leaks []string
	}{
		"closed block":            {[]string{header, body1, body2, tail, end}, []string{body1, body2, tail, end}},
		"open block, 64-wide":     {[]string{header, body1, body2}, []string{body1, body2}},
		"open block, blank first": {[]string{header, "", body1, body2}, []string{body1, body2}},
		"open block, narrow wrap": {[]string{header, narrow1, narrow2, body1}, []string{narrow1, narrow2, body1}},
		// iss-2609090932441377: a block truncated or re-wrapped to ONE visible
		// body line, which the pair rule cannot see and a width bar cannot
		// reach.
		"open block, one narrow line":     {[]string{header, narrow32}, []string{narrow32}},
		"open block, padding tail":        {[]string{header, "", tail26}, []string{tail26}},
		"open block, narrow then a blank": {[]string{header, narrow30, ""}, []string{narrow30}},
		"armour header":                   {[]string{header, "Proc-Type: 4,ENCRYPTED", "DEK-Info: AES-128-CBC,0123456789ABCDEF0123456789ABCDEF", "", body1, tail, end}, []string{body1, tail, end}},
		"indented":                        {[]string{"    " + header, "    " + body1, "    " + body2, "    " + tail, "    " + end}, []string{body1, body2, tail, end}},
		"quoted":                          {[]string{`  "` + header + `",`, `  "` + body1 + `",`, `  "` + body2 + `",`, `  "` + tail + `",`, `  "` + end + `"`}, []string{body1, body2, tail, end}},
		"diff-prefixed":                   {[]string{"+" + header, "+" + body1, "-" + body2, "+" + tail, "+" + end}, []string{body1, body2, tail, end}},
		"line-numbered":                   {[]string{"  1\t" + header, "  2\t" + body1, "  3\t" + body2, "  4\t" + tail, "  5\t" + end}, []string{body1, body2, tail, end}},
	}
	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			out := redactAll(t, "before the key\n"+strings.Join(c.block, "\n")+"\nafter the key")
			if !strings.Contains(out, "[redacted-pem-body") {
				t.Errorf("no block was collapsed; the opener bar is too high:\n%s", out)
			}
			for _, leak := range c.leaks {
				if strings.Contains(out, leak) {
					t.Errorf("%q… survived the block consumer:\n%s", leak[:8], out)
				}
			}
			if !strings.HasPrefix(out, "before the key\n") || !strings.HasSuffix(out, "\nafter the key") {
				t.Errorf("prose around the block was disturbed:\n%s", out)
			}
		})
	}
}

// TestRedactPEMNarrowOpenerDiscriminates is the one table that has to hold
// both directions at once, because the two failures it stands between are the
// same rule read from opposite ends. Raising the single-line opener to a key
// body's characteristic width kept a mentioned header from eating an
// identifier (eb7b6502) and, in the same stroke, stopped redacting a block
// whose visible body is ONE line under that width — a key truncated by a log
// rotation or re-wrapped narrow by a mail client, which the pair rule cannot
// see because there is no second line (iss-2609090932441377). Width alone
// cannot separate them: the identifier is 25 characters and the lost body is
// 32, so any bar that clears one clears the other.
//
// The discriminator is therefore not width but two things width says nothing
// about, and the table isolates each by holding the other fixed. A pasted
// header is one whose marker ENDS its line, as a block pasted into a
// transcript does behind whatever gutter or speaker prefix it carries; a
// NAMED header has prose after it, because a sentence goes on. And a key
// body's run carries at least one byte of the base64 alphabet that is not a
// letter — a digit, '+', '/' or the '=' pad — where an identifier a person
// types is letters and word structure. Rows four and five each fail exactly
// one half, and each must be refused on that half alone.
func TestRedactPEMNarrowOpenerDiscriminates(t *testing.T) {
	pasted := "-----BEGIN " + "RSA PRIVATE KEY-----"
	named := "We rotate on the " + pasted + " header, per:"
	narrow32, tail26, narrow30, identifier := pemNarrowFixture()
	underline := strings.Repeat("=", 24)

	cases := map[string]struct {
		block []string
		body  string
		taken bool
	}{
		"pasted header, one narrow run":         {[]string{pasted, narrow32}, narrow32, true},
		"pasted header, padding tail":           {[]string{pasted, "", tail26}, tail26, true},
		"pasted header, narrow run and a blank": {[]string{pasted, narrow30, ""}, narrow30, true},
		"pasted header behind a gutter":         {[]string{`  "` + pasted + `",`, `  "` + narrow32 + `",`}, narrow32, true},
		"pasted header, an identifier":          {[]string{pasted, identifier}, identifier, false},
		"named header, one narrow run":          {[]string{named, narrow32}, narrow32, false},
		"pasted header, a setext underline":     {[]string{pasted, underline}, underline, false},
	}
	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			out := redactAll(t, "before the key\n"+strings.Join(c.block, "\n")+"\nafter the key")
			collapsed := strings.Contains(out, "[redacted-pem-body")
			if c.taken {
				if !collapsed {
					t.Errorf("no block was collapsed; a truncated body survives:\n%s", out)
				}
				if strings.Contains(out, c.body) {
					t.Errorf("the body line %q… survived the block consumer:\n%s", c.body[:8], out)
				}
			} else {
				if collapsed {
					t.Errorf("a block was collapsed where none opened:\n%s", out)
				}
				if !strings.Contains(out, c.body) {
					t.Errorf("the line %q… was consumed as key material:\n%s", c.body[:8], out)
				}
			}
			if !strings.HasPrefix(out, "before the key\n") || !strings.HasSuffix(out, "\nafter the key") {
				t.Errorf("prose around the block was disturbed:\n%s", out)
			}
		})
	}
}
