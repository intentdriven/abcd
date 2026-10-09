package guard

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
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
	// guard did not read: binary (block), unreadable or over the cap (warn),
	// or past the check's read budget (warn).
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
var trackedVars = map[string]bool{"BASH_ENV": true, "ENV": true, "HOME": true, "ZDOTDIR": true, "PATH": true}

// Files is the directory a command runs in, and the files the guard has read
// for it. A front door that checks one command against several registries
// hands each the same Files, so each file is read once per command.
type Files struct {
	dir  string
	home string
	path string

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
	return &Files{dir: filepath.Clean(dir), home: home, path: os.Getenv("PATH"), cache: map[string]fileRead{}, stats: map[string]statResult{}}
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
// set and which of them are exported, and the descriptors the line opened on
// files.
type shellState struct {
	dir      string
	dirOK    bool
	cd       *pendingCD
	vars     map[string]varVal
	exported map[string]bool
	fds      map[int]fdFile
}

type pendingCD struct {
	dir   string
	chain int
}

// varVal is a variable's value as the line set it; ok is false where the
// line does not fix it.
type varVal struct {
	text string
	ok   bool
}

// fdFile is a descriptor open on a file; ok is false where the line does not
// fix which.
type fdFile struct {
	path string
	ok   bool
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

// at is the state a segment runs in: a pending cd holds for a command chained
// after it with `&&`, and once the chain breaks the directory is unknown for
// the rest of the text — never the one the shell would be in had the cd failed.
func (st *shellState) at(s segment) *shellState {
	if st.cd == nil {
		return st
	}
	n := st.clone()
	if s.chain == st.cd.chain && s.afterAnd {
		n.dir, n.dirOK = st.cd.dir, true
		return n
	}
	n.dir, n.dirOK, n.cd = "", false, nil
	return n
}

// unresolved returns st with its directory unknown.
func (st *shellState) unresolved() *shellState {
	n := st.clone()
	n.dir, n.dirOK, n.cd = "", false, nil
	return n
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
	if filepath.IsAbs(word) {
		return filepath.Clean(word), true
	}
	if !st.dirOK || st.dir == "" {
		return "", false
	}
	return filepath.Join(st.dir, word), true
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
	return st.resolve(rc, tok, strings.HasPrefix(tok, "~"))
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
	if !isAssignment(tok) {
		return "", varVal{}, false
	}
	eq := strings.IndexByte(tok, '=')
	v := tok[eq+1:]
	return tok[:eq], varVal{text: v, ok: !isUnknown(v) && !strings.ContainsRune(v, varMark)}, true
}

// alwaysExported are the variables every shell inherits exported, so a plain
// assignment changes what a child sees.
var alwaysExported = map[string]bool{"HOME": true, "PATH": true}

// env is the environment the command at site in s runs with: the exported
// variables of st, and the assignments written before the command (a prefix,
// or an env wrapper's).
func (st *shellState) env(s segment, site int) map[string]varVal {
	out := map[string]varVal{}
	for k, v := range st.vars {
		if st.exported[k] || alwaysExported[k] {
			out[k] = v
		}
	}
	for i := 0; i < site && i < len(s.tokens); i++ {
		if name, v, ok := assignment(s.tokens[i]); ok && trackedVars[name] {
			out[name] = v
		}
	}
	return out
}

// childState is the state a child shell starts in: the directory st runs
// in, and the environment it hands down, every variable of it exported.
func childState(st *shellState, env map[string]varVal) *shellState {
	n := &shellState{dir: st.dir, dirOK: st.dirOK, vars: map[string]varVal{}, exported: map[string]bool{}, fds: map[int]fdFile{}}
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
				p, ok := st.resolve(rc, t, r.target.tilde)
				set(fd, fdFile{path: p, ok: ok})
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
		p, ok := st.resolve(rc, r.target.text, r.target.tilde)
		set(fd, fdFile{path: p, ok: ok})
	}
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
}

// scriptSeg is the reading's record of one segment: the state it runs in,
// its place on the line, and the files it writes.
type scriptSeg struct {
	state  *shellState
	order  []int
	writes []writeTarget
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

	// The state each segment runs in, walked in order within each shell: the
	// text's own commands from top, and a string's commands from the state
	// its carrier hands its child shell.
	scopes := map[int]*shellState{}
	for i, s := range segs {
		st, ok := scopes[s.carrier]
		if !ok {
			if s.carrier == 0 || s.carrier-1 >= i {
				st = top
			} else {
				c := s.carrier - 1
				cs := recs[c].state
				site := lastSite(segs[c])
				st = childState(cs, cs.env(segs[c], site))
				cs.applyRedirects(rc, st.fds, segs[c].redirects)
				if s.fromFixedOutput || isEvalCarrier(segs[c]) {
					// Another reading of the carrier, or a string eval runs in
					// the carrier's own shell.
					st = cs.clone()
					st.cd = nil
				}
			}
		}
		st = st.at(s)
		recs[i].state = st
		if s.carrier > 0 && s.carrier-1 < i {
			recs[i].order = append(append([]int(nil), recs[s.carrier-1].order...), i)
		} else {
			recs[i].order = []int{i}
		}
		scopes[s.carrier] = r.after(rc, st, s, dirChangeBelow[i])
	}

	// The targets each segment points a shell at, and only then the writes:
	// most lines run no script, and pay for nothing more.
	type pending struct {
		seg int
		t   readTarget
	}
	var targets []pending
	var signals []payloadSignal
	for i, s := range segs {
		ts, sigs := r.targetsOf(rc, s, recs[i].state)
		signals = append(signals, sigs...)
		for _, t := range ts {
			targets = append(targets, pending{seg: i, t: t})
		}
	}
	if len(targets) == 0 {
		return signals
	}
	for i, s := range segs {
		recs[i].writes = writesOf(rc, s, recs[i].state)
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
		signals = append(signals, r.readTarget(rc, segs[p.seg], recs[p.seg].state, p.t, before)...)
	}
	return signals
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
	allAssign := len(s.tokens) > 0
	for _, t := range s.tokens {
		if !isAssignment(t) {
			allAssign = false
			break
		}
	}
	if allAssign {
		n := st
		for _, t := range s.tokens {
			if name, v, _ := assignment(t); trackedVars[name] {
				if n == st {
					n = st.clone()
				}
				n.vars[name] = v
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
		case isUnknown(tok) && nameCouldBeAny(tok, directoryChanges):
			out = out.unresolved()
		case name == "cd" || name == "pushd":
			out = cdTarget(rc, out, s, a.idx)
		case name == "popd":
			out = out.unresolved()
		case name == "eval" && stringChangesDir:
			out = out.unresolved()
		case (name == "source" || name == ".") && sourcedChangesDir(rc, out, s, a.idx):
			out = out.unresolved()
		case name == "export" || name == "declare" || name == "typeset":
			// `declare -x`/`typeset -x` export as `export` does; `export -n`
			// and `declare +x` take the export away. A declare without -x sets
			// the variable unexported.
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
			for _, w := range args {
				nm, v, ok := assignment(w)
				if !ok {
					nm = w
				}
				if !trackedVars[nm] {
					continue
				}
				if ok {
					n.vars[nm] = v
				}
				switch {
				case exports:
					n.exported[nm] = true
				case unexports:
					delete(n.exported, nm)
				}
			}
			out = n
		case name == "unset":
			n := out.clone()
			for _, w := range args {
				delete(n.vars, w)
				delete(n.exported, w)
			}
			out = n
		}
	}
	return out
}

// cdTarget is the state after a `cd`/`pushd` at site: its target, when it
// exists as a directory now, holds for the commands chained after it with
// `&&`; any other cd leaves the directory unknown.
func cdTarget(rc *readCtx, st *shellState, s segment, site int) *shellState {
	i := site + 1
	for ; i < len(s.tokens); i++ {
		t := s.tokens[i]
		if t == "--" {
			i++
			break
		}
		if t == "-L" || t == "-P" || t == "-e" || t == "-@" || t == "-LP" || t == "-PL" {
			continue
		}
		break
	}
	var target string
	var ok bool
	if i >= len(s.tokens) {
		target, ok = st.home(rc)
	} else if s.tokens[i] == "-" || strings.HasPrefix(s.tokens[i], "+") {
		ok = false
	} else {
		target, ok = st.resolveToken(rc, s, i)
	}
	n := st.unresolved()
	if !ok || target == "" {
		return n
	}
	if fi, err := os.Stat(target); err != nil || !fi.IsDir() {
		return n
	}
	n.cd = &pendingCD{dir: filepath.Clean(target), chain: s.chain}
	return n
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
	// startup are the startup files a direct-run shell script reads
	// (BASH_ENV), read once it is classified as one.
	startupEnv bool
}

// targetsOf is every file a segment's commands point a shell at, and the
// stream verdicts on a startup option or variable handed a stream.
func (r Registry) targetsOf(rc *readCtx, s segment, st *shellState) ([]readTarget, []payloadSignal) {
	var out []readTarget
	var sigs []payloadSignal
	for _, a := range commandSites(s) {
		tok := s.tokens[a.idx]
		if isUnknown(tok) || s.globAt(a.idx) || strings.ContainsRune(tok, varMark) {
			continue
		}
		name := strings.ToLower(path.Base(tok))
		if !isShellFamily(name) && name != "source" && name != "." && !strings.Contains(tok, "/") {
			continue
		}
		// The bound holds before any lookup: a line naming thousands of
		// scripts resolves sixty-four of them and warns through the budget.
		if rc.nSites >= maxScriptTargets {
			sigs = append(sigs, rc.budgetSignal()...)
			break
		}
		rc.nSites++
		env := st.env(s, a.idx)
		switch {
		case isShellFamily(name):
			ts, ss := shellTargets(rc, s, st, a.idx, name, env)
			out = append(out, ts...)
			sigs = append(sigs, ss...)
		case name == "source" || name == ".":
			if p, ok := sourcePath(rc, st, s, a.idx); ok {
				out = append(out, readTarget{path: p, shown: sourceWord(s, a.idx), kind: targetScript, sourced: true, env: env})
			} else if w := sourceWord(s, a.idx); w != "" && !strings.Contains(w, "/") {
				if _, fixed := fixedToken(s, sourceIndex(s, a.idx)); fixed && !scriptIsStream(w, s.stdinStream) {
					rc.note("abcd guard: `%s %s` names no file the shell can find, so there was nothing to read; if the shell reaches it, it refuses it.", tok, w)
				}
			}
		case strings.Contains(tok, "/"):
			if p, ok := st.resolveToken(rc, s, a.idx); ok {
				out = append(out, readTarget{path: p, shown: tok, kind: targetDirect, env: env, startupEnv: true})
			}
		}
	}
	return out, sigs
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

// sourcePath resolves a `source`/`.` operand as bash does: a word with a
// slash as a path, any other searched on PATH and then in the working
// directory. ok is false where the line does not fix the word, where it is a
// stream (the stream refusal reads it), or where no file is found by search.
func sourcePath(rc *readCtx, st *shellState, s segment, site int) (string, bool) {
	i := sourceIndex(s, site)
	w, ok := fixedToken(s, i)
	if !ok || scriptIsStream(w, s.stdinStream) {
		return "", false
	}
	if strings.Contains(w, "/") || strings.HasPrefix(w, "~") {
		return st.resolveToken(rc, s, i)
	}
	return searchPath(rc, st, w, true)
}

// searchPath finds a bare name on PATH, as the line set it or as the
// account's, and then, for `source`, in the working directory.
func searchPath(rc *readCtx, st *shellState, name string, thenCwd bool) (string, bool) {
	pathVar := rc.files.path
	if v, ok := st.vars["PATH"]; ok {
		if !v.ok {
			return "", false
		}
		pathVar = v.text
	}
	for _, d := range filepath.SplitList(pathVar) {
		if d == "" || !filepath.IsAbs(d) {
			continue
		}
		p := filepath.Join(d, name)
		if fi, err := rc.files.stat(p, false); err == nil && fi.Mode().IsRegular() {
			return p, true
		}
	}
	if thenCwd {
		if p, ok := st.resolve(rc, name, false); ok {
			if _, err := rc.files.stat(p, false); err == nil {
				return p, true
			}
		}
	}
	return "", false
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
		return readTarget{path: p, shown: shown, kind: kind, env: env}
	}
	startup := func(varName, value string, tilde bool) {
		if strings.Contains(value, procSubOperand) {
			sigs = append(sigs, interpreterStreamSignal())
			return
		}
		if p, ok := st.resolve(rc, value, tilde); ok {
			out = append(out, child(p, "the startup file "+varName+" names ("+value+")", targetStartup))
		}
	}
	// BASH_ENV is sourced by a non-interactive bash; read for every shell
	// the line hands it to.
	if v, ok := env["BASH_ENV"]; ok && v.ok && v.text != "" {
		startup("BASH_ENV", v.text, strings.HasPrefix(v.text, "~"))
	}
	if c.unresolved {
		return out, sigs
	}
	if v, ok := env["ENV"]; ok && v.ok && v.text != "" && c.interactive {
		startup("ENV", v.text, strings.HasPrefix(v.text, "~"))
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
				if p, ok := st.resolve(rc, filepath.Join(base.text, f), strings.HasPrefix(base.text, "~")); ok {
					out = append(out, child(p, "the startup file "+p, targetStartup))
				}
			}
		}
	case "bash", "rbash", "sh":
		if home, ok := env["HOME"]; ok && home.ok && home.text != "" {
			tilde := strings.HasPrefix(home.text, "~")
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
			if p, ok := st.resolve(rc, filepath.Join(home.text, f), strings.HasPrefix(home.text, "~")); ok {
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
			}
		default:
			if p, ok := st.resolveToken(rc, s, idx); ok {
				if _, err := rc.files.stat(p, true); err != nil && !strings.Contains(w, "/") {
					if q, found := searchPath(rc, st, w, false); found {
						p = q
					}
				}
				out = append(out, child(p, w, targetScript))
			}
		}
	} else if c.stdin && !c.cmdString {
		if f, ok := st.stdinOf(rc, s); ok && f.ok {
			out = append(out, child(f.path, "its standard input ("+f.path+")", targetScript))
		}
	}
	return out, sigs
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
	// program, which runs unread like every program.
	written, unplaced := writtenBefore(t.path, before)
	if written {
		return []payloadSignal{scriptWrittenSignal(t.shown)}
	}
	if unplaced && t.kind != targetDirect {
		sigs = append(sigs, scriptWriteUnplacedSignal(t.shown))
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
		state = childState(st, t.env)
		state.fds = map[int]fdFile{}
	}
	if t.kind == targetDirect && t.startupEnv {
		// A shell script run directly is run by a non-interactive shell, which
		// reads BASH_ENV first.
		if v, ok := t.env["BASH_ENV"]; ok && v.ok && v.text != "" {
			if p, ok := st.resolve(rc, v.text, strings.HasPrefix(v.text, "~")); ok {
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
