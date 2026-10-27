---
schema_version: 1
id: "iss-2610262034225527"
slug: "itd-78-still-a-draft-promises-the"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 491 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/drafts/itd-78-intent-dependency-graph.md:17"
remedy: "none (filed automatically)"
related_intents: [itd-78]
finding_ref: cfk-ac1c9072faf5bfe6
---

itd-78, still a draft, promises the `blocked_by` and `builds_on` edges; the intent chapter already ships them through the edge sub-verb with the same definitions.

A scope leakage (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/drafts/itd-78-intent-dependency-graph.md:17` — "its **edges** — `blocked_by` (cannot ship before) and `builds_on` (cheaper or better if the other exists first)"
- End B: `.abcd/development/brief/04-surfaces/05-intent.md:339` — "Adds and removes the record's dependency edges, `blocked_by` (cannot ship before) and `builds_on` (cheaper or better if the other exists first)"

itd-78 defines an intent's edges as `blocked_by` (cannot ship before) and `builds_on` (cheaper or better if the other exists first). 05-intent.md's shipped edge sub-verb writes exactly those two edges, in the same words, and the chapter lints them with stale_edge and edge_cycle. So both documents claim the edge half, and the draft still reads as if that half were undelivered.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 491.
