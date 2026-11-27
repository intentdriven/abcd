package scanner

import (
	"path/filepath"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/intentdriven/abcd/internal/abcdhome"
	"github.com/intentdriven/abcd/internal/termsafe"
)

// notUTF8Case is one identity value that is not valid UTF-8: the setting the
// refusal must name, where the invalid byte sits, and the value.
type notUTF8Case struct {
	setting, at, value string
}

// notUTF8Values is every git identity value and the account name, invalid at
// the start, in the middle and at the end.
func notUTF8Values() map[string][]notUTF8Case {
	return map[string][]notUTF8Case{
		"user.name": {
			{"user.name", "start", "\xe9orvan Quill"},
			{"user.name", "middle", "Zorv\xe9n Quill"},
			{"user.name", "end", "Zorvan Quill\xe9"},
		},
		"user.email": {
			{"user.email", "start", "\xe9orvan@example.com"},
			{"user.email", "middle", "zorv\xe9n@example.com"},
			{"user.email", "end", "zorvan@example.co\xe9"},
		},
		"remote.origin.url": {
			{"remote.origin.url", "start", "\xe9orvan"},
			{"remote.origin.url", "middle", "zorv\xe9n"},
			{"remote.origin.url", "end", "zorvan\xe9"},
		},
		"the home folder's name": {
			{"the home folder's name", "start", "\xe9orvan"},
			{"the home folder's name", "middle", "zorv\xe9n"},
			{"the home folder's name", "end", "zorvan\xe9"},
		},
	}
}

// checkReason holds a refusal's reason to naming the setting and never the
// value, in any form.
func checkReason(t *testing.T, c notUTF8Case, reason string) {
	t.Helper()
	if !strings.Contains(reason, c.setting) {
		t.Errorf("%s invalid at the %s: the reason does not name %q: %q", c.setting, c.at, c.setting, reason)
	}
	if !strings.Contains(reason, "UTF-8") {
		t.Errorf("%s invalid at the %s: the reason does not say to re-save it as UTF-8: %q", c.setting, c.at, reason)
	}
	low := strings.ToLower(reason)
	if strings.Contains(low, "orv") || strings.Contains(low, "quill") || !utf8.ValidString(reason) ||
		strings.ContainsRune(reason, utf8.RuneError) {
		t.Errorf("%s invalid at the %s: the reason prints the value: %+q", c.setting, c.at, reason)
	}
}

// An identity value that is not valid UTF-8 makes the scanner unavailable, so
// every store that consults Unavailable before a write refuses it, with a
// reason naming the setting to re-save and never its value.
func TestIdentityValueThatIsNotUTF8MakesTheScannerUnavailable(t *testing.T) {
	set := map[string]func(*Identity, string){
		"user.name":              func(id *Identity, v string) { id.GitUserName = v },
		"user.email":             func(id *Identity, v string) { id.GitUserEmail = v },
		"remote.origin.url":      func(id *Identity, v string) { id.GitRemoteUsername = v },
		"the home folder's name": func(id *Identity, v string) { id.HomeUser = v },
	}
	for key, cases := range notUTF8Values() {
		for _, c := range cases {
			sc, err := New(t.TempDir())
			if err != nil {
				t.Fatal(err)
			}
			sc.identity = Identity{}
			set[key](&sc.identity, c.value)
			bad, reason := sc.Unavailable()
			if !bad {
				t.Errorf("%s invalid at the %s: the scanner stayed available", c.setting, c.at)
				continue
			}
			checkReason(t, c, reason)
		}
	}
	for _, c := range []struct {
		setting string
		id      Identity
	}{
		{"user.name", Identity{GitUserName: "Zorvan Quill", OtherGitUserNames: []string{"Zorv\xe9n Quill"}}},
		{"user.email", Identity{GitUserEmail: "zorvan@example.com", OtherGitUserEmails: []string{"zorv\xe9n@example.com"}}},
	} {
		sc, err := New(t.TempDir())
		if err != nil {
			t.Fatal(err)
		}
		sc.identity = c.id
		bad, reason := sc.Unavailable()
		if !bad {
			t.Errorf("another scope's %s that is not UTF-8 left the scanner available", c.setting)
			continue
		}
		checkReason(t, notUTF8Case{c.setting, "middle", "Zorv\xe9n Quill"}, reason)
	}
}

// configure writes one identity value where the probe reads it: the global
// git config for a git value, HOME for the account name.
func configure(t *testing.T, c *pinCtx, setting, value string) {
	t.Helper()
	switch setting {
	case "user.name":
		appendFile(t, c.global, "[user]\n\tname = "+value+"\n")
	case "user.email":
		appendFile(t, c.global, "[user]\n\temail = "+value+"\n")
	case "remote.origin.url":
		appendFile(t, c.global, "[remote \"origin\"]\n\turl = git@github.com:"+value+"/notes.git\n")
	case "the home folder's name":
		t.Setenv("HOME", filepath.Join(c.home, value))
	}
	t.Setenv(abcdhome.StageHomeEnv, "")
}

// The scanner entry refuses while any identity value is not UTF-8: the text
// comes back described, never echoed, whatever form of the value it holds.
func TestRedactRefusalRefusesWhileAnIdentityValueIsNotUTF8(t *testing.T) {
	for _, cases := range notUTF8Values() {
		for _, c := range cases {
			pc := newPinCtx(t, false)
			configure(t, pc, c.setting, c.value)
			text := "refused for " + strings.ToValidUTF8(c.value, "�") + " and " + strings.ToValidUTF8(c.value, "")
			if got, want := RedactRefusal(pc.repo, text), termsafe.DescribeRefused(text); got != want {
				t.Errorf("%s invalid at the %s: RedactRefusal = %+q, want %q", c.setting, c.at, got, want)
			}
			sc, err := New(pc.repo)
			if err != nil {
				t.Fatal(err)
			}
			bad, reason := sc.Unavailable()
			if !bad {
				t.Errorf("%s invalid at the %s: the probed scanner stayed available", c.setting, c.at)
				continue
			}
			checkReason(t, c, reason)
		}
	}
}

// An identity that is valid UTF-8, non-ASCII included, leaves the scanner
// available and is matched as before.
func TestIdentityThatIsUTF8IsMatchedAsBefore(t *testing.T) {
	pc := newPinCtx(t, false)
	configure(t, pc, "user.name", "Zorvan Quillé")
	configure(t, pc, "user.email", "zorvan.quille@example.com")
	configure(t, pc, "remote.origin.url", "zorvan-q")
	configure(t, pc, "the home folder's name", "zorvané")
	sc, err := New(pc.repo)
	if err != nil {
		t.Fatal(err)
	}
	if bad, reason := sc.Unavailable(); bad {
		t.Fatalf("a UTF-8 identity made the scanner unavailable: %s", reason)
	}
	text := "Zorvan Quillé mailed zorvan.quille@example.com, see github.com/zorvan-q now, signed by zorvané today"
	found := map[string]bool{}
	for _, f := range sc.ScanText(text, "f") {
		found[f.Kind] = true
	}
	for _, k := range []string{kindRealName, kindRealEmail, kindGithubUser, kindLocalUser} {
		if !found[k] {
			t.Errorf("%s not found in %q", k, text)
		}
	}
	got := RedactRefusal(pc.repo, text)
	if got == termsafe.DescribeRefused(text) || strings.Contains(got, "Zorvan Quillé") || strings.Contains(got, "zorvan.quille@") {
		t.Errorf("RedactRefusal = %q, want the identity redacted in place", got)
	}
}
