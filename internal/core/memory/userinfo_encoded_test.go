package memory

import (
	"strings"
	"testing"
)

// TestRefusalRenderersMaskAPercentEncodedPassword is the memory half of the
// sweep for iss-2609020630232658. A URL's userinfo is percent-decoded before
// use, so https://user%3Apw@host carries the password "pw" behind an encoded
// colon. url.Parse reads that as a USERNAME containing a colon and no password,
// so url.URL.Redacted (which masks only a parsed password) echoed the encoded
// password into a refusal, and the textual fallback kept everything before a
// literal colon. Both must decode before they decide where the login ends.
func TestRefusalRenderersMaskAPercentEncodedPassword(t *testing.T) {
	const secret = "pwSECRET"
	for _, in := range []string{
		"https://user%3A" + secret + "@example.com/a",
		"https://user%3a" + secret + "@example.com/a",
	} {
		got := redactedSource(in)
		if strings.Contains(got, secret) {
			t.Errorf("redactedSource(%q) = %q: the encoded password survived", in, got)
		}
		if !strings.Contains(got, "user") {
			t.Errorf("redactedSource(%q) = %q: the login name the operator recognises was lost", in, got)
		}
	}
	// The textual fallback, reached when url.Parse refuses the string.
	for _, in := range []string{
		"https://user%3A" + secret + "@exa mple.com/%zz",
		"https://user%zz" + secret + "@example.com/a",
	} {
		if got := maskUserinfo(in); strings.Contains(got, secret) {
			t.Errorf("maskUserinfo(%q) = %q: the encoded password survived", in, got)
		}
	}
	if got := maskUserinfo("https://user:" + secret + "@example.com/a"); got != "https://user:xxxxx@example.com/a" {
		t.Errorf("maskUserinfo literal-colon form = %q", got)
	}
}
