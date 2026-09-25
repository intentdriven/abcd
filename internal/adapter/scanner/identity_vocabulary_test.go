package scanner

import (
	"strings"
	"testing"
)

// identity_vocabulary_test.go — iss-236 and iss-2609061504302157: an account
// name that is ordinary product vocabulary. A developer machine whose account is
// "dev" hard-failed the launch payload on the documented `--dev` flag, and the
// forge's hosted machines, whose account is "runner", hard-failed it on the noun;
// the capture redactor rewrote the same words in prose to [redacted-user]. A
// generic account name identifies no person, so the bare word is not a leak.
// What still is: the name standing where an account name stands — under a home
// root, after a tilde, as an address's local part — which is where a real home
// path or login leaks.

func TestLocalUsernameGenericAccountNameIsNotAFindingInProse(t *testing.T) {
	pats := DefaultPatterns()
	sev := DefaultIdentitySeverities()
	cases := map[string][]string{
		"dev": {
			"install with --dev to track the latest build",
			"the source checkout's dev build prints its version",
			"the -dev flag is the short form",
			"the historical dev-sync verb",
			"DEV builds are unsigned",
		},
		"runner": {
			"in CI the runner checks out the repository",
			"the hosted Runner image carries git",
			"a runner-scoped cache",
		},
	}
	for user, lines := range cases {
		id := Identity{HomeUser: user}
		for _, line := range lines {
			if got := ScanText(line, id, pats, sev, "f"); hasKind(got, kindLocalUser) {
				t.Errorf("account %q: ordinary vocabulary in %q flagged as the local username: %+v", user, line, got)
			}
			red, n := Redact(line, ScanText(line, id, pats, sev, "f"))
			if n != 0 || red != line {
				t.Errorf("account %q: redactor rewrote ordinary vocabulary %q -> %q", user, line, red)
			}
		}
	}
}

func TestLocalUsernameGenericAccountNameStillCaughtWhereAnAccountStands(t *testing.T) {
	pats := DefaultPatterns()
	sev := DefaultIdentitySeverities()
	cases := map[string][]string{
		"dev": {
			"backup written to /home/dev/data",   // abcd-audit:allow
			"see /Users/dev/notes.md",            // abcd-audit:allow
			`copied to C:\Users\dev\Desktop`,     // abcd-audit:allow
			"the store is ~dev/.abcd",            // tilde-user
			"ssh dev@buildhost.example.com",      // login@host
			"mail dev.smith@example.com",         // address local part
			"transcripts under -Users-dev-proj/", // abcd-audit:allow
		},
		"runner": {
			"work tree at /home/runner/work/repo", // abcd-audit:allow
			"log in as runner@ci.example.com",
		},
	}
	for user, lines := range cases {
		id := Identity{HomeUser: user}
		for _, line := range lines {
			got := ScanText(line, id, pats, sev, "f")
			if !hasKind(got, kindLocalUser) {
				t.Errorf("account %q: the account name where an account stands in %q was not flagged: %+v", user, line, got)
				continue
			}
			for _, f := range got {
				if f.Kind == kindLocalUser && f.Severity != SeverityHardFail {
					t.Errorf("account %q: %q flagged at %s, want hard_fail", user, line, f.Severity)
				}
			}
			if red, _ := Redact(line, got); strings.Contains(strings.ToLower(red), "/"+user+"/") {
				t.Errorf("account %q: the home segment survived redaction: %q", user, red)
			}
		}
	}
}

// A generic account name is still reported where it closes the caller's own
// home literal, whatever precedes it: under HOME=/root a blob-shaped
// "…0/root/deck.key" is declined by home_path_self's leading anchor, and the
// username rule is the one that catches it (TestBytesAndTextAgreeOnAnAlnumPrecededHome
// in the launch package pins the payload half).
func TestLocalUsernameGenericAccountNameStillCaughtInTheHomeLiteral(t *testing.T) {
	id := Identity{HomePath: "/root", HomeUser: "root"}
	line := "tEXtCreator0/root/deck.key"
	f := ScanText(line, id, DefaultPatterns(), DefaultIdentitySeverities(), "f")
	if !hasKind(f, kindLocalUser) {
		t.Errorf("the generic account name closing the caller's home literal was not reported: %+v", f)
	}
	// The same word elsewhere is still vocabulary.
	if f := ScanText("the repository root is clean", id, DefaultPatterns(), DefaultIdentitySeverities(), "f"); hasKind(f, kindLocalUser) {
		t.Errorf("ordinary vocabulary reported under HOME=/root: %+v", f)
	}
}

// A name that is not generic keeps the whole-word rule: a bare mention in prose
// is the caller's login and a hard_fail, and so is a flag-shaped one, because
// "--<name>" is also how a message is signed.
func TestLocalUsernameSpecificAccountNameKeepsTheWordRule(t *testing.T) {
	pats := DefaultPatterns()
	sev := DefaultIdentitySeverities()
	id := Identity{HomeUser: "zq8home"}
	for _, line := range []string{
		"last commit authored by zq8home",
		"thanks for the review --zq8home",
		"the zq8home build",
	} {
		if got := ScanText(line, id, pats, sev, "f"); !hasKind(got, kindLocalUser) {
			t.Errorf("specific account name in %q not flagged (false negative): %+v", line, got)
		}
	}
}

// The finding names where the matched value came from, so a reader can tell a
// collision from a leak without reopening the scanner.
func TestLocalUsernameFindingNamesItsIdentitySource(t *testing.T) {
	id := Identity{HomeUser: "zq8home"}
	got := ScanText("authored by zq8home", id, DefaultPatterns(), DefaultIdentitySeverities(), "f")
	for _, f := range got {
		if f.Kind == kindLocalUser {
			if !strings.Contains(f.Suggested, "$HOME") {
				t.Errorf("local_username suggestion does not name its source ($HOME): %q", f.Suggested)
			}
			return
		}
	}
	t.Fatalf("no local_username finding: %+v", got)
}

func TestGenericAccountNameFloor(t *testing.T) {
	for _, name := range []string{"dev", "DEV", "runner", "root", "ubuntu", "ec2-user", "me", "a"} {
		if !isGenericAccountName(name) {
			t.Errorf("%q should be under the generic-account floor", name)
		}
	}
	for _, name := range []string{"zq8home", "alice", "colleague", ""} { // abcd-audit:allow
		if isGenericAccountName(name) {
			t.Errorf("%q should not be under the generic-account floor", name)
		}
	}
}

// A JSON encoder doubles every backslash, so a Windows path quoted in a
// transcript line reaches the redactor as C:\\Users\\<login>\\…; the account
// name stands there exactly as it does in the single-backslash spelling
// (iss-2609251543293588). The home-literal clause holds for a home written
// with backslashes in either spelling.
func TestLocalUsernameGenericAccountNameCaughtUnderAJSONEscapedWindowsRoot(t *testing.T) {
	pats := DefaultPatterns()
	sev := DefaultIdentitySeverities()
	for _, tc := range []struct {
		id   Identity
		line string
	}{
		{Identity{HomeUser: "dev"}, `{"cwd":"C:\\Users\\dev\\Desktop"}`},               // abcd-audit:allow
		{Identity{HomeUser: "dev"}, `"path": "c:\\users\\dev"`},                        // abcd-audit:allow
		{Identity{HomePath: `D:\build\dev`, HomeUser: "dev"}, `"D:\\build\\dev\\out"`}, // abcd-audit:allow
	} {
		got := ScanText(tc.line, tc.id, pats, sev, "f")
		if !hasKind(got, kindLocalUser) {
			t.Errorf("the account name in %q was not flagged: %+v", tc.line, got)
			continue
		}
		if red, _ := Redact(tc.line, got); strings.Contains(strings.ToLower(red), `\dev`) {
			t.Errorf("the account name survived redaction: %q -> %q", tc.line, red)
		}
	}
	// A doubled separator before an ordinary word is still vocabulary.
	if got := ScanText(`"msg": "see docs\\dev notes"`, Identity{HomeUser: "dev"}, pats, sev, "f"); hasKind(got, kindLocalUser) {
		t.Errorf("a word after an escaped separator was flagged as the account name: %+v", got)
	}
}

// A tool result that is itself JSON text (go env -json, any --json output)
// already carries C:\\Users\\<login>, and the transcript line serialises that
// string again, so the separators arrive quadrupled; a third encoding doubles
// them once more. Every depth is the same account position, for the root
// prefix and for the caller's own home literal alike (iss-2609251638574543).
func TestLocalUsernameGenericAccountNameCaughtUnderAMultiplyEscapedWindowsRoot(t *testing.T) {
	pats := DefaultPatterns()
	sev := DefaultIdentitySeverities()
	winHome := Identity{HomePath: `C:\Users\dev`, HomeUser: "dev"} // abcd-audit:allow
	for _, tc := range []struct {
		id   Identity
		line string
	}{
		{winHome, `C:\\\\Users\\\\dev\\\\Desktop`},                                    // abcd-audit:allow
		{Identity{HomeUser: "dev"}, `x "C:\\\\Users\\\\dev\\\\go\\\\pkg\\\\mod"`},     // abcd-audit:allow
		{Identity{HomeUser: "dev"}, `C:\\\\\\\\Users\\\\\\\\dev\\\\\\\\AppData`},      // abcd-audit:allow
		{Identity{HomeUser: "dev"}, `c:\\\\users\\dev`},                               // abcd-audit:allow
		{Identity{HomePath: `D:\build\dev`, HomeUser: "dev"}, `"D:\\\\build\\\\dev"`}, // abcd-audit:allow
	} {
		got := ScanText(tc.line, tc.id, pats, sev, "f")
		if !hasKind(got, kindLocalUser) {
			t.Errorf("the account name in %q was not flagged: %+v", tc.line, got)
			continue
		}
		if red, _ := Redact(tc.line, got); strings.Contains(strings.ToLower(red), `\dev`) {
			t.Errorf("the account name survived redaction: %q -> %q", tc.line, red)
		}
	}
	// An escaped separator before an ordinary word is still vocabulary at any
	// depth.
	if got := ScanText(`"msg": "see docs\\\\dev notes"`, Identity{HomeUser: "dev"}, pats, sev, "f"); hasKind(got, kindLocalUser) {
		t.Errorf("a word after a quadrupled separator was flagged as the account name: %+v", got)
	}
}

// The positions a shell session or a config dump puts a login in are account
// positions too (iss-2609251547568307): a home root with no leading slash, a
// key/value pair naming the user, and the argument of a command that takes an
// account. A generic login there is reported; the same word elsewhere stays
// vocabulary.
func TestLocalUsernameGenericAccountNameCaughtInShellAndConfigPositions(t *testing.T) {
	pats := DefaultPatterns()
	sev := DefaultIdentitySeverities()
	id := Identity{HomeUser: "dev"}
	for _, line := range []string{
		"Users/dev/Desktop/notes.txt",      // abcd-audit:allow
		"x  Users/dev/Library/Caches/a.db", // abcd-audit:allow
		`"home/dev/.config/app.toml"`,      // abcd-audit:allow
		"USER=dev",
		"LOGNAME=dev",
		"export USER='dev'",
		"username: dev",
		"login: dev",
		`{"user": "dev", "shell": "/bin/zsh"}`,
		"docker run --user=dev image",
		"su - dev",
		"su dev",
		"sudo su -l dev",
		"chown dev notes.txt",
		"chown dev:staff notes.txt",
		"chown -R dev:staff build/",
		// A column-aligned dump pads the value out past a single space.
		"USER=" + strings.Repeat(" ", 10) + "dev",
		"username:" + strings.Repeat(" ", 20) + "dev",
		"su" + strings.Repeat(" ", 12) + "dev",
	} {
		got := ScanText(line, id, pats, sev, "f")
		if !hasKind(got, kindLocalUser) {
			t.Errorf("the account name in %q was not flagged: %+v", line, got)
		}
	}
	for _, line := range []string{
		"the user dev builds are unsigned",
		"a sudo dev build",
		"chown the dev tree later",
		"superuser: developer",
		"dev: the source build",
		"su" + strings.Repeat(" ", 2*maxKeyGap) + "dev", // a gap past the walk's bound
		"USER" + strings.Repeat(" ", 2*maxKeyGap) + "= dev",
	} {
		if got := ScanText(line, id, pats, sev, "f"); hasKind(got, kindLocalUser) {
			t.Errorf("ordinary vocabulary in %q flagged as the account name: %+v", line, got)
		}
	}
}

// Under a single-segment home such as /root, a deeper segment of an absolute
// path that happens to share the home's name is a directory named root, not
// the caller's home (iss-2609251551533470); the blob-shaped literal the clause
// exists for is still caught.
func TestLocalUsernameRootHomeSpareADeeperAbsoluteSegment(t *testing.T) {
	pats := DefaultPatterns()
	sev := DefaultIdentitySeverities()
	id := Identity{HomePath: "/root", HomeUser: "root"}
	for _, line := range []string{
		"mounted at /sys/fs/cgroup/root",
		"layer /var/lib/docker/overlay2/4f2a/root/etc",
		"the superuser's home on macOS is /var/root",
	} {
		if got := ScanText(line, id, pats, sev, "f"); hasKind(got, kindLocalUser) {
			t.Errorf("a directory named root in %q was flagged as the account name: %+v", line, got)
		}
	}
	for _, line := range []string{
		"tEXtCreator0/root/deck.key",
		"see build/root/deck.key",
	} {
		if got := ScanText(line, id, pats, sev, "f"); !hasKind(got, kindLocalUser) {
			t.Errorf("the home literal in %q was not flagged: %+v", line, got)
		}
	}
}
