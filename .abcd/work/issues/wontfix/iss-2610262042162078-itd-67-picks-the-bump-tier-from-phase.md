---
schema_version: 1
id: "iss-2610262042162078"
slug: "itd-67-picks-the-bump-tier-from-phase"
severity: "major"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 1058 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/shipped/itd-67-installable-versioned-plugin.md:47"
remedy: "none (filed automatically)"
related_intents: [itd-67]
finding_ref: cfk-f652bc07912593e8
wontfix_reason: "Closed in the 2026-10-27 triage (dcn-2610271112330536): a later record already settles which end holds. The committed manifests carry no version and the render stamps only the payload; both the bump source and the version's place changed by later shipped records."
---

itd-67 picks the bump tier from phase completion and writes the version into plugin.json; the version entry keeps the working tree unversioned and derives the version from impact.

A premise contradiction (severity major) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/shipped/itd-67-installable-versioned-plugin.md:47` — "auto-selects the bump tier (patch/minor/major) via the brief §4 phase-completion detection, writes the new version into `plugin.json`"
- End B: `.abcd/development/brief/glossary/distribution/version.md:22` — "The working tree stays unversioned; the version lives only in the cut artifact and its tag"

itd-67 has `launch ship` choose the bump tier through the brief's phase-completion detection and write the new version into plugin.json, which it calls the single source of the installed version. The version entry, which names itd-67 as its origin, says the version comes from the impact of what shipped and lives only in the cut artifact and its tag. It also says the working tree stays unversioned, and the phase whose completion itd-67 would detect is retired. Both the source of the bump and the place the version lives disagree.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 1058.

## Grounds

- declined: Closed in the 2026-10-27 triage (dcn-2610271112330536): a later record already settles which end holds. The committed manifests carry no version and the render stamps only the payload; both the bump source and the version's place changed by later shipped records.
