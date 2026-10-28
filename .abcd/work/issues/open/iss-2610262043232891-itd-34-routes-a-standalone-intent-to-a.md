---
schema_version: 1
id: "iss-2610262043232891"
slug: "itd-34-routes-a-standalone-intent-to-a"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 1134 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/disciplines/itd-37-modification-grammar.md:55"
remedy: "none (filed automatically)"
related_intents: [itd-37, itd-34]
finding_ref: cfk-e20a3d3d17057814
---

itd-34 routes a standalone intent to a flow-next spec, while itd-37 places the spec template in abcd's native Go binary

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/disciplines/itd-37-modification-grammar.md:55` — "carried by the native spec template in the Go binary"
- End B: `.abcd/development/intents/shipped/itd-34-three-intent-kinds.md:21` — "A standalone intent maps 1:1 to a flow-next `spec`"

itd-34 states that a standalone intent maps 1:1 to a flow-next spec and is planned through /flow-next:plan and plan-review; itd-37 says every spec carries its Modification Grammar section through the native spec template in the Go binary. The two name different spec stores and planning paths for the same promotion, so a builder cannot tell which one intent plan mints into or which template carries the inherited gates.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 1134.
