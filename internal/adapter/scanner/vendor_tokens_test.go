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

// letters maps a synthetic run onto letters alone, for the shapes whose body
// carries no digit.
func letters(s string) string {
	return strings.Map(func(r rune) rune {
		if r >= '0' && r <= '9' {
			return 'a' + (r - '0')
		}
		return r
	}, s)
}

// pypiLocation is the base64 of the macaroon header naming pypi.org, split so
// the source holds no run that reads as a token's opening.
const pypiLocation = "AgEIcHlw" + "aS5vcmc"

// vendorToken is one prefix-anchored vendor token, its kind, and the same
// shape one character short of the minimum.
type vendorToken struct {
	kind, key, short string
}

func vendorTokens() []vendorToken {
	syn, hex := testsecret.Synthetic, testsecret.SyntheticHex
	slackID := func(seed uint64) string { return strings.ToUpper(syn(seed, 10)) }
	return []vendorToken{
		{"token:npm", "npm_" + syn(101, 36), "npm_" + syn(101, 35)},
		{"token:gitlab_pat", "glpat-" + syn(102, 20), "glpat-" + syn(102, 19)},
		{"token:gitlab_pat", "glpat-" + syn(103, 27) + ".01" + hex(104, 7), "glpat-" + syn(103, 19) + ".01" + hex(104, 7)},
		{"token:huggingface", "hf_" + letters(syn(105, 34)), "hf_" + letters(syn(105, 33))},
		{"token:digitalocean", "dop_v1_" + hex(106, 64), "dop_v1_" + hex(106, 63)},
		{"token:digitalocean", "doo_v1_" + hex(107, 64), "doo_v1_" + hex(107, 63)},
		{"token:digitalocean", "dor_v1_" + hex(108, 64), "dor_v1_" + hex(108, 63)},
		{"token:vault", "hvs." + syn(109, 95), "hvs." + syn(109, 89)},
		{"token:vault", "hvb." + syn(110, 140), "hvb." + syn(110, 137)},
		{"token:shopify", "shpat_" + hex(111, 32), "shpat_" + hex(111, 31)},
		{"token:shopify", "shpca_" + hex(112, 32), "shpca_" + hex(112, 31)},
		{"token:shopify", "shppa_" + hex(113, 32), "shppa_" + hex(113, 31)},
		{"token:shopify", "shpss_" + hex(114, 32), "shpss_" + hex(114, 31)},
		{"token:pypi", "pypi-" + pypiLocation + syn(115, 120), "pypi-" + pypiLocation + syn(115, 49)},
		{"token:sendgrid", "SG." + syn(116, 22) + "." + syn(117, 43), "SG." + syn(116, 22) + "." + syn(117, 42)},
		{"token:slack_webhook",
			"hooks.slack.com/services/T" + slackID(118) + "/B" + slackID(119) + "/" + syn(120, 24),
			"hooks.slack.com/services/T" + slackID(118) + "/B" + slackID(119) + "/" + syn(120, 23)},
	}
}

// TestVendorTokensHardFailOnceByTheirKind: each prefix-anchored vendor token
// is reported exactly once, by its own kind, at hard_fail, over the whole
// token, bare, quoted in an assignment and inside prose; and Redact leaves
// none of its body behind.
func TestVendorTokensHardFailOnceByTheirKind(t *testing.T) {
	for _, c := range vendorTokens() {
		for _, line := range []string{c.key, `TOKEN="` + c.key + `"`, "use " + c.key + " here", "https://" + c.key} {
			fs := scanLine(line)
			got := tokenFindings(fs)
			if len(got) != 1 {
				t.Errorf("%s in %q: want exactly one token finding, got %d: %+v", c.kind, line, len(got), got)
				continue
			}
			if got[0].Kind != c.kind || got[0].Severity != SeverityHardFail {
				t.Errorf("%s: reported as %s/%s, want %s/hard_fail", c.kind, got[0].Kind, got[0].Severity, c.kind)
			}
			if got[0].Matched != c.key {
				t.Errorf("%s: matched %d bytes of a %d-byte token", c.kind, len(got[0].Matched), len(c.key))
			}
			out, _ := Redact(line, fs)
			if tail := c.key[len(c.key)-12:]; strings.Contains(out, tail) {
				t.Errorf("%s: the token survived redaction: %q", c.kind, out)
			}
		}
	}
}

// TestVendorTokensSurviveNoSuffix: each token is still caught with an
// underscore, a letter or a dash glued after it, the trailing-boundary gap the
// set's other token rules close (trailing_boundary_test.go).
func TestVendorTokensSurviveNoSuffix(t *testing.T) {
	for _, c := range vendorTokens() {
		for _, suffix := range []string{"_old", "_", "-x", "x"} {
			if !hasKind(scanLine(c.key+suffix), c.kind) {
				t.Errorf("%s survived the suffix %q", c.kind, suffix)
			}
		}
	}
}

// TestVendorTokenNearMissesStayQuiet: one character short of each shape is
// not taken for a token, nor are the ordinary words and identifiers that open
// on the same prefixes.
func TestVendorTokenNearMissesStayQuiet(t *testing.T) {
	quiet := []string{
		"npm_config_cache", "npm_package_version=1.2.3", "hf_hub_download(repo_id)",
		"glpat-placeholder", "dop_v1_", "hvs.example", "shpat_xxxxxxxx", "pypi-token",
		"SG.example.com", "https://hooks.slack.com/services/", "see hooks.slack.com for setup",
		"sendgrid SG." + strings.Repeat("x", 22) + ".short",
	}
	for _, c := range vendorTokens() {
		quiet = append(quiet, c.short)
	}
	for _, line := range quiet {
		if got := tokenFindings(scanLine(line)); len(got) != 0 {
			t.Errorf("%q taken for a token: %+v", line, got)
		}
	}
}

// TestGluedVendorTokenIsFound: a vendor token glued behind a word character
// has no leading boundary, and the glued sweep recovers it as it does every
// other token rule.
func TestGluedVendorTokenIsFound(t *testing.T) {
	for _, c := range vendorTokens() {
		for _, glue := range []string{"my_", "X", "9"} {
			line := "set " + glue + c.key + " now"
			fs := scanLine(line)
			if !hasKind(fs, c.kind) {
				t.Errorf("%s glued behind %q not found: %+v", c.kind, glue, fs)
				continue
			}
			if out, _ := Redact(line, fs); strings.Contains(out, c.key[len(c.key)-12:]) {
				t.Errorf("%s glued behind %q survived redaction: %q", c.kind, glue, out)
			}
		}
	}
}
