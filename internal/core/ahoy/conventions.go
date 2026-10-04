package ahoy

// Tools' own conventions files in an adopted project (itd-2610030814013772,
// adr-2610030814023326: AGENTS.md is the one conventions file abcd writes, and
// it never edits a file a tool reads in its place). Some agent tools read a
// file of their own INSTEAD of AGENTS.md when it exists, so a project that
// keeps one hides AGENTS.md, and abcd's block in it, from that tool. Setup
// classifies each such file at the project root, with one guarded read and
// never through a link:
//
//   - a file that only repeats AGENTS.md (a link to it, that link checked out
//     as text, a byte-for-byte copy, a lone @AGENTS.md import line, or a file
//     left blank once abcd's own block is stripped) holds none of the owner's
//     words, so setup offers to retire it. The offer follows the drain rule's
//     precedent (drain_rule.go): asked only of a person at a terminal, never
//     under --yes and never off one, listed under optional_skipped when not
//     asked. Each file is one question; on retire the file is classified
//     again at that moment and removed, from the working tree only, only if
//     it still repeats AGENTS.md. keep and later write and record nothing, so
//     the next setup asks again.
//   - anything else, a file abcd cannot read, cannot read whole or is not a
//     regular file included, is the owner's words. It is never edited, moved,
//     merged or removed, and install names it in a warning printed first.

import (
	"bytes"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/intentdriven/abcd/internal/fsutil"
)

// ConventionsFile covers retiring a tool's own conventions file that only
// repeats AGENTS.md. Its gap is advisory and never acted on under --yes: the
// removal is of a file in the person's project, so only an answered question
// at a terminal removes one.
const ConventionsFile GapCategory = "conventions-file"

// ConventionsOwnerFileGapID is a tool's own conventions file holding the
// owner's words: a warning, never resolvable, since abcd never touches it.
const ConventionsOwnerFileGapID = "conventions.owner_file"

// ConventionsRetireGapID is the offer to retire a tool's own conventions file
// that only repeats AGENTS.md, raised once per such file.
const ConventionsRetireGapID = "conventions.retire_offered"

// conventionsRetirePromptPrefix keys the retirement questions. The key carries
// the file and how it repeats AGENTS.md, so helpFor renders the question's
// words from the key alone and a scripted answer can name the file it answers.
const conventionsRetirePromptPrefix = "conventions_file:"

// toolConventionsFile is one file a tool reads in place of AGENTS.md.
type toolConventionsFile struct {
	// Rel is the file's path from the project root, slash-separated.
	Rel string
	// Tool names the tool that reads Rel and then not AGENTS.md.
	Tool string
	// Qualifier narrows when Tool does so, set off by commas, or "".
	Qualifier string
}

// toolConventionsFiles is the registry, from the 2026-10-03 research note
// (.abcd/development/research/notes/2026-10-03-agents-md-native-reading-recheck-sota.md,
// spc-2610031156364295 open question 4): each file a tool reads in place of
// AGENTS.md, with that tool. Zed reads the first match of a fixed order in
// which these three come before AGENTS.md.
var toolConventionsFiles = []toolConventionsFile{
	{Rel: "CLAUDE.md", Tool: "Claude Code"},
	{Rel: ".claude/CLAUDE.md", Tool: "Claude Code"},
	{Rel: "GEMINI.md", Tool: "Gemini CLI", Qualifier: ", at its default settings,"},
	{Rel: ".rules", Tool: "Zed"},
	{Rel: ".cursorrules", Tool: "Zed"},
	{Rel: ".github/copilot-instructions.md", Tool: "Zed"},
}

// toolFileClass is what a tool's own conventions file is to setup.
type toolFileClass int

const (
	// toolFileAbsent: there is no file of that name.
	toolFileAbsent toolFileClass = iota
	// toolFileOwners: the file holds the owner's words, or cannot be read
	// whole and so is taken to.
	toolFileOwners
	// toolFileRepeats: the file only repeats AGENTS.md.
	toolFileRepeats
)

// The ways a file repeats AGENTS.md, in the order they are checked.
const (
	repeatsLink     = "link"      // a link whose target resolves to the root AGENTS.md
	repeatsLinkText = "link-text" // a regular file holding only such a link's target text
	repeatsBlank    = "blank"     // blank once abcd's own block is stripped
	repeatsCopy     = "copy"      // a byte-for-byte copy of AGENTS.md
	repeatsImport   = "import"    // its only non-blank line imports AGENTS.md
)

// repeatKinds is every way a file repeats AGENTS.md.
var repeatKinds = []string{repeatsLink, repeatsLinkText, repeatsBlank, repeatsCopy, repeatsImport}

// repeatsWhat says, after the file's name, what the file is: the material the
// retirement question opens on.
var repeatsWhat = map[string]string{
	repeatsLink:     "is a link to AGENTS.md",
	repeatsLinkText: "holds only the text AGENTS.md, a link to it saved as a plain file",
	repeatsBlank:    "is empty once abcd's own block is taken out",
	repeatsCopy:     "is an exact copy of AGENTS.md",
	repeatsImport:   "holds only a line that loads AGENTS.md",
}

// The retirement question's answers: decide-later last, and the default.
const (
	retireRetire = "retire"
	retireKeep   = "keep"
	retireLater  = "later"
)

var retireChoices = []string{retireRetire, retireKeep, retireLater}

// conventionsScan classifies the registry files under one project root. It
// holds the root open as an os.Root, so every read and the removal stay inside
// the project, and it reads the root's AGENTS.md at most once: a scan is one
// look at the project, and a look taken later (the check at a retire answer)
// is a new scan that reads AGENTS.md afresh.
type conventionsScan struct {
	root string   // the project root, absolute
	r    *os.Root // the root held open, or nil when it cannot be opened

	agents     []byte // the root AGENTS.md, guarded, once read
	agentsErr  error
	agentsRead bool

	agentsResolved     string // the root AGENTS.md as the kernel resolves it, or ""
	agentsResolvedDone bool

	rootResolved     string // the root as the kernel resolves it, or ""
	rootResolvedDone bool

	names map[string][]string // each folder's entry names, once listed, by its path from the root
}

// newConventionsScan opens a scan of root; close releases it.
func newConventionsScan(root string) *conventionsScan {
	s := &conventionsScan{root: root, names: map[string][]string{}}
	if r, err := os.OpenRoot(root); err == nil {
		s.r = r
	}
	return s
}

func (s *conventionsScan) close() {
	if s.r != nil {
		s.r.Close()
	}
}

// readRootAgents reads the root's AGENTS.md, guarded and never through a link.
// A variable so a test can count the reads a scan makes.
var readRootAgents = func(r *os.Root) ([]byte, error) {
	return fsutil.ReadGuardedInRoot(r, "AGENTS.md", maxAhoyFileBytes)
}

// rootAgents is the root's AGENTS.md, read on the scan's first need of it.
func (s *conventionsScan) rootAgents() ([]byte, error) {
	if !s.agentsRead {
		s.agentsRead = true
		s.agents, s.agentsErr = readRootAgents(s.r)
	}
	return s.agents, s.agentsErr
}

// agentsResolvedPath is where the root's AGENTS.md physically is: its path with
// every link resolved, by lstat and readlink alone, reading no file's content.
// "" when it cannot be resolved (there is none).
func (s *conventionsScan) agentsResolvedPath() string {
	if !s.agentsResolvedDone {
		s.agentsResolvedDone = true
		s.agentsResolved, _ = filepath.EvalSymlinks(filepath.Join(s.root, "AGENTS.md"))
	}
	return s.agentsResolved
}

// rootResolvedPath is the root with every link in its own path resolved, or "".
func (s *conventionsScan) rootResolvedPath() string {
	if !s.rootResolvedDone {
		s.rootResolvedDone = true
		s.rootResolved, _ = filepath.EvalSymlinks(s.root)
	}
	return s.rootResolved
}

// toolFileAt is one registry file as a scan found it.
type toolFileAt struct {
	class toolFileClass
	how   string // how it repeats AGENTS.md, when it does
	// shown is the file's path from the root as it is spelt on disk, which on
	// a case-insensitive filesystem may differ from the registry's (a
	// claude.md the tool reads as CLAUDE.md), slash-separated.
	shown string
	// dir is the file's folder, held open, and leaf its name there: the very
	// entry that was classified, so a removal takes that entry and no other.
	// Set only for a file that repeats AGENTS.md; release closes it.
	dir   *os.Root
	leaf  string
	owned bool // dir was opened for this file and is closed by release
}

func (at toolFileAt) release() {
	if at.owned && at.dir != nil {
		at.dir.Close()
	}
}

// classify classifies the registry file rel. It reads the file at most once,
// guarded, inside the root, and never through a link: a link is judged by
// where the kernel resolves it, by lstat and readlink alone, and a file in a
// linked folder is the owner's, unread. Each folder between the root and the
// file is held open as it is checked, and checked to be the folder that was
// looked at, so a folder swapped for a link mid-look cannot carry the read or
// a later removal anywhere else.
func (s *conventionsScan) classify(rel string) toolFileAt {
	parts := strings.Split(rel, "/")
	owners := func(shown []string) toolFileAt {
		return toolFileAt{class: toolFileOwners, shown: strings.Join(append(shown, parts[len(shown):]...), "/")}
	}
	// presentThroughLink says whether the tool, reading through whatever stands
	// in the folder chain, finds a file: metadata only, nothing is read.
	presentThroughLink := func() bool {
		_, err := os.Lstat(filepath.Join(s.root, filepath.FromSlash(rel)))
		return err == nil
	}
	if s.r == nil {
		if presentThroughLink() {
			return owners(nil)
		}
		return toolFileAt{class: toolFileAbsent, shown: rel}
	}
	dir, owned := s.r, false
	closeDir := func() {
		if owned {
			dir.Close()
		}
	}
	var shown []string
	for _, part := range parts[:len(parts)-1] {
		fi, err := dir.Lstat(part)
		if errors.Is(err, fs.ErrNotExist) {
			closeDir()
			return toolFileAt{class: toolFileAbsent, shown: rel}
		}
		if err != nil || !fi.IsDir() || fi.Mode()&os.ModeSymlink != 0 {
			// The tool reads through whatever this is; abcd reads nothing.
			closeDir()
			if !presentThroughLink() {
				return toolFileAt{class: toolFileAbsent, shown: rel}
			}
			return owners(shown)
		}
		name := s.onDisk(dir, strings.Join(shown, "/"), part, fi)
		sub, err := dir.OpenRoot(part)
		if err != nil {
			closeDir()
			return owners(shown)
		}
		st, err := sub.Stat(".")
		if err != nil || !os.SameFile(fi, st) {
			// Swapped for something else while it was looked at.
			sub.Close()
			closeDir()
			return owners(shown)
		}
		closeDir()
		dir, owned = sub, true
		shown = append(shown, name)
	}
	leaf := parts[len(parts)-1]
	fi, err := dir.Lstat(leaf)
	if errors.Is(err, fs.ErrNotExist) {
		closeDir()
		return toolFileAt{class: toolFileAbsent, shown: rel}
	}
	if err != nil {
		closeDir()
		return owners(shown)
	}
	name := s.onDisk(dir, strings.Join(shown, "/"), leaf, fi)
	at := toolFileAt{class: toolFileOwners, shown: strings.Join(append(shown, name), "/")}
	if how := s.repeats(dir, at.shown, name, fi); how != "" {
		at.class, at.how, at.dir, at.leaf, at.owned = toolFileRepeats, how, dir, name, owned
		return at
	}
	closeDir()
	return at
}

// repeats says how the entry name in dir, at shown from the root, repeats
// AGENTS.md, or "" when it is the owner's words.
func (s *conventionsScan) repeats(dir *os.Root, shown, name string, fi fs.FileInfo) string {
	if fi.Mode()&os.ModeSymlink != 0 {
		// A link repeats AGENTS.md only when the kernel resolves it to the
		// root's AGENTS.md itself. Its target text alone could name AGENTS.md
		// through a linked folder (docs/../AGENTS.md with docs a link out of
		// the project) and so reach a file outside it.
		agents := s.agentsResolvedPath()
		resolved, err := filepath.EvalSymlinks(filepath.Join(s.root, filepath.FromSlash(shown)))
		if err == nil && resolved != "" && resolved == agents {
			return repeatsLink
		}
		// Before the root AGENTS.md exists there is nothing to resolve to, and
		// the link dangles: it holds no words, and once AGENTS.md is written
		// the tool reads AGENTS.md through it. Its target text alone is judged
		// then, by the plain spelling only (namesRootAgents), and a check made
		// after AGENTS.md is written judges it by resolution again.
		if agents == "" {
			if text, lerr := dir.Readlink(name); lerr == nil && s.namesRootAgents(shown, text) {
				return repeatsLink
			}
		}
		return ""
	}
	if !fi.Mode().IsRegular() {
		return ""
	}
	data, err := fsutil.ReadGuardedInRoot(dir, name, maxAhoyFileBytes)
	if err != nil {
		return ""
	}
	trimmed := strings.TrimSpace(string(data))
	oneLine := trimmed != "" && !strings.ContainsAny(trimmed, "\r\n")
	if oneLine && s.namesRootAgents(shown, trimmed) {
		return repeatsLinkText
	}
	if stripped, _ := StripMarkerBlock(data); len(bytes.TrimSpace(stripped)) == 0 {
		return repeatsBlank
	}
	if agents, aerr := s.rootAgents(); aerr == nil && bytes.Equal(data, agents) {
		return repeatsCopy
	}
	if imp, ok := strings.CutPrefix(trimmed, "@"); ok && oneLine && s.namesRootAgents(shown, imp) {
		return repeatsImport
	}
	return ""
}

// namesRootAgents reports whether ref, the text of a link saved as a plain
// file or an import path, written in the file at shown, names the root's
// AGENTS.md. Text is judged as written, and nothing is followed, so only the
// plain spelling counts: from the file's own folder, as many ../ as the file
// is deep and then AGENTS.md (./AGENTS.md at the root too), or the root's
// AGENTS.md by its absolute path. A spelling that steps into a folder and back
// out (x/../AGENTS.md), or out of the project and back in, is the owner's: a
// tool following it physically could leave the project through a linked
// folder, and abcd will not judge where it lands without following it.
func (s *conventionsScan) namesRootAgents(shown, ref string) bool {
	if ref == "" {
		return false
	}
	if p := filepath.FromSlash(ref); filepath.IsAbs(p) {
		if p != filepath.Clean(p) {
			return false
		}
		if p == filepath.Join(s.root, "AGENTS.md") {
			return true
		}
		// The root by its resolved path (a temporary folder reached through a
		// linked parent); the root's own resolution reads no file of the
		// project.
		resolved := s.rootResolvedPath()
		return resolved != "" && p == filepath.Join(resolved, "AGENTS.md")
	}
	want := strings.Repeat("../", strings.Count(shown, "/")) + "AGENTS.md"
	return ref == want || (want == "AGENTS.md" && ref == "./AGENTS.md")
}

// onDisk is how the entry name in dir (at folder from the root) is spelt on
// disk. It differs from name only on a case-insensitive filesystem, where
// lstat finds a claude.md under the name CLAUDE.md; the folder is listed, once
// per scan, only when a case-swapped lstat finds the same file.
func (s *conventionsScan) onDisk(dir *os.Root, folder, name string, fi fs.FileInfo) string {
	swapped := strings.ToLower(name)
	if swapped == name {
		swapped = strings.ToUpper(name)
	}
	if swapped == name {
		return name
	}
	if other, err := dir.Lstat(swapped); err != nil || !os.SameFile(fi, other) {
		return name // a case-sensitive filesystem: name is the spelling
	}
	names, listed := s.names[folder]
	if !listed {
		if f, err := dir.Open("."); err == nil {
			names, _ = f.Readdirnames(-1)
			f.Close()
		}
		s.names[folder] = names
	}
	folded := ""
	for _, n := range names {
		if n == name {
			return name
		}
		if folded == "" && strings.EqualFold(n, name) {
			folded = n
		}
	}
	if folded != "" {
		return folded
	}
	return name
}

// ownerFileWarning is the one line install prints first for a tool's own
// conventions file holding the owner's words.
func ownerFileWarning(f toolConventionsFile) string {
	return f.Rel + " holds your own words, so " + f.Tool + f.Qualifier + " reads it and not AGENTS.md: " +
		"abcd's rules stay hidden from " + f.Tool + " in this project until you move those words into AGENTS.md " +
		"and remove " + f.Rel + ". abcd never edits or removes this file."
}

// detectToolConventionsFiles raises, for each registry file at the root, the
// owner's-file warning or the retirement offer, naming each file as it is
// spelt on disk.
func detectToolConventionsFiles(root string) []Gap {
	s := newConventionsScan(root)
	defer s.close()
	var gaps []Gap
	for _, reg := range toolConventionsFiles {
		at := s.classify(reg.Rel)
		at.release()
		f := reg
		f.Rel = at.shown
		switch at.class {
		case toolFileOwners:
			gaps = append(gaps, Gap{
				ID: ConventionsOwnerFileGapID, Category: ConventionsFile, Scope: "repo",
				Title:    f.Rel + " holds your own words and hides AGENTS.md from " + f.Tool,
				Detail:   ownerFileWarning(f),
				FixHint:  "Move the words in " + f.Rel + " into AGENTS.md and remove " + f.Rel + "; abcd never edits or removes it.",
				Required: false, Resolvable: false,
			})
		case toolFileRepeats:
			gaps = append(gaps, Gap{
				ID: ConventionsRetireGapID, Category: ConventionsFile, Scope: "repo",
				Title: f.Rel + " only repeats AGENTS.md",
				Detail: f.Rel + " " + repeatsWhat[at.how] + ", and " + f.Tool + f.Qualifier + " reads it in place of AGENTS.md. " +
					"It holds nothing of your own, so it can go.",
				FixHint: "ahoy install at a terminal offers to remove it and removes it only on your answer; --yes never does. " +
					"Or remove it yourself.",
				Required: false, Resolvable: true,
			})
		}
	}
	return gaps
}

// installWarnings is the warning lines the gaps carry, the owner's files in
// gap order and then the host-reach warnings in the order they were raised.
func installWarnings(gaps, hostReach []Gap) []string {
	out := []string{}
	for _, g := range gaps {
		if g.ID == ConventionsOwnerFileGapID {
			out = append(out, g.Detail)
		}
	}
	for _, g := range hostReach {
		out = append(out, g.Detail)
	}
	return out
}

// retirePromptKey is the key of the question retiring rel, which repeats
// AGENTS.md as how.
func retirePromptKey(rel, how string) string {
	return conventionsRetirePromptPrefix + rel + ":" + how
}

// retireAsk is the question itself, the last sentence of the help.
func retireAsk(rel string) string { return "Remove " + rel + " from this project?" }

// conventionsRetireHelp is helpFor's answer for a retirement question's key:
// the file and what it repeats first, the question last, then the answers
// with decide-later last. The key names the file as it is spelt on disk, which
// matches the registry's spelling but for letter case. A key naming a file the
// registry does not hold, or a way of repeating AGENTS.md that does not exist,
// has no help.
func conventionsRetireHelp(key string) (PromptHelp, bool) {
	rest, ok := strings.CutPrefix(key, conventionsRetirePromptPrefix)
	if !ok {
		return PromptHelp{}, false
	}
	i := strings.LastIndex(rest, ":")
	if i < 0 {
		return PromptHelp{}, false
	}
	rel, how := rest[:i], rest[i+1:]
	what, known := repeatsWhat[how]
	if !known {
		return PromptHelp{}, false
	}
	for _, f := range toolConventionsFiles {
		if !strings.EqualFold(f.Rel, rel) {
			continue
		}
		f.Rel = rel
		return PromptHelp{
			Key: key,
			About: f.Rel + " " + what + ", so it holds nothing of your own. " + f.Tool + f.Qualifier +
				" reads it in place of AGENTS.md; without it, " + f.Tool + " reads AGENTS.md. " + retireAsk(f.Rel),
			Choices: []ChoiceHelp{
				{Value: retireRetire, Meaning: "removes it from the working tree; you commit the removal."},
				{Value: retireKeep, Meaning: "leaves it as it is; the next install asks again."},
				{Value: retireLater, Meaning: "leaves it for now; the next install asks again."},
			},
		}, true
	}
	return PromptHelp{}, false
}

// beforeToolFileRemove, when set, runs just before a retired file is removed:
// a test's seam for changing the project in that window.
var beforeToolFileRemove func(rel string)

// stepConventionsFiles asks, of a person at a terminal, whether to retire
// each tool's own conventions file that only repeats AGENTS.md. It runs after
// stepDrainRule, the last consent question, and so after stepMarker, whose
// retraction may just have left a CLAUDE.md blank. It never runs under --yes,
// and never off a terminal.
func (a *applyCtx) stepConventionsFiles() {
	if a.autoYes || !atTerminal(a.prompter) || !a.approved[ConventionsFile] || !a.has(ConventionsRetireGapID) {
		return
	}
	s := newConventionsScan(a.cwd)
	defer s.close()
	for _, f := range toolConventionsFiles {
		at := s.classify(f.Rel)
		at.release()
		if at.class != toolFileRepeats {
			continue
		}
		if a.prompter.Prompt(retirePromptKey(at.shown, at.how), retireChoices, retireLater) != retireRetire {
			continue // keep and later write nothing and record nothing
		}
		a.retireToolFile(f.Rel, at.shown)
	}
}

// retireToolFile removes the registry file rel, shown as spelt on disk, after
// classifying it again: a file that changed while the question was open may
// now hold the owner's words, and is left. The check is a new scan, so it
// compares against AGENTS.md as it is now, and the removal takes the very
// entry it classified, in its folder held open: never a path walked afresh,
// which a folder swapped for a link would carry out of the project.
func (a *applyCtx) retireToolFile(rel, shown string) {
	s := newConventionsScan(a.cwd)
	defer s.close()
	again := s.classify(rel)
	defer again.release()
	if again.class != toolFileRepeats {
		a.refuse(shown + " was not removed: it changed while the question was open and no longer only repeats AGENTS.md, so it is left as it is.")
		return
	}
	if beforeToolFileRemove != nil {
		beforeToolFileRemove(rel)
	}
	if err := again.dir.Remove(again.leaf); err != nil {
		a.refuse("could not remove " + again.shown + ": " + errText(err) + "; it is left as it is.")
		return
	}
	a.note(writeToolFileRetired, filepath.Join(a.cwd, filepath.FromSlash(again.shown)))
}
