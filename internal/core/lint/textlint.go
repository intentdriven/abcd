package lint

// The exported text-linting seam.
//
// The banned-token family (check A) is the linter's one rule that is about
// WORDS rather than about record shape, and it is the one rule another surface
// legitimately needs: `abcd lint site` runs the docs-lint bans over the text
// the website composes, which is selected out of several trees and never was a
// file the linter walks (adr-47 decision 3).
//
// The alternative was a second reader of `banned_tokens` living in the site
// package. Two readings of one ban is how a ban comes to mean two things — one
// of them silently weaker — so the compiled set is exported here instead, as a
// value the caller holds and runs. The compile is `compileTokens`, the match is
// `checkBannedTokens`, and the file walk in `LintAt` goes through this same
// type: there is one implementation, and this is a door onto it.

import "strings"

// TokenChecker is a compiled banned-token set, ready to run over text.
type TokenChecker struct {
	checks []tokenCheck
}

// NewTokenChecker compiles a configuration's banned_tokens. It refuses the same
// entries `LintAt` refuses, on the same compile, so a caller cannot arm a set
// the gate itself would not load.
func NewTokenChecker(tokens []BannedToken) (*TokenChecker, error) {
	checks, err := compileTokens(tokens)
	if err != nil {
		return nil, err
	}
	return &TokenChecker{checks: checks}, nil
}

// Len reports how many bans are armed. A checker with none is inert, and a
// caller that would otherwise walk a whole tree to find nothing can say so.
func (tc *TokenChecker) Len() int {
	if tc == nil {
		return 0
	}
	return len(tc.checks)
}

// lintLines is the file walk's entry: lines and their fence mask are already in
// hand there, and recomputing either would be a second answer to a question
// already asked.
func (tc *TokenChecker) lintLines(name string, lines []string, mask []bool) []Finding {
	if tc == nil {
		return nil
	}
	return checkBannedTokens(name, lines, mask, tc.checks)
}

// LintComposed lints text that has been LIFTED out of a source file — a span
// the website selected and rendered — and so no longer carries the source's
// line structure, its fence markers, or its line-level allow comments.
//
// The escape has to keep working across that lift, or every ban would harden
// silently the moment its text was composed. `source` is the source file's own
// lines, and a match is exempt when some source line both CONTAINS the matched
// text and carries the token's allow context: the same line-level judgement the
// file walk makes, made about the line the composed text actually came from.
//
// `fenced` says whether the text was rendered as a code block, so a ban that
// skips code fences in a file skips it here for the same reason. It is a
// parameter rather than something read back out of the text because composed
// text HAS no fence markers: looking for them would find a backtick that is
// content and silently stop checking from there, which is a gate passing by not
// looking.
//
// Passing a nil `source` is the strict reading — no line can grant an escape —
// which is what a caller with no source to point at should get.
func (tc *TokenChecker) LintComposed(name, text string, fenced bool, source []string) []Finding {
	if tc == nil {
		return nil
	}
	var out []Finding
	for i, line := range strings.Split(text, "\n") {
		for _, c := range tc.checks {
			if fenced && c.token.skipFences() {
				continue
			}
			loc := c.pattern.FindStringIndex(line)
			if loc == nil {
				continue
			}
			m := line[loc[0]:loc[1]]
			if matchesAny(c.allow, line) {
				continue
			}
			if allowedBySource(c, m, source) {
				continue
			}
			out = append(out, Finding{
				File: name, Line: i + 1, RuleID: c.token.ID,
				Severity: c.token.Severity, Message: bannedTokenMessage(c.token),
			})
		}
	}
	return out
}

// allowedBySource reports whether the source the composed text was lifted from
// grants this match its escape: some line carrying the matched text also
// carries the token's allow context. An empty match (a pattern that matched but
// consumed nothing) can be carried by no particular line, so it is never
// excused.
func allowedBySource(c tokenCheck, match string, source []string) bool {
	if match == "" || len(source) == 0 {
		return false
	}
	for _, ln := range source {
		if strings.Contains(ln, match) && matchesAny(c.allow, ln) {
			return true
		}
	}
	return false
}
