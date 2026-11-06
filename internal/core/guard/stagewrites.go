package guard

import (
	"bytes"
	"crypto/sha256"
	"io"
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

// addsAWorktree reports whether the git command at tokens[idx] adds or moves a worktree, or
// has a subcommand or an action no line fixes where those would stand.
func addsAWorktree(tokens []string, idx int) bool {
	var args []string
	switch strings.ToLower(filepath.Base(tokens[idx])) {
	case "find":
		for i := idx + 1; i < len(tokens); i++ {
			switch tokens[i] {
			case "-exec", "-execdir", "-ok", "-okdir":
				if i+1 < len(tokens) && addsAWorktree(tokens, i+1) {
					return true
				}
			}
		}
		return false
	case "git-worktree":
		args = tokens[idx+1:]
	case "git":
		i := idx + 1
		for i < len(tokens) && strings.HasPrefix(tokens[i], "-") {
			if gitValueOptions[tokens[i]] || tokens[i] == "--attr-source" || tokens[i] == "--shallow-file" {
				i++
			}
			i++
		}
		if i >= len(tokens) {
			return false
		}
		if isUnknown(tokens[i]) {
			return true
		}
		// Case folds where the filesystem does, and a program in git's own directory answers.
		if !strings.EqualFold(tokens[i], "worktree") {
			return false
		}
		args = tokens[i+1:]
	default:
		return false
	}
	for _, w := range args {
		if strings.HasPrefix(w, "-") {
			continue
		}
		return isUnknown(w) || w == "add" || w == "move"
	}
	return false
}

// startsAnAgent reports whether the command at tokens[idx] starts a further
// harness or a drive, lands a branch, moves a run, or reads the keychain. A stage's
// tests go behind the load gate and an issue lane's brief has it widen its own hold, so
// `implement load` and `implement widen` stay open.
func startsAnAgent(tokens []string, idx int) bool {
	switch filepath.Base(tokens[idx]) {
	case "claude", "security":
		return true
	case "abcd", "abcd.sh":
		rest := tokens[idx+1:]
		var words []string
		for _, w := range rest {
			if !strings.HasPrefix(w, "-") {
				words = append(words, w)
			}
		}
		if len(words) > 0 && (words[0] == "land" || (words[0] == "implement" && (len(words) < 2 || (words[1] != "load" && words[1] != "widen")))) {
			return true
		}
		return slices.Contains(rest, "drain") && slices.ContainsFunc(rest, func(w string) bool {
			return w == "--drive" || strings.HasPrefix(w, "--drive=")
		})
	}
	return false
}

// stageAgentWhy and stageAgentInstead are that refusal's words.
const (
	stageAgentWhy     = "a drive's stage starts no further agent and no drive, moves no run and lands nothing: its work is the brief it was given, and the loop does the rest."
	stageAgentInstead = "do the work in this stage, or say in the receipt that the task needs another agent."
)

// stageVersionName is a program named as a release is named, as a native install names
// the harness's own file.
var stageVersionName = regexp.MustCompile(`^v?[0-9]+\.[0-9]+\.[0-9]+([-+.][0-9A-Za-z.+-]*)?$`)

// stageStartsHarness reports whether a program the placed line runs is named as a release
// is named, or is the harness file the run proved: that file through any link, or a copy
// of it. A file of another size is read no further, so the content is hashed only for one
// of the same size.
func (f *Files) stageStartsHarness(rc *readCtx, placed []placedSeg) bool {
	for _, p := range placed {
		for _, a := range commandSites(p.seg) {
			if stageVersionName.MatchString(filepath.Base(p.seg.tokens[a.idx])) {
				return true
			}
			for _, st := range p.states {
				if f.namesProvedHarness(rc, st, p.seg, a.idx) {
					return true
				}
			}
		}
	}
	return false
}

// namesProvedHarness reports whether word i of s, read where st places it, is the file
// the run proved or a copy of it.
func (f *Files) namesProvedHarness(rc *readCtx, st *shellState, s segment, i int) bool {
	if f.stageHarness == "" {
		return false
	}
	path, ok := "", false
	if tok := s.tokens[i]; strings.ContainsRune(tok, filepath.Separator) || strings.HasPrefix(tok, "~") {
		path, ok = st.resolveToken(rc, s, i)
	} else {
		path, ok = f.onPath(tok)
	}
	if !ok {
		return false
	}
	proved, err := f.stat(f.stageHarness, false)
	if err != nil {
		return false
	}
	named, err := f.stat(path, false)
	if err != nil || !named.Mode().IsRegular() {
		return false
	}
	return os.SameFile(proved, named) || (proved.Size() == named.Size() && sameContent(f.stageHarness, path))
}

// onPath is the first file named name in a directory of the PATH f reads.
func (f *Files) onPath(name string) (string, bool) {
	if isUnknown(name) || strings.ContainsRune(name, varMark) {
		return "", false
	}
	for _, dir := range filepath.SplitList(f.path) {
		if !filepath.IsAbs(dir) {
			continue
		}
		path := filepath.Join(dir, name)
		if fi, err := f.stat(path, false); err == nil && fi.Mode().IsRegular() {
			return path, true
		}
	}
	return "", false
}

// sameContent reports whether two files read the same; one that cannot be read does not.
func sameContent(a, b string) bool {
	sum := func(path string) ([]byte, bool) {
		file, err := os.Open(path)
		if err != nil {
			return nil, false
		}
		defer file.Close()
		h := sha256.New()
		if _, err := io.Copy(h, file); err != nil {
			return nil, false
		}
		return h.Sum(nil), true
	}
	sa, oka := sum(a)
	sb, okb := sum(b)
	return oka && okb && bytes.Equal(sa, sb)
}

// StageProgramRefusal is the guard hook's refusal of a line from a walled stage that starts
// a further harness, read as the approver reads it, "" when it does not or f is no stage's.
// A line the reading cannot take is the registry's to refuse.
func StageProgramRefusal(f *Files, line string) string {
	if f == nil || f.stageHome == "" {
		return ""
	}
	r := (&Registry{}).ReadingFrom(f)
	rc := newReadCtx(r.files)
	rc.record = true
	v, err := r.judge(line, rc, nil)
	if err != nil {
		return ""
	}
	placed := placedLine(rc.line, v.segs)
	refused := f.stageStartsHarness(rc, placed)
	for _, s := range v.segs {
		for _, a := range commandSites(s) {
			refused = refused || startsAnAgent(s.tokens, a.idx)
		}
	}
	if !refused {
		return ""
	}
	return "Blocked by the abcd guard (" + stageAgentEntryID + "): " + stageAgentWhy + " Run instead: " + stageAgentInstead
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

// nestedLocalGate is the local tier's gate rel lands in deeper in the tree, judged
// the same as the tier at the top.
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
