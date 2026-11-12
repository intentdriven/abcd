---
schema_version: 1
id: "iss-2611121843363011"
slug: "the-spec-and-intent-page-readers-pair-an"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "manual-capture"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/mdrecord/mdrecord.go"
remedy: "One primitive that pairs code spans as the page does, used by every reader of a whole page."
resolution: "The page readers pair an inline code span on each line alone through one walk in mdrecord, which marks a line unsure on the lines below a backtick run an earlier line of its stretch leaves with no closer, to the next blank line, inside a fence or comment as well as outside, and after the run on its own line. The spec readiness filter blanks no span on the lines below such a run, the run's own line keeping its spans, and reads a span over a line break as prose; there a comment opener that is the first thing on its line, at up to three columns, opens a comment, and any other opener with no closer on its line reads the page whole from that line and is named. The scope-condition and hand-audit readers, the stamp and the prose-citation rule pair spans on every line alone, as before, and the stamp refuses a record whose comments read two ways. A backtick behind an odd run of backslashes opens no span (dcn-2611122109097376)."
impact: fix
---

The spec and intent page readers pair an inline code span's backticks on one line; a rendered page pairs them across a line break inside a paragraph, so the readers and the page disagree about which text is prose.
