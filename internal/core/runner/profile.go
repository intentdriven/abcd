package runner

// profile.go is the one place that answers "the profile this drive runs on"
// (spc-2610230813348813 step 1; adr-2610222222326104's amendment of
// 2026-11-04): the one directory under ~/.abcd.noindex/profiles/, the launch's
// environment and its arguments. The person makes the directory and logs the
// harness into it once, in their own shell; abcd never reads, stores or
// forwards the login. Nothing else reads the directory, so
// itd-2611040344170206, which admits several named profiles, replaces this
// file's one function and moves nothing the person logged in.

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"syscall"

	"github.com/intentdriven/abcd/internal/abcdhome"
	"github.com/intentdriven/abcd/internal/fsutil"
)

// StageWritesEnv names where a walled launch's file tools may write.
const StageWritesEnv = "ABCD_STAGE_WRITES"

// The rest of what a walled launch's guard reads: the shell's home, what is
// protected from the stage, what its work declares it changes, and the commit
// its lane was cut from.
const (
	StageHomeEnv     = "ABCD_STAGE_HOME"
	StageProtectEnv  = "ABCD_STAGE_PROTECT"
	StageDeclaredEnv = "ABCD_STAGE_DECLARED"
	StageBaseEnv     = "ABCD_STAGE_BASE"
)

// The intents a drive's refusals name.
const (
	// ProfilesIntent admits several named profiles.
	ProfilesIntent = "itd-2611040344170206"
	// BudgetIntent covers how a key reaches the harness and what it may spend.
	BudgetIntent = "itd-2611011230226453"
)

// profilesLeaf is the walled directories' folder below abcd's home.
const profilesLeaf = "profiles"

// The refusals DriveProfile names, through errors.Is.
var (
	ErrNoProfile       = errors.New("no walled directory")
	ErrSeveralProfiles = errors.New("more than one walled directory")
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

// CredentialVars names the variables a walled launch never carries.
func CredentialVars() []string { return append([]string(nil), credentialVars...) }

// HeldCredentials names each of CredentialVars set in this process's
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
	// ShellHome is the home a stage's shell runs under.
	ShellHome string
	// PersonHome is the person's own home directory.
	PersonHome string
	// Repo is the checkout the drive runs in, whose git configuration names the
	// identity a stage commits under.
	Repo string
	// Env is the launch's environment switches (sealedEnv): each KEY=value is
	// set over the parent's, and each bare name taken out.
	Env []string
	// Args are the launch's walled arguments.
	Args []string
}

// DriveProfile is the profile a drive runs on, below home, the person's home
// directory. It refuses none, several, and a directory it would not trust
// with the settings that decide what the drive's agents run.
func DriveProfile(home string) (Profile, error) {
	root := abcdhome.Path(home, profilesLeaf)
	shown := abcdhome.Display(profilesLeaf) + "/"
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
	switch len(names) {
	case 0:
		return Profile{}, fmt.Errorf("%w under %s: make one and log the harness into it once, in your own shell: "+
			"`mkdir -p %sdrive && CLAUDE_CONFIG_DIR=%sdrive claude`, then the harness's own login step; abcd never logs a harness in", ErrNoProfile, shown, shown, shown)
	case 1:
	default:
		return Profile{}, fmt.Errorf("%w under %s (%s): a drive runs on exactly one until %s admits several; move the others out",
			ErrSeveralProfiles, shown, strings.Join(names, ", "), ProfilesIntent)
	}
	dir := filepath.Join(root, names[0])
	fi, err := os.Lstat(dir)
	if err != nil {
		return Profile{}, fmt.Errorf("%s%s could not be read: %w", shown, names[0], err)
	}
	if !fi.IsDir() {
		return Profile{}, fmt.Errorf("%s%s is a link, not a directory of its own; a drive's settings are written only into a real directory", shown, names[0])
	}
	if st, ok := fi.Sys().(*syscall.Stat_t); ok && int(st.Uid) != os.Geteuid() {
		return Profile{}, fmt.Errorf("%s%s belongs to another account; a drive runs only on a directory this account owns", shown, names[0])
	}
	if fsutil.WritableByOthers(fi) {
		return Profile{}, fmt.Errorf("%s%s is one group or other can write; chmod go-w it", shown, names[0])
	}
	return Profile{
		Dir:        dir,
		ShellHome:  abcdhome.Path(home, stageHomesLeaf, names[0]),
		PersonHome: home,
		Env: append([]string{
			"CLAUDE_CONFIG_DIR=" + dir,
			"CLAUDE_CODE_DISABLE_CLAUDE_MDS=1",
			"CLAUDE_CODE_DISABLE_AUTO_MEMORY=1",
			"CLAUDE_ENV_FILE=" + filepath.Join(dir, stageEnvName),
			StageHomeEnv + "=" + abcdhome.Path(home, stageHomesLeaf, names[0]),
			StageProtectEnv + "=" + dir,
		}, credentialVars...),
		// The user layer is the file under CLAUDE_CONFIG_DIR: naming it alone
		// leaves out the project and local layers of the working tree.
		Args: []string{"--no-session-persistence", "--strict-mcp-config", "--setting-sources=user"},
	}, nil
}

// AdmitClaude reports why the claude harness would not be run from the checkout
// repoRoot, nil when it would.
func AdmitClaude(repoRoot string) error {
	_, err := defaultLauncher().admit(Claude, "claude", repoRoot)
	return err
}

// recordLaunch is the one call a walled launch makes when it ends, whatever
// came of it, for the launch ledger (itd-2611031342011261) to fill. It records
// nothing yet.
var recordLaunch = func(Profile, Request, Answer, error) {}
