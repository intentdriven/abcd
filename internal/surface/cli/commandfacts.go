package cli

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/intentdriven/abcd/internal/core/frontmatter"
	"github.com/intentdriven/abcd/internal/core/surface"
	"github.com/intentdriven/abcd/internal/termsafe"
)

// commandfacts.go — the derived copies of the command facts
// (itd-2610301346085246, spc-2610301431075964 step 3, approaches 4 and 6).
// The facts (CommandFacts, surface.go) are the one source; from them this file
// renders:
//   - one JSON file per top-level verb under docs/reference/cli/commands/,
//     holding the verb's visible subtree in the shape `abcd explain --json`
//     prints for one command;
//   - the command-facts block at the end of each command page: each visible
//     command's usage line, its inputs with their schema versions, and its
//     example;
//   - the ask-the-binary block straight after each page's H1 heading, the
//     page's first instruction: run `abcd explain <verb>` before filling in an
//     input, and follow the binary where it and the page disagree.
//
// The rendering lives here, beside GenerateReference, so the generator
// (cmd/command-facts-sync) and the drift tests (commandfacts_drift_test.go)
// render byte for byte the same. Everything here is a pure function of the
// facts and the page bytes: nothing reads the clock, the environment or the
// binary's version, so a copy changes only when a fact does.

// CommandFactsDir is the directory the per-verb JSON files are written to,
// repo-relative. It sits beside the generated CLI reference and outside
// commands/, which hosts read as command pages (the spec's first decision).
const CommandFactsDir = "docs/reference/cli/commands"

// CommandPagesDir is the plugin's command pages, repo-relative.
const CommandPagesDir = "commands"

// The markers fencing the two generated blocks of a command page. The end
// marker is the one every generated block in the repository closes with, and
// every begin marker opens with GeneratedBeginPrefix, this file's two and
// asking-sync's alike.
const (
	CommandFactsBegin    = "<!-- generated: command-facts -->"
	AskTheBinaryBegin    = "<!-- generated: ask-the-binary -->"
	GeneratedBeginPrefix = "<!-- generated:"
	GeneratedEnd         = "<!-- /generated -->"
)

// pluginRung is the binary as a plugin page invokes it: the plugin-root copy,
// the rung that resolves first for a plugin user. A page that never names it
// invokes the binary on PATH.
const (
	pluginRung = `"${CLAUDE_PLUGIN_ROOT}/abcd"`
	pathRung   = "abcd"
)

// userInputLine is the line a command page closes with, which hands the
// person's text to the page. The command-facts block stands just above it, so
// the person's input stays the page's last line.
const userInputLine = "**User input:** $ARGUMENTS"

// CommandFactsDocument is the shape of every derived JSON copy of the facts:
// each per-verb file and `abcd explain --json` for one command. It carries the
// facts' format version and no binary version, so a file changes only when a
// fact does.
type CommandFactsDocument struct {
	FormatVersion int             `json:"format_version"`
	Commands      []surface.Facts `json:"commands"`
}

// factsRoot is the path of the tree's root command in facts: the one path
// with no space in it.
func factsRoot(facts []surface.Facts) string {
	for _, f := range facts {
		if !strings.Contains(f.Path, " ") {
			return f.Path
		}
	}
	return ""
}

// visibleFacts is the facts of every command that is neither hidden nor below
// a hidden command, in the order given. The derived copies hold these alone,
// as the CLI reference does; a hidden command is answered by `abcd explain`
// from the running binary only.
func visibleFacts(facts []surface.Facts) []surface.Facts {
	hidden := map[string]bool{}
	for _, f := range facts {
		if f.Hidden {
			hidden[f.Path] = true
		}
	}
	var out []surface.Facts
	for _, f := range facts {
		words := strings.Fields(f.Path)
		under := false
		for i := 1; i <= len(words) && !under; i++ {
			under = hidden[strings.Join(words[:i], " ")]
		}
		if !under {
			out = append(out, f)
		}
	}
	return out
}

// factsOfPage is the facts a page named name covers, in canonical order: the
// root command alone for the root's own name (the dispatcher page and
// abcd.json), and otherwise the top-level verb of that name and every visible
// command below it. It is empty when the tree has no visible verb by that
// name.
func factsOfPage(facts []surface.Facts, name string) []surface.Facts {
	root := factsRoot(facts)
	var out []surface.Facts
	for _, f := range surface.CanonicalFacts(visibleFacts(facts)) {
		switch {
		case name == root && f.Path == root:
			out = append(out, f)
		case name != root && (f.Path == root+" "+name || strings.HasPrefix(f.Path, root+" "+name+" ")):
			out = append(out, f)
		}
	}
	return out
}

// EncodeCommandFacts encodes a facts document as the derived copies hold it:
// indented by two spaces, with no HTML escaping, so a usage line's <verb>
// reads as written, and with a closing newline.
func EncodeCommandFacts(doc CommandFactsDocument) ([]byte, error) {
	var b bytes.Buffer
	enc := json.NewEncoder(&b)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	if err := enc.Encode(doc); err != nil {
		return nil, err
	}
	return b.Bytes(), nil
}

// CommandFactsFiles renders every per-verb JSON file from facts, keyed by its
// repo-relative path: one file per visible top-level verb holding that verb's
// visible subtree, and one named for the root holding the root command alone,
// whose persistent flags every verb inherits. A verb with no plugin page gets
// its file too.
func CommandFactsFiles(facts []surface.Facts) (map[string][]byte, error) {
	root := factsRoot(facts)
	if root == "" {
		return nil, fmt.Errorf("the facts hold no root command")
	}
	names := []string{root}
	for _, f := range surface.CanonicalFacts(visibleFacts(facts)) {
		if verb, ok := strings.CutPrefix(f.Path, root+" "); ok && !strings.Contains(verb, " ") {
			names = append(names, verb)
		}
	}
	out := make(map[string][]byte, len(names))
	for _, name := range names {
		data, err := EncodeCommandFacts(CommandFactsDocument{
			FormatVersion: surface.FactsFormatVersion,
			Commands:      factsOfPage(facts, name),
		})
		if err != nil {
			return nil, fmt.Errorf("encode %s: %w", name, err)
		}
		out[CommandFactsDir+"/"+name+".json"] = data
	}
	return out, nil
}

// pageRung is the binary as the page invokes it: the plugin-root copy when the
// page's hand-written text names it anywhere, and the binary on PATH
// otherwise. The generated blocks are left out of the reading, so a rung the
// sync once wrote into a block never keeps itself once the text stops naming
// it.
func pageRung(page []byte) string {
	if bytes.Contains(withoutGeneratedBlocks(page), []byte(pluginRung)) {
		return pluginRung
	}
	return pathRung
}

// explainCall is `abcd explain <words>` spelled with rung. The words for a
// page are its name: the verb, or the root's own name for the dispatcher page,
// which explain reads as the root.
func explainCall(rung, words string) string {
	return rung + " explain " + words
}

// AskTheBinaryBlock is the block that stands straight after a page's H1
// heading, from its begin marker through its end marker: one sentence sending
// the agent to the running binary's `abcd explain` before it fills in an
// input, spelled with the page's own rung.
func AskTheBinaryBlock(name, rung string) string {
	return AskTheBinaryBegin + "\n" +
		"Before you fill in an input for this command, run " + codeSpan(explainCall(rung, name)) +
		", and where its answer and this page disagree, follow the binary, since this page may be older than it.\n" +
		GeneratedEnd
}

// CommandFactsBlock is the block that stands at the end of a page, from its
// begin marker through its end marker. Its first line names the facts' format
// version and sends the reader to the running binary where the two disagree.
// For each command the page covers it lists the usage line, each input with
// its schema version, and the example. A page whose commands declare no
// visible flag and read no input gets the first line alone.
func CommandFactsBlock(facts []surface.Facts, name, rung string) string {
	var b strings.Builder
	b.WriteString(CommandFactsBegin + "\n")
	fmt.Fprintf(&b, "**Command facts, format version %d.** Where this block and %s from the running binary disagree, follow the binary.\n",
		surface.FactsFormatVersion, codeSpan(explainCall(rung, name)))
	covered := factsOfPage(facts, name)
	if !declaresAnything(covered) {
		b.WriteString(GeneratedEnd)
		return b.String()
	}
	b.WriteString("<!-- Written by `make command-facts-sync` from the command tree; change the command in internal/surface/cli or internal/core/surface, never this block. -->\n\n")
	for _, f := range covered {
		b.WriteString("- " + codeSpan(f.Usage) + "\n")
		for _, in := range f.Inputs {
			b.WriteString("  - " + factsInputLine(f.Path, in, rung) + "\n")
		}
		switch lines := strings.Split(f.Example, "\n"); {
		case f.Example == "":
		case len(lines) == 1:
			b.WriteString("  - An example call is " + codeSpan(strings.TrimSpace(lines[0])) + ".\n")
		default:
			b.WriteString("  - The example calls are these:\n")
			for _, line := range lines {
				if line = strings.TrimSpace(line); line != "" {
					b.WriteString("    - " + codeSpan(line) + "\n")
				}
			}
		}
	}
	b.WriteString(GeneratedEnd)
	return b.String()
}

// declaresAnything reports whether any of the commands declares a visible flag
// of its own or reads an input.
func declaresAnything(facts []surface.Facts) bool {
	for _, f := range facts {
		if len(f.Inputs) > 0 {
			return true
		}
		for _, fl := range f.Flags {
			if !fl.Hidden {
				return true
			}
		}
	}
	return false
}

// factsInputLine is one input in the command-facts block: the flag, what the
// input is, and for a filled-in JSON input its schema version and the call
// that prints its shape, in the words `abcd explain` uses.
func factsInputLine(path string, in surface.InputFact, rung string) string {
	if in.Kind != surface.InputJSON {
		return "Reads " + codeSpan("--"+in.Flag) + ", the " + in.Name + "."
	}
	return fmt.Sprintf("Reads %s, the %s, at %s; %s prints its shape.",
		codeSpan("--"+in.Flag), in.Name, explainVersionText(in.SchemaVersion, in.VersionIn),
		codeSpan(explainCall(rung, explainArgs(path)+" --input "+in.Flag)))
}

// codeSpan is s as a Markdown code span, through termsafe.CodeSpan: fenced by
// a run of backticks longer than any run inside it, and padded where CommonMark
// needs it, so any text reads back as written.
func codeSpan(s string) string {
	return termsafe.CodeSpan(s)
}

// SyncCommandPage returns page with both generated blocks rendered from facts
// for the page named name, every byte outside the blocks kept. A block that is
// present is rewritten where it stands, which must be its place: the
// ask-the-binary block straight after the H1 heading, with nothing but blank
// lines between, and the command-facts block last, followed by nothing but the
// closing user-input line. A block that is absent is written into its place.
// A page with no H1 heading, a block opened twice or never closed (before the
// page ends, or before another generated block opens), a block out of its
// place, or a name the tree has no visible verb for is a fault: the generator
// never guesses.
func SyncCommandPage(page []byte, name string, facts []surface.Facts) ([]byte, error) {
	if len(factsOfPage(facts, name)) == 0 {
		return nil, fmt.Errorf("%s/%s.md names no visible command of the tree", CommandPagesDir, name)
	}
	rung := pageRung(page)
	out, err := syncAskTheBinary(string(page), name, AskTheBinaryBlock(name, rung))
	if err != nil {
		return nil, err
	}
	out, err = syncCommandFactsBlock(out, name, CommandFactsBlock(facts, name, rung))
	if err != nil {
		return nil, err
	}
	return []byte(out), nil
}

// findBlock locates the one block opened by begin: the offset of its begin
// marker and the offset just past its end marker, or -1 for both when the
// page has none.
func findBlock(page, name, begin string) (int, int, error) {
	switch n := strings.Count(page, begin); n {
	case 0:
		return -1, -1, nil
	case 1:
	default:
		return 0, 0, fmt.Errorf("%s/%s.md carries %d %s markers; it must carry one", CommandPagesDir, name, n, begin)
	}
	start := strings.Index(page, begin)
	inner := start + len(begin)
	stop := strings.Index(page[inner:], GeneratedEnd)
	if stop < 0 {
		return 0, 0, fmt.Errorf("%s/%s.md opens %s and never closes it with %s", CommandPagesDir, name, begin, GeneratedEnd)
	}
	if strings.Contains(page[inner:inner+stop], GeneratedBeginPrefix) {
		return 0, 0, fmt.Errorf("%s/%s.md opens %s and never closes it with %s before the next generated block opens; "+
			"close it, or the sync would take that block's end marker and delete the text between", CommandPagesDir, name, begin, GeneratedEnd)
	}
	return start, inner + stop + len(GeneratedEnd), nil
}

// withoutGeneratedBlocks is page with every generated block removed, from its
// begin marker through the first end marker after it, every generated block
// of the repository's included (the asking-rules block of intent.md as much as
// this file's two). A block left open is removed through the end of the page;
// the sync refuses such a page in any case.
func withoutGeneratedBlocks(page []byte) []byte {
	var out []byte
	for {
		start := bytes.Index(page, []byte(GeneratedBeginPrefix))
		if start < 0 {
			return append(out, page...)
		}
		out = append(out, page[:start]...)
		stop := bytes.Index(page[start:], []byte(GeneratedEnd))
		if stop < 0 {
			return out
		}
		page = page[start+stop+len(GeneratedEnd):]
	}
}

// headingEnd is the offset just past the newline that ends the page's first
// H1 heading, read after the frontmatter, or -1 when the page has none.
func headingEnd(page string) int {
	head, _ := frontmatter.Split(page)
	at := len(head)
	for at < len(page) {
		next := strings.IndexByte(page[at:], '\n')
		line := page[at:]
		if next >= 0 {
			line = page[at : at+next]
		}
		if strings.HasPrefix(line, "# ") {
			if next < 0 {
				return len(page)
			}
			return at + next + 1
		}
		if next < 0 {
			break
		}
		at += next + 1
	}
	return -1
}

// syncAskTheBinary writes block straight after the page's H1 heading, one
// blank line on each side of it.
func syncAskTheBinary(page, name, block string) (string, error) {
	h := headingEnd(page)
	if h < 0 {
		return "", fmt.Errorf("%s/%s.md has no H1 heading for the ask-the-binary block to follow", CommandPagesDir, name)
	}
	start, stop, err := findBlock(page, name, AskTheBinaryBegin)
	if err != nil {
		return "", err
	}
	rest := page[h:]
	if start >= 0 {
		if start < h || strings.TrimSpace(page[h:start]) != "" {
			return "", fmt.Errorf("%s/%s.md carries the ask-the-binary block away from its place straight after the H1 heading; "+
				"move it there, or delete it and rerun the sync", CommandPagesDir, name)
		}
		rest = page[stop:]
	}
	return page[:h] + "\n" + block + "\n\n" + strings.TrimLeft(rest, "\n"), nil
}

// syncCommandFactsBlock writes block at the end of the page, above the
// closing user-input line when the page has one, one blank line on each side
// of it.
func syncCommandFactsBlock(page, name, block string) (string, error) {
	start, stop, err := findBlock(page, name, CommandFactsBegin)
	if err != nil {
		return "", err
	}
	body, tail := page, ""
	if start >= 0 {
		after := strings.TrimSpace(page[stop:])
		if after != "" && after != userInputLine {
			return "", fmt.Errorf("%s/%s.md carries the command-facts block away from its place at the end of the page; "+
				"move it there, or delete it and rerun the sync", CommandPagesDir, name)
		}
		body, tail = page[:start], after
	} else if trimmed := strings.TrimRight(page, "\n \t"); strings.HasSuffix(trimmed, "\n"+userInputLine) {
		body, tail = strings.TrimSuffix(trimmed, userInputLine), userInputLine
	}
	out := strings.TrimRight(body, "\n") + "\n\n" + block + "\n"
	if tail != "" {
		out += "\n" + tail + "\n"
	}
	return out, nil
}
