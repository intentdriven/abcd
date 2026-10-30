---
schema_version: 1
id: "iss-2610262039106613"
slug: "itd-44-routes-through-an-abcd-intent-new"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 844 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/drafts/itd-44-fourth-intent-kind-decision.md:111"
remedy: "In itd-44 drop the 'abcd-intent-new skill is wrapper-only' justification and route the new `decision` verdict through the `/abcd:intent` command page and its binary verb; abcd ships no skills."
related_intents: [itd-44]
finding_ref: cfk-d161ce3aa9b71ae6
---

itd-44 routes through an `abcd-intent-new` skill, but the surfaces register says abcd ships zero skills

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/drafts/itd-44-fourth-intent-kind-decision.md:111` — "because the `abcd-intent-new` skill is wrapper-only and has no Python"
- End B: `.abcd/development/brief/04-surfaces/README.md:382` — "abcd ships zero skills"

itd-44 justifies its new CLI primitive by saying the `abcd-intent-new` skill is wrapper-only, and has that skill call the CLI and route the result. The surfaces register says abcd ships zero skills: the `/abcd:` surface is commands only, and there is no `skills/` directory. The caller itd-44 is designed around does not exist, so its routing has nowhere to live.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 844.
