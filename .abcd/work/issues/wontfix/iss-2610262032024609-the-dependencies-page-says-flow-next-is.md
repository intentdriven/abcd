---
schema_version: 1
id: "iss-2610262032024609"
slug: "the-dependencies-page-says-flow-next-is"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 211 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/02-constraints/02-dependencies.md:41"
remedy: "none (filed automatically)"
related_intents: [itd-34]
finding_ref: cfk-c762661af6c7780f
wontfix_reason: "Closed in the 2026-10-27 triage (dcn-2610271112330536): a later record already settles which end holds. Both ends were true when written; adr-26 and adr-21 already say which holds, and the brief is current."
---

The dependencies page says flow-next is not a dependency and is not built; itd-34's lifecycle paths call `/flow-next:plan`.

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/02-constraints/02-dependencies.md:41` — "flow-next is not a dependency and is not built."
- End B: `.abcd/development/intents/shipped/itd-34-three-intent-kinds.md:52` — "`/flow-next:plan` once with all intents as joint input"

The brief rules flow-next out entirely: the native spec store is the default and ccpm is the deeper backend. itd-34, a shipped record, defines the standalone and bundle-member plan paths as runs of `/flow-next:plan`. Someone building the plan path from itd-34 would depend on a tool the brief says abcd neither depends on nor builds.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 211.

## Grounds

- declined: Closed in the 2026-10-27 triage (dcn-2610271112330536): a later record already settles which end holds. Both ends were true when written; adr-26 and adr-21 already say which holds, and the brief is current.
