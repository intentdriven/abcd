---
schema_version: 1
id: "iss-2610262045357230"
slug: "itd-34-maps-a-standalone-intent-to-a"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 1284 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/shipped/itd-34-three-intent-kinds.md:21"
remedy: "none (filed automatically)"
related_intents: [itd-34, itd-80]
finding_ref: cfk-ec698684c3beaff8
---

itd-34 maps a standalone intent to a flow-next spec made by /flow-next:plan; itd-80 has intent plan mint a native spc-N record

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/shipped/itd-34-three-intent-kinds.md:21` — "A standalone intent maps 1:1 to a flow-next `spec`"
- End B: `.abcd/development/intents/shipped/itd-80-intent-lifecycle-automation.md:31` — "**A minimal native spec store** (`internal/core/spec`): directory-as-truth spec records"

itd-34 says a standalone intent maps one to one to a flow-next spec and that its plan path calls /flow-next:plan and plan-review. itd-80 builds a native spec store in internal/core/spec, where abcd intent plan mints the spc-N record and the link itself. The two shipped records disagree on which store a spec lives in and which tool makes it.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 1284.
