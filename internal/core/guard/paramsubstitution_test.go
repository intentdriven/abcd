package guard

import "testing"

// TestParameterExpansionCarryingASubstitutionIsUnknown — review4-guard finding
// 3. A `${…}` whose default, alternative or pattern holds a command
// substitution expands to that substitution's output, but the word was read
// with the expansion's closing `}` as fixed text after the output, so the name
// could only end in `}` and a dash-word only be a flag that did. A word whose
// text holds an unclosed `${` where a substitution's output lands is unknown
// from that `${` on. A plain `$X` or `${X}` with no substitution in it is the
// half iss-2609251824244354 still defers.
func TestParameterExpansionCarryingASubstitutionIsUnknown(t *testing.T) {
	runVerdictCases(t, []verdictCase{
		{`${X:-$(echo git)} push --force origin main`, VerdictBlock, ""},
		{`"${X:-$(echo git)}" push --force origin main`, VerdictBlock, ""},
		{`${X:+$(echo gh)} repo delete o/r`, VerdictBlock, ""},
		{`/usr/bin/${X:-$(echo git)} push --force origin main`, VerdictBlock, ""},
		{`${A}${X:-$(echo git)} push --force origin main`, VerdictBlock, ""},
		{`$(true)${X:-$(echo git)} push --force origin main`, VerdictBlock, ""},
		{"${X:-`echo git`} push --force origin main", VerdictBlock, ""},
		{`git push --${X:-$(echo force)} origin main`, VerdictBlock, "git-push-force"},
		{`git push "--${X:-$(echo force)}" origin main`, VerdictBlock, "git-push-force"},
		{`git push -${F:-$(echo f)} origin main`, VerdictBlock, "git-push-force"},
		{`git commit -m x --${X:-$(echo no-verify)}`, VerdictBlock, "git-commit-no-verify"},
		{`cd s && rm -${F:-$(echo rf)} *`, VerdictBlock, "rm-rf-after-cd-chain"},

		// Everyday parameter expansions with a substitution for a default.
		{`cd "${DIR:-$(pwd)}" && ls`, VerdictAllow, ""},
		{`echo "${1:-$(date)}"`, VerdictAllow, ""},
		{`git push origin "${BRANCH:-$(git branch --show-current)}"`, VerdictAllow, ""},
		{`ls "${HOME}/$(date +%F)"`, VerdictAllow, ""},
		{`"${EDITOR:-vi}" notes.md`, VerdictAllow, ""},
	})
}

// TestUnknownFromOpenExpansion pins the word-level rule: unknown from the
// outermost `${` still open where a mark lands, and nothing else changed.
func TestUnknownFromOpenExpansion(t *testing.T) {
	m := unknownText
	for in, want := range map[string]string{
		"${X:-" + m + "}":              m,
		"--${X:-" + m + "}":            "--" + m,
		"a${A}b${X:-${Y:-" + m + "}}c": "a${A}b" + m,
		"${A}" + m + "}":               "${A}" + m + "}",
		m + "${X:-" + m + "}":          m + m,
		"${X}":                         "${X}",
		"plain" + m:                    "plain" + m,
	} {
		if got := unknownFromOpenExpansion(in); got != want {
			t.Errorf("unknownFromOpenExpansion(%q) = %q, want %q", in, got, want)
		}
	}
}
