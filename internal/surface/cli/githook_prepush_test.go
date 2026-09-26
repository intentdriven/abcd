package cli

import (
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/gittest"
)

// The committed pre-push hook and the preflight receipt it reads
// (scripts/preflight-receipt.sh). Two records meet here:
//
//   - iss-2608290810036869: git opens a push's connection before it runs
//     pre-push, so a preflight run inside the hook held the connection for its
//     whole length, and one outlasted the idle timeout — the push reported
//     success and moved nothing. The gate runs BEFORE the push (ruling M16,
//     2026-09-23): `make preflight` mints a receipt, and the hook only checks it.
//   - iss-2608210738378295: the gates read the working tree while CI reads the
//     commit, so a staged/unstaged divergence passed locally and failed CI. A
//     receipt is minted only when the tree matched HEAD for the whole run.
//
// The fixture is a throwaway clone carrying copies of the committed hook and
// script, with a bare repository as its remote, and a Makefile whose preflight
// only leaves a marker: the hook must never run it.

type prePushCase struct {
	t      *testing.T
	dir    string // the working clone
	origin string // its bare remote
	env    []string
}

func newPrePushCase(t *testing.T) *prePushCase {
	t.Helper()
	for _, tool := range []string{"bash", "git"} {
		if _, err := exec.LookPath(tool); err != nil {
			t.Skipf("%s unavailable", tool)
		}
	}
	top := exec.Command("git", "rev-parse", "--show-toplevel")
	top.Env = gittest.Env(t)
	out, err := top.Output()
	if err != nil {
		t.Skip("not in a git checkout")
	}
	root := strings.TrimSpace(string(out))

	base := t.TempDir()
	c := &prePushCase{t: t, dir: filepath.Join(base, "work"), origin: filepath.Join(base, "origin.git"), env: gittest.Env(t)}
	c.run(base, "git", "init", "-q", "--bare", "-b", "main", c.origin)
	c.run(base, "git", "init", "-q", "-b", "main", c.dir)
	c.git("config", "user.name", "Alice Example")
	c.git("config", "user.email", "alice@example.com")

	for _, rel := range []string{".githooks/pre-push", "scripts/preflight-receipt.sh"} {
		src, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(rel)))
		if err != nil {
			t.Fatalf("read the committed %s: %v", rel, err)
		}
		c.write(rel, string(src), 0o755)
	}
	c.write(".gitignore", ".abcd/.work.local/\n", 0o644)
	// A preflight that passes and leaves a marker. The hook must never run it: a
	// preflight inside pre-push is the connection-holding shape this replaces.
	c.write("Makefile", "preflight:\n\t@mkdir -p .abcd/.work.local && touch .abcd/.work.local/preflight-ran\n", 0o644)
	c.write("seed.md", "seed\n", 0o644)
	c.git("add", "-A")
	c.git("-c", "core.hooksPath=/dev/null", "commit", "-q", "-m", "seed")
	c.git("remote", "add", "origin", c.origin)
	c.git("-c", "core.hooksPath=/dev/null", "push", "-q", "origin", "main")
	c.git("config", "core.hooksPath", ".githooks")
	c.git("checkout", "-q", "-b", "feature")
	return c
}

func (c *prePushCase) run(dir string, name string, args ...string) (string, error) {
	c.t.Helper()
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	cmd.Env = c.env
	out, err := cmd.CombinedOutput()
	return string(out), err
}

func (c *prePushCase) git(args ...string) string {
	c.t.Helper()
	out, err := c.run(c.dir, "git", args...)
	if err != nil {
		c.t.Fatalf("git %v: %v\n%s", args, err, out)
	}
	return strings.TrimSpace(out)
}

func (c *prePushCase) write(rel, content string, mode os.FileMode) {
	c.t.Helper()
	p := filepath.Join(c.dir, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		c.t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(content), mode); err != nil {
		c.t.Fatal(err)
	}
}

func (c *prePushCase) commit(rel, content string) string {
	c.t.Helper()
	c.write(rel, content, 0o644)
	c.git("add", rel)
	c.git("commit", "-q", "-m", "add "+rel)
	return c.git("rev-parse", "HEAD")
}

// receipt runs the script's subcommands in the clone.
func (c *prePushCase) receipt(dir string, args ...string) (string, error) {
	c.t.Helper()
	return c.run(dir, "bash", append([]string{filepath.Join(dir, "scripts", "preflight-receipt.sh")}, args...)...)
}

// preflight stands in for `make preflight`: the state before, the gates, the mint
// after. The Makefile wiring that does the same is pinned separately below.
func (c *prePushCase) preflight(dir string) string {
	c.t.Helper()
	began, err := c.receipt(dir, "state")
	if err != nil {
		c.t.Fatalf("state: %v\n%s", err, began)
	}
	out, err := c.receipt(dir, "mint", strings.TrimSpace(began))
	if err != nil {
		c.t.Fatalf("mint: %v\n%s", err, out)
	}
	return out
}

func (c *prePushCase) push(args ...string) (string, error) {
	c.t.Helper()
	return c.run(c.dir, "git", append([]string{"push"}, args...)...)
}

// remoteTip is the commit the bare remote holds for a branch, or "".
func (c *prePushCase) remoteTip(branch string) string {
	c.t.Helper()
	out, err := c.run(c.origin, "git", "rev-parse", "--verify", "--quiet", "refs/heads/"+branch)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(out)
}

func TestPrePushRefusesANewCommitWithoutAReceiptAndNeverRunsThePreflight(t *testing.T) {
	c := newPrePushCase(t)
	c.commit("feature.md", "a feature\n")
	out, err := c.push("origin", "feature")
	if err == nil {
		t.Fatalf("a commit with no passing preflight was pushed\n%s", out)
	}
	if !strings.Contains(out, "no passing preflight") {
		t.Errorf("the refusal does not say what is missing\n%s", out)
	}
	if tip := c.remoteTip("feature"); tip != "" {
		t.Errorf("the remote moved to %s although the push was refused", tip)
	}
	if _, err := os.Stat(filepath.Join(c.dir, ".abcd", ".work.local", "preflight-ran")); err == nil {
		t.Errorf("the hook ran the preflight itself, which holds the push's connection open for its whole length")
	}
}

func TestPrePushPassesACommitWithAReceipt(t *testing.T) {
	c := newPrePushCase(t)
	head := c.commit("feature.md", "a feature\n")
	if out := c.preflight(c.dir); !strings.Contains(out, "receipt minted for "+head[:12]) {
		t.Fatalf("a preflight on a clean tree minted no receipt\n%s", out)
	}
	if out, err := c.push("origin", "feature"); err != nil {
		t.Fatalf("a commit with a passing preflight was refused\n%s", out)
	}
	if tip := c.remoteTip("feature"); tip != head {
		t.Errorf("the remote holds %q, want %s", tip, head)
	}
}

// The iss-2608210738378295 shape, reproduced: a staged rename committed while the
// follow-up edit to the renamed file stays unstaged. The working tree holds the
// edit and the commit does not, so a gate reading the tree passes what CI fails.
func TestPreflightReceiptIsWithheldFromATreeThatDiffersFromTheCommit(t *testing.T) {
	cases := map[string]func(c *prePushCase){
		"staged rename, unstaged edit": func(c *prePushCase) {
			c.commit("iss-1.md", "id: iss-1\n")
			c.git("mv", "iss-1.md", "iss-2.md")
			c.git("commit", "-q", "-m", "renumber")
			c.write("iss-2.md", "id: iss-2\n", 0o644) // the edit that never got staged
		},
		"staged change left uncommitted": func(c *prePushCase) {
			c.commit("a.md", "one\n")
			c.write("a.md", "two\n", 0o644)
			c.git("add", "a.md")
		},
		"untracked file": func(c *prePushCase) {
			c.commit("a.md", "one\n")
			c.write("b.md", "untracked\n", 0o644)
		},
	}
	for name, diverge := range cases {
		t.Run(name, func(t *testing.T) {
			c := newPrePushCase(t)
			diverge(c)
			out := c.preflight(c.dir)
			if strings.Contains(out, "receipt minted for") {
				t.Fatalf("a preflight on a tree that differs from HEAD minted a receipt\n%s", out)
			}
			if !strings.Contains(out, "no push receipt") {
				t.Errorf("the preflight does not say why no receipt was minted\n%s", out)
			}
			if pushed, err := c.push("origin", "feature"); err == nil {
				t.Fatalf("a commit whose tree diverged during its preflight was pushed\n%s", pushed)
			}
		})
	}
}

// git status hides a tracked file flagged skip-worktree or assume-unchanged: its
// local edit is invisible to the clean-tree comparison, so the gates read content
// the commit does not carry while the tree reads as clean. No receipt is minted
// while any such flag is set, edited or not, because the flag is exactly what
// stops the script from knowing.
func TestPreflightReceiptIsWithheldWhileAnIndexFlagHidesAnEdit(t *testing.T) {
	for name, flag := range map[string]string{
		"skip-worktree":    "--skip-worktree",
		"assume-unchanged": "--assume-unchanged",
	} {
		t.Run(name, func(t *testing.T) {
			c := newPrePushCase(t)
			c.commit("a.md", "one\n")
			c.git("update-index", flag, "a.md")
			c.write("a.md", "two\n", 0o644) // the edit git status no longer reports
			if st := c.git("status", "--porcelain"); st != "" {
				t.Fatalf("the fixture's premise is wrong: git status reports the edit\n%s", st)
			}
			out := c.preflight(c.dir)
			if strings.Contains(out, "receipt minted for") {
				t.Fatalf("a preflight over a %s edit minted a receipt\n%s", name, out)
			}
			if !strings.Contains(out, flag[2:]) {
				t.Errorf("the refusal does not name the flag that hides the edit\n%s", out)
			}
			if pushed, err := c.push("origin", "feature"); err == nil {
				t.Fatalf("a commit preflighted over a hidden edit was pushed\n%s", pushed)
			}
		})
	}
}

func TestPreflightReceiptIsWithheldWhenHeadMovesDuringTheRun(t *testing.T) {
	c := newPrePushCase(t)
	c.commit("a.md", "one\n")
	began, err := c.receipt(c.dir, "state")
	if err != nil {
		t.Fatal(err)
	}
	c.commit("b.md", "two\n") // a commit made while the gates were running
	out, err := c.receipt(c.dir, "mint", strings.TrimSpace(began))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(out, "receipt minted for") {
		t.Fatalf("a preflight during which HEAD moved minted a receipt\n%s", out)
	}
	if pushed, err := c.push("origin", "feature"); err == nil {
		t.Fatalf("a commit no preflight ran on was pushed\n%s", pushed)
	}
}

// A commit the remote already holds leaves nothing new: a tag on a merged commit
// needs no receipt, or cutting a release from the forge's merge would be refused.
func TestPrePushLetsACommitTheRemoteHoldsThroughWithoutAReceipt(t *testing.T) {
	c := newPrePushCase(t)
	c.git("fetch", "-q", "origin")
	c.git("tag", "-a", "v0.0.1", "-m", "v0.0.1", "origin/main")
	if out, err := c.push("origin", "v0.0.1"); err != nil {
		t.Fatalf("a tag on a commit the remote already holds was refused\n%s", out)
	}
}

// A receipt minted in a sibling worktree of the same repository counts: the same
// commit is the same tree everywhere, and a branch preflighted in its own worktree
// is often pushed from the primary checkout.
func TestPrePushHonoursAReceiptFromASiblingWorktree(t *testing.T) {
	c := newPrePushCase(t)
	sibling := filepath.Join(t.TempDir(), "sibling")
	c.git("worktree", "add", "-q", "-b", "lane", sibling)
	if err := os.WriteFile(filepath.Join(sibling, "lane.md"), []byte("lane\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if out, err := c.run(sibling, "git", "add", "lane.md"); err != nil {
		t.Fatal(out)
	}
	if out, err := c.run(sibling, "git", "commit", "-q", "-m", "lane"); err != nil {
		t.Fatal(out)
	}
	if out := c.preflight(sibling); !strings.Contains(out, "receipt minted for") {
		t.Fatalf("the sibling's clean preflight minted no receipt\n%s", out)
	}
	if out, err := c.push("origin", "lane"); err != nil {
		t.Fatalf("a commit preflighted in a sibling worktree was refused from the primary checkout\n%s", out)
	}
}

// The protected-branch refusal is unchanged, and it needs no receipt to fire.
func TestPrePushStillRefusesADirectPushToMain(t *testing.T) {
	c := newPrePushCase(t)
	c.git("checkout", "-q", "main")
	c.commit("direct.md", "direct\n")
	c.preflight(c.dir)
	out, err := c.push("origin", "main")
	if err == nil {
		t.Fatalf("a direct push to main was accepted\n%s", out)
	}
	if !strings.Contains(out, "protected branch") {
		t.Errorf("the refusal does not name the protected branch\n%s", out)
	}
}

// `make preflight` is what mints the receipt, so its recipe must record the tree's
// state before any gate runs and mint after the last one. Read through `make -n`,
// which prints the recipe the real target would run without running it.
func TestMakePreflightMintsTheReceiptAfterItsLastGate(t *testing.T) {
	if _, err := exec.LookPath("make"); err != nil {
		t.Skip("make unavailable")
	}
	top := exec.Command("git", "rev-parse", "--show-toplevel")
	top.Env = gittest.Env(t)
	out, err := top.Output()
	if err != nil {
		t.Skip("not in a git checkout")
	}
	cmd := exec.Command("make", "-n", "preflight")
	cmd.Dir = strings.TrimSpace(string(out))
	cmd.Env = gittest.Env(t)
	dry, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("make -n preflight: %v\n%s", err, dry)
	}
	lines := strings.Split(strings.TrimSpace(string(dry)), "\n")
	last := lines[len(lines)-1]
	mint := regexp.MustCompile(`^scripts/preflight-receipt\.sh mint "[0-9a-f]{40} (clean|dirty|hidden)"$`)
	if !mint.MatchString(last) {
		t.Fatalf("the preflight recipe's last step is %q; want the receipt minted from the state recorded "+
			"before the first gate ran\n%s", last, dry)
	}
	raceAt, mintAt := -1, len(lines)-1
	for i, l := range lines {
		if strings.HasPrefix(l, "go test -race") {
			raceAt = i
		}
	}
	if raceAt < 0 || raceAt > mintAt {
		t.Errorf("the receipt is not minted after the race-enabled tests\n%s", dry)
	}
}
