package scanner

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/gittest"
)

// identity_scopes_test.go — GHSA-v826-5jf4-p8xg / GHSA-gxhr-pmwv-r99p /
// GHSA-rvhr-3455-c5jw: ProbeIdentity read the ONE identity git resolves in
// the repository, so a repo-local or includeIf persona REPLACED the caller's
// global identity in the matcher set and the other identity was stored in
// clear text. A persona must ADD an identity to redact, never displace one.

// threeScopeRepo builds a caller with three identities: an unconditional
// global one, an includeIf persona keyed on the repository's parent
// directory (nothing in .git/config — git resolves it from the global file
// because of where the repository sits), and a repo-local one.
func threeScopeRepo(t *testing.T) (repo string, global, persona, local gittest.Person) {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}
	home := t.TempDir()
	t.Setenv("HOME", home)
	// git matches gitdir: against the REAL path of the .git dir, and a macOS
	// temp dir sits behind a /var -> /private/var symlink.
	work, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	repo = filepath.Join(work, "repo")
	if err := os.MkdirAll(repo, 0o755); err != nil {
		t.Fatal(err)
	}
	global, local = gittest.SplitIdentity(t, repo)

	persona = gittest.Person{Name: "Include Persona", Email: "persona@include.example"}
	inc := filepath.Join(home, "work.inc")
	if err := os.WriteFile(inc, []byte("[user]\n\tname = "+persona.Name+"\n\temail = "+persona.Email+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	f, err := os.OpenFile(os.Getenv("GIT_CONFIG_GLOBAL"), os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.WriteString("[includeIf \"gitdir:" + work + "/\"]\n\tpath = " + inc + "\n"); err != nil {
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
	return repo, global, persona, local
}

// TestProbeIdentityUnionsEveryScope pins the probe's shape: the effective
// (repo-local) identity in the name/email fields, and the two it displaced
// — the global one and the includeIf persona — in the Other* fields.
func TestProbeIdentityUnionsEveryScope(t *testing.T) {
	repo, global, persona, local := threeScopeRepo(t)
	id := ProbeIdentity(repo)
	if id.GitUserEmail != local.Email || id.GitUserName != local.Name {
		t.Errorf("effective identity = %q <%s>, want the repo-local %q <%s>", id.GitUserName, id.GitUserEmail, local.Name, local.Email)
	}
	for _, p := range []gittest.Person{global, persona} {
		if !containsFold(id.OtherGitUserEmails, p.Email) {
			t.Errorf("OtherGitUserEmails %v lacks %q", id.OtherGitUserEmails, p.Email)
		}
		if !containsFold(id.OtherGitUserNames, p.Name) {
			t.Errorf("OtherGitUserNames %v lacks %q", id.OtherGitUserNames, p.Name)
		}
	}
	if containsFold(id.OtherGitUserEmails, local.Email) || containsFold(id.OtherGitUserNames, local.Name) {
		t.Errorf("the effective identity is listed among the others: %v / %v", id.OtherGitUserNames, id.OtherGitUserEmails)
	}
}

// TestScannerRedactsEveryGitIdentityScope proves the per-repo scanner flags
// all three identities, so a text naming any of them is redacted.
func TestScannerRedactsEveryGitIdentityScope(t *testing.T) {
	repo, global, persona, local := threeScopeRepo(t)
	sc, err := New(repo)
	if err != nil {
		t.Fatal(err)
	}
	for _, p := range []gittest.Person{global, persona, local} {
		text := "mail " + p.Email + " and ask for " + p.Name + " by name"
		findings := sc.ScanText(text, "t")
		if !hasKind(findings, kindRealEmail) {
			t.Errorf("%s: email %q not flagged as real_email: %v", p.Name, p.Email, findings)
		}
		if !hasKind(findings, kindRealName) {
			t.Errorf("%s: name %q not flagged as real_name: %v", p.Name, p.Name, findings)
		}
		out, _ := Redact(text, findings)
		if strings.Contains(out, p.Email) || strings.Contains(out, p.Name) {
			t.Errorf("%s survives redaction: %q", p.Name, out)
		}
	}
}

// TestOtherIdentitiesArmMatchersAndHandleStaysPublic: the Other* fields arm
// the email and name matchers alongside the effective values, and a name
// among them that is the caller's public handle is still reported as
// github_username, never promoted to a hard-fail real_name (iss-283 holds
// for every scope's name, not only the effective one).
func TestOtherIdentitiesArmMatchersAndHandleStaysPublic(t *testing.T) {
	id := Identity{
		GitUserName: "Work Persona", GitUserEmail: "work@corp.example",
		OtherGitUserNames:  []string{"Personal Name", "octopat"},
		OtherGitUserEmails: []string{"personal@private.example"},
		GitRemoteUsername:  "octopat",
	}
	text := "Personal Name <personal@private.example>, Work Persona <work@corp.example>, octopat"
	findings := ScanText(text, id, DefaultPatterns(), DefaultIdentitySeverities(), "t")
	var emails, names, handles int
	for _, f := range findings {
		switch f.Kind {
		case kindRealEmail:
			emails++
		case kindRealName:
			names++
			if strings.EqualFold(f.Matched, "octopat") {
				t.Errorf("the public handle was promoted to real_name: %+v", f)
			}
		case kindGithubUser:
			handles++
		}
	}
	if emails != 2 || names != 2 || handles != 1 {
		t.Errorf("got %d real_email, %d real_name, %d github_username; want 2, 2, 1: %v", emails, names, handles, kindsOf(findings))
	}
	out, _ := Redact(text, findings)
	for _, v := range []string{"Personal Name", "personal@private.example", "Work Persona", "work@corp.example"} {
		if strings.Contains(out, v) {
			t.Errorf("%q survives redaction: %q", v, out)
		}
	}
}

// TestShortIdentityValuesDoNotArmTheEmailMatcher: the name matcher drops a
// value under three runes, the email matcher dropped nothing. A one-letter or
// truncated value from ANY scope — a placeholder in a CI config, a fragment
// left by splitting a value with an embedded newline — compiled straight into
// the alternation, and every "e" in the text then scanned as the caller's
// hard-fail real_email. A value must look like an address to arm the matcher.
func TestShortIdentityValuesDoNotArmTheEmailMatcher(t *testing.T) {
	id := Identity{
		GitUserName:        "Real Name",
		GitUserEmail:       "real@example.com",
		OtherGitUserEmails: []string{"e", "a@b", "@example.com", "user@", "  "},
	}
	text := "There are seven e characters here, and a@b, and user@ on their own."
	findings := ScanText(text, id, DefaultPatterns(), DefaultIdentitySeverities(), "t")
	for _, f := range findings {
		if f.Kind == kindRealEmail {
			t.Errorf("a value too short to be an address armed real_email: matched %q", f.Matched)
		}
	}
	out, _ := Redact(text, findings)
	if out != text {
		t.Errorf("prose was redacted on the strength of a one-letter identity value:\ngot  %q\nwant %q", out, text)
	}

	// The real address still fires: the guard drops the implausible values, not
	// the matcher.
	real := "write to real@example.com about it"
	rf := ScanText(real, id, DefaultPatterns(), DefaultIdentitySeverities(), "t")
	if !hasKind(rf, kindRealEmail) {
		t.Errorf("the caller's own address no longer arms real_email: %v", kindsOf(rf))
	}
	if ro, _ := Redact(real, rf); strings.Contains(ro, "real@example.com") {
		t.Errorf("the caller's own address survives redaction: %q", ro)
	}
}

// TestNoreplyLoginInAnotherScopeDoesNotDisarmRealName: the public-handle rule
// is what keeps a GitHub login out of the hard-fail real_name class (iss-283),
// and it reads a name against an ADDRESS. Once the probe unions every scope,
// letting any scope's noreply login answer for any scope's name disarms
// real_name on a value the single-identity probe redacted: a global
// "<id>+<login>@users.noreply.github.com" made the repo-local user.name equal
// to that login a "public handle", and with an org-owned remote the
// github_username matcher — built from the remote's owner alone — does not
// cover it either, so nothing redacted the name at all.
//
// The noreply provision therefore stays where git asserts the pairing: the
// effective name against the effective address. Every other name is a public
// handle only when it equals the remote's owner.
func TestNoreplyLoginInAnotherScopeDoesNotDisarmRealName(t *testing.T) {
	id := Identity{
		GitUserName:        "zq8handle",          // repo-local, the effective name
		GitUserEmail:       "work@corp.example",  // repo-local address: not a noreply
		OtherGitUserNames:  []string{"Octo Pat"}, // the global name it displaced
		OtherGitUserEmails: []string{"77722411+zq8handle@users.noreply.github.com"},
		GitRemoteUsername:  "some-org", // an org-owned remote: not the caller
	}
	text := "Reviewed-by: zq8handle"
	findings := ScanText(text, id, DefaultPatterns(), DefaultIdentitySeverities(), "t")
	if !hasKind(findings, kindRealName) {
		t.Errorf("a noreply login in ANOTHER scope disarmed real_name for the effective name: %v", kindsOf(findings))
	}
	if out, _ := Redact(text, findings); strings.Contains(out, "zq8handle") {
		t.Errorf("the effective user.name is redacted by nothing: %q", out)
	}
}
