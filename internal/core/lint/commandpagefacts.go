package lint

import (
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/intentdriven/abcd/internal/core/mdrecord"
	"github.com/intentdriven/abcd/internal/core/surface"
	"github.com/intentdriven/abcd/internal/termsafe"
)

// commandpagefacts.go — the prose lint of the command pages
// (itd-2610301346085246, spc-2610301431075964 step 4, approach 5).
//
// The generated blocks of a command page are written from the command facts
// and gated by drift tests; the hand-written prose around them is not, and it
// is where a page names a flag that was renamed, a command that was removed or
// an input version the binary no longer reads. CommandPageFactsForTest reads that
// prose against the facts and reports each such claim with its page and line.
//
// It is a pure function: the page text and the facts are passed in, and
// nothing here reads a file, the clock or the environment. The facts are the
// whole tree, hidden commands included (the front door's CommandFacts), so a
// page that documents the hook plane is checked like any other.
//
// What the lint reads as an invocation of abcd is a line of a fenced block, or
// a backticked span (which may wrap onto the next line, as Markdown allows),
// whose text starts with one of the binary's spellings followed by a space or
// the end of the text. The spellings are the rungs a page names: the
// plugin-root copy in quotes, the bare name on PATH, the source checkout's
// wrapper script and `go run` of the source tree (commandPageBinaries). Text
// that only mentions abcd (a span holding the bare name, or a word such as
// abcd's) names no command beyond the root and is never reported, and a
// variable that holds a checkout's path is not one of the spellings.
//
// An invocation's command path is the longest run of words after the binary
// that names a command, stopping at the first word that is not a sub-command
// of the command reached so far. A word that stops the walk is an argument,
// except where the command reached has sub-commands, the word has the shape
// of a command name (lower-case letters joined by single hyphens, the shape
// every command of the tree has), and the command's usage line declares no
// positional argument that may be such a word (takesWordArgument): that is a
// sub-command the tree does not have. A record id, a tag, a placeholder, a
// quoted text or a path never has that shape, so the root's record-id
// argument is read as an argument while a misspelt verb after the root is
// reported, and `abcd mode managed` passes because mode takes a state word.
// The invocation ends at a shell operator outside quotes and
// brackets (a pipe, `;`, `&&`, `||`, a redirection or a comment), so a flag of
// the next command in a pipeline is never read as abcd's, and its flags end at
// a bare `--`, after which every word is passed through.

// CommandPageHistorical marks a line whose claims the lint does not check: a
// line that names an old version, flag or command on purpose, such as a page
// saying that an input's version 1 is refused.
const CommandPageHistorical = "<!-- facts-lint: historical -->"

// The markers fencing every generated block of a page. They are the front
// door's markers (internal/surface/cli, GeneratedBeginPrefix and
// GeneratedEnd), which a test there holds equal to these.
const (
	GeneratedBlockBegin = "<!-- generated:"
	GeneratedBlockEnd   = "<!-- /generated -->"
)

// commandPageBinaries are the spellings of the binary a command page invokes
// it by, longest first so the wrapper script is not read as a bare name.
var commandPageBinaries = []string{
	`"${CLAUDE_PLUGIN_ROOT}/abcd"`,
	"go run ./cmd/abcd",
	"./scripts/abcd.sh",
	"scripts/abcd.sh",
	"abcd",
}

// CommandPageFinding is one claim of a page's prose that the facts do not hold.
type CommandPageFinding struct {
	// Page is the page as the caller named it.
	Page string
	// Line is the 1-based line the claim starts on.
	Line int
	// Message says what the claim names and what the facts hold instead.
	Message string
}

// String is the finding as a test or a report prints it: page, line, message.
func (f CommandPageFinding) String() string {
	return fmt.Sprintf("%s:%d: %s", f.Page, f.Line, f.Message)
}

var (
	// commandWordRE is the shape of every command name of the tree.
	commandWordRE = regexp.MustCompile(`^[a-z]+(-[a-z]+)*$`)
	// flagNameRE is the shape of a long flag's name, after its dashes.
	flagNameRE = regexp.MustCompile(`^[a-z0-9][a-z0-9-]*$`)
	// schemaVersionRE is a JSON schema_version member with an integer value.
	schemaVersionRE = regexp.MustCompile(`"schema_version"\s*:\s*(\d+)`)
	// longFlagRE is a long flag named in prose.
	longFlagRE = regexp.MustCompile(`--([a-z0-9][a-z0-9-]*)`)
)

// commandIndex is the facts arranged for the lint: each command by path,
// which paths have sub-commands, the root's path, and the flags any command
// reads a filled-in input through.
type commandIndex struct {
	root       string
	byPath     map[string]surface.Facts
	hasChild   map[string]bool
	inputFlags map[string]bool
}

func newCommandIndex(facts []surface.Facts) commandIndex {
	ix := commandIndex{byPath: map[string]surface.Facts{}, hasChild: map[string]bool{}, inputFlags: map[string]bool{}}
	for _, f := range facts {
		ix.byPath[f.Path] = f
		for _, in := range f.Inputs {
			ix.inputFlags[in.Flag] = true
		}
		if parent, _, ok := cutLastWord(f.Path); ok {
			ix.hasChild[parent] = true
		} else {
			ix.root = f.Path
		}
	}
	return ix
}

// cutLastWord splits a command path into its parent's path and its last word.
func cutLastWord(path string) (string, string, bool) {
	i := strings.LastIndexByte(path, ' ')
	if i < 0 {
		return "", path, false
	}
	return path[:i], path[i+1:], true
}

// flagRegistered reports whether a command registers the flag, or a command on
// its path registers it as persistent. The help flag is the flag library's
// own, on every command, and is not among the facts.
func (ix commandIndex) flagRegistered(path, name string) bool {
	if name == "help" {
		return true
	}
	if f, ok := ix.byPath[path]; ok {
		for _, fl := range f.Flags {
			if fl.Name == name {
				return true
			}
		}
	}
	for p, ok := cutParent(path); ok; p, ok = cutParent(p) {
		for _, fl := range ix.byPath[p].Flags {
			if fl.Persistent && fl.Name == name {
				return true
			}
		}
	}
	return false
}

func cutParent(path string) (string, bool) {
	parent, _, ok := cutLastWord(path)
	return parent, ok
}

// inputVersions is the schema versions a document carrying a top-level
// schema_version may hold for the input read through flag: by the command at
// path when path is given and reads it, and otherwise by the commands of the
// page's own subtree that read it, or failing those by every command that
// does. It is empty when no command reads the flag as such an input.
func (ix commandIndex) inputVersions(flag, path, pageRoot string) (map[int]bool, []string) {
	collect := func(keep func(string) bool) (map[int]bool, []string) {
		versions := map[int]bool{}
		var readers []string
		for _, p := range sortedPaths(ix.byPath) {
			if !keep(p) {
				continue
			}
			for _, in := range ix.byPath[p].Inputs {
				if in.Flag == flag && in.Kind == surface.InputJSON && in.VersionIn == surface.VersionField {
					versions[in.SchemaVersion] = true
					readers = append(readers, p)
				}
			}
		}
		return versions, readers
	}
	if path != "" {
		if v, r := collect(func(p string) bool { return p == path }); len(v) > 0 {
			return v, r
		}
	}
	if pageRoot != "" {
		if v, r := collect(func(p string) bool { return p == pageRoot || strings.HasPrefix(p, pageRoot+" ") }); len(v) > 0 {
			return v, r
		}
	}
	return collect(func(string) bool { return true })
}

func sortedPaths(m map[string]surface.Facts) []string {
	out := make([]string, 0, len(m))
	for p := range m {
		out = append(out, p)
	}
	sort.Strings(out)
	return out
}

// pageText is one stretch of a page's prose the lint reads: a fenced line
// (with its continuation lines joined) or a code span, with the lines it
// starts and ends on.
type pageText struct {
	line   int
	last   int
	text   string
	exempt bool
}

// pageInvocation is one invocation of abcd on a page: the words after the
// binary's spelling, with the lines the text holding it starts and ends on.
type pageInvocation struct {
	line   int
	last   int
	exempt bool
	rest   string
}

// pageInvocations is every invocation of abcd in the prose, in page order: each
// command of a fenced line or a code span (shellCommands) that starts with one
// of the binary's spellings.
func pageInvocations(lines []string, fences []mdrecord.Span) []pageInvocation {
	var out []pageInvocation
	for _, t := range pageTexts(lines, fences) {
		for _, cmd := range shellCommands(t.text) {
			if rest, ok := afterBinary(cmd); ok {
				out = append(out, pageInvocation{line: t.line, last: t.last, exempt: t.exempt, rest: rest})
			}
		}
	}
	return out
}

// anchor is what a "schema_version" member may stand next to: an invocation
// of abcd, with the command it resolved to and the flags it passes, or a
// mention in prose of an input flag, which names no command.
type anchor struct {
	line    int
	section int
	command string
	flags   []string
}

// CommandPageFactsForTest lints one command page's hand-written prose against the
// command facts: page is the page as findings name it (its basename less
// ".md" is the verb it documents, or the root's own name for the dispatcher
// page), text is its content, and facts is the whole command tree, hidden
// commands included. It reports, in line order, each invocation whose command
// path the tree lacks, each long flag in an invocation that neither the
// command nor a persistent flag on its path registers, and each
// "schema_version" member whose value is not a version the input beside it
// reads (nextTo says which input that is). Generated blocks are skipped, and
// a line carrying CommandPageHistorical is exempt.
//
// Its one caller is the test in internal/surface/cli that runs it over the
// committed pages with the real command tree, which this package cannot
// import, so it carries the cross-package test seam's name.
func CommandPageFactsForTest(page, text string, facts []surface.Facts) []CommandPageFinding {
	ix := newCommandIndex(facts)
	if ix.root == "" {
		return nil
	}
	lines := strings.Split(text, "\n")
	blankGeneratedBlocks(lines)
	pageRoot := pageRootPath(ix, page)
	reading := mdrecord.Read(lines, pageFenceRule)
	sections := sectionsOf(lines, reading.Mask)

	var findings []CommandPageFinding
	report := func(line int, format string, args ...any) {
		findings = append(findings, CommandPageFinding{Page: page, Line: line, Message: fmt.Sprintf(format, args...)})
	}

	var anchors []anchor
	// inInvocation is, per line, the flags an invocation spanning the line
	// passes. Every line of a call continued over several lines, or of a
	// span wrapped onto the next, is marked, so a flag there anchors with its
	// command.
	inInvocation := map[int]map[string]bool{}
	for _, t := range pageInvocations(lines, reading.Fences) {
		inv := parseInvocation(ix, t.rest)
		if inv.unknown != "" {
			if !t.exempt {
				report(t.line, "%s names no command the tree has: %s has no sub-command %s",
					quoteCode(ix.root+" "+strings.Join(append(inv.pathWords, inv.unknown), " ")),
					quoteCode(inv.path), quoteCode(inv.unknown))
			}
			continue
		}
		for _, fl := range inv.flags {
			if !t.exempt && !ix.flagRegistered(inv.path, fl) {
				report(t.line, "%s is not a flag of %s, nor a persistent flag on its path",
					quoteCode("--"+fl), quoteCode(inv.path))
			}
			for ln := t.line; ln <= t.last; ln++ {
				if inInvocation[ln] == nil {
					inInvocation[ln] = map[string]bool{}
				}
				inInvocation[ln][fl] = true
			}
		}
		// A span holding only the binary's name, or the root with an argument
		// and no flag, names no command a document is passed to, and is no
		// anchor, since it would hide the real call before it.
		if len(inv.pathWords) == 0 && len(inv.flags) == 0 {
			continue
		}
		anchors = append(anchors, anchor{line: t.line, section: sections[t.line-1], command: inv.path, flags: inv.flags})
	}
	// A mention of an input flag outside an invocation is an anchor naming no
	// command.
	for i, l := range lines {
		for _, m := range longFlagRE.FindAllStringSubmatch(l, -1) {
			if ix.inputFlags[m[1]] && !inInvocation[i+1][m[1]] {
				anchors = append(anchors, anchor{line: i + 1, section: sections[i], flags: []string{m[1]}})
			}
		}
	}
	sort.SliceStable(anchors, func(a, b int) bool { return anchors[a].line < anchors[b].line })

	for i, l := range lines {
		if strings.Contains(l, CommandPageHistorical) {
			continue
		}
		for _, m := range schemaVersionRE.FindAllStringSubmatch(l, -1) {
			n, err := strconv.Atoi(m[1])
			if err != nil {
				continue
			}
			near, ok := nextTo(anchors, i+1, sections[i])
			if !ok {
				continue
			}
			versions := map[int]bool{}
			var named, readers []string
			for _, fl := range near.flags {
				if !ix.inputFlags[fl] {
					continue
				}
				v, r := ix.inputVersions(fl, near.command, pageRoot)
				if len(v) == 0 {
					continue
				}
				named = append(named, quoteCode("--"+fl))
				readers = append(readers, r...)
				for k := range v {
					versions[k] = true
				}
			}
			if len(versions) == 0 || versions[n] {
				continue
			}
			report(i+1, "%s stands next to %s (line %d), which %s reads at %s",
				quoteCode(m[0]), strings.Join(named, " and "), near.line,
				joinCode(readers), versionList(versions))
		}
	}

	sort.SliceStable(findings, func(a, b int) bool { return findings[a].Line < findings[b].Line })
	return findings
}

// nextTo is the anchor a "schema_version" member on line, in section, stands
// next to: the nearest anchor at or before it in the same section, which is
// how a page introduces a document (the call, then the document it reads), or
// failing any, the nearest one after it in the same section. A member whose
// nearest anchor is a call passing no input flag is next to no input, such as
// a receipt file another flag reads, and is not checked.
func nextTo(anchors []anchor, line, section int) (anchor, bool) {
	best, found := anchor{}, false
	for _, a := range anchors {
		if a.section == section && a.line <= line {
			best, found = a, true
		}
	}
	if found {
		return best, true
	}
	for _, a := range anchors {
		if a.section == section && a.line > line {
			return a, true
		}
	}
	return anchor{}, false
}

// sectionsOf numbers each line by the section it stands in: the count of
// headings (an ATX heading line outside a fence, mask being mdrecord's
// reading of the page) at or before it.
func sectionsOf(lines []string, mask []uint8) []int {
	out := make([]int, len(lines))
	section := 0
	for i, l := range lines {
		if mask[i]&mdrecord.MaskFence == 0 && mdrecord.IsHeading(l) {
			section++
		}
		out[i] = section
	}
	return out
}

// pageRootPath is the command path of the page's subtree: the root for the
// dispatcher page, the verb's path for a verb page, or empty when the tree has
// no command of the page's name.
func pageRootPath(ix commandIndex, page string) string {
	name := page
	if i := strings.LastIndexByte(name, '/'); i >= 0 {
		name = name[i+1:]
	}
	name = strings.TrimSuffix(name, ".md")
	if name == ix.root {
		return ix.root
	}
	if _, ok := ix.byPath[ix.root+" "+name]; ok {
		return ix.root + " " + name
	}
	return ""
}

// blankGeneratedBlocks empties every line of every generated block, its
// markers included, keeping the line count so findings keep their lines. A
// block left open runs to the end of the page.
func blankGeneratedBlocks(lines []string) {
	in := false
	for i, l := range lines {
		begins := strings.Contains(l, GeneratedBlockBegin)
		if begins {
			in = true
		}
		if in {
			lines[i] = ""
			if strings.Contains(l, GeneratedBlockEnd) {
				in = false
			}
		}
	}
}

// pageFenceRule is the fence rule the lint reads a page by: the site's, under
// which a fence indented under a list item is a fence, as the command pages
// write them in their numbered steps.
const pageFenceRule = mdrecord.ListNested

// pageTexts is every fenced line and every code span of the prose, in page
// order. Which lines a fence holds is mdrecord's reading (fences, Read under
// pageFenceRule), so the lint and sectionsOf agree on where every fence
// opens and closes. A fenced line ending in a backslash is joined with the
// next line of its fence; the closing fence, or a delimiter-shaped line
// inside the fence, ends the joined command and is not read. A code span may
// wrap onto following lines of its paragraph; it is read with its line breaks
// as spaces and reported on the line it opens on.
func pageTexts(lines []string, fences []mdrecord.Span) []pageText {
	var out []pageText
	var para []int // indexes of the lines of the current paragraph
	flush := func() {
		out = append(out, spansOf(lines, para)...)
		para = para[:0]
	}
	delimiter := func(i int) bool {
		return mdrecord.OpensFence(strings.TrimLeft(lines[i], " \t"), mdrecord.TopLevel)
	}
	next := 0 // the first fence not yet read
	for i := 0; i < len(lines); i++ {
		if next < len(fences) && i == fences[next].Start {
			flush()
			end := fences[next].End
			next++
			for j := i + 1; j < end; j++ {
				if delimiter(j) {
					continue
				}
				start := j
				joined := strings.TrimLeft(lines[j], " \t")
				exempt := strings.Contains(lines[j], CommandPageHistorical)
				for strings.HasSuffix(joined, `\`) && j+1 < end && !delimiter(j+1) {
					j++
					exempt = exempt || strings.Contains(lines[j], CommandPageHistorical)
					joined = strings.TrimSuffix(joined, `\`) + " " + strings.TrimLeft(lines[j], " \t")
				}
				out = append(out, pageText{line: start + 1, last: j + 1, text: joined, exempt: exempt})
			}
			i = end - 1
			continue
		}
		if strings.TrimSpace(lines[i]) == "" {
			flush()
			continue
		}
		para = append(para, i)
	}
	flush()
	return out
}

// spansOf is the code spans of one paragraph, the lines whose indexes are
// given. The spans are the tree's own pairing (mdrecord.CodeSpanRanges, by
// termsafe.PairCodeSpan's rule) over the paragraph joined with its line
// breaks, which that rule reads as span content, so a span may wrap; an
// unclosed run is literal text, and a backtick behind a backslash opens
// nothing. A span's text is what it renders as (termsafe.CodeSpanText).
func spansOf(lines []string, para []int) []pageText {
	if len(para) == 0 {
		return nil
	}
	// The paragraph as one text, and the line each byte stands on.
	var b strings.Builder
	var lineOf []int
	for k, i := range para {
		if k > 0 {
			b.WriteByte('\n')
			lineOf = append(lineOf, i)
		}
		b.WriteString(lines[i])
		for range len(lines[i]) {
			lineOf = append(lineOf, i)
		}
	}
	text := b.String()
	var out []pageText
	for _, r := range mdrecord.CodeSpanRanges(text) {
		sp, ok := termsafe.PairCodeSpan(text, r[0])
		if !ok {
			continue
		}
		first, last := lineOf[sp.Start], lineOf[sp.End-1]
		exempt := false
		for i := first; i <= last; i++ {
			exempt = exempt || strings.Contains(lines[i], CommandPageHistorical)
		}
		out = append(out, pageText{line: first + 1, last: last + 1, text: termsafe.CodeSpanText(sp.Raw(text)), exempt: exempt})
	}
	return out
}

// afterBinary is the text after the binary's spelling when the text starts
// with one followed by a space or the end of the text.
func afterBinary(text string) (string, bool) {
	text = strings.TrimSpace(text)
	for _, bin := range commandPageBinaries {
		rest, ok := strings.CutPrefix(text, bin)
		if !ok {
			continue
		}
		if rest == "" || rest[0] == ' ' || rest[0] == '\t' {
			return rest, true
		}
		return "", false
	}
	return "", false
}

// invocation is what the lint reads from one invocation of abcd.
type invocation struct {
	// path is the command the words resolve to, the root's path when none.
	path string
	// pathWords are the words after the binary that name path.
	pathWords []string
	// unknown is the command-shaped word the walk stopped at under a
	// command with sub-commands, or empty.
	unknown string
	// flags are the long flags' names, without their dashes or values.
	flags []string
}

// parseInvocation reads the words after the binary.
func parseInvocation(ix commandIndex, rest string) invocation {
	words := shellWords(rest)
	inv := invocation{path: ix.root}
	i := 0
	for ; i < len(words); i++ {
		w := words[i]
		child := inv.path + " " + w
		if _, ok := ix.byPath[child]; ok && commandWordRE.MatchString(w) {
			inv.path = child
			inv.pathWords = append(inv.pathWords, w)
			continue
		}
		if commandWordRE.MatchString(w) && ix.hasChild[inv.path] && !takesWordArgument(ix.byPath[inv.path]) {
			inv.unknown = w
		}
		break
	}
	for _, w := range words {
		if w == "--" {
			break
		}
		if name, ok := longFlag(w); ok {
			inv.flags = append(inv.flags, name)
		}
	}
	return inv
}

// takesWordArgument reports whether a command's usage line declares a
// positional argument that may be a word with a command name's shape, such as
// `abcd mode [<state>]`, whose states are words like `managed`. A placeholder
// naming an id or a tag (`<record-id>`, `<itd-N|iss-N>`, `<release-tag>`) or
// free text (`[text]`) takes no such word: an id and a tag carry digits, and
// the commands reading text refuse a lone word, and a word before a record
// id, as a misspelt sub-command. The usage line is read up to its first
// alternative form (a bare `|`), and the placeholder after a flag is that
// flag's value, not a positional argument.
func takesWordArgument(f surface.Facts) bool {
	rest, ok := strings.CutPrefix(f.Usage, f.Path)
	if !ok {
		return false
	}
	afterFlag := false
	for _, tok := range strings.Fields(rest) {
		if tok == "|" {
			break
		}
		bare := strings.TrimLeft(tok, "[(")
		if strings.HasPrefix(bare, "-") {
			afterFlag = !strings.HasSuffix(tok, "]") && !strings.HasSuffix(tok, ")")
			continue
		}
		if afterFlag {
			afterFlag = false
			continue
		}
		name := strings.Trim(bare, "[]()<>\"'.…")
		if name == "flags" || name == "command" || name == "" {
			continue
		}
		for _, alt := range strings.Split(name, "|") {
			alt = strings.ToLower(strings.Trim(alt, "<>\"'"))
			if alt == "text" || strings.HasSuffix(alt, "id") || strings.HasSuffix(alt, "-n") || strings.HasSuffix(alt, "tag") {
				continue
			}
			return true
		}
	}
	return false
}

// longFlag is the name of the long flag a word spells, without its dashes and
// any value: a usage line's brackets, parentheses and a trailing comma or
// full stop are not part of it, and a placeholder such as --<flag> is no flag.
func longFlag(w string) (string, bool) {
	w = strings.TrimLeft(w, "[(")
	if !strings.HasPrefix(w, "--") || len(w) == 2 {
		return "", false
	}
	name := w[2:]
	if i := strings.IndexByte(name, '='); i >= 0 {
		name = name[:i]
	}
	name = strings.TrimRight(name, "])},.;:")
	if !flagNameRE.MatchString(name) {
		return "", false
	}
	return name, true
}

// shellWords splits an invocation into its words the way a shell would read
// them for the lint's purpose: whitespace separates words outside quotes, a
// quoted run stays one word with its quotes, and the words end at the first
// shell operator outside quotes and brackets (a pipe, `;`, `&`, a
// redirection, a comment or a command substitution). A usage line's
// alternatives, `[--private | --public]`, and a placeholder's,
// `<itd-N|iss-N>`, are inside brackets, so their bar is no pipe.
func shellWords(s string) []string {
	words, _, _ := scanWords(s)
	return words
}

// shellCommands is the text of each command a line runs: the line split at
// each pipe, `;`, `&&` and `||` outside quotes and brackets, up to a
// redirection, a comment or a command substitution, after which nothing on
// the line is read. So `printf %s x | abcd banlist add` holds an invocation of
// abcd as its second command.
func shellCommands(s string) []string {
	var out []string
	for {
		_, stop, chained := scanWords(s)
		out = append(out, s[:stop])
		if !chained {
			return out
		}
		s = strings.TrimLeft(s[stop:], "|;& \t")
	}
}

// opensPlaceholder reports whether the `<` at s[i] opens a placeholder, such
// as <record-id> or <full commit id>, rather than an input redirection. A
// placeholder is closed by a later `>` that directly follows a character of
// its text. A `>` after a blank is an output redirection (`<in.json
// >out.json`), and so is one after a lone file-descriptor number naming its
// target (`<in.json 2>/dev/null`); a number closed by `>` and a blank, as in
// <chunk 1>, ends a placeholder's text. A `<` followed by a blank, `<`, `>`,
// `&` or `(` is a redirection, a here-document or a process substitution, and
// `<word` with no `>` after it is a redirection, as the shell reads it.
func opensPlaceholder(s string, i int) bool {
	if i+1 >= len(s) || strings.IndexByte(" \t<>&(", s[i+1]) >= 0 {
		return false
	}
	j := strings.IndexByte(s[i+1:], '>')
	if j < 0 {
		return false
	}
	// Step back over the digits directly before the `>`. What stands before
	// them, or before the `>` when there are none, says which it is.
	gt := i + 1 + j
	k := gt
	for k > i+1 && s[k-1] >= '0' && s[k-1] <= '9' {
		k--
	}
	if s[k-1] != ' ' && s[k-1] != '\t' {
		return true
	}
	// A blank before the digits makes them a file-descriptor number only when
	// the redirection names its target straight after the `>` (`2>/dev/null`,
	// `2>&1`); a blank or the end of the text there closes a placeholder
	// whose text ends in a number, such as <chunk 1>.
	return k < gt && (gt+1 >= len(s) || s[gt+1] == ' ' || s[gt+1] == '\t')
}

// scanWords is the words of the command s opens, the offset its text ends at,
// and whether a further command may follow there (the command ended at a
// pipe, `;` or `&`, not at a redirection, a comment, a substitution or the end
// of the text).
func scanWords(s string) ([]string, int, bool) {
	var words []string
	var cur strings.Builder
	have := false
	depth := 0
	endWord := func() {
		if have {
			words = append(words, cur.String())
		}
		cur.Reset()
		have = false
	}
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch {
		case c == '\'' || c == '"':
			j := i + 1
			for j < len(s) && s[j] != c {
				if c == '"' && s[j] == '\\' {
					j++
				}
				j++
			}
			if j >= len(s) {
				j = len(s) - 1
			}
			cur.WriteString(s[i : j+1])
			have = true
			i = j
		case c == ' ' || c == '\t':
			endWord()
		case depth == 0 && (c == '|' || c == ';' || c == '&'):
			endWord()
			return words, i, true
		case depth == 0 && (c == '>' || c == '`'):
			endWord()
			return words, i, false
		case c == '<' && !opensPlaceholder(s, i):
			// An input redirection, a here-document or a process
			// substitution, not a placeholder.
			endWord()
			return words, i, false
		case c == '#' && !have:
			endWord()
			return words, i, false
		case c == '$' && i+1 < len(s) && s[i+1] == '(':
			endWord()
			return words, i, false
		default:
			// A placeholder's angle brackets nest like a usage line's
			// brackets, so <itd-N|iss-N> holds no pipe and its closing
			// bracket is no redirection.
			switch c {
			case '[', '(', '<':
				depth++
			case ']', ')', '>':
				if depth > 0 {
					depth--
				}
			}
			cur.WriteByte(c)
			have = true
		}
	}
	endWord()
	return words, len(s), false
}

func quoteCode(s string) string { return "`" + s + "`" }

func joinCode(paths []string) string {
	q := make([]string, len(paths))
	for i, p := range paths {
		q[i] = quoteCode(p)
	}
	return strings.Join(q, " and ")
}

func versionList(versions map[int]bool) string {
	var vs []int
	for v := range versions {
		vs = append(vs, v)
	}
	sort.Ints(vs)
	parts := make([]string, len(vs))
	for i, v := range vs {
		parts[i] = "schema_version " + strconv.Itoa(v)
	}
	return strings.Join(parts, " or ")
}
