---
schema_version: 1
id: "iss-2610262040499690"
slug: "itd-5-says-abcd-ships-14-now-15-agents"
severity: "nitpick"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 957 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/disciplines/itd-5-prompt-quality-additions.md:25"
remedy: "none (filed automatically)"
related_intents: [itd-5]
finding_ref: cfk-6491a5eee2c261d3
---

itd-5 says abcd ships 14 (now 15) agents under its rule; the internals index says eighteen agent prompts ship.

A premise contradiction (severity nitpick) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/disciplines/itd-5-prompt-quality-additions.md:25` — "abcd ships all 14 (now 15) agents under this rule."
- End B: `.abcd/development/brief/05-internals/README.md:7` — "The eighteen shipped agent prompts"

itd-5 gives the shipped roster its rule covers as 14, now 15. The internals index describes eighteen shipped agent prompts. Both are counts of the same shipped set.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 957.
