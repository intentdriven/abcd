package tools

// walk.go is the one reading of a program's route: Walk resolves a path as the
// system does, one part at a time, and says which links and directories it
// passed on the way. The installer's tree check (treeContains), the gitleaks
// adapter's binary check and the needs judge all hold what it returns to
// their trees, so a link a repository holds is caught wherever on the route it
// sits: a step of a link chain, or a directory link in the middle of a folder
// path, reached through a link outside the repository or not.

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"syscall"
)

// maxWalkLinks bounds the links Walk follows, as the system's own limit does.
const maxWalkLinks = 40

// ErrReadOtherwise is Walk's answer when its route is not the one a reader of
// the path and its link texts would take, or does not end where
// filepath.EvalSymlinks ends.
var ErrReadOtherwise = errors.New("a link on its way is read otherwise by the system")

// ErrLinksDoNotEnd is Walk's answer when more than maxWalkLinks links lie on
// the route, as a loop's do.
var ErrLinksDoNotEnd = fmt.Errorf("its links do not end within %d steps", maxWalkLinks)

// Route is how the system reaches a path.
type Route struct {
	// Real is the path with every link resolved: the file the system opens.
	// When Missing, it is where the system would look: the resolved part of
	// the route joined with the parts it did not reach.
	Real string
	// Links is the location of every link the route passed, in order: the
	// directory it lies in, itself resolved, joined with its own name. A
	// directory link in the middle of a folder path is one of them as much as
	// a link the path ends at.
	Links []string
	// Dirs is every real directory the route passed through, in the order it
	// first reached each, "/" first.
	Dirs []string
	// Missing is true when a part of the route does not exist, a dangling
	// link's target included; the walk stops there, as the system does.
	Missing bool
}

// segment is one text the walk reads parts from: the path itself, or a link's
// target spliced in where the link was reached.
type segment struct {
	id    int
	parts []string
	// done marks the directory a link's target ends at, once the target is
	// read, with the segment the link's name came from and the directory the
	// link lies in.
	done *mark
}

// mark says a directory was reached through a link whose name came from
// segment seg, lying in parent.
type mark struct {
	seg    int
	parent string
}

type frame struct {
	path  string
	isDir bool
	marks []mark
}

// Walk resolves the absolute path p the way the system does: from "/", one
// part at a time, with each prefix examined by Lstat. A link's own location
// (its directory, already resolved, joined with its name) is recorded, and its
// target is spliced in where it was reached: an absolute target restarts at
// "/", a relative one continues from the directory the link lies in. ".."
// leaves the resolved directory, never the text, and a part after a file that
// is not a directory is refused as the system refuses it (not a directory).
//
// Two routes are refused as read otherwise (ErrReadOtherwise). One is a ".."
// that a text places straight after a directory link written in the same
// text, when the link's target does not lie beside it (d/../x, d -> /elsewhere/sub):
// the system takes that ".." from where d leads and a reader of the text from
// where d sits, so the route is not the one its text shows. The other is a
// route that does not end where filepath.EvalSymlinks(p) ends, or, when the
// walk stops at a missing part, where EvalSymlinks finds something; the walk
// is cross-checked against it so that a difference between the two readings
// refuses rather than admits.
//
// A route with more than 40 links is refused (ErrLinksDoNotEnd). Any other
// failure to examine a part is the *fs.PathError that says why.
func Walk(p string) (Route, error) {
	if !filepath.IsAbs(p) {
		return Route{}, &fs.PathError{Op: "walk", Path: p, Err: errors.New("not an absolute path")}
	}
	var r Route
	seen := map[string]bool{}
	addDir := func(d string) {
		if !seen[d] {
			seen[d] = true
			r.Dirs = append(r.Dirs, d)
		}
	}
	frames := []frame{{path: "/", isDir: true}}
	addDir("/")
	segs := []*segment{{id: 0, parts: strings.Split(p, "/")}}
	nextID, links := 1, 0
	for {
		top := &frames[len(frames)-1]
		seg := segs[len(segs)-1]
		if len(seg.parts) == 0 {
			segs = segs[:len(segs)-1]
			if seg.done != nil {
				top.marks = append(top.marks, *seg.done)
			}
			if len(segs) == 0 {
				break
			}
			continue
		}
		part := seg.parts[0]
		seg.parts = seg.parts[1:]
		if part == "" || part == "." {
			continue
		}
		if !top.isDir {
			return Route{}, &fs.PathError{Op: "lstat", Path: filepath.Join(top.path, part), Err: syscall.ENOTDIR}
		}
		if part == ".." {
			parent := filepath.Dir(top.path)
			for _, m := range top.marks {
				if m.seg == seg.id && m.parent != parent {
					return Route{}, ErrReadOtherwise
				}
			}
			if len(frames) > 1 {
				frames = frames[:len(frames)-1]
			}
			continue
		}
		next := filepath.Join(top.path, part)
		fi, err := os.Lstat(next)
		if errors.Is(err, fs.ErrNotExist) {
			rest := []string{next}
			for i := len(segs) - 1; i >= 0; i-- {
				rest = append(rest, segs[i].parts...)
			}
			r.Real, r.Missing = filepath.Join(rest...), true
			return r, crossCheck(p, r)
		}
		if err != nil {
			return Route{}, err
		}
		if fi.Mode()&os.ModeSymlink == 0 {
			frames = append(frames, frame{path: next, isDir: fi.IsDir()})
			if fi.IsDir() {
				addDir(next)
			}
			continue
		}
		if links++; links > maxWalkLinks {
			return Route{}, ErrLinksDoNotEnd
		}
		r.Links = append(r.Links, next)
		target, err := os.Readlink(next)
		if err != nil {
			return Route{}, err
		}
		if target == "" {
			return Route{}, &fs.PathError{Op: "readlink", Path: next, Err: syscall.ENOENT}
		}
		at := top.path
		if filepath.IsAbs(target) {
			frames = frames[:1]
		}
		segs = append(segs, &segment{id: nextID, parts: strings.Split(target, "/"), done: &mark{seg: seg.id, parent: at}})
		nextID++
	}
	r.Real = frames[len(frames)-1].path
	return r, crossCheck(p, r)
}

// crossCheck holds a walk to the system's own resolution of p.
func crossCheck(p string, r Route) error {
	sys, err := filepath.EvalSymlinks(p)
	if r.Missing {
		switch {
		case err == nil:
			return ErrReadOtherwise
		case errors.Is(err, fs.ErrNotExist):
			return nil
		}
		return err
	}
	if err != nil {
		return err
	}
	if sys != r.Real {
		return ErrReadOtherwise
	}
	return nil
}
