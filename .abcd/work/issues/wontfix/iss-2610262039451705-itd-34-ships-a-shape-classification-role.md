---
schema_version: 1
id: "iss-2610262039451705"
slug: "itd-34-ships-a-shape-classification-role"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 884 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/05-internals/01-agents.md:176"
remedy: "none (filed automatically)"
related_intents: [itd-34]
finding_ref: cfk-923f527aa1cdb309
wontfix_reason: "Closed in the 2026-10-27 triage (dcn-2610271112330536): a later record already settles which end holds. The shipped record itself says the role was not part of its delivery, so the two agree when read in full."
---

itd-34 ships a shape-classification role that runs from the pre-commit hook and `/abcd:intent shape`; 01-agents says the role is a design target with no `shape` sub-verb.

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/05-internals/01-agents.md:176` — "`shape` sub-verb is registered, and no cached suggestions exist for bare"
- End B: `.abcd/development/intents/shipped/itd-34-three-intent-kinds.md:90` — "It runs continuously via the pre-commit hook (writing findings to the latest report) and on-demand via `/abcd:intent shape`."

itd-34 is in shipped/. Its scope says the third role runs continuously from the pre-commit hook and on demand through `/abcd:intent shape`, and that bare `/abcd:intent` shows cached suggestions. 01-agents says the kind-classification role is a design target, that no `shape` sub-verb is registered and that no cached suggestions exist. Both cannot be accurate: one is a shipped record promising a surface, the other is the catalogue saying it was never built.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 884.

## Grounds

- declined: Closed in the 2026-10-27 triage (dcn-2610271112330536): a later record already settles which end holds. The shipped record itself says the role was not part of its delivery, so the two agree when read in full.
