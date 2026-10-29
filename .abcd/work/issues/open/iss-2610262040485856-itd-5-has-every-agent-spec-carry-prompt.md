---
schema_version: 1
id: "iss-2610262040485856"
slug: "itd-5-has-every-agent-spec-carry-prompt"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 956 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/disciplines/itd-5-prompt-quality-additions.md:97"
remedy: "In itd-5's Inheritance section (line 97) and the first acceptance criterion (line 117), replace 'will carry prompt_version 1.0.0 at close' and the 1.0.0 changelog entry with itd-81's rule: an agent stays in the 0.x band until it clears its calibration corpus, and 1.0.0 is stamped at that earned lock"
related_intents: [itd-5]
finding_ref: cfk-5c7c8ce47c4fd62a
---

itd-5 has every agent spec carry prompt_version 1.0.0 at close; the verification matrix says no agent is at 1.0.0 and none is due to be.

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/disciplines/itd-5-prompt-quality-additions.md:97` — "The agent's `agents/<name>.md` will carry `prompt_version: 1.0.0` at close."
- End B: `.abcd/development/brief/06-delivery/02-verification-matrix.md:82` — "so no agent is at `1.0.0` and none is due to be"

itd-5's inheritance section has plan-review check that each agent will carry prompt_version 1.0.0 at close. The matrix says agents stay in the 0.x band until a calibration corpus is cleared, that itd-81 overrides the lock-at-close expectation, and that no agent is or is due to be at 1.0.0. The gate itd-5 lists cannot pass.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 956.
