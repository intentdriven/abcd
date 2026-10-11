package cli

import (
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"
	"unicode/utf8"

	"github.com/intentdriven/abcd/internal/core/frontmatter"
	"github.com/intentdriven/abcd/internal/core/surface"
	"github.com/spf13/cobra"
)

// sentences.go — itd-2609212113220149: every visible verb and sub-verb opens with
// one sentence naming what it does, what it writes and when it refuses. The
// sentence is declared once, in the surface manifest (internal/core/surface,
// sentences.go), and this front door renders it in every place a reader meets
// the verb: as the command's Short it is the line the parent's command list,
// the root's groups and the agents block print; the help template puts it first
// in the verb's own --help; and the generator writes it as the `description:`
// of the verb's plugin page (SentencePages). No constructor declares a Short
// for a visible command, so there is no second copy to drift.
//
// A page that carries a host workflow its bare verb does not describe also
// declares "use when" text in pageUseWhen, which the generator appends to that
// page's description after the sentence. It is the page's alone: the command
// list, the help and the reference keep the bare sentence.
//
// The gate is internal/surface/cli/sentences_test.go: it walks every visible
// command and fails naming the verb and the defect when a sentence is missing,
// is not of the form, runs past the cap, or differs between those places.

// verbHelpTemplate is cobra's help template with one change: the sentence
// opens the help and the long help follows it, where cobra prints the long help
// INSTEAD of the short line whenever there is one. The root's help is
// renderRootHelp, which opens the same way.
const verbHelpTemplate = `{{with .Short}}{{. | trimTrailingWhitespaces}}

{{end}}{{with .Long}}{{. | trimTrailingWhitespaces}}

{{end}}{{if or .Runnable .HasSubCommands}}{{.UsageString}}{{end}}`

// applySentences sets every command's Short from the manifest and installs the
// help template that opens each verb's help with it. A command the manifest has
// no sentence for keeps whatever it declares, which for a visible command is
// nothing, so the gate names it.
func applySentences(root *cobra.Command, lookup func(path string) (string, bool)) {
	var walk func(*cobra.Command)
	walk = func(c *cobra.Command) {
		if s, ok := lookup(c.CommandPath()); ok {
			c.Short = s
		}
		for _, sub := range c.Commands() {
			walk(sub)
		}
	}
	walk(root)
	root.SetHelpTemplate(verbHelpTemplate)
}

// UseWhenCap is the most characters a page's use-when text may run to. The
// longest use-when clause of the deleted pages whose trigger phrases it carries
// ran to about 200.
const UseWhenCap = 300

// pageUseWhen maps a plugin page's name to the "use when" text appended to its
// description after the verb's sentence. The host picks a page to load from its
// description, and the verb split (itd-2610090831227812) folded host
// workflows into pages whose bare verb describes something else: library
// carries the register and consult workflows of the deleted /abcd:ingest and
// /abcd:consult pages, and ahoy's install carries the deleted
// /abcd:prepare-this-repo workflow. Each entry carries the trigger phrases of
// the deleted pages' descriptions. Registering a document is `library add`;
// `library ingest` distils a source into the memory store and registers
// nothing (iss-2610110233253365), so library's text names the former.
var pageUseWhen = map[string]string{
	"ahoy": "Use when the user asks to install abcd in a repository, or to prepare, onboard, scaffold or " +
		"bootstrap a repository for agent work or bring it up to current conventions (that is ahoy install).",
	"library": "Use when the user says 'ingest this', 'add this source/URL/paper to the corpus' or " +
		"'register this source' (that is library add), or 'consult sources', 'check the corpus' or " +
		"'what do my sources say'.",
}

// pageDescription is the description of the page named page: the verb's
// sentence, followed by the page's use-when text where useWhen declares one.
func pageDescription(useWhen map[string]string, page, sentence string) string {
	if when, ok := useWhen[page]; ok {
		return sentence + " " + when
	}
	return sentence
}

// checkUseWhen refuses use-when text that is not one line opening with
// "Use when " and closing with a period, within UseWhenCap characters.
func checkUseWhen(when string) error {
	switch {
	case strings.ContainsAny(when, "\r\n"):
		return errors.New("the use-when text runs over more than one line")
	case !strings.HasPrefix(when, "Use when "):
		return errors.New(`the use-when text does not open with "Use when "`)
	case !strings.HasSuffix(when, "."):
		return errors.New("the use-when text does not close with a period")
	case utf8.RuneCountInString(when) > UseWhenCap:
		return fmt.Errorf("the use-when text is %d characters, over the cap of %d", utf8.RuneCountInString(when), UseWhenCap)
	}
	return nil
}

// SentencePage is one plugin command page: its repo-relative path, the bytes
// committed, and the bytes it holds once its description is the verb's
// sentence and the page's use-when text, if it declares one. The generator
// writes Want where the two differ, and the drift test fails on any page where
// they do.
type SentencePage struct {
	File      string
	Committed string
	Want      string
}

// SentencePages renders every plugin command page that backs a visible verb of
// the tree: commands/<verb>.md for each visible top-level verb, and the root's
// own page, commands/abcd.md. A verb with no page (abcd has verbs no page
// documents on its own) contributes nothing, and a page with no verb (a page
// whose work runs in the host) is not this manifest's to rewrite.
//
// A page that exists but carries no frontmatter description is an error naming
// the page, rather than a description inserted at a guessed position.
func SentencePages(repoRoot string) ([]SentencePage, error) {
	root := NewRootCommand()
	cmds := []*cobra.Command{root}
	for _, sub := range root.Commands() {
		if !sub.Hidden && sub.Deprecated == "" {
			cmds = append(cmds, sub)
		}
	}
	var out []SentencePage
	for _, cmd := range cmds {
		sentence, ok := surface.SentenceFor(cmd.CommandPath())
		if !ok {
			continue
		}
		rel := path.Join("commands", cmd.Name()+".md")
		data, err := os.ReadFile(filepath.Join(repoRoot, filepath.FromSlash(rel)))
		if errors.Is(err, os.ErrNotExist) {
			continue
		}
		if err != nil {
			return nil, err
		}
		if when, ok := pageUseWhen[cmd.Name()]; ok {
			if err := checkUseWhen(when); err != nil {
				return nil, fmt.Errorf("%s: %w", rel, err)
			}
		}
		want, err := withDescription(string(data), pageDescription(pageUseWhen, cmd.Name(), sentence))
		if err != nil {
			return nil, fmt.Errorf("%s: %w", rel, err)
		}
		out = append(out, SentencePage{File: rel, Committed: string(data), Want: want})
	}
	return out, nil
}

// withDescription returns page with its frontmatter `description:` line
// replaced by the description, double-quoted because every sentence carries a
// colon followed by a space, which a plain YAML scalar may not. Nothing else in
// the page changes. The block's extent is frontmatter.Close's, so a leading
// `---` nothing closes is no frontmatter here exactly as it is to the reader.
func withDescription(page, description string) (string, error) {
	lines := strings.Split(page, "\n")
	closing := frontmatter.Close(lines)
	if closing < 0 {
		return "", errors.New("the page has no frontmatter to carry the sentence")
	}
	for i := 1; i < closing; i++ {
		if strings.HasPrefix(lines[i], "description:") {
			lines[i] = "description: " + frontmatter.QuoteScalar(description)
			return strings.Join(lines, "\n"), nil
		}
	}
	return "", errors.New("the page's frontmatter has no description line to carry the sentence")
}

// applyExamples sets every command's Example from the manifest's worked
// examples (iss-2609100508565741), indented the way cobra's usage template
// prints an example block, so it renders under the verb's own --help and in
// the generated CLI reference.
func applyExamples(root *cobra.Command, lookup func(path string) (string, bool)) {
	var walk func(*cobra.Command)
	walk = func(c *cobra.Command) {
		if e, ok := lookup(c.CommandPath()); ok {
			c.Example = "  " + e
		}
		for _, sub := range c.Commands() {
			walk(sub)
		}
	}
	walk(root)
}
