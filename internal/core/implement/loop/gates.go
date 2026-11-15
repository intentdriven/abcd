package loop

import (
	"path/filepath"
	"slices"
	"strings"

	"github.com/intentdriven/abcd/internal/fsutil"
)

// DriveGates are what a lane's preflight, the driver's own git and the forge's
// workflows run, and abcd's own configuration, receipts and approvals, as
// paths: a change to one is a change to what judges the lane. They are kept in
// this one place for everything that names them: `abcd drain --drive`, which
// lands no lane that changes one and passes over an issue on one; and the issue
// brief, which names them to the implementer. A directory ends in a slash. The
// files under .abcd are the ones a drive's stage may not write either.
var DriveGates = []string{"Makefile", "GNUmakefile", "makefile", "scripts/", ".githooks/", "hooks/", ".github/",
	".abcd/rules.json", ".abcd/guard.json", ".abcd/config.json", ".abcd/config/",
	".abcd/.work.local/preflight-receipts/", ".abcd/.work.local/approvals/", ".abcd/.work.local/review-receipts/",
	".abcd/.work.local/reservation/"}

// DriveGate reports whether path, relative to the checkout, is one of DriveGates,
// or under one of the hooks paths the repository configures, hooksPaths
// (DriveHooksPaths). A directory counts with or without its trailing slash. Each
// part of the name is compared as a case-folding filesystem compares it,
// whatever this machine's filesystem does (fsutil.FoldedPathMatch). A hooks path
// of the tree's top makes every path a gate, as a hooks directory counts with
// all below it, and an empty one names none. An empty path names nothing.
func DriveGate(path string, hooksPaths ...string) bool {
	path = filepath.ToSlash(filepath.Clean(path))
	if path == "." {
		return false
	}
	for _, gate := range DriveGates {
		dir := strings.TrimSuffix(gate, "/")
		if fsutil.FoldedPathMatch(path, dir, dir != gate) {
			return true
		}
	}
	for _, hooksPath := range hooksPaths {
		if hooksPath == "" {
			continue
		}
		switch hooks := strings.Trim(filepath.ToSlash(filepath.Clean(hooksPath)), "/"); hooks {
		case "":
		case ".":
			return true
		default:
			if fsutil.FoldedPathMatch(path, hooks, true) {
				return true
			}
		}
	}
	return false
}

// DriveHooksPaths are the paths, relative to dir, that gate the hooks the
// repository at dir configures: nil for none or a value outside dir. A relative
// value is taken from dir, where git runs hooks from, and cleaned; each value is
// then placed as written and as resolved on both sides, both compared as this
// machine's filesystem compares names (fsutil.RelWithin), and each placement
// inside dir is one of the paths, the same place named once. So a value that
// leaves the tree and comes back, or spells the checkout in another case, names
// the directory it reaches, and a hooks directory that is a link names both the
// link and the directory it leads to. It is read in the person's own
// environment, where what lands is pushed (personEnv): git saying the key is not
// set (exit 1, nothing written) is none; any other failure, a read that did not
// finish, or an answer abcd does not take is an error naming the read, so a
// caller never takes a hooks directory it could not read for none.
func DriveHooksPaths(dir string) ([]string, error) {
	what := "asking git for the core.hooksPath settings (git config --type=path --get-all core.hooksPath)"
	a, err := personEnv.read(dir, what, 1<<16, "config", "-z", "--type=path", "--get-all", "core.hooksPath")
	if err != nil {
		return nil, err
	}
	switch {
	case a.code == 1 && a.out == "":
		return nil, nil
	case a.code != 0:
		return nil, personEnv.exited(what, a)
	}
	values, ok := strings.CutSuffix(a.out, "\x00")
	if !ok {
		return nil, personEnv.failed(what, "git did not end its answer as documented, so it is not taken")
	}
	all := strings.Split(values, "\x00")
	path := all[len(all)-1]
	if path == "" {
		return nil, nil
	}
	if !filepath.IsAbs(path) {
		path = filepath.Join(dir, path)
	}
	// Placed as written on both sides, and as resolved on both sides, so a
	// checkout reached through a linked parent, with the value in the other
	// form, is placed by where each really is, and a linked hooks directory by
	// both its own name and where it leads.
	fold := fsutil.CaseFoldingFS()
	var paths []string
	for _, place := range [][2]string{
		{filepath.Clean(dir), filepath.Clean(path)},
		{fsutil.RealExistingPath(filepath.Clean(dir)), fsutil.RealExistingPath(filepath.Clean(path))},
	} {
		rel, ok := fsutil.RelWithin(place[0], place[1], fold)
		if !ok {
			continue
		}
		rel = filepath.ToSlash(rel)
		if !slices.ContainsFunc(paths, func(p string) bool { return fsutil.FoldedPathMatch(p, rel, false) }) {
			paths = append(paths, rel)
		}
	}
	return paths, nil
}
