package scanner

import (
	"strings"
	"testing"
)

// pem_renderings_test.go — iss-2609020127210042: the block consumer redacted
// the common renderings of a key body but not every one. A body line the shape
// rule declined ended the block there and the rest of the key was stored
// verbatim; the tail of a block longer than maxPEMBodyLines survived by the
// same route; the RFC 4716 armour opened no block at all; and a one-line open
// key followed by prose kept its short padding chunk. Markers are assembled
// from halves and bodies are repeated letters, as in pem_block_test.go.

func rfc4716Fixture() (header, end string) {
	return "---- BEGIN " + "SSH2 ENCRYPTED PRIVATE KEY ----", "---- END " + "SSH2 ENCRYPTED PRIVATE KEY ----"
}

// TestRedactPEMBodyRenderings: every multi-line rendering the record names is
// consumed through its END line, and the prose on either side is untouched.
func TestRedactPEMBodyRenderings(t *testing.T) {
	header, body1, body2, tail, end := pemFixture()
	each := func(f func(string) string) []string {
		return []string{f(header), f(body1), f(body2), f(tail), f(end)}
	}
	cases := map[string][]string{
		"iso log prefix": each(func(s string) string { return "2026-09-01T10:00:00Z INFO " + s }),
		"bracketed log prefix": each(func(s string) string {
			return "[2026-09-01 10:00:00,123] [WARN] " + s
		}),
		"process log prefix":   each(func(s string) string { return "10:00:00 sshd[4242]: " + s }),
		"csv cell":             each(func(s string) string { return `7,"` + s + `"` }),
		"csv bare cell":        each(func(s string) string { return "7;" + s }),
		"xml element per line": each(func(s string) string { return "  <line>" + s + "</line>" }),
		"trailing hash comment": each(func(s string) string {
			return s + "  # fixture line"
		}),
		"source concatenation": each(func(s string) string { return `  "` + s + `\n" +` }),
		"leading concatenation": each(func(s string) string {
			return `  + "` + s + `"`
		}),
	}
	for name, block := range cases {
		t.Run(name, func(t *testing.T) {
			out := redactAll(t, "before the key\n"+strings.Join(block, "\n")+"\nafter the key")
			for _, leak := range []string{body1, body2, tail, "END "} {
				if strings.Contains(out, leak) {
					t.Errorf("%q… survived the block consumer:\n%s", leak[:4], out)
				}
			}
			if !strings.HasPrefix(out, "before the key\n") || !strings.HasSuffix(out, "\nafter the key") {
				t.Errorf("prose around the block was disturbed:\n%s", out)
			}
		})
	}
}

// TestRedactPEMOneLineRenderings: renderings that put the whole block on the
// header's own line — a JSON array, and a string whose newlines were escaped
// twice — are reached by the same-line pattern.
func TestRedactPEMOneLineRenderings(t *testing.T) {
	header, body1, body2, tail, end := pemFixture()
	cases := map[string]string{
		"json array":              `"key": ["` + header + `", "` + body1 + `", "` + body2 + `", "` + tail + `", "` + end + `"],`,
		"double-escaped newlines": `"key": "` + header + `\\n` + body1 + `\\n` + body2 + `\\n` + tail + `\\n` + end + `\\n",`,
	}
	for name, text := range cases {
		t.Run(name, func(t *testing.T) {
			out := redactAll(t, text)
			for _, leak := range []string{body1, body2, tail, "END "} {
				if strings.Contains(out, leak) {
					t.Errorf("%q… survived on the header's line:\n%s", leak[:4], out)
				}
			}
			if !strings.HasPrefix(out, `"key": `) {
				t.Errorf("the text before the key was disturbed: %q", out)
			}
		})
	}
}

// TestPEMOpenOneLineKeepsNoPaddingChunk is residual (a): an open one-line key
// followed by prose kept a short final chunk, because the chunk was taken only
// where it ended the line. The chunk is judged on itself now: a run carrying a
// byte a word cannot hold (the pad, a digit, '+', '/') is the key's tail.
func TestPEMOpenOneLineKeepsNoPaddingChunk(t *testing.T) {
	header, body1, _, _, _ := pemFixture()
	for _, tailChunk := range []string{"QQQ=", "Qq7Z", "Q+/Q"} {
		prose := " and then prose continues."
		out := redactAll(t, "pasted "+header+" "+body1+" "+tailChunk+prose)
		if strings.Contains(out, tailChunk) {
			t.Errorf("the padding chunk %q survived: %q", tailChunk, out)
		}
		if !strings.HasPrefix(out, "pasted ") || !strings.HasSuffix(out, prose) {
			t.Errorf("prose around the key is not byte-identical: %q", out)
		}
	}
	// A word after the body is still prose, not a tail.
	prose := " and it was rotated on Tuesday, fine."
	if out := redactAll(t, "pasted "+header+" "+body1+prose); !strings.HasSuffix(out, prose) {
		t.Errorf("a word after an open body was taken as its tail: %q", out)
	}
}

// TestPEMRFC4716ArmourIsABlock: the SSH2 armour (four dashes and a space) is
// detected and its body, headers included, consumed through its END line.
func TestPEMRFC4716ArmourIsABlock(t *testing.T) {
	_, body1, body2, tail, _ := pemFixture()
	header, end := rfc4716Fixture()
	cases := map[string][]string{
		"plain":        {header, `Comment: "rsa-key-20260901"`, body1, body2, tail, end},
		"subject":      {header, "Subject: fixture", "x-command: none", body1, tail, end},
		"continuation": {header, `Comment: "a long comment that \`, `wraps onto a second line"`, body1, tail, end},
		"open":         {header, body1, body2},
	}
	for name, block := range cases {
		t.Run(name, func(t *testing.T) {
			text := "before the key\n" + strings.Join(block, "\n") + "\nafter the key"
			findings := ScanText(text, Identity{}, DefaultPatterns(), DefaultIdentitySeverities(), "t")
			if !hasKind(findings, kindPEMPrivateKey) {
				t.Fatalf("the RFC 4716 header was not detected: %v", findings)
			}
			out, _ := Redact(text, findings)
			for _, leak := range []string{body1, body2, tail, "END ", "wraps onto"} {
				if strings.Contains(out, leak) {
					t.Errorf("%q… survived the block consumer:\n%s", leak[:4], out)
				}
			}
			if !strings.HasPrefix(out, "before the key\n") || !strings.HasSuffix(out, "\nafter the key") {
				t.Errorf("prose around the block was disturbed:\n%s", out)
			}
		})
	}
	// The one-line form reaches its END marker on the same line.
	out := redactAll(t, "note: "+header+" "+body1+" "+tail+" "+end+" rotated since")
	if strings.Contains(out, body1) || strings.Contains(out, "END ") || !strings.HasSuffix(out, " rotated since") {
		t.Errorf("one-line RFC 4716 block not masked through its END marker: %q", out)
	}
}

// TestRedactPEMMoreOneLineRenderings holds the renderings the separator did not
// admit (iss-2609251553082721): a JSON array serialised inside a JSON string,
// whose elements are joined by escaped quotes, and the HTML renderings that
// join the lines with a <br> element or a newline entity.
func TestRedactPEMMoreOneLineRenderings(t *testing.T) {
	header, body1, body2, tail, end := pemFixture()
	join := func(sep string) string {
		return strings.Join([]string{header, body1, body2, tail, end}, sep)
	}
	cases := map[string]string{
		"json array inside a json string": `"key": "[\"` + join(`\",\"`) + `\"]",`,
		"json array escaped twice":        `"key": "[\\\"` + join(`\\\",\\\"`) + `\\\"]",`,
		"html br":                         `"key": "` + join("<br>") + `",`,
		"html self-closing br":            `"key": "` + join("<br />") + `",`,
		"html newline entity":             `"key": "` + join("&#10;") + `",`,
		"html carriage-return entity":     `"key": "` + join("&#13;&#10;") + `",`,
	}
	for name, text := range cases {
		t.Run(name, func(t *testing.T) {
			out := redactAll(t, text)
			for _, leak := range []string{body1, body2, tail, "END "} {
				if strings.Contains(out, leak) {
					t.Errorf("%q… survived on the header's line:\n%s", leak[:4], out)
				}
			}
			if !strings.HasPrefix(out, `"key": `) {
				t.Errorf("the text before the key was disturbed: %q", out)
			}
		})
	}
}

// TestRedactPEMLowerCasedBlock: a tool that lowers case writes the armour
// markers in lower case, and the block is the same key; the header is detected
// and the body consumed through its END line, on one line and on many.
func TestRedactPEMLowerCasedBlock(t *testing.T) {
	header, body1, body2, tail, end := pemFixture()
	lh, le := strings.ToLower(header), strings.ToLower(end)
	for name, text := range map[string]string{
		"block":    "before\n" + lh + "\n" + body1 + "\n" + body2 + "\n" + tail + "\n" + le + "\nafter",
		"one line": `"key": "` + lh + `\n` + body1 + `\n` + body2 + `\n` + tail + `\n` + le + `",`,
	} {
		t.Run(name, func(t *testing.T) {
			out := redactAll(t, text)
			for _, leak := range []string{body1, body2, tail} {
				if strings.Contains(out, leak) {
					t.Errorf("%q… survived under a lower-cased header:\n%s", leak[:4], out)
				}
			}
		})
	}
}
