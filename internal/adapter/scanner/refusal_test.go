package scanner

import (
	"path/filepath"
	"strings"
	"testing"
)

// TestRedactRefusalKeepsTheNameAndDropsTheLeak: a decoder message naming an
// undeclared key keeps the key's readable part, and loses a planted home path
// and the caller's own home.
func TestRedactRefusalKeepsTheNameAndDropsTheLeak(t *testing.T) {
	home := filepath.Join(t.TempDir(), "zzcallerhome")
	t.Setenv("HOME", home)
	repo := t.TempDir()
	msg := `json: unknown field "reviewer_notes /Users/zzotherperson/notes" beside ` + home + "/secret\x1b[31m" // abcd-lint:allow — a planted home path the refusal must not echo
	got := RedactRefusal(repo, msg)
	for _, leak := range []string{"zzotherperson", "zzcallerhome", "\x1b"} {
		if strings.Contains(got, leak) {
			t.Errorf("RedactRefusal kept %q: %q", leak, got)
		}
	}
	if !strings.Contains(got, "reviewer_notes") {
		t.Errorf("RedactRefusal lost the field name the reader needs: %q", got)
	}
}

// TestRedactRefusalFailsClosed: a scanner that cannot be trusted leaves the text
// described, never echoed.
func TestRedactRefusalFailsClosed(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "absent")
	const msg = `json: unknown field "zzleak-7f3a"`
	got := RedactRefusal(missing, msg)
	if strings.Contains(got, "zzleak") {
		t.Errorf("a degraded scanner echoed the text: %q", got)
	}
	if !strings.Contains(got, "not quoted") {
		t.Errorf("a degraded scanner did not describe the text: %q", got)
	}
}
