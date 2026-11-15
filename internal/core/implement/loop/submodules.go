package loop

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"syscall"

	"github.com/intentdriven/abcd/internal/gitutil"
	"github.com/intentdriven/abcd/internal/termsafe"
)

// addAllGit runs `git add -A` in dir, the root of a lane's worktree or of the
// checkout, on its own index (addCommand), limited to pathspec ("." when
// none), with each checked-out submodule left out whose commit is the one the
// index records (addAllArgs).
func addAllGit(dir string, pathspec ...string) (string, error) {
	return addAll(dir, "", pathspec)
}

// addAllGitIndex is addAllGit on the scratch index at index (addCommand).
func addAllGitIndex(dir, index string, pathspec ...string) (string, error) {
	return addAll(dir, index, pathspec)
}

// addAll runs addAllArgs's add through addCommand.
func addAll(dir, index string, pathspec []string) (string, error) {
	args, stdin, err := addAllArgs(dir, index, pathspec)
	if err != nil {
		return "", err
	}
	cmd, err := addCommand(dir, index, stdin, args)
	if err != nil {
		return "", err
	}
	return gitutil.RunTreeCommand(cmd, args)
}

// addCommand is the command the drive's add runs: pickGitCommand at dir, or,
// on the scratch index at index, pickGitIndexCommand, with
// GIT_ICASE_PATHSPECS taken out of its environment. git has no option that
// undoes that variable, and with it set every pathspec matches without regard
// to case, so an exclude would also leave out a path that differs from the
// submodule's only in case; without it an exclude matches the submodule's own
// path alone. No program the add starts sees the variable.
func addCommand(dir, index, stdin string, args []string) (*exec.Cmd, error) {
	var cmd *exec.Cmd
	if index == "" {
		cmd = pickGitCommand(dir, stdin, args)
	} else {
		var err error
		if cmd, err = pickGitIndexCommand(dir, index, stdin, args); err != nil {
			return nil, err
		}
	}
	cmd.Env = slices.DeleteFunc(cmd.Env, func(kv string) bool {
		return strings.HasPrefix(kv, "GIT_ICASE_PATHSPECS=")
	})
	return cmd, nil
}

// addAllArgs is the command line, and the standard input, of a `git add -A`
// the drive runs at the root dir, on the index at index ("" for dir's own):
// `git --no-literal-pathspecs add -A --pathspec-from-file=-
// --pathspec-file-nul`, given on its standard input pathspec ("." when none)
// and then an exclude for each gitlink of that index whose checked-out commit
// is the one the index records, each NUL-terminated. On standard input the
// pathspecs are bounded by nothing the system's limit on a command line's
// length sets, and --no-literal-pathspecs keeps a GIT_LITERAL_PATHSPECS in
// the environment from reading the excludes as file names: git then sets
// GIT_LITERAL_PATHSPECS=0 for every program the add starts (a content filter;
// the add runs no hook, since WorkingTreeCommand pins core.hooksPath to
// /dev/null).
//
// git's add compares the index with the working tree, and for such a
// submodule starts a status inside it, under that submodule's own settings,
// so a content filter they name runs there. git add takes no
// --ignore-submodules, and it reads no submodule.<name>.ignore or
// diff.ignoreSubmodules setting, so a pathspec is the command-line form that
// keeps it out: staging such a submodule changes nothing in the index. A
// submodule whose checked-out commit moved, or whose path is gone or holds
// something else, is staged as before, and git does not look inside it to
// tell. Which gitlinks moved is read first, isolated, by
// gitutil.MovedGitlinks, which starts nothing in a submodule and runs no
// filter; the gitlinks of the index are listed by gitutil.Gitlinks or
// GitlinksWithIndex. Neither bounds their number.
func addAllArgs(dir, index string, pathspec []string) (args []string, stdin string, err error) {
	if len(pathspec) == 0 {
		pathspec = []string{"."}
	}
	args = []string{"--no-literal-pathspecs", "add", "-A", "--pathspec-from-file=-", "--pathspec-file-nul"}
	var links []string
	if index == "" {
		links, err = gitutil.Gitlinks(dir)
	} else {
		links, err = gitutil.GitlinksWithIndex(dir, index)
	}
	if err != nil {
		return nil, "", fmt.Errorf("listing the submodules before staging: %w", err)
	}
	var in strings.Builder
	for _, p := range pathspec {
		in.WriteString(p + "\x00")
	}
	if len(links) == 0 {
		return args, in.String(), nil
	}
	moved, err := gitutil.MovedGitlinks(dir, index)
	if err != nil {
		return nil, "", fmt.Errorf("reading which submodules moved before staging: %w", err)
	}
	for _, l := range links {
		if !moved[l] {
			in.WriteString(":(top,exclude,literal)" + l + "\x00")
		}
	}
	return args, in.String(), nil
}

// checkedOutSubmodule is a sync's merge not made (syncMerge): the lane's
// index records a gitlink at Path with a repository checked out there.
type checkedOutSubmodule struct{ Path string }

// Error names the path through termsafe.Sanitize, since the lane's index
// holds it.
func (e *checkedOutSubmodule) Error() string {
	return fmt.Sprintf("the lane's index records a submodule at %q with a repository checked out there, so the sync's merge is not made: git's merge can start a status inside it, under that submodule's own settings, and no command-line form reaches what the merge runs without reaching the hooks it runs", termsafe.Sanitize(e.Path))
}

// checkedOutGitlink is the first gitlink of the index of the worktree at dir
// whose path holds a checked-out repository (a .git file or directory, or
// anything else of that name, under it), whether or not .gitmodules names
// it, as a *checkedOutSubmodule; nil when none does. A gitlink with nothing
// checked out at its path is no such one.
func checkedOutGitlink(dir string) error {
	links, err := gitutil.Gitlinks(dir)
	if err != nil {
		return fmt.Errorf("listing the submodules before the sync's merge: %w", err)
	}
	for _, l := range links {
		_, err := os.Lstat(filepath.Join(dir, filepath.FromSlash(l), ".git"))
		switch {
		case err == nil:
			return &checkedOutSubmodule{Path: l}
		case errors.Is(err, fs.ErrNotExist), errors.Is(err, syscall.ENOTDIR):
		default:
			// The path is named once, sanitized; the error the lstat gave
			// is kept without the path it would repeat raw.
			var pe *fs.PathError
			if errors.As(err, &pe) {
				err = pe.Err
			}
			return fmt.Errorf("reading whether the submodule at %q is checked out before the sync's merge: %w", termsafe.Sanitize(l), err)
		}
	}
	return nil
}
