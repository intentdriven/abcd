package loop

import (
	"path/filepath"
	"strings"

	"github.com/intentdriven/abcd/internal/gitutil"
)

// DriveGates are what a lane's preflight and the driver's own git run, as paths: a
// change to one is a change to what judges the lane. They are kept in this one
// place for everything that names them: `abcd drain --drive`, which lands no lane
// that changes one and passes over an issue on one; and the issue brief, which
// names them to the implementer. A directory ends in a slash.
var DriveGates = []string{"Makefile", "GNUmakefile", "makefile", "scripts/", ".githooks/", "hooks/"}

// DriveGate reports whether path, relative to the checkout, is one of DriveGates,
// or under the hooks path the repository configures, hooksPath (DriveHooksPath).
// A directory counts with or without its trailing slash.
func DriveGate(path, hooksPath string) bool {
	path = filepath.ToSlash(filepath.Clean(path))
	for _, gate := range DriveGates {
		dir := strings.TrimSuffix(gate, "/")
		if path == dir || (dir != gate && strings.HasPrefix(path, gate)) {
			return true
		}
	}
	if hooksPath = strings.Trim(hooksPath, "/"); hooksPath != "" {
		return path == hooksPath || strings.HasPrefix(path, hooksPath+"/")
	}
	return false
}

// DriveHooksPath is the hooks path the repository at dir configures, relative to
// dir, "" for none or one outside it.
func DriveHooksPath(dir string) string {
	paths, err := gitutil.HooksPaths(dir)
	if err != nil || len(paths) == 0 {
		return ""
	}
	path := paths[len(paths)-1]
	if filepath.IsAbs(path) {
		rel, err := filepath.Rel(dir, path)
		if err != nil || rel == ".." || strings.HasPrefix(rel, "../") {
			return ""
		}
		path = rel
	}
	return filepath.ToSlash(filepath.Clean(path))
}
