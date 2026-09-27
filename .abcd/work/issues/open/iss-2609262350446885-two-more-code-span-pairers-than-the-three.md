---
schema_version: 1
id: "iss-2609262350446885"
slug: "two-more-code-span-pairers-than-the-three"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "autonomous run A resumed 2026-09-25: drainSpan"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/lint/lint.go"
---

Two more code-span pairers than the three iss-2609262322244502 names pair backticks one at a time rather than by run. lint's stripInlineCode (GL002 synonyms, links_resolve, citations) pairs single backticks inside longer runs, so a span opened by two backticks reads as two empty spans with live prose between them: a glossary synonym quoted in a double-backtick span is flagged as live prose and a link quoted in one is checked, while a word between a single backtick and a double run no span holds is blanked. lab's parseCorrections closes a quoted literal on the next single backtick, so a retract literal quoted in a double-backtick span because it holds a backtick is read as empty and refused as noise. Both are loud or over-strict rather than silent. Found by the one-pairer guard while fixing iss-2609262322244502.
