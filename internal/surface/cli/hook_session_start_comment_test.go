package cli

import (
	"os"
	"strings"
	"testing"
)

// sessionStartHookComment returns the comment block that sits directly above
// the session-start hook's registration in cli.go, as one line of prose: the
// comment markers dropped and the line breaks folded to spaces.
func sessionStartHookComment(t *testing.T) string {
	t.Helper()
	src, err := os.ReadFile("cli.go")
	if err != nil {
		t.Fatalf("read cli.go: %v", err)
	}
	lines := strings.Split(string(src), "\n")
	use := -1
	for i, l := range lines {
		if strings.TrimSpace(l) == `Use:   "session-start",` {
			if use != -1 {
				t.Fatalf("cli.go registers session-start twice, at lines %d and %d", use+1, i+1)
			}
			use = i
		}
	}
	if use < 1 {
		t.Fatal(`cli.go has no line reading Use:   "session-start",`)
	}
	if got := strings.TrimSpace(lines[use-1]); got != "hookCmd.AddCommand(&cobra.Command{" {
		t.Fatalf("line %d above the session-start Use is %q, not the hook's registration", use, got)
	}
	var words []string
	for i := use - 2; i >= 0; i-- {
		l := strings.TrimSpace(lines[i])
		if !strings.HasPrefix(l, "//") {
			break
		}
		words = append(strings.Fields(strings.TrimPrefix(l, "//")), words...)
	}
	if len(words) == 0 {
		t.Fatal("the session-start hook's registration has no comment above it")
	}
	return strings.Join(words, " ")
}

// TestSessionStartHookCommentStatesItsExitAndChannels holds the comment above
// the session-start hook to what the hook does (iss-2610161125118921): it
// exits 0 on every path, its notices go to stderr, and stdout carries a
// constant count line. The comment said the loud path deliberately exits 2,
// which the code stopped doing at iss-2608241115201044. The behaviour itself
// is asserted by the hook's own tests; this reads the prose only.
func TestSessionStartHookCommentStatesItsExitAndChannels(t *testing.T) {
	comment := sessionStartHookComment(t)
	for _, stale := range []string{"exits 2", "exit 2", "non-zero exit"} {
		if strings.Contains(comment, stale) {
			t.Errorf("the comment above the session-start hook says %q; the hook exits 0 on every path", stale)
		}
	}
	for _, want := range []string{
		"exits 0 on every path",
		"notices on stderr",
		"constant count line on stdout",
	} {
		if !strings.Contains(comment, want) {
			t.Errorf("the comment above the session-start hook does not say %q", want)
		}
	}
}
