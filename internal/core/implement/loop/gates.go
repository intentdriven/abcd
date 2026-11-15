package loop

import (
	"os"
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
// or at, under or above one of the hooks paths the repository configures,
// hooksPaths (DriveHooksPaths): a change list names no directory, so a path above
// a hooks path is an entry, such as a link, standing where one of its parents
// was. A directory counts with or without its trailing slash. Each
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
			if fsutil.FoldedPathMatch(path, hooks, true) || fsutil.FoldedPathMatch(hooks, path, true) {
				return true
			}
		}
	}
	return false
}

// DriveHooksPaths are the paths, relative to dir, that gate the hooks the
// repository at dir configures: nil for none or a value outside dir. A relative
// value is taken from dir, where git runs hooks from. The value is placed as its
// text reads, cleaned, and as the filesystem resolves it, one part at a time
// (hooksHops): each link met on the way is a path, named by its own place, and
// so is where the walk ends, so a chain of links, a link above the hooks
// directory and a ".." after a link each name what git reaches. Every place is
// compared as this machine's filesystem compares names (fsutil.RelWithin), each
// one inside dir is one of the paths, and the same place is named once. It is
// read in the person's own environment, where what lands is pushed (personEnv):
// git saying the key is not set (exit 1, nothing written) is none; any other
// failure, a read that did not finish, or an answer abcd does not take is an
// error naming the read, so a caller never takes a hooks directory it could not
// read for none.
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
	fold := fsutil.CaseFoldingFS()
	var paths []string
	add := func(base, place string) {
		rel, ok := fsutil.RelWithin(base, place, fold)
		if !ok {
			return
		}
		rel = filepath.ToSlash(rel)
		if !slices.ContainsFunc(paths, func(p string) bool { return fsutil.FoldedPathMatch(p, rel, false) }) {
			paths = append(paths, rel)
		}
	}
	text := path
	if !filepath.IsAbs(text) {
		text = filepath.Join(dir, text)
	}
	realDir := fsutil.RealExistingPath(filepath.Clean(dir))
	// The text as written on both sides, then as resolved on both sides, so a
	// checkout reached through a linked parent, with the value in the other
	// form, is placed by where each really is.
	add(filepath.Clean(dir), filepath.Clean(text))
	add(realDir, fsutil.RealExistingPath(filepath.Clean(text)))
	start := realDir
	if filepath.IsAbs(path) {
		start = filepath.VolumeName(path) + string(filepath.Separator)
	}
	for _, hop := range hooksHops(start, path) {
		add(realDir, hop)
	}
	return paths, nil
}

// maxHooksLinks bounds the links hooksHops follows, as the kernel bounds the
// links in one path's resolution.
const maxHooksLinks = 40

// hooksHops are the places the filesystem passes resolving value from start, one
// part at a time: each link met, named by its own place, then where the walk
// ends. A ".." steps back from what has been resolved so far, as the filesystem
// does, not from the text. A loop, or more links than maxHooksLinks, ends the
// walk with the links met so far, each of them a place.
func hooksHops(start, value string) []string {
	cur := start
	rest := pathParts(value)
	var hops []string
	links := 0
	for len(rest) > 0 {
		part := rest[0]
		rest = rest[1:]
		switch part {
		case ".":
			continue
		case "..":
			cur = filepath.Dir(cur)
			continue
		}
		next := filepath.Join(cur, part)
		target, err := os.Readlink(next)
		if err != nil {
			// No link here, or nothing at all: taken as named.
			cur = next
			continue
		}
		hops = append(hops, next)
		if links++; links > maxHooksLinks {
			return hops
		}
		if filepath.IsAbs(target) {
			vol := filepath.VolumeName(target)
			cur = vol + string(filepath.Separator)
			target = target[len(vol):]
		}
		rest = append(pathParts(target), rest...)
	}
	return append(hops, cur)
}

// pathParts are the parts of p between its separators, the volume left out.
func pathParts(p string) []string {
	p = p[len(filepath.VolumeName(p)):]
	return strings.FieldsFunc(p, func(r rune) bool { return r == '/' || r == filepath.Separator })
}
