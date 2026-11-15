package loop

import (
	"fmt"
	"strings"

	"github.com/intentdriven/abcd/internal/gitutil"
)

// addAllGit runs `git add -A` in dir, the root of a lane's worktree or of the
// checkout, on its own index (pickGit), limited to pathspec ("." when none),
// with each checked-out submodule left out whose commit is the one the index
// records (addAllArgs).
func addAllGit(dir string, pathspec ...string) (string, error) {
	args, err := addAllArgs(dir, "", pathspec)
	if err != nil {
		return "", err
	}
	return pickGit(dir, args...)
}

// addAllGitIndex is addAllGit on the scratch index at index (pickGitIndex).
func addAllGitIndex(dir, index string, pathspec ...string) (string, error) {
	args, err := addAllArgs(dir, index, pathspec)
	if err != nil {
		return "", err
	}
	return pickGitIndex(dir, index, args...)
}

// addAllArgs is the command line of a `git add -A` the drive runs at the
// root dir, on the index at index ("" for dir's own): "add", "-A", "--",
// pathspec ("." when none), then an exclude for each gitlink of that index
// whose checked-out commit is the one the index records.
//
// git's add compares the index with the working tree, and for such a
// submodule starts a status inside it, under that submodule's own settings,
// so a content filter they name runs there. git add takes no
// --ignore-submodules, and it reads no submodule.<name>.ignore or
// diff.ignoreSubmodules setting, so a pathspec is the command-line form that
// keeps it out: staging such a submodule changes nothing in the index. A
// submodule whose checked-out commit moved, or whose path is gone or holds
// something else, is staged as before, and git does not look inside it to
// tell. Which gitlinks moved is read first, isolated, by `git diff-files
// --ignore-submodules=dirty` over the gitlinks alone, which starts nothing in
// a submodule.
func addAllArgs(dir, index string, pathspec []string) ([]string, error) {
	if len(pathspec) == 0 {
		pathspec = []string{"."}
	}
	args := append([]string{"add", "-A", "--"}, pathspec...)
	var links []string
	var err error
	if index == "" {
		links, err = gitutil.Gitlinks(dir)
	} else {
		links, err = gitutil.GitlinksWithIndex(dir, index)
	}
	if err != nil {
		return nil, fmt.Errorf("listing the submodules before staging: %w", err)
	}
	if len(links) == 0 {
		return args, nil
	}
	diff := append([]string{"--literal-pathspecs", "diff-files", "--ignore-submodules=dirty", "--name-only", "-z", "--"}, links...)
	var out []byte
	if index == "" {
		out, err = gitutil.RunCappedBytes(dir, maxGitOutput, diff...)
	} else {
		out, err = gitutil.RunWithIndex(dir, index, maxGitOutput, diff...)
	}
	if err != nil {
		return nil, fmt.Errorf("reading which submodules moved before staging: %w", err)
	}
	moved := map[string]bool{}
	for _, p := range strings.Split(string(out), "\x00") {
		if p != "" {
			moved[p] = true
		}
	}
	for _, l := range links {
		if !moved[l] {
			args = append(args, ":(top,exclude,literal)"+l)
		}
	}
	return args, nil
}

// submoduleIgnorePins is the `-c submodule.<name>.ignore=dirty` pair, for a
// lane's sync merge (syncMerge), of each submodule name a .gitmodules of the
// worktree at dir declares: the working tree's file, the index's and HEAD's
// (git reads the first of these that exists), each read isolated by `git
// config`, which follows no include of a file or blob named to it.
//
// git's merge first saves the lane's uncommitted changes (`git stash
// create`), which compares the index with the working tree and, where a
// submodule's settings or the .gitmodules entry set its ignore to none,
// starts a status inside that submodule, under its own settings, so a content
// filter they name runs there. git merge takes no --ignore-submodules, and a
// diff.ignoreSubmodules on the command line loses to submodule.<name>.ignore;
// the per-name setting on the command line outranks every settings file and
// .gitmodules. A name holding "=" cannot be named that way (git splits a -c
// at its first "="), so it is refused, naming it.
func submoduleIgnorePins(dir string) ([]string, error) {
	seen := map[string]bool{}
	var pins []string
	for _, src := range [][]string{{"--file", ".gitmodules"}, {"--blob", ":.gitmodules"}, {"--blob", "HEAD:.gitmodules"}} {
		// Exit 1 is no submodule entry, or no such file or blob.
		out, _, err := gitutil.RunCappedExitOne(dir, maxGitOutput, append(append([]string{"config"}, src...), "-z", "--name-only", "--get-regexp", `^submodule\.`)...)
		if err != nil {
			return nil, fmt.Errorf("reading the submodule names of %s before the merge: %w", src[1], err)
		}
		for _, key := range strings.Split(string(out), "\x00") {
			name, ok := strings.CutPrefix(key, "submodule.")
			i := strings.LastIndexByte(name, '.')
			if !ok || i <= 0 {
				continue
			}
			name = name[:i]
			if seen[name] {
				continue
			}
			seen[name] = true
			if strings.Contains(name, "=") {
				return nil, fmt.Errorf("the submodule name %q in %s holds \"=\", which git's command line cannot name, so the merge cannot be kept from looking inside it", name, src[1])
			}
			pins = append(pins, "-c", "submodule."+name+".ignore=dirty")
		}
	}
	return pins, nil
}
