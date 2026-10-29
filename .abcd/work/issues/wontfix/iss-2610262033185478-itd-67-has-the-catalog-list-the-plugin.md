---
schema_version: 1
id: "iss-2610262033185478"
slug: "itd-67-has-the-catalog-list-the-plugin"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 387 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/shipped/itd-67-installable-versioned-plugin.md:45"
remedy: "none (filed automatically)"
related_intents: [itd-67]
finding_ref: cfk-4f9008ca74a8f7ad
wontfix_reason: "Closed in the 2026-10-27 triage (dcn-2610271112330536): a later record already settles which end holds. The committed catalog now pins a release archive; the amendment note sits on the acceptance criterion, while the scope bullet and press release keep the original wording."
---

itd-67 has the catalog list the plugin with `source: "./"`; 04-launch has the cut replace the catalog's source with the pinned release archive

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/shipped/itd-67-installable-versioned-plugin.md:45` — "marketplace `abcd-marketplace`, one plugin `abcd`, `source: "./"`"
- End B: `.abcd/development/brief/04-surfaces/04-launch.md:846` — "the catalog's plugin source becomes the release's pinned archive"

itd-67 defines `.claude-plugin/marketplace.json` with one plugin whose source is `./`, the repository itself. 04-launch says the cut turns the catalog's plugin source into the release's pinned archive wherever the version-location contract declares `publishes_plugin_archive`, which abcd does, so an install receives the cut release and not the working tree. The catalog cannot name both the repository root and a pinned archive as abcd's plugin source.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 387.

## Grounds

- declined: Closed in the 2026-10-27 triage (dcn-2610271112330536): a later record already settles which end holds. The committed catalog now pins a release archive; the amendment note sits on the acceptance criterion, while the scope bullet and press release keep the original wording.
