package guard

import (
	"errors"
	"testing"
)

// TestSubshellHeredocIsNotArithmetic pins the line between the two shapes a `<<`
// followed by a paren can be. `( cmd <<EOF )` is a subshell opening a genuine
// here-document, which every shell here runs with the following lines as its
// document (probed on bash 3.2: the subshell runs, the body is data, and the
// line after the terminator runs). `$(( x << y ))` is an arithmetic shift whose
// right-hand identifier readHeredocDelim reads as a delimiter word, stopping at
// the expression's own closing paren.
//
// Testing only for "a paren after the delimiter" read the subshell as
// arithmetic, so the document was tokenized as commands: one apostrophe in it
// became ErrUnparsableCommand, which the pre-tool-use hook maps to fail-OPEN, so
// a blocked command ran unguarded; and a document merely NAMING a hazard warned
// as if it ran one. The discriminator is WHICH paren: arithmetic closes with
// `))` on the same line and has no body.
func TestSubshellHeredocIsNotArithmetic(t *testing.T) {
	cases := []struct {
		name    string
		line    string
		verdict Verdict
		entry   string
	}{
		{"apostrophe body in a subshell heredoc", "( gh api -X DELETE repos/owner/repo <<EOF )\nit's fine\nEOF", VerdictBlock, "gh-api-repo-delete"},
		{"body of a subshell heredoc is data", "( cat <<EOF )\ngit clean -fd\nEOF", VerdictAllow, ""},
		{"spaced arithmetic still parses", "echo $(( x << y ))\ngit push --force origin main", VerdictBlock, "git-push-force"},
		{"tight arithmetic still parses", "echo $((x<<y))\ngit push --force origin main", VerdictBlock, "git-push-force"},
		{"spaced arithmetic alone", "echo $(( x << y ))", VerdictAllow, ""},
		{"arithmetic command group", "(( x << y ))\ngit push --force origin main", VerdictBlock, "git-push-force"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			d, err := Defaults().Check(tc.line)
			if err != nil {
				t.Fatalf("Check(%q): %v — bash runs this line, so the guard must answer, not error", tc.line, err)
			}
			if d.Verdict != tc.verdict {
				t.Fatalf("Check(%q).Verdict = %q via %q, want %q", tc.line, d.Verdict, d.EntryID, tc.verdict)
			}
			if tc.entry != "" && d.EntryID != tc.entry {
				t.Fatalf("Check(%q).EntryID = %q, want %q", tc.line, d.EntryID, tc.entry)
			}
		})
	}
	// Where the shape stays ambiguous the answer is the fail-closed
	// heredoc-unterminated verdict, never an error the hook fails open on —
	// and asserting only "no error" would let a silent allow pass for it. All
	// four of these open a document whose delimiter line never comes, whether
	// the input runs out at a newline or at its last byte, so all four block.
	for _, line := range []string{
		"( cat <<EOF ) | tee out",
		"( cat <<EOF\t)",
		"x=$(cat <<EOF )\nit's",
		"x=$(cat <<EOF )",
	} {
		d, err := Defaults().Check(line)
		if err != nil {
			t.Errorf("Check(%q): %v — an ambiguous `<<` must take a verdict, never an error", line, err)
			continue
		}
		if d.Verdict != VerdictBlock || d.EntryID != heredocEntryID {
			t.Errorf("Check(%q) = %q via %q, want the fail-closed block via %q", line, d.Verdict, d.EntryID, heredocEntryID)
		}
	}
}

// TestHeredocBodyStartsDespiteTrailingListOperator pins where a here-document
// body begins: on the line after the redirection, always. bash collects the
// bodies at the end of the PHYSICAL line, so a trailing `&&`, `||` or `|` does
// not defer them — probed on bash 3.2, `cat <<EOF &&` / body / `EOF` / `echo ok`
// prints ok and never runs the body.
//
// Deferring the body until the command list completed put document text in
// command position twice over: an apostrophe in a document became
// ErrUnparsableCommand and the hook failed OPEN, and the scan for the terminator
// started past an early delimiter line, so a stray delimiter at the end
// swallowed the real commands between them as body — a SILENT allow.
func TestHeredocBodyStartsDespiteTrailingListOperator(t *testing.T) {
	const line = "cat <<EOF &&\ndon't\nEOF\necho ok"
	if _, err := Defaults().Check(line); err != nil {
		t.Fatalf("Check(%q): %v — bash runs this; the body starts on the next line regardless of the `&&`", line, err)
	}
	const hazard = "cat <<EOF &&\ndon't\nEOF\ngit push --force origin main"
	d, err := Defaults().Check(hazard)
	if err != nil {
		t.Fatalf("Check(%q): %v", hazard, err)
	}
	if d.Verdict != VerdictBlock || d.EntryID != "git-push-force" {
		t.Fatalf("Check(%q) = %q via %q, want block via git-push-force", hazard, d.Verdict, d.EntryID)
	}
	const swallowed = "cat <<EOF &&\nEOF\ngit push --force origin main\nEOF"
	d, err = Defaults().Check(swallowed)
	if err != nil {
		t.Fatalf("Check(%q): %v", swallowed, err)
	}
	if d.Verdict != VerdictBlock || d.EntryID != "git-push-force" {
		t.Fatalf("Check(%q) = %q via %q, want block via git-push-force", swallowed, d.Verdict, d.EntryID)
	}
	// What stays an error is what no shell runs: an unterminated quote in
	// COMMAND text. A quote inside a body is document text and is not one.
	if _, err := tokenize(`echo 'unterminated`); !errors.Is(err, ErrUnparsableCommand) {
		t.Fatalf("an unterminated quote in command text must stay ErrUnparsableCommand")
	}
}

// TestParenthesisedArithmeticIsNotAHeredoc pins the discriminator between a
// shift and a here-document as the ENCLOSING CONTEXT, not the bytes that follow
// the delimiter word. `$(( (1 << n) + 1 ))` closes its parenthesised
// sub-expression with a SINGLE `)`, so a rule reading "arithmetic closes with
// `))` right after the operand" took `n` for a delimiter word — and then a later
// line equal to `n` swallowed every command between as document text (a silent
// allow of a Tier-1 hazard), while with no such line an ordinary bit-mask script
// blocked under heredoc-unterminated.
//
// A `<<` reached while an arithmetic context is open is a shift, always: bash is
// inside `$((`/`((` and no redirection is possible there. A plain `(` is not an
// arithmetic opener — `( cmd <<EOF )` is a genuine subshell here-document — and
// a `$(`/backtick between the `<<` and an outer arithmetic frame starts a fresh
// command string, where a here-document is possible again.
func TestParenthesisedArithmeticIsNotAHeredoc(t *testing.T) {
	cases := []struct {
		name    string
		line    string
		verdict Verdict
		entry   string
	}{
		{
			"parenthesised shift with a later line equal to its operand",
			"echo $(( (1 << n) + 1 ))\ngit push --force origin main\nn",
			VerdictBlock, "git-push-force",
		},
		{
			"parenthesised shift over a hazard with its operand below",
			"x=$(( (a << b) | 1 ))\ngit clean -fd\nb",
			VerdictWarn, "git-clean",
		},
		{
			"an ordinary bit mask is not a document",
			"mask=$(( (1 << n) - 1 ))\necho done",
			VerdictAllow, "",
		},
		{
			"parenthesised shift in the bare arithmetic command form",
			"if (( (1 << n) > 4 )); then echo big; fi",
			VerdictAllow, "",
		},
		{
			"a here-document after a CLOSED arithmetic is still a document",
			"echo $(( 1 << 2 ))\ncat <<EOF\ngit clean -fd\nEOF",
			VerdictAllow, "",
		},
		{
			"a here-document inside a command substitution inside arithmetic",
			"echo $(( $(cat <<EOF) ))\ngit clean -fd\nEOF",
			VerdictAllow, "",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			d, err := Defaults().Check(tc.line)
			if err != nil {
				t.Fatalf("Check(%q): %v — bash runs this line, so the guard must answer, not error", tc.line, err)
			}
			if d.Verdict != tc.verdict {
				t.Fatalf("Check(%q).Verdict = %q via %q, want %q", tc.line, d.Verdict, d.EntryID, tc.verdict)
			}
			if d.EntryID != tc.entry {
				t.Fatalf("Check(%q).EntryID = %q, want %q", tc.line, d.EntryID, tc.entry)
			}
		})
	}
}

// TestNonAlphabeticHeredocDelimiterOpensADocument pins the delimiter WORD.
// bash accepts any word after `<<` — `cat <<20` and `cat <<$D` are ordinary
// here-documents, and `let mask=1<<20` is one too, because the shell tokenises
// the redirection before the `let` builtin ever sees the arithmetic. The guard
// read only a letter- or underscore-led word as a delimiter, so a `<<` followed
// by anything else fell to the shift branch and the DOCUMENT was tokenised as
// commands: one apostrophe in the body became ErrUnparsableCommand, which the
// pre-tool-use hook maps to fail-OPEN, and a hazard on a later line ran
// unguarded. That also made the two front doors' own claim — "a quote inside a
// here-document body is document text and is not unparsable" — false for every
// delimiter the heuristic rejected.
//
// The word set is the conservative superset bash allows: letters, digits, `_`
// and a `$`-led word (whose value the guard cannot know, so the document never
// terminates and the fail-closed block answers). What still decides a shift is
// the ENCLOSING context, which is where the discriminator belongs.
func TestNonAlphabeticHeredocDelimiterOpensADocument(t *testing.T) {
	cases := []struct {
		name    string
		line    string
		verdict Verdict
		entry   string
	}{
		{
			"a numeric delimiter, an apostrophe in the body, a hazard below",
			"cat <<20\nit's a doc\n20\ngit push --force origin main",
			VerdictBlock, "git-push-force",
		},
		{
			"a numeric delimiter whose body is only data",
			"cat <<20\nit's a doc\n20\necho done",
			VerdictAllow, "",
		},
		{
			"a delimiter the guard cannot evaluate never terminates",
			"cat <<$D\nit's a doc\ngit push --force origin main",
			VerdictBlock, "heredoc-unterminated",
		},
		{
			"let tokenises as a redirection before the builtin sees it",
			"let mask=1<<20\necho done",
			VerdictBlock, "heredoc-unterminated",
		},
		{
			"an underscore-led delimiter, unchanged",
			"cat <<_doc\nit's a doc\n_doc\ngit push --force origin main",
			VerdictBlock, "git-push-force",
		},
		{
			"an arithmetic shift is still a shift",
			"echo $((1<<20))\necho done",
			VerdictAllow, "",
		},
		{
			"a parenthesised arithmetic shift is still a shift",
			"mask=$(( (1 << 20) - 1 ))\necho done",
			VerdictAllow, "",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			d, err := Defaults().Check(tc.line)
			if err != nil {
				t.Fatalf("Check(%q): %v — bash runs this line, so the guard must answer, not error", tc.line, err)
			}
			if d.Verdict != tc.verdict || d.EntryID != tc.entry {
				t.Fatalf("Check(%q) = %q via %q, want %q via %q", tc.line, d.Verdict, d.EntryID, tc.verdict, tc.entry)
			}
		})
	}
}
