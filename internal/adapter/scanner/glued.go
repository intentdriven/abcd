package scanner

import (
	"regexp"
	"strings"
)

// gluedFindings finds the secret tokens ScanText cannot see because a word
// character sits right before them (iss-2609290541525428).
//
// Every bundled secret pattern anchors its start on a leading \b, and '_', a
// letter and a digit are all word characters, so a token glued behind one —
// `notes_<token>` in a key, `x<token>y`, `v2<token>` in a path — has no word
// boundary in front of it and the pattern never matches. The adjacency probes
// in scanAllPatterns recover a token that abuts a token already FOUND, and
// nothing recovers one that abuts ordinary text.
//
// The sweep is the page-name suffix sweep (memory's filenameJudgeTexts) carried
// to every position at once. Re-scanning each suffix that begins after a word
// character would find the same tokens at a cost quadratic in the text's
// length. Removing the leading \b from each pattern is the same test in one
// pass: an unanchored boundary-free pattern matches at every start position a
// suffix sweep would have tried, and each pattern stays one linear RE2 pass.
// The pass runs through scanAllPatterns with boundary-free probes and junction
// generators, so a run of glued tokens unwinds one junction at a time under the
// shared growth budget, and the whole sweep stays linear in the line.
//
// The sweep is narrower than ScanText by construction:
//   - only hard_fail secret patterns (secretPatterns: no identity or network
//     kind, whose looser shapes would match inside ordinary words);
//   - only patterns whose source opens on \b, because a pattern with no leading
//     boundary already matches a glued token in ScanText itself;
//   - each pattern's Skip and SkipAt still apply, so the documentation
//     example key stays accepted wherever it is glued.
//
// It does not change ScanText. The launch scan and judgeFilename keep the
// boundary they have; this is for callers that must never echo a token, where
// sealing a few more bytes of their own message costs nothing.
//
// ok is false when a pattern's boundary-free form will not compile: the sweep
// cannot vouch for the text then, and a caller that must not echo a token
// fails closed on it.
func gluedFindings(text string, patterns []Pattern, file string) (findings []Finding, ok bool) {
	glued, ok := gluedPatterns(patterns)
	if !ok {
		return nil, false
	}
	if len(glued) == 0 {
		return nil, true
	}
	probes := make([]matcher, len(glued))
	for i, p := range glued {
		probes[i] = adjacencyProbe(p.Re)
	}
	junctions := newJunctionSet(glued)
	for i, line := range strings.Split(text, "\n") {
		line = strings.TrimRight(line, "\r")
		for _, m := range scanAllPatterns(glued, probes, junctions, line) {
			p := glued[m.patIdx]
			matched := line[m.start:m.end]
			scanMeter.charge(stageSkip, len(matched))
			if p.Skip != nil && p.Skip(matched) {
				continue
			}
			if p.SkipAt != nil && p.SkipAt(line, m.start, m.end) {
				continue
			}
			findings = append(findings, Finding{
				File: file, Line: i + 1, Column: m.start + 1, Kind: p.Kind,
				Severity: p.Severity, Snippet: snippet(line), Matched: matched,
				Suggested: p.Suggestion, line: line,
			})
		}
	}
	return findings, true
}

// gluedPatterns is the sweep's pattern set: every hard_fail secret pattern
// whose source opens on \b (after an inline flag group), recompiled without
// that one anchor. A pattern that does not open on \b is left out: its bounded
// form already matches a glued token in ScanText. A boundary-free form that will
// not compile (a configured pattern whose \b carries a quantifier) makes the
// whole set not ok, never a silently narrower one.
func gluedPatterns(patterns []Pattern) ([]Pattern, bool) {
	var out []Pattern
	for _, p := range secretPatterns(patterns) {
		src := p.Re.String()
		flags := leadingFlagGroup.FindString(src)
		if !strings.HasPrefix(src[len(flags):], `\b`) {
			continue
		}
		re, err := regexp.Compile(flags + src[len(flags)+len(`\b`):])
		if err != nil {
			return nil, false
		}
		p.Re = re
		out = append(out, p)
	}
	return out, true
}
