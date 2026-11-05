package guard

import (
	"path/filepath"
	"strings"

	"github.com/intentdriven/abcd/internal/fsutil"
)

// StageWriteRefusal says why a walled stage's file tool may not write path, ""
// when it may: such a stage writes only in dirs, its working tree and run
// directories, and nothing protected inside them. Both the path as written and
// where it really lands are judged.
func StageWriteRefusal(dirs []string, path string) string {
	fold := approveFoldsCase()
	clean := filepath.Clean(path)
	for _, candidate := range []string{clean, fsutil.RealExistingPath(clean)} {
		inside := false
		for _, d := range dirs {
			for _, dir := range []string{filepath.Clean(d), fsutil.RealExistingPath(filepath.Clean(d))} {
				if !fsutil.PathWithin(candidate, dir, fold) {
					continue
				}
				inside = true
				if name := protectedComponent(candidate, dir, fold); name != "" {
					return "a walled stage does not write " + name + ", which is protected"
				}
			}
		}
		if !inside {
			return "a walled stage writes only in its working tree and its run directories"
		}
	}
	return ""
}

// protectedComponent is the first part of path below dir that is a protected
// name, "" for none.
func protectedComponent(path, dir string, fold bool) string {
	rel, err := filepath.Rel(dir, path)
	if err != nil {
		return ""
	}
	for _, part := range strings.Split(rel, string(filepath.Separator)) {
		for _, name := range protectedNames {
			if part == name || (fold && strings.EqualFold(part, name)) {
				return name
			}
		}
	}
	return ""
}
