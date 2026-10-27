---
schema_version: 1
id: "iss-2610262031592610"
slug: "itd-24-uses-voyage-for-a-project-s"
severity: "nitpick"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 206 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/shipped/itd-24-reflect-command.md:55"
remedy: "none (filed automatically)"
related_intents: [itd-24]
finding_ref: cfk-b401f9851689a614
wontfix_reason: "Closed in the 2026-10-27 triage (dcn-2610271112330536): a later record already settles which end holds. 04-naming and the glossary reserve voyage for the operations namespace; the same word choice survives in commands/embark.md and the embark lessons prompt, tracked there."
---

itd-24 uses voyage for a project's working life; 04-naming reserves voyage for the operator-level operations namespace.

A terminology drift (severity nitpick) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/shipped/itd-24-reflect-command.md:55` — "packs *all* of the voyage's retrospectives into the lifeboat"
- End B: `.abcd/development/brief/02-constraints/04-naming.md:231` — "The **operations** namespace (verb-side: what we did)"

itd-24 says the pack carries all of 'the voyage's retrospectives into the lifeboat' and speaks of a new voyage's brief, using voyage to mean a project's journey. 04-naming registers voyage/ as 'The **operations** namespace (verb-side: what we did)' at the operator level, keyed on the source's root commit, which holds no retrospectives. The reserved term is used in a second sense.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 206.

## Grounds

- declined: Closed in the 2026-10-27 triage (dcn-2610271112330536): a later record already settles which end holds. 04-naming and the glossary reserve voyage for the operations namespace; the same word choice survives in commands/embark.md and the embark lessons prompt, tracked there.
