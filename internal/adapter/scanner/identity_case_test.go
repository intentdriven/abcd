package scanner

import "testing"

// TestOtherHomePathIsDetectedCaseInsensitively is the last member of the
// case-fold sweep this file's identity matchers went through
// (iss-2609251544556874). m.email, m.github, m.homeSelf and m.localBare fold,
// and githubRemoteRe/noreplyRe/noreplyLoginRe carry (?i) by construction;
// genericHomeRe did not, and it is the sole detector for home_path_other.
//
// macOS and Windows filesystems fold case, so /USERS/<name>/notes.md names
// exactly the file /Users/<name>/notes.md names. A path written either way is a
// working path to another person's home directory, and the case-sensitive
// matcher saw only one of them: the write-time redactors that route through
// ScanText left the other in the committed record and the privacy lint
// reported it clean.
func TestOtherHomePathIsDetectedCaseInsensitively(t *testing.T) {
	id := Identity{
		GitUserName:       "Zed Q Eight",
		GitUserEmail:      "zq8@example.test",
		GitRemoteUsername: "zq8handle",
		HomePath:          "/Users/zq8home", // abcd-audit:allow
		HomeUser:          "zq8home",
	}
	for _, path := range []string{
		"/Users/othername/notes.md", // abcd-audit:allow
		"/USERS/othername/notes.md", // abcd-audit:allow
		"/users/othername/notes.md", // abcd-audit:allow
		"/home/othername/notes.md",  // abcd-audit:allow
		"/Home/othername/notes.md",  // abcd-audit:allow
		"/HOME/othername/notes.md",  // abcd-audit:allow
	} {
		text := "see " + path + " for the draft\n"
		var got bool
		for _, f := range ScanText(text, id, DefaultPatterns(), DefaultIdentitySeverities(), "t") {
			if f.Kind == kindHomeOther {
				got = true
			}
		}
		if !got {
			t.Errorf("%s was not detected as %s; on a case-folding filesystem it names the same directory as its canonical spelling", path, kindHomeOther)
		}
	}
}

// TestOtherHomePathFoldingDoesNotSwallowOrdinaryProse is the negative side: a
// folded matcher must not start firing on text that is not a path, and the
// macOS system directories stay exempt whatever their case. A guard proved
// only against forbidden input may simply refuse everything.
func TestOtherHomePathFoldingDoesNotSwallowOrdinaryProse(t *testing.T) {
	id := Identity{HomePath: "/Users/zq8home", HomeUser: "zq8home"} // abcd-audit:allow
	for _, text := range []string{
		"the users of this tool\n",
		"welcome home, everyone\n",
		"a homeless value\n",
		"UsersAndGroups is a struct\n",
		"files under /users/shared/tmp are shared\n", // abcd-audit:allow
		"files under /USERS/Shared are shared\n",     // abcd-audit:allow
	} {
		for _, f := range ScanText(text, id, DefaultPatterns(), DefaultIdentitySeverities(), "t") {
			if f.Kind == kindHomeOther {
				t.Errorf("%q produced a spurious %s finding (matched %q)", text, kindHomeOther, f.Matched)
			}
		}
	}
}
