package ahoy

import "testing"

// TestScrubRemoteUserinfoDecodesBeforeTheColonTest pins iss-2609020630232658.
// git percent-decodes a URL's userinfo, so the colon separating a login from a
// password need not appear literally: ssh://user%3Apw@host carries a password
// exactly as ssh://user:pw@host does. A literal-colon test read it as a bare
// login — a route under ssh, and kept — so the encoded password went to rest in
// the history store, and the at-rest detector (defined as this function
// disagreeing with its input) never fired either. One round of decoding is
// what git applies, so a double-encoded %253A is literal text, not a
// separator; an undecodable userinfo is treated as a credential (fail closed).
func TestScrubRemoteUserinfoDecodesBeforeTheColonTest(t *testing.T) {
	for _, tc := range []struct{ in, want string }{
		{"ssh://user%3Apw@example.com/owner/repo.git", "ssh://example.com/owner/repo.git"},
		{"ssh://user%3apw@example.com/owner/repo.git", "ssh://example.com/owner/repo.git"},
		{"git+ssh://user%3Apw@example.com/owner/repo.git", "git+ssh://example.com/owner/repo.git"},
		{"user%3Apw@example.com:owner/repo.git", "example.com:owner/repo.git"},
		{"ssh://user%zzpw@example.com/owner/repo.git", "ssh://example.com/owner/repo.git"},
		// Bare logins stay, encoded or not, and a double encoding is not a colon.
		{"ssh://git@example.com/owner/repo.git", "ssh://git@example.com/owner/repo.git"},
		{"ssh://first%20last@example.com/owner/repo.git", "ssh://first%20last@example.com/owner/repo.git"},
		{"ssh://user%253Apw@example.com/owner/repo.git", "ssh://user%253Apw@example.com/owner/repo.git"},
		{"git@example.com:owner/repo.git", "git@example.com:owner/repo.git"},
		{"us%65r@example.com:owner/repo.git", "us%65r@example.com:owner/repo.git"},
	} {
		if got := scrubRemoteUserinfo(tc.in); got != tc.want {
			t.Errorf("scrubRemoteUserinfo(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}
