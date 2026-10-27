---
schema_version: 1
id: "iss-2610262041183220"
slug: "the-decision-verdict-draft-routes"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 991 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/05-internals/08-skills.md:8"
remedy: "none (filed automatically)"
related_intents: [itd-44]
finding_ref: cfk-80d19ba9caaf647f
---

The decision-verdict draft routes through an `abcd-intent-new` skill; the skills chapter says abcd ships zero skills.

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/05-internals/08-skills.md:8` — "abcd ships zero skills."
- End B: `.abcd/development/intents/drafts/itd-44-fourth-intent-kind-decision.md:111` — "because the `abcd-intent-new` skill is wrapper-only and has no Python"

08-skills.md says abcd ships no skills and the `/abcd:` namespace is commands only. itd-44 bases its design on an `abcd-intent-new` skill that is 'wrapper-only' and calls a CLI classifier, a surface that does not exist.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 991.
