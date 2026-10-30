// Package issuemention is RS004's vocabulary in Go: the issue ids a text names,
// and the relation it declares to each with a `Resolves:` or `Refs:` line.
//
// scripts/check-issue-resolution.sh decides the rule (MENTION_RE, DECLARE_RE)
// and runs in CI without Go, so the two share a test rather than code
// (TestUndeclaredMatchesTheGate). This is its one Go copy, and a leaf: the
// shell guard judges a pull request's title and body with it before gh posts
// them (iss-2610261152125261), `abcd land` judges the body it composes with it,
// and capture's mentions listing reads history with it, and capture sits above
// the guard.
package issuemention

import (
	"regexp"
	"strings"
)

// mentionRe finds an issue id anywhere in a text. The leading guard is the
// gate's hand-rolled word boundary: `xiss-1` inside a longer token is not a
// mention of iss-1, and the greedy digit run keeps `iss-12` from reading as a
// mention of iss-1.
var mentionRe = regexp.MustCompile(`(?:^|[^A-Za-z0-9])(iss-[0-9]+)`)

// declareRe is the whole declaration vocabulary, a closed set of two
// spellings with nothing else on the line, its ids a comma-separated list of
// issue ids only. The gate's [[:space:]] is spelled out without the newline,
// since a grep line holds none and this pattern reads the whole text.
var declareRe = regexp.MustCompile(`(?m)^(Resolves|Refs):[ \t\v\f]+(iss-[0-9]+(?:[ \t\v\f]*,[ \t\v\f]*iss-[0-9]+)*)[ \t\v\f]*$`)

// declaredIDRe takes the ids back out of a declaration line's list.
var declaredIDRe = regexp.MustCompile(`iss-[0-9]+`)

// normalise drops every carriage return, as the gate does before it reads a
// line: a body typed into the forge's web form arrives CRLF-terminated.
func normalise(text string) string { return strings.ReplaceAll(text, "\r", "") }

// Mentioned returns every issue id text names, declarations included, once
// each, in the order first named.
func Mentioned(text string) []string {
	var out []string
	seen := map[string]bool{}
	for _, m := range mentionRe.FindAllStringSubmatch(normalise(text), -1) {
		if !seen[m[1]] {
			seen[m[1]] = true
			out = append(out, m[1])
		}
	}
	return out
}

// Declared returns the relation text declares to each id it declares one to:
// "Resolves" or "Refs". `Resolves:` outranks `Refs:` when a text carries both
// for one id, since the stronger claim is the one its author answers for.
func Declared(text string) map[string]string {
	out := map[string]string{}
	for _, m := range declareRe.FindAllStringSubmatch(normalise(text), -1) {
		for _, id := range declaredIDRe.FindAllString(m[2], -1) {
			if out[id] != "Resolves" {
				out[id] = m[1]
			}
		}
	}
	return out
}

// Undeclared returns the ids a pull request's title or body names that
// neither declares, once each, in the order first named: RS004 over a pull
// request, whose declaration set is read from both halves, since a title has
// no room for a trailer.
func Undeclared(title, body string) []string {
	declared := Declared(title + "\n" + body)
	var out []string
	for _, id := range Mentioned(title + "\n" + body) {
		if declared[id] == "" {
			out = append(out, id)
		}
	}
	return out
}
