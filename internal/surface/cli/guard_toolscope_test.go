package cli

import (
	"os"
	"strings"
	"testing"
)

// guardToolReachClaim is the clause every guard surface carries to state the
// guard's standing reach: the manifest hands the hook the shell tool and the
// question tool and nothing else, so a call through any other tool is never
// seen by it, and no warning marks that absence (iss-2609091955574760). It is
// compared through flatten, so it is written lower-case.
const guardToolReachClaim = "any other tool never reaches the guard"

// TestEveryGuardSurfaceStatesItsToolReach holds the reader's half of the
// manifest's matcher: TestGuardHookIsInstalledForBashCalls pins what the guard
// is asked about, and this pins that every surface describing the guard says
// so — the live `guard hook` help, and each file in guardScopeSurfaces. In the
// brief the claim must sit in the Fail-open-loud section, beside the states
// that can be false, because the limit is a standing scope rather than a
// degradation and a reader of that section would otherwise take its
// enumerated states as the only ways coverage is absent.
func TestEveryGuardSurfaceStatesItsToolReach(t *testing.T) {
	root := NewRootCommand()
	hook, _, err := root.Find([]string{"guard", "hook"})
	if err != nil || hook.Name() != "hook" {
		t.Fatalf("guard hook is not reachable from the command tree: %v", err)
	}
	if !strings.Contains(flatten(hook.Long), guardToolReachClaim) {
		t.Errorf("guard hook --help does not state the guard's tool reach: missing %q", guardToolReachClaim)
	}
	for _, path := range guardScopeSurfaces {
		body, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("cannot read guard surface %s: %v", path, err)
		}
		text := string(body)
		if strings.HasSuffix(path, "17-guard.md") {
			start := strings.Index(text, "\n## Fail-open-loud\n")
			if start < 0 {
				t.Fatalf("%s has no Fail-open-loud section", path)
			}
			text = text[start+1:]
			if end := strings.Index(text[3:], "\n## "); end >= 0 {
				text = text[:3+end]
			}
		}
		if !strings.Contains(flatten(text), guardToolReachClaim) {
			t.Errorf("%s does not state the guard's tool reach: missing %q", path, guardToolReachClaim)
		}
	}
}
