package runner

// session_profile.go answers one question for the front doors: does this
// session's setting name a profile? It is how abcd warns a session the person
// started off a profile (dcn-2611041550129278), and it writes nothing and prints
// nothing: the words and the stream are the front door's.
//
// The answer reads a setting and proves no wall. No host shows a session's tools
// to any hook, so a session on a profile could hold other tools, which is why
// the ruling warns and never refuses. A profile is a separate login and settings
// on one operating-system account.

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/intentdriven/abcd/internal/abcdhome"
)

// ConfigDirEnv is the variable that points the agent tool at its configuration
// directory. Another host's session reports no such variable and so reads as off
// a profile.
const ConfigDirEnv = "CLAUDE_CONFIG_DIR"

// SessionOnProfile reports whether configDir, the value of ConfigDirEnv in the
// session's environment, names a profile of the person whose home directory is
// home. When it does not, why says how the setting reads, naming neither the
// variable nor a path.
//
// A profile is a directory directly under the profiles' folder, judged after
// resolving configDir's links, so a link out of the folder is off and a link
// into it is on. Only home itself may be reached through a link, as temporary
// and some home directories are: a linked ~/.abcd.noindex or profiles folder
// reads as off, as a linked level of abcd's home is refused everywhere else.
// Every doubt reads as off, since a warning printed once too often costs a line.
func SessionOnProfile(home, configDir string) (on bool, why string) {
	const noHome = "cannot be judged with no home directory"
	switch {
	case configDir == "":
		return false, "is not set"
	case !filepath.IsAbs(configDir):
		return false, "is relative"
	}
	if !filepath.IsAbs(home) {
		return false, noHome
	}
	realHome, err := filepath.EvalSymlinks(home)
	if err != nil {
		return false, noHome
	}
	dir, err := filepath.EvalSymlinks(configDir)
	if err != nil {
		return false, "names no directory"
	}
	if fi, err := os.Stat(dir); err != nil || !fi.IsDir() {
		return false, "names no directory"
	}
	rel, err := filepath.Rel(abcdhome.Path(realHome, profilesLeaf), dir)
	if err != nil || rel == "." || rel == ".." || strings.ContainsRune(rel, filepath.Separator) {
		return false, "names no profile"
	}
	return true, ""
}
