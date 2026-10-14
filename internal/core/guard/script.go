package guard

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"reflect"
	"strings"
	"sync"

	"github.com/intentdriven/abcd/internal/fsutil"
)

// The guard reads a script the command names before it judges the command
// (adr-2610091150447054). The stream refusal (interpreter-reads-stream) told
// the agent to save a script and run it as a file, and every file form was an
// allow that the shell then ran: `printf '<blocker>' > s.sh; bash s.sh`,
// `source s.sh`, `BASH_ENV=e bash -c true`, `bash --init-file e -i -c true`,
// `bash < s.sh`. The successor was the route around the guard.
//
// So a shell-family shell pointed at a file has the file read and judged with
// the registry's Tier 1 rules: its script operand, a `source`/`.` operand, the
// file a redirection makes its standard input, and the startup files the line
// selects (BASH_ENV, ENV for an interactive shell, --rcfile/--init-file, the
// zsh files under an assigned ZDOTDIR or HOME, and a login or interactive
// bash's under an assigned HOME). A path run directly is classified by its
// first bytes, and read only when it is a shell script. A file written earlier
// on the same line blocks, because the file read at check time is not the one
// that runs. Where the guard cannot be sure which bytes the shell will run it
// warns; it blocks only for a matched hazard or for text it knows runs unread.
//
// What stays unseen is named in commands/guard.md (decision 8): other
// interpreters' files, programs, the account's own startup files, file text
// substituted into a command string, the check-to-run race, a writer missing
// from the list below, and any spelling the decision does not list. An operand
// the line does not fix (a variable, a substitution, a glob) is left as it was,
// an allow, until the class (2) ruling of iss-2609281134544802 sets it.

const (
	// scriptHazardEntryID is the reserved id a registry entry matched inside a
	// script the line runs is carried out under; the entry itself is named in
	// the reason and in Matches.
	scriptHazardEntryID = "script-runs-hazard"

	// scriptWrittenEntryID is the reserved id for a script written earlier on
	// the line that runs it (block), or run after a write the guard cannot
	// place (warn).
	scriptWrittenEntryID = "script-written-then-run"

	// scriptUnreadEntryID is the reserved id for a file a shell runs that the
	// guard did not read: binary (block), named by a startup variable the
	// shell expands first (block), unreadable or over the cap (warn), or past
	// the check's read budget (warn).
	scriptUnreadEntryID = "script-unread"

	familyScript = "script file"

	// maxScriptBytes caps one script the guard reads, the cap the registry
	// load uses for .abcd/guard.json. A shell script over it warns, unread.
	maxScriptBytes = 256 << 10

	// sniffBytes is how much of a file is read to classify it: a NUL byte in
	// it is a binary, and its first line is the shebang.
	sniffBytes = 8 << 10

	// maxScriptFiles and maxScriptReadBytes are the read budget of one check,
	// across every script it reads, nested ones included. Past either the
	// guard stops reading and warns.
	maxScriptFiles     = 16
	maxScriptReadBytes = 1 << 20

	// maxScriptTargets bounds how many files one check considers at all —
	// resolved, compared with the line's writes, classified — whether or not
	// each is read, so a line naming thousands of scripts costs what sixty-four
	// do. Past it the guard warns through the budget's signal.
	maxScriptTargets = 4 * maxScriptFiles

	// maxTrackedFDs bounds the descriptors the reading follows; one opened
	// past it is not followed, and a shell's standard input duplicated from it
	// is not read.
	maxTrackedFDs = 64
)

// trackedVars are the only variables the reading follows: the ones that
// choose a file a shell reads or where a path resolves. Following no other
// keeps each state a handful of entries however many a line assigns.
var trackedVars = map[string]bool{"BASH_ENV": true, "ENV": true, "HOME": true, "ZDOTDIR": true, "PATH": true,
	"FPATH": true, "fpath": true, "CDPATH": true, "cdpath": true}

// trackedName is the name a tracked variable is kept under: zsh ties the
// fpath array to FPATH and cdpath to CDPATH, so an assignment to either of a
// pair sets both.
func trackedName(name string) string {
	switch name {
	case "fpath":
		return "FPATH"
	case "cdpath":
		return "CDPATH"
	}
	return name
}

// The shells a state's commands can run in: one the guard cannot place, zsh,
// or a shell that is not zsh (bash, sh, ksh and the rest of the family),
// which read a few words differently (iss-2610111730513071,
// iss-2610111730551327).
const (
	shellUnknown = ""
	shellZsh     = "zsh"
	shellOther   = "sh"
)

// shellOf is the shell a child started by the program called name runs.
func shellOf(name string) string {
	if name == "zsh" {
		return shellZsh
	}
	return shellOther
}

// trackedKey is the name an assignment to name is kept under in st's shell,
// and whether the reading follows it at all. zsh ties the path array to PATH
// as it ties fpath to FPATH; in every other shell path is an ordinary
// variable, so a state whose shell is unknown is split before it reads one
// (needsShell).
func (st *shellState) trackedKey(name string) (string, bool) {
	if name == "path" {
		return "PATH", st.shell == shellZsh
	}
	return trackedName(name), trackedVars[name]
}

// Files is the directory a command runs in, and the files the guard has read
// for it. A front door that checks one command against several registries
// hands each the same Files, so each file is read once per command.
type Files struct {
	dir  string
	home string
	path string
	// cdpath is CDPATH in the environment the check runs in, which a `cd`
	// searches unless the line sets its own.
	cdpath string

	mu    sync.Mutex
	cache map[string]fileRead
	stats map[string]statResult
	// statCalls counts the filesystem lookups made, so a test can prove the
	// site bound holds before any lookup (maxScriptTargets).
	statCalls int
}

// statResult is one cached lookup of a path.
type statResult struct {
	fi  os.FileInfo
	err error
}

// stat is os.Stat (or os.Lstat when link is set) of p, kept for the rest of
// the command's checks, so the hook's second registry pays no lookup again.
func (f *Files) stat(p string, link bool) (os.FileInfo, error) {
	key := p
	if link {
		key = "\x00l" + p
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	if r, ok := f.stats[key]; ok {
		return r.fi, r.err
	}
	f.statCalls++
	var r statResult
	if link {
		r.fi, r.err = os.Lstat(p)
	} else {
		r.fi, r.err = os.Stat(p)
	}
	if f.stats == nil {
		f.stats = map[string]statResult{}
	}
	f.stats[key] = r
	return r.fi, r.err
}

// NewFiles returns the reading context for a command run in dir: the host's
// working directory when it names one, else the session directory, and for
// `abcd guard check` the process's own. A relative or empty dir resolves no
// relative path.
func NewFiles(dir string) *Files {
	home, _ := os.UserHomeDir()
	if dir != "" && !filepath.IsAbs(dir) {
		dir = ""
	}
	return &Files{dir: filepath.Clean(dir), home: home, path: os.Getenv("PATH"), cdpath: os.Getenv("CDPATH"), cache: map[string]fileRead{}, stats: map[string]statResult{}}
}

// ReadingFrom returns r reading the files a command names from f.
func (r Registry) ReadingFrom(f *Files) Registry {
	r.files = f
	return r
}

// fileRead is one file as the guard read it: its size, its bytes when it is
// within maxScriptBytes (else the first sniffBytes), and the error that
// stopped the read.
type fileRead struct {
	size   int64
	data   []byte
	tooBig bool
	err    error
}

// read opens real (a path whose symlinks are already resolved) through
// fsutil's guarded open — regular files only, no symlinked leaf — and keeps
// what it read for the rest of the command's checks.
func (f *Files) read(real string) fileRead {
	f.mu.Lock()
	defer f.mu.Unlock()
	if fr, ok := f.cache[real]; ok {
		return fr
	}
	fr := readFile(real)
	if f.cache == nil {
		f.cache = map[string]fileRead{}
	}
	f.cache[real] = fr
	return fr
}

func readFile(real string) fileRead {
	fh, fi, err := fsutil.OpenRegular(real)
	if err != nil {
		return fileRead{err: err}
	}
	defer fh.Close()
	fr := fileRead{size: fi.Size()}
	limit := int64(maxScriptBytes)
	if fr.size > limit {
		fr.tooBig = true
		limit = sniffBytes
	}
	data, err := io.ReadAll(io.LimitReader(fh, limit+1))
	if err != nil {
		return fileRead{err: err}
	}
	if !fr.tooBig && int64(len(data)) > maxScriptBytes {
		// It grew between the stat and the read.
		fr.tooBig = true
		data = data[:sniffBytes]
	}
	fr.data = data
	return fr
}

// readCtx is one check's file reading: the files, the budget spent so far,
// the scripts being read now (a cycle is skipped), and the notes the check
// hands back.
type readCtx struct {
	files        *Files
	counted      map[string]bool
	nFiles       int
	nBytes       int
	nTargets     int
	nSites       int
	nDirProbes   int
	budgetWarned bool
	stack        []string
	notes        []string
	// dirChange caches, per file a `source` reads, whether it changes
	// directory, so a line sourcing one file many times reads it once.
	dirChange map[string]bool
	// readings counts the texts read, so a directory one reading names
	// (shellState.dirGen) is never one another names.
	readings int
}

func newReadCtx(f *Files) *readCtx {
	return &readCtx{files: f, counted: map[string]bool{}, dirChange: map[string]bool{}}
}

// take reads real against the check's budget: each distinct file counts
// once, and every byte handed back to be judged counts each time, so judging
// one cached script many times is bounded like reading many. Past the budget
// the read is refused.
func (rc *readCtx) take(real string) (fileRead, bool) {
	if rc.nBytes >= maxScriptReadBytes || (!rc.counted[real] && rc.nFiles >= maxScriptFiles) {
		return fileRead{}, false
	}
	if !rc.counted[real] {
		rc.counted[real] = true
		rc.nFiles++
	}
	fr := rc.files.read(real)
	rc.nBytes += len(fr.data)
	tally(len(fr.data))
	return fr, true
}

// budgetSignal is the budget's warn, raised once per check.
func (rc *readCtx) budgetSignal() []payloadSignal {
	if rc.budgetWarned {
		return nil
	}
	rc.budgetWarned = true
	return []payloadSignal{scriptBudgetSignal()}
}

func (rc *readCtx) note(format string, args ...any) {
	n := fmt.Sprintf(format, args...)
	if !containsString(rc.notes, n) {
		rc.notes = append(rc.notes, n)
	}
}

func (rc *readCtx) onStack(real string) bool { return containsString(rc.stack, real) }

// scriptRun is how a script the line runs is read: the depth its commands
// start at, and the shell state they start in.
type scriptRun struct {
	depth int
	state *shellState
}

// shellState is what the guard knows of the shell a command runs in: its
// directory (dirOK false where it cannot say), a `cd` the next commands run
// in only while each is chained after it with `&&`, the variables the line
// set and which of them it exported or took the export from, and the
// descriptors the line opened on files. A cd bash and zsh read differently can leave the shell in one of
// several directories; the reading then carries one state for each.
type shellState struct {
	dir      string
	dirOK    bool
	cd       *pendingCD
	vars     map[string]varVal
	exported map[string]bool
	fds      map[int]fdFile
	// stack is the top of the directory stack below the current directory,
	// as the line's own pushd commands fixed it, nearest first: the
	// directory a bare pushd swaps to is stack[0]. An entry is "" where the
	// line does not fix it, and nil is a stack the line did not fix, which
	// a bare pushd leaves unknown (iss-2610111815024967). It is never
	// changed in place.
	stack []string
	// shell is the shell the commands run in (shellZsh, shellOther), or
	// shellUnknown where the guard cannot place it.
	shell string
	// emptied is the value a `NAME=` word emptied, kept for the array
	// assignment it opens (`fpath=(dir $fpath)`), whose elements can name
	// the value from before it (withArray); nil where no such word ran.
	emptied *emptiedVar
	// dirGen names the directory the commands run in where the line leaves
	// it unknown: two states share it only where no change of directory
	// separates them, so a relative name in both is the same file. A change
	// marks it freshGen, and the reading stamps it with where the change
	// was (readScripts).
	dirGen [2]int
}

// freshGen marks a directory a change has just left unknown, until the
// reading names it; it names no directory, so it never equals another.
var freshGen = [2]int{-1, -1}

// emptiedVar is a tracked variable's value before a `NAME=` word emptied
// it; set is false where the line had not set it.
type emptiedVar struct {
	name string
	val  varVal
	set  bool
}

// pendingCD is a cd the next commands run after: dirs is every directory it
// can enter, one per reading of it that fixes one.
type pendingCD struct {
	dirs  []string
	chain int
}

// varVal is a variable's value as the line set it; ok is false where the
// line does not fix it. litTilde records that a `~` opening it was quoted
// or escaped in the assignment (`BASH_ENV='~/x'`), which leaves it text:
// only an unquoted one is the home (homeTilde).
type varVal struct {
	text     string
	ok       bool
	litTilde bool
}

// homeTilde reports whether v opens with a `~` the assignment expanded.
func (v varVal) homeTilde() bool {
	return strings.HasPrefix(v.text, "~") && !v.litTilde
}

// fdFile is a descriptor open on a file; ok is false where the line does not
// fix which. rel is, for a file the guard cannot place because it is named
// relative to a working directory the line leaves unknown, that name, and
// gen the directory it was opened in (shellState.dirGen).
type fdFile struct {
	path string
	ok   bool
	rel  string
	gen  [2]int
}

func (st *shellState) clone() *shellState {
	n := *st
	n.vars = make(map[string]varVal, len(st.vars))
	for k, v := range st.vars {
		n.vars[k] = v
	}
	n.exported = make(map[string]bool, len(st.exported))
	for k, v := range st.exported {
		n.exported[k] = v
	}
	n.fds = make(map[int]fdFile, len(st.fds))
	for k, v := range st.fds {
		n.fds[k] = v
	}
	return &n
}

// at is every state a segment can run in: a pending cd holds for a command
// chained after it with `&&`, one state for each directory it can enter, and
// once the chain breaks the directory is unknown for the rest of the text —
// never the one the shell would be in had the cd failed.
func (st *shellState) at(s segment) []*shellState {
	// The elements of an array assignment are words of the `NAME=(` command
	// before them, which the `&&` chain runs through, not a command that
	// breaks it.
	if st.cd == nil || s.arrayOf != "" {
		return []*shellState{st}
	}
	if s.chain == st.cd.chain && s.afterAnd {
		out := make([]*shellState, 0, len(st.cd.dirs))
		for _, d := range st.cd.dirs {
			n := st.clone()
			n.dir, n.dirOK = d, true
			out = append(out, n)
		}
		return out
	}
	n := st.clone()
	n.dir, n.dirOK, n.cd, n.stack = "", false, nil, nil
	n.dirGen = freshGen
	return []*shellState{n}
}

// unresolved returns st with its directory unknown, and its directory stack,
// which whatever left the directory unknown may have changed.
func (st *shellState) unresolved() *shellState {
	n := st.clone()
	n.dir, n.dirOK, n.cd, n.stack = "", false, nil, nil
	n.dirGen = freshGen
	return n
}

// sameUnknownDir reports whether st and o run in one directory the line
// leaves unknown, with no change of directory between them.
func (st *shellState) sameUnknownDir(o [2]int) bool {
	return !st.dirOK && st.dirGen != freshGen && st.dirGen == o
}

// cwdName is a relative word as the name of a file in the working
// directory, cleaned (`./s.sh` is s.sh), or "" where a `..` in it makes the
// file depend on where the directories lead.
func cwdName(word string) string {
	for _, c := range strings.Split(word, "/") {
		if c == ".." {
			return ""
		}
	}
	return path.Clean(word)
}

// relativeUnknown reports whether word names a file relative to a working
// directory the line leaves unknown. tilde is whether its leading `~` is
// tilde expansion, the home; a quoted `~` is a directory of that name, as
// relative as any other.
func (st *shellState) relativeUnknown(word string, tilde bool) bool {
	return word != "" && !filepath.IsAbs(word) && !tilde && (!st.dirOK || st.dir == "")
}

// home is the directory `~` expands to: HOME as the line set it, else the
// account's.
func (st *shellState) home(rc *readCtx) (string, bool) {
	if v, ok := st.vars["HOME"]; ok {
		return v.text, v.ok
	}
	return rc.files.home, rc.files.home != ""
}

// resolve makes word an absolute, clean path the way the shell opens it, or
// reports that the line does not fix one.
func (st *shellState) resolve(rc *readCtx, word string, tilde bool) (string, bool) {
	return st.resolveIn(rc, word, tilde, false)
}

// resolveIn is resolve, or with logical set the path `cd -L` computes: a `..`
// drops the name before it, as bash does to the operand before it changes
// directory.
func (st *shellState) resolveIn(rc *readCtx, word string, tilde, logical bool) (string, bool) {
	if word == "" {
		return "", false
	}
	if tilde {
		if word != "~" && !strings.HasPrefix(word, "~/") {
			return "", false // ~user
		}
		home, ok := st.home(rc)
		if !ok || home == "" {
			return "", false
		}
		word = fsutil.ExpandTilde(word, home)
	}
	if !filepath.IsAbs(word) {
		if !st.dirOK || st.dir == "" {
			return "", false
		}
		word = st.dir + string(filepath.Separator) + word
	}
	if logical {
		return filepath.Clean(word), true
	}
	return physicalPath(word)
}

// physicalPath cleans p the way the kernel opens it. A `..` is resolved
// against the directory the path has reached, symlinks followed, never by
// dropping the name before it: after `cd link` (logical or not), `../s.sh`
// is the file beside the link's target (iss-2610111600031244). A path with
// no `..` cleans lexically, which names the same file. ok is false where a
// directory before the last `..` does not exist, so no file is there.
func physicalPath(p string) (string, bool) {
	parts := strings.Split(p, string(filepath.Separator))
	last := -1
	for i, c := range parts {
		if c == ".." {
			last = i
		}
	}
	if last < 0 {
		return filepath.Clean(p), true
	}
	// filepath.EvalSymlinks walks the unclean path one name at a time and
	// takes a `..` from the directory it has resolved to, as the kernel does.
	real, err := filepath.EvalSymlinks(strings.Join(parts[:last+1], string(filepath.Separator)))
	if err != nil {
		return "", false
	}
	return filepath.Join(append([]string{real}, parts[last+1:]...)...), true
}

// resolveToken resolves token i of s, which the tokenizer has already
// unquoted. A word holding a substitution's output, a variable's value or a
// glob is not fixed by the line.
func (st *shellState) resolveToken(rc *readCtx, s segment, i int) (string, bool) {
	if i < 0 || i >= len(s.tokens) {
		return "", false
	}
	tok := s.tokens[i]
	if isUnknown(tok) || s.globAt(i) || strings.ContainsRune(tok, varMark) {
		return "", false
	}
	return st.resolve(rc, tok, s.homeTilde(i))
}

// fixedToken reports token i of s when the line fixes it.
func fixedToken(s segment, i int) (string, bool) {
	if i < 0 || i >= len(s.tokens) {
		return "", false
	}
	tok := s.tokens[i]
	if isUnknown(tok) || s.globAt(i) || strings.ContainsRune(tok, varMark) {
		return "", false
	}
	return tok, true
}

// assignment splits a NAME=VALUE word; ok is false for any other word.
func assignment(tok string) (name string, val varVal, ok bool) {
	name, eq, appends, subscript, ok := splitAssignment(tok)
	if !ok {
		return "", varVal{}, false
	}
	if appends || subscript {
		return name, varVal{}, true // a value the word alone does not fix
	}
	v := tok[eq+1:]
	return name, varVal{text: v, ok: !isUnknown(v) && !strings.ContainsRune(v, varMark)}, true
}

// assignOp splits an assignment word: NAME=VALUE, or NAME+=VALUE, which bash
// and zsh read as appending VALUE to NAME's value (iss-2610120953445472). eq
// is the index of the `=`; ok is false for any other word.
func assignOp(tok string) (name string, eq int, appends, ok bool) {
	name, eq, appends, _, ok = splitAssignment(tok)
	return name, eq, appends, ok
}

// assigned is the value the assignment word at i of s gives the tracked
// variable key, in st: its text where the line fixes it, a PATH value read
// entry by entry (pathValue), and a `+=` appended to the value before it.
func (st *shellState) assigned(rc *readCtx, s segment, i int, key string) varVal {
	tok := s.tokens[i]
	_, eq, appends, subscript, _ := splitAssignment(tok)
	if subscript {
		// An element of the variable (zsh's PATH[1,0]=dir:): a value the
		// reading does not follow, so a search reaching it stops.
		return varVal{}
	}
	v := tok[eq+1:]
	val := varVal{text: v, ok: !isUnknown(v) && !strings.ContainsRune(v, varMark)}
	val.litTilde = strings.HasPrefix(v, "~") && !s.tildeAt(i, eq+1)
	if (key == "PATH" || key == "FPATH") && val.ok {
		val.text = st.tildeList(rc, s, i, eq+1, v)
	}
	if key == "PATH" && !val.ok {
		val = st.pathValue(rc, s, i, eq)
	}
	if !appends {
		return val
	}
	prev := st.valueBefore(rc, key)
	if !prev.ok || !val.ok {
		return varVal{}
	}
	return varVal{text: prev.text + val.text, ok: true, litTilde: prev.litTilde || (prev.text == "" && val.litTilde)}
}

// valueBefore is key's value before an assignment that names it: the line's
// where it set one, else the account's, which the line does not fix but a
// search can still reach (accountPath, accountDirs).
func (st *shellState) valueBefore(rc *readCtx, key string) varVal {
	if v, set := st.vars[key]; set {
		return v
	}
	switch key {
	case "PATH":
		return varVal{text: accountPath, ok: true}
	case "FPATH":
		return varVal{text: accountDirs, ok: true}
	case "HOME":
		return varVal{text: rc.files.home, ok: rc.files.home != ""}
	case "CDPATH":
		return varVal{text: rc.files.cdpath, ok: true}
	}
	return varVal{}
}

// pathValue reads a PATH value holding variables entry by entry, as the
// search walks it (iss-2610120953445472): a fixed entry is kept, an entry
// that is $PATH or ${PATH} whole is the value before the assignment, one
// that $HOME or ${HOME} opens is HOME's value with the rest after it, and
// any other entry holding a value the line does not fix is unknownText,
// where a search stops. A word with a substitution's output, or spelled more
// than one way, is not fixed at all.
func (st *shellState) pathValue(rc *readCtx, s segment, i, eq int) varVal {
	tmpl, ok := s.variable[i]
	sp := s.spelled[i]
	if !ok || len(sp) != 1 || len(tmpl) <= eq || len(sp[0]) <= eq {
		return varVal{}
	}
	tents := strings.Split(tmpl[eq+1:], ":")
	sents := strings.Split(sp[0][eq+1:], ":")
	if len(tents) != len(sents) {
		return varVal{}
	}
	out := make([]string, 0, len(tents))
	at := eq + 1
	for k, te := range tents {
		off := at
		at += len(te) + 1
		if !strings.ContainsRune(te, varMark) {
			out = append(out, st.listEntry(rc, s, i, off, te))
			continue
		}
		se := sents[k]
		entry := unknownText
		switch {
		case se == "$PATH" || se == "${PATH}":
			if prev := st.valueBefore(rc, "PATH"); prev.ok {
				entry = prev.text
			}
		case strings.Count(te, varText) == 1 && strings.HasPrefix(te, varText):
			rest := te[len(varText):]
			if se == "$HOME"+rest || se == "${HOME}"+rest {
				if h, ok := st.home(rc); ok && h != "" {
					entry = h + rest
				}
			}
		}
		out = append(out, entry)
	}
	return varVal{text: strings.Join(out, ":"), ok: true}
}

// tildeList is a colon list's value v, which opens at offset off of the
// assignment word at token i of s, with each entry as the assignment leaves
// it (listEntry).
func (st *shellState) tildeList(rc *readCtx, s segment, i, off int, v string) string {
	if !strings.Contains(v, "~") {
		return v
	}
	parts := strings.Split(v, ":")
	for k, e := range parts {
		parts[k] = st.listEntry(rc, s, i, off, e)
		off += len(e) + 1
	}
	return strings.Join(parts, ":")
}

// listEntry is one entry e of a colon list (PATH, FPATH), at offset off of
// token i of s, as the shell assigns it: bash and zsh expand an unquoted `~`
// at the value's start and after each `:`, so `~` or `~/…` is the home with
// the rest after it, and `~user` a home the guard does not know
// (unknownText, where a search stops). A quoted `~` stays text, a directory
// of that name.
func (st *shellState) listEntry(rc *readCtx, s segment, i, off int, e string) string {
	if e == "" || e[0] != '~' || !s.tildeAt(i, off) {
		return e
	}
	if e != "~" && !strings.HasPrefix(e, "~/") {
		return unknownText
	}
	h, ok := st.home(rc)
	if !ok || h == "" || isUnknown(h) || strings.ContainsRune(h, varMark) {
		return unknownText
	}
	return h + e[1:]
}

// accountPath stands in a PATH the line set for the account's own PATH,
// which the line put back on the search without fixing it (`PATH=dir:$PATH`,
// `PATH+=:dir`, `path=(dir $path)`): the search walks the account's entries
// there, as it walks them where the line sets no PATH.
const accountPath = "\x02path"

// alwaysExported are the variables every shell inherits exported, so a plain
// assignment changes what a child sees whatever the line does.
var alwaysExported = map[string]bool{"HOME": true, "PATH": true}

// env is the environment the command at site in s runs with: the variables
// st set, and the assignments written before the command (a prefix, or an env
// wrapper's).
//
// A variable the environment already exports stays exported when the line
// assigns it without export, so a child is handed the line's value
// (iss-2610111730574385). Whether the shell that runs the line exports it is
// the environment's, which the guard does not see, so every tracked variable
// the line set is read as handed down unless the line itself took the export
// away (`export -n`, `typeset +x`); where it is not handed down, the child
// reads the account's own, which stays unseen.
func (st *shellState) env(rc *readCtx, s segment, site int) map[string]varVal {
	out := map[string]varVal{}
	for k, v := range st.vars {
		if e, set := st.exported[k]; !set || e || alwaysExported[k] {
			out[k] = v
		}
	}
	for k, v := range st.prefixVars(rc, s, site) {
		out[k] = v
	}
	return out
}

// prefixVars is the tracked variables the assignments written before the
// command at site in s set (a prefix, or an env wrapper's), each read in
// order, so a `+=` appends to the value a prefix before it gave; nil where
// there is none.
func (st *shellState) prefixVars(rc *readCtx, s segment, site int) map[string]varVal {
	var n *shellState
	var out map[string]varVal
	for i := 0; i < site && i < len(s.tokens); i++ {
		name, _, _, ok := assignOp(s.tokens[i])
		if !ok {
			continue
		}
		key, tracked := st.trackedKey(name)
		if !tracked {
			continue
		}
		if n == nil {
			n, out = st.clone(), map[string]varVal{}
		}
		v := n.assigned(rc, s, i, key)
		n.vars[key] = v
		out[key] = v
	}
	return out
}

// childState is the state a child shell starts in: the directory st runs
// in, and the environment it hands down, every variable of it exported.
// shell is the child's shell, shellUnknown where the guard cannot name it.
func childState(st *shellState, env map[string]varVal, shell string) *shellState {
	n := &shellState{dir: st.dir, dirOK: st.dirOK, dirGen: st.dirGen, vars: map[string]varVal{}, exported: map[string]bool{}, fds: map[int]fdFile{}, shell: shell}
	for k, v := range env {
		n.vars[k] = v
		n.exported[k] = true
	}
	for k, v := range st.fds {
		n.fds[k] = v
	}
	return n
}

// applyRedirects opens the descriptors redirections name, in order, on top
// of fds, which it changes.
func (st *shellState) applyRedirects(rc *readCtx, fds map[int]fdFile, rs []redirect) {
	set := func(fd int, f fdFile) {
		if fd >= 0 && fd < maxTrackedFDs {
			fds[fd] = f
		}
	}
	for _, r := range rs {
		fd := r.fd
		switch r.op {
		case "&>", "&>>":
			delete(fds, 1)
			delete(fds, 2)
			continue
		case "<&", ">&":
			if fd < 0 {
				fd = 0
				if r.op == ">&" {
					fd = 1
				}
			}
			switch t := r.target.text; {
			case !r.target.ok:
				set(fd, fdFile{})
			case t == "-":
				delete(fds, fd)
			case isAllDigits([]byte(t)):
				src := 0
				fmt.Sscan(t, &src)
				if f, ok := fds[src]; ok {
					set(fd, f)
				} else {
					delete(fds, fd)
				}
			default:
				// `>&file` is a file opened for writing on fd 1 and 2.
				set(fd, st.openAt(rc, t, r.target.tilde))
			}
			continue
		}
		if fd < 0 {
			fd = 0
			if strings.HasPrefix(r.op, ">") {
				fd = 1
			}
		}
		if !r.target.ok {
			set(fd, fdFile{})
			continue
		}
		set(fd, st.openAt(rc, r.target.text, r.target.tilde))
	}
}

// openAt is a descriptor opened on word, resolved as the shell opens it.
// Where the line leaves the working directory unknown, a relative word is
// not placed, and keeps its name and the directory's identity (fdFile).
func (st *shellState) openAt(rc *readCtx, word string, tilde bool) fdFile {
	p, ok := st.resolve(rc, word, tilde)
	f := fdFile{path: p, ok: ok}
	if !ok && st.relativeUnknown(word, tilde) {
		f.rel, f.gen = word, st.dirGen
	}
	return f
}

// stdinOf is the file the command s reads as standard input, when a
// redirection on the line puts one there; ok reports one does, and fixed
// whether the line fixes which.
func (st *shellState) stdinOf(rc *readCtx, s segment) (f fdFile, ok bool) {
	fds := make(map[int]fdFile, len(st.fds))
	for k, v := range st.fds {
		fds[k] = v
	}
	st.applyRedirects(rc, fds, s.redirects)
	f, ok = fds[0]
	return f, ok
}

// writeTarget is a path a command writes; ok is false where the line does
// not fix which.
type writeTarget struct {
	path string
	ok   bool
	// rel marks a write the guard cannot place because it names a path
	// relative to a working directory the line leaves unknown; name is that
	// path as a name in the directory (cwdName) and gen the directory
	// (shellState.dirGen). base is the last component of a path the guard
	// does not place, where the line fixes it, else "".
	rel  bool
	name string
	gen  [2]int
	base string
}

// scriptSeg is the reading's record of one segment: every state it can run
// in, its place on the line, and the files it writes.
type scriptSeg struct {
	states []*shellState
	order  []int
	writes []writeTarget
}

// maxShellStates bounds the states one shell's commands are read in. A line
// whose cds the shells read differently, one after another, multiplies them;
// past the bound the directory is unknown and the guard warns through the
// budget's signal.
const maxShellStates = 16

// distinctStates drops a state equal to an earlier one: two readings that
// land in the same directory are read once.
func distinctStates(sts []*shellState) []*shellState {
	out := sts[:0:0]
	for _, st := range sts {
		dup := false
		for _, o := range out {
			if reflect.DeepEqual(o, st) {
				dup = true
				break
			}
		}
		if !dup {
			out = append(out, st)
		}
	}
	return out
}

// carrierShell is the shell the string the command at site carries runs in:
// zsh's or another family member's by name, and one the guard cannot place
// for any other carrier or a name the line does not fix.
func carrierShell(s segment, site int) string {
	if site >= len(s.tokens) || isUnknown(s.tokens[site]) {
		return shellUnknown
	}
	switch tok := s.tokens[site]; {
	case nameCouldBe(tok, "zsh"):
		return shellZsh
	case nameCouldBeAny(tok, shellFamily):
		return shellOther
	}
	return shellUnknown
}

// placed is st, or where st's shell is unknown and s reads a word bash and
// zsh read differently (needsShell), st once as each shell: from s on, the
// line is read both ways, and each file either shell runs is judged.
func (st *shellState) placed(s segment) []*shellState {
	if st.shell != shellUnknown || !needsShell(s) {
		return []*shellState{st}
	}
	z, o := st.clone(), st.clone()
	z.shell, o.shell = shellZsh, shellOther
	return []*shellState{z, o}
}

// needsShell reports whether s reads a word bash and zsh read differently:
// an assignment to path, which zsh ties to PATH, or a `source` or `.` of a
// bare name, which zsh and bash search for in different places
// (sourceLookup).
func needsShell(s segment) bool {
	if s.arrayOf == "path" {
		return true
	}
	for _, t := range s.tokens {
		if name, _, ok := assignment(t); ok && name == "path" {
			return true
		}
	}
	for _, a := range commandSites(s) {
		tok := s.tokens[a.idx]
		if isUnknown(tok) || (tok != "source" && tok != ".") {
			continue
		}
		i := sourceIndex(s, a.idx)
		if w, ok := fixedToken(s, i); ok && !strings.Contains(w, "/") && !s.homeTilde(i) {
			return true
		}
	}
	return false
}

// earlier reports whether a runs before b on the line: a command before
// another, or the command that carries a string before the string's own.
func earlier(a, b []int) bool {
	for i := 0; i < len(a) && i < len(b); i++ {
		if a[i] != b[i] {
			return a[i] < b[i]
		}
	}
	return len(a) < len(b)
}

// readScripts reads every file the segments point a shell at and returns the
// verdicts on them. segs are the text's segments after the payload expansion.
func (r Registry) readScripts(segs []segment, rc *readCtx, run *scriptRun) []payloadSignal {
	n := len(segs)
	recs := make([]scriptSeg, n)
	rc.readings++
	reading := rc.readings
	// stamp names the directory a change at segment i left unknown.
	stamp := func(sts []*shellState, i, at int) {
		for _, st := range sts {
			if st.dirGen == freshGen {
				st.dirGen = [2]int{reading, 2*i + at}
			}
		}
	}

	// A string whose commands change directory changes it for an eval, which
	// runs the string in the shell it stands in.
	dirChangeBelow := make([]bool, n)
	for i := n - 1; i >= 0; i-- {
		c := segs[i].carrier
		if c > 0 && c-1 < n && (dirChangeBelow[i] || segChangesDir(segs[i])) {
			dirChangeBelow[c-1] = true
		}
	}

	var top *shellState
	if run != nil {
		top = run.state
	} else {
		top = &shellState{dir: rc.files.dir, dirOK: rc.files.dir != "" && filepath.IsAbs(rc.files.dir),
			vars: map[string]varVal{}, exported: map[string]bool{}, fds: map[int]fdFile{}}
	}

	// The states each segment runs in, walked in order within each shell:
	// the text's own commands from top, and a string's commands from the
	// states its carrier hands its child shell.
	var signals []payloadSignal
	scopes := map[int][]*shellState{}
	for i, s := range segs {
		sts, ok := scopes[s.carrier]
		if !ok {
			if s.carrier == 0 || s.carrier-1 >= i {
				sts = []*shellState{top}
			} else {
				c := s.carrier - 1
				site := lastSite(segs[c])
				shell := carrierShell(segs[c], site)
				sts = nil
				for _, cs := range recs[c].states {
					st := childState(cs, cs.env(rc, segs[c], site), shell)
					cs.applyRedirects(rc, st.fds, segs[c].redirects)
					if s.fromFixedOutput || isEvalCarrier(segs[c]) {
						// Another reading of the carrier, or a string eval runs
						// in the carrier's own shell.
						st = cs.clone()
						st.cd = nil
					}
					sts = append(sts, st)
				}
			}
		}
		var here []*shellState
		for _, st := range sts {
			for _, a := range st.at(s) {
				here = append(here, a.placed(s)...)
			}
		}
		stamp(here, i, 1)
		here = distinctStates(here)
		if len(here) > maxShellStates {
			signals = append(signals, rc.budgetSignal()...)
			here = []*shellState{here[0].unresolved()}
		}
		recs[i].states = here
		if s.carrier > 0 && s.carrier-1 < i {
			recs[i].order = append(append([]int(nil), recs[s.carrier-1].order...), i)
		} else {
			recs[i].order = []int{i}
		}
		next := make([]*shellState, len(here))
		for j, st := range here {
			next[j] = r.after(rc, st, s, dirChangeBelow[i])
		}
		stamp(next, i, 2)
		scopes[s.carrier] = next
	}

	// The targets each segment points a shell at, in every state it can run
	// in, and only then the writes: most lines run no script, and pay for
	// nothing more.
	type pending struct {
		seg int
		st  *shellState
		t   readTarget
	}
	var targets []pending
	for i, s := range segs {
		var seen []readTarget
		var raised []payloadSignal
		for _, st := range recs[i].states {
			ts, sigs := r.targetsOf(rc, s, st)
			for _, sig := range sigs {
				if !containsSignal(raised, sig) {
					raised = append(raised, sig)
				}
			}
			for _, t := range ts {
				if containsTarget(seen, t) {
					continue
				}
				seen = append(seen, t)
				targets = append(targets, pending{seg: i, st: st, t: t})
			}
		}
		signals = append(signals, raised...)
	}
	if len(targets) == 0 {
		return signals
	}
	for i, s := range segs {
		for _, st := range recs[i].states {
			recs[i].writes = append(recs[i].writes, writesOf(rc, s, st)...)
		}
	}
	for _, p := range targets {
		if rc.nTargets >= maxScriptTargets {
			signals = append(signals, rc.budgetSignal()...)
			break
		}
		rc.nTargets++
		// The writes of every earlier segment on the line (decision 6): the
		// commands before it, and the command that carries its string.
		var before []writeTarget
		for j := range segs {
			if j != p.seg && earlier(recs[j].order, recs[p.seg].order) {
				before = append(before, recs[j].writes...)
			}
		}
		signals = append(signals, r.readTarget(rc, segs[p.seg], p.st, p.t, before)...)
	}
	return signals
}

// containsSignal reports whether sigs holds sig, so a verdict two states of
// one segment raise alike is reported once.
func containsSignal(sigs []payloadSignal, sig payloadSignal) bool {
	for _, o := range sigs {
		if reflect.DeepEqual(o, sig) {
			return true
		}
	}
	return false
}

// containsTarget reports whether ts holds t, so a file two states of one
// segment name alike is read once.
func containsTarget(ts []readTarget, t readTarget) bool {
	for _, o := range ts {
		if reflect.DeepEqual(o, t) {
			return true
		}
	}
	return false
}

// lastSite is the last place the segment's command can sit.
func lastSite(s segment) int {
	site := len(s.tokens)
	for _, a := range commandSites(s) {
		site = a.idx
	}
	return site
}

func isEvalCarrier(s segment) bool {
	for _, a := range commandSites(s) {
		if nameCouldBe(s.tokens[a.idx], "eval") {
			return true
		}
	}
	return false
}

// segChangesDir reports whether a segment can change its shell's directory.
func segChangesDir(s segment) bool {
	for _, a := range commandSites(s) {
		if nameCouldBeAny(s.tokens[a.idx], directoryChanges) {
			return true
		}
	}
	return false
}

// after is the state the commands after s run in, in s's shell.
func (r Registry) after(rc *readCtx, st *shellState, s segment, stringChangesDir bool) *shellState {
	if key, ok := st.trackedKey(s.arrayOf); s.arrayOf != "" && ok {
		return st.withArray(rc, s, key)
	}
	allAssign := len(s.tokens) > 0
	for _, t := range s.tokens {
		if _, _, _, ok := assignOp(t); !ok {
			allAssign = false
			break
		}
	}
	if allAssign {
		n := st
		for i, t := range s.tokens {
			name, _, _, _ := assignOp(t)
			if key, ok := st.trackedKey(name); ok {
				if n == st {
					n = st.clone()
				}
				prev, set := n.vars[key]
				v := n.assigned(rc, s, i, key)
				n.emptied = nil
				if t == name+"=" {
					n.emptied = &emptiedVar{name: key, val: prev, set: set}
				}
				n.vars[key] = v
			}
		}
		return n
	}
	if len(s.tokens) == 1 && s.tokens[0] == "exec" && len(s.redirects) > 0 {
		n := st.clone()
		st.applyRedirects(rc, n.fds, s.redirects)
		return n
	}
	out := st
	for _, a := range commandSites(s) {
		tok := s.tokens[a.idx]
		args := s.tokens[a.idx+1:]
		switch name := strings.ToLower(path.Base(tok)); {
		case tok == "":
		case isUnknown(tok) && nameCouldBeAny(tok, directoryChanges):
			out = out.unresolved()
		case name == "cd" || name == "pushd":
			out = cdTarget(rc, out, s, a.idx)
		case name == "popd":
			out = out.unresolved()
		case name == "dirs" && len(args) > 0:
			// `dirs -c` clears the stack and `+N`/`-N` name entries of
			// it; any operand leaves it as the line does not fix it.
			n := out.clone()
			n.stack = nil
			out = n
		case name == "eval" && stringChangesDir:
			out = out.unresolved()
		case (name == "source" || name == ".") && sourcedChangesDir(rc, out, s, a.idx):
			out = out.unresolved()
		case name == "export" || name == "declare" || name == "typeset":
			// `declare -x`/`typeset -x` export as `export` does; `export -n`
			// and `declare +x` take the export away. A declare without -x sets
			// the value and leaves the export attribute as it was, as bash does.
			exports, unexports := name == "export", false
			for _, w := range args {
				switch {
				case w == "-n" && name == "export":
					exports, unexports = false, true
				case len(w) > 1 && w[0] == '-' && strings.ContainsRune(w[1:], 'x') && name != "export":
					exports = true
				case len(w) > 1 && w[0] == '+' && strings.ContainsRune(w[1:], 'x'):
					exports, unexports = false, true
				}
			}
			n := out.clone()
			for k, w := range args {
				nm, _, _, ok := assignOp(w)
				if !ok {
					nm = w
				}
				nm, tracked := out.trackedKey(nm)
				if !tracked {
					continue
				}
				if ok {
					n.vars[nm] = n.assigned(rc, s, a.idx+1+k, nm)
				}
				switch {
				case exports:
					n.exported[nm] = true
				case unexports:
					n.exported[nm] = false
				}
			}
			out = n
		case name == "unset":
			out = out.unset(args)
		}
	}
	return out
}

// unset is the state after `unset` with args. A variable it unsets is no
// longer the line's: the shell falls back to nothing, so an unset CDPATH is
// one that searches nothing, here and in a child the shell starts
// (iss-2610111730597708). zsh's fpath, cdpath and path are their tied
// variables only in zsh; elsewhere unsetting one leaves the tied variable as
// it was. `-f` unsets functions, never a variable, and a word the line does
// not fix may be `-f`, so it leaves every variable as it was.
func (st *shellState) unset(args []string) *shellState {
	for _, w := range args {
		if isUnknown(w) || strings.ContainsRune(w, varMark) || len(w) > 1 && w[0] == '-' && strings.ContainsRune(w[1:], 'f') {
			return st
		}
	}
	n := st.clone()
	for _, w := range args {
		key, tracked := st.trackedKey(w)
		if !tracked || (key != w && st.shell != shellZsh) {
			continue
		}
		if key == "CDPATH" {
			n.vars[key] = varVal{ok: true}
			n.exported[key] = true
			continue
		}
		delete(n.vars, key)
		delete(n.exported, key)
	}
	return n
}

// withArray is the state after a command whose words are elements of a
// tracked array assignment (`fpath=(a b)`): the `NAME=` word before it set
// the value empty, and each element is added in order, joined as the tied
// scalar joins them. An element the line does not fix, or one holding the
// separator, is kept as unknownText, so a search reaching it stops there; a
// `+=` onto a value the line did not set is the account's value, unknown,
// followed by the elements.
//
// An fpath element that is the array itself (`$fpath`, `"${fpath[@]}"`,
// `$FPATH`) is the value before the assignment: the line's where it set
// one, else the account's own directories, kept as accountDirs, and so is
// what a `+=` appends to where the line set no value
// (iss-2610111749099986). A path element that is the array itself
// (`$path`, `$PATH`) is read the same way, the account's PATH kept as
// accountPath (iss-2610120953445472).
func (st *shellState) withArray(rc *readCtx, s segment, name string) *shellState {
	// before is the value the array held before this assignment.
	before, beforeSet := st.vars[name]
	if e := st.emptied; !s.arrayAppend && e != nil && e.name == name {
		before, beforeSet = e.val, e.set
	}
	elems := make([]string, 0, len(s.tokens))
	for i := range s.tokens {
		w, ok := fixedToken(s, i)
		switch {
		case !ok && (name == "FPATH" || name == "PATH") && namesArray(s, i, name):
			switch {
			case !beforeSet && name == "PATH":
				w = accountPath
			case !beforeSet:
				w = accountDirs
			case !before.ok:
				w = unknownText
			default:
				w = before.text
			}
		case !ok || strings.ContainsRune(w, ':'):
			w = unknownText
		default:
			w = st.listEntry(rc, s, i, 0, w)
		}
		if w != "" {
			elems = append(elems, w)
		}
	}
	n := st.clone()
	n.emptied = nil
	prev, set := n.vars[name]
	switch {
	case !set && s.arrayAppend && name == "FPATH":
		prev = varVal{text: accountDirs, ok: true}
	case !set && s.arrayAppend && name == "PATH":
		prev = varVal{text: accountPath, ok: true}
	case !set && s.arrayAppend:
		prev = varVal{text: unknownText, ok: true}
	case !set:
		prev = varVal{ok: true}
	case !prev.ok:
		prev = varVal{text: unknownText, ok: true}
	}
	text := strings.Join(elems, ":")
	switch {
	case prev.text != "" && text != "":
		text = prev.text + ":" + text
	case prev.text != "":
		text = prev.text
	}
	n.vars[name] = varVal{text: text, ok: true}
	return n
}

// accountDirs stands in an FPATH the line set for the account's own
// directories, which the line put back on the search without fixing them
// (`fpath=(~/.zfunc $fpath)`). A function found only there is the
// account's own, unseen as its startup files are (autoloadTargets).
const accountDirs = "\x02account"

// namesArray reports whether word i of s is the tied array or scalar whole
// as the line spelled it (fpath or FPATH for FPATH, path or PATH for PATH),
// which as an element of the array is its value before the assignment.
func namesArray(s segment, i int, name string) bool {
	sp := s.spelled[i]
	if len(sp) == 0 {
		return false
	}
	for _, rest := range sp[1:] {
		if rest != "" {
			return false
		}
	}
	low := strings.ToLower(name)
	switch sp[0] {
	case "$" + low, "${" + low + "}", "$" + low + "[@]", "$" + low + "[*]", "$" + name, "${" + name + "}":
		return true
	}
	return false
}

// cdTarget is the state after a `cd`/`pushd` at site: every directory a
// reading of it enters (bash's and zsh's, cdOperand), where it exists as a
// directory now, holds for the commands chained after it with `&&`, and the
// scripts they name are read in each; a cd no reading fixes a directory for
// leaves the directory unknown.
//
// A pushd with no operand swaps the top two directories of the stack in
// bash and zsh, so it enters the directory the line's own pushd left, and
// no directory where the line did not fix the stack; it never enters HOME,
// which is `cd`'s (iss-2610111815024967). A pushd that enters a directory
// pushes the one it left. A cd leaves the stack as the line does not fix
// it, since zsh's AUTO_PUSHD makes it push too.
func cdTarget(rc *readCtx, st *shellState, s segment, site int) *shellState {
	n := st.unresolved()
	c := cdOperand(s, site)
	pushd := strings.ToLower(path.Base(s.tokens[site])) == "pushd"
	var dirs []string
	add := func(d string, ok bool) {
		if ok && d != "" && !containsString(dirs, d) {
			dirs = append(dirs, d)
		}
	}
	swap := false
	for _, e := range c.enters {
		if pushd && e.i >= len(s.tokens) {
			swap = true
			if len(st.stack) > 0 {
				add(st.stack[0], true)
			}
			continue
		}
		add(st.cdEnters(rc, s, site, e))
	}
	if c.stays {
		add(st.dir, st.dirOK)
	}
	if c.subst != nil {
		add(st.cdSubstitutes(rc, s, *c.subst))
	}
	if len(dirs) > 0 {
		n.cd = &pendingCD{dirs: dirs, chain: s.chain}
	}
	if pushd && len(dirs) > 0 && len(c.enters) == 1 && !c.stays && c.subst == nil {
		left := ""
		if st.dirOK {
			left = st.dir
		}
		rest := st.stack
		if swap {
			rest = rest[1:]
		}
		n.stack = append([]string{left}, rest...)
	}
	return n
}

// cdEnters is the directory a cd reading e enters by name: the one its
// CDPATH search can enter (targetsOf blocks one that is not one the guard can
// name), the operand, or HOME where there is none.
func (st *shellState) cdEnters(rc *readCtx, s segment, site int, e cdEnter) (string, bool) {
	if c := cdpathLookup(rc, st, s, site, e); c.searches {
		return c.dir, c.dir != ""
	}
	if e.i < len(s.tokens) {
		w, fixed := fixedToken(s, e.i)
		if !fixed || w == "-" || strings.HasPrefix(w, "+") {
			return "", false
		}
		return st.cdLands(rc, w, s.homeTilde(e.i), e.physical)
	}
	target, ok := st.home(rc)
	if !ok || target == "" {
		return "", false
	}
	if fi, err := os.Stat(target); err != nil || !fi.IsDir() {
		return "", false
	}
	if e.physical {
		real, err := filepath.EvalSymlinks(target)
		if err != nil {
			return "", false
		}
		target = real
	}
	return filepath.Clean(target), true
}

// cdSubstitutes is the directory zsh's substitution form enters: the current
// directory's path with the first occurrence of one word replaced by the
// other, where that names a directory.
func (st *shellState) cdSubstitutes(rc *readCtx, s segment, c cdSubst) (string, bool) {
	from, ok1 := fixedToken(s, c.from)
	to, ok2 := fixedToken(s, c.to)
	if !ok1 || !ok2 || from == "" || !st.dirOK || !strings.Contains(st.dir, from) {
		return "", false
	}
	return st.cdLands(rc, strings.Replace(st.dir, from, to, 1), false, c.physical)
}

// cdLands is the directory a `cd` to word enters, or ok false where it names
// no directory or the line does not fix one. bash's `cd -L` drops a `..` with
// the name before it, and falls back to the physical path when that names no
// directory; `-P` resolves it physically, and the directory it lands in is
// the physical one (iss-2610111600031244).
func (st *shellState) cdLands(rc *readCtx, word string, tilde, physical bool) (string, bool) {
	var target string
	var ok bool
	if !physical {
		target, ok = st.resolveIn(rc, word, tilde, true)
		if fi, err := os.Stat(target); ok && (err != nil || !fi.IsDir()) {
			physical = true
		}
	}
	if physical {
		target, ok = st.resolveIn(rc, word, tilde, false)
	}
	if !ok || target == "" {
		return "", false
	}
	if fi, err := os.Stat(target); err != nil || !fi.IsDir() {
		return "", false
	}
	if physical {
		real, err := filepath.EvalSymlinks(target)
		if err != nil {
			return "", false
		}
		target = real
	}
	return filepath.Clean(target), true
}

// cdEnter is a reading of a `cd`/`pushd` that enters the operand at i, or
// HOME where i is past the last word, physically or logically.
type cdEnter struct {
	i        int
	physical bool
}

// cdSubst is zsh's substitution form of a `cd`/`pushd`: the word at from in
// the current directory's path replaced by the word at to.
type cdSubst struct {
	from, to int
	physical bool
}

// cdReadings is what a `cd`/`pushd` does in bash and in zsh, which read some
// forms differently; the guard does not know which shell runs the line, so it
// takes every reading. enters are the readings that enter a directory by
// name, stays is bash's pushd -n, which stays put, and subst is zsh's
// substitution form. None of them is set where no reading is one the reader
// can name.
type cdReadings struct {
	enters []cdEnter
	stays  bool
	subst  *cdSubst
}

// cdOperand reads the options and operands of the `cd`/`pushd` at site the
// way each shell does. bash takes -L, -P (the last of the two deciding), -e
// and -@, pushd -n too, refuses any other option, and enters its operand, or
// stays put for pushd -n; with two or more operands an older bash (3.2)
// enters the first and a newer one refuses the cd, so the first is taken as
// entered whatever follows it. zsh takes -L, -P, -q and -s;
// at the first word it does not take as an option its operands start, and
// two operands are the substitution form, one is the directory entered, and
// more are refused. A word starting with `-` with a letter neither shell
// takes is refused by bash and is zsh's first operand where no earlier word
// was, so with one more word after zsh's first operand it is the
// substitution form; alone it is the directory zsh enters by that name,
// unless it is a directory-stack word (`-N`), which names none; and
// otherwise no reading is taken.
func cdOperand(s segment, site int) cdReadings {
	bashLetters := "LPe@"
	if strings.ToLower(path.Base(s.tokens[site])) == "pushd" {
		bashLetters += "n"
	}
	bashOK, stays, bashPhysical, zshPhysical := true, false, false, false
	zshFirst := -1
	i := site + 1
	for ; i < len(s.tokens); i++ {
		t := s.tokens[i]
		if t == "--" {
			i++
			break
		}
		if len(t) < 2 || t[0] != '-' {
			break
		}
		if strings.Trim(t[1:], "LPe@qsn") != "" {
			if zshFirst < 0 {
				zshFirst = i
			}
			switch len(s.tokens) - zshFirst {
			case 1:
				if isAllDigits([]byte(t[1:])) {
					return cdReadings{}
				}
				return cdReadings{enters: []cdEnter{{i: zshFirst, physical: zshPhysical}}}
			case 2:
				return cdReadings{subst: &cdSubst{from: zshFirst, to: zshFirst + 1, physical: zshPhysical}}
			}
			return cdReadings{}
		}
		if strings.Trim(t[1:], bashLetters) != "" {
			bashOK = false
		}
		zshOption := zshFirst < 0 && strings.Trim(t[1:], "LPqs") == ""
		if zshFirst < 0 && !zshOption {
			zshFirst = i
		}
		for _, c := range t[1:] {
			switch c {
			case 'L', 'P':
				bashPhysical = c == 'P'
				if zshOption {
					zshPhysical = c == 'P'
				}
			case 'n':
				stays = true
			}
		}
	}
	var r cdReadings
	if bashOK {
		if stays {
			r.stays = true
		} else {
			r.enters = append(r.enters, cdEnter{i: i, physical: bashPhysical})
		}
	}
	if zshFirst < 0 {
		zshFirst = i
	}
	switch len(s.tokens) - zshFirst {
	case 0, 1:
		if e := (cdEnter{i: zshFirst, physical: zshPhysical}); len(r.enters) == 0 || r.enters[0] != e {
			r.enters = append(r.enters, e)
		}
	case 2:
		r.subst = &cdSubst{from: zshFirst, to: zshFirst + 1, physical: zshPhysical}
	}
	return r
}

// cdpathSearch is what the CDPATH search of a `cd`/`pushd` comes to.
type cdpathSearch struct {
	// searches is set where the shell searches CDPATH for the operand.
	searches bool
	// fromEnv is set where that CDPATH is the one the check's environment
	// exports, not one the command sets.
	fromEnv bool
	// dir is the one directory the search can enter, or "" where the guard
	// cannot name it.
	dir string
}

// cdpathLookup is the CDPATH search of the `cd`/`pushd` at site. The shell
// searches CDPATH when the operand is not absolute and does not start with
// `./` or `../` (nor is `.` or `..`), and CDPATH — as the line set it, a
// prefix on the cd included, else as the environment the check runs in holds
// it — is not empty. zsh's cdpath is the same variable. The shell then enters
// the first candidate holding the operand, bash and zsh in different orders:
// bash tries the CDPATH entries (an empty or `.` entry is the current
// directory) and then the current directory, zsh the current directory first
// unless `.` is an entry. Where exactly one candidate exists both enter it,
// and that is dir; none, two, more entries than maxScriptTargets, or an entry
// or a current directory the line does not fix, leave dir empty, and the cd
// blocks (iss-2610111600031244, iss-2610111749083184). e is the reading of
// the cd that enters its operand by name (cdOperand).
func cdpathLookup(rc *readCtx, st *shellState, s segment, site int, e cdEnter) cdpathSearch {
	i, physical := e.i, e.physical
	w, fixed := fixedToken(s, i)
	if !fixed {
		// No operand goes HOME; one the line does not fix leaves the
		// directory unknown whatever CDPATH holds.
		return cdpathSearch{}
	}
	if w == "-" || strings.HasPrefix(w, "+") || strings.HasPrefix(w, "/") || s.homeTilde(i) ||
		w == "." || w == ".." || strings.HasPrefix(w, "./") || strings.HasPrefix(w, "../") {
		return cdpathSearch{}
	}
	v, set := st.vars["CDPATH"]
	for j := 0; j < site && j < len(s.tokens); j++ {
		if name, pv, ok := assignment(s.tokens[j]); ok && trackedName(name) == "CDPATH" {
			v, set = pv, true
		}
	}
	c := cdpathSearch{searches: true}
	if !set {
		v, c.fromEnv = varVal{text: rc.files.cdpath, ok: true}, true
	}
	if v.ok && v.text == "" {
		return cdpathSearch{}
	}
	if !v.ok {
		return c
	}
	entries := strings.Split(v.text, ":")
	if len(entries) > maxScriptTargets {
		return c
	}
	words := make([]string, 0, len(entries)+1)
	cwdEntry := false
	for _, e := range entries {
		switch {
		case isUnknown(e) || strings.ContainsRune(e, varMark) || strings.HasPrefix(e, "~"):
			return c
		case e == "" || e == ".":
			cwdEntry = true
			words = append(words, w)
		default:
			words = append(words, strings.TrimSuffix(e, "/")+"/"+w)
		}
	}
	if !cwdEntry {
		words = append(words, w)
	}
	found := ""
	for _, cw := range words {
		if !filepath.IsAbs(cw) && (!st.dirOK || st.dir == "") {
			return c
		}
		d, ok := st.cdLands(rc, cw, false, physical)
		if !ok || d == found {
			continue
		}
		if found != "" {
			return c
		}
		found = d
	}
	c.dir = found
	return c
}

// sourcedChangesDir reports whether the file a `source` at site reads holds a
// directory change, which leaves the shell's directory unknown after it.
func sourcedChangesDir(rc *readCtx, st *shellState, s segment, site int) bool {
	// The same bound as the targets, before any lookup: past it the directory
	// after the source is unknown, the safe reading.
	if rc.nDirProbes >= maxScriptTargets {
		return true
	}
	rc.nDirProbes++
	p, ok := sourcePath(rc, st, s, site)
	if !ok {
		return false
	}
	real, err := filepath.EvalSymlinks(p)
	if err != nil {
		return false
	}
	if moves, ok := rc.dirChange[real]; ok {
		return moves
	}
	moves := true // unread past the budget, or too big to read: it may
	if fr, ok := rc.take(real); ok && fr.err != nil {
		moves = false // a file the shell cannot read runs nothing
	} else if ok && !fr.tooBig {
		moves = false
		segs, err := tokenize(string(fr.data))
		if err != nil {
			moves = true
		}
		for _, x := range segs {
			if segChangesDir(x) {
				moves = true
				break
			}
		}
	}
	rc.dirChange[real] = moves
	return moves
}

// The kinds of file a command points a shell at.
const (
	// targetScript is a shell's script: its operand, a `source` operand, or
	// the file its standard input is. One that does not exist is noted.
	targetScript = iota
	// targetStartup is a startup file the line selects; the shell skips one
	// that does not exist, and so does the guard, silently.
	targetStartup
	// targetDirect is a path run directly, classified before it is read.
	targetDirect
)

// readTarget is one file a command points a shell at.
type readTarget struct {
	path  string
	shown string
	kind  int
	// sourced records a `source`/`.`, which runs the file in the shell the
	// command stands in; env is what a child shell is handed otherwise.
	sourced bool
	env     map[string]varVal
	// shell is the shell a child runs the file in, shellUnknown where the
	// guard cannot name it (a file run directly).
	shell string
	// startup are the startup files a direct-run shell script reads
	// (BASH_ENV), read once it is classified as one.
	startupEnv bool
	// shadows are the paths a source's search looked at before path, where a
	// file the line writes first is the one the shell runs; missing is the
	// note for a source whose search found no file, path "", which only a
	// write on the search can supply (iss-2610120953449511).
	shadows []string
	missing string
	// blind records a file named relative to a working directory the line
	// leaves unknown, or a search that looked in one: the file that runs is
	// not one the guard can name, so a write before it may be that file.
	// cwdName is, where the file in that directory is the one that runs when
	// it is there, its name in the directory (cwdName), and "" otherwise.
	// base is the last component of the name the run looks up, where
	// shown is not that name; "" means filepath.Base(shown).
	blind   bool
	cwdName string
	base    string
}

// targetsOf is every file a segment's commands point a shell at, and the
// stream verdicts on a startup option or variable handed a stream.
func (r Registry) targetsOf(rc *readCtx, s segment, st *shellState) ([]readTarget, []payloadSignal) {
	if s.arrayOf != "" {
		// The words of an array assignment are its elements: none runs.
		return nil, nil
	}
	var out []readTarget
	var sigs []payloadSignal
	for _, a := range commandSites(s) {
		tok := s.tokens[a.idx]
		// An empty word (a value placed empty) names no command, and never
		// `.`, which is what path.Base makes of it.
		if tok == "" || isUnknown(tok) || s.globAt(a.idx) || strings.ContainsRune(tok, varMark) {
			continue
		}
		name := strings.ToLower(path.Base(tok))
		if name == "cd" || name == "pushd" {
			blocked := false
			for _, e := range cdOperand(s, a.idx).enters {
				if c := cdpathLookup(rc, st, s, a.idx, e); c.searches && c.dir == "" {
					sigs = append(sigs, cdpathSignal(c.fromEnv))
					blocked = true
					break
				}
			}
			if blocked {
				continue
			}
		}
		first, opaque, autoload := autoloadCall(s, a.idx, name)
		if !isShellFamily(name) && name != "source" && name != "." && !strings.Contains(tok, "/") && !autoload {
			continue
		}
		// The bound holds before any lookup: a line naming thousands of
		// scripts resolves sixty-four of them and warns through the budget.
		if rc.nSites >= maxScriptTargets {
			sigs = append(sigs, rc.budgetSignal()...)
			break
		}
		rc.nSites++
		env := st.env(rc, s, a.idx)
		switch {
		case isShellFamily(name):
			ts, ss := shellTargets(rc, s, st, a.idx, name, env)
			out = append(out, ts...)
			sigs = append(sigs, ss...)
		case name == "source" || name == ".":
			p, ok, unfixed, blind, cwdRuns, tried := sourceLookup(rc, st, s, a.idx)
			if unfixed {
				sigs = append(sigs, sourcePathUnfixedSignal())
				continue
			}
			w := sourceWord(s, a.idx)
			cwd := ""
			if cwdRuns {
				cwd = cwdName(w)
			}
			if ok {
				out = append(out, readTarget{path: p, shown: w, kind: targetScript, sourced: true, env: env, shadows: tried, blind: blind, cwdName: cwd})
			} else if w != "" {
				if _, fixed := fixedToken(s, sourceIndex(s, a.idx)); fixed && !scriptIsStream(w, s.stdinStream) {
					if !strings.Contains(w, "/") {
						// No file is there now, but one the line writes on the
						// search before the source runs is (readTarget).
						out = append(out, readTarget{shown: w, kind: targetScript, sourced: true, env: env, shadows: tried, blind: blind, cwdName: cwd,
							missing: fmt.Sprintf("abcd guard: `%s %s` names no file the shell can find, so there was nothing to read; if the shell reaches it, it refuses it.", tok, w)})
					} else if blind {
						out = append(out, readTarget{shown: w, kind: targetScript, sourced: true, env: env, blind: true, cwdName: cwd})
					}
				}
			}
		case autoload:
			ts, ss := autoloadTargets(rc, s, st, a.idx, first, opaque, env)
			out = append(out, ts...)
			sigs = append(sigs, ss...)
		case strings.Contains(tok, "/"):
			if p, ok := st.resolveToken(rc, s, a.idx); ok {
				out = append(out, readTarget{path: p, shown: tok, kind: targetDirect, env: env, startupEnv: true})
			} else if st.relativeUnknown(tok, s.homeTilde(a.idx)) {
				out = append(out, readTarget{shown: tok, kind: targetDirect, env: env, blind: true, cwdName: cwdName(tok)})
			}
		}
	}
	return out, sigs
}

// autoloadCall reports whether the command at site marks functions to be
// loaded from their files, as zsh's and ksh's `autoload` does, and so do
// `functions -u` and `typeset -fu` (declare is typeset's other name): the
// index of its first operand, and whether its operands are not plain names
// (-m patterns, -w compiled files).
func autoloadCall(s segment, site int, name string) (first int, opaque, ok bool) {
	var f, u bool
	switch name {
	case "autoload":
		f, u = true, true
	case "functions":
		f = true
	case "typeset", "declare":
	default:
		return 0, false, false
	}
	i := site + 1
	for ; i < len(s.tokens); i++ {
		t := s.tokens[i]
		if t == "--" {
			i++
			break
		}
		if len(t) < 2 || (t[0] != '-' && t[0] != '+') {
			break
		}
		if isUnknown(t) || strings.ContainsRune(t, varMark) {
			// An option the line does not fix may be any of them.
			opaque = true
			continue
		}
		for _, c := range t[1:] {
			switch c {
			case 'f':
				f = true
			case 'u':
				u = true
			case 'm', 'w':
				opaque = true
			}
		}
	}
	return i, opaque, f && u
}

// autoloadTargets is the function file each operand of an autoload at site
// loads. An operand with a slash is that file. A name is searched in the
// FPATH the line set — the first directory holding it, as a compiled
// `dir.zwc` or `dir/name.zwc` or as `dir/name` — and where the line leaves
// FPATH as the account set it, the file is the account's own and stays
// unseen, as its startup files do. Where the line set FPATH but does not fix
// a name, or a directory searched before the file is found, what loads is
// not a file the guard can read, and that blocks (iss-2610111600022820).
func autoloadTargets(rc *readCtx, s segment, st *shellState, site, first int, opaque bool, env map[string]varVal) ([]readTarget, []payloadSignal) {
	fp, lineSet := st.vars["FPATH"]
	for i := 0; i < site && i < len(s.tokens); i++ {
		if name, v, ok := assignment(s.tokens[i]); ok && trackedName(name) == "FPATH" {
			if v.ok {
				v.text = st.tildeList(rc, s, i, len(name)+1, v.text)
			}
			fp, lineSet = v, true
		}
	}
	unfixed := []payloadSignal{autoloadUnfixedSignal()}
	var out []readTarget
	for i := first; i < len(s.tokens); i++ {
		w, fixed := fixedToken(s, i)
		switch {
		case fixed && strings.Contains(w, "/"):
			if p, ok := st.resolveToken(rc, s, i); ok {
				out = append(out, readTarget{path: p, shown: "the function file autoload loads (" + w + ")", kind: targetScript, sourced: true, env: env})
			} else if lineSet {
				return nil, unfixed
			}
			continue
		case !lineSet:
			continue // the account's own FPATH
		case !fixed || opaque || !fp.ok:
			return nil, unfixed
		}
		var dirs []string
		if fp.text != "" {
			dirs = strings.Split(fp.text, ":")
		}
		if len(dirs) > maxScriptTargets {
			return nil, unfixed
		}
		for _, d := range dirs {
			if d == accountDirs {
				continue // the account's own, searched by the shell, unseen
			}
			if d == "" || isUnknown(d) || strings.ContainsRune(d, varMark) {
				return nil, unfixed
			}
			// The assignment expanded every `~` it could (listEntry): one
			// left is text.
			dir, ok := st.resolve(rc, d, false)
			if !ok {
				return nil, unfixed
			}
			p, found := "", false
			for _, c := range []string{dir + ".zwc", filepath.Join(dir, w+".zwc"), filepath.Join(dir, w)} {
				if fi, err := rc.files.stat(c, false); err == nil && fi.Mode().IsRegular() {
					p, found = c, true
					break
				}
			}
			if found {
				out = append(out, readTarget{path: p, shown: "the function file autoload " + w + " loads (" + p + ")", kind: targetScript, sourced: true, env: env})
				break
			}
		}
	}
	return out, nil
}

// sourceIndex is the token index of a `source` at site's file operand, or -1.
func sourceIndex(s segment, site int) int {
	i := site + 1
	if i < len(s.tokens) && s.tokens[i] == "--" {
		i++
	}
	if i >= len(s.tokens) {
		return -1
	}
	return i
}

func sourceWord(s segment, site int) string {
	if i := sourceIndex(s, site); i >= 0 {
		return s.tokens[i]
	}
	return ""
}

// sourcePath resolves a `source`/`.` operand as the state's shell does: a
// word with a slash as a path, any other searched on PATH and then in the
// working directory, or in zsh, for `source`, in the working directory and
// then on PATH, and for `.` on PATH alone. ok is false where the line does not fix the word, where it is a
// stream (the stream refusal reads it), or where no file is found by search.
func sourcePath(rc *readCtx, st *shellState, s segment, site int) (string, bool) {
	p, ok, _, _, _, _ := sourceLookup(rc, st, s, site)
	return p, ok
}

// sourceLookup is sourcePath, and whether the search ran into a PATH entry
// the line does not fix. A `PATH=` prefix on the `source` itself is the PATH
// its search reads (iss-2610111600031244), and there an entry the line does
// not fix — a variable, or a relative entry where the working directory is
// unknown — means the file the shell finds is not one the guard can name.
// tried is every path the search looked at before the file it found, or
// every one it looked at where it found none: a file the line writes at one
// of them is the file that runs (iss-2610120953449511). blind is whether the
// file is named relative to a working directory the line leaves unknown, or
// the search looked in one before it found the file or found none: there
// the file that runs is not one the guard can name, so a write before it
// may be that file (readTarget). cwdRuns is whether, there, the file in the
// working directory is the one that runs when it is there: a relative path,
// zsh's `source`, which looks there first, or a search that reached the
// directory past no PATH entry it skipped.
func sourceLookup(rc *readCtx, st *shellState, s segment, site int) (p string, ok, unfixed, blind, cwdRuns bool, tried []string) {
	i := sourceIndex(s, site)
	w, ok := fixedToken(s, i)
	if !ok || scriptIsStream(w, s.stdinStream) {
		return "", false, false, false, false, nil
	}
	if strings.Contains(w, "/") || s.homeTilde(i) {
		p, ok := st.resolveToken(rc, s, i)
		blind := !ok && st.relativeUnknown(w, s.homeTilde(i))
		return p, ok, false, blind, blind, nil
	}
	pv, set := st.vars["PATH"]
	if v, ok := st.prefixVars(rc, s, site)["PATH"]; ok {
		pv, set = v, true
	}
	// A PATH the line set, before the command or as its prefix, is searched
	// strictly: an entry the line does not fix stops the search
	// (iss-2610120953445472).
	var skipped searchSkips
	if st.shell != shellZsh {
		p, ok, unfixed := searchPath(rc, st, pv, set, w, true, set, &tried, &skipped)
		return p, ok, unfixed, skipped.entry || skipped.cwd, skipped.cwd && !skipped.entry, tried
	}
	if s.tokens[site] == "source" {
		// zsh looks in the working directory first. Where the line leaves
		// it unknown, the file there is unread, as a `./name` source there
		// is, and PATH's, which runs when the directory does not hold the
		// name, is read.
		if p, ok := st.resolve(rc, w, false); ok {
			if fi, err := rc.files.stat(p, false); err == nil && fi.Mode().IsRegular() {
				return p, true, false, false, false, nil
			}
			tried = append(tried, p)
		} else {
			blind, cwdRuns = true, true
		}
	}
	p, ok, unfixed = searchPath(rc, st, pv, set, w, false, set, &tried, &skipped)
	return p, ok, unfixed, blind || skipped.entry || skipped.cwd, cwdRuns, tried
}

// searchSkips is what a search skipped because the line leaves the working
// directory unknown: a relative PATH entry (entry), or the working directory
// itself (cwd).
type searchSkips struct{ entry, cwd bool }

// searchPath finds a bare name on PATH — pv when set, as the line set it,
// else the account's — and then, for `source`, in the working directory. An
// empty entry is the working directory, and a relative entry resolves
// against it, as the shell's search does. An entry that is the account's
// PATH (accountPath) is searched as the account's PATH always is. With
// strict, an entry the line does not fix — a value it does not fix, or a
// relative entry where the working directory is unknown — stops the search
// and is reported (unfixed); without, the search skips it, as it always has.
// skipped, when not nil, records where the search skipped a directory
// relative to a working directory the line leaves unknown.
func searchPath(rc *readCtx, st *shellState, pv varVal, set bool, name string, thenCwd, strict bool, tried *[]string, skipped *searchSkips) (string, bool, bool) {
	look := func(p string) {
		if tried != nil {
			*tried = append(*tried, p)
		}
	}
	skip := func(cwd bool) {
		if skipped != nil {
			if cwd {
				skipped.cwd = true
			} else {
				skipped.entry = true
			}
		}
	}
	type entry struct {
		dir     string
		account bool
	}
	var entries []entry
	account := func() {
		for _, d := range filepath.SplitList(rc.files.path) {
			entries = append(entries, entry{d, true})
		}
	}
	switch {
	case !set:
		account()
	case !pv.ok:
		return "", false, strict
	default:
		for _, d := range filepath.SplitList(pv.text) {
			if d == accountPath {
				account()
				continue
			}
			entries = append(entries, entry{d, false})
		}
	}
	for _, e := range entries {
		d := e.dir
		lineStrict := strict && !e.account
		if !e.account && (isUnknown(d) || strings.ContainsRune(d, varMark) || strings.Contains(d, accountPath)) {
			if lineStrict {
				return "", false, true
			}
			continue
		}
		if d == "" {
			d = "."
		}
		p := filepath.Join(d, name)
		if !filepath.IsAbs(d) {
			q, ok := st.resolve(rc, p, false)
			if !ok {
				if lineStrict {
					return "", false, true
				}
				if st.relativeUnknown(p, strings.HasPrefix(p, "~")) {
					skip(false)
				}
				continue
			}
			p = q
		}
		if fi, err := rc.files.stat(p, false); err == nil && fi.Mode().IsRegular() {
			return p, true, false
		}
		look(p)
	}
	if thenCwd {
		if p, ok := st.resolve(rc, name, false); ok {
			if _, err := rc.files.stat(p, false); err == nil {
				return p, true, false
			}
			look(p)
		} else if st.relativeUnknown(name, strings.HasPrefix(name, "~")) {
			skip(true)
		}
	}
	return "", false, false
}

// shellCall is how a shell-family shell reads its arguments.
type shellCall struct {
	cmdString, stdin, versionOnly, unresolved  bool
	interactive, login, norc, noprofile, norcs bool
	// script and rcfile index the arguments: the script operand and the
	// --rcfile/--init-file value, -1 for none.
	script, rcfile int
}

// parseShellCall walks a shell's arguments the way its own parser reads them.
// An argument the line does not fix leaves the rest unread (unresolved).
func parseShellCall(name string, args []string) shellCall {
	c := shellCall{script: -1, rcfile: -1}
	zsh := name == "zsh"
	i := 0
	for i < len(args) {
		a := args[i]
		if isUnknown(a) || strings.ContainsRune(a, varMark) {
			c.unresolved = true
			return c
		}
		switch {
		case a == "--":
			if !c.cmdString && !c.stdin && i+1 < len(args) {
				c.script = i + 1
			} else if i+1 >= len(args) && !c.cmdString {
				c.stdin = true
			}
			return c
		case a == "-":
			if !c.cmdString {
				c.stdin = true
			}
			return c
		case a == "--version" || a == "--help":
			c.versionOnly = true
			return c
		case a == "--rcfile" || a == "--init-file":
			if i+1 < len(args) {
				c.rcfile = i + 1
			}
			i += 2
			continue
		case a == "--login":
			c.login = true
		case a == "--interactive":
			c.interactive = true
		case a == "--norc":
			c.norc = true
		case a == "--noprofile":
			c.noprofile = true
		case a == "--no-rcs" || a == "--norcs":
			c.norcs = true
		case a == "--emulate":
			i += 2
			continue
		case strings.HasPrefix(a, "--"):
		case len(a) >= 2 && (a[0] == '-' || a[0] == '+'):
			set := a[0] == '-'
			var values []bool // for each o/O in the cluster: its sign
			for k := 1; k < len(a); k++ {
				switch a[k] {
				case 'c':
					c.cmdString = c.cmdString || set
				case 's':
					c.stdin = c.stdin || set
				case 'i':
					c.interactive = c.interactive || set
				case 'l':
					c.login = c.login || set
				case 'f':
					if zsh && set {
						c.norcs = true
					}
				case 'o', 'O':
					values = append(values, set)
				}
			}
			for k, on := range values {
				if i+1+k >= len(args) {
					break
				}
				switch v := strings.ToLower(strings.ReplaceAll(args[i+1+k], "_", "")); {
				case v == "login" && on:
					c.login = true
				case v == "interactive" && on:
					c.interactive = true
				case (v == "norcs" && on) || (v == "rcs" && !on):
					c.norcs = true
				}
			}
			i += 1 + len(values)
			continue
		default:
			if !c.cmdString && !c.stdin {
				c.script = i
			}
			return c
		}
		i++
	}
	if !c.cmdString {
		c.stdin = true
	}
	return c
}

// shellTargets is every file the shell at site in s reads before or as its
// commands, and the stream verdict on a startup file handed a stream.
func shellTargets(rc *readCtx, s segment, st *shellState, site int, name string, env map[string]varVal) ([]readTarget, []payloadSignal) {
	args := s.tokens[site+1:]
	c := parseShellCall(name, args)
	if c.versionOnly {
		return nil, nil
	}
	var out []readTarget
	var sigs []payloadSignal
	child := func(p, shown string, kind int) readTarget {
		return readTarget{path: p, shown: shown, kind: kind, env: env, shell: shellOf(name)}
	}
	// stdinBlind is the file on the shell's standard input where the line
	// leaves the directory it was opened in unknown: unread, and a write
	// before the run may be it (readTarget). Its name is the file that runs
	// only where it was opened in the directory the shell runs in.
	stdinBlind := func(f fdFile, shown string) readTarget {
		t := child("", shown, targetScript)
		t.blind, t.base = true, filepath.Base(f.rel)
		if f.gen == st.dirGen {
			t.cwdName = cwdName(f.rel)
		}
		return t
	}
	startup := func(varName, value string, tilde bool) {
		if strings.Contains(value, procSubOperand) {
			sigs = append(sigs, interpreterStreamSignal())
			return
		}
		if shellExpandsStartup(value) {
			sigs = append(sigs, startupExpandedSignal(varName))
			return
		}
		if p, ok := st.resolve(rc, value, tilde); ok {
			out = append(out, child(p, "the startup file "+varName+" names ("+value+")", targetStartup))
		}
	}
	// BASH_ENV is sourced by a non-interactive bash; read for every shell
	// the line hands it to.
	if v, ok := env["BASH_ENV"]; ok && v.ok && v.text != "" {
		startup("BASH_ENV", v.text, v.homeTilde())
	}
	if c.unresolved {
		return out, sigs
	}
	if v, ok := env["ENV"]; ok && v.ok && v.text != "" && c.interactive {
		startup("ENV", v.text, v.homeTilde())
	}
	if c.rcfile >= 0 {
		idx := site + 1 + c.rcfile
		w := s.tokens[idx]
		if wordCouldBe(w, procSubOperand) && !variableCarried(s, idx) {
			sigs = append(sigs, interpreterStreamSignal())
		} else if p, ok := st.resolveToken(rc, s, idx); ok {
			out = append(out, child(p, "the startup file "+w, targetStartup))
		}
	}
	switch name {
	case "zsh":
		base, ok := env["ZDOTDIR"]
		if !ok {
			base, ok = env["HOME"]
		}
		if ok && base.ok && base.text != "" && !c.norcs {
			files := []string{".zshenv"}
			if c.login {
				files = append(files, ".zprofile")
			}
			if c.interactive {
				files = append(files, ".zshrc")
			}
			if c.login {
				// .zlogin after the others, and .zlogout when a login zsh exits.
				files = append(files, ".zlogin", ".zlogout")
			}
			for _, f := range files {
				if p, ok := st.resolve(rc, filepath.Join(base.text, f), base.homeTilde()); ok {
					out = append(out, child(p, "the startup file "+p, targetStartup))
				}
			}
		}
	case "bash", "rbash", "sh":
		if home, ok := env["HOME"]; ok && home.ok && home.text != "" {
			tilde := home.homeTilde()
			if c.login && !c.noprofile {
				for _, f := range []string{".bash_profile", ".bash_login", ".profile"} {
					p, ok := st.resolve(rc, filepath.Join(home.text, f), tilde)
					if !ok {
						break
					}
					if _, err := rc.files.stat(p, true); err == nil {
						out = append(out, child(p, "the startup file "+p, targetStartup))
						break
					}
				}
			}
			if c.login {
				// Read when a login bash exits.
				if p, ok := st.resolve(rc, filepath.Join(home.text, ".bash_logout"), tilde); ok {
					out = append(out, child(p, "the startup file "+p, targetStartup))
				}
			}
			if c.interactive && !c.norc && c.rcfile < 0 {
				if p, ok := st.resolve(rc, filepath.Join(home.text, ".bashrc"), tilde); ok {
					out = append(out, child(p, "the startup file "+p, targetStartup))
				}
			}
		}
	default:
		// The other members read a profile under HOME when they log in, and
		// ksh, mksh and yash an rc file when interactive (ksh and mksh only
		// when ENV names none).
		home, ok := env["HOME"]
		if !ok || !home.ok || home.text == "" {
			break
		}
		var files []string
		if c.login {
			profile := ".profile"
			if name == "yash" {
				profile = ".yash_profile"
			}
			files = append(files, profile)
		}
		if c.interactive {
			if _, envSet := env["ENV"]; !envSet || name == "yash" {
				if rcf := map[string]string{"ksh": ".kshrc", "mksh": ".mkshrc", "yash": ".yashrc"}[name]; rcf != "" {
					files = append(files, rcf)
				}
			}
		}
		for _, f := range files {
			if p, ok := st.resolve(rc, filepath.Join(home.text, f), home.homeTilde()); ok {
				out = append(out, child(p, "the startup file "+p, targetStartup))
			}
		}
	}
	if c.script >= 0 {
		idx := site + 1 + c.script
		w := s.tokens[idx]
		switch {
		case variableCarried(s, idx) || scriptIsStream(w, s.stdinStream):
			// A stream the stream refusal reads; a variable's value the class
			// (2) ruling owes a verdict.
		case containsString(stdinDevices, w):
			if f, ok := st.stdinOf(rc, s); ok && f.ok {
				out = append(out, child(f.path, w+" (its standard input, "+f.path+")", targetScript))
			} else if ok && f.rel != "" {
				out = append(out, stdinBlind(f, w+" (its standard input, "+f.rel+")"))
			}
		default:
			if p, ok := st.resolveToken(rc, s, idx); ok {
				if _, err := rc.files.stat(p, true); err != nil && !strings.Contains(w, "/") {
					// bash searches the PATH it was handed: a prefix on it
					// included.
					pv, set := env["PATH"]
					if q, found, _ := searchPath(rc, st, pv, set, w, false, false, nil, nil); found {
						p = q
					}
				}
				out = append(out, child(p, w, targetScript))
			} else if _, fixed := fixedToken(s, idx); fixed && st.relativeUnknown(w, s.homeTilde(idx)) {
				// The file in a directory the line leaves unknown is unread,
				// and a write before the run may be it (readTarget).
				t := child("", w, targetScript)
				t.blind, t.cwdName = true, cwdName(w)
				out = append(out, t)
			}
		}
	} else if c.stdin && !c.cmdString {
		if f, ok := st.stdinOf(rc, s); ok && f.ok {
			out = append(out, child(f.path, "its standard input ("+f.path+")", targetScript))
		} else if ok && f.rel != "" {
			out = append(out, stdinBlind(f, "its standard input ("+f.rel+")"))
		}
	}
	return out, sigs
}

// shellExpandsStartup reports whether a BASH_ENV or ENV value holds text the
// shell expands before it opens the file: bash runs parameter expansion,
// command substitution and arithmetic expansion on the value
// (iss-2610111600039059), so a `$` or a backquote in it means the file is not
// the one the text names, and a substitution in it runs during the expansion,
// before any file is read. The value is the line's text after its own quoting,
// so a substitution the line ran itself is not here: its output is unknown.
func shellExpandsStartup(value string) bool {
	return strings.ContainsAny(value, "$`")
}

// readTarget reads one file a command points a shell at and judges it.
// before are the files written before the command runs.
func (r Registry) readTarget(rc *readCtx, s segment, st *shellState, t readTarget, before []writeTarget) []payloadSignal {
	if t.path == os.DevNull {
		return nil
	}
	var sigs []payloadSignal
	// A file the line writes before running it is not the file the guard can
	// read now, so the run blocks whatever the file is, a direct run included
	// (product thinker ruling 2026-10-09). An unplaced write is noted only for a
	// file a shell is pointed at: a direct run the guard cannot read is a
	// program, which runs unread like every program. A source runs a file
	// the line writes earlier on its search, whether or not one is there now.
	var written, unplaced bool
	for _, p := range append(append([]string(nil), t.shadows...), t.path) {
		if p == "" {
			continue
		}
		w, u := writtenBefore(p, before)
		written, unplaced = written || w, unplaced || u
	}
	if !written && t.blind && t.cwdName != "" {
		// The same name written in the same unknown directory with no change
		// of directory between is the file that runs, as a placed write is.
		for _, w := range before {
			if w.rel && w.name != "" && sameName(w.name, t.cwdName) && st.sameUnknownDir(w.gen) {
				written = true
				break
			}
		}
	}
	if written {
		return []payloadSignal{scriptWrittenSignal(t.shown)}
	}
	// A file named relative to a working directory the line leaves unknown,
	// or found by a search that looked in one, is not one the guard can
	// name, so a file the line writes before the run may be the file that
	// runs, a direct run's too: a file the guard places, or one written
	// relative to an unknown directory. A write whose path the line does not
	// fix (a variable's value) is not counted for that alone, as it was not
	// before such a run had a target.
	base := t.base
	if base == "" {
		base = filepath.Base(t.shown)
	}
	blind := t.blind && writesAFile(rc, before, base)
	switch {
	case blind:
		sigs = append(sigs, scriptWriteBlindSignal(t.shown))
	case unplaced && t.kind != targetDirect && !(t.blind && t.path == "" && t.missing == ""):
		sigs = append(sigs, scriptWriteUnplacedSignal(t.shown))
	}
	if t.path == "" {
		if !unplaced && !blind && t.missing != "" {
			rc.note("%s", t.missing)
		}
		return sigs
	}
	real, err := filepath.EvalSymlinks(t.path)
	if err != nil {
		switch {
		case t.kind == targetDirect:
		case errors.Is(err, os.ErrNotExist):
			if t.kind == targetScript {
				rc.note("abcd guard: %s does not exist, so there was nothing to read; the shell will refuse it.", t.shown)
			}
		default:
			sigs = append(sigs, scriptUnreadableSignal(t.shown))
		}
		return sigs
	}
	if rc.onStack(real) {
		return sigs // a cycle: everything in it is being read already
	}
	if s.depth+1 > maxPayloadDepth {
		if t.kind == targetDirect {
			// Classified through the budget like any other read.
			fr, ok := rc.take(real)
			if !ok {
				return append(sigs, rc.budgetSignal()...)
			}
			if fr.err != nil || fr.tooBig || !isShellScript(fr) {
				return sigs
			}
		}
		return append(sigs, scriptDepthSignal(t.shown))
	}
	fr, ok := rc.take(real)
	if !ok {
		return append(sigs, rc.budgetSignal()...)
	}
	if t.kind == targetDirect {
		if fr.err != nil || fr.tooBig || !isShellScript(fr) {
			return sigs // a program, allowed unread as every program is
		}
		if unplaced {
			sigs = append(sigs, scriptWriteUnplacedSignal(t.shown))
		}
	} else {
		switch {
		case fr.err != nil:
			return append(sigs, scriptUnreadableSignal(t.shown))
		case bytes.IndexByte(sniff(fr.data), 0) >= 0:
			return append(sigs, scriptBinarySignal(t.shown))
		case fr.tooBig:
			return append(sigs, scriptOversizedSignal(t.shown, fr.size))
		}
	}

	var state *shellState
	if t.sourced {
		state = st.clone()
		state.cd = nil
	} else {
		state = childState(st, t.env, t.shell)
		state.fds = map[int]fdFile{}
	}
	if t.kind == targetDirect && t.startupEnv {
		// A shell script run directly is run by a non-interactive shell, which
		// reads BASH_ENV first.
		if v, ok := t.env["BASH_ENV"]; ok && v.ok && v.text != "" {
			if shellExpandsStartup(v.text) {
				sigs = append(sigs, startupExpandedSignal("BASH_ENV"))
			} else if p, ok := st.resolve(rc, v.text, v.homeTilde()); ok {
				sigs = append(sigs, r.readTarget(rc, s, st, readTarget{path: p, shown: "the startup file BASH_ENV names (" + v.text + ")",
					kind: targetStartup, env: t.env}, before)...)
			}
		}
	}
	text := string(fr.data)
	rc.stack = append(rc.stack, real)
	v, err := r.judge(text, rc, &scriptRun{depth: s.depth + 1, state: state})
	rc.stack = rc.stack[:len(rc.stack)-1]
	if err != nil {
		sig := unparsableSignal(err)
		return append(sigs, carriedOut(sig, t.shown))
	}
	return append(sigs, r.scriptSignals(v, text, t.shown)...)
}

// sniff is the part of a file read to classify it.
func sniff(data []byte) []byte {
	if len(data) > sniffBytes {
		return data[:sniffBytes]
	}
	return data
}

// isShellScript classifies a file run directly: a shell-family shebang, or no
// shebang and no NUL byte, is a shell script; anything else is a program.
func isShellScript(fr fileRead) bool {
	head := sniff(fr.data)
	if bytes.IndexByte(head, 0) >= 0 {
		return false
	}
	if !bytes.HasPrefix(head, []byte("#!")) {
		return true
	}
	line := string(head[2:])
	if i := strings.IndexByte(line, '\n'); i >= 0 {
		line = line[:i]
	}
	words := strings.Fields(line)
	if len(words) == 0 {
		return true
	}
	interp := path.Base(words[0])
	if interp == "env" {
		for _, w := range words[1:] {
			if strings.HasPrefix(w, "-") || isAssignment(w) {
				continue
			}
			interp = path.Base(w)
			break
		}
	}
	return isShellFamily(interp)
}

// writtenBefore reports whether a write before the command lands on p or on a
// directory above it, and whether any write before it is one the guard cannot
// place. Both sides are compared as real locations (a symlinked ancestor, the
// /tmp alias), and case-folded where the filesystem folds case: erring
// toward "the same file" is the safe direction.
func writtenBefore(p string, before []writeTarget) (written, unplaced bool) {
	fold := fsutil.CaseFoldingFS()
	cp := fsutil.RealExistingPath(p)
	for _, w := range before {
		if !w.ok {
			unplaced = true
			continue
		}
		if fsutil.PathWithin(cp, fsutil.RealExistingPath(w.path), fold) {
			return true, unplaced
		}
	}
	return false, unplaced
}

// writesAFile reports whether a write before the command lands on a path
// relative to an unknown working directory, on one the guard places that can
// hold a script (not a device, a pipe or a socket, as /dev/null and
// /dev/stderr are), or on one the guard does not place whose last component
// is literal and is name, the file the run looks up: `> "$T/n.sh"` before a
// run of n.sh, never `> "$f"`.
func writesAFile(rc *readCtx, before []writeTarget, name string) bool {
	for _, w := range before {
		if w.rel || (w.base != "" && sameName(w.base, name)) {
			return true
		}
		if !w.ok || w.path == "" {
			continue
		}
		if fi, err := rc.files.stat(w.path, false); err == nil &&
			fi.Mode()&(os.ModeDevice|os.ModeCharDevice|os.ModeNamedPipe|os.ModeSocket) != 0 {
			continue
		}
		return true
	}
	return false
}

// sameName compares two file names as the filesystem does, case-folded
// where it folds case: erring toward "the same file" is the safe direction.
func sameName(a, b string) bool {
	if fsutil.CaseFoldingFS() {
		return strings.EqualFold(a, b)
	}
	return a == b
}

// scriptSignals carries out of a script what its reading found that
// propagates (decision 5): every registry blocker matched at command position,
// named with the script, the line and the entry; every entry-less block; and
// the reading's own verdicts (a file it could not read). Only block-level
// verdicts on the script's commands propagate (product thinker ruling
// 2026-10-09): a warn-tier entry inside a script, a Tier 2 hit and an
// entry-less warn stay inside, so a script that runs `git clean` on its own
// scratch does not make every run of it warn.
func (r Registry) scriptSignals(v verdicts, text, shown string) []payloadSignal {
	var out []payloadSignal
	entry := func(verdict Verdict, ids []string, named bool) {
		for _, id := range ids {
			e := r.Entries[id]
			line := 1
			if i, ok := v.hitSeg[id]; ok && i < len(v.segs) {
				end := v.segs[i].end
				if end > len(text) {
					end = len(text)
				}
				line += strings.Count(text[:end], "\n")
			}
			sig := payloadSignal{
				id:      scriptHazardEntryID,
				verdict: verdict,
				family:  familyScript,
				reason: fmt.Sprintf("The script %s, which this command runs, runs a command the registry refuses at line %d (%s): %s",
					shown, line, id, e.Why),
				successor: e.Successor + " Fix the script at that line, then run it.",
				also:      []string{id},
			}
			if !named {
				// It fired only where the program's name is a substitution's
				// output or a variable's value, as unknownProgramDecision says.
				u := unknownProgramSignal(verdict, id)
				sig.reason = fmt.Sprintf("The script %s, which this command runs, has a command at line %d whose program name the line does not fix (%s): %s",
					shown, line, id, u.reason)
				sig.successor = u.successor
				sig.also = []string{id, unknownProgramEntryID}
			}
			out = append(out, sig)
		}
	}
	entry(VerdictBlock, v.blockers, true)
	entry(VerdictBlock, v.unnamedBlockers, false)
	for _, sig := range v.signals {
		if sig.verdict == VerdictBlock || sig.fromRead {
			out = append(out, carriedOut(sig, shown))
		}
	}
	return out
}

// carriedOut is an entry-less verdict raised inside a script, as the command
// that runs the script reports it.
func carriedOut(sig payloadSignal, shown string) payloadSignal {
	sig.reason = "In the script " + shown + ", which this command runs: " + sig.reason
	sig.fromRead = sig.fromRead || sig.verdict == VerdictBlock
	return sig
}

func scriptWrittenSignal(shown string) payloadSignal {
	return payloadSignal{
		id: scriptWrittenEntryID, verdict: VerdictBlock, family: familyScript, fromRead: true,
		reason: "This command line writes " + shown + " and then has a shell run it, so the file the guard can read now " +
			"is not the file that runs, and what that shell runs has not been checked.",
		successor: "Split the line: write the script in one command, and run it in the next, so the guard reads what runs.",
	}
}

func scriptWriteUnplacedSignal(shown string) payloadSignal {
	return payloadSignal{
		id: scriptWrittenEntryID, verdict: VerdictWarn, family: familyScript, fromRead: true,
		reason: "This command line writes a file the guard cannot place before a shell runs " + shown +
			", so the script it read may not be the one that runs.",
		successor: "Name the file the line writes, or write it in one command and run the script in the next.",
	}
}

// scriptWriteBlindSignal is the warn for a run of a file named relative to a
// working directory the line leaves unknown, after a write on the line: the
// file that runs may be the one written, wherever the guard placed it.
func scriptWriteBlindSignal(shown string) payloadSignal {
	return payloadSignal{
		id: scriptWrittenEntryID, verdict: VerdictWarn, family: familyScript, fromRead: true,
		reason: "This command line writes a file and then runs " + shown + " in a working directory the line does not " +
			"fix, so the file that runs may be the one it wrote, which the guard has not read.",
		successor: "Run the file by its absolute path, or write it in one command and run it in the next.",
	}
}

func scriptUnreadableSignal(shown string) payloadSignal {
	return payloadSignal{
		id: scriptUnreadEntryID, verdict: VerdictWarn, family: familyScript, fromRead: true,
		reason:    "A shell runs " + shown + ", which the guard could not read (not a regular file it may open), so it has not checked what runs.",
		successor: "Run a regular file the account can read, so the guard reads the script that runs.",
	}
}

func scriptOversizedSignal(shown string, size int64) payloadSignal {
	return payloadSignal{
		id: scriptUnreadEntryID, verdict: VerdictWarn, family: familyScript, fromRead: true,
		reason: fmt.Sprintf("A shell runs %s, which is %d bytes, over the %d the guard reads, so it has not checked what runs.",
			shown, size, maxScriptBytes),
		successor: "Split the script into files under 256 KiB, so the guard reads what runs.",
	}
}

// startupExpandedSignal is the block for a startup variable whose value the
// shell expands before it reads the file: what runs is not text the guard can
// read from the line.
func startupExpandedSignal(varName string) payloadSignal {
	return payloadSignal{
		id: scriptUnreadEntryID, verdict: VerdictBlock, family: familyScript, fromRead: true,
		reason: "This command hands a shell " + varName + " holding a substitution or a variable, which the shell " +
			"expands before it reads the file: a substitution there runs, and the file it names is not one the guard can read.",
		successor: "Set " + varName + " to the startup file's fixed path, or source the file in the command string instead.",
	}
}

// cdpathSignal is the block for a `cd` that searches CDPATH where the guard
// cannot name the directory it enters; fromEnv says the CDPATH is the one the
// check's environment exports, which the command never names.
func cdpathSignal(fromEnv bool) payloadSignal {
	where := "the CDPATH this command sets"
	successor := "Name the directory as an absolute path or with a leading ./ (cd ./sub), which the shell does not search CDPATH for."
	if fromEnv {
		where = "the CDPATH exported in the environment this check runs in, which the command does not set,"
		successor = "Name the directory as an absolute path or with a leading ./ (cd ./sub), which the shell does not search " +
			"CDPATH for, or unset CDPATH in the environment the commands run in."
	}
	return payloadSignal{
		id: scriptUnreadEntryID, verdict: VerdictBlock, family: familyScript, fromRead: true,
		reason: "This command changes directory by a name the shell searches CDPATH for, and " + where +
			" offers no directory holding it, more than one, more entries than the guard looks through, or one the line " +
			"does not fix, so the directory the " +
			"commands after it run in, and the scripts they name, are not ones the guard can read.",
		successor: successor,
	}
}

// sourcePathUnfixedSignal is the block for a `source` whose search reaches an
// entry of a PATH the line sets that the line does not fix.
func sourcePathUnfixedSignal() payloadSignal {
	return payloadSignal{
		id: scriptUnreadEntryID, verdict: VerdictBlock, family: familyScript, fromRead: true,
		reason: "This command sources a file by name through a PATH the line sets, and an entry of that PATH is not " +
			"fixed by the line, so the file the shell finds is not one the guard can read.",
		successor: "Source the file by its path (source ./lib.sh), or set PATH to fixed absolute directories.",
	}
}

// autoloadUnfixedSignal is the block for an autoload whose function file the
// line does not fix: a name, or a directory searched before the file is found.
func autoloadUnfixedSignal() payloadSignal {
	return payloadSignal{
		id: scriptUnreadEntryID, verdict: VerdictBlock, family: familyScript, fromRead: true,
		reason: "This command autoloads a function from an FPATH the line sets, but the line does not fix the name " +
			"or a directory searched for it, so the file the shell loads and runs is not one the guard can read.",
		successor: "Set FPATH (or fpath) to fixed directories and autoload a fixed name, or source the function file by its path.",
	}
}

func scriptBinarySignal(shown string) payloadSignal {
	return payloadSignal{
		id: scriptUnreadEntryID, verdict: VerdictBlock, family: familyScript, fromRead: true,
		reason:    "A shell is handed " + shown + " as text to run, and it holds binary bytes the guard cannot judge.",
		successor: "Run the program directly instead of handing it to a shell.",
	}
}

// scriptDepthSignal is the block for a script run past the depth the reading
// follows (decision 7): it names the script and its own fix, not the
// execute-string layers the shared depth was first written for.
func scriptDepthSignal(shown string) payloadSignal {
	return payloadSignal{
		id: scriptUnreadEntryID, verdict: VerdictBlock, family: familyScript, fromRead: true,
		reason: fmt.Sprintf("This command runs %s inside a chain of scripts deeper than the guard reads "+
			"(%d layers, shared with `sh -c`), so its commands have not been checked.", shown, maxPayloadDepth),
		successor: "Run the inner script directly, in a command of its own, so the guard reads it; or flatten the chain.",
	}
}

func scriptBudgetSignal() payloadSignal {
	return payloadSignal{
		id: scriptUnreadEntryID, verdict: VerdictWarn, family: familyScript, fromRead: true,
		reason: fmt.Sprintf("This command runs more script text than the guard reads for one command (%d files, %d bytes), "+
			"so the scripts past that point have not been checked.", maxScriptFiles, maxScriptReadBytes),
		successor: "Run the scripts in separate commands, so the guard reads each one.",
	}
}
