package scanner

import (
	"strings"
	"testing"
)

// own_slug_test.go — iss-2608270645473170: the github_username rule is armed
// with the owner of the repository's own remote, and it masked that owner in
// the repository's own owner/repo slug ("marketplace add <owner>/<repo>" in the
// README, quoted into a shipped intent six times). The slug names the very
// repository the record is committed to, so it is public by construction.
func TestGithubUsernameSparesTheRepositorysOwnSlug(t *testing.T) {
	id := Identity{GitRemoteUsername: "acme", GitRemoteRepo: "tool"}
	pats, sev := DefaultPatterns(), DefaultIdentitySeverities()
	for _, line := range []string{
		"/plugin marketplace add acme/tool",
		"clone acme/tool.git and build",
		"see Acme/Tool for the source",
	} {
		f := ScanText(line, id, pats, sev, "f")
		if hasKind(f, kindGithubUser) {
			t.Errorf("the repository's own slug was reported: %q %+v", line, f)
		}
		if red, _ := Redact(line, f); red != line {
			t.Errorf("the repository's own slug was rewritten: %q -> %q", line, red)
		}
	}
	// The owner alone, or as the owner of another repository, is still the
	// handle the rule exists for.
	for _, line := range []string{
		"reviewed by acme yesterday",
		"forked acme/toolbox",
		"acme/other has the fix",
	} {
		f := ScanText(line, id, pats, sev, "f")
		if !hasKind(f, kindGithubUser) {
			t.Errorf("the owner outside the repository's own slug was not reported: %q", line)
		}
		if red, _ := Redact(line, f); strings.Contains(red, "acme") {
			t.Errorf("the owner survived redaction: %q", red)
		}
	}
}

func TestParseGitHubRemote(t *testing.T) {
	cases := map[string][2]string{
		"https://github.com/acme/tool.git": {"acme", "tool"},
		"https://github.com/acme/tool":     {"acme", "tool"},
		"git@github.com:acme/tool.git":     {"acme", "tool"},
		"ssh://git@GitHub.com/acme/tool/":  {"acme", "tool"},
		"https://example.com/acme/tool":    {"", ""},
	}
	for url, want := range cases {
		owner, repo := parseGitHubRemote(url)
		if owner != want[0] || repo != want[1] {
			t.Errorf("parseGitHubRemote(%q) = %q, %q; want %q, %q", url, owner, repo, want[0], want[1])
		}
	}
}
