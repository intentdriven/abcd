package cli

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/intentdriven/abcd/internal/core/decisions"
)

// refusal_raisers_decide_test.go — the probes for the codes of decide and
// ideate (TestRaisedByIsWhatTheVerbsRaise, spc-2610301431108175 step 3).

var _ = registerRaiserProbes("decide", decideRaiserProbes)

// decideRaiserProbes builds a checkout with no shared tier, one with it, one
// with no commit, a shallow clone and a directory outside any checkout, and
// returns the probes of decide and ideate.
func decideRaiserProbes(t *testing.T) []raiserProbe {
	t.Setenv("HOME", t.TempDir())
	tierless := t.TempDir()
	gitInitAt(t, tierless)
	tiered := ideateRepo(t)
	if err := os.MkdirAll(filepath.Join(tiered, filepath.FromSlash(decisions.SharedTierRel), "decisions"), 0o755); err != nil {
		t.Fatal(err)
	}
	unborn := t.TempDir()
	gitInitAt(t, unborn)
	// A shallow clone of a history of two commits holds the tip alone.
	full := t.TempDir()
	gitInitAt(t, full)
	gitCommitAt(t, full, "root")
	gitCommitAt(t, full, "second")
	shallow := filepath.Join(t.TempDir(), "shallow")
	gitCmd(t, filepath.Dir(shallow), "clone", "-q", "--depth", "1", "file://"+full, shallow)
	verdict := filepath.Join(t.TempDir(), "absent.json")
	outside := inDir(t.TempDir())
	return []raiserProbe{
		probe("decide/outside-checkout", "decide", "we keep the parser as it stands").from(outside),
		probe("ideate/outside-checkout", "ideate", "record", "an-idea", "--verdict-json", verdict),
		probe("decide/missing-shared-tier", "decide", "we keep the parser as it stands").from(inDir(tierless)),
		probe("decide/invalid-text", "decide", "!!! ???").from(inDir(tiered)),
		probe("decide/invalid-text", "decide", "--adr", "!!! ???"),
		probe("usage/extra-argument", "decide", "one decision", "and another"),
		probe("usage/unknown-command", "decide", "list"),
		probe("usage/missing-argument", "ideate", "record", "--verdict-json", verdict),
		probe("usage/flag-conflict", "ideate", "record", "an-idea", "--private", "--public", "--verdict-json", verdict),
		probe("ideate/invalid-verdict", "ideate", "record", "an-idea", "--verdict-json", verdict),
		probe("ideate/missing-root-commit", "ideate", "record", "an-idea", "--verdict-json", verdict).from(inDir(unborn)),
		probe("ideate/shallow-checkout", "ideate", "record", "an-idea", "--verdict-json", verdict).from(inDir(shallow)),
	}
}
