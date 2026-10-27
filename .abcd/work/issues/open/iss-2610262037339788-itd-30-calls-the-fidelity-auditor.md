---
schema_version: 1
id: "iss-2610262037339788"
slug: "itd-30-calls-the-fidelity-auditor"
severity: "nitpick"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 731 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/drafts/itd-30-design-fictions-as-intent-format.md:18"
remedy: "none (filed automatically)"
related_intents: [itd-30]
finding_ref: cfk-392f79ac0824978c
---

itd-30 calls the fidelity auditor intent-fidelity-reviewer; the build brief calls that validator intent-auditor

A naming conflict (severity nitpick) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/drafts/itd-30-design-fictions-as-intent-format.md:18` — "`intent-fidelity-reviewer` reads either dialect against the same delivered reality"
- End B: `.abcd/development/brief/04-surfaces/34-build.md:683` — "on the lane whose landing closes the spec, an `intent-auditor`"

itd-30 says an agent named `intent-fidelity-reviewer` reads an intent of either format against delivered reality. The build chapter gives the fidelity audit to `intent-auditor`, on the lane whose landing closes the spec. One role has two names, and a format built from itd-30 would be aimed at an agent the build loop never dispatches.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 731.
