---
schema_version: 1
id: "iss-2610262034182114"
slug: "itd-91-puts-its-setup-question-in-abcd"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 485 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/drafts/itd-91-ai-attribution-preference.md:31"
remedy: "In itd-91 replace '/abcd:init / prepare-this-repo' (press-release rationale, scope bullet and acceptance criterion) with abcd ahoy install; the prepare-this-repo page retired into the install section of 01-ahoy.md (itd-2610090831227812) and no init command exists."
related_intents: [itd-91]
finding_ref: cfk-e0a335f28315aaaa
---

itd-91 puts its setup question in `/abcd:init` / `prepare-this-repo`; the board chapter says the `prepare-this-repo` page retired into the install.

A sequencing impossibility (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/drafts/itd-91-ai-attribution-preference.md:31` — "in the onboarding surface (`/abcd:init` / `prepare-this-repo`)"
- End B: `.abcd/development/brief/04-surfaces/08-abcd.md:459` — "`prepare-this-repo`, whose page retired into"

itd-91 puts the AI-attribution question in an onboarding surface it names `/abcd:init` / `prepare-this-repo`. 08-abcd.md says the `prepare-this-repo` page retired into the install, a name answered with what replaced it. The intent depends on an onboarding surface that the brief no longer has.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 485.
