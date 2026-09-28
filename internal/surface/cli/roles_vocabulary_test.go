package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/mode"
	"github.com/spf13/cobra"
)

// The two people abcd addresses are the product thinker, who decides what to
// build, and the technical facilitator, who decides how
// (itd-2609212137129937). One word blurred them, and agents said it because
// abcd's own pages did. These tests hold the two surfaces an agent reads —
// the rendered help and the plugin pages — to naming the role.

// retiredRoleWord is spelled apart so this file never carries the word it
// refuses.
var retiredRoleWord = regexp.MustCompile(`(?i)\b` + "main" + "tainer")

// pluginPages are the markdown pages the plugin ships for an agent to read: the
// command pages and the agent prompts. The agent prompts' CHANGELOG is a dated
// history log whose entries keep their words, so it is not a page.
func pluginPages(t *testing.T) []string {
	t.Helper()
	var pages []string
	for _, glob := range []string{"commands/*.md", "agents/*.md"} {
		m, err := filepath.Glob(filepath.Join("..", "..", "..", filepath.FromSlash(glob)))
		if err != nil {
			t.Fatal(err)
		}
		for _, p := range m {
			if filepath.Base(p) == "CHANGELOG.md" {
				continue
			}
			pages = append(pages, p)
		}
	}
	if len(pages) < 20 {
		t.Fatalf("found %d plugin pages; the walk is not reading the plugin", len(pages))
	}
	return pages
}

// TestRenderedHelpAndPluginPagesNameTheRoles (AC5): every command's rendered
// help, and every plugin page, is free of the retired word.
func TestRenderedHelpAndPluginPagesNameTheRoles(t *testing.T) {
	root := NewRootCommand()
	var walk func(c *cobra.Command)
	n := 0
	walk = func(c *cobra.Command) {
		var buf bytes.Buffer
		c.SetOut(&buf)
		c.SetErr(&buf)
		if err := c.Help(); err != nil {
			t.Fatalf("%s: help did not render: %v", c.CommandPath(), err)
		}
		n++
		for i, line := range strings.Split(buf.String(), "\n") {
			if retiredRoleWord.MatchString(line) {
				t.Errorf("`%s --help` line %d names neither role: %q", c.CommandPath(), i+1, line)
			}
		}
		for _, child := range c.Commands() {
			walk(child)
		}
	}
	walk(root)
	if n < 20 {
		t.Fatalf("rendered %d help pages; the walk is not reading the command tree", n)
	}
	for _, p := range pluginPages(t) {
		data, err := os.ReadFile(p)
		if err != nil {
			t.Fatal(err)
		}
		for i, line := range strings.Split(string(data), "\n") {
			if retiredRoleWord.MatchString(line) {
				t.Errorf("%s:%d names neither role: %q", filepath.ToSlash(p), i+1, strings.TrimSpace(line))
			}
		}
	}
}

// questionBlock finds a paragraph in which a page tells the agent to put a
// question to a human: an imperative ask (sentence-initial, after "then" or
// "and", or after a bold lead-in) whose object is a person, "them" or the
// question itself; an instruction to relay or present a question; or a
// `--yes` that answers a question in advance, since pre-answering a stop is
// answering it and the paragraph must say whose answer that is. An ask whose
// object is a verb, a binary or another agent ("ask the update verb", "Ask it
// for kill attempts") is not a question to a human.
var questionBlock = regexp.MustCompile(`(?:(?:^|[.!?:;,—]\s+|\*\*\s*|\bthen\s+|\band\s+)(?:Ask|ask)\s+(?:once\b|whether\b|why\b|what\b|first\b|them\b|the (?:user|human|researcher|person|product thinker|technical facilitator)\b|for (?:the|every)\b)|\b(?:[Rr]elay|[Pp]resent) the question\b|` + "`--yes`" + `[^.]*\bin advance\b)`)

// TestPluginQuestionBlocksNameTheAddressee (AC3): every question a plugin page
// has the agent put to a human names which of the two roles it asks, in the
// paragraph that asks it, and the page's section sets the mode to that role —
// the addressee comes from the mode (itd-2609212130146198), whose vocabulary
// this test reads rather than restating.
func TestPluginQuestionBlocksNameTheAddressee(t *testing.T) {
	type role struct{ name, setter string }
	var roles []role
	for _, s := range mode.States() {
		if who := s.Addressee(); who != "" {
			roles = append(roles, role{name: who, setter: "mode " + string(s)})
		}
	}
	if len(roles) != 2 {
		t.Fatalf("the mode names %d roles, want the product thinker and the technical facilitator", len(roles))
	}
	found := 0
	for _, p := range pluginPages(t) {
		data, err := os.ReadFile(p)
		if err != nil {
			t.Fatal(err)
		}
		for _, sec := range pageSections(string(data)) {
			// A setter wrapped across two lines is still the setter.
			flat := strings.Join(strings.Fields(sec.text), " ")
			for _, para := range sec.paragraphs {
				if !questionBlock.MatchString(para.text) {
					continue
				}
				found++
				low := strings.ToLower(para.text)
				named := false
				for _, r := range roles {
					if !strings.Contains(low, r.name) {
						continue
					}
					named = true
					if !strings.Contains(flat, r.setter) {
						t.Errorf("%s:%d asks the %s but its section never sets `abcd %s` first",
							filepath.ToSlash(p), para.line, r.name, r.setter)
					}
				}
				if !named {
					t.Errorf("%s:%d puts a question to a human without naming the product thinker or the technical facilitator: %q",
						filepath.ToSlash(p), para.line, clipQuestion(para.text))
				}
			}
		}
	}
	if found < 10 {
		t.Fatalf("found %d question blocks; the detector is not reading the pages", found)
	}
}

type pageParagraph struct {
	line int
	text string
}

type pageSection struct {
	text       string
	paragraphs []pageParagraph
}

// pageSections splits a page at its headings, and each section into its
// blank-line paragraphs, skipping fenced code; a paragraph's text is its lines
// joined, so a question that wraps is read whole.
func pageSections(page string) []pageSection {
	var out []pageSection
	cur := pageSection{}
	var para []string
	start := 0
	flush := func() {
		if len(para) > 0 {
			cur.paragraphs = append(cur.paragraphs, pageParagraph{line: start, text: strings.Join(para, " ")})
			para = nil
		}
	}
	fence := false
	for i, line := range strings.Split(page, "\n") {
		trim := strings.TrimSpace(line)
		if !strings.HasPrefix(line, "#") || fence {
			cur.text += line + "\n"
		}
		if strings.HasPrefix(trim, "```") {
			flush()
			fence = !fence
			continue
		}
		if fence {
			continue
		}
		if strings.HasPrefix(line, "#") {
			flush()
			out = append(out, cur)
			cur = pageSection{}
			continue
		}
		if trim == "" {
			flush()
			continue
		}
		if len(para) == 0 {
			start = i + 1
		}
		para = append(para, trim)
	}
	flush()
	return append(out, cur)
}

func clipQuestion(s string) string {
	if len(s) > 160 {
		return s[:160] + "…"
	}
	return s
}
