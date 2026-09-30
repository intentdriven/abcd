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
	id := Identity{GitRemoteUsername: "acme"}
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
	// The owner alone is still the handle the rule exists for.
	for _, line := range []string{
		"reviewed by acme yesterday",
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

// iss-2609291409053602: the remote's owner followed by '/' and ANY repository
// name is an owner/repo slug on the forge the remote names, and the owner is
// public there by the same argument that spares the repository's own slug. A
// record of acme/tool that names acme/sibling (a repository it publishes to)
// kept losing the handle to the github_username rule.
func TestGithubUsernameSparesTheOwnersSiblingSlug(t *testing.T) {
	id := Identity{GitRemoteUsername: "acme"}
	pats, sev := DefaultPatterns(), DefaultIdentitySeverities()
	for _, line := range []string{
		"publishes to acme/sibling-repo on every release",
		"forked acme/toolbox",
		"acme/other has the fix",
		"see Acme/Sibling_Two.site for the pages",
		"the pages live in acme/acme.github.io.",
		"(acme/sibling)",
		"`acme/sibling`",
		"acme/sibling",
		"acme/sibling/blob/main/README.md",
		"go get github.com/acme/sibling@v1",
		"GitHub.com/acme/sibling",
		"gist.github.com/acme/sibling",
		"gh api repos/acme/sibling/pulls",
		"git@github.com:acme/sibling.git and acme/other",
	} {
		f := ScanText(line, id, pats, sev, "f")
		if hasKind(f, kindGithubUser) {
			t.Errorf("the owner in a sibling repository's slug was reported: %q %+v", line, f)
		}
		if red, _ := Redact(line, f); red != line {
			t.Errorf("the owner in a sibling repository's slug was rewritten: %q -> %q", line, red)
		}
	}
}

// The sibling exemption is anchored on both sides. The owner must be a whole
// owner segment (not the tail of another owner or of a dotted name), it must be
// followed by '/' and a name GitHub could hold, the name must end where a
// GitHub name ends, and a host written before it must be the forge's: on
// another host the same spelling names an account there, and sparing it would
// link the two. These rules hold for the repository's own slug too, because
// both are one rule.
func TestGithubUsernameSlugExemptionIsAnchored(t *testing.T) {
	id := Identity{GitRemoteUsername: "acme"}
	pats, sev := DefaultPatterns(), DefaultIdentitySeverities()
	long := strings.Repeat("r", 101)
	for _, line := range []string{
		"reviewed by acme yesterday",
		"acme/ is the org",
		"acme/",
		"acme/. is not a repository",
		"acme/.. is not a repository",
		"not-acme/tool",
		"x.acme/tool",
		"gitlab.example.com/acme/tool",
		"gitlab.example.com/acme/sibling",
		"example.org:acme/sibling",
		"notgithub.com/acme/sibling",
		"github.com.example.net/acme/sibling",
		"acme/sib\u00e9",
		"acme/sib\u00e9 and more",
		"acme/" + long,
	} {
		f := ScanText(line, id, pats, sev, "f")
		if !hasKind(f, kindGithubUser) {
			t.Errorf("the owner outside an anchored owner/repo slug was not reported: %q", line)
		}
		if red, _ := Redact(line, f); strings.Contains(strings.ToLower(red), "acme") {
			t.Errorf("the owner survived redaction: %q", red)
		}
	}
}

// The host rule bounds the whole path the owner sits in, not only the segment
// next to it: a host other than GitHub's anywhere earlier in the same path
// token names an account on that host, whatever sits between the host and the
// owner ("groups/", "-/", "~", a backslash) and however its dots are spelled.
// On GitHub's own host the owner must stand where GitHub puts an owner:
// straight after the host, or after one of its account path words.
func TestGithubUsernameSlugExemptionJudgesTheWholePath(t *testing.T) {
	id := Identity{GitRemoteUsername: "acme"}
	pats, sev := DefaultPatterns(), DefaultIdentitySeverities()
	for _, line := range []string{
		"gitlab.example.com/groups/acme/tool",
		"gitlab.example.com/groups/sub/acme/tool",
		"gitlab.example.com/-/acme/tool",
		"gitlab.example.com/~acme/tool",
		"gitlab.example.com\\acme/tool",
		"gitlab．example．com/acme/tool",
		"gitlab。example。com/acme/tool",
		"gitlab｡example｡com/acme/tool",
		"gitlab․example․com/acme/tool",
		"x．acme/tool",
		"see gitlab.example.com/groups/acme/tool today",
		"(gitlab.example.com/groups/acme/tool)",
		"user@evil.example/acme/x",
		"evil.example:github.com/acme/x",
		"example.org:8080/acme/x",
		"github.com/someone/acme/x",
		"github.com/someone/repos/acme/x",
		strings.Repeat("d/", 1200) + "acme/tool",
	} {
		short := line
		if len(short) > 80 {
			short = "..." + short[len(short)-40:]
		}
		f := ScanText(line, id, pats, sev, "f")
		if !hasKind(f, kindGithubUser) {
			t.Errorf("the owner under another host's path was not reported: %q", short)
		}
		if red, _ := Redact(line, f); strings.Contains(strings.ToLower(red), "acme") {
			t.Errorf("the owner survived redaction: %q", short)
		}
	}
	for _, line := range []string{
		"github.com/acme/x",
		"github.com/orgs/acme/x",
		"github.com/users/acme/x",
		"api.github.com/repos/acme/x",
		"github.com:443/acme/x",
		"user@github.com/acme/x",
		"repos/acme/x",
		"src/vendor/acme/x",
		"./acme/x",
		"../vendor/acme/x",
		"acme/x",
		"acme/acme.github.io",
		"see gitlab.example.com and acme/tool",
		"(github.com/acme/x)",
	} {
		f := ScanText(line, id, pats, sev, "f")
		if hasKind(f, kindGithubUser) {
			t.Errorf("the owner in a forge slug was reported: %q %+v", line, f)
		}
	}
}

// The byte scan runs the same identity matcher with a narrower Identity; a
// repository that raised github_username for its bytes must see the slug rule
// hold there exactly as on text, or renaming notes.md to notes.pdf changes
// the verdict on the same slug.
func TestSlugExemptionHoldsOnBytesWhenRaised(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, ".abcd/config/pii.json", `{"identity_severities":{"github_username":"hard_fail"}}`)
	body := "%PDF-1.4\n/Subject (see acme/tool and acme/sibling)\n"
	md := writeFile(t, root, "notes.md", body)
	pdf := writeFile(t, root, "notes.pdf", body)
	sc, err := New(root)
	if err != nil {
		t.Fatal(err)
	}
	if bad, why := sc.Unavailable(); bad {
		t.Fatalf("override must load: %s", why)
	}
	sc.identity = Identity{GitRemoteUsername: "acme"}
	text := scanOne(t, sc, "notes.md", md)
	bin := scanOne(t, sc, "notes.pdf", pdf)
	if hasKind(text.Findings, kindGithubUser) {
		t.Fatalf("control: text spares the owner in a slug: %+v", text.Findings)
	}
	if hasKind(bin.Findings, kindGithubUser) {
		t.Errorf("bytes reported the owner in a slug text spares: %+v", bin.Findings)
	}
}

func TestParseGitHubRemote(t *testing.T) {
	cases := map[string]string{
		"https://github.com/acme/tool.git": "acme",
		"https://github.com/acme/tool":     "acme",
		"git@github.com:acme/tool.git":     "acme",
		"ssh://git@GitHub.com/acme/tool/":  "acme",
		"https://github.com/acme/":         "acme",
		"https://example.com/acme/tool":    "",
	}
	for url, want := range cases {
		if owner := parseGitHubRemote(url); owner != want {
			t.Errorf("parseGitHubRemote(%q) = %q; want %q", url, owner, want)
		}
	}
}

// U+FE52 SMALL FULL STOP folds to '.' under NFKC and UTS 46 maps it to one,
// exactly as U+FF0E and U+2024 do, so a segment it separates reads as another
// host and the owner under it stays a finding.
func TestGithubUsernameSlugExemptionReadsTheSmallFullStopAsADot(t *testing.T) {
	id := Identity{GitRemoteUsername: "acme"}
	pats, sev := DefaultPatterns(), DefaultIdentitySeverities()
	for _, line := range []string{
		"gitlab﹒example﹒com/acme/tool",
		"evil﹒example/acme/x",
		"see evil﹒example/acme/x today",
	} {
		f := ScanText(line, id, pats, sev, "f")
		if !hasKind(f, kindGithubUser) {
			t.Errorf("the owner under a small-full-stop host was not reported: %q", line)
		}
		if red, _ := Redact(line, f); strings.Contains(strings.ToLower(red), "acme") {
			t.Errorf("the owner survived redaction: %q", line)
		}
	}
}
