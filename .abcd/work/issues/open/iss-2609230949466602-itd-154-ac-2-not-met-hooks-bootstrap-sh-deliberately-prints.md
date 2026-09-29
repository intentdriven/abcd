---
schema_version: 1
id: "iss-2609230949466602"
slug: "itd-154-ac-2-not-met-hooks-bootstrap-sh-deliberately-prints"
severity: "minor"
category: "drift"
source: "user-observation"
found_during: "autonomous run 2026-09-23 fidelity audit"
origin: researcher-authored
production_mode: hand-written
deferred_after: "v0.11.1"
deferral_reason: "ruling owed to the product thinker (renewed by run A 2026-09-29 after the v0.10.0 grant lapsed at the v0.11.0 anchor): Amend itd-154 ac-2 to drop the eager 'provisioning' line (only the first stderr line reaches the transcript), or must every fetch print it?"
remedy: "Make the record and the delivery agree, per the owed ruling: if amended, add an Audit Notes line to itd-154 (and spc-47 AC2) saying no eager provisioning line is printed because only the first stderr line reaches the transcript, citing iss-208; if every fetch must announce itself, build the breadcrumb iss-2608282026177429 describes rather than a second stderr line. Prove the amendment with record-lint, the breadcrumb with that record's kill -9 detector."
---

itd-154 ac-2 not met: hooks/bootstrap.sh deliberately prints no eager 'provisioning the abcd binary…' line when it fetches the binary (the phrase is reserved for the EXIT trap's death report), so a successful provisioning run emits only the terminal 'abcd bootstrap: installed' line; the intent's criterion promised a visible loud-staged provisioning line on every fetch. The reasoning (only the first stderr line reaches the transcript, iss-208/iss-207) is recorded in the script, not in the intent or its spec — the record and the delivery disagree and one of them should move

## Remedy grounds (2026-09-29)

- This is the same shipped criterion as iss-2608282026177429 (both name itd-154 ac-2 / spc-47 AC2 and the first-stderr-line conflict), so the remedy defers the build branch to that record's breadcrumb design rather than inventing a second.
- Rejected: printing the literal line first, which the adversarial review showed takes the only transcript line from a refusal's cause.
