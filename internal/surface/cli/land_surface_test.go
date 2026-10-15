package cli

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/landing"
	"github.com/intentdriven/abcd/internal/gittest"
)

// The `abcd land` front door (iss-2610150524235929, iss-2610150524246898): it
// reaches the core, says which layer turned an act off, exits 2 on a refusal,
// and its --json carries the steps. The core's steps are proved in
// internal/core/landing; these tests prove the verb drives it.

// landWorld is a checkout on the branch feat/x with a bare origin under a temp
// HOME, its seams set so no forge client, network or preflight runs.
func landWorld(t *testing.T) (dir, origin string) {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("ABCD_PLUGIN_ROOT", "")
	t.Setenv("CLAUDE_PLUGIN_ROOT", "")
	origin = filepath.Join(home, "origin.git")
	gittest.InitBare(t, origin)
	dir = filepath.Join(home, "work")
	gitInitAt(t, dir)
	for _, kv := range [][2]string{{"user.name", "Pat Example"}, {"user.email", "pat@example.com"}, {"commit.gpgsign", "false"}} {
		gitCmd(t, dir, "config", kv[0], kv[1])
	}
	writeRel(t, dir, ".gitignore", ".abcd/.work.local/\n")
	gitCmd(t, dir, "add", "-A")
	gitCommit(t, dir, "commit", "-q", "-m", "base")
	gitCmd(t, dir, "remote", "add", "origin", origin)
	gitCmd(t, dir, "push", "-q", "origin", "main")
	gitCmd(t, dir, "fetch", "-q", "origin")
	gitCmd(t, dir, "checkout", "-q", "-b", "feat/x")
	writeRel(t, dir, "change.txt", "a change\n")
	gitCmd(t, dir, "add", "-A")
	gitCommit(t, dir, "commit", "-q", "-m", "change")

	prev := landSeams
	t.Cleanup(func() { landSeams = prev })
	landSeams = landing.Options{
		Forge: func(string, ...string) (string, error) { return "", errors.New("no forge in this test") },
		Gate: func(d string) error {
			head := strings.TrimSpace(gitCmd(t, d, "rev-parse", "HEAD"))
			p := filepath.Join(d, filepath.FromSlash(landing.ReceiptsRelDir), head)
			if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
				return err
			}
			return os.WriteFile(p, nil, 0o644)
		},
	}
	return dir, origin
}

// With both acts off the verb gates and says, for each act, the layer that
// turned it off; the push step prints the command.
func TestLandWithBothActsOffGatesAndNamesTheFlags(t *testing.T) {
	dir, origin := landWorld(t)
	code, stdout, stderr := reserveRun(t, dir, "land", "--no-push", "--no-enqueue")
	if code != 0 {
		t.Fatalf("exit %d, stderr %q", code, stderr)
	}
	for _, want := range []string{"feat/x", "gate", "land.push is off (flag layer, --no-push)", "land.enqueue is off (flag layer, --no-enqueue)",
		"run: git push origin refs/heads/feat/x:refs/heads/feat/x"} {
		if !strings.Contains(stdout, want) {
			t.Errorf("stdout lacks %q:\n%s", want, stdout)
		}
	}
	if tip := gitCmd(t, origin, "for-each-ref", "refs/heads/feat/x"); tip != "" {
		t.Fatalf("origin holds feat/x (%s); nothing should be pushed", tip)
	}
}

func TestLandJSONCarriesTheSteps(t *testing.T) {
	dir, _ := landWorld(t)
	code, stdout, stderr := reserveRun(t, dir, "land", "--no-push", "--no-enqueue", "--json")
	if code != 0 {
		t.Fatalf("exit %d, stderr %q", code, stderr)
	}
	var doc struct {
		Branch string `json:"branch"`
		Push   struct {
			On    bool   `json:"on"`
			Layer string `json:"layer"`
		} `json:"push"`
		Steps []struct {
			Name, Status, Command string
		} `json:"steps"`
	}
	if err := json.Unmarshal([]byte(stdout), &doc); err != nil {
		t.Fatalf("stdout is not one JSON document: %v\n%s", err, stdout)
	}
	if doc.Branch != "feat/x" || doc.Push.On || doc.Push.Layer != "flag" || len(doc.Steps) != 3 {
		t.Fatalf("doc %+v", doc)
	}
	if strings.Contains(stdout, os.Getenv("HOME")) {
		t.Fatalf("the JSON carries the home path:\n%s", stdout)
	}
}

func TestLandOnTheDefaultBranchExitsTwoAndPushesNothing(t *testing.T) {
	dir, _ := landWorld(t)
	gitCmd(t, dir, "checkout", "-q", "main")
	code, _, stderr := reserveRun(t, dir, "land")
	if code != 2 || !strings.Contains(stderr, "default branch") {
		t.Fatalf("exit %d, stderr %q: want 2 naming the default branch", code, stderr)
	}
}

// A refusal after a step names what was done before it.
func TestLandRefusalNamesTheStepsBeforeIt(t *testing.T) {
	dir, _ := landWorld(t)
	code, _, stderr := reserveRun(t, dir, "land", "--no-enqueue")
	if code != 2 || !strings.Contains(stderr, "no forge in this test") || !strings.Contains(stderr, "before it, gate:") {
		t.Fatalf("exit %d, stderr %q: want the forge's refusal after the gate", code, stderr)
	}
}

func TestLandTakesAtMostOneBranch(t *testing.T) {
	dir, _ := landWorld(t)
	code, _, stderr := reserveRun(t, dir, "land", "a", "b")
	if code != 2 || !strings.Contains(stderr, "at most one branch") {
		t.Fatalf("exit %d, stderr %q", code, stderr)
	}
}
