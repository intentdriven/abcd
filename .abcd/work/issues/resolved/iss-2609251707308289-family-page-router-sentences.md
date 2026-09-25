---
schema_version: 1
id: "iss-2609251707308289"
slug: "family-page-router-sentences"
severity: "major"
category: "inconsistency"
source: "impl-review"
found_during: "autonomous run A resumed 2026-09-25"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/surface/sentences.go"
resolution: "The seven family parents and docs cite name their work and write discipline; TestAParentSentenceNamesItsFamilysWork holds every parent's doing clause to naming no verb."
impact: fix
resolved_by:
  commit: "7a130cbc"
---

Seven family plugin pages (ideate, disembark, embark, history, guard, docs, implement) carry a router sentence as the description a host lists the skill by: the parent verbs' manifest sentences describe the dispatcher (List the verbs that ...) or the bare form alone, so an agent choosing a skill from the host list learns only that verbs exist, not the family's work or its write discipline

## Grounds

- pursued: every family page's description names the family's work; shown wrong by a parent sentence whose doing clause names verbs, which the new test fails on
