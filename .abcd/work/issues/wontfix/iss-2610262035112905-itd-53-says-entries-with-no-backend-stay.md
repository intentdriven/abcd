---
schema_version: 1
id: "iss-2610262035112905"
slug: "itd-53-says-entries-with-no-backend-stay"
severity: "nitpick"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 554 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/shipped/itd-53-review-queue-auto-drain-fidelity-gate.md:35"
remedy: "none (filed automatically)"
related_intents: [itd-53]
finding_ref: cfk-028e6b59d1ecda3a
wontfix_reason: "Closed in the 2026-10-27 triage (dcn-2610271112330536): a later record already settles which end holds. The code names the state owed (internal/core/capture/auditowed.go) and has no deferred state for fidelity reviews; itd-53's banner drops the drainer that used the old word."
---

itd-53 says entries with no backend stay deferred; the intent chapter says such entries stay owed

A terminology drift (severity nitpick) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/shipped/itd-53-review-queue-auto-drain-fidelity-gate.md:35` — "No-backend behavior: entries stay `deferred` (not failed), so a headless run never blocks on an unreachable backend."
- End B: `.abcd/development/brief/04-surfaces/05-intent.md:328` — "with no auditor available every entry stays owed and the summary says why nothing ran"

itd-53 names the no-backend state of a queued review deferred. The intent chapter describes the same situation, no auditor available, as every entry staying owed, and its receipt vocabulary (owed, ingested, dead-lettered) has no deferred state. The same state is named with two different words.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 554.

## Grounds

- declined: Closed in the 2026-10-27 triage (dcn-2610271112330536): a later record already settles which end holds. The code names the state owed (internal/core/capture/auditowed.go) and has no deferred state for fidelity reviews; itd-53's banner drops the drainer that used the old word.
