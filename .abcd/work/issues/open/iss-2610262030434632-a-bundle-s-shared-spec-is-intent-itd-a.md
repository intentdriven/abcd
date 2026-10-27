---
schema_version: 1
id: "iss-2610262030434632"
slug: "a-bundle-s-shared-spec-is-intent-itd-a"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 36 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/01-product/03-mental-model.md:65"
remedy: "none (filed automatically)"
finding_ref: cfk-acae076f8784401c
---

A bundle's shared spec is intent: [itd-A, itd-B] in the mental model but a scalar intent: beside an intents: list in the intent chapter

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/01-product/03-mental-model.md:65` — "`/abcd:intent plan` accepts multiple intent IDs and creates a single spec with `intent: [itd-A, itd-B]`"
- End B: `.abcd/development/brief/04-surfaces/05-intent.md:324` — "It mints one spec whose frontmatter lists every member (`intents:` beside `intent:`, and `bundle:`)"

03-mental-model says bundle planning creates a single spec whose intent field is a list of the members, while 05-intent says the bundle command mints a spec whose frontmatter carries intent: as the first member with every member listed under a separate intents: key. A reader of the spec's intent field gets a different shape from each.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 36.
