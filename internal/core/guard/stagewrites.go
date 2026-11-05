package guard

import (
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"

	"github.com/intentdriven/abcd/internal/abcdhome"
	"github.com/intentdriven/abcd/internal/fsutil"
)

// stageAgentEntryID is the refusal of a further agent from a walled stage.
const stageAgentEntryID = "stage-further-agent"

// stageGateFiles are the first parts of a path, below a stage's working tree, that
// decide how its lane is judged. No work's footprint opens them.
var stageGateFiles = []string{".githooks", "hooks", "Makefile", "GNUmakefile", "makefile"}

// stageAbcdGates are the files under .abcd that do the same, by path below the
// tree: each, and all below each.
var stageAbcdGates = []string{".abcd/rules.json", ".abcd/guard.json", ".abcd/config.json", ".abcd/config",
	".abcd/.work.local/preflight-receipts", ".abcd/.work.local/approvals", ".abcd/.work.local/review-receipts"}

// stageLocalTier is the one place under .abcd a stage writes freely, and
// stageAbcd the rest of it, which it writes where its work declares.
const (
	stageLocalTier = ".abcd/.work.local"
	stageAbcd      = ".abcd"
)

// stageMovesHome is a line that sets the home or unsets it.
var stageMovesHome = regexp.MustCompile(`(^|[^A-Za-z0-9_])HOME[ \t]*=|(^|[^A-Za-z0-9_])unset[ \t][^;&|]*(^|[^A-Za-z0-9_])HOME($|[^A-Za-z0-9_])`)

// stageLineRefused says why a stage's shell may not run line, "" when the line
// itself is no reason: its home stays the one it was given.
func stageLineRefused(line string) string {
	if stageMovesHome.MatchString(line) || strings.Contains(line, abcdhome.PersonHomeEnv) || strings.Contains(line, abcdhome.StageHomeEnv) {
		return "a drive's stage keeps the home its shell was given."
	}
	return ""
}

// startsAnAgent reports whether the command at tokens[idx] starts a further
// harness or a drive, lands a branch, or reads the keychain.
func startsAnAgent(tokens []string, idx int) bool {
	switch filepath.Base(tokens[idx]) {
	case "claude", "security":
		return true
	case "abcd", "abcd.sh":
		rest := tokens[idx+1:]
		if i := slices.IndexFunc(rest, func(w string) bool { return !strings.HasPrefix(w, "-") }); i >= 0 && rest[i] == "land" {
			return true
		}
		return slices.Contains(rest, "drain") && slices.ContainsFunc(rest, func(w string) bool {
			return w == "--drive" || strings.HasPrefix(w, "--drive=")
		})
	}
	return false
}

// stageGateFile is the gate file path lands in below tree, "" when it lands in
// none or tree does not hold it. What decides how a lane is judged is never
// opened by a declaration; a record under .abcd is, where declared names it.
func stageGateFile(tree, path string, declared []string, fold bool) string {
	if !fsutil.PathWithin(path, tree, fold) {
		return ""
	}
	rel, err := filepath.Rel(tree, path)
	if err != nil {
		return ""
	}
	rel = filepath.ToSlash(rel)
	below := func(top string) bool {
		if fold {
			return strings.EqualFold(rel, top) || (len(rel) > len(top) && strings.EqualFold(rel[:len(top)+1], top+"/"))
		}
		return rel == top || strings.HasPrefix(rel, top+"/")
	}
	if name := firstOf(stageAbcdGates, below); name != "" {
		return name
	}
	if name := nestedLocalGate(rel, fold); name != "" {
		return name
	}
	if name := firstOf(stageGateFiles, below); name != "" {
		return name
	}
	if !below(stageAbcd) || below(stageLocalTier) {
		return ""
	}
	for _, d := range declared {
		if d = filepath.ToSlash(filepath.Clean(d)); rel == d || strings.HasPrefix(rel, d+"/") {
			return ""
		}
	}
	return stageAbcd
}

// nestedLocalGate is the local tier's gate rel lands in deeper in the tree: a
// folder there can become a worktree of the repository, whose tier counts too.
func nestedLocalGate(rel string, fold bool) string {
	if fold {
		rel = strings.ToLower(rel)
	}
	return firstOf(stageAbcdGates, func(gate string) bool {
		if !strings.HasPrefix(gate, stageLocalTier+"/") {
			return false
		}
		g := "/" + gate
		if fold {
			g = strings.ToLower(g)
		}
		return strings.Contains(rel+"/", g+"/")
	})
}

// firstOf is the first of names that holds.
func firstOf(names []string, holds func(string) bool) string {
	for _, n := range names {
		if holds(n) {
			return n
		}
	}
	return ""
}

// StageWriteRefusal says why a drive's stage's file tool may not write path, ""
// when it may: such a stage writes only in dirs, its working tree and run
// directories, and nothing there the approver protects, no gate file, and no
// record its work does not declare. A path that is no absolute one, or whose last
// part is a link, is refused, and both the path as written and where it really
// lands are judged.
func StageWriteRefusal(dirs, declared []string, path string) string {
	if !filepath.IsAbs(path) {
		return "a drive's stage's file tool names a file by its full path"
	}
	if len(dirs) == 0 {
		return "a drive's stage writes only in its working tree and its run directories"
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
					return "a drive's stage writes " + name + " only where its work may, and its work does not"
				}
			}
		}
		if !inside {
			return "a drive's stage writes only in its working tree and its run directories"
		}
	}
	facts := factsFor(NewFiles(dirs[0]).WithRunDirs(dirs[1:]...))
	if name, ok := facts.protects(pathUse{path: clean, ok: true, written: path}); ok {
		return "a drive's stage does not write " + fsutil.DisplayPath(name) + ", which is protected"
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
