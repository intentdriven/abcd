---
schema_version: 1
id: "iss-2610262044429007"
slug: "the-attribution-preference-draft-puts"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 1224 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/drafts/itd-91-ai-attribution-preference.md:31"
remedy: "none (filed automatically)"
related_intents: [itd-91, itd-2610090831227812]
finding_ref: cfk-97010c9a01c6065b
wontfix_reason: "Closed in the 2026-10-27 triage (dcn-2610271112330536) as a duplicate of iss-2610262034182114. Same contradiction; the shipped verb-split intent that folds prepare-this-repo into ahoy install is one more witness."
duplicates: [iss-2610262034182114]
---

The attribution-preference draft puts its setup question in `/abcd:init` / `prepare-this-repo`, surfaces the verb split retired or never had

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/drafts/itd-91-ai-attribution-preference.md:31` — "in the onboarding surface (`/abcd:init` / `prepare-this-repo`) offering a selection of acknowledgement conventions"
- End B: `.abcd/development/intents/shipped/itd-2610090831227812-a-person-who-types-abcd-sees-only-the.md:48` — "`prepare-this-repo` folds into `ahoy install`; `consult`, `ingest`, `memory` and `source` merge into `library`"

itd-91 adds its question to an onboarding surface named `/abcd:init` and `prepare-this-repo`. itd-2610090831227812 folds `prepare-this-repo` into `ahoy install` with no alias, and no `init` command exists in its placement of the person's commands. The draft targets a surface that is not there.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 1224.

## Grounds

- declined: Closed in the 2026-10-27 triage (dcn-2610271112330536) as a duplicate of iss-2610262034182114. Same contradiction; the shipped verb-split intent that folds prepare-this-repo into ahoy install is one more witness.
