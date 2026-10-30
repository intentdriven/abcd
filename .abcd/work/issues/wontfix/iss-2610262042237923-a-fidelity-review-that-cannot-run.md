---
schema_version: 1
id: "iss-2610262042237923"
slug: "a-fidelity-review-that-cannot-run"
severity: "nitpick"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 1066 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/shipped/itd-53-review-queue-auto-drain-fidelity-gate.md:35"
remedy: "none (filed automatically)"
related_intents: [itd-53]
finding_ref: cfk-38505a3b26d8251c
wontfix_reason: "Closed in the 2026-10-27 triage (dcn-2610271112330536) as a duplicate of iss-2610262035112905. Same A line and root cause; the matrix uses owed as the chapter does."
duplicates: [iss-2610262035112905]
---

A fidelity review that cannot run because no backend is reachable is 'deferred' in itd-53 and 'owed' in the verification matrix.

A naming conflict (severity nitpick) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/shipped/itd-53-review-queue-auto-drain-fidelity-gate.md:35` — "No-backend behavior: entries stay `deferred` (not failed), so a headless run never blocks on an unreachable backend."
- End B: `.abcd/development/brief/06-delivery/02-verification-matrix.md:56` — "otherwise the audit stays owed until the host runs it"

itd-53 says queue entries stay deferred, not failed, when no review backend is reachable. The matrix says that when no provider connection is reached the audit stays owed until the host runs it, and other documents call this an OWED receipt. Two state names describe one condition.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 1066.

## Grounds

- declined: Closed in the 2026-10-27 triage (dcn-2610271112330536) as a duplicate of iss-2610262035112905. Same A line and root cause; the matrix uses owed as the chapter does.
