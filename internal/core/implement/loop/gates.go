package loop

import (
	"path/filepath"
	"strings"
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
// dir, "" for none or one outside it. It is read in the person's own environment,
// where what lands is pushed (personEnv): git saying the key is not set (exit 1,
// nothing written) is none; any other failure, a read that did not finish, or an
// answer abcd does not take is an error naming the read, so a caller never takes
// a hooks directory it could not read for none.
func DriveHooksPath(dir string) (string, error) {
	what := "asking git for the core.hooksPath settings (git config --type=path --get-all core.hooksPath)"
	a, err := personEnv.read(dir, what, 1<<16, "config", "-z", "--type=path", "--get-all", "core.hooksPath")
	if err != nil {
		return "", err
	}
	switch {
	case a.code == 1 && a.out == "":
		return "", nil
	case a.code != 0:
		return "", personEnv.exited(what, a)
	}
	values, ok := strings.CutSuffix(a.out, "\x00")
	if !ok {
		return "", personEnv.failed(what, "git did not end its answer as documented, so it is not taken")
	}
	paths := strings.Split(values, "\x00")
	path := paths[len(paths)-1]
	if path == "" {
		return "", nil
	}
	if filepath.IsAbs(path) {
		rel, err := filepath.Rel(dir, path)
		if err != nil || rel == ".." || strings.HasPrefix(rel, "../") {
			return "", nil
		}
		path = rel
	}
	return filepath.ToSlash(filepath.Clean(path)), nil
}
