---
schema_version: 1
id: "iss-2610262039476904"
slug: "itd-34-maps-a-standalone-intent"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 886 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/README.md:275"
remedy: "none (filed automatically)"
related_intents: [itd-34]
finding_ref: cfk-1cf9980d6a5a4160
---

itd-34 maps a standalone intent one-to-one to a flow-next spec; the surfaces register says an intent owns one or more specs in the native spec store.

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/04-surfaces/README.md:275` — "an intent owns one or more specs"
- End B: `.abcd/development/intents/shipped/itd-34-three-intent-kinds.md:21` — "A standalone intent maps 1:1 to a flow-next `spec`"

itd-34's press release and lifecycle paths say a standalone intent maps 1:1 to a flow-next `spec`, planned through `/flow-next:plan`. The README's `spec` row describes the native spec store, where `spec close` ships an intent only when no open spec still names it, because an intent owns one or more specs. The two documents disagree on which store holds specs and on how many specs one intent can have.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 886.
