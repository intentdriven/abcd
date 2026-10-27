---
schema_version: 1
id: "iss-2610262041065877"
slug: "the-attribution-draft-puts-its-setup"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 977 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/06-delivery/01-build-sequence.md:59"
remedy: "none (filed automatically)"
related_intents: [itd-91]
finding_ref: cfk-db9385f498843665
---

The attribution draft puts its setup question in `/abcd:init` / `prepare-this-repo`; the build sequence says there is no `abcd init` and install is `abcd ahoy install`.

A naming conflict (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/06-delivery/01-build-sequence.md:59` — "There is no `abcd init` and no config get/set pair"
- End B: `.abcd/development/intents/drafts/itd-91-ai-attribution-preference.md:31` — "in the onboarding surface (`/abcd:init` / `prepare-this-repo`)"

01-build-sequence.md says installing abcd is `abcd ahoy install`, that there is no `abcd init`, and that install is the write path a person uses. itd-91 puts its onboarding question in `/abcd:init` and `prepare-this-repo`, giving a second, nonexistent name to the surface where setup happens.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 977.
