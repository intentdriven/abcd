---
schema_version: 1
id: "iss-2610262031532200"
slug: "itd-199-ships-a-scope-operand-at"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 192 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/shipped/itd-199-a-reading-is-about-something-narrower.md:19"
remedy: "none (filed automatically)"
related_intents: [itd-199]
finding_ref: cfk-503260010f74889f
---

itd-199 ships a scope operand at invocation that it says was 'carried into brief invariant 15'; invariant 15 says that operand is superseded and the operands are position and target only.

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/shipped/itd-199-a-reading-is-about-something-narrower.md:19` — "A scope may be widened or narrowed at invocation while calibrating"
- End B: `.abcd/development/brief/02-constraints/03-invariants.md:167` — "which supersedes adr-58's scope operand and restores the 2026-08-28 M8 ruling"

itd-199 adds a third operand (a record id, a family or a preset name), says 'A scope may be widened or narrowed at invocation while calibrating', records the ruling as adr-58 and says it was carried into invariant 15; it also has the comparative position refuse because no channel exists. Invariant 15 says the operands are a position and a target state and nothing else, under an ADR 'which supersedes adr-58's scope operand and restores the 2026-08-28 M8 ruling', and has the comparative position derive its widening run from the record. The shipped intent still describes an operand as live that the invariant records as withdrawn.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 192.
