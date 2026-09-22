---
id: itd-2609221017023290
slug: abcd-keeps-every-external-credential-the-same-way-one
spec_id: spc-2609221017544877
kind: standalone
suggested_kind: null
reclassification_history: []
builds_on: [itd-63]
severity: minor
impact: additive
origin: researcher-authored
production_mode: hand-written
related_intents: [itd-2609081951381895, itd-2609061543533170, itd-6]
related_adrs: [adr-2609221017021499]
---

# abcd keeps every external credential the same way

## Press Release

> **One credential store, three homes the person chooses once, and every adapter reads through it; no secret ever enters the harness or the repository.**
>
> "OpenRouter today, a hosting provider next month, my transcript cloud after that, and each one wanted a key somewhere," said a product thinker. "Now abcd asks me once per service where the secret should live, explains what it unlocks first, and every part of abcd reads it the same way. Nothing lands in the harness's settings. Nothing lands in the repo."

## Why This Matters

The first adapter to need a secret was given its own walkthrough and its own file; the second was about to be. The record's rule is one canonical primitive, and a secret is the last thing to have three copies of. Ruled 2026-09-22 (adr-2609221017021499): one store, three homes, one reader, scanned before written.

## Mechanism

We expect one store with one reader to make every secret abcd holds auditable in one place and impossible to commit by accident, because the only write path runs the scanner and the only read path is named; shown wrong if a secret is found in a harness file or a tracked path after this ships, or if an adapter is found reading one another way.

## Scope Conditions

- Holds on macOS with the Keychain and on Linux with a secret service; a platform with neither offers the two other homes and says why. <!-- cond: cond-2609221017547155 -->

## What's In Scope

- **The store** (`internal/core/credential`): `Resolve(name)` for every adapter; `Set(name, home, value)` used only by the walkthrough; homes `external` (a pointer to an existing tool's configuration or an environment variable name), `abcd` (`~/.abcd/credentials.json`, mode 0600), `keychain` (the platform keychain under abcd's service name).
- **The walkthrough** at `ahoy`, through itd-63's explain-then-install mode: what the credential unlocks, what works without it, then the three homes with the keychain recommended in prose, then a verification call the calling adapter supplies.
- **Refusals**: a name that resolves to nothing refuses naming the walkthrough; a write that would land in a tracked path is refused; the write path runs the secret scanner.
- **The readers**: the API adapter, the site setup and any later hook resolve by name; a review finding of any other read is a defect.
- **The record**: the run record names which credential names a run used, never a value.

## What's Out of Scope

- Rotating or expiring credentials.
- Sharing a credential between machines.
- Any credential the host itself holds for its own model.

## Decisions

Ruled by the product thinker on 2026-09-21, in the interview that filed and planned this intent (adr-2609221017021499 records the vocabulary rulings it rests on):

1. One store, three homes, one reader, scanned before written (adr-2609221017021499).
2. The keychain is recommended in prose; the choice is the person's.

## Open Questions

_None open._

## Acceptance Criteria

- **Given** an adapter needing a credential that is not set, **when** it resolves the name, **then** it refuses naming the walkthrough, and no unauthenticated call is made.
- **Given** the walkthrough accepted for a service, **when** it runs, **then** it explains what the credential unlocks and what works without it, offers the three homes with the keychain recommended in the prose above the choice, stores the value in the chosen home, and verifies with the adapter's own call.
- **Given** any home, **when** the tree and the harness's settings are inspected afterwards, **then** neither carries the value; a write that would land in a tracked path is refused and the write path runs the scanner.
- **Given** the API adapter and the site setup, **when** they read their credentials, **then** both call the store by name, and a test walks the adapters for any other read.
- **Given** a run, **when** its record is read, **then** it names the credential names used and no value.

## Audit Notes

_Empty. Populated by intent-auditor when intent moves to shipped/._

## Grounds

- pursued: three adapters are about to be built and each would otherwise invent its own store; we expect one store to be the one place a secret is audited; shown wrong if a secret is found in a harness file or a tracked path, or an adapter reads one another way
