package runner

// profile.go is the one place that answers "the profile this drive runs on"
// (spc-2610230813348813 step 1; adr-2610222222326104's amendment of
// 2026-11-04): the directory under ~/.abcd.noindex/profiles/ the person names,
// or the one directory there is, the launch's environment and its arguments.
// The person makes each directory and logs the harness into it once, in their
// own shell; abcd never reads, stores or forwards the login. Nothing else reads
// the directory, so itd-2611040344170206, whose choosing among several, registry
// and creating command go further, still replaces this file's one function and
// moves nothing the person logged in. Nothing here moves a drive between
// directories: the person's name is the whole of the choice.

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"syscall"

	"crypto/sha256"
	"encoding/hex"
	"github.com/intentdriven/abcd/internal/abcdhome"
	"github.com/intentdriven/abcd/internal/fsutil"
	"github.com/intentdriven/abcd/internal/termsafe"
	"slices"
)

// StageWritesEnv names where a walled launch's file tools may write.
const StageWritesEnv = "ABCD_STAGE_WRITES"

// The rest of what a walled launch's guard reads: the shell's home, what is
// protected from the stage, what its work declares it changes, and the commit
// its lane was cut from.
const (
	StageHomeEnv     = abcdhome.StageHomeEnv
	StageProtectEnv  = "ABCD_STAGE_PROTECT"
	StageDeclaredEnv = "ABCD_STAGE_DECLARED"
	StageBaseEnv     = "ABCD_STAGE_BASE"
)

// BudgetIntent, which a drive's refusals name, covers how a key reaches the harness
// and what it may spend.
const BudgetIntent = "itd-2611011230226453"

// profilesLeaf is the walled directories' folder below abcd's home.
const profilesLeaf = "profiles"

// The refusals DriveProfile names, through errors.Is.
var (
	ErrNoProfile       = errors.New("no profile directory")
	ErrSeveralProfiles = errors.New("more than one profile directory")
	ErrProfileName     = errors.New("no plain profile name")
)

// credentialVars are the variables the harness would bill through or sign in
// with instead of the walled login: never set for a walled launch.
var credentialVars = []string{
	"ANTHROPIC_API_KEY",
	"ANTHROPIC_AUTH_TOKEN",
	"ANTHROPIC_FOUNDRY_API_KEY",
	"AWS_BEARER_TOKEN_BEDROCK",
	"CLAUDE_CODE_OAUTH_TOKEN",
	"CLAUDE_CODE_USE_BEDROCK",
	"CLAUDE_CODE_USE_FOUNDRY",
	"CLAUDE_CODE_USE_VERTEX",
}

// HeldCredentials names each of credentialVars set in this process's
// environment. It reports the names alone and keeps no value.
func HeldCredentials() []string {
	var held []string
	for _, k := range credentialVars {
		if os.Getenv(k) != "" {
			held = append(held, k)
		}
	}
	return held
}

// Profile is what a drive's launch runs on.
type Profile struct {
	// Dir is the walled directory's absolute path.
	Dir string
	// StageRoot holds what a stage's shell reads: ShellHome is the home it runs
	// under, and EnvFile the file it sources before each command.
	StageRoot string
	ShellHome string
	EnvFile   string
	// PersonHome is the person's own home directory.
	PersonHome string
	// Repo is the checkout the drive runs in, whose git configuration names the
	// identity a stage commits under.
	Repo string
	// Ledger is the launch ledger the proof's launch is written to.
	Ledger *Ledger
	// Env is the launch's environment switches (sealedEnv): each KEY=value is
	// set over the parent's, and each bare name taken out.
	Env []string
	// Args are the launch's walled arguments.
	Args []string
}

// plainProfileName reports whether name is one path element a drive may run on: no
// way out of the profiles' folder or round to a sibling, no separator the
// environment's lists would split it on, and nothing a terminal would act on.
func plainProfileName(name string) bool {
	return name != "." && name != ".." && !strings.ContainsRune(name, filepath.Separator) &&
		!strings.ContainsRune(name, os.PathListSeparator) && termsafe.Sanitize(name) == name
}

// DriveProfile is the profile a drive runs on, below home, the person's home
// directory: the directory name names, or with name empty the one directory there
// is. It refuses a name that is no plain element or that the folder does not hold
// exactly, none, several with no name, and a directory it would not trust with the
// settings that decide what the drive's agents run. A refusal lists what the folder
// holds and never repeats the name, as abcd's other refusals of a typed value do not.
func DriveProfile(home, name string) (Profile, error) {
	root := abcdhome.Path(home, profilesLeaf)
	shown := abcdhome.Display(profilesLeaf) + "/"
	if name != "" && !plainProfileName(name) {
		return Profile{}, fmt.Errorf("%w: --profile takes one directory's name under %s, with no separator and no control character (the value given is not repeated here)", ErrProfileName, shown)
	}
	entries, err := os.ReadDir(root)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return Profile{}, fmt.Errorf("%s could not be read: %w", shown, err)
	}
	var names []string
	for _, e := range entries {
		if e.IsDir() || e.Type()&os.ModeSymlink != 0 {
			names = append(names, e.Name())
		}
	}
	sort.Strings(names)
	makeOne := fmt.Sprintf("make one and log the harness into it once, in your own shell: "+
		"`mkdir -p %sdrive && CLAUDE_CONFIG_DIR=%sdrive claude`, then the harness's own login step; abcd never logs a harness in", shown, shown)
	switch {
	case name != "" && slices.Contains(names, name):
	case name != "" && len(names) == 0:
		return Profile{}, fmt.Errorf("%w named by --profile under %s, which holds none: %s", ErrNoProfile, shown, makeOne)
	case name != "":
		return Profile{}, fmt.Errorf("%w named by --profile under %s (the value given is not repeated here), which holds %s", ErrNoProfile, shown, strings.Join(names, ", "))
	case len(names) == 0:
		return Profile{}, fmt.Errorf("%w under %s: %s", ErrNoProfile, shown, makeOne)
	case len(names) > 1:
		return Profile{}, fmt.Errorf("%w under %s (%s): name the one this drive runs on with --profile <name>; a drive never chooses among them or moves from one to another",
			ErrSeveralProfiles, shown, strings.Join(names, ", "))
	default:
		name = names[0]
	}
	dir := filepath.Join(root, name)
	fi, err := os.Lstat(dir)
	if err != nil {
		return Profile{}, fmt.Errorf("%s%s could not be read: %w", shown, name, err)
	}
	if !fi.IsDir() {
		return Profile{}, fmt.Errorf("%s%s is a link, not a directory of its own; a drive's settings are written only into a real directory", shown, name)
	}
	if st, ok := fi.Sys().(*syscall.Stat_t); ok && int(st.Uid) != os.Geteuid() {
		return Profile{}, fmt.Errorf("%s%s belongs to another account; a drive runs only on a directory this account owns", shown, name)
	}
	if fsutil.WritableByOthers(fi) {
		return Profile{}, fmt.Errorf("%s%s is one group or other can write; chmod go-w it", shown, name)
	}
	return Profile{
		Dir:        dir,
		PersonHome: home,
		Env: append([]string{
			"CLAUDE_CONFIG_DIR=" + dir,
			"CLAUDE_CODE_DISABLE_CLAUDE_MDS=1",
			"CLAUDE_CODE_DISABLE_AUTO_MEMORY=1",
			StageProtectEnv + "=" + dir,
		}, credentialVars...),
		// The user layer is the file under CLAUDE_CONFIG_DIR: naming it alone
		// leaves out the project and local layers of the working tree.
		Args: []string{"--no-session-persistence", "--strict-mcp-config", "--setting-sources=user"},
	}.withStageRoot(abcdhome.Path(home, stageHomesLeaf, name, "no-checkout")), nil
}

// Name is the profile's name: its directory's, directly under the profiles' folder.
func (p Profile) Name() string { return filepath.Base(p.Dir) }

// ForCheckout is p for a drive in the checkout at repo: its stage's home and shell
// file are that checkout's own, so a drive in another deletes nothing of this one.
func (p Profile) ForCheckout(repo string) Profile {
	p.Repo = repo
	sum := sha256.Sum256([]byte(filepath.Clean(repo)))
	return p.withStageRoot(abcdhome.Path(p.PersonHome, stageHomesLeaf, p.Name(), hex.EncodeToString(sum[:8])))
}

// withStageRoot is p with its stage's files under root.
func (p Profile) withStageRoot(root string) Profile {
	p.StageRoot, p.ShellHome, p.EnvFile = root, filepath.Join(root, "home"), filepath.Join(root, stageEnvName)
	p.Env = slices.DeleteFunc(slices.Clone(p.Env), func(kv string) bool {
		return strings.HasPrefix(kv, "CLAUDE_ENV_FILE=") || strings.HasPrefix(kv, StageHomeEnv+"=")
	})
	p.Env = append(p.Env, "CLAUDE_ENV_FILE="+p.EnvFile, StageHomeEnv+"="+p.ShellHome)
	return p
}

// AdmitClaude reports why the claude harness would not be run from the checkout
// repoRoot, nil when it would.
func AdmitClaude(repoRoot string) error {
	_, err := defaultLauncher().admit(Claude, "claude", repoRoot)
	return err
}
