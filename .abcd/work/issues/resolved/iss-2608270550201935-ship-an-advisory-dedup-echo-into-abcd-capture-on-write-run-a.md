---
schema_version: 1
id: "iss-2608270550201935"
slug: "ship-an-advisory-dedup-echo-into-abcd-capture-on-write-run-a"
severity: "minor"
category: "future-work-seed"
source: "user-observation"
found_during: "capture-dedup-echo-2026-08-27"
found_at: "internal/core/capture"
resolution: "Delivered by itd-2609212137116617, shipped in v0.11.1: abcd capture matches the new text against every open and resolved issue and every intent (drafts included) at filing, writes a duplicates: or refines: link for a likely match, prints the match, and never refuses or drops the capture (internal/core/capture/match.go matchAndLink, wired at internal/surface/cli/cli.go:4280). That is this record's non-blocking advisory echo; the periodic cross-corpus combine stays itd-25's."
impact: additive
shipped_in: v0.11.1
resolved_by:
  intent: "itd-2609212137116617"
---

Ship an advisory-dedup echo into 'abcd capture': on write, run a NON-BLOCKING similarity scan over open/ issues plus draft intents, and when a strong match is found return it in the capture result (e.g. 'captured iss-N; possible overlap: itd-91, iss-193 — review?') WITHOUT gating the write. This gives the user immediate 'this area is already tracked' reassurance while keeping capture one-line and friction-free — verifier-selects/gates-decide: the tool proposes the overlap, the human disposes (keep / wontfix / promote). Distinct from itd-84's capture-time VALIDATOR (a deterministic pre-pass that decides INTENT decomposition FILE-AS-IS/SPLIT/HOLD, not issue-duplicate detection) and from itd-87 (recurrence-escalation in capture). The deeper cross-corpus combine — merging issue clusters and promoting to intents — is a separate periodic step (e.g. a 'capture list --dedupe' view), not per-capture. Motivated 2026-08-27 by a session that recorded a duplicate of itd-91 and caught it only after the write; until this ships, the documented protocol (agent recall-checks the ledger before recording) is the gate. Candidate for promotion to an intent.

## Grounds

- pursued: the filing-time match is the echo this record asked for; shown wrong if a capture whose text closely matches an open issue is filed with no match printed and no link written
