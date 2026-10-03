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
// the file and how it repeats AGENTS.md, so HelpFor renders the question's
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

// classifyToolFile classifies the tool's own conventions file at rel under
// root, and says how it repeats AGENTS.md when it does. It reads the file at
// most once, guarded, and never through a link: a link is judged by its target
// text alone, and a file in a linked folder is the owner's, unread.
func classifyToolFile(root, rel string) (toolFileClass, string) {
	path := filepath.Join(root, filepath.FromSlash(rel))
	// Every folder between the root and the file must be a real directory: a
	// linked folder would carry the read outside the project.
	dir := root
	for _, part := range strings.Split(rel, "/")[:strings.Count(rel, "/")] {
		dir = filepath.Join(dir, part)
		fi, err := os.Lstat(dir)
		if errors.Is(err, fs.ErrNotExist) {
			return toolFileAbsent, ""
		}
		if err != nil || !fi.IsDir() || fi.Mode()&os.ModeSymlink != 0 {
			// The tool reads through whatever this is; abcd reads nothing.
			if _, serr := os.Lstat(path); serr != nil {
				return toolFileAbsent, ""
			}
			return toolFileOwners, ""
		}
	}
	fi, err := os.Lstat(path)
	if errors.Is(err, fs.ErrNotExist) {
		return toolFileAbsent, ""
	}
	if err != nil {
		return toolFileOwners, ""
	}
	if fi.Mode()&os.ModeSymlink != 0 {
		target, rerr := os.Readlink(path)
		if rerr == nil && namesRootAgents(root, path, target) {
			return toolFileRepeats, repeatsLink
		}
		return toolFileOwners, ""
	}
	if !fi.Mode().IsRegular() {
		return toolFileOwners, ""
	}
	data, err := fsutil.ReadGuarded(path, maxAhoyFileBytes)
	if err != nil {
		return toolFileOwners, ""
	}
	trimmed := strings.TrimSpace(string(data))
	if trimmed != "" && !strings.ContainsAny(trimmed, "\r\n") && namesRootAgents(root, path, trimmed) {
		return toolFileRepeats, repeatsLinkText
	}
	if stripped, _ := StripMarkerBlock(data); len(bytes.TrimSpace(stripped)) == 0 {
		return toolFileRepeats, repeatsBlank
	}
	if agents, aerr := fsutil.ReadGuarded(filepath.Join(root, "AGENTS.md"), maxAhoyFileBytes); aerr == nil && bytes.Equal(data, agents) {
		return toolFileRepeats, repeatsCopy
	}
	if imp, ok := strings.CutPrefix(trimmed, "@"); ok && !strings.ContainsAny(trimmed, "\r\n") && namesRootAgents(root, path, imp) {
		return toolFileRepeats, repeatsImport
	}
	return toolFileOwners, ""
}

// namesRootAgents reports whether ref, a link target or an import path written
// in the file at path, names the root's AGENTS.md: relative to the file's own
// folder, or absolute. The comparison is of paths only; nothing is followed.
func namesRootAgents(root, path, ref string) bool {
	if ref == "" {
		return false
	}
	p := filepath.FromSlash(ref)
	if !filepath.IsAbs(p) {
		p = filepath.Join(filepath.Dir(path), p)
	}
	p = filepath.Clean(p)
	if p == filepath.Join(root, "AGENTS.md") {
		return true
	}
	// An absolute target may name the root by its resolved path (a temporary
	// folder reached through a linked parent); the root's own resolution reads
	// no file of the project.
	if real, err := filepath.EvalSymlinks(root); err == nil && p == filepath.Join(real, "AGENTS.md") {
		return true
	}
	return false
}

// ownerFileWarning is the one line install prints first for a tool's own
// conventions file holding the owner's words.
func ownerFileWarning(f toolConventionsFile) string {
	return f.Rel + " holds your own words, so " + f.Tool + f.Qualifier + " reads it and not AGENTS.md: " +
		"abcd's rules stay hidden from " + f.Tool + " in this project until you move those words into AGENTS.md " +
		"and remove " + f.Rel + ". abcd never edits or removes this file."
}

// detectToolConventionsFiles raises, for each registry file at the root, the
// owner's-file warning or the retirement offer.
func detectToolConventionsFiles(root string) []Gap {
	var gaps []Gap
	for _, f := range toolConventionsFiles {
		switch class, how := classifyToolFile(root, f.Rel); class {
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
				Detail: f.Rel + " " + repeatsWhat[how] + ", and " + f.Tool + f.Qualifier + " reads it in place of AGENTS.md. " +
					"It holds nothing of your own, so it can go.",
				FixHint: "ahoy install at a terminal offers to remove it and removes it only on your answer; --yes never does. " +
					"Or remove it yourself.",
				Required: false, Resolvable: true,
			})
		}
	}
	return gaps
}

// installWarnings is the warning lines the gaps carry, in gap order.
func installWarnings(gaps []Gap) []string {
	out := []string{}
	for _, g := range gaps {
		if g.ID == ConventionsOwnerFileGapID {
			out = append(out, g.Detail)
		}
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

// conventionsRetireHelp is HelpFor's answer for a retirement question's key:
// the file and what it repeats first, the question last, then the answers
// with decide-later last. A key naming a file the registry does not hold, or a
// way of repeating AGENTS.md that does not exist, has no help.
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
		if f.Rel != rel {
			continue
		}
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

// stepConventionsFiles asks, of a person at a terminal, whether to retire
// each tool's own conventions file that only repeats AGENTS.md. It runs after
// stepDrainRule, the last consent question, and so after stepMarker, whose
// retraction may just have left a CLAUDE.md blank. It never runs under --yes,
// and never off a terminal.
func (a *applyCtx) stepConventionsFiles() {
	if a.autoYes || !atTerminal(a.prompter) || !a.approved[ConventionsFile] || !a.has(ConventionsRetireGapID) {
		return
	}
	for _, f := range toolConventionsFiles {
		class, how := classifyToolFile(a.cwd, f.Rel)
		if class != toolFileRepeats {
			continue
		}
		if a.prompter.Prompt(retirePromptKey(f.Rel, how), retireChoices, retireLater) != retireRetire {
			continue // keep and later write nothing and record nothing
		}
		// Classified again at the answer: a file that changed while the
		// question was open may now hold the owner's words, and is left.
		if again, _ := classifyToolFile(a.cwd, f.Rel); again != toolFileRepeats {
			a.refuse(f.Rel + " was not removed: it changed while the question was open and no longer only repeats AGENTS.md, so it is left as it is.")
			continue
		}
		path := filepath.Join(a.cwd, filepath.FromSlash(f.Rel))
		if err := os.Remove(path); err != nil {
			a.refuse("could not remove " + f.Rel + ": " + errText(err) + "; it is left as it is.")
			continue
		}
		a.note(writeToolFileRetired, path)
	}
}
