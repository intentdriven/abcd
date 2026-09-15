package intent

import (
	"strings"
	"testing"
)

// TestIntentRedactsHomeWithANameSuffix pins the intent redactor to the three
// continuation shapes main redacted and the trailing anchor let through:
// the caller's home followed by ".zip", "-old" and "_snapshot", behind a URL
// host and as a plain path. The intent store has no backstop either.
func TestIntentRedactsHomeWithANameSuffix(t *testing.T) {
	repo := t.TempDir()
	home := "/Users/zzhomeuser42" // abcd-audit:allow
	t.Setenv("HOME", home)
	for _, text := range []string{
		"the archive is at https://ci.example.com" + home + ".zip for review",
		"old copy under " + home + "-old/x here",
		"snapshot under /Volumes/T7" + home + "_snapshot/x here",
	} {
		out, _, err := redactIntentText(repo, text)
		if err != nil {
			t.Fatalf("redactIntentText: %v", err)
		}
		if strings.Contains(out, "zzhomeuser42") {
			t.Errorf("%q: the caller's name reached the intent text:\n%s", text, out)
		}
	}
}
