---
schema_version: 1
id: "iss-2610262031337587"
slug: "itd-5-preserves-a-fixed-oracle-cascade"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 144 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/disciplines/itd-5-prompt-quality-additions.md:87"
remedy: "In itd-5 Add 4, replace 'does NOT modify the oracle cascade' and the pre-cascade selector wording with the oracle-seam wording the verification matrix already uses: capability_scope leaves the oracle seam unchanged and capability-aware routing, when built, is a pre-dispatch selector above the seam; drop the 'fixed cascade' reference to universal-patterns section 7"
related_intents: [itd-5]
finding_ref: cfk-a32d99498eaf9b44
---

itd-5 preserves a fixed oracle cascade that the dependencies chapter says does not exist

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/disciplines/itd-5-prompt-quality-additions.md:87` — "Adding `capability_scope` does NOT modify the oracle cascade"
- End B: `.abcd/development/brief/02-constraints/02-dependencies.md:35` — "There is no fixed RP→codex→in-session cascade."

itd-5 says adding capability_scope does NOT modify the oracle cascade, and designs capability routing as a selector layered above that cascade. The dependencies chapter says the oracle is host-delegated by default with opt-in adapters and that there is no fixed RP→codex→in-session cascade. The discipline's design rests on a mechanism the current architecture has removed.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 144.
