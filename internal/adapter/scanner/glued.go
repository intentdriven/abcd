package scanner

import (
	"regexp"
	"strings"
)

// gluedSweep finds the secret tokens the bounded patterns cannot see because a
// word character sits right before them (iss-2609290541525428).
//
// Every bundled secret pattern anchors its start on a leading \b, and '_', a
// letter and a digit are all word characters, so a token glued behind one —
// `notes_<token>` in a key, `x<token>y`, `v2<token>` in a path — has no word
// boundary in front of it and the pattern never matches. The adjacency probes
// in scanAllPatterns recover a token that abuts a token already FOUND; nothing
// recovered one that abuts ordinary text.
//
// The sweep is the page-name suffix sweep (memory's filenameJudgeTexts) carried
// to every position at once. Re-scanning each suffix that begins after a word
// character would find the same tokens at a cost quadratic in the line's
// length. Removing the leading \b from each pattern is the same test in one
// pass: an unanchored boundary-free pattern matches at every start position a
// suffix sweep would have tried, and each pattern stays one linear RE2 pass.
// The pass runs through scanAllPatterns with boundary-free probes and junction
// generators, so a run of glued tokens unwinds one junction at a time under the
// shared growth budget, and the sweep stays linear in the line.
//
// The sweep is narrower than the bounded scan by construction:
//   - only hard_fail secret patterns (secretPatterns: no identity or network
//     kind, whose looser shapes would match inside ordinary words);
//   - only patterns whose source opens on \b, because a pattern with no leading
//     boundary already matches a glued token in the bounded pass;
//   - each pattern's Skip and SkipAt still apply, so the documentation example
//     key stays accepted wherever it is glued.
//
// A glued finding at the span a bounded one already holds is the same finding,
// and scanText's dedupFindings keeps it once, so a bounded token keeps the one
// report and the fingerprint it always had.
type gluedSweep struct {
	patterns  []Pattern
	probes    []matcher
	junctions junctionSet
	// unbuilt names each pattern whose boundary-free form would not compile.
	// The sweep then runs the patterns it could build — never narrower than
	// the bounded scan alone — and New reports the gap as a degraded scanner
	// (Unavailable), so every write-time redactor, the launch scan and
	// RedactRefusal fail closed on it rather than trust a narrower sweep
	// (iss-2609290743362554).
	unbuilt []string
}

// newGluedSweep builds the sweep for a pattern set.
func newGluedSweep(patterns []Pattern) gluedSweep {
	glued, unbuilt := gluedPatterns(patterns)
	g := gluedSweep{patterns: glued, unbuilt: unbuilt}
	if len(glued) == 0 {
		return g
	}
	g.probes = make([]matcher, len(glued))
	for i, p := range glued {
		g.probes[i] = adjacencyProbe(p.Re)
	}
	g.junctions = newJunctionSet(glued)
	return g
}

// findings returns the sweep's findings on one line, in scanText's shape.
func (g gluedSweep) findings(line string, lineno int, file string) []Finding {
	if len(g.patterns) == 0 {
		return nil
	}
	var out []Finding
	for _, m := range scanAllPatterns(g.patterns, g.probes, g.junctions, line) {
		p := g.patterns[m.patIdx]
		matched := line[m.start:m.end]
		scanMeter.charge(stageSkip, len(matched))
		if p.Skip != nil && p.Skip(matched) {
			continue
		}
		if p.SkipAt != nil && p.SkipAt(line, m.start, m.end) {
			continue
		}
		out = append(out, Finding{
			File: file, Line: lineno, Column: m.start + 1, Kind: p.Kind,
			Severity: p.Severity, Snippet: snippet(line), Matched: matched,
			Suggested: p.Suggestion, line: line,
		})
	}
	return out
}

// gluedFindings runs the sweep alone over text, line by line — the raw line
// and each of its decoded views, as scanText runs it; ok is false when a
// pattern's boundary-free form could not be built. The cost guard and the fail-closed test read it.
func gluedFindings(text string, patterns []Pattern, file string) (findings []Finding, ok bool) {
	g := newGluedSweep(patterns)
	for i, line := range strings.Split(text, "\n") {
		line = strings.TrimRight(line, "\r")
		findings = append(findings, g.findings(line, i+1, file)...)
		if len(g.patterns) == 0 {
			continue
		}
		for _, v := range lineViews(line) {
			findings = append(findings, viewTokenFindings(g.patterns, g.probes, g.junctions, line, v, i+1, file)...)
		}
	}
	return findings, len(g.unbuilt) == 0
}

// gluedPatterns is the sweep's pattern set: every hard_fail secret pattern
// whose source opens on \b (after an inline flag group), recompiled without
// that one anchor. A pattern that does not open on \b is left out: its bounded
// form already matches a glued token in ScanText. A boundary-free form that will
// not compile (a configured pattern whose \b carries a quantifier) is left out
// and named in unbuilt, never a silently narrower set.
func gluedPatterns(patterns []Pattern) (out []Pattern, unbuilt []string) {
	for _, p := range secretPatterns(patterns) {
		src := p.Re.String()
		flags := leadingFlagGroup.FindString(src)
		if !strings.HasPrefix(src[len(flags):], `\b`) {
			continue
		}
		re, err := regexp.Compile(flags + src[len(flags)+len(`\b`):])
		if err != nil {
			unbuilt = append(unbuilt, p.Name)
			continue
		}
		p.Re = re
		out = append(out, p)
	}
	return out, unbuilt
}
