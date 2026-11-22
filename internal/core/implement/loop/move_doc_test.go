package loop

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestBuildIntentSaysAnInvocationPerformsOneMove holds the build intent to the
// build chapter's vocabulary (BU1, iss-2609291313276243): a spec's steps and a
// lane's stages are two words, and what one invocation of the loop performs is
// one move, a stage of a lane, never "one step" (iss-2610262037301480). The
// host-facing verb name `implement step` stays.
func TestBuildIntentSaysAnInvocationPerformsOneMove(t *testing.T) {
	path := filepath.Join("..", "..", "..", "..", ".abcd", "development", "intents", "planned",
		"itd-2609201916151817-one-verb-takes-a-single-intent-from.md")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read the build intent: %v", err)
	}
	text := string(raw)
	for _, drift := range []string{"does one step", "drives it step by step"} {
		if strings.Contains(text, drift) {
			t.Errorf("%s says %q; one invocation performs one move (one stage of a lane), and a step is a spec's", path, drift)
		}
	}
	for _, want := range []string{"performs one move (one stage of a lane)", "one move per call"} {
		if !strings.Contains(text, want) {
			t.Errorf("%s does not say %q", path, want)
		}
	}
}
