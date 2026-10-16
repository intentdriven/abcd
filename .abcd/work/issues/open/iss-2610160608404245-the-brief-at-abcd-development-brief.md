---
schema_version: 1
id: "iss-2610160608404245"
slug: "the-brief-at-abcd-development-brief"
severity: "minor"
category: "documentation"
source: "review-followup"
found_during: "v0.15.0 release review"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/04-launch.md"
remedy: "Say at 04-launch.md:498 that impact is required at the close that moves an intent to shipped/, and checked for its value wherever it is set."
---

The brief at .abcd/development/brief/04-surfaces/04-launch.md:498 says every intent carries an impact, set when the intent is shaped and enforced by the record lint (adr-31). At 0de27c795, 91 intents in drafts/ and 4 in planned/ carry no impact key; the record lint's intent_impact_valid checks only the value set, and impact is required only at the close that moves an intent to shipped/, as 05-intent.md states.
