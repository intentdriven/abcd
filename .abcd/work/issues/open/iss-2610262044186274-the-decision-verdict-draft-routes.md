---
schema_version: 1
id: "iss-2610262044186274"
slug: "the-decision-verdict-draft-routes"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 1197 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/drafts/itd-44-fourth-intent-kind-decision.md:111"
remedy: "none (filed automatically)"
related_intents: [itd-44, itd-2610090831227812]
finding_ref: cfk-3dc1e7d6e7a8732b
---

The decision-verdict draft routes capture through an abcd-intent-new skill, but abcd ships zero skills

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/drafts/itd-44-fourth-intent-kind-decision.md:111` — "because the `abcd-intent-new` skill is wrapper-only and has no Python classifier seam"
- End B: `.abcd/development/intents/shipped/itd-2610090831227812-a-person-who-types-abcd-sees-only-the.md:54` — "abcd ships zero skills and the `/abcd:` namespace is commands only"

itd-44 puts its routing in the `abcd-intent-new` skill, which calls the CLI classifier and then routes. itd-2610090831227812 states that abcd ships zero skills and that the `/abcd:` namespace is commands only. The seam the draft's routing depends on is not there.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 1197.
