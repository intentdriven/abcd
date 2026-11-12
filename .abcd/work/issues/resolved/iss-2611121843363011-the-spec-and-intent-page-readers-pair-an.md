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
resolution: "The page readers no longer blank text the page may show as prose: through one walk in mdrecord.Read that the spec readiness filter, the scope-condition and hand-audit readers and the prose-citation rule share, a code span is blanked only where it opens and closes on one line and no earlier line of its stretch (the lines with no blank line between them) leaves a backtick run open; from such a run to the stretch end the text is read whole and no comment opens, a span over a line break is read as prose, and a comment opener inside a one-line span opens nothing (dcn-2611122004497694)."
impact: fix
---

The spec and intent page readers pair an inline code span's backticks on one line; a rendered page pairs them across a line break inside a paragraph, so the readers and the page disagree about which text is prose.
