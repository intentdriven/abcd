---
schema_version: 1
id: "iss-2610262039224536"
slug: "itd-91-puts-its-setup-question-in-abcd"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 856 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/drafts/itd-91-ai-attribution-preference.md:31"
remedy: "none (filed automatically)"
related_intents: [itd-91]
finding_ref: cfk-5740da52f3187afb
wontfix_reason: "Closed in the 2026-10-27 triage (dcn-2610271112330536) as a duplicate of iss-2610262034182114. Same contradiction (itd-91 names a retired onboarding surface); the surfaces register is one more witness."
duplicates: [iss-2610262034182114]
---

itd-91 puts its setup question in `/abcd:init` / `prepare-this-repo`, but the register says repository preparation moved into ahoy's install section

A naming conflict (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/drafts/itd-91-ai-attribution-preference.md:31` — "in the onboarding surface (`/abcd:init` / `prepare-this-repo`)"
- End B: `.abcd/development/brief/04-surfaces/README.md:388` — "the preparation of a target repo, mutates that repo; its page retired into the"

itd-91 names the onboarding surface `/abcd:init` / `prepare-this-repo`. The surfaces register lists no `init` command, and says that preparing a target repository, once a skill, moved into the install section of `ahoy.md`. itd-91 refers to the setup surface by two names the register does not carry.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 856.

## Grounds

- declined: Closed in the 2026-10-27 triage (dcn-2610271112330536) as a duplicate of iss-2610262034182114. Same contradiction (itd-91 names a retired onboarding surface); the surfaces register is one more witness.
