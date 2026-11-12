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
resolution: "The page readers pair inline code spans over a line break inside a paragraph, as the page does, through one walk in mdrecord.Read that the spec readiness filter, the scope-condition and hand-audit readers and the prose-citation rule share; the comment walk opens no comment inside such a span."
impact: fix
---

The spec and intent page readers pair an inline code span's backticks on one line; a rendered page pairs them across a line break inside a paragraph, so the readers and the page disagree about which text is prose.
