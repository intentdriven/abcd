---
id: itd-2611150750323478
slug: abcd-implement-load-s-help-says-less
spec_id: null
kind: null
suggested_kind: null
reclassification_history: []
builds_on: []
severity: minor
related_issues: [iss-2610252211480171]
origin: extracted-from-record
production_mode: hand-written
---

# abcd implement load's help says less than the verb does, in three places. (1) Its sentence (internal/core/surface/sentences.go:195-196; page line 2286) says 'Writes a run-log event or the wait's hold', but --wait also renews the caller's reservations, each renewal a commit pushed to origin (internal/surface/cli/implement_load.go:283,334-392; internal/core/claim/renewdue.go:45-64; claim/remote.go:155-159), added in 6931d4ac4. (2) The Long help says 'Without --wait it never refuses, never waits and never stops anything' (implement_load.go:65-66; page line 2298), yet --race-gate without --wait waits (implement_load.go:221-232), and an unknown --site or a stray --timeout is refused at exit 2 (implement_load.go:234-240). (3) The Long help names a symlinked ~/.abcd.noindex as a refusal (implement_load.go:186-187) but not the refusal under a HOME inside a git working tree that is not the account's own (internal/core/implement/load_wait.go:473-478, cd798880f), which commands/implement.md states; and it says the waiter renews 'each of your live reservations' (implement_load.go:189-191) while release reservations are never renewed (renewdue.go:53). The --wait flag line is already held by iss-2610141706070227.

## Press Release

> _Seeded by promotion from iss-2610252211480171. Expand into the full press-release narrative before planning._

## Why This Matters

Graduated from `iss-2610252211480171`: abcd implement load's help says less than the verb does, in three places. (1) Its sentence (internal/core/surface/sentences.go:195-196; page line 2286) says 'Writes a run-log event or the wait's hold', but --wait also renews the caller's reservations, each renewal a commit pushed to origin (internal/surface/cli/implement_load.go:283,334-392; internal/core/claim/renewdue.go:45-64; claim/remote.go:155-159), added in 6931d4ac4. (2) The Long help says 'Without --wait it never refuses, never waits and never stops anything' (implement_load.go:65-66; page line 2298), yet --race-gate without --wait waits (implement_load.go:221-232), and an unknown --site or a stray --timeout is refused at exit 2 (implement_load.go:234-240). (3) The Long help names a symlinked ~/.abcd.noindex as a refusal (implement_load.go:186-187) but not the refusal under a HOME inside a git working tree that is not the account's own (internal/core/implement/load_wait.go:473-478, cd798880f), which commands/implement.md states; and it says the waiter renews 'each of your live reservations' (implement_load.go:189-191) while release reservations are never renewed (renewdue.go:53). The --wait flag line is already held by iss-2610141706070227.. Read that issue record for the source observation.

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
