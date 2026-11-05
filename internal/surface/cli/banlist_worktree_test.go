package cli

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/gittest"
)

// linkedWorktree seeds a commit in repo and adds a linked worktree, returning its
// root. It is the shape itd-150 exists for: one repository, two working trees, and
// a gitignored local-ephemeral tier that belongs to whichever one you stand in.
func linkedWorktree(t *testing.T, repo *gittest.Repo) string {
	t.Helper()
	repo.Git("add", "-A")
	repo.Git("-c", "core.hooksPath=/dev/null", "commit", "-m", "seed")
	dir := filepath.Join(t.TempDir(), "linked")
	repo.Git("worktree", "add", "-b", "linked", dir)
	return dir
}

// TestBanlistRendersTheLayerALinkedWorktreeInherits is the CLI half of itd-150's
// parity requirement. The guard resolves the primary checkout's store and enforces
// it inside a linked worktree; a status render that showed only the (absent)
// worktree-local store would announce "INACTIVE on this machine" about a checkout
// whose every commit is being checked — the board disagreeing with the guard about
// what is protected.
func TestBanlistRendersTheLayerALinkedWorktreeInherits(t *testing.T) {
	hermeticEnv(t)
	repo := newRepo(t)
	// The tier's fence first: the private store is only written where git already
	// ignores it, and the linked worktree inherits the fence through the commit.
	repo.Write(".gitignore", ".abcd/.work.local/\n")
	t.Chdir(repo.Root())
	if _, err := runCLIErr(t, "banlist", "add", "--private", "widget-partner", "widgetworks"); err != nil {
		t.Fatalf("banlist add: %v", err)
	}
	linked := linkedWorktree(t, repo)
	t.Chdir(linked)

	out := string(runCLI(t, "banlist"))
	if !strings.Contains(out, "inherited from the primary checkout") {
		t.Fatalf("the worktree render does not name the inherited layer:\n%s", out)
	}
	if !strings.Contains(out, "widget-partner") {
		t.Errorf("the inherited layer's key is not rendered:\n%s", out)
	}
	if strings.Contains(out, "widgetworks") {
		t.Errorf("the render leaks the pattern; private entries render by key only:\n%s", out)
	}

	// `list --private` is the same read scoped to one layer, so it must agree.
	scoped := string(runCLI(t, "banlist", "list", "--private"))
	if !strings.Contains(scoped, "inherited from the primary checkout") {
		t.Errorf("`banlist list --private` omits the inherited layer:\n%s", scoped)
	}
}

// TestBanlistJSONCarriesTheInheritedLayerOnlyInAWorktree pins both halves of the
// envelope contract: a linked worktree carries the inherited layer, and a
// standalone checkout's JSON is exactly what it always was — the field is absent,
// not present-and-null, so no consumer has to learn a new shape it will never see.
func TestBanlistJSONCarriesTheInheritedLayerOnlyInAWorktree(t *testing.T) {
	hermeticEnv(t)
	repo := newRepo(t)
	// The tier's fence first: the private store is only written where git already
	// ignores it, and the linked worktree inherits the fence through the commit.
	repo.Write(".gitignore", ".abcd/.work.local/\n")
	t.Chdir(repo.Root())
	if _, err := runCLIErr(t, "banlist", "add", "--private", "widget-partner", "widgetworks"); err != nil {
		t.Fatalf("banlist add: %v", err)
	}

	var standalone map[string]any
	if err := json.Unmarshal(runCLI(t, "--json", "banlist"), &standalone); err != nil {
		t.Fatalf("standalone JSON: %v", err)
	}
	if _, ok := standalone["inherited"]; ok {
		t.Errorf("a standalone checkout's envelope carries an inherited layer: %v", standalone["inherited"])
	}

	linked := linkedWorktree(t, repo)
	t.Chdir(linked)
	var worktree map[string]any
	if err := json.Unmarshal(runCLI(t, "--json", "banlist"), &worktree); err != nil {
		t.Fatalf("worktree JSON: %v", err)
	}
	inh, ok := worktree["inherited"].(map[string]any)
	if !ok {
		t.Fatalf("a linked worktree's envelope carries no inherited layer: %v", worktree)
	}
	// The layer names the STORE, never the checkout: `private.path` is repo-relative,
	// and a `primary_root` would be the primary checkout's absolute path — whose
	// directory name is very often the private name that very store bans.
	priv, ok := inh["private"].(map[string]any)
	if !ok || priv["path"] == "" || priv["path"] == nil {
		t.Errorf("the inherited layer does not name the store it read: %v", inh)
	}
	if !strings.Contains(string(runCLI(t, "--json", "banlist")), "widget-partner") {
		t.Errorf("the inherited layer's key is missing from the envelope")
	}
}

// TestBanlistNeverRendersThePrimaryCheckoutsPath is the Go twin of the shell
// guard's own TestPreCommitHook_NeverPrintsThePrimaryCheckoutsPath, and it holds
// this front door to the same contract for the same reason.
//
// A checkout's directory name is very often the private name its own store bans —
// a project codename is the commonest entry there is — and this layer's whole
// contract is that no pattern value reaches output. The inherited layer renders on
// the SUCCESS path of an ordinary status read, so an absolute path there prints the
// banned string to stdout, into scrollback, into an agent transcript, into any CI
// log that captures it, and into whatever file the caller redirected `--json` to.
// The remedy a reader needs is "the primary checkout's store", which is the same
// sentence without the leak.
func TestBanlistNeverRendersThePrimaryCheckoutsPath(t *testing.T) {
	hermeticEnv(t)
	env := gittest.Env(t)
	git := func(dir string, args ...string) {
		t.Helper()
		cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
		cmd.Env = env
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v in %s: %v\n%s", args, dir, err, out)
		}
	}
	// The primary checkout is NAMED for the very string its own store bans: the
	// hazard is not the path, it is the directory name inside it.
	const codename = "widgetworks-platform"
	primary := filepath.Join(t.TempDir(), codename)
	if err := os.MkdirAll(primary, 0o755); err != nil {
		t.Fatal(err)
	}
	git(primary, "init", "--initial-branch=main")
	git(primary, "config", "user.name", "Alice Example")
	git(primary, "config", "user.email", "alice@example.com")
	if err := os.WriteFile(filepath.Join(primary, ".gitignore"), []byte(".abcd/.work.local/\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Chdir(primary)
	if _, err := runCLIErr(t, "banlist", "add", "--private", "widget-partner", "widgetworks"); err != nil {
		t.Fatalf("banlist add: %v", err)
	}
	git(primary, "add", "-A")
	git(primary, "-c", "core.hooksPath=/dev/null", "commit", "-m", "seed")
	linked := filepath.Join(t.TempDir(), "linked")
	git(primary, "worktree", "add", "-b", "linked", linked)
	t.Chdir(linked)

	renders := map[string]string{
		"text `banlist`":                string(runCLI(t, "banlist")),
		"text `banlist list --private`": string(runCLI(t, "banlist", "list", "--private")),
		"json `banlist`":                string(runCLI(t, "--json", "banlist")),
		"json `banlist list --private`": string(runCLI(t, "--json", "banlist", "list", "--private")),
	}
	for name, out := range renders {
		if strings.Contains(out, primary) {
			t.Errorf("%s prints the primary checkout's absolute path, whose directory name may itself be a banned string:\n%s", name, out)
		}
		if strings.Contains(out, codename) {
			t.Errorf("%s prints the primary checkout's directory name, which here IS the string the store bans:\n%s", name, out)
		}
		// The layer must still be VISIBLE — withholding the path must not withhold the
		// fact that these entries are enforced here, which is the parity itd-150 exists
		// for. In text the words say it; in JSON the `inherited` key does.
		marker := "primary checkout"
		if strings.HasPrefix(name, "json") {
			marker = `"inherited"`
		}
		if !strings.Contains(out, marker) {
			t.Errorf("%s does not say the layer was inherited, so the remedy is invisible:\n%s", name, out)
		}
		if !strings.Contains(out, "widget-partner") {
			t.Errorf("%s omits the inherited entry's key:\n%s", name, out)
		}
	}

	var envelope map[string]any
	if err := json.Unmarshal(runCLI(t, "--json", "banlist"), &envelope); err != nil {
		t.Fatalf("worktree JSON: %v", err)
	}
	inh, ok := envelope["inherited"].(map[string]any)
	if !ok {
		t.Fatalf("a linked worktree's envelope carries no inherited layer: %v", envelope)
	}
	if v, ok := inh["primary_root"]; ok {
		t.Errorf("the envelope still carries primary_root = %v; the field is the leak, not a formatting choice", v)
	}
}
