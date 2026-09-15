package site

// The glossary page set, and the term links that reach it.
//
// The glossary names every sense of a colliding word, and a reader on the site
// could not reach the entry that does it (iss-2609020922150364). These pin both
// halves of the answer: the entries are published as pages, and a record page's
// FIRST use of a term is a link to the entry — while a use inside a code span,
// a heading or an existing link is left exactly as the record wrote it.

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/glossary"
)

// glossaryTermFile writes one term file in the shape the glossary directory
// keeps: an attribution comment, the index-bearing frontmatter, and a body.
func glossaryTermFile(term, context, definition, aliases, body string) string {
	return "<!-- A fixture term file. -->\n---\nterm: " + term +
		"\nbounded_context: " + context +
		"\ndefinition: " + definition +
		"\naliases: " + aliases +
		"\nforbidden_synonyms: []\nstatus: stable\n---\n\n# " + term + "\n\n" + body
}

// withGlossary gives a fixture a two-term glossary and a record whose body uses
// both words — once where a link belongs, and three times where one does not.
func withGlossary(t *testing.T, f *fixture) {
	t.Helper()
	f.write(".abcd/development/brief/glossary/README.md", "# Glossary\n\nThe fixture glossary.\n")
	f.write(".abcd/development/brief/glossary/core/README.md", "# core\n\nThe core context.\n")
	f.write(".abcd/development/brief/glossary/core/phase.md", glossaryTermFile(
		"phase", "core", "An ordered stretch of work that ends in a milestone.", `[]`,
		"A phase is the sequencing layer, and this sentence uses the word again.\n\nIt sits beside the [spec](spec.md).\n"))
	f.write(".abcd/development/brief/glossary/core/spec.md", glossaryTermFile(
		"spec", "core", "The buildable statement of one intent.", `["specification"]`,
		"A spec says what a phase will contain.\n"))

	f.write(".abcd/development/decisions/adrs/0001-the-first-decision.md", `---
id: adr-1
slug: the-first-decision
status: accepted
date: 2026-01-02
supersedes: null
superseded_by: null
related_intents: [itd-1]
related_rfcs: []
related_adrs: [adr-2]
---

# ADR-1: The first decision

The first decision's body, which mentions iss-1 in prose. A phase is the unit of
sequencing, and a second phase repeats the word. A specification is what an
alias reaches.

## A heading naming the phase

A `+"`phase`"+` in a code span, and a [phase](https://example.invalid/) that is
already a link, are both left as the record wrote them.
`)
}

// TestGlossaryIsExportedAsItsOwnPageSet is the first half of the answer: the
// glossary the record keeps becomes an index and one page per entry, reachable
// from the explorer's own navigation.
func TestGlossaryIsExportedAsItsOwnPageSet(t *testing.T) {
	f := newFixture(t)
	withGlossary(t, f)
	out := t.TempDir()
	res := buildFixture(t, f, out)

	for _, route := range []string{
		"record/glossary/index.html",
		"record/glossary/core/phase/index.html",
		"record/glossary/core/spec/index.html",
	} {
		if !containsString(res.Files, route) {
			t.Errorf("the build wrote no %s", route)
		}
	}

	index := outFile(t, out, "record/glossary/index.html")
	for _, want := range []string{
		`href="/record/glossary/core/phase/"`,
		`href="/record/glossary/core/spec/"`,
		"An ordered stretch of work that ends in a milestone.",
	} {
		if !strings.Contains(index, want) {
			t.Errorf("the glossary index does not carry %q", want)
		}
	}

	// The tab strip reaches it, on every explorer page rather than only its own.
	dash := outFile(t, out, "record/index.html")
	if !strings.Contains(dash, `href="/record/glossary/"`) {
		t.Error("the explorer's navigation does not reach the glossary")
	}

	term := outFile(t, out, "record/glossary/core/phase/index.html")
	for _, want := range []string{
		"An ordered stretch of work that ends in a milestone.",
		"A phase is the sequencing layer",
		// A term file's link to a sibling term reaches that term's page rather
		// than the forge.
		`href="/record/glossary/core/spec/"`,
	} {
		if !strings.Contains(term, want) {
			t.Errorf("the phase entry page does not carry %q", want)
		}
	}
	// A term never links to itself: the reader is already there.
	if strings.Contains(term, `class="gterm" href="/record/glossary/core/phase/"`) {
		t.Error("the phase entry page links its own word to itself")
	}
	// A term file carries an attribution comment ABOVE its frontmatter (the
	// template it was adapted from does), and a stripper that only recognises
	// frontmatter at byte zero publishes the whole block as prose — the fields
	// arriving on the page as a heading nobody wrote.
	if strings.Contains(term, "forbidden_synonyms") {
		t.Error("the entry page published the term file's frontmatter as prose")
	}
}

// TestRecordPageLinksTheFirstTermOccurrenceOnly is the second half: the reader
// meets the word and can reach the entry, once, and the rest of the page is
// exactly what the record wrote.
func TestRecordPageLinksTheFirstTermOccurrenceOnly(t *testing.T) {
	f := newFixture(t)
	withGlossary(t, f)
	out := t.TempDir()
	buildFixture(t, f, out)

	page := outFile(t, out, "record/adr/adr-1/index.html")
	link := `<a class="gterm" href="/record/glossary/core/phase/">phase</a>`
	if n := strings.Count(page, link); n != 1 {
		t.Errorf("the page carries %d links to the phase entry, want exactly 1", n)
	}
	if !strings.Contains(page, "A "+link+" is the unit of") {
		t.Error("the FIRST use of the term is not the one that was linked")
	}
	if !strings.Contains(page, "a second phase repeats the word") {
		t.Error("a later use of the term was linked as well as the first")
	}
	// An alias reaches the same entry.
	if !strings.Contains(page, `<a class="gterm" href="/record/glossary/core/spec/">specification</a>`) {
		t.Error("an alias of a term does not reach the term's entry")
	}
	// Three places a link must never appear.
	if !strings.Contains(page, `<code>phase</code>`) {
		t.Error("the code span was rewritten")
	}
	if !strings.Contains(page, `>A heading naming the phase</h2>`) {
		t.Error("the heading was rewritten")
	}
	if !strings.Contains(page, `<a href="https://example.invalid/">phase</a>`) {
		t.Error("a word already inside a link was rewritten")
	}
}

// TestARepositoryWithNoGlossaryRendersNone is graceful absence (itd-140): the
// page set and its navigation entry are omitted, and no record page grows a
// link to a glossary that is not there.
func TestARepositoryWithNoGlossaryRendersNone(t *testing.T) {
	f := newFixture(t)
	if err := os.RemoveAll(filepath.Join(f.Root(),
		filepath.FromSlash(".abcd/development/brief/glossary"))); err != nil {
		t.Fatal(err)
	}
	out := t.TempDir()
	res := buildFixture(t, f, out)

	for _, file := range res.Files {
		if strings.Contains(file, "record/glossary/") {
			t.Errorf("a repository with no glossary got %s", file)
		}
	}
	dash := outFile(t, out, "record/index.html")
	if strings.Contains(dash, `href="/record/glossary/"`) {
		t.Error("the navigation points at a glossary page the build did not write")
	}
	if strings.Contains(outFile(t, out, "record/adr/adr-1/index.html"), "gterm") {
		t.Error("a record page carries a term link with no glossary to reach")
	}
}

// TestOneLinkPerEntryPerPage pins WHICH first occurrence is meant. A term and
// its aliases are spellings of one entry, so a page that says "spec" and then
// "specification" carries one link, not two: the reader who followed the first
// already knows what the second means. The real glossary makes the difference
// concrete — "user", "installer" and "plugin consumer" are three spellings of
// the one end-user entry, and a page linking each of them three times is the
// noise the first-occurrence rule exists to prevent.
func TestOneLinkPerEntryPerPage(t *testing.T) {
	l := newTermLinker([]glossaryEntry{
		{
			Term:  glossary.Term{Name: "spec", Aliases: []string{"specification", "implementation spec"}},
			Route: "record/glossary/core/spec/",
		},
		{
			Term:  glossary.Term{Name: "phase", Aliases: []string{"roadmap phase"}},
			Route: "record/glossary/core/phase/",
		},
	})
	got := l.link("<p>A spec is a specification, and an implementation spec too. "+
		"A roadmap phase is a phase.</p>", "")
	want := `<p>A <a class="gterm" href="/record/glossary/core/spec/">spec</a> is a ` +
		`specification, and an implementation spec too. ` +
		`A <a class="gterm" href="/record/glossary/core/phase/">roadmap phase</a> is a phase.</p>`
	if got != want {
		t.Errorf("link() =\n%s\nwant\n%s", got, want)
	}
}
