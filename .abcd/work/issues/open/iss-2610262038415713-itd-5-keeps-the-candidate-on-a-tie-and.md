---
schema_version: 1
id: "iss-2610262038415713"
slug: "itd-5-keeps-the-candidate-on-a-tie-and"
severity: "major"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 812 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/disciplines/itd-5-prompt-quality-additions.md:58"
remedy: "none (filed automatically)"
related_intents: [itd-5]
finding_ref: cfk-20ad23af547e97e7
---

itd-5 keeps the candidate on a tie and says length is no tiebreak; 05-prompt-quality accepts a rewrite that ties and is more than 10% shorter

A premise contradiction (severity major) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/disciplines/itd-5-prompt-quality-additions.md:58` — "If the oracle variant scores better on the corpus, accept it; on a tie, keep the candidate. Length is not a tiebreak (see § Why)."
- End B: `.abcd/development/brief/05-internals/05-prompt-quality.md:127` — "accepts the reviewer's variant only if it scores at least as well and is more than 10% shorter"

itd-5 accepts the oracle's rewrite only if it scores better, keeps the candidate on a tie, and says length is not a tiebreak. 05-prompt-quality accepts the reviewer's variant when it scores at least as well and is more than 10% shorter. When a shorter rewrite ties, one rule locks the rewrite and the other locks the candidate, so the lock-time gate has two decision rules.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 812.
