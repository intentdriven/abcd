---
schema_version: 1
id: "iss-2610262042367104"
slug: "persona-alice-is-a-product-thinker-in"
severity: "nitpick"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 1081 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/drafts/itd-106-abcd-sets-up-the-ci-a-repo-requires-and.md:30"
remedy: "none (filed automatically)"
related_intents: [itd-106, itd-107]
finding_ref: cfk-dd637a7ea4e144bf
---

Persona Alice is a product thinker in itd-106 and a maintainer in itd-107

A naming conflict (severity nitpick) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/drafts/itd-106-abcd-sets-up-the-ci-a-repo-requires-and.md:30` — "said Alice, a product thinker"
- End B: `.abcd/development/intents/drafts/itd-107-autonomous-routines-assemble-from-one.md:29` — "said Alice, a maintainer"

The persona convention picks a name by role, so one registered name stands for one role. itd-106 quotes Alice as a product thinker and itd-107 quotes Alice as a maintainer, so the name now stands for two different roles.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 1081.
