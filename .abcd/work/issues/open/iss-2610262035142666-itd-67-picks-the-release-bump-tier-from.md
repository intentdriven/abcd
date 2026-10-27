---
schema_version: 1
id: "iss-2610262035142666"
slug: "itd-67-picks-the-release-bump-tier-from"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 557 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/shipped/itd-67-installable-versioned-plugin.md:47"
remedy: "none (filed automatically)"
related_intents: [itd-67]
finding_ref: cfk-61e6fb596ac25b4f
---

itd-67 picks the release bump tier from phase-completion detection; the intent chapter derives the version from each intent's impact judgement

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/shipped/itd-67-installable-versioned-plugin.md:47` — "`/abcd:launch ship` auto-selects the bump tier (patch/minor/major) via the brief §4 phase-completion detection"
- End B: `.abcd/development/brief/04-surfaces/05-intent.md:393` — "the compatibility judgement the derived version is computed from"

itd-67 has /abcd:launch ship choose patch, minor or major through the brief's phase-completion detection. The intent chapter says the intent's impact field is the compatibility judgement the derived version is computed from, and elsewhere says phases are retired. Two incompatible inputs are named for the same bump decision.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 557.
