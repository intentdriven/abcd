package gitutil

import (
	"net/url"
	"regexp"
	"strings"
)

// ScrubURLUserinfo drops any credential from a git remote URL, keeping a
// login the transport needs (the rule ahoy's registry and `abcd land` share,
// GHSA-qc3w-8pv5-crc3):
//   - scheme form (`scheme://[userinfo@]host/path`): a userinfo carrying a
//     password is dropped whole under any scheme; under http(s) a bare user is
//     dropped too, because the forge accepts a token as the user with no
//     password; under any other scheme (ssh, git+ssh) a bare user is the login
//     name the transport needs, so it stays;
//   - scp-like form (`[user@]host:path`): a user segment carrying a password is
//     dropped; `git@host:path` is an SSH login and stays. The form is recognised
//     by the host:path colon that FOLLOWS the userinfo, so a remote whose path
//     contains an @ is not mistaken for one.
//
// Everything else — a local path, an empty value — is returned unchanged. That
// includes `user:secret@host/path`, which is neither form: with no scheme and no
// host:path colon, git reads it as a relative local path, not a remote.
func ScrubURLUserinfo(s string) string {
	if i := strings.Index(s, "://"); i >= 0 {
		scheme := strings.ToLower(s[:i])
		rest := s[i+3:]
		authority := rest
		if slash := strings.IndexByte(rest, '/'); slash >= 0 {
			authority = rest[:slash]
		}
		at := strings.LastIndexByte(authority, '@')
		if at < 0 {
			return s
		}
		if !userinfoCarriesPassword(authority[:at]) && scheme != "http" && scheme != "https" {
			return s
		}
		return s[:i+3] + authority[at+1:] + rest[len(authority):]
	}
	at := strings.IndexByte(s, '@')
	if at < 0 {
		return s
	}
	// What makes a value scp-like is the host:path colon AFTER the userinfo, not
	// merely a colon somewhere before the @. Reading it the other way misclassifies
	// a remote whose PATH contains an @ — `example.com:owner/re@po.git` has a colon
	// before the first @ because that colon is the host:path separator — and
	// "scrubbing" it would rewrite the address to "po.git" and lose the host.
	if !strings.Contains(s[at+1:], ":") {
		return s
	}
	// A userinfo with no colon is a bare login (`git@host:path`): the transport
	// needs it and it is a route, not a secret.
	if !userinfoCarriesPassword(s[:at]) {
		return s
	}
	return s[at+1:]
}

// userinfoCarriesPassword reports whether a userinfo holds a login:password
// pair. The test runs on the DECODED userinfo, because git percent-decodes it
// before use: `user%3Apw` is the login "user" with the password "pw", and a
// literal-colon test read it as a bare login (iss-2609020630232658). One round
// of decoding is what git applies, so `%253A` decodes to the literal text
// `%3A` and is not a separator. A userinfo that does not decode is treated as
// carrying one: this function decides what may go to rest, so it fails closed.
func userinfoCarriesPassword(userinfo string) bool {
	decoded, err := url.PathUnescape(userinfo)
	if err != nil {
		return true
	}
	return strings.Contains(decoded, ":")
}

// urlInTextRe is a scheme-form URL inside free text, up to the next space or
// quote.
var urlInTextRe = regexp.MustCompile(`[A-Za-z][A-Za-z0-9+.-]*://[^\s'"<>]+`)

// queryTokenRe is a token carried in a URL's query: the value of an
// access_token or token parameter, any case.
var queryTokenRe = regexp.MustCompile(`(?i)([?&](?:access_token|token)=)[^&#]*`)

// ScrubURLCredentials applies ScrubURLUserinfo to every scheme-form URL in
// text (git's or the forge client's message, which can quote a remote URL
// carrying a token: `https://user:token@host/...`, `https://token@host/...`),
// and empties the value of an access_token or token query parameter
// (`https://host/...?access_token=...`), so none reaches what abcd prints.
func ScrubURLCredentials(text string) string {
	if !strings.Contains(text, "://") {
		return text
	}
	return urlInTextRe.ReplaceAllStringFunc(text, func(u string) string {
		return queryTokenRe.ReplaceAllString(ScrubURLUserinfo(u), "${1}")
	})
}
