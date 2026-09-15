// Package glossary derives the brief glossary's own index from the term files.
//
// `.abcd/development/brief/glossary/` is the project's one canonical glossary:
// one Markdown file per term, grouped by bounded context, each carrying the
// frontmatter the `GL002` forbidden-synonym rule reads (`internal/core/lint`).
// Its README used to enumerate the contexts and the terms by hand, which is a
// second copy of what the directory already states — this package makes the
// directory the source and the README a render of it, so a term file that lands
// without an index row fails the build instead of going unlisted.
//
// It is transport-agnostic like the rest of `core/`: it reads the working tree
// and returns strings; nothing here writes to stdout or to the README.
package glossary

import (
	"fmt"
	"os"
	"path"
	"sort"
	"strings"

	"github.com/intentdriven/abcd/internal/core/frontmatter"
	"github.com/intentdriven/abcd/internal/fsutil"
)

// maxTermBytes bounds one term-file read.
const maxTermBytes = 1 << 20

const (
	// DirRelPath is the canonical glossary's repo-relative home (adr-30).
	DirRelPath = ".abcd/development/brief/glossary"
	// READMERelPath is the index page that carries the rendered blocks.
	READMERelPath = DirRelPath + "/README.md"

	// LayoutMarkerBegin and LayoutMarkerEnd delimit the rendered directory
	// layout inside the glossary README.
	LayoutMarkerBegin = "<!-- BEGIN GENERATED: glossary-layout -->"
	LayoutMarkerEnd   = "<!-- END GENERATED: glossary-layout -->"

	// IndexMarkerBegin and IndexMarkerEnd delimit the rendered term index
	// inside the glossary README.
	IndexMarkerBegin = "<!-- BEGIN GENERATED: glossary-index -->"
	IndexMarkerEnd   = "<!-- END GENERATED: glossary-index -->"
)

// Term is one term file's index-bearing frontmatter.
type Term struct {
	// Name is the canonical term, from the `term` field.
	Name string
	// File is the term file's name within its context directory.
	File string
	// Status is the `status` field (draft, stable, or deprecated).
	Status string
	// Definition is the `definition` field, verbatim.
	Definition string
	// Aliases are the `aliases` field's members — the other spellings that mean
	// this term. Absent, null or empty, there are none.
	Aliases []string
}

// Context is one bounded context: a directory of term files.
type Context struct {
	// Name is the directory name, which every member's `bounded_context` matches.
	Name string
	// Files are the context directory's Markdown files, sorted — term files and
	// the context README alike, so the rendered layout matches the directory.
	Files []string
	// Terms are the context's term files, sorted by file name.
	Terms []Term
}

// Glossary is the whole glossary as the directory states it.
type Glossary struct {
	// RootFiles are the Markdown files at the glossary root (the README and the
	// term template), sorted.
	RootFiles []string
	// Contexts are the bounded-context directories, sorted by name.
	Contexts []Context
}

// isTermFile reports whether a Markdown file in a context directory is a term
// file. The context README documents the context, and a leading underscore marks
// the template — neither registers a term.
func isTermFile(name string) bool {
	return strings.HasSuffix(name, ".md") && name != "README.md" && !strings.HasPrefix(name, "_")
}

// readDirInRoot lists one directory through the root, so an ancestor symlink a
// clone commits cannot walk the scan out of the repository.
func readDirInRoot(root *os.Root, rel string) ([]os.DirEntry, error) {
	f, err := root.Open(rel)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	entries, err := f.ReadDir(-1)
	if err != nil {
		return nil, err
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].Name() < entries[j].Name() })
	return entries, nil
}

// markdownNames returns the sorted Markdown file names directly inside dir.
func markdownNames(root *os.Root, dir string) ([]string, error) {
	entries, err := readDirInRoot(root, dir)
	if err != nil {
		return nil, err
	}
	var out []string
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") {
			continue
		}
		out = append(out, e.Name())
	}
	sort.Strings(out)
	return out, nil
}

// Scan reads the glossary directory and returns what it states. A term file
// missing the `term`, `status`, or `definition` field is an error, not a silent
// omission: an unreadable term must not vanish from the index it belongs in.
func Scan(repoRoot string) (Glossary, error) {
	root, err := os.OpenRoot(repoRoot)
	if err != nil {
		return Glossary{}, err
	}
	defer root.Close()
	return ScanInRoot(root)
}

// ScanInRoot is Scan over a repository root a caller already holds open. It is
// the implementation both forms share: a caller that opens its own root — the
// site build does, so a committed ancestor symlink cannot walk a read out of the
// repository — must not pay for a second one, and must not get a second scanner.
func ScanInRoot(root *os.Root) (Glossary, error) {
	rootFiles, err := markdownNames(root, DirRelPath)
	if err != nil {
		return Glossary{}, err
	}
	entries, err := readDirInRoot(root, DirRelPath)
	if err != nil {
		return Glossary{}, err
	}
	g := Glossary{RootFiles: rootFiles}
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		ctx, err := scanContext(root, e.Name())
		if err != nil {
			return Glossary{}, err
		}
		g.Contexts = append(g.Contexts, ctx)
	}
	sort.Slice(g.Contexts, func(i, j int) bool { return g.Contexts[i].Name < g.Contexts[j].Name })
	return g, nil
}

// scanContext reads one bounded-context directory.
func scanContext(root *os.Root, name string) (Context, error) {
	dir := path.Join(DirRelPath, name)
	files, err := markdownNames(root, dir)
	if err != nil {
		return Context{}, err
	}
	ctx := Context{Name: name, Files: files}
	for _, f := range files {
		if !isTermFile(f) {
			continue
		}
		t, err := readTerm(root, path.Join(dir, f))
		if err != nil {
			return Context{}, err
		}
		t.File = f
		ctx.Terms = append(ctx.Terms, t)
	}
	return ctx, nil
}

// readTerm parses one term file's frontmatter. Term files may carry an
// attribution comment above the opening `---`, so the scan starts at the
// delimiter rather than at line 0.
func readTerm(root *os.Root, rel string) (Term, error) {
	raw, err := fsutil.ReadGuardedInRoot(root, rel, maxTermBytes)
	if err != nil {
		return Term{}, err
	}
	lines := strings.Split(string(raw), "\n")
	if open := frontmatterOpen(lines); open > 0 {
		lines = lines[open:]
	}
	fields := frontmatter.Fields(lines)
	t := Term{}
	for _, spec := range []struct {
		key string
		dst *string
	}{
		{"term", &t.Name},
		{"status", &t.Status},
		{"definition", &t.Definition},
	} {
		f, ok := fields[spec.key]
		// A YAML null (`term: NULL`, `status: ~`, `definition: null`) is a MISSING
		// field, not a literal value: without this gate `term: NULL` passed the
		// TrimSpace-empty presence check and minted a phantom term named "NULL"
		// (iss-2608270908339164). frontmatter.IsNull is the one canonical predicate.
		if !ok || frontmatter.IsNull(strings.TrimSpace(f.Value)) {
			return Term{}, fmt.Errorf("%s: term file has no %s field", rel, spec.key)
		}
		*spec.dst = strings.TrimSpace(f.Value)
	}
	// `aliases` is OPTIONAL, and its absence is the same as an empty list: a
	// term with no other spelling declares none. A YAML null is asked about
	// through the one predicate, so `aliases: null` is an absence rather than a
	// term called "null".
	if f, ok := fields["aliases"]; ok {
		if v := strings.TrimSpace(f.Value); !frontmatter.IsNull(v) {
			t.Aliases = frontmatter.StringList(v)
		}
	}
	return t, nil
}

// frontmatterOpen returns the index of the opening `---` delimiter, skipping
// leading blank lines and HTML comments (single- and multi-line); -1 when the
// leading block is not frontmatter. The first line is BOM-stripped before every
// comparison so a UTF-8 byte-order mark ahead of the `---` does not read as
// missing frontmatter.
func frontmatterOpen(lines []string) int {
	norm := func(idx int) string {
		s := lines[idx]
		if idx == 0 {
			s = frontmatter.TrimBOM(s)
		}
		return strings.TrimSpace(s)
	}
	inComment := false
	for i := 0; i < len(lines); i++ {
		t := norm(i)
		switch {
		case inComment:
			if strings.Contains(t, "-->") {
				inComment = false
			}
		case t == "":
			// blank line: skip.
		case strings.HasPrefix(t, "<!--") && strings.HasSuffix(t, "-->"):
			// a complete single-line comment: skip.
		case strings.HasPrefix(t, "<!--"):
			inComment = true
		case t == "---":
			return i
		default:
			return -1
		}
	}
	return -1
}

// RenderLayout returns the glossary's directory layout as a fenced tree, exactly
// as it appears between the layout markers in the glossary README.
func RenderLayout(g Glossary) string {
	var b strings.Builder
	b.WriteString("```\n")
	b.WriteString("glossary/\n")
	// Root files come first, then one branch per context directory; the last
	// context closes the tree with the terminal connector.
	for _, f := range g.RootFiles {
		b.WriteString("├── " + f + "\n")
	}
	for i, ctx := range g.Contexts {
		last := i == len(g.Contexts)-1
		connector, indent := "├── ", "│   "
		if last {
			connector, indent = "└── ", "    "
		}
		b.WriteString(connector + ctx.Name + "/\n")
		for j, f := range ctx.Files {
			child := "├── "
			if j == len(ctx.Files)-1 {
				child = "└── "
			}
			b.WriteString(indent + child + f + "\n")
		}
	}
	b.WriteString("```")
	return b.String()
}

// RenderIndex returns the term index — one subsection and table per bounded
// context — exactly as it appears between the index markers in the README.
func RenderIndex(g Glossary) string {
	var b strings.Builder
	for i, ctx := range g.Contexts {
		if i > 0 {
			b.WriteString("\n")
		}
		fmt.Fprintf(&b, "### %s/\n\n", ctx.Name)
		b.WriteString("| Term | Status | Definition |\n")
		b.WriteString("|---|---|---|\n")
		for _, t := range ctx.Terms {
			fmt.Fprintf(&b, "| [%s](%s/%s) | %s | %s |\n",
				t.Name, ctx.Name, t.File, t.Status, escapeCell(t.Definition))
		}
	}
	return strings.TrimRight(b.String(), "\n")
}

// escapeCell makes a definition safe inside a Markdown table cell: a pipe would
// otherwise split the row into a column the header does not have.
func escapeCell(s string) string { return strings.ReplaceAll(s, "|", `\|`) }
