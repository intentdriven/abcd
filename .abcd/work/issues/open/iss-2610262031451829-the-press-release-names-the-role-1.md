---
schema_version: 1
id: "iss-2610262031451829"
slug: "the-press-release-names-the-role-1"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 172 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/01-product/01-press-release.md:26"
remedy: "none (filed automatically)"
related_intents: [itd-30]
finding_ref: cfk-4cd38483a5d1f00c
refines: [itd-48, itd-32]
---

The press release names the Role 1 auditor `intent-auditor`, but itd-30 calls the same reader `intent-fidelity-reviewer`.

A naming conflict (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/01-product/01-press-release.md:26` — "`/abcd:intent audit` (Role 1 of `intent-auditor`) reviews delivered reality against the press release"
- End B: `.abcd/development/intents/drafts/itd-30-design-fictions-as-intent-format.md:18` — "`intent-fidelity-reviewer` reads either dialect against the same delivered reality"

01-press-release says `/abcd:intent audit` is Role 1 of `intent-auditor`, which reviews delivered reality against the press release. itd-30 gives the same job, reading an intent against delivered reality, to an agent it names `intent-fidelity-reviewer`. That is two agent names for one agent.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 172.
