package lint

import "strconv"

// ruleBriefFlagLanded keeps the brief's review flags naming text the brief
// carries. `docs fidelity --apply` writes a reviewer's drafted replacement into
// its chapter and flags it in .abcd/work/brief-review-flags.json for the
// product thinker to read; a person then tidies the sentence by hand before
// committing, and the flag went on naming the draft. In the v0.13.0 cut five
// chapters landed wording that differs from their flags, so the record of what
// the product thinker must read named sentences that are not in the brief
// (iss-2610050259233425). This rule is that drift reported at the change that
// makes it: a flag whose replacement no line of its chapter contains is a
// finding until the flag records the sentence as committed.
const ruleBriefFlagLanded = "brief_flag_landed"

// briefFlagsFile is where the flags are kept (docfidelity.FlagsPath). The rule
// names it only when no reader is registered; every other finding takes its
// file from the reader.
const briefFlagsFile = ".abcd/work/brief-review-flags.json"

// BriefFlagMiss is one review flag whose replacement no line of its chapter
// carries: the flag's line in File, the chapter it names, its replacement, and
// why the chapter does not carry it.
type BriefFlagMiss struct {
	File        string
	Line        int
	Chapter     string
	Replacement string
	Why         string
}

// briefFlagMisses is the doc-fidelity package's reading of the flags against
// their chapters, docfidelity.UnlandedFlags, registered by the front doors that
// run this gate (cmd/record-lint, and the CLI for `abcd lint`), because this
// package cannot import core/docfidelity: docfidelity reads its saved reviews
// through this package, and Go refuses the cycle. It is the one reader of the
// flags file. Unregistered, brief_flag_landed says so in one finding instead
// of reporting nothing.
var briefFlagMisses func(repoRoot string) ([]BriefFlagMiss, error)

// SetBriefFlagCheck registers the reader brief_flag_landed asks. Pass
// docfidelity.UnlandedFlags.
func SetBriefFlagCheck(fn func(repoRoot string) ([]BriefFlagMiss, error)) {
	briefFlagMisses = fn
}

// checkBriefFlagLanded implements brief_flag_landed: one finding per flag whose
// replacement no line of its chapter contains, on the flag's line. A flags file
// the reader cannot read is a configuration error: an armed gate with nothing
// to read is never a pass.
func checkBriefFlagLanded(repoRoot string, rc RuleConfig) ([]Finding, error) {
	if briefFlagMisses == nil {
		return []Finding{{
			File: briefFlagsFile, Line: 0, RuleID: ruleBriefFlagLanded, Severity: rc.Severity,
			Message: "brief_flag_landed reads the review flags through docfidelity.UnlandedFlags, and the front door " +
				"running it registered none (lint.SetBriefFlagCheck), so no flag was checked; " +
				"register it where the gate is wired, as cmd/record-lint and the CLI do",
		}}, nil
	}
	misses, err := briefFlagMisses(repoRoot)
	if err != nil {
		return nil, &configError{ruleBriefFlagLanded + ": " + err.Error()}
	}
	out := make([]Finding, 0, len(misses))
	for _, m := range misses {
		out = append(out, Finding{
			File: m.File, Line: m.Line, RuleID: ruleBriefFlagLanded, Severity: rc.Severity,
			Message: "the review flag on " + m.Chapter + " records the replacement " + strconv.Quote(m.Replacement) +
				", which " + m.Why + ": the flag names what the product thinker must read, so it records the " +
				"sentence as committed. Bring its replacement to the chapter's wording (the part on one line, " +
				"where the sentence wraps), or remove the flag once the product thinker has read it",
		})
	}
	return out, nil
}
