---
schema_version: 1
id: "iss-2610262033173969"
slug: "itd-67-has-launch-ship-write-plugin-json"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 384 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/shipped/itd-67-installable-versioned-plugin.md:47"
remedy: "none (filed automatically)"
related_intents: [itd-67]
finding_ref: cfk-5a7add6eab4ad3a5
wontfix_reason: "Closed in the 2026-10-27 triage (dcn-2610271112330536): a later record already settles which end holds. Ship code has no tag-writing step (only a tag listing for retention), and the working-tree manifests stay unversioned; the closed spec says what replaced itd-67's wording."
---

itd-67 has launch ship write plugin.json and tag the repo; 04-launch says the verb neither commits, tags nor publishes

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/shipped/itd-67-installable-versioned-plugin.md:47` — "`/abcd:launch ship` auto-selects the bump tier (patch/minor/major) via the brief §4 phase-completion detection, writes the new version into `plugin.json`, updates `marketplace.json` (version + changelog entry), tags the repo"
- End B: `.abcd/development/brief/04-surfaces/04-launch.md:179` — "The verb neither commits, tags, nor publishes"

itd-67 promises that `/abcd:launch ship` picks the bump tier by phase-completion detection, writes the version into `plugin.json`, updates `marketplace.json` and tags the repo. 04-launch says the shipped verb neither commits, tags nor publishes, so a human and CI perform every step after the changelog heading and the release page. The same chapter also derives the tier from the records' declared impact and leaves the working-tree manifests unversioned. A ship verb cannot both tag the repo and never tag it.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 384.

## Grounds

- declined: Closed in the 2026-10-27 triage (dcn-2610271112330536): a later record already settles which end holds. Ship code has no tag-writing step (only a tag listing for retention), and the working-tree manifests stay unversioned; the closed spec says what replaced itd-67's wording.
