---
id: itd-2611012048006057
slug: macos-ran-out-of-application-memory-on-a
spec_id: null
kind: null
suggested_kind: null
reclassification_history: []
builds_on: []
severity: minor
related_issues: [iss-2611011737317454]
origin: extracted-from-record
production_mode: hand-written
---

# macOS ran out of application memory on a 128 GB, 16-core machine on 2026-11-01 at about 17:32, with only this person's sessions active, while several agent sessions drained, built and reviewed in parallel. The Force Quit dialog lists only GUI apps (Ghostty at 8.93 GB, falling to 937 MB afterwards), so the go test processes, race-enabled test binaries and agent sessions running inside the terminal never show. The machine's counters show about 155 GiB swapped out since it booted five days earlier, so this was not the first time it ran past its RAM. The system log needs admin rights to read, so the peak was not attributed to one process; a read shortly after, with 22 test processes live at a load of 57, found the test workload at about 4 GiB and the agent sessions at about 3.8 GiB, so the consumer is unproven.

## Press Release

> _Seeded by promotion from iss-2611011737317454. Expand into the full press-release narrative before planning._

## Why This Matters

Graduated from `iss-2611011737317454`: macOS ran out of application memory on a 128 GB, 16-core machine on 2026-11-01 at about 17:32, with only this person's sessions active, while several agent sessions drained, built and reviewed in parallel. The Force Quit dialog lists only GUI apps (Ghostty at 8.93 GB, falling to 937 MB afterwards), so the go test processes, race-enabled test binaries and agent sessions running inside the terminal never show. The machine's counters show about 155 GiB swapped out since it booted five days earlier, so this was not the first time it ran past its RAM. The system log needs admin rights to read, so the peak was not attributed to one process; a read shortly after, with 22 test processes live at a load of 57, found the test workload at about 4 GiB and the agent sessions at about 3.8 GiB, so the consumer is unproven.. Read that issue record for the source observation.

## Mechanism

> _Prompted (the claim-recording gradient): why the authors expect this to work, as a falsifiable "we expect X because Y" — not the outcome restated. Replace this line with the claim, or with the exact token `None stated.` alone on its line to record the claim as considered and declined._

## Scope Conditions

> _Required (the claim-recording gradient): the population, platform, scale, or assumptions this claim holds under, one per top-level bullet — `abcd intent plan` stamps each with a persistent identity. Replace this line with those bullets, or with the exact token `None stated.` alone on its line._

## Acceptance Criteria

> _Required (the itd-1 discipline): add at least one Given-When-Then bullet describing the verifiable bar for "shipped" before this draft can be planned._

## Open Questions

_None recorded yet._

## Audit Notes

_Empty. Populated by intent-auditor when intent moves to shipped/._
