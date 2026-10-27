---
schema_version: 1
id: "iss-2610262041504223"
slug: "the-internals-index-counts-eighteen"
severity: "nitpick"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 1028 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/05-internals/README.md:7"
remedy: "none (filed automatically)"
related_intents: [itd-36]
finding_ref: cfk-7f7ea2ac9e153325
---

The internals index counts eighteen shipped agent prompts; itd-36 says the agent count stays at 15.

A premise contradiction (severity nitpick) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/05-internals/README.md:7` — "The eighteen shipped agent prompts and the design roster still to be built"
- End B: `.abcd/development/intents/shipped/itd-36-memory-unification.md:47` — "agent count stays 15"

The 05-internals index describes 01-agents.md as holding eighteen shipped agent prompts. itd-36 justifies adding the curator role to principle-distiller by stating that the agent count stays at 15. The two documents give different counts for the same agent catalogue.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 1028.
