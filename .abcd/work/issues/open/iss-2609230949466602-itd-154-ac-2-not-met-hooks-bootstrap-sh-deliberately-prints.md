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
deferred_after: "v0.10.0"
deferral_reason: "ruling owed to the product thinker (away; run A 2026-09-25): Amend itd-154 ac-2 to drop the eager 'provisioning' line (only the first stderr line reaches the transcript), or must every fetch print it?"
---

itd-154 ac-2 not met: hooks/bootstrap.sh deliberately prints no eager 'provisioning the abcd binary…' line when it fetches the binary (the phrase is reserved for the EXIT trap's death report), so a successful provisioning run emits only the terminal 'abcd bootstrap: installed' line; the intent's criterion promised a visible loud-staged provisioning line on every fetch. The reasoning (only the first stderr line reaches the transcript, iss-208/iss-207) is recorded in the script, not in the intent or its spec — the record and the delivery disagree and one of them should move
