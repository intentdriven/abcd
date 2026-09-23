package surface

// The brief's surface chapters carry a generated appendix (itd-147,
// spc-2609020906356450). A chapter mixes two kinds of content with two failure
// modes: SHAPE — flags and sub-verbs — is derivable from the command tree, so it
// is generated here and never hand-written; INTENT — why a surface exists, what
// it refuses, which trade was made — is not derivable and stays prose. The
// marker pair is the physical form of that cut: everything above the opening
// marker is hand-written and never touched, everything between the markers is
// machine-written and never hand-edited.
//
// The appendix is composed from the same []Command the compatibility snapshot
// is built from, so the repository holds one walk of the command tree and not a
// second one that can disagree with it. The walk itself needs cobra and lives in
// internal/surface/cli; everything here is pure over its result.

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

// BriefSurfacesDir is the brief's surface-chapter directory, repo-relative and
// slash-separated. Its README.md is the surfaces register: the table whose rows
// map each user-facing command to the chapter documenting it.
const BriefSurfacesDir = ".abcd/development/brief/04-surfaces"

// RegisterFile is the register's basename inside BriefSurfacesDir.
const RegisterFile = "README.md"

// AppendixBegin and AppendixEnd delimit the generated region. Each must stand
// alone on its line. The generator refuses a chapter whose markers are absent,
// crossed, duplicated, misspelt, fenced, or followed by more prose, rather than
// guessing where the region is.
const (
	AppendixBegin = "<!-- surface-appendix:begin — generated from the command tree by `go generate ./internal/surface/cli`; never edit by hand -->"
	AppendixEnd   = "<!-- surface-appendix:end -->"
	markerPrefix  = "<!-- surface-appendix:"
)

// appendixHeading and appendixPreamble open every appendix that lists at least
// one shipped command.
const (
	appendixHeading  = "## Appendix: the shipped surface"
	appendixPreamble = "_Generated from the command tree; a drift test fails the build when this appendix and the tree disagree. " +
		"It lists flags and sub-verbs only. What each flag means is in the " +
		"[CLI reference](../../../../docs/reference/cli/commands.md), and exit codes, output fields and behaviour are the prose's to state._"
)

// The refusals a malformed chapter earns. Each is wrapped with the chapter's
// name and the offending line, so a failure points at the file to fix.
var (
	ErrMarkerAbsent      = errors.New("the surface appendix markers are absent")
	ErrMarkerCrossed     = errors.New("the surface appendix end marker precedes its begin marker")
	ErrMarkerDuplicated  = errors.New("a surface appendix marker appears more than once")
	ErrMarkerInFence     = errors.New("a surface appendix marker sits inside a fenced code block")
	ErrMarkerMalformed   = errors.New("a surface appendix marker is misspelt")
	ErrMarkerNotAtEnd    = errors.New("prose follows the surface appendix end marker; the appendix sits at the end of the chapter")
	ErrChapterWithoutRow = errors.New("a surface chapter has no row in the surfaces register")
	ErrRowWithoutChapter = errors.New("a surfaces register row names a chapter that does not exist")
)

// UnbuiltSentence is the whole appendix of a chapter whose surface the command
// tree does not register: a staged design target, or a host-delegated command
// with no Go verb. Every chapter carries a block, so a reader learns the surface
// is unbuilt from the place they would have read its flags, never from absence.
func UnbuiltSentence(path string) string {
	return "There is no shipped surface: the command tree registers no `" + path +
		"` verb, so there are no flags and no sub-verbs to list."
}

// ComposeAppendix renders the generated region for a chapter documenting the
// given command paths, in the order given (the register's order). Each shipped
// command is listed with every descendant, depth-first by path, each with its
// direct sub-verbs and its own flags — flags a command declares, never those it
// inherits, so a persistent flag appears once, where it is declared. The bare
// root is the exception: its children are verbs with chapters of their own, so
// its section lists its flags alone.
//
// The inputs are the path, the flags and the sub-verbs, and nothing else, so an
// exit code or an output field cannot reach the block until the tree records it
// somewhere this function is handed.
func ComposeAppendix(paths []string, tree []Command) string {
	byPath := make(map[string]Command, len(tree))
	for _, c := range tree {
		byPath[c.Path] = c
	}
	shipped := false
	for _, p := range paths {
		if _, ok := byPath[p]; ok {
			shipped = true
		}
	}
	var b strings.Builder
	b.WriteString("\n")
	if !shipped {
		for _, p := range paths {
			b.WriteString(UnbuiltSentence(p) + "\n\n")
		}
		return b.String()
	}
	b.WriteString(appendixHeading + "\n\n" + appendixPreamble + "\n\n")
	for _, p := range paths {
		if _, ok := byPath[p]; !ok {
			fmt.Fprintf(&b, "### `%s`\n\n%s\n\n", p, UnbuiltSentence(p))
			continue
		}
		for _, c := range subtree(p, tree) {
			writeCommandSection(&b, c, tree, isRoot(p))
		}
	}
	return b.String()
}

func isRoot(path string) bool { return !strings.Contains(path, " ") }

// subtree returns the command at path and every descendant, sorted by path. The
// bare root's subtree is the root alone.
func subtree(path string, tree []Command) []Command {
	var out []Command
	for _, c := range tree {
		if c.Path == path || (!isRoot(path) && strings.HasPrefix(c.Path, path+" ")) {
			out = append(out, c)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Path < out[j].Path })
	return out
}

// children returns the direct sub-commands of path, sorted.
func children(path string, tree []Command) []string {
	var out []string
	for _, c := range tree {
		if strings.HasPrefix(c.Path, path+" ") && !strings.Contains(c.Path[len(path)+1:], " ") {
			out = append(out, c.Path)
		}
	}
	sort.Strings(out)
	return out
}

func writeCommandSection(b *strings.Builder, c Command, tree []Command, root bool) {
	fmt.Fprintf(b, "### `%s`\n\n", c.Path)
	if !root {
		kids := children(c.Path, tree)
		if len(kids) == 0 {
			b.WriteString("Sub-verbs: none.\n\n")
		} else {
			quoted := make([]string, len(kids))
			for i, k := range kids {
				quoted[i] = "`" + k + "`"
			}
			b.WriteString("Sub-verbs: " + strings.Join(quoted, ", ") + ".\n\n")
		}
	}
	if len(c.Flags) == 0 {
		b.WriteString("Flags: none.\n\n")
		return
	}
	flags := append([]Flag(nil), c.Flags...)
	sort.Slice(flags, func(i, j int) bool { return flags[i].Name < flags[j].Name })
	b.WriteString("| Flag | Type |\n|---|---|\n")
	for _, f := range flags {
		cell := "`--" + f.Name + "`"
		if f.Shorthand != "" {
			cell += ", `-" + f.Shorthand + "`"
		}
		if f.Required {
			cell += " (required)"
		}
		if f.Hidden {
			cell += " (hidden)"
		}
		fmt.Fprintf(b, "| %s | %s |\n", cell, f.Type)
	}
	b.WriteString("\n")
}

// markerLines locates the two markers, refusing every shape that would make the
// region ambiguous. begin and end are 0-based line indices.
func markerLines(lines []string) (begin, end int, err error) {
	begin, end = -1, -1
	fenced := false
	for i, line := range lines {
		t := strings.TrimSpace(line)
		if strings.HasPrefix(t, "```") || strings.HasPrefix(t, "~~~") {
			fenced = !fenced
			continue
		}
		if !strings.HasPrefix(t, markerPrefix) {
			continue
		}
		if fenced {
			return -1, -1, fmt.Errorf("line %d: %w", i+1, ErrMarkerInFence)
		}
		switch t {
		case AppendixBegin:
			if begin >= 0 {
				return -1, -1, fmt.Errorf("line %d: %w", i+1, ErrMarkerDuplicated)
			}
			begin = i
		case AppendixEnd:
			if end >= 0 {
				return -1, -1, fmt.Errorf("line %d: %w", i+1, ErrMarkerDuplicated)
			}
			end = i
		default:
			return -1, -1, fmt.Errorf("line %d: %w (want %q and %q, each alone on its line)", i+1, ErrMarkerMalformed, AppendixBegin, AppendixEnd)
		}
	}
	switch {
	case begin < 0 || end < 0:
		return -1, -1, fmt.Errorf("%w: end the chapter with the line %q and, after it, the line %q, then regenerate", ErrMarkerAbsent, AppendixBegin, AppendixEnd)
	case end < begin:
		return -1, -1, fmt.Errorf("line %d: %w", end+1, ErrMarkerCrossed)
	}
	for i := end + 1; i < len(lines); i++ {
		if strings.TrimSpace(lines[i]) != "" {
			return -1, -1, fmt.Errorf("line %d: %w", i+1, ErrMarkerNotAtEnd)
		}
	}
	return begin, end, nil
}

// SplitChapter returns the hand-written prose above the opening marker, byte for
// byte, or the refusal that names why the region cannot be located.
func SplitChapter(text string) (string, error) {
	lines := strings.Split(text, "\n")
	begin, _, err := markerLines(lines)
	if err != nil {
		return "", err
	}
	return strings.Join(lines[:begin], "\n") + "\n", nil
}

// RenderChapter replaces the bytes between the markers with appendix and leaves
// every byte above the opening marker as it was. The chapter ends with the
// closing marker and one newline.
func RenderChapter(text, appendix string) (string, error) {
	prose, err := SplitChapter(text)
	if err != nil {
		return "", err
	}
	return prose + AppendixBegin + "\n" + appendix + AppendixEnd + "\n", nil
}

// DriftLines compares a committed chapter with its regenerated form line by line
// as multisets: missing holds the lines the regenerated chapter has and the
// committed one lacks (a claim the tree makes that the chapter does not), stale
// the reverse (a claim the chapter makes that the tree no longer does).
func DriftLines(committed, regenerated string) (missing, stale []string) {
	count := map[string]int{}
	for _, l := range strings.Split(committed, "\n") {
		count[l]++
	}
	for _, l := range strings.Split(regenerated, "\n") {
		if count[l] > 0 {
			count[l]--
			continue
		}
		missing = append(missing, l)
	}
	want := map[string]int{}
	for _, l := range strings.Split(regenerated, "\n") {
		want[l]++
	}
	for _, l := range strings.Split(committed, "\n") {
		if want[l] > 0 {
			want[l]--
			continue
		}
		stale = append(stale, l)
	}
	return missing, stale
}

// ShapeClaim is one flag or sub-verb stated in a chapter's hand-written prose.
type ShapeClaim struct {
	Line     int    // 1-based line in the chapter
	Spelling string // the flag, the command path, or the backticked sub-verb name
}

var (
	flagSpelling     = regexp.MustCompile(`(^|[^\w-])(--[a-z][a-z0-9-]*)`)
	subVerbHeadingRe = regexp.MustCompile(`^##\s+Sub-verbs\s*$`)
	headingRe        = regexp.MustCompile(`^#{1,6}\s`)
)

// ProseShapeClaims reports every flag and sub-verb the prose states. own is the
// chapter's command paths. Three spellings count as a claim:
//
//   - a flag spelling (`--name`), anywhere, fenced examples included;
//   - a sub-verb's command path below its top-level verb (`capture list`, with
//     or without an `abcd ` or `/abcd:` prefix), for every sub-verb the tree
//     registers;
//   - a backticked sub-verb name below one of the chapter's own verbs
//     (“ `list` “ in the capture chapter).
//
// The `## Sub-verbs` section's table and its standard blockquote note are
// exempt: the table is compared against the command-tree snapshot by
// surface_coverage and carries the adr-40 bucket, which the tree does not
// record, so it is a checked register rather than a hand-written shape claim;
// the note names the bucket vocabulary, some of whose words are also sub-verb
// names. Any other prose in that section is checked like the rest. A plain word
// ("listing", "the cut") is not a spelling and is how prose refers to a
// behaviour.
func ProseShapeClaims(prose string, own []string, tree []Command) []ShapeClaim {
	var paths []string // sub-verb command paths without the root word
	for _, c := range tree {
		words := strings.Fields(c.Path)
		if len(words) >= 3 {
			paths = append(paths, strings.Join(words[1:], " "))
		}
	}
	// Longest first, so a nested path is reported before the prefix it contains.
	sort.SliceStable(paths, func(i, j int) bool { return len(paths[i]) > len(paths[j]) })

	var bare []string
	for _, o := range own {
		if isRoot(o) {
			continue
		}
		for _, c := range tree {
			if strings.HasPrefix(c.Path, o+" ") {
				bare = append(bare, c.Path[len(o)+1:])
			}
		}
	}
	sort.Strings(bare)

	// The exempt lines are blanked rather than removed, so line numbers hold and
	// a spelling cannot bridge from checked prose into an exempt line.
	lines := strings.Split(prose, "\n")
	inSubVerbs := false
	for i, line := range lines {
		t := strings.TrimSpace(line)
		if headingRe.MatchString(line) {
			inSubVerbs = subVerbHeadingRe.MatchString(t)
		}
		if inSubVerbs && (strings.HasPrefix(t, "|") || strings.HasPrefix(t, ">")) {
			lines[i] = ""
		}
	}
	text := strings.Join(lines, "\n")
	lineOf := func(offset int) int { return strings.Count(text[:offset], "\n") + 1 }

	var out []ShapeClaim
	for _, m := range flagSpelling.FindAllStringSubmatchIndex(text, -1) {
		out = append(out, ShapeClaim{Line: lineOf(m[4]), Spelling: text[m[4]:m[5]]})
	}
	// A command path is matched across a soft line wrap: markdown prose wraps
	// wherever the line fills, so `capture` ending one line and `resolve`
	// opening the next is still the spelling.
	masked := text
	for _, p := range paths {
		re := regexp.MustCompile(`(^|[^\w-])(` + strings.ReplaceAll(regexp.QuoteMeta(p), " ", `\s+`) + `)([^\w-]|$)`)
		for {
			m := re.FindStringSubmatchIndex(masked)
			if m == nil {
				break
			}
			out = append(out, ShapeClaim{Line: lineOf(m[4]), Spelling: p})
			masked = masked[:m[4]] + blankKeepingNewlines(masked[m[4]:m[5]]) + masked[m[5]:]
		}
	}
	for _, b := range bare {
		re := regexp.MustCompile("`" + strings.ReplaceAll(regexp.QuoteMeta(b), " ", `\s+`) + "`")
		for _, m := range re.FindAllStringIndex(text, -1) {
			out = append(out, ShapeClaim{Line: lineOf(m[0]), Spelling: "`" + b + "`"})
		}
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].Line < out[j].Line })
	return out
}

// blankKeepingNewlines replaces every byte of s but a newline with a space, so
// a matched span cannot match again and offsets and line numbers hold.
func blankKeepingNewlines(s string) string {
	b := []byte(s)
	for i := range b {
		if b[i] != '\n' {
			b[i] = ' '
		}
	}
	return string(b)
}

// RegisterRow is one row of the surfaces register.
type RegisterRow struct {
	Command string // command path: "abcd" for the bare `/abcd`, "abcd capture" for `/abcd:capture`
	Status  string // lower-cased Status cell
	Chapter string // the chapter's basename in the register's directory; "" when the row points elsewhere
	Line    int    // 1-based line in the register
}

var (
	slashCommandRe = regexp.MustCompile("^`?/abcd(?::([a-z][a-z0-9-]*))?`?$")
	linkTargetRe   = regexp.MustCompile(`\]\(([^)]+)\)`)
)

// ParseRegister reads the register's table — the first table whose header names
// a Command, a Status and a File column. A row whose File link leaves the
// directory, or carries an anchor, documents its surface outside the chapters
// and has an empty Chapter.
func ParseRegister(text string) []RegisterRow {
	lines := strings.Split(text, "\n")
	cmdCol, statusCol, fileCol, header := -1, -1, -1, -1
	for i, line := range lines {
		if !strings.HasPrefix(strings.TrimSpace(line), "|") {
			continue
		}
		c, s, f := -1, -1, -1
		for j, cell := range cells(line) {
			switch strings.ToLower(cell) {
			case "command":
				c = j
			case "status":
				s = j
			case "file":
				f = j
			}
		}
		if c >= 0 && s >= 0 && f >= 0 {
			cmdCol, statusCol, fileCol, header = c, s, f, i
			break
		}
	}
	if header < 0 {
		return nil
	}
	var rows []RegisterRow
	for i := header + 1; i < len(lines); i++ {
		t := strings.TrimSpace(lines[i])
		if !strings.HasPrefix(t, "|") {
			break
		}
		cs := cells(lines[i])
		if len(cs) <= cmdCol || len(cs) <= statusCol || len(cs) <= fileCol {
			continue
		}
		m := slashCommandRe.FindStringSubmatch(cs[cmdCol])
		if m == nil {
			continue // the separator row, or a row that names no /abcd surface
		}
		row := RegisterRow{Command: "abcd", Status: strings.ToLower(cs[statusCol]), Line: i + 1}
		if m[1] != "" {
			row.Command = "abcd " + m[1]
		}
		if lt := linkTargetRe.FindStringSubmatch(cs[fileCol]); lt != nil && !strings.ContainsAny(lt[1], "/#") {
			row.Chapter = lt[1]
		}
		rows = append(rows, row)
	}
	return rows
}

func cells(line string) []string {
	t := strings.Trim(strings.TrimSpace(line), "|")
	parts := strings.Split(t, "|")
	for i := range parts {
		parts[i] = strings.TrimSpace(parts[i])
	}
	return parts
}

// Chapter is one surface chapter and the command paths it documents, in
// register order.
type Chapter struct {
	File     string
	Commands []string
}

// Chapters pairs the chapter files with the register's rows. Every chapter file
// must be named by at least one row, and every row naming a chapter must name a
// file that exists: a chapter the register does not know has no command to
// derive an appendix from, and the refusal says so rather than skipping it.
func Chapters(rows []RegisterRow, files []string) ([]Chapter, error) {
	have := map[string]bool{}
	for _, f := range files {
		have[f] = true
	}
	byFile := map[string][]string{}
	for _, r := range rows {
		if r.Chapter == "" {
			continue
		}
		if !have[r.Chapter] {
			return nil, fmt.Errorf("%s (register line %d, %s): %w", r.Chapter, r.Line, r.Command, ErrRowWithoutChapter)
		}
		byFile[r.Chapter] = append(byFile[r.Chapter], r.Command)
	}
	sorted := append([]string(nil), files...)
	sort.Strings(sorted)
	var out []Chapter
	for _, f := range sorted {
		cmds, ok := byFile[f]
		if !ok {
			return nil, fmt.Errorf("%s: %w; add its row to %s/%s", f, ErrChapterWithoutRow, BriefSurfacesDir, RegisterFile)
		}
		out = append(out, Chapter{File: f, Commands: cmds})
	}
	return out, nil
}

// chapterFileRe is a surface chapter's basename: NN-<name>.md.
var chapterFileRe = regexp.MustCompile(`^[0-9]+-[a-z0-9-]+\.md$`)

// RegeneratedChapter is one chapter as committed and as the tree would have it.
type RegeneratedChapter struct {
	Chapter
	Committed string
	Want      string
}

// RegenerateChapters reads the register and every chapter in dir and returns
// each chapter regenerated against tree. It writes nothing: the generator writes
// Want, and the drift test compares it with Committed, so the file written and
// the file checked come from one code path. A refusal names the chapter.
func RegenerateChapters(dir string, tree []Command) ([]RegeneratedChapter, error) {
	register, err := os.ReadFile(filepath.Join(dir, RegisterFile))
	if err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	var files []string
	for _, e := range entries {
		if !e.IsDir() && chapterFileRe.MatchString(e.Name()) {
			files = append(files, e.Name())
		}
	}
	chapters, err := Chapters(ParseRegister(string(register)), files)
	if err != nil {
		return nil, err
	}
	out := make([]RegeneratedChapter, 0, len(chapters))
	for _, ch := range chapters {
		text, err := os.ReadFile(filepath.Join(dir, ch.File))
		if err != nil {
			return nil, err
		}
		want, err := RenderChapter(string(text), ComposeAppendix(ch.Commands, tree))
		if err != nil {
			return nil, fmt.Errorf("%s: %w", ch.File, err)
		}
		out = append(out, RegeneratedChapter{Chapter: ch, Committed: string(text), Want: want})
	}
	return out, nil
}

// Drift reports how the committed chapter differs from its regeneration, naming
// the chapter and each claim, or "" when they agree. A line the tree implies and
// the chapter lacks is missing; a line the chapter carries and the tree no longer
// implies is stale.
func (c RegeneratedChapter) Drift() string {
	if c.Committed == c.Want {
		return ""
	}
	missing, stale := DriftLines(c.Committed, c.Want)
	var b strings.Builder
	fmt.Fprintf(&b, "%s/%s: the generated appendix disagrees with the command tree", BriefSurfacesDir, c.File)
	for _, l := range missing {
		if strings.TrimSpace(l) != "" {
			fmt.Fprintf(&b, "\n  missing: %s", l)
		}
	}
	for _, l := range stale {
		if strings.TrimSpace(l) != "" {
			fmt.Fprintf(&b, "\n  stale:   %s", l)
		}
	}
	b.WriteString("\n  regenerate with `go generate ./internal/surface/cli` and commit the chapter")
	return b.String()
}
