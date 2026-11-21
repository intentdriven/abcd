package lint

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// TestLoopGlossaryAllowsBareLoopInTheImplementLoopChapters holds the loop
// glossary entry to the chapters whose subject is the implement loop
// (iss-2610262037101841). The entry once kept bare "the loop" for the record
// loop alone, while 34-build.md writes "Only the loop writes a verdict" for
// the build run's own iteration. So the entry registers the implement loop as
// a sense, allows bare "the loop" for it inside 34-build.md, 27-implement.md
// and commands/build.md once the chapter has named it, and each of those
// chapters names "the implement loop" before its first bare use.
func TestLoopGlossaryAllowsBareLoopInTheImplementLoopChapters(t *testing.T) {
	read := func(rel string) string {
		raw, err := os.ReadFile(filepath.Join(repoRootFromPackage, filepath.FromSlash(rel)))
		if err != nil {
			t.Fatalf("read %s: %v", rel, err)
		}
		return string(raw)
	}
	section := func(doc, heading string) string {
		start := strings.Index(doc, "\n"+heading+"\n")
		if start < 0 {
			return ""
		}
		rest := doc[start+len(heading)+2:]
		if end := strings.Index(rest, "\n## "); end >= 0 {
			rest = rest[:end]
		}
		return strings.Join(strings.Fields(rest), " ")
	}

	const glossary = ".abcd/development/brief/glossary/core/loop.md"
	entry := read(glossary)
	if !strings.Contains(section(entry, "## Senses"), "**the implement loop**") {
		t.Errorf("%s's Senses table does not register **the implement loop**", glossary)
	}
	// Each chapter as the repository holds it, and as the entry names it.
	chapters := map[string]string{
		".abcd/development/brief/04-surfaces/34-build.md":     "`34-build.md`",
		".abcd/development/brief/04-surfaces/27-implement.md": "`27-implement.md`",
		"commands/build.md": "`commands/build.md`",
	}
	use := section(entry, "## When to use")
	for _, shown := range chapters {
		if !strings.Contains(use, shown) {
			t.Errorf("%s's When to use section does not name %s among the chapters where bare \"the loop\" is the implement loop", glossary, shown)
		}
	}
	if !strings.Contains(use, "once") || !strings.Contains(use, "named") {
		t.Errorf("%s's When to use section does not tie the bare form to the chapter having named the implement loop first", glossary)
	}

	bare := regexp.MustCompile(`(?i)\bthe loop\b`)
	named := regexp.MustCompile(`(?i)\bthe implement loop\b`)
	for ch := range chapters {
		body := read(ch)
		// A command page's frontmatter is the listing's description, not the chapter.
		if strings.HasPrefix(body, "---\n") {
			if end := strings.Index(body[4:], "\n---\n"); end >= 0 {
				body = body[4+end+5:]
			}
		}
		body = strings.Join(strings.Fields(body), " ")
		first := bare.FindStringIndex(body)
		if first == nil {
			continue
		}
		name := named.FindStringIndex(body)
		if name == nil || name[0] > first[0] {
			t.Errorf("%s writes bare \"the loop\" before it names \"the implement loop\"; the glossary allows the bare form there only once the chapter has named it", ch)
		}
	}
}
