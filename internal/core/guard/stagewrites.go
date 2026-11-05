package guard

import (
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/intentdriven/abcd/internal/fsutil"
)

// stageAgentEntryID is the refusal of a further agent from a walled stage.
const stageAgentEntryID = "stage-further-agent"

// stageGateFiles are the first parts of a path, below a stage's working tree,
// that decide how its lane is judged. The stage's own local tier is not among
// them (stageLocalTier).
var stageGateFiles = []string{".abcd", ".githooks", "hooks", "Makefile", "GNUmakefile", "makefile"}

// stageLocalTier is the one place under .abcd a stage writes freely.
const stageLocalTier = ".abcd/.work.local"

// startsAnAgent reports whether the command at tokens[idx] starts a further
// harness or a drive.
func startsAnAgent(tokens []string, idx int) bool {
	switch filepath.Base(tokens[idx]) {
	case "claude":
		return true
	case "abcd":
		rest := tokens[idx+1:]
		return slices.Contains(rest, "drain") && slices.ContainsFunc(rest, func(w string) bool {
			return w == "--drive" || strings.HasPrefix(w, "--drive=")
		})
	}
	return false
}

// stageGateFile is the gate file path lands in below tree, "" when it lands in
// none, or tree does not hold it, or declared names it.
func stageGateFile(tree, path string, declared []string, fold bool) string {
	if !fsutil.PathWithin(path, tree, fold) {
		return ""
	}
	rel, err := filepath.Rel(tree, path)
	if err != nil {
		return ""
	}
	rel = filepath.ToSlash(rel)
	first, _, _ := strings.Cut(rel, "/")
	gate := slices.ContainsFunc(stageGateFiles, func(n string) bool {
		return first == n || (fold && strings.EqualFold(first, n))
	})
	if !gate || rel == stageLocalTier || strings.HasPrefix(rel, stageLocalTier+"/") {
		return ""
	}
	for _, d := range declared {
		if d = filepath.ToSlash(filepath.Clean(d)); rel == d || strings.HasPrefix(rel, d+"/") {
			return ""
		}
	}
	return first
}

// StageWriteRefusal says why a walled stage's file tool may not write path, ""
// when it may: such a stage writes only in dirs, its working tree and run
// directories, and nothing protected inside them, and no gate file its work does
// not declare. A path that is no absolute one, or whose last part is a link, is
// refused, and both the path as written and where it really lands are judged.
func StageWriteRefusal(dirs, declared []string, path string) string {
	if !filepath.IsAbs(path) {
		return "a drive's stage's file tool names a file by its full path"
	}
	fold := approveFoldsCase()
	clean := filepath.Clean(path)
	if fi, err := os.Lstat(clean); err == nil && fi.Mode()&os.ModeSymlink != 0 {
		return "a drive's stage does not write through a link"
	}
	for _, candidate := range []string{clean, fsutil.RealExistingPath(clean)} {
		inside := false
		for _, d := range dirs {
			for _, dir := range []string{filepath.Clean(d), fsutil.RealExistingPath(filepath.Clean(d))} {
				if !fsutil.PathWithin(candidate, dir, fold) {
					continue
				}
				inside = true
				if name := protectedComponent(candidate, dir, fold); name != "" {
					return "a drive's stage does not write " + name + ", which is protected"
				}
				if name := stageGateFile(dir, candidate, declared, fold); name != "" {
					return "a drive's stage writes " + name + " only where its work declares it"
				}
			}
		}
		if !inside {
			return "a drive's stage writes only in its working tree and its run directories"
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
