---
schema_version: 1
id: "iss-2609251600029607"
slug: "testnosecondfencerule-s-delimiter-only"
severity: "minor"
category: "tech-debt"
source: "impl-review"
found_during: "autonomous run A resumed 2026-09-25"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/mdrecord"
resolution: "TestNoFenceRunReaderOutsideMdrecord reads the parsed source for regexp literals matching a fence run and first-byte comparisons against a backtick or tilde, allowlisted per file with a count and reason; TestNoSecondFenceRule's doc names what each detector reaches"
impact: internal
resolved_by:
  commit: "38958d6d4"
---

TestNoSecondFenceRule's delimiter-only detector misses a private toggle written as a literal regexp that matches a run (for example a pattern of 0-3 spaces then a backtick or tilde run, with a length check) or as byte comparisons on the first character; its doc says a pattern 'assembled at run time' escapes, which understates a literal regexp. Extend the detector to regexp literals whose pattern can match a fence run, and to first-byte comparisons against a backtick or tilde, or narrow the doc to the truth.

## Grounds

- pursued: a private toggle in either shape now fails a detector; a planted regexp run toggle or ln[0] == '~' toggle passing both detectors would show it wrong
