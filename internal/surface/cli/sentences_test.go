package cli

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/frontmatter"
	"github.com/intentdriven/abcd/internal/core/lint"
	"github.com/intentdriven/abcd/internal/core/surface"
	"github.com/spf13/cobra"
)

// sentences_test.go holds itd-2609212113220149: every visible verb and sub-verb
// opens with one sentence (does / writes / refuses), declared once in the
// surface manifest and rendered byte-identically onto the command list, the
// verb's own --help, the agents block and the verb's plugin page.

// helpOf renders the help of the command at path on a fresh tree from build,
// as `<path> --help` prints it. The root's list is rendered with --agent where
// the tree has that flag, so both blocks are in view.
func helpOf(t *testing.T, build func() *cobra.Command, path []string) string {
	t.Helper()
	root := build()
	args := append(append([]string{}, path...), "--help")
	if len(path) == 0 && root.Flags().Lookup("agent") != nil {
		args = append(args, "--agent")
	}
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs(args)
	if err := root.Execute(); err != nil {
		t.Fatalf("%s %v: %v\n%s", root.Name(), args, err, out.String())
	}
	return out.String()
}

// visibleCommands is every command of the tree a reader can see: the root and
// each command with no hidden command on its path. A stub a moved spelling
// leaves (itd-2609212130136102) is deprecated, which cobra lists nowhere, so a
// reader never meets it and it carries no sentence.
func visibleCommands(root *cobra.Command) []*cobra.Command {
	out := []*cobra.Command{root}
	var walk func(*cobra.Command)
	walk = func(c *cobra.Command) {
		for _, sub := range c.Commands() {
			if sub.Hidden || sub.Deprecated != "" {
				continue
			}
			out = append(out, sub)
			walk(sub)
		}
	}
	walk(root)
	return out
}

// listsSentence reports whether a rendered help lists the entry name with the
// sentence as its whole summary, as cobra's command list and the root's groups
// print it: the name, padding, the sentence, and nothing after it but the page
// an agents-block line names.
func listsSentence(help, name, sentence string) bool {
	re := regexp.MustCompile(`(?m)^ +` + regexp.QuoteMeta(name) + ` +` + regexp.QuoteMeta(sentence) + `( \(read [^)]+\))?$`)
	return re.MatchString(help)
}

// sentenceDefects walks every visible command of the tree build makes and names
// each one whose sentence is missing, malformed, or rendered differently in any
// of its places: the manifest (form), the command list of its parent, its own
// --help (which must open with it), and — for a verb with a plugin page — the
// page's description. pages maps a page name (a top-level verb, or the root's
// name) to the page's bytes.
func sentenceDefects(t *testing.T, build func() *cobra.Command, manifest func(string) (string, bool),
	pages map[string]string) []string {
	t.Helper()
	var out []string
	for _, cmd := range visibleCommands(build()) {
		path := cmd.CommandPath()
		sentence, ok := manifest(path)
		if !ok {
			out = append(out, path+": no sentence in the manifest")
			continue
		}
		if _, err := surface.ParseSentence(sentence); err != nil {
			out = append(out, fmt.Sprintf("%s: %v", path, err))
		}
		words := strings.Fields(path)[1:]
		if cmd.HasParent() {
			parentHelp := helpOf(t, build, words[:len(words)-1])
			if !listsSentence(parentHelp, cmd.Name(), sentence) {
				out = append(out, fmt.Sprintf("%s: the command list of %q does not show the manifest's sentence", path, cmd.Parent().CommandPath()))
			}
		}
		own := helpOf(t, build, words)
		if first, _, _ := strings.Cut(own, "\n"); first != sentence {
			out = append(out, fmt.Sprintf("%s: --help opens with %q, not the manifest's sentence", path, first))
		}
		page := cmd.Name()
		if cmd.HasParent() && cmd.Parent().HasParent() {
			continue // a sub-verb has no page of its own
		}
		body, ok := pages[page]
		if !ok {
			continue
		}
		head, _ := frontmatter.Split(body)
		field, has := frontmatter.Fields(strings.Split(head, "\n"))["description"]
		value, scalar := frontmatter.ScalarString(field.Value)
		switch {
		case !has || !scalar:
			out = append(out, fmt.Sprintf("%s: commands/%s.md has no single-line description", path, page))
		case value != sentence:
			out = append(out, fmt.Sprintf("%s: commands/%s.md's description differs from the manifest's sentence: %q", path, page, value))
		}
	}
	return out
}

// TestEveryVisibleVerbCarriesItsSentenceEverywhere is criteria 1, 2 and 3 on the
// tree that ships: every visible verb and sub-verb has a well-formed sentence in
// the manifest, and the command list, its own --help and its plugin page all
// render that sentence byte for byte.
func TestEveryVisibleVerbCarriesItsSentenceEverywhere(t *testing.T) {
	if n := len(visibleCommands(NewRootCommand())); n < 100 {
		t.Fatalf("the walk saw %d visible commands; the tree has more than a hundred, so the check would pass on a fraction", n)
	}
	for _, d := range sentenceDefects(t, NewRootCommand, surface.SentenceFor, commandFileBodies(t)) {
		t.Error(d)
	}
}

// TestSentenceDefectsNamesTheVerbAndTheDefect is criterion 3's negative control:
// on a synthetic tree, a verb with no sentence, one over the cap, one missing a
// clause, one whose command list shows other text and one whose page differs are
// each named with their defect, and the well-formed verb is not named at all.
func TestSentenceDefectsNamesTheVerbAndTheDefect(t *testing.T) {
	good := "Do the thing: Writes nothing; refuses an unknown thing."
	manifest := map[string]string{
		"tool":         "Run the tool: Writes nothing; refuses an unknown verb.",
		"tool good":    good,
		"tool long":    "Do the " + strings.Repeat("long ", 40) + "thing: Writes nothing; refuses nothing.",
		"tool clause":  "Do the thing: Writes nothing.",
		"tool differs": good,
		"tool paged":   good,
	}
	build := func() *cobra.Command {
		root := &cobra.Command{Use: "tool", Run: func(*cobra.Command, []string) {}}
		for _, name := range []string{"good", "missing", "long", "clause", "differs", "paged"} {
			root.AddCommand(&cobra.Command{Use: name, Run: func(*cobra.Command, []string) {}})
		}
		applySentences(root, func(path string) (string, bool) { s, ok := manifest[path]; return s, ok })
		findByPath(root, []string{"differs"}).Short = "Some other text."
		return root
	}
	pages := map[string]string{
		"good":  "---\nname: good\ndescription: " + frontmatter.QuoteScalar(good) + "\n---\n\n# good\n",
		"paged": "---\nname: paged\ndescription: Something else.\n---\n\n# paged\n",
	}
	lookup := func(path string) (string, bool) { s, ok := manifest[path]; return s, ok }
	defects := sentenceDefects(t, build, lookup, pages)

	joined := strings.Join(defects, "\n")
	for _, want := range []struct{ verb, defect string }{
		{"tool missing", "no sentence"},
		{"tool long", "over the cap"},
		{"tool clause", "semicolon"},
		{"tool differs", "command list"},
		{"tool differs", "--help opens with"},
		{"tool paged", "commands/paged.md's description differs"},
	} {
		found := false
		for _, d := range defects {
			if strings.HasPrefix(d, want.verb+":") && strings.Contains(d, want.defect) {
				found = true
			}
		}
		if !found {
			t.Errorf("no defect names %q with %q; got:\n%s", want.verb, want.defect, joined)
		}
	}
	for _, d := range defects {
		if strings.HasPrefix(d, "tool good:") || strings.HasPrefix(d, "tool:") {
			t.Errorf("a well-formed verb was named: %s", d)
		}
	}
}

// TestVerbHelpOpensWithTheSentenceThenTheLongHelp pins the help's shape on one
// verb with a long body: the sentence is the first line, the long help follows
// it after one blank line, and the usage comes after both.
func TestVerbHelpOpensWithTheSentenceThenTheLongHelp(t *testing.T) {
	sentence, ok := surface.SentenceFor("abcd peers")
	if !ok {
		t.Fatal("abcd peers has no sentence in the manifest")
	}
	help := helpOf(t, NewRootCommand, []string{"peers"})
	cmd := findByPath(NewRootCommand(), []string{"peers"})
	want := sentence + "\n\n" + strings.TrimSpace(cmd.Long) + "\n\nUsage:"
	if !strings.HasPrefix(help, want) {
		t.Fatalf("abcd peers --help does not open with the sentence and then the long help:\n%s", help)
	}
}

// TestAgentBlockLinesAreTheSentences is criterion 4: with --help --agent, each
// line of the agents-and-hosts block is the entry's name, its sentence from the
// manifest, and the page it names, and nothing else.
func TestAgentBlockLinesAreTheSentences(t *testing.T) {
	help, root := executedHelp(t, "--help", "--agent")
	_, block, found := strings.Cut(help, "\n"+helpAgentsTitle+"\n")
	if !found {
		t.Fatalf("no agents-and-hosts block\n%s", help)
	}
	block, _, _ = strings.Cut(block, "\n\n")
	entries := agentEntries(root)
	if len(entries) == 0 {
		t.Fatal("the agents block lists nothing; the check would pass vacuously")
	}
	lines := strings.Split(block, "\n")
	if len(lines) != len(entries) {
		t.Fatalf("the block has %d lines for %d entries\n%s", len(lines), len(entries), block)
	}
	for i, e := range entries {
		sentence, ok := surface.SentenceFor("abcd " + e.name)
		if !ok {
			t.Errorf("abcd %s: no sentence in the manifest", e.name)
			continue
		}
		re := regexp.MustCompile(`^  ` + regexp.QuoteMeta(e.name) + ` +` + regexp.QuoteMeta(sentence) +
			regexp.QuoteMeta(" (read "+e.page+")") + `$`)
		if !re.MatchString(lines[i]) {
			t.Errorf("agents block line %d = %q, want %q's sentence and page", i+1, lines[i], e.name)
		}
	}
}

// TestSentencesAreCheckedByTheDocsLint is criterion 5: the sentences are on a
// page the docs lint walks, so they are checked as any page is. The generated
// CLI reference renders every sentence as its own line, the reference lies
// under a root the committed docs-lint configuration declares, and the lint's
// banned-token rules find nothing in any sentence — while a sentence carrying a
// banned token is caught, so the check is not passing by not looking.
func TestSentencesAreCheckedByTheDocsLint(t *testing.T) {
	repo := testRepoRoot()
	cfg, err := lint.LoadConfig(filepath.Join(repo, ".abcd", "docs-lint.json"))
	if err != nil {
		t.Fatalf("loading the docs-lint configuration: %v", err)
	}
	const reference = "docs/reference/cli/commands.md"
	covered := false
	for _, root := range cfg.Roots {
		if root == reference || strings.HasPrefix(reference, strings.TrimSuffix(root, "/")+"/") {
			covered = true
		}
	}
	if !covered {
		t.Fatalf("no docs-lint root %v covers %s, so the lint never reads the sentences", cfg.Roots, reference)
	}
	data, err := os.ReadFile(filepath.Join(repo, filepath.FromSlash(reference)))
	if err != nil {
		t.Fatalf("reading %s: %v", reference, err)
	}
	lines := map[string]bool{}
	for _, l := range strings.Split(string(data), "\n") {
		lines[l] = true
	}
	checker, err := lint.NewTokenChecker(cfg.BannedTokens)
	if err != nil {
		t.Fatalf("compiling the banned tokens: %v", err)
	}
	if checker.Len() == 0 {
		t.Fatal("the docs-lint configuration arms no banned token; the check would pass vacuously")
	}
	for _, cmd := range visibleCommands(NewRootCommand()) {
		sentence, ok := surface.SentenceFor(cmd.CommandPath())
		if !ok {
			continue // named by TestEveryVisibleVerbCarriesItsSentenceEverywhere
		}
		if !lines[sentence] {
			t.Errorf("%s: its sentence is not a line of %s, so the docs lint does not read it", cmd.CommandPath(), reference)
		}
		for _, f := range checker.LintComposed(reference, sentence, false, nil) {
			t.Errorf("%s: the docs lint refuses its sentence: %s", cmd.CommandPath(), f.Message)
		}
	}
	if len(checker.LintComposed(reference, "Do the thing as it previously did: Writes nothing; refuses nothing.", false, nil)) == 0 {
		t.Fatal("a sentence carrying a banned token passed the docs lint's rules")
	}
}

// TestSentencePagesReproduceTheCommittedPages is the pages' drift test: the
// generator's rendering of every page equals the committed bytes, so running
// `go generate ./internal/surface/cli` over a current tree changes nothing.
func TestSentencePagesReproduceTheCommittedPages(t *testing.T) {
	pages, err := SentencePages(testRepoRoot())
	if err != nil {
		t.Fatalf("SentencePages: %v", err)
	}
	if len(pages) < 20 {
		t.Fatalf("SentencePages rendered %d pages; every top-level verb with a page has one", len(pages))
	}
	for _, p := range pages {
		if p.Committed != p.Want {
			t.Errorf("%s is stale against the sentence manifest; run `go generate ./internal/surface/cli`", p.File)
		}
	}
}

// TestWithDescriptionRewritesOnlyTheDescription: the page rewrite replaces the
// frontmatter's description line with the quoted sentence and leaves every other
// byte of the page alone, and a page with no description is refused rather than
// given one in a guessed place.
func TestWithDescriptionRewritesOnlyTheDescription(t *testing.T) {
	page := "---\nname: x\ndescription: Old words, by invoking the binary.\nblock: people\n---\n\n# x\n\ndescription: body text stays.\n"
	got, err := withDescription(page, `Do "x": Writes nothing; refuses y.`)
	if err != nil {
		t.Fatalf("withDescription: %v", err)
	}
	want := "---\nname: x\ndescription: \"Do \\\"x\\\": Writes nothing; refuses y.\"\nblock: people\n---\n\n# x\n\ndescription: body text stays.\n"
	if got != want {
		t.Fatalf("withDescription:\ngot:\n%s\nwant:\n%s", got, want)
	}
	if _, err := withDescription("---\nname: x\n---\n\n# x\n", "S: Writes nothing; refuses y."); err == nil {
		t.Fatal("a page with no description was rewritten instead of refused")
	}
	if _, err := withDescription("# x\n", "S: Writes nothing; refuses y."); err == nil {
		t.Fatal("a page with no frontmatter was rewritten instead of refused")
	}
}
