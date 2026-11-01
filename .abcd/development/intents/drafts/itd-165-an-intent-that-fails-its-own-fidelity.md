---
id: itd-165
slug: an-intent-that-fails-its-own-fidelity
spec_id: null
kind: null
suggested_kind: null
reclassification_history: []
builds_on: []
severity: minor
---

# A failed fidelity verdict becomes work somebody can see

## Press Release

> _Seeded from a quoted-text intent capture. Expand into the full press-release narrative before planning._

## Why This Matters

When an audit judges an acceptance criterion not met, the ingest counts it into a rollup, renders it into the shipped intent's audit notes, and stops. No record is created, no exit code changes, and no gate reads it. An intent can ship with every criterion failed and the only consequence is prose nobody queries.

The verdict's consumer is the facilitator, which is a machine by default, so turning a failure into a tracked item is the loop's own bookkeeping rather than an automation somebody has to be persuaded to accept. Each not-met criterion becomes one ledger record stamped with the receipt identifier and deduplicated on it, so a re-ingest cannot mint a second record for the same failure, and the record identifier is written back into the audit notes so the edge is two-sided like every other link in the record.

An inconclusive verdict deliberately does not create a record. It means the auditor was under-fed, which is an input fault rather than a product defect, and minting records for it at volume would fill the ledger with noise and teach a reader to ignore it. It must still leave the receipt visibly outstanding, or a verdict that decided nothing becomes indistinguishable from one that passed.

What blocks depends on the facilitator's mode ([adr-2609151528057260](../../decisions/adrs/2609151528057260-three-roles-who-each-artefact-addresses.md)): a machine facilitator works the queue, and an activated human facilitator is what the queue waits for. The ratchet that would make a new failure fail a gate is deliberately not part of this: it is sequenced behind a corpus of real verdicts, because there are none yet and a ratchet baselines whatever number it finds.

## Acceptance Criteria

> _Required (the itd-1 discipline): add at least one Given-When-Then bullet describing the verifiable bar for "shipped" before this draft can be planned._

## SOTA

Surveyed on 2026-10-23; sources, evidence tiers and the local fit risks are in the [verification-assurance SOTA note](../../research/notes/2026-10-23-verification-assurance-sota.md).

- **A finding's identity is a fingerprint built to survive edits.** SARIF 2.1.0: a fingerprint "SHALL be a string that provides a stable identifier for the result", "resistant to changes that do not affect the logical identity of the result, such as ... the line number where a result appears"; `baselineState` marks each result `new`, `unchanged`, `updated` or `absent` against a baseline run (SARIF 2.1.0 specification).
- **The forge dedups on the fingerprint and closes on disappearance.** Code scanning matches results across runs "so they only appear once"; `ruleId` must be stable; a changed path creates a new alert and closes the old; an upload without fingerprints yields duplicate alerts (forge documentation, SARIF support for code scanning).
- **A bot filing one record per recurring failure states its rules in the record**: "The bots will update this with information from each new failure", "will re-open if the flake re-occurs", and will "follow the duplicate chain" if a person closes it as a duplicate (a WebKit flaky-test bug filed by the review bot).
- **Test formats keep "could not judge" apart from "failed", and still count it.** JUnit XML separates `failure` (an assertion failed) from `error` (unexpected problem running the test) and `skipped`, each with its own suite count; TAP 14 says a harness "must not treat failing SKIP test points as a test failure" yet counts them and may report a passing TODO so it can be promoted. The intent's inconclusive-mints-nothing rule matches; the counted, listed, visibly-outstanding rendering is what keeps it from reading as a pass.
- **Where research touches the intent's mechanism:** deduplicating on the receipt identifier alone stops a re-ingest minting twice, but a later verdict on the same still-unmet criterion carries a new receipt and would mint a second record. SARIF's shape keys on what the finding is (here: intent and criterion id) and records each run that found it.
- **Locally**, the verdict already carries `receipt_id`, `criterion_id`, `verifier` and per-criterion `NOT_MET` / `INCONCLUSIVE`; `findingkey.Derive` is the repository's existing fingerprint for consistency findings; the capture duplicate check matches prose.

**Path 2: native records keyed by a fingerprint shaped to SARIF, not by the receipt alone.** One record per (intent, criterion id) found `NOT_MET`, carrying the receipt that minted it and every later receipt that found it again; a re-ingest marks it `unchanged`; a later receipt that no longer finds it marks it `absent` and leaves it for the facilitator to resolve with that receipt as grounds, never an automatic close or reopen. `INCONCLUSIVE` mints nothing and renders as a counted, named outstanding set on the receipt. The key's name is versioned, as SARIF versions its fingerprint keys, so a later change to its inputs does not orphan existing records. The record id is written back into the audit notes, as the intent requires. No dependency. The independent fit-challenge runs at plan time.

## Open Questions

**Decided 2026-08-29 (product thinker).** An unmet criterion goes on the facilitator's list. The product thinker hears about it only when the shortfall changes what they asked for.

- (SOTA 2026-10-23) Dedup key: the receipt identifier as written, or a fingerprint of (intent, criterion id) with the receipts recorded on the record, so a second verdict on the same unmet criterion extends one record rather than minting another?
- (SOTA 2026-10-23) Is the fingerprint a second use of `findingkey.Derive` or a sibling with the same versioning convention, so the repository does not hold two unrelated key schemes?

## Audit Notes

_Empty. Populated by intent-auditor when intent moves to shipped/._
