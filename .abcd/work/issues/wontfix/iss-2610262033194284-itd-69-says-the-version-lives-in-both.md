---
schema_version: 1
id: "iss-2610262033194284"
slug: "itd-69-says-the-version-lives-in-both"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 389 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/shipped/itd-69-plugin-metadata-lockstep-update.md:31"
remedy: "none (filed automatically)"
related_intents: [itd-69]
finding_ref: cfk-ad2a84f61b2f686c
wontfix_reason: "Closed in the 2026-10-27 triage (dcn-2610271112330536): a later record already settles which end holds. The both-plugin.json wording reads as the pre-single-repo layout; lockstep.go checks one primary location from the version-location contract plus the one canonical marketplace manifest."
---

itd-69 says the version lives in both plugin.json locations plus marketplace.json; 04-launch says two locations are the whole of it

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/shipped/itd-69-plugin-metadata-lockstep-update.md:31` — "both `plugin.json` locations and `.claude-plugin/marketplace.json`"
- End B: `.abcd/development/brief/04-surfaces/04-launch.md:640` — "Stamps nothing else. Those two locations are the whole of it"

itd-69 builds its lockstep checker on the version and changelog living in more than one place: both `plugin.json` locations and `.claude-plugin/marketplace.json`, which makes three surfaces. 04-launch says the cut stamps the selected version location and the one canonical marketplace.json, stamps nothing else, and that those two locations are the whole of it. The checker's path set cannot be both three surfaces and two.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 389.

## Grounds

- declined: Closed in the 2026-10-27 triage (dcn-2610271112330536): a later record already settles which end holds. The both-plugin.json wording reads as the pre-single-repo layout; lockstep.go checks one primary location from the version-location contract plus the one canonical marketplace manifest.
