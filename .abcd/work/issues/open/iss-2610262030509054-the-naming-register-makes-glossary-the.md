---
schema_version: 1
id: "iss-2610262030509054"
slug: "the-naming-register-makes-glossary-the"
severity: "nitpick"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 51 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/02-constraints/04-naming.md:144"
remedy: "Drop 'Glossary-aware once `terminology/` exists' from the grill row (05-intent.md line 319), or delete the row, which describes a sub-verb that does not ship; the glossary is `brief/glossary/`."
finding_ref: cfk-14d1c566f0dbb3a1
---

The naming register makes glossary/ the only glossary; the intent chapter waits on a terminology/ directory for the same role

A naming conflict (severity nitpick) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/02-constraints/04-naming.md:144` — "is the one canonical glossary and the only place a glossary lives"
- End B: `.abcd/development/brief/04-surfaces/05-intent.md:319` — "Glossary-aware once `terminology/` exists."

04-naming says the glossary at ../glossary/ is the one canonical glossary and the only place a glossary lives, while 05-intent says the grill step becomes glossary-aware once terminology/ exists. Two names are given for the one glossary location.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 51.
