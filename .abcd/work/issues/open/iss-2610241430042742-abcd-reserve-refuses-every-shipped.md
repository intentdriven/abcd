---
schema_version: 1
id: "iss-2610241430042742"
slug: "abcd-reserve-refuses-every-shipped"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "owed fidelity-review drain, 2026-10-24"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/claim/claim.go"
remedy: "Give a post-ship act on a shipped intent a reservation the finished-record back-off exempts: reserve the review receipt (rcp-N) or take a flag naming the act, made and renewed by intent audit <itd-N> and intent audit --owed, released by the ingest, and backed off on exit 3 by a second copy; document it in commands/reserve.md and commands/intent.md; prove it with a claim test that an audit reservation of a shipped intent is made while a plain one still backs off, and an intent test that --owed in a second copy skips a reserved receipt. Grounds: internal, the concurrency rule in AGENTS.md and the back-off in internal/core/claim."
---

abcd reserve refuses every shipped intent as finished on origin's default branch (exit 3, nothing pushed), yet an owed fidelity review writes into the shipped intent's file (its Audit Notes block) and can capture an issue, so the work an intent audit drain does can never be reserved, despite the AGENTS.md rule to reserve an intent before working on it. Two sessions paying owed reviews, or running intent condition, can pick the same shipped intent and each spend a whole review on it, with only the ingest's replace-or-noop keeping the record whole. Seen on 2026-10-24: reserve itd-3 before its owed audit exited 3 with 'itd-3 is finished on origin's default branch'.
