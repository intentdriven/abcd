---
schema_version: 1
id: "iss-2610262040552734"
slug: "the-glossary-has-a-pluggable-adapter"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 964 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/glossary/core/loop.md:39"
remedy: "none (filed automatically)"
related_intents: [itd-2610222215485265]
finding_ref: cfk-c2e3a0f600f5f051
---

The glossary has a pluggable adapter supply the run loop; the drain draft puts the driving loop inside `abcd drain` itself.

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/glossary/core/loop.md:39` — "a pluggable adapter supplies the loop itself"
- End B: `.abcd/development/intents/planned/itd-2610222215485265-abcd-drain-run-from-a-plain-terminal.md:79` — "The driver is a loop inside `abcd drain`, entered only by `--drive`"

loop.md says abcd only defines the run-loop contract and an adapter supplies the loop itself. itd-2610222215485265 decides that the driver which works through ready issues lane by lane is a loop inside the `abcd drain` verb, not an adapter. The draft builds the loop the glossary says abcd never owns.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 964.
