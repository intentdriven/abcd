package guard

// approvepaths.go is the approver's reading of what a line writes and
// removes (approve.go). It reads the guard's own parse: the targets the write
// model resolves (writesOf, scriptwrites.go), in every shell state the reading
// placed each segment in (readScripts), the operands of the commands that
// change or remove a path, and the configuration a git command writes. A
// target the line does not fix is refused; a fixed one is compared with the
// paths the harness protects, after its existing directories are resolved
// through symlinks, folding case where the filesystem folds it.

import (
	"os"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/intentdriven/abcd/internal/abcdhome"
	"github.com/intentdriven/abcd/internal/fsutil"
	"github.com/intentdriven/abcd/internal/gitutil"
	"github.com/intentdriven/abcd/internal/termsafe"
)

// approveFacts is what the approver knows of the place a command runs in,
// read afresh for every prompt.
type approveFacts struct {
	dir, home string
	fold      bool
	// protected is every path a write or a removal may not reach, each in
	// its spelling and with its existing directories resolved.
	protected []string
	// trees are the launched working tree and its home, each spelled both
	// ways, a protected name being judged relative to the nearest of them.
	trees []string
	// others are the repository's other checkouts: the main one and every
	// registered worktree but the launched one.
	others []string
	// userLayer is abcd's own folder in the home directory, each spelling:
	// the files that decide what abcd trusts on the machine. own are the
	// launched working tree and the run's own directories, which stay
	// writable inside it.
	userLayer, own []string
	// stage says the line is a walled stage's, and declared the paths its work
	// declares it changes (Files.WithStage).
	stage    bool
	declared []string
	// tree is the launched working tree, each spelling.
	tree []string
	// removable is where a removal may reach: the launched working tree, the
	// run's own directories and the temporary directories, each spelling.
	removable []string
	// hooksUnread, when git could not answer the read of the hooks
	// directories it runs here (gitutil.HooksPaths), names that read and why:
	// no path can then be checked against them, so no write or removal is
	// approved (hooksUnreadRefusal).
	hooksUnread string
	// gitTrees are the launched working tree and the repository's toplevel,
	// and ownGitDirs the git directories that are theirs: the common
	// directory and each one's .git, each spelling (approvegitdir.go).
	gitTrees, ownGitDirs []string
}

// approveTempDirs is the temporary directories a removal may reach: the
// system's, and /tmp; a variable so a test can name its own.
var approveTempDirs = func() []string { return []string{os.TempDir(), "/tmp"} }

// homeStartupFiles are the files a login or interactive shell reads from the
// home directory at start-up, and the directories some shells read whole.
var homeStartupFiles = []string{
	".bashrc", ".bash_profile", ".bash_login", ".bash_logout", ".bash_aliases", ".profile",
	".zshrc", ".zprofile", ".zshenv", ".zlogin", ".zlogout", ".config/fish",
}

// protectedNames are the names the harness protects from writes wherever they
// stand in a working tree (its own documentation, "Protected paths", read
// 2026-10-16): repository state, tool start-up files and its own
// configuration. Each is matched as a whole path component, relative to the
// working tree or the home directory the path is in.
var protectedNames = []string{
	".git", ".vscode", ".idea", ".husky", ".cargo", ".devcontainer", ".yarn", ".mvn", ".claude",
	".gitconfig", ".gitmodules", ".bashrc", ".bash_profile", ".bash_login", ".bash_aliases", ".bash_logout",
	".zshrc", ".zprofile", ".zshenv", ".zlogin", ".zlogout", ".profile", ".envrc",
	".npmrc", ".yarnrc", ".yarnrc.yml", ".pnp.cjs", ".pnp.loader.mjs", ".pnpmfile.cjs", "bunfig.toml", ".bunfig.toml",
	".bazelrc", ".bazelversion", ".bazeliskrc", ".pre-commit-config.yaml", "lefthook.yml", "lefthook.yaml",
	".lefthook.yml", ".lefthook.yaml", "gradle-wrapper.properties", "maven-wrapper.properties", ".devcontainer.json",
	".ripgreprc", "pyrightconfig.json", ".mcp.json", ".claude.json",
}

// factsFor reads the facts for a command run in f's directory: the
// repository's git common directory, its hooks directories and every
// checkout's .git, its worktrees, and the home directory's start-up files,
// git configuration and the host's settings.
func factsFor(f *Files) approveFacts {
	facts := approveFacts{dir: f.dir, home: f.home, fold: approveFoldsCase()}
	add := func(p string) {
		if p == "" || !filepath.IsAbs(p) {
			return
		}
		p = filepath.Clean(p)
		facts.protected = append(facts.protected, p)
		if real := fsutil.RealExistingPath(p); real != p {
			facts.protected = append(facts.protected, real)
		}
	}
	both := func(p string) []string {
		if real := fsutil.RealExistingPath(p); real != p {
			return []string{p, real}
		}
		return []string{p}
	}
	tops := []string{}
	if filepath.IsAbs(f.dir) {
		tops = append(tops, f.dir)
		facts.trees = append(facts.trees, both(f.dir)...)
		facts.gitTrees = append(facts.gitTrees, both(f.dir)...)
		facts.ownGitDirs = append(facts.ownGitDirs, both(filepath.Join(f.dir, ".git"))...)
		// The toplevel is asked through gitutil.Toplevel, which checks its
		// shape; either answer missing skips the repository's facts, as a
		// directory outside any checkout has none.
		top, err := gitutil.Toplevel(f.dir)
		var common string
		if err == nil {
			common, err = gitutil.RevParseAbsPath(f.dir, "--git-common-dir")
		}
		if err == nil {
			add(common)
			top = filepath.Clean(top)
			facts.ownGitDirs = append(facts.ownGitDirs, both(filepath.Clean(common))...)
			facts.ownGitDirs = append(facts.ownGitDirs, both(filepath.Join(top, ".git"))...)
			facts.gitTrees = append(facts.gitTrees, both(top)...)
			if top != f.dir {
				tops = append(tops, top)
			}
			if wts, err := gitutil.ListWorktrees(top, gitutil.MaxWorktreeListing); err == nil {
				for _, wt := range wts {
					p := filepath.Clean(wt.Path)
					if !filepath.IsAbs(p) || fsutil.PathWithin(fsutil.RealExistingPath(top), fsutil.RealExistingPath(p), facts.fold) &&
						fsutil.PathWithin(fsutil.RealExistingPath(p), fsutil.RealExistingPath(top), facts.fold) {
						continue
					}
					facts.others = append(facts.others, both(p)...)
					tops = append(tops, p)
				}
			}
			hooks, err := gitutil.HooksPaths(top)
			if err != nil {
				facts.hooksUnread = "the read of core.hooksPath (git config --get-all core.hooksPath) failed: " +
					termsafe.Sanitize(fsutil.RedactHome(err.Error()))
			}
			if err == nil {
				for _, h := range hooks {
					switch {
					case h == "":
					case filepath.IsAbs(h):
						add(h)
					default:
						// A relative hooks path is relative to the
						// working tree git runs the hook in: any of them.
						for _, t := range tops {
							add(filepath.Join(t, h))
						}
					}
				}
			}
		}
	}
	for _, t := range tops {
		for _, rel := range []string{".git", ".mcp.json", ".claude", RepoRelPath} {
			add(filepath.Join(t, filepath.FromSlash(rel)))
		}
	}
	if filepath.IsAbs(f.dir) {
		facts.tree = both(f.dir)
		facts.own = append(facts.own, facts.tree...)
	}
	for _, d := range f.runDirs {
		facts.own = append(facts.own, both(d)...)
	}
	if f.stageHome != "" {
		facts.stage, facts.declared = true, f.stageDeclared
		facts.own = append(facts.own, both(f.stageHome)...)
		facts.trees = append(facts.trees, both(f.stageHome)...)
		for _, p := range f.stageProtect {
			add(p)
		}
		for _, rel := range append(append([]string(nil), homeStartupFiles...), ".gitconfig", ".config/git") {
			add(filepath.Join(f.stageHome, filepath.FromSlash(rel)))
		}
	}
	facts.removable = append(facts.removable, facts.own...)
	for _, d := range approveTempDirs() {
		if d = filepath.Clean(d); filepath.IsAbs(d) {
			facts.removable = append(facts.removable, both(d)...)
		}
	}
	if filepath.IsAbs(f.home) {
		facts.userLayer = both(abcdhome.Path(f.home))
		facts.trees = append(facts.trees, both(f.home)...)
		for _, rel := range append(append([]string(nil), homeStartupFiles...), ".gitconfig", ".config/git", ".claude", ".claude.json") {
			add(filepath.Join(f.home, filepath.FromSlash(rel)))
		}
	}
	return facts
}

// pathUse is one path a command writes, removes or links to, as the line
// fixes it (ok) and as written, for the refusal to name.
type pathUse struct {
	path    string
	ok      bool
	remove  bool
	written string
	// rel marks a relative name after a change of directory the line leaves
	// unknown: the name is spelled out, the directory it is in is not.
	rel bool
	// glob, on a removal the line names by a glob, is the pattern after the
	// directory it lists (path), the part of the word the line does fix.
	glob string
	// bound marks a removal judged only for where it is (a find's starting
	// point that it deletes below, rsync --delete's destination), not as a
	// removal of every protected path inside it.
	bound bool
	// tree, in and top are the write's (writeTarget): a write that may place
	// a whole tree, judged for every path it can reach below where it lands.
	tree, in bool
	top      string
	// why is the writer's own reason for a write the guard cannot place.
	why *writeWhy
}

// unplaced is the shell state of a segment the reading did not place (a
// command a git alias expands to): its directory is not known.
func unplaced() *shellState {
	return &shellState{vars: map[string]varVal{}, exported: map[string]bool{}, fds: map[int]fdFile{}}
}

// placedLine is every segment the verdict read, with the states the reading
// placed it in; a segment the reading did not place runs in a directory the
// approver does not know.
func placedLine(line []placedSeg, segs []segment) []placedSeg {
	out := append([]placedSeg(nil), line...)
	for _, s := range segs {
		found := false
		for _, p := range line {
			if sameSegment(p.seg, s) {
				found = true
				break
			}
		}
		if !found {
			out = append(out, placedSeg{seg: s, states: []*shellState{unplaced()}})
		}
	}
	return out
}

// sameSegment reports whether a and b are one segment of the line: the same
// words and redirections, not a copy built from them.
func sameSegment(a, b segment) bool {
	if len(a.tokens) != len(b.tokens) || len(a.redirects) != len(b.redirects) {
		return false
	}
	switch {
	case len(a.tokens) > 0:
		return &a.tokens[0] == &b.tokens[0]
	case len(a.redirects) > 0:
		return &a.redirects[0] == &b.redirects[0]
	}
	return true
}

// pathUses is every path s writes, removes or links to in state st: the
// guard's write model, and the operands of the commands that change, remove
// or link a path.
func pathUses(rc *readCtx, s segment, st *shellState) []pathUse {
	var out []pathUse
	for _, w := range writesOf(rc, s, st) {
		written := w.path
		if !w.ok {
			written = w.base
		}
		out = append(out, pathUse{path: w.path, ok: w.ok, written: written, rel: !w.ok && w.rel, tree: w.tree, in: w.in, top: w.top, why: w.why})
	}
	at := func(i int, remove bool) pathUse {
		tok, ok := fixedToken(s, i)
		if !ok {
			return pathUse{remove: remove, written: tokenBase(s.tokens[i])}
		}
		p, ok := st.resolve(rc, tok, strings.HasPrefix(tok, "~"))
		return pathUse{path: p, ok: ok, remove: remove, written: tok, rel: !ok && st.relativeUnknown(tok)}
	}
	for _, a := range commandSites(s) {
		tok := s.tokens[a.idx]
		if isUnknown(tok) {
			continue
		}
		w := writerArgs{rc: rc, s: s, st: st, from: a.idx + 1}
		has := func(flags ...string) bool {
			for i := w.from; i < len(s.tokens); i++ {
				for _, f := range flags {
					if t := s.tokens[i]; t == f || strings.HasPrefix(t, f+"=") {
						return true
					}
				}
			}
			return false
		}
		switch strings.ToLower(path.Base(tok)) {
		case "find":
			// What a find deletes is below its starting points ("." with
			// none named).
			if !findDeleting(s, a.idx+1) {
				break
			}
			starts := findStarts(s, a.idx+1)
			if len(starts) == 0 {
				out = append(out, pathUse{path: st.dir, ok: st.dirOK && st.dir != "", remove: true, written: ".", bound: true})
			}
			for _, i := range starts {
				u := at(i, true)
				u.bound = true
				out = append(out, u)
			}
		case "rsync":
			if !flagGiven(s, a.idx+1, "--del") && !flagPrefixGiven(s, a.idx+1, "--delete") {
				break
			}
			if idx := w.operandIdx(rsyncValueFlags); len(idx) > 1 {
				if tok, ok := fixedToken(s, idx[len(idx)-1]); !ok || !remoteSpec(tok) {
					u := at(idx[len(idx)-1], true)
					u.bound = true
					out = append(out, u)
				}
			}
		case "touch":
			for _, i := range w.operandIdx([]string{"-d", "-t", "-r", "--date", "--reference"}) {
				out = append(out, at(i, false))
			}
		case "mkdir":
			for _, i := range w.operandIdx([]string{"-m", "--mode"}) {
				out = append(out, at(i, false))
			}
		case "chmod", "chown", "chgrp":
			idx := w.operandIdx(nil)
			if len(idx) > 0 && !has("--reference") {
				idx = idx[1:] // the mode, or the owner
			}
			for _, i := range idx {
				out = append(out, at(i, false))
			}
		case "truncate", "shred":
			for _, i := range w.operandIdx([]string{"-s", "--size", "-r", "--reference", "-n", "--iterations", "--random-source"}) {
				out = append(out, at(i, false))
			}
		case "rm", "rmdir", "unlink":
			for _, i := range w.operandIdx(nil) {
				if u, ok := globRemoval(rc, s, st, i); ok {
					out = append(out, u)
					continue
				}
				out = append(out, at(i, true))
			}
		case "mv":
			// The sources leave where they were: every operand with -t,
			// else every one but the destination, read as mv reads its
			// options and the words an xargs before it adds.
			view := xargsView(s, a.idx)
			for _, i := range eachSources(readOptsEach(view, a.idx+1, mvOpts)) {
				if i >= len(s.tokens) {
					out = append(out, pathUse{remove: true})
					continue
				}
				u := at(i, true)
				if view.tokens[i] != s.tokens[i] {
					u = pathUse{remove: true, written: tokenBase(view.tokens[i])}
				}
				out = append(out, u)
			}
		case "git":
			// git mv's sources leave where they were, as mv's do, and git
			// rm removes the files it names, but for --cached (the index
			// alone) or --dry-run.
			if _, reads, kind, ok := moverReads(s, a); ok && kind == "mv" {
				for _, i := range eachSources(reads) {
					out = append(out, at(i, true))
				}
				break
			}
			if j, elsewhere := w.gitSub(); !elsewhere && j < len(s.tokens) && s.tokens[j] == "rm" {
				r, _ := w.gitRead(j)
				if r.has(0, "cached") || r.has('n', "dry-run") {
					break
				}
				for _, i := range r.operands {
					if rc.files.stageHome != "" && pathspecMagic(s.tokens[i]) {
						why := pathspecUnplaced("rm").why
						out = append(out, pathUse{remove: true, why: why})
						continue
					}
					out = append(out, at(i, true))
				}
			}
		case "sed":
			// The files a sed script's w commands write, as sed opens them:
			// no tilde expansion, relative to the directory it runs in.
			scripts, _ := sedScripts(s, a.idx)
			for _, sc := range scripts {
				_, ws, _ := sedReads(sc)
				for _, name := range ws {
					if name == "" || name == "/dev/stdout" || name == "/dev/stderr" {
						continue
					}
					p, ok := st.resolve(rc, name, false)
					out = append(out, pathUse{path: p, ok: ok, written: name})
				}
			}
		case "ln":
			// A link to a protected path is a way to write it.
			for _, r := range readOptsEach(s, a.idx+1, lnOpts) {
				idx := r.operands
				if len(idx) > 1 {
					idx = copySources(r, idx)
				}
				for _, i := range idx {
					out = append(out, at(i, false))
				}
			}
		}
	}
	return out
}

// globRemoval reads operand i of s, when it is a glob the line otherwise
// fixes, as the directory it lists and the pattern after it: `build/*.o` is
// build and `*.o`. ok is false for any other operand.
func globRemoval(rc *readCtx, s segment, st *shellState, i int) (pathUse, bool) {
	tok := s.tokens[i]
	if !s.globAt(i) || isUnknown(tok) || strings.ContainsRune(tok, varMark) {
		return pathUse{}, false
	}
	first := strings.IndexAny(tok, "*?[")
	if first < 0 {
		return pathUse{}, false
	}
	dir, rest := ".", tok
	if slash := strings.LastIndexByte(tok[:first], '/'); slash >= 0 {
		dir, rest = tok[:slash], tok[slash+1:]
		if dir == "" {
			dir = "/"
		}
	}
	p, ok := st.resolve(rc, dir, strings.HasPrefix(dir, "~"))
	return pathUse{path: p, ok: ok, remove: true, written: tok, glob: rest, rel: !ok && st.relativeUnknown(dir)}, true
}

// maxGlobEntries bounds the entries of a globbed directory the approver reads
// for a protected name already standing there.
const maxGlobEntries = 4096

// globReaches names the protected path or name a glob removal u could reach,
// if it could reach one: its directory protected (a write into it would be),
// a protected name a component with literal text spells or can match, an
// entry standing in its directory that the pattern can match and that is or
// holds a protected path, or a protected path below its directory whose
// first name there the pattern can match. A pattern matches a name that
// starts with a dot only where it spells the dot, as the shell does by
// default; an entry already in the directory is matched either way.
func (f approveFacts) globReaches(u pathUse) (string, bool) {
	if name, ok := f.protects(pathUse{path: u.path, ok: true}); ok {
		return name, true
	}
	fold := func(x string) string { return fsutil.FoldPath(x, f.fold) }
	comps := strings.Split(u.glob, "/")
	for _, c := range comps {
		// A component of wildcards alone (`*`, `?*`) aims at nothing in
		// particular: what it can reach is what stands in the directory,
		// read below. One with literal text is read for the names it aims at.
		if strings.Trim(c, "*?") == "" {
			continue
		}
		for _, n := range protectedNames {
			if fold(c) == fold(n) || strings.ContainsAny(c, "*?[") && shellGlobMatch(fold(c), fold(n)) {
				return n, true
			}
		}
	}
	if d, err := os.Open(u.path); err == nil {
		entries, _ := d.ReadDir(maxGlobEntries)
		_ = d.Close()
		for _, e := range entries {
			if !globMatches(fold(comps[0]), fold(e.Name())) {
				continue
			}
			if name, ok := f.protects(pathUse{path: filepath.Join(u.path, e.Name()), ok: true, remove: true}); ok {
				return name, true
			}
		}
	}
	for _, p := range f.protected {
		if p == u.path || !fsutil.PathWithin(p, u.path, f.fold) {
			continue
		}
		if rel, err := filepath.Rel(u.path, p); err == nil {
			if c1, _, _ := strings.Cut(filepath.ToSlash(rel), "/"); shellGlobMatch(fold(comps[0]), fold(c1)) {
				return p, true
			}
		}
	}
	return "", false
}

// shellGlobMatch reports whether a glob component matches name the way the
// shell expands it by default: a name that starts with a dot only by a
// pattern that spells the dot.
func shellGlobMatch(pattern, name string) bool {
	if strings.HasPrefix(name, ".") && !strings.HasPrefix(pattern, ".") {
		return false
	}
	return globMatches(pattern, name)
}

// unfixedRefusal is the refusal of a write or a removal whose target the line
// does not fix, in the writer's own words where it gives them (why).
func unfixedRefusal(u pathUse) Approval {
	if u.why != nil {
		return refusal(unfixedPathEntryID, u.why.reason, u.why.remedy)
	}
	what := "a path"
	if u.written != "" {
		what = "a path ending " + termsafe.Sanitize(u.written)
	}
	verb := "writes"
	if u.remove {
		verb = "removes"
	}
	if u.rel && u.written != "" {
		return refusal(unfixedPathEntryID,
			"this "+verb+" "+termsafe.Sanitize(u.written)+", a relative name after a change of directory whose result the line leaves unknown (a directory held in a variable or a substitution's output, or a cd not joined to the command by &&), so the guard cannot say which directory the name is in.",
			"join the change of directory to the command with &&, to a directory spelled out in full, or name the file by its full path.")
	}
	return refusal(unfixedPathEntryID,
		"this "+verb+" "+what+" that the line does not spell out (a variable, a substitution's output, a glob, or a name after a change of directory the line leaves unknown), so the guard cannot say which file it is.",
		"the same command with the path spelled out in full.")
}

// hooksUnreadRefusal is the refusal of a write or a removal u where git could
// not answer the read of the hooks directories it runs (why, the facts'
// hooksUnread): the path cannot be shown not to be one of them.
func hooksUnreadRefusal(u pathUse, why string) Approval {
	what := "a path"
	if u.written != "" {
		what = termsafe.Sanitize(u.written)
	}
	return refusal(protectedPathEntryID,
		"this line writes or removes "+what+", and git could not say which hooks directories it runs here ("+why+"), so the guard cannot show the path is not one of them, and abcd does not approve it unattended.",
		"check that git can read this repository's settings and your own (git config --show-origin --get-all core.hooksPath), then run the line again.")
}

// parenRefusal is the refusal of a command with an unquoted ( in one of its
// words (segment.parenWord).
func parenRefusal() Approval {
	return refusal(unfixedPathEntryID,
		"an unquoted ( stands in a word of this command, which zsh reads as glob qualifiers or modifiers that can change the path the word names (x(:A) is the path a symlink leads to), and the words after it are not read as the command's own, so the guard cannot say which files it touches.",
		"the same command with every path spelled out in full, and any ( in a name quoted.")
}

// protects names the protected path u reaches, if it reaches one: a write or
// a link into one, a write that places a tree over one (a move, a copy, an
// archive unpacked), or a removal of one or of a directory holding one,
// judged as written and with its existing directories resolved through
// symlinks (a removal's own last name is not followed: it removes the link).
// A tree whose top names the line fixes only by a glob reaches each protected
// name the glob matches; one whose names no line fixes is judged by the
// protected paths below where it lands.
func (f approveFacts) protects(u pathUse) (string, bool) {
	cands := []string{u.path}
	if u.remove {
		cands = append(cands, filepath.Join(fsutil.RealExistingPath(filepath.Dir(u.path)), filepath.Base(u.path)))
	} else {
		cands = append(cands, fsutil.RealExistingPath(u.path))
	}
	for _, c := range cands {
		below := func(string) bool { return false }
		if u.tree {
			dir, names := treeLanding(c, u.in, u.top, f.fold)
			below = func(p string) bool { return treeReaches(dir, names, p, f.fold) }
		}
		for _, p := range f.protected {
			if fsutil.PathWithin(c, p, f.fold) || (u.remove && fsutil.PathWithin(p, c, f.fold)) || below(p) {
				return p, true
			}
		}
		for _, l := range f.userLayer {
			if (fsutil.PathWithin(c, l, f.fold) || (u.remove && fsutil.PathWithin(l, c, f.fold)) || below(l)) && !f.within(c, f.own) {
				return l, true
			}
		}
		if name, ok := f.protectedName(c); ok {
			return name, true
		}
		if u.tree && u.in && u.top != "" {
			for _, n := range protectedNames {
				if shellGlobMatch(fsutil.FoldPath(u.top, f.fold), fsutil.FoldPath(n, f.fold)) {
					return n, true
				}
			}
		}
		if f.stage {
			if !f.within(c, f.removable) && !strings.HasPrefix(c, "/dev/") {
				return c, true
			}
			for _, t := range f.tree {
				if name := stageGateFile(t, c, f.declared, f.fold); name != "" {
					return name, true
				}
				if name := stageGateLanded(t, c, u); name != "" {
					return name, true
				}
			}
		}
	}
	return "", false
}

// within reports whether p is in one of dirs, or is one.
func (f approveFacts) within(p string, dirs []string) bool {
	for _, d := range dirs {
		if fsutil.PathWithin(p, d, f.fold) {
			return true
		}
	}
	return false
}

// outsideRemoval reports whether u is a removal whose fixed target lies
// outside where a removal may reach (removable), judged as written and with
// its existing directories resolved through symlinks (its own last name not
// followed: a removal removes the link).
func (f approveFacts) outsideRemoval(u pathUse) bool {
	if !u.remove || !u.ok || u.path == "" {
		return false
	}
	return !f.withinResolved(u.path, f.removable, false)
}

// withinResolved reports whether p is in one of dirs, or is one, both as
// written and with its existing directories resolved through symlinks. Its
// own last name is followed only where followLast says: a removal removes a
// link, not what it leads to, while the directory a glob lists is what the
// link leads to.
func (f approveFacts) withinResolved(p string, dirs []string, followLast bool) bool {
	resolved := filepath.Join(fsutil.RealExistingPath(filepath.Dir(p)), filepath.Base(p))
	if followLast {
		resolved = fsutil.RealExistingPath(p)
	}
	return f.within(p, dirs) && f.within(resolved, dirs)
}

// protectedName reports a protected name among p's components, read relative
// to the working tree or the home directory p is in, else whole.
func (f approveFacts) protectedName(p string) (string, bool) {
	rel := p
	best := -1
	for _, t := range f.trees {
		if fsutil.PathWithin(p, t, f.fold) && len(t) > best {
			if r, err := filepath.Rel(t, p); err == nil {
				rel, best = r, len(t)
			}
		}
	}
	for _, c := range strings.Split(filepath.ToSlash(rel), "/") {
		for _, n := range protectedNames {
			if fsutil.FoldPath(c, f.fold) == fsutil.FoldPath(n, f.fold) {
				return n, true
			}
		}
	}
	return "", false
}

// The git configuration keys that start a program, folded: every key
// gitutil.ExecPins pins, read from it, and each setting whose job is to start
// a program that these pages document: git 2.52's git-config(1), and
// git-archive(1) for tar.<format>.command, which git-config(1) does not list,
// and Git LFS 3.7.1's git-lfs-config(5). The one rule: a setting is listed
// when its job is to name a command, a program path or a helper; to pick the
// tool, viewer, browser or signing program git starts (diff.tool, merge.tool,
// help.browser, help.format, man.viewer, instaweb.browser, gpg.format,
// merge.<driver>.recursive, lfs.standalonetransferagent); or to switch on a
// program git starts (the signing and verifying switches; protocol.allow and
// protocol.ext.allow, which let a remote address of the ext transport run the
// command it names; init.templateDir, whose hooks git init copies in). A key
// is matched exactly, or by its section and variable around any subsection,
// or by programKeyRE. hook.<name>.command is no setting of git 2.52; a later
// git runs it as a configured hook. submodule.<name>.update starts a program
// only for a value beginning with `!` (settingStartsProgram).
//
// Left off, as settings whose job is something else, even where a listed or
// default program may run after them: format.pretty and pretty.<name> (a
// format; a signature it checks starts gpg.program), merge.renormalize and
// core.attributesFile (which filters and drivers apply; each is listed by its
// own key), ssh.variant (how core.sshCommand's program is called),
// protocol.<name>.allow for any other transport, a remote's url and
// url.<base>.insteadOf (an address; the ext transport needs the protocol keys
// above), receive.procReceiveRefs (which pushed refs the proc-receive hook is
// given; the hook is a file in the hooks directory, judged as a path), and
// gitcvs.dbDriver (a Perl database driver git cvsserver loads by name; no
// program starts).
//
// A drive's hold across its lanes (driveHeldConfig and watchedConfig in
// internal/core/implement/loop) is a list of its own: it answers which
// settings the drive's own git commands read, network and trust settings
// included and programs no drive command starts (an alias, core.pager and
// pager.<cmd>, sequence.editor, send-email's commands) left out, while this
// one answers which settings an agent's command may make any git start. Each
// program class the drive holds is held here too:
// TestApproverListHoldsEachProgramClassTheDriveHolds.
var (
	programKeys = append(execPinKeys(), "core.sshcommand", "core.pager", "core.editor", "core.askpass",
		"core.gitproxy", "core.alternaterefscommand", "sequence.editor", "gpg.program", "gpg.ssh.defaultkeycommand",
		"diff.external", "credential.helper", "web.browser", "tag.gpgsign", "push.gpgsign", "include.path",
		"uploadpack.packobjectshook", "tag.forcesignannotated", "gc.recentobjectshook", "init.templatedir",
		"help.browser", "man.viewer", "imap.tunnel", "instaweb.browser", "instaweb.httpd", "instaweb.modulepath",
		"interactive.difffilter", "diff.tool", "diff.guitool", "merge.tool", "merge.guitool",
		"protocol.allow", "protocol.ext.allow", "help.format", "gpg.format")
	// programSubKeys are section.*.variable keys; a "" variable is any.
	programSubKeys = [][2]string{
		{"alias", ""}, {"pager", ""}, {"includeif", "path"}, {"credential", "helper"}, {"diff", "command"},
		{"diff", "textconv"}, {"difftool", "cmd"}, {"difftool", "path"}, {"mergetool", "cmd"}, {"mergetool", "path"},
		{"merge", "driver"}, {"merge", "recursive"}, {"filter", "clean"}, {"filter", "smudge"}, {"filter", "process"},
		{"browser", "cmd"}, {"browser", "path"}, {"man", "cmd"}, {"man", "path"}, {"guitool", "cmd"},
		{"remote", "uploadpack"}, {"remote", "receivepack"}, {"remote", "vcs"}, {"gpg", "program"},
		{"sendemail", "smtpserver"}, {"sendemail", "sendmailcmd"}, {"sendemail", "tocmd"}, {"sendemail", "cccmd"},
		{"sendemail", "headercmd"}, {"trailer", "cmd"}, {"trailer", "command"}, {"submodule", "update"},
		{"hook", "command"}, {"tar", "command"},
	}
	// programKeyRE matches the Git LFS keys whose subsection is itself
	// named: a custom transfer's program and its arguments (which the shell
	// expands), an extension's clean and smudge commands, and the
	// standalone transfer agent, plain or per URL.
	programKeyRE = regexp.MustCompile(`^lfs\.(customtransfer\..+\.(path|args)|extension\..+\.(clean|smudge)|(.+\.)?standalonetransferagent)$`)
)

// treeKeys are the configuration keys, folded, that move the working tree
// git reads and writes to another directory, outside the one the run's
// commands are judged in.
var treeKeys = []string{"core.worktree"}

// movesTree reports whether key, as a line spells it, is one of treeKeys.
func movesTree(key string) bool {
	return containsString(treeKeys, strings.ToLower(strings.TrimSpace(key)))
}

// treeKeyWhy is the refusal's reason for setting a key that moves the
// working tree.
func treeKeyWhy(k string) string {
	return "this sets " + keyName(k) + ", which moves the working tree git reads and writes to another directory, outside the one this run's commands are judged in."
}

// execPinKeys is the keys gitutil.ExecPins sets, folded.
func execPinKeys() []string {
	var out []string
	pins := gitutil.ExecPins()
	for i := 0; i+1 < len(pins); i += 2 {
		if pins[i] == "-c" {
			k, _, _ := strings.Cut(pins[i+1], "=")
			out = append(out, strings.ToLower(k))
		}
	}
	return out
}

// startsProgram reports whether key, as a line spells it, may be one that
// starts a program: a listed key, or one the line does not fix.
func startsProgram(key string) bool {
	if isUnknown(key) || strings.ContainsRune(key, varMark) {
		return true
	}
	k := strings.ToLower(strings.TrimSpace(key))
	if containsString(programKeys, k) || programKeyRE.MatchString(k) {
		return true
	}
	first, rest, ok := strings.Cut(k, ".")
	if !ok {
		return false
	}
	last := rest
	if i := strings.LastIndexByte(rest, '.'); i >= 0 {
		last = rest[i+1:]
	}
	for _, sk := range programSubKeys {
		if first == sk[0] && (sk[1] == "" || last == sk[1]) {
			return true
		}
	}
	return false
}

// settingStartsProgram reports whether setting key to value may start a
// program: startsProgram's answer, except that submodule.<name>.update starts
// one only for a value beginning with `!` (git-config(1)), so a value the
// line fixes that does not is no program, and an editor or askpass setting
// (noOpKeys) given a value noOpCommand takes starts none. known says the
// setting's value is given; a value is fixed when it is given and holds no
// substitution or variable.
func settingStartsProgram(key, value string, known bool) bool {
	if !startsProgram(key) {
		return false
	}
	k := strings.ToLower(strings.TrimSpace(key))
	if known && containsString(noOpKeys, k) && noOpCommand(value) {
		return false
	}
	if !strings.HasPrefix(k, "submodule.") || !strings.HasSuffix(k, ".update") || isUnknown(k) {
		return true
	}
	if !known || isUnknown(value) || strings.ContainsRune(value, varMark) {
		return true
	}
	return strings.HasPrefix(strings.TrimSpace(value), "!")
}

// programSections are the configuration sections, folded, that hold a listed
// key: a section renamed onto one of them may carry any of its keys.
func programSections() []string {
	out := []string{"lfs"}
	add := func(s string) {
		if !containsString(out, s) {
			out = append(out, s)
		}
	}
	for _, k := range append(append([]string{}, programKeys...), treeKeys...) {
		first, _, _ := strings.Cut(k, ".")
		add(first)
	}
	for _, sk := range programSubKeys {
		add(sk[0])
	}
	return out
}

// renameStartsProgram reports whether a section renamed to name, as a line
// spells it, may land on a section that holds a listed key, or on one the
// line does not spell out.
func renameStartsProgram(name string) bool {
	if isUnknown(name) || strings.ContainsRune(name, varMark) {
		return true
	}
	section, _, _ := strings.Cut(strings.ToLower(strings.TrimSpace(name)), ".")
	if i := strings.IndexAny(section, " \t"); i >= 0 {
		section = section[:i]
	}
	return containsString(programSections(), section)
}

// programFlags are, per git subcommand, the long options git documents as
// the command-line form of a listed setting: --upload-pack on clone, fetch,
// pull, ls-remote and fetch-pack, the --exec of ls-remote, archive and
// fetch-pack, and the --receive-pack and --exec of push and send-pack name
// the program git starts for the other side (remote.<name>.uploadpack and
// .receivepack), run on this machine when that side is a local repository
// (git-clone(1), git-fetch(1), git-pull(1), git-ls-remote(1),
// git-archive(1), git-push(1), git-fetch-pack(1), git-send-pack(1));
// --template on clone and init names the directory whose hooks git copies in
// (init.templateDir; git-clone(1), git-init(1)). fetch and pull are read for
// --exec as well, the name fetch-pack gives the same program: git 2.52's
// fetch and pull refuse it as an unknown option, so reading it there refuses
// only a command git would not run. Each takes a value. No other option of
// these subcommands is a strict prefix of one of them, so a word that is one
// is that option abbreviated, as git's option parser reads it, or one git
// refuses as ambiguous.
var programFlags = map[string][]string{
	"clone":      {"--upload-pack", "--template"},
	"init":       {"--template"},
	"fetch":      {"--upload-pack", "--exec"},
	"pull":       {"--upload-pack", "--exec"},
	"ls-remote":  {"--upload-pack", "--exec"},
	"archive":    {"--exec"},
	"push":       {"--receive-pack", "--exec"},
	"fetch-pack": {"--upload-pack", "--exec"},
	"send-pack":  {"--receive-pack", "--exec"},
}

// cloneShortValues are the letters of git clone's short options that take a
// value (git-clone(1)): -j, -o, -b, -u (--upload-pack) and -c (--config).
const cloneShortValues = "jobuc"

// longFlagIs reports whether a word can be the long option flag, written
// whole or abbreviated, with or without its `=value`.
func longFlagIs(w, flag string) bool {
	if isUnknown(w) {
		lead := knownLead(w)
		if strings.Contains(lead, "=") {
			return longFlagIs(lead, flag)
		}
		return flagCouldBe(w, flag)
	}
	name, _, _ := strings.Cut(w, "=")
	return name == flag || len(name) > 2 && strings.HasPrefix(name, "--") && strings.HasPrefix(flag, name)
}

// gitFlagRefusal reads one git subcommand's arguments, after its name, for
// an option that is the command-line form of a listed setting, and says why
// it is refused, if it is: one of programFlags, or a key git clone's -c or
// --config writes into the new repository's configuration that a `git
// config` write would be refused for.
func gitFlagRefusal(sub string, args []string) (string, bool) {
	subs := []string{sub}
	if isUnknown(sub) {
		subs = []string{"clone", "init", "fetch", "pull", "ls-remote", "archive", "push", "fetch-pack", "send-pack"}
	}
	var flags []string
	clone := false
	for _, s := range subs {
		flags = append(flags, programFlags[s]...)
		clone = clone || s == "clone"
	}
	for i := 0; i < len(args); i++ {
		w := args[i]
		if w == "--" {
			break
		}
		if !strings.HasPrefix(w, "-") {
			continue
		}
		for _, f := range flags {
			if longFlagIs(w, f) {
				return "this git command names, with " + f + ", " + flagWhat(f) + ", which the guard does not read.", true
			}
		}
		if !clone {
			continue
		}
		var setting string
		var given bool
		switch {
		case isUnknown(w) && (flagCouldBe(w, "--config") || clusterCouldCarry(w, 'c') || clusterCouldCarry(w, 'u')):
			return "this git clone carries an option the line does not spell out, which may name a program git starts or a key it writes into the new repository.", true
		case longFlagIs(w, "--config"):
			if _, v, ok := strings.Cut(w, "="); ok {
				setting, given = v, true
			} else if i+1 < len(args) {
				i++
				setting, given = args[i], true
			}
		case !strings.HasPrefix(w, "--"):
			letter, glued, ok := shortValueLetter(w, cloneShortValues)
			if !ok {
				continue
			}
			switch {
			case letter == 'u':
				return "this git command names, with -u, " + flagWhat("--upload-pack") + ", which the guard does not read.", true
			case letter != 'c':
				if glued == "" {
					i++
				}
				continue
			case glued != "":
				setting, given = glued, true
			case i+1 < len(args):
				i++
				setting, given = args[i], true
			}
		}
		if !given {
			continue
		}
		k, v, hasValue := strings.Cut(setting, "=")
		switch {
		case isUnknown(k) || strings.ContainsRune(k, varMark):
			return "this git clone writes " + keyName(k) + " into the new repository's configuration.", true
		case movesTree(k):
			return treeKeyWhy(k), true
		case settingStartsProgram(k, v, hasValue):
			return "this git clone writes " + keyName(k) + " into the new repository's configuration, a setting that starts a program the guard does not read.", true
		}
	}
	return "", false
}

// flagWhat names what one of programFlags gives git.
func flagWhat(flag string) string {
	if flag == "--template" {
		return "a template directory whose hooks git copies into the repository (as init.templateDir does)"
	}
	return "a program git starts for the other side of the transfer (as remote.<name>.uploadpack and .receivepack do)"
}

// shortValueLetter reads a short-option cluster for a parser whose
// value-taking letters are takes: the first such letter the cluster carries,
// and the value glued after it ("" when the next word is the value).
func shortValueLetter(w, takes string) (byte, string, bool) {
	if len(w) < 2 || w[0] != '-' || w[1] == '-' {
		return 0, "", false
	}
	for i := 1; i < len(w); i++ {
		if strings.IndexByte(takes, w[i]) >= 0 {
			return w[i], w[i+1:], true
		}
	}
	return 0, "", false
}

// programEnv maps each environment variable git documents as naming a
// program it starts, or the directory it takes one from, to the setting it
// stands for, as a refusal names it: git(1)'s GIT_SSH, GIT_SSH_COMMAND,
// GIT_PAGER, GIT_EDITOR, GIT_SEQUENCE_EDITOR, GIT_EXTERNAL_DIFF, GIT_ASKPASS,
// GIT_ALLOW_PROTOCOL and GIT_PROTOCOL_FROM_USER (both read as protocol.allow
// is) and GIT_EXEC_PATH (where git finds its sub-programs and transport
// helpers); git-var(1)'s PAGER, EDITOR and VISUAL, which git falls back to;
// git-config(1)'s SSH_ASKPASS and GIT_PROXY_COMMAND; git-init(1)'s
// GIT_TEMPLATE_DIR; and git-help(1)'s GIT_MAN_VIEWER. Left off, as naming no
// program: GIT_SSH_VARIANT (as ssh.variant), GIT_ATTR_SOURCE (as
// core.attributesFile), GIT_EXTERNAL_DIFF_TRUST_EXIT_CODE (how a listed
// program's exit is read), GIT_TERMINAL_PROMPT, GIT_SSL_NO_VERIFY and the
// GIT_TRACE family (a file written). GIT_CONFIG_* is read as the settings it
// carries (readGitConfig).
var programEnv = map[string]string{
	"GIT_SSH": "core.sshcommand", "GIT_SSH_COMMAND": "core.sshcommand",
	"GIT_PAGER": "core.pager", "PAGER": "core.pager",
	"GIT_EDITOR": "core.editor", "EDITOR": "core.editor", "VISUAL": "core.editor",
	"GIT_SEQUENCE_EDITOR": "sequence.editor", "GIT_EXTERNAL_DIFF": "diff.external",
	"GIT_ASKPASS": "core.askpass", "SSH_ASKPASS": "core.askpass",
	"GIT_TEMPLATE_DIR": "init.templatedir", "GIT_PROXY_COMMAND": "core.gitproxy",
	"GIT_ALLOW_PROTOCOL": "protocol.allow", "GIT_PROTOCOL_FROM_USER": "protocol.allow",
	"GIT_EXEC_PATH": "", "GIT_MAN_VIEWER": "man.viewer",
}

// commandEnv are the variables of programEnv whose value is a command line
// git runs through a shell, as core.pager's and core.sshCommand's is: a
// value the line fixes is judged as that command. GIT_SSH names a program
// alone, run with no shell, and keeps no such reading.
var commandEnv = []string{"GIT_PAGER", "PAGER", "GIT_SSH_COMMAND"}

// noOpEnv and noOpKeys are the editor and askpass variables of programEnv
// and their settings, folded, approved when the value is one noOpCommand
// takes. git appends a file to the editor's value and a prompt to askpass's
// before it runs them, so a value is judged by what it is, not as a command
// line of its own: only one that exits at once whatever follows it passes.
var (
	noOpEnv  = []string{"GIT_EDITOR", "GIT_SEQUENCE_EDITOR", "EDITOR", "VISUAL", "GIT_ASKPASS", "SSH_ASKPASS"}
	noOpKeys = []string{"core.editor", "sequence.editor", "core.askpass"}
)

// noOpCommand reports whether v, trimmed, is exactly `true` or `:`: a
// command that runs nothing and exits 0, whatever words git adds after it.
func noOpCommand(v string) bool {
	t := strings.TrimSpace(v)
	return t == "true" || t == ":"
}

// programEnvRefused names the first variable of programEnv, in name order,
// that env sets and that is refused: any of them, but one of commandEnv set
// to a value the line fixes that clean approves, and one of noOpEnv set to a
// value noOpCommand takes.
func programEnvRefused(env map[string]string, clean func(string) bool) (string, bool) {
	names := make([]string, 0, len(programEnv))
	for name := range programEnv {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		v, ok := env[name]
		if !ok {
			continue
		}
		if containsString(commandEnv, name) && clean != nil && !isUnknown(v) && !strings.ContainsRune(v, varMark) && clean(v) {
			continue
		}
		if containsString(noOpEnv, name) && noOpCommand(v) {
			continue
		}
		return name, true
	}
	return "", false
}

// commandKeys are the program keys, folded, whose value is the command git
// runs for that one invocation (through a shell, as a bang alias is): a
// `-c` setting of one to a command the line fixes is read as that command.
var commandKeys = []string{"core.pager", "core.sshcommand"}

// gitConfigRefusal reports a git command in s that writes configuration
// outside the repository's own file or sets a key that starts a program:
// `git config` with --global, --system or --file on a write, or setting such
// a key, or renaming a section onto one that holds such a key, or opening the
// editor, or with GIT_CONFIG set; any git command setting such a key for
// itself (-c, --config-env, GIT_CONFIG_*), reading configuration from a file
// the guard does not read (GIT_CONFIG_GLOBAL, GIT_CONFIG_SYSTEM, an include),
// or run with one of programEnv set; an option that is the command-line form
// of such a key (gitFlagRefusal); and an option that hands git a command line
// to run or picks the program it starts (gitRunRefusal). A `-c` setting of
// one of commandKeys, or one of commandEnv, to a value the line fixes is not
// refused when clean, which judges the value as the command line git runs,
// approves it.
func gitConfigRefusal(s segment, valueFlags []string, clean func(string) bool, carried map[string]string) (Approval, bool) {
	for _, site := range sitesNamed(s, "git") {
		args := s.tokens[site.idx+1:]
		if why, ok := execPathRefusal(args, valueFlags); ok {
			return refusal(gitConfigWriteEntryID, why, "the same git command without the option."), true
		}
		read := readGitConfigIn(carried, s.tokens[:site.idx], args, valueFlags)
		for i, k := range read.keys {
			if movesTree(k) {
				return refusal(gitConfigWriteEntryID, treeKeyWhy(k), "the same git command without the setting."), true
			}
			if v, ok := read.flagValues[i]; ok && containsString(commandKeys, k) && clean != nil &&
				!isUnknown(v) && !strings.ContainsRune(v, varMark) && clean(v) {
				continue
			}
			v, given := read.values[i]
			if settingStartsProgram(k, v, given) {
				return refusal(gitConfigWriteEntryID,
					"this git command sets "+keyName(k)+" for itself, a setting that starts a program the guard does not read.",
					"the same git command without the setting."), true
			}
		}
		if read.unread {
			// Set on the command itself, the same reading already raised the
			// guard's own warning; carried from an earlier command, only here.
			return refusal(gitConfigWriteEntryID,
				"this git command reads configuration the guard does not read (a file GIT_CONFIG_GLOBAL or GIT_CONFIG_SYSTEM names, a global configuration XDG_CONFIG_HOME moves, or one an include pulls in), which may set a program git starts or an alias for another command.",
				"the same git command without the variable or the setting."), true
		}
		if name, ok := programEnvRefused(read.env, clean); ok {
			key := programEnv[name]
			as := ""
			if key != "" {
				as = " (as " + key + " does)"
			}
			return refusal(gitConfigWriteEntryID,
				"this git command runs with "+name+" set, which names a program git starts"+as+" that the guard does not read.",
				"the same git command without the variable."), true
		}
		for _, f := range firstOperands(args, valueFlags) {
			if why, ok := gitFlagRefusal(args[f], args[f+1:]); ok {
				return refusal(gitConfigWriteEntryID, why,
					"the same git command without the option, or say in the receipt that the task needs it."), true
			}
			if why, ok := gitRunRefusal(args[f], args[f+1:], clean); ok {
				return refusal(gitConfigWriteEntryID, why,
					"the same git command with the command line it hands git spelled out in full, and one the guard clears, or without the option."), true
			}
			if args[f] != "config" && !isUnknown(args[f]) {
				continue
			}
			if why, ok := gitConfigWrite(args[f+1:], read.env); ok {
				return refusal(gitConfigWriteEntryID, why,
					"leave git's configuration as it is, or say in the receipt that the task needs it changed."), true
			}
		}
	}
	return Approval{}, false
}

// valueOf is the value a `git config` setting's operands give, after its
// key: "" when there is none.
func valueOf(ops []string) string {
	if len(ops) < 2 {
		return ""
	}
	return ops[1]
}

// execPathRefusal reads git's own options, before its subcommand, for
// --exec-path given a directory (git(1)): the option form of GIT_EXEC_PATH,
// the directory git runs its sub-programs and transport helpers from. The
// option with no word after it prints the path and is not refused. git's own
// options take no abbreviation.
func execPathRefusal(args, valueFlags []string) (string, bool) {
	limit := firstOperandLimit(args, valueFlags)
	for i := 0; i < limit && i < len(args); i++ {
		w := args[i]
		glued := strings.HasPrefix(w, "--exec-path=") ||
			isUnknown(w) && (strings.HasPrefix(knownLead(w), "--exec-path=") || unknownFlagCouldBe(w, "--exec-path=") && !strings.Contains(knownLead(w), "="))
		if glued || flagCouldBe(w, "--exec-path") && i+1 < len(args) {
			return "this git command names, with --exec-path, the directory git runs its sub-programs from (as GIT_EXEC_PATH does), which the guard does not read.", true
		}
	}
	return "", false
}

// carryEnvironment adds to env what one command of a line leaves in the
// environment of the commands after it: each NAME=value of a command that is
// nothing but assignments, and each NAME=value operand of export, declare,
// typeset, local or readonly (declarers). A plain assignment reaches a later
// command only once exported, or when the environment exports the name
// already, so each is carried. An operand whose name the line does not spell
// out may set any variable, and is carried as GIT_CONFIG_PARAMETERS the line
// does not fix, which readGitConfig reads as a key it cannot name. A
// program-naming or GIT_CONFIG_* name the command assigns by any other means
// (carryAssigned) is carried as a value the line does not fix.
//
// Which command reaches which is read past order only: a pipeline's or a
// subshell's assignment is carried too. Where the line runs a command again
// or later than its text places it, approveShellAt also takes the carry over
// the whole line, and judges each git command in it under that carry as well
// as under text order's, refusing where either refuses (runsOutOfTextOrder): a
// loop runs its body's export before the next pass's git command, a
// function's body runs where the function is called, an eval body is read
// after the line's other commands (its segments are placed at the end), and
// a ${NAME:=…} is assigned where its word is expanded, though its record
// rides on a segment placed at the end of the line. unset and env -u are
// not read as dropping a carried name: the carry also holds the stand-in
// for an operand the line does not spell out, which no name it unsets can
// be shown to remove, so a carried name only ever refuses more.
func carryEnvironment(s segment, env map[string]string) {
	// set is each name this command gives a value the line spells; bare each
	// name a declarer only names, which keeps the value it holds.
	set, bare := map[string]bool{}, map[string]bool{}
	assign := func(tok string) {
		name, eq, appends, subscript, ok := splitAssignment(tok)
		switch {
		case !ok:
			if isUnknown(tok) || strings.ContainsRune(tok, varMark) {
				env["GIT_CONFIG_PARAMETERS"] = unknownText
				set["GIT_CONFIG_PARAMETERS"] = true
			} else if isName(tok) {
				bare[tok] = true
			}
		case subscript:
		case appends:
			env[name], set[name] = unknownText+tok[eq+1:], true
		default:
			env[name], set[name] = tok[eq+1:], true
		}
	}
	sites := commandSites(s)
	if len(sites) == 0 {
		for _, tok := range s.tokens {
			if _, _, _, _, ok := splitAssignment(tok); ok {
				assign(tok)
			}
		}
	}
	for name := range declarers {
		for _, site := range sitesNamed(s, name) {
			for _, tok := range s.tokens[site.idx+1:] {
				if strings.HasPrefix(tok, "-") || tok == "" {
					continue
				}
				assign(tok)
			}
		}
	}
	carryAssigned(s, sites, env, set, bare)
}

// carriesProgram reports whether a carried name is one the approver reads on
// a git command: a variable of programEnv, one of the GIT_CONFIG family, or
// one that names the git directory git reads (gitDirEnv).
func carriesProgram(name string) bool {
	_, ok := programEnv[name]
	return ok || strings.HasPrefix(name, "GIT_CONFIG") || containsString(gitDirEnv, name)
}

// carryAssigned carries, as a value the line does not fix, each name
// namesAssigned reports s assigns that carriesProgram reads and that s gives
// no value the line spells (set) nor only names after a declarer (bare): a
// read, mapfile or printf -v operand, a for loop's name, a ${NAME:=…}, an
// array or an element of one. An assignment before a command that assigns
// none of its operands (git, make) holds for that command alone, and is the
// git reading's own to judge; one before a command that does (read, printf),
// or one the line does not name, counts as assigned. Only git itself keeps
// its prefix to itself here: a prefix before any other command (eval, a
// shell, a function the line defines, make) can reach a git command that
// command runs, so it is carried with the value it spells, or as a value
// the line does not fix where it spells none.
func carryAssigned(s segment, sites []arrival, env map[string]string, set, bare map[string]bool) {
	names := namesAssigned([]segment{s})
	prefixOnly := map[string]bool{}
	if len(sites) > 0 {
		first, plain, assigner := sites[0].idx, true, false
		for _, a := range sites {
			if s.tokens[a.idx] != "git" {
				plain = false
			}
			if nameCouldBeAny(s.tokens[a.idx], operandAssigners) {
				assigner = true
			}
			if a.idx < first {
				first = a.idx
			}
		}
		for i := 0; i < first; i++ {
			name, eq, appends, subscript, ok := splitAssignment(s.tokens[i])
			switch {
			case !ok || subscript || !carriesProgram(name):
			case plain:
				prefixOnly[name] = true
			case assigner || appends:
			default:
				env[name], set[name] = s.tokens[i][eq+1:], true
			}
		}
		for i := first; i < len(s.tokens); i++ {
			if name, _, _, _, ok := splitAssignment(s.tokens[i]); ok {
				delete(prefixOnly, name)
			}
		}
	}
	for name := range names {
		if set[name] || bare[name] || prefixOnly[name] || !carriesProgram(name) {
			continue
		}
		env[name] = unknownText
	}
}

// runsOutOfTextOrder reports whether a line runs a command again, or later
// than its text places it, so that what a command after a git command
// leaves in the environment can reach it: a loop (a command starting for,
// while, until or select), a function definition (a command starting
// function, or a lone word followed by a `{` group), an eval, or a
// ${NAME:=…} (its record a segment with no words). carryEnvironment says
// why each is read so. A trap's handler is read where its text stands, and
// a function whose body is a subshell rather than a `{` group is not told
// apart from a plain command.
func runsOutOfTextOrder(placed []placedSeg) bool {
	for k, p := range placed {
		s := p.seg
		if len(s.tokens) == 0 {
			if len(s.assignsInText) > 0 {
				return true
			}
			continue
		}
		switch s.tokens[0] {
		case "for", "while", "until", "select", "function":
			return true
		}
		if len(sitesNamed(s, "eval")) > 0 {
			return true
		}
		if len(s.tokens) == 1 && k+1 < len(placed) {
			if next := placed[k+1].seg.tokens; len(next) > 0 && next[0] == "{" {
				return true
			}
		}
	}
	return false
}

// keyName is a configuration key as a refusal names it.
func keyName(k string) string {
	if isUnknown(k) || strings.ContainsRune(k, varMark) {
		return "a key the line does not spell out"
	}
	return termsafe.Sanitize(k)
}

// gitConfigWrite reads one `git config` command's arguments, after the
// subcommand, and says why it is refused, if it is. env is the environment
// it runs with: GIT_CONFIG set there names the file git config reads and
// writes in place of the repository's (git-config(1)), so a write with it
// is refused as a --file write is.
func gitConfigWrite(args []string, env map[string]string) (string, bool) {
	var scope string
	if _, ok := env["GIT_CONFIG"]; ok {
		scope = "GIT_CONFIG set"
	}
	var read, write, setting, edit, unknownFlag, rename bool
	var ops []string
	for i := 0; i < len(args); i++ {
		t := args[i]
		switch {
		case isUnknown(t) || strings.ContainsRune(t, varMark):
			// Any word: a scope, a mode, or the key.
			unknownFlag = true
			ops = append(ops, t)
		case t == "--":
			ops = append(ops, args[i+1:]...)
			i = len(args)
		case t == "--global" || t == "--system":
			scope = t
		case t == "--file" || t == "-f":
			scope = "--file"
			i++
		case strings.HasPrefix(t, "--file="), strings.HasPrefix(t, "-f") && !strings.HasPrefix(t, "--"):
			scope = "--file"
		case t == "--blob":
			read = true
			i++
		case strings.HasPrefix(t, "--blob="), t == "-l", t == "--list", strings.HasPrefix(t, "--get"):
			read = true
		case t == "--add" || t == "--replace-all":
			write, setting = true, true
		case t == "--rename-section":
			write, rename = true, true
		case t == "--unset" || t == "--unset-all" || t == "--remove-section":
			write = true
		case t == "-e" || t == "--edit":
			write, edit = true, true
		case t == "--type" || t == "--default" || t == "--comment" || t == "--value":
			i++
		case strings.HasPrefix(t, "-"):
			// Another option (--local, --worktree, --type=path, --bool …).
		default:
			ops = append(ops, t)
		}
	}
	if len(ops) > 0 {
		switch ops[0] {
		case "get", "list":
			read, ops = true, ops[1:]
		case "set":
			write, setting, ops = true, true, ops[1:]
		case "rename-section":
			write, rename, ops = true, true, ops[1:]
		case "unset", "remove-section":
			write, ops = true, ops[1:]
		case "edit":
			write, edit, ops = true, true, ops[1:]
		}
	}
	if !read && !write && len(ops) >= 2 {
		write, setting = true, true
	}
	if unknownFlag && !read {
		write = true
	}
	switch {
	case edit:
		return "this opens git's configuration in an editor, a program the guard does not read.", true
	case write && scope != "":
		return "this writes git configuration with " + scope + ", outside the repository's own configuration, which every repository on the machine (or the file it names) reads.", true
	case write && unknownFlag:
		return "this writes git configuration with a word the line does not spell out, which may name a scope outside the repository or any key.", true
	case setting && len(ops) > 0 && movesTree(ops[0]):
		return treeKeyWhy(ops[0]), true
	case rename && (len(ops) < 2 || renameStartsProgram(ops[1])):
		to := "a name the line does not spell out"
		if len(ops) >= 2 && !isUnknown(ops[1]) && !strings.ContainsRune(ops[1], varMark) {
			to = termsafe.Sanitize(ops[1])
		}
		return "this renames a section of git's configuration to " + to + ", which may hold a setting that starts a program the guard does not read.", true
	case setting && (len(ops) == 0 || settingStartsProgram(ops[0], valueOf(ops), len(ops) >= 2)):
		key := "a key"
		if len(ops) > 0 {
			key = keyName(ops[0])
		}
		return "this sets " + key + " in git's configuration, a setting that starts a program the guard does not read.", true
	}
	return "", false
}

// unfixedTreeRefusal is the refusal of a stage's write or removal u below a
// tree the command by placed earlier on the line, whose contents the line
// does not fix: any name in it may be a link leading anywhere.
func unfixedTreeRefusal(u pathUse, by string) Approval {
	verb := "writes"
	if u.remove {
		verb = "removes"
	}
	what := "a path"
	if u.written != "" {
		what = termsafe.Sanitize(u.written)
	}
	return refusal(unfixedPathEntryID,
		"this "+verb+" "+what+" below what "+termsafe.Sanitize(by)+" placed earlier on the same line, whose contents the line does not fix: any name in it may be a link leading out of the working tree, which the guard cannot read before the line runs.",
		"run the write as a separate command, after "+termsafe.Sanitize(by)+" has run, so the guard reads what it placed.")
}

// lineLink is a link a command on the line makes, at path, made by segment
// seg of the placed line: where it leads (target), with ok false where the
// line does not fix that; anyName marks links made in the directory path under
// names the line does not fix. text is a symbolic link's own relative text
// (rel), which leads from wherever the link is moved or copied to.
type lineLink struct {
	path, target string
	ok, anyName  bool
	seg          int
	text         string
	rel          bool
}

// lineTree is a tree a command on the line places, by segment seg: at path
// at, a copy of what stood at src (srcOK, else the line does not fix what the
// tree holds); in marks a tree of names landing in the directory at, matching
// top (any name when top is "").
type lineTree struct {
	at, src   string
	srcOK, in bool
	top       string
	seg       int
	// by is the command that placed it, as a refusal names it.
	by string
}

// lineMoves is what the line's commands move about before a later command
// writes: the links they make (lineLinks) and the trees they place.
type lineMoves struct {
	links []lineLink
	trees []lineTree
}

// lineLinks is every link the placed line's commands make, and every tree
// they place: what `ln` makes (a symbolic link leads where its text names,
// read from the link's own directory; a hard link is the file it names), what
// `cp -s` makes, and a link `cp` or `mv` places by copying or moving one,
// standing now or made earlier on the line, to where that one leads. A tree
// a move or copy places carries the links made earlier on the line below its
// source to the same place below where it lands. They are read in text
// order, so a link moved after it is made is followed to the first one's
// target.
func lineLinks(rc *readCtx, placed []placedSeg) lineMoves {
	var m lineMoves
	for k, p := range placed {
		for _, st := range p.states {
			for _, a := range commandSites(p.seg) {
				name := strings.ToLower(path.Base(p.seg.tokens[a.idx]))
				if s, reads, kind, ok := moverReads(p.seg, a); ok {
					m.linksAt(rc, s, st, a, kind, reads, k)
				}
				m.treesAt(rc, p.seg, st, a, name, k)
			}
		}
	}
	return m
}

// moverReads is how the command at a in seg moves, copies or links its
// operands, where it is one that does (cp, mv, install, ln, and git mv, which
// moves as mv does): the segment as the command receives its words
// (xargsView), each reading of them, and the command's kind; ok is false for
// any other command, and for a git that -C or --work-tree moves elsewhere,
// whose writes are not placed at all.
func moverReads(seg segment, a arrival) (s segment, reads []optRead, kind string, ok bool) {
	switch name := strings.ToLower(path.Base(seg.tokens[a.idx])); name {
	case "cp", "mv", "install", "ln":
		s = xargsView(seg, a.idx)
		return s, readOptsEach(s, a.idx+1, copySpec(name)), name, true
	case "git":
		w := writerArgs{s: seg, from: a.idx + 1}
		if j, elsewhere := w.gitSub(); !elsewhere && j < len(seg.tokens) && seg.tokens[j] == "mv" {
			r, _ := w.gitRead(j)
			return seg, []optRead{r}, "mv", true
		}
	}
	return seg, nil, "", false
}

// linksAt adds the links the cp, mv, install or ln at a in segment s of the
// line, k, makes, read as reads (moverReads).
func (m *lineMoves) linksAt(rc *readCtx, s segment, st *shellState, a arrival, name string, reads []optRead, k int) {
	w := writerArgs{rc: rc, s: s, st: st, from: a.idx + 1}
	var pairs []linkPair
	symbolic := false
	for _, r := range reads {
		if name == "install" {
			// macOS install -l makes a link to each source instead of a
			// copy, whatever its flags: a symbolic link's text is the source
			// made absolute or relative, so it leads where the source is.
			if v, ok := r.last('l', ""); !ok || !v.ok {
				continue
			}
			symbolic = true
		}
		symbolic = symbolic || (name == "ln" && r.has('s', "symbolic")) || (name == "cp" && r.has('s', "symbolic-link"))
		pairs = append(pairs, w.linkPairs(r, name == "ln")...)
	}
	fold := approveFoldsCase()
	for _, pr := range pairs {
		if !pr.at.ok {
			continue
		}
		l := lineLink{path: pr.at.path, anyName: pr.at.in, seg: k}
		dir := filepath.Dir(l.path)
		if l.anyName {
			dir = l.path
		}
		tok, fixed := fixedToken(s, pr.src)
		var target string
		switch {
		case !fixed && (name == "ln" || symbolic):
		case !fixed:
			continue
		case name == "install":
			target, l.ok = st.resolve(rc, tok, strings.HasPrefix(tok, "~"))
		case symbolic && !filepath.IsAbs(tok) && !strings.HasPrefix(tok, "~"):
			target, l.ok = filepath.Join(dir, tok), true
			l.text, l.rel = tok, true
		case name == "ln" || symbolic:
			target, l.ok = st.resolve(rc, tok, strings.HasPrefix(tok, "~"))
		default:
			src, ok := st.resolve(rc, tok, strings.HasPrefix(tok, "~"))
			if !ok {
				continue
			}
			if t, through, unfixed := m.through(pathUse{path: src, ok: true}, -1, false, fold); through || unfixed {
				target, l.ok = t.path, !unfixed
			} else if fi, err := os.Lstat(src); err == nil && fi.Mode()&os.ModeSymlink != 0 {
				target, l.ok = src, true
				if text, err := os.Readlink(src); err == nil && !filepath.IsAbs(text) {
					target, l.text, l.rel = filepath.Join(dir, text), text, true
				}
			} else {
				continue
			}
		}
		m.add(l, target, fold)
	}
}

// add appends l, leading to target, judged through what the line moved
// before it.
func (m *lineMoves) add(l lineLink, target string, fold bool) {
	if l.ok {
		if t, through, unfixed := m.through(pathUse{path: target, ok: true}, -1, false, fold); unfixed {
			l.ok = false
		} else if through {
			target = t.path
		}
		l.target = fsutil.RealExistingPath(target)
	}
	m.links = append(m.links, l)
}

// treesAt adds the trees the command at a in segment k of the line places:
// for cp and mv (git mv among them, moverReads), each source's tree where it lands (copyPlan), with the links
// made earlier below the source carried along; for any other writer that
// places a tree (rsync, ditto, an archive unpacked, a clone), a tree whose
// contents the line does not fix. ln places a link, which linksAt reads, and
// install copies files, not trees.
func (m *lineMoves) treesAt(rc *readCtx, seg segment, st *shellState, a arrival, name string, k int) {
	s, reads, kind, mover := moverReads(seg, a)
	switch {
	case kind == "ln" || kind == "install":
		return
	case mover:
	case name == "git":
		w := writerArgs{rc: rc, s: seg, st: st, from: a.idx + 1}
		for _, t := range w.gitTrees() {
			m.trees = append(m.trees, lineTree{at: t.path, seg: k, by: "git"})
		}
		fallthrough
	default:
		for _, t := range siteTargets(rc, seg, st, a) {
			if t.ok && t.tree {
				m.trees = append(m.trees, lineTree{at: t.path, in: t.in, top: t.top, seg: k, by: name})
			}
		}
		return
	}
	w := writerArgs{rc: rc, s: s, st: st, from: a.idx + 1}
	fold := approveFoldsCase()
	for _, r := range reads {
		for _, operands := range r.readings() {
			p := w.planCopy(r, operands, 1)
			if !p.ok || !p.dest.ok {
				continue
			}
			for _, i := range p.srcs {
				tok, fixed := fixedToken(s, i)
				src, ok := "", false
				if fixed && !s.globAt(i) {
					src, ok = st.resolve(rc, tok, strings.HasPrefix(tok, "~"))
				}
				base := path.Base(tok)
				if !ok || strings.HasSuffix(tok, "/") || base == ".." || base == "/" {
					m.trees = append(m.trees, lineTree{at: p.dest.path, in: true, seg: k, by: name})
					continue
				}
				var ats []string
				switch {
				case base == ".":
					ats = append(ats, p.dest.path)
				case p.into:
					ats = append(ats, filepath.Join(p.dest.path, base))
					if p.asWell {
						ats = append(ats, p.dest.path)
					}
				default:
					ats = append(ats, p.dest.path)
				}
				for _, at := range ats {
					m.place(lineTree{at: at, src: src, srcOK: true, seg: k}, fold)
				}
			}
		}
	}
}

// place appends t, a copy of what stood at its source: a source below a tree
// placed earlier on the line is read back to what stood there (or as not
// fixed, below a tree whose contents the line does not fix), and each link
// made earlier on the line below the source is carried to the same place
// below t, a relative one leading from its new place.
func (m *lineMoves) place(t lineTree, fold bool) {
	for i := len(m.trees) - 1; i >= 0; i-- {
		e := m.trees[i]
		rel, ok := fsutil.RelWithin(e.at, t.src, fold)
		if !ok || rel == "." || !e.covers(rel) {
			continue
		}
		if !e.srcOK {
			m.trees = append(m.trees, lineTree{at: t.at, in: true, seg: t.seg, by: e.by})
			return
		}
		t.src = filepath.Join(e.src, rel)
	}
	for _, l := range m.links {
		rel, ok := fsutil.RelWithin(t.src, l.path, fold)
		if !ok || rel == "." {
			continue
		}
		c := l
		c.path, c.seg = filepath.Join(t.at, rel), t.seg
		target := l.target
		if l.rel {
			target = filepath.Join(filepath.Dir(c.path), l.text)
		}
		m.add(c, target, fold)
	}
	m.trees = append(m.trees, t)
}

// covers says whether the path rel below a tree's place lies in the tree: in
// a tree of names landing in a directory, its first name must match top.
func (t lineTree) covers(rel string) bool {
	if !t.in || t.top == "" {
		return true
	}
	first, _, _ := strings.Cut(filepath.ToSlash(rel), "/")
	return shellGlobMatch(t.top, first) || t.top == first
}

// linkPair is a link a command places at at, made from its operand src.
type linkPair struct {
	at  writeTarget
	src int
}

// linkPairs is each link a move, copy or link command read by r places, by
// copyPlan's reading; ln with a single operand links it in the directory it
// runs in under its own name.
func (w writerArgs) linkPairs(r optRead, ln bool) []linkPair {
	if _, target := r.last('t', "target-directory"); ln && len(r.operands) == 1 && !target {
		i := r.operands[0]
		tok, ok := fixedToken(w.s, i)
		if !ok {
			return nil
		}
		return []linkPair{{writeAt(w.rc, w.st, path.Base(tok), false), i}}
	}
	min := 1
	if ln {
		min = 2
	}
	p := w.planCopy(r, r.operands, min)
	if !p.ok || !p.dest.ok {
		return nil
	}
	var out []linkPair
	for _, i := range p.srcs {
		if p.into {
			out = append(out, linkPair{w.landingIn(p.dest, i), i})
		} else {
			out = append(out, linkPair{p.dest, i})
		}
	}
	return out
}

// through is u reached through what the line moved before it (lineMoves),
// but what segment seg moves itself: its path where each link it passes
// leads, in turn, whether the line made the link or a tree the line placed
// carried it there from where it stands now. through is false where it passes
// none; unfixed says it passes one that leads where the line does not fix,
// or, in a stage, lies below a tree whose contents the line does not fix,
// any name in which may be a link. A removal of a link itself passes none.
func (m lineMoves) through(u pathUse, seg int, stage, fold bool) (v pathUse, through, unfixed bool) {
	v, through, unfixed, _ = m.follow(u, seg, stage, fold)
	return v, through, unfixed
}

// follow is through, also naming the command that placed the tree whose
// contents the line does not fix (by), where that is what stopped it; a
// removal of a single name directly below such a tree removes that name, a
// link there included, and passes nothing.
func (m lineMoves) follow(u pathUse, seg int, stage, fold bool) (v pathUse, through, unfixed bool, by string) {
	v = u
	for range len(m.links) + len(m.trees) + 8 {
		moved := false
		for _, l := range m.links {
			if l.seg == seg {
				continue
			}
			rel, ok := fsutil.RelWithin(l.path, v.path, fold)
			if !ok || (rel == "." && (v.remove || l.anyName)) {
				continue
			}
			if l.anyName {
				if !strings.Contains(filepath.ToSlash(rel), "/") {
					continue
				}
				return v, true, true, ""
			}
			if !l.ok {
				return v, true, true, ""
			}
			v.path = filepath.Join(l.target, rel)
			moved, through = true, true
			break
		}
		if moved {
			continue
		}
		for _, t := range m.trees {
			if t.seg == seg || (seg >= 0 && t.seg > seg) {
				continue
			}
			rel, ok := fsutil.RelWithin(t.at, v.path, fold)
			if !ok || rel == "." || !t.covers(rel) {
				continue
			}
			if !t.srcOK {
				if stage && !(v.remove && !strings.Contains(filepath.ToSlash(rel), "/")) {
					return v, true, true, t.by
				}
				continue
			}
			if p, ok := t.standing(rel, v.remove); ok {
				v.path, moved, through = p, true, true
				break
			}
		}
		if !moved {
			return v, through, false, ""
		}
	}
	return v, through, true, ""
}

// standing is the path rel below tree t's place leads to through a link
// standing now below t's source: each name of rel is read at the source, and
// the first that is a link leads, from its new place, where its text names;
// ok is false where none is. The last name of a removal is the link itself.
func (t lineTree) standing(rel string, remove bool) (string, bool) {
	names := strings.Split(filepath.ToSlash(rel), "/")
	src, at := t.src, t.at
	for i, n := range names {
		src, at = filepath.Join(src, n), filepath.Join(at, n)
		if i == len(names)-1 && remove {
			return "", false
		}
		fi, err := os.Lstat(src)
		if err != nil {
			return "", false
		}
		if fi.Mode()&os.ModeSymlink == 0 {
			continue
		}
		text, err := os.Readlink(src)
		if err != nil {
			return "", false
		}
		if !filepath.IsAbs(text) {
			text = filepath.Join(filepath.Dir(at), text)
		}
		return filepath.Join(append([]string{text}, names[i+1:]...)...), true
	}
	return "", false
}
