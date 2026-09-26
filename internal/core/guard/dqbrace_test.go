package guard

import (
	"strings"
	"testing"
)

// TestDoubleQuotedBraceExpansionNestsQuotes — review5-guard finding 2. Inside
// double quotes, a `${…}` is parsed as bash parses it: a `"` in it opens a
// nested double-quoted string instead of closing the outer one, a `'` pairs
// with the next `'` (for the parse only; the substitutions inside still run),
// and the expansion ends at its own `}`. Reading the first nested `"` as the
// end of the string made an apostrophe inside the nested quotes an
// unterminated single quote, and the line — valid bash an agent writes —
// blocked as command-unparsable.
func TestDoubleQuotedBraceExpansionNestsQuotes(t *testing.T) {
	const push = "git push --force origin main"
	runVerdictCases(t, []verdictCase{
		{`echo "${MSG:-"don't"}"`, VerdictAllow, ""},
		{`echo "${1:-"it's"}"`, VerdictAllow, ""},
		{`printf '%s\n' "${NAME:-"O'Brien"}"`, VerdictAllow, ""},
		{`echo "${X//"'"/x}"`, VerdictAllow, ""},
		{`echo "${X#"'"}"`, VerdictAllow, ""},
		{`echo "${X-"'"}" "'"`, VerdictAllow, ""},
		{`echo "${X:-"${Y:-"it's"}"}"`, VerdictAllow, ""},
		{`echo "${X:-"}"} it's"`, VerdictAllow, ""},
		{`echo "${X:-"a\"b"}" "c'd"`, VerdictAllow, ""},
		{`x=$(echo "${MSG:-"don't"}")`, VerdictAllow, ""},
		{`x="$(echo "${MSG:-"don't"}")"`, VerdictAllow, ""},
		{"x=`echo \"${MSG:-\"don't\"}\"`", VerdictAllow, ""},
		{`git commit -m "${MSG:-"it's done"}"`, VerdictAllow, ""},
		// The nested quotes hold data, however hazardous it reads.
		{`echo "${X:-"; ` + push + `; "}"`, VerdictAllow, ""},
		{`echo "${X:-'}' ; ` + push + ` ; '}'}"`, VerdictAllow, ""},

		// The string still ends where bash ends it, and what follows it runs.
		{`echo "${X:-"a"}"; ` + push, VerdictBlock, "git-push-force"},
		{`echo "${X:-"it's"}" && ` + push, VerdictBlock, "git-push-force"},
		{`echo "${X:-'"'}"; ` + push, VerdictBlock, "git-push-force"},
		{`echo "${X:-"}"}"; ` + push, VerdictBlock, "git-push-force"},
		{`x=$(echo "${X:-"'"}"; ` + push + `)`, VerdictBlock, "git-push-force"},
		// A substitution inside the expansion runs, in nested quotes, in a
		// single-quoted span, or bare.
		{`echo "${X:-"$(` + push + `)"}"`, VerdictBlock, "git-push-force"},
		{`echo "${X:-'$(` + push + `)'}"`, VerdictBlock, "git-push-force"},
		{"echo \"${X:-'`" + push + "`'}\"", VerdictBlock, "git-push-force"},
		{`echo "${X:-"it's $(` + push + `)"}"`, VerdictBlock, "git-push-force"},
		{`echo "${X:-"${Y:-"$(` + push + `)"}"}"`, VerdictBlock, "git-push-force"},
		{`echo "${X:-"'"}$(` + push + `)"`, VerdictBlock, "git-push-force"},
		{`x=$(echo "${X:-"'"}$(` + push + `)")`, VerdictBlock, "git-push-force"},
		// A word that is wholly such an expansion is unknown from its `${`.
		{`git push "--${X:-"$(echo force)"}" origin main`, VerdictBlock, "git-push-force"},
	})
}

// TestDoubleQuotedBraceExpansionStaysLinear pins the cost class of the
// expansion's own parse: each `${` finds its `}` in one scan, and the text in
// it is read once more for the substitutions it runs.
func TestDoubleQuotedBraceExpansionStaysLinear(t *testing.T) {
	shapes := map[string]func(int) string{
		"nested quotes in many expansions": func(n int) string {
			return `echo "` + strings.Repeat(`${X:-"it's"} `, n) + `"`
		},
		"many strings with expansions": func(n int) string {
			return strings.Repeat(`x="${X:-"it's"}"; `, n)
		},
		"one expansion nested deep": func(n int) string {
			return `echo "` + strings.Repeat(`${X:-"`, n) + "it's" + strings.Repeat(`"}`, n) + `"`
		},
		"unclosed expansions": func(n int) string {
			return `echo "` + strings.Repeat(`${X:-'a' `, n) + `"`
		},
		"substitutions in single-quoted spans": func(n int) string {
			return `echo "${X:-` + strings.Repeat(`'$(a)' "b" `, n) + `}"`
		},
	}
	for name, build := range shapes {
		build := build
		t.Run(name, func(t *testing.T) {
			assertWorkGrowth(t, build, 1<<9, "each expansion is scanned for its end once, and read once")
		})
	}
}
