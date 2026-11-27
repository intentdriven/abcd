package scanner

import "testing"

func kinds(text string, id Identity) map[string]bool {
	out := map[string]bool{}
	for _, f := range ScanText(text, id, nil, DefaultIdentitySeverities(), "f") {
		out[f.Kind] = true
	}
	return out
}

// The git identity is matched in both Unicode normal forms: a name typed
// composed in git config reaches a transcript decomposed as often as not.
func TestGitIdentityIsMatchedInBothNormalForms(t *testing.T) {
	const nfc, nfd = "José Núñez", "José Núñez"
	for _, c := range []struct{ value, text string }{{nfc, nfd}, {nfd, nfc}} {
		if !kinds("written by "+c.text+" today", Identity{GitUserName: c.value})[kindRealName] {
			t.Errorf("user.name %+q not found in %+q", c.value, c.text)
		}
	}
	const enfc, enfd = "josé@example.com", "josé@example.com"
	for _, c := range []struct{ value, text string }{{enfc, enfd}, {enfd, enfc}} {
		if !kinds("mail "+c.text+" now", Identity{GitUserEmail: c.value})[kindRealEmail] {
			t.Errorf("user.email %+q not found in %+q", c.value, c.text)
		}
	}
	const gnfc, gnfd = "josé-dev", "josé-dev"
	for _, c := range []struct{ value, text string }{{gnfc, gnfd}, {gnfd, gnfc}} {
		if !kinds("see github.com/"+c.text+" now", Identity{GitRemoteUsername: c.value})[kindGithubUser] {
			t.Errorf("remote owner %+q not found in %+q", c.value, c.text)
		}
	}
}

// A git config value that is not valid UTF-8 cannot be held in a pattern; the
// scan leaves it out instead of panicking, and the scanner refuses every store
// write while it is set (TestIdentityValueThatIsNotUTF8MakesTheScannerUnavailable).
func TestGitIdentityThatIsNotUTF8DoesNotPanicTheScan(t *testing.T) {
	id := Identity{GitUserName: "Jos\xe9 Nunez", GitUserEmail: "jos\xe9@example.com", GitRemoteUsername: "jos\xe9-dev"}
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("ScanText panicked on a git identity that is not UTF-8: %v", r)
		}
	}()
	kinds("by Jos\xe9 Nunez, mail jos\xe9@example.com, github.com/jos\xe9-dev", id)
}
