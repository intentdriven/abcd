---
schema_version: 1
id: "iss-2609261036355193"
slug: "scribe-ingest-never-holds-a-disposition-s-state-to-the"
severity: "major"
category: "security"
source: "review-followup"
found_during: "autonomous run A resumed 2026-09-25: review-scribe"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/scribe/ingest.go"
resolution: "scribe ingest holds a disposition's state, whole-word and in any case, to a line of the supplied dispositions that names the item, and an admission to a line that admits or accepts it; the chapter's disclosed limit states the residue (words, not sense)"
impact: internal
resolved_by:
  commit: "1d82650c"
---

scribe ingest never holds a disposition's state to the supplied text: a researcher line 'rdi-X: rejected — <ground>' ingests as state accepted with the verbatim ground, so the ruling itself is the one field the scribe can author, against the spec's out-of-scope rule that a state the material does not carry is refused, never supplied

## Grounds

- pursued: a payload state the item's own supplied line does not carry is refused before anything is written; a payload that lands a state no line naming its item carries would show it wrong
