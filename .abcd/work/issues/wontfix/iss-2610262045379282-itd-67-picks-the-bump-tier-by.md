---
schema_version: 1
id: "iss-2610262045379282"
slug: "itd-67-picks-the-bump-tier-by"
severity: "major"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 1287 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/shipped/itd-67-installable-versioned-plugin.md:47"
remedy: "none (filed automatically)"
related_intents: [itd-67, itd-73]
finding_ref: cfk-b3008e67efd51a45
wontfix_reason: "Closed in the 2026-10-27 triage (dcn-2610271112330536): a later record already settles which end holds. Both intents shipped in the same release and the closed spec for itd-67 explicitly picks itd-73's rule; code follows itd-73."
---

itd-67 picks the bump tier by phase-completion detection and keeps the version in plugin.json; itd-73 derives it from shipped intents' impact on an unversioned tree

A premise contradiction (severity major) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/shipped/itd-67-installable-versioned-plugin.md:47` — "auto-selects the bump tier (patch/minor/major) via the brief §4 phase-completion detection"
- End B: `.abcd/development/intents/shipped/itd-73-derived-versioning.md:30` — "take the highest-severity `impact`, and compute the next SemVer"

itd-67 has launch ship choose patch, minor or major through the brief's phase-completion detection, write the version into plugin.json as the single source of the installed version, and keep --version as an override. itd-73 derives the version from the highest impact among intents shipped since the last release, says the working tree stays unversioned and that no one types a version. The two shipped records give incompatible rules for how a release's number is chosen and where it lives.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 1287.

## Grounds

- declined: Closed in the 2026-10-27 triage (dcn-2610271112330536): a later record already settles which end holds. Both intents shipped in the same release and the closed spec for itd-67 explicitly picks itd-73's rule; code follows itd-73.
