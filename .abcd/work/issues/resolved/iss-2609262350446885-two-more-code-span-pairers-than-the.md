---
schema_version: 1
id: "iss-2609262350446885"
slug: "two-more-code-span-pairers-than-the"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "autonomous run A resumed 2026-09-25: drainSpan"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/lint/lint.go"
resolution: "lint's stripInlineCode and lab's parseCorrections pair backtick runs through termsafe.PairCodeSpan: a double-backtick span is one span, so a synonym or a link quoted in one is a mention, a word no span holds stays live, and a retract literal holding a backtick is read whole. TestNoSecondCodeSpanPairer holds both to the one pairer."
impact: fix
resolved_by:
  commit: "da875bc58"
---

Two more code-span pairers than the three iss-2609262322244502 names pair backticks one at a time rather than by run. lint's stripInlineCode (GL002 synonyms, links_resolve, citations) pairs single backticks inside longer runs, so a span opened by two backticks reads as two empty spans with live prose between them: a glossary synonym quoted in a double-backtick span is flagged as live prose and a link quoted in one is checked, while a word between a single backtick and a double run no span holds is blanked. lab's parseCorrections closes a quoted literal on the next single backtick, so a retract literal quoted in a double-backtick span because it holds a backtick is read as empty and refused as noise. Both are loud or over-strict rather than silent. Found by the one-pairer guard while fixing iss-2609262322244502.

## Grounds

- pursued: a glossary synonym quoted in a double-backtick span raises no GL002 finding and a retract literal quoted in one sweeps as its text; a GL002 finding on such a span, or a double-backtick literal refused as noise, would show it wrong
