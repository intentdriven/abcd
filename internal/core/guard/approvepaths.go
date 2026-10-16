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
	"path"
	"path/filepath"
	"strings"

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
}

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
		if out, err := gitutil.Run(f.dir, "rev-parse", "--path-format=absolute", "--git-common-dir", "--show-toplevel"); err == nil {
			if lines := strings.Split(out, "\n"); len(lines) == 2 {
				add(lines[0])
				top := filepath.Clean(lines[1])
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
				if hooks, err := gitutil.HooksPaths(top); err == nil {
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
	}
	for _, t := range tops {
		for _, rel := range []string{".git", ".mcp.json", ".claude", RepoRelPath} {
			add(filepath.Join(t, filepath.FromSlash(rel)))
		}
	}
	if filepath.IsAbs(f.home) {
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
		out = append(out, pathUse{path: w.path, ok: w.ok, written: written, rel: !w.ok && w.rel})
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
				out = append(out, at(i, true))
			}
		case "mv":
			// The sources leave where they were: every operand with -t,
			// else every one but the destination.
			idx := w.operandIdx([]string{"-S", "--suffix", "-t", "--target-directory"})
			if !has("-t", "--target-directory") && len(idx) > 0 {
				idx = idx[:len(idx)-1]
			}
			for _, i := range idx {
				out = append(out, at(i, true))
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
			idx := w.operandIdx([]string{"-S", "--suffix", "-t", "--target-directory"})
			if !has("-t", "--target-directory") && len(idx) > 1 {
				idx = idx[:len(idx)-1]
			}
			for _, i := range idx {
				out = append(out, at(i, false))
			}
		}
	}
	return out
}

// unfixedRefusal is the refusal of a write or a removal whose target the line
// does not fix.
func unfixedRefusal(u pathUse) Approval {
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

// protects names the protected path u reaches, if it reaches one: a write or
// a link into one, or a removal of one or of a directory holding one, judged
// as written and with its existing directories resolved through symlinks (a
// removal's own last name is not followed: it removes the link).
func (f approveFacts) protects(u pathUse) (string, bool) {
	cands := []string{u.path}
	if u.remove {
		cands = append(cands, filepath.Join(fsutil.RealExistingPath(filepath.Dir(u.path)), filepath.Base(u.path)))
	} else {
		cands = append(cands, fsutil.RealExistingPath(u.path))
	}
	for _, c := range cands {
		for _, p := range f.protected {
			if fsutil.PathWithin(c, p, f.fold) || (u.remove && fsutil.PathWithin(p, c, f.fold)) {
				return p, true
			}
		}
		if name, ok := f.protectedName(c); ok {
			return name, true
		}
	}
	return "", false
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
// gitutil.ExecPins pins, read from it, and the others git documents. A key is
// matched exactly, or by its section and variable around any subsection.
var (
	programKeys = append(execPinKeys(), "core.sshcommand", "core.pager", "core.editor", "core.askpass",
		"core.gitproxy", "core.alternaterefscommand", "sequence.editor", "gpg.program", "gpg.ssh.defaultkeycommand",
		"diff.external", "credential.helper", "web.browser", "tag.gpgsign", "push.gpgsign", "include.path",
		"uploadpack.packobjectshook", "sendemail.smtpserver", "sendemail.sendmailcmd", "sendemail.tocmd", "sendemail.cccmd")
	// programSubKeys are section.*.variable keys; a "" variable is any.
	programSubKeys = [][2]string{
		{"alias", ""}, {"pager", ""}, {"includeif", "path"}, {"credential", "helper"}, {"diff", "command"},
		{"diff", "textconv"}, {"difftool", "cmd"}, {"difftool", "path"}, {"mergetool", "cmd"}, {"mergetool", "path"},
		{"merge", "driver"}, {"filter", "clean"}, {"filter", "smudge"}, {"filter", "process"}, {"browser", "cmd"},
		{"browser", "path"}, {"man", "cmd"}, {"man", "path"}, {"remote", "uploadpack"}, {"remote", "receivepack"},
		{"remote", "vcs"}, {"gpg", "program"},
	}
)

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
	if containsString(programKeys, k) {
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

// gitConfigRefusal reports a git command in s that writes configuration
// outside the repository's own file or sets a key that starts a program:
// `git config` with --global, --system or --file on a write, or setting such
// a key, or opening the editor; and any git command setting such a key for
// itself (-c, --config-env, GIT_CONFIG_*).
func gitConfigRefusal(s segment, valueFlags []string) (Approval, bool) {
	for _, site := range sitesNamed(s, "git") {
		args := s.tokens[site.idx+1:]
		for _, k := range readGitConfig(s.tokens[:site.idx], args, valueFlags).keys {
			if startsProgram(k) {
				return refusal(gitConfigWriteEntryID,
					"this git command sets "+keyName(k)+" for itself, a setting that starts a program the guard does not read.",
					"the same git command without the setting."), true
			}
		}
		for _, f := range firstOperands(args, valueFlags) {
			if args[f] != "config" && !isUnknown(args[f]) {
				continue
			}
			if why, ok := gitConfigWrite(args[f+1:]); ok {
				return refusal(gitConfigWriteEntryID, why,
					"leave git's configuration as it is, or say in the receipt that the task needs it changed."), true
			}
		}
	}
	return Approval{}, false
}

// keyName is a configuration key as a refusal names it.
func keyName(k string) string {
	if isUnknown(k) || strings.ContainsRune(k, varMark) {
		return "a key the line does not spell out"
	}
	return termsafe.Sanitize(k)
}

// gitConfigWrite reads one `git config` command's arguments, after the
// subcommand, and says why it is refused, if it is.
func gitConfigWrite(args []string) (string, bool) {
	var scope string
	var read, write, setting, edit, unknownFlag bool
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
		case t == "--unset" || t == "--unset-all" || t == "--rename-section" || t == "--remove-section":
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
		case "unset", "rename-section", "remove-section":
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
	case setting && (len(ops) == 0 || startsProgram(ops[0])):
		key := "a key"
		if len(ops) > 0 {
			key = keyName(ops[0])
		}
		return "this sets " + key + " in git's configuration, a setting that starts a program the guard does not read.", true
	}
	return "", false
}
