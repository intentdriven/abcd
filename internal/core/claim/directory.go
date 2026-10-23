package claim

// directory.go names the directories a reservation's footprint holds
// (iss-2610230016544142). A footprint path that names a directory holds off
// every other reservation of any file inside it, so a lane reserving a whole
// package serialises every unrelated lane that touches one file of it. The
// reservation is still made: the act names each such directory and the
// tracked files it covers, so the person names the files instead.

import (
	"fmt"
	"path"
	"strings"

	"github.com/intentdriven/abcd/internal/gitutil"
	"github.com/intentdriven/abcd/internal/termsafe"
)

// DirectoryFootprint is a footprint path that names a directory git tracks.
type DirectoryFootprint struct {
	// Path is the directory, cleaned.
	Path string
	// Files is how many tracked files lie beneath it on On, -1 when git could
	// not count them.
	Files int
	// On is the branch the directory was read from: origin's default branch
	// as last fetched (origin/main), or HEAD when that holds no such path,
	// terminal-safe, since git allows a bidi or zero-width rune in a branch
	// name.
	On string
}

// Warning is how a person is told a footprint path names a directory, the
// same words wherever it is shown.
func (d DirectoryFootprint) Warning() string {
	on := termsafe.Sanitize(d.On)
	covers := "a directory"
	switch d.Files {
	case -1:
		covers = "a directory whose files git could not count"
	case 1:
		covers = fmt.Sprintf("a directory covering 1 tracked file on %s", on)
	default:
		if d.Files >= 0 {
			covers = fmt.Sprintf("a directory covering %d tracked files on %s", d.Files, on)
		}
	}
	return fmt.Sprintf("footprint %s is %s, and holds off every other reservation of a file inside it; reserved anyway: name the files the work changes instead",
		echo(d.Path), covers)
}

// directoryFootprints is each path of fp that names a directory git tracks,
// read from def, origin's default branch as the act resolved it ("" for
// none) and, for a path it does not hold, from HEAD; a path neither holds as
// a directory (a file, or one the work will add) is not named. It reads what
// is stored and never fails: a ref git cannot read names nothing.
func directoryFootprints(root, def string, fp []string) []DirectoryFootprint {
	refs := []string{"HEAD"}
	if def != "" {
		refs = []string{def, "HEAD"}
	}
	var out []DirectoryFootprint
	seen := map[string]bool{}
	for _, p := range fp {
		p = path.Clean(p)
		if seen[p] || !validFootprint(p) {
			continue
		}
		seen[p] = true
		for _, ref := range refs {
			kind, err := storedGit(root, 64, "cat-file", "-t", ref+":"+p)
			if err != nil || strings.TrimSpace(kind) != "tree" {
				continue
			}
			d := DirectoryFootprint{Path: p, Files: -1, On: termsafe.Sanitize(gitutil.ShortRef(ref))}
			if list, err := storedGit(root, treeListCap, "ls-tree", "-r", "-z", "--name-only", ref+":"+p); err == nil {
				d.Files = 0
				for _, name := range strings.Split(list, "\x00") {
					if name != "" {
						d.Files++
					}
				}
			}
			out = append(out, d)
			break
		}
	}
	return out
}
