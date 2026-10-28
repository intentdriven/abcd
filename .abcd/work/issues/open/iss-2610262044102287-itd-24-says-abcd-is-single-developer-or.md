---
schema_version: 1
id: "iss-2610262044102287"
slug: "itd-24-says-abcd-is-single-developer-or"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 1188 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/shipped/itd-24-reflect-command.md:65"
remedy: "none (filed automatically)"
related_intents: [itd-24, itd-99]
finding_ref: cfk-1ba27dfdb13e04eb
---

itd-24 says abcd is single-developer or pair shaped; itd-99 says the product thinker can be a team

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/shipped/itd-24-reflect-command.md:65` — "abcd is single-developer-shaped (or pair-shaped); team retrospective patterns belong elsewhere."
- End B: `.abcd/development/intents/drafts/itd-99-a-team-of-product-thinkers-decides-as.md:19` — "The product thinker can be a team."

itd-24 leaves team retrospectives out of scope on the premise that abcd is single-developer-shaped, or at most pair-shaped. itd-99's premise is that the product thinker role can be held by a team of several people, each with their own thinking session and recorded dissent. The two state incompatible assumptions about how many people abcd is built for.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 1188.
