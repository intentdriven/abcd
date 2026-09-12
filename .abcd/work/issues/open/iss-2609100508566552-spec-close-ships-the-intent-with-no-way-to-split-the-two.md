---
schema_version: 1
id: "iss-2609100508566552"
slug: "spec-close-ships-the-intent-with-no-way-to-split-the-two"
severity: "major"
category: "future-work-seed"
source: "user-observation"
found_during: "autonomous-run field experiment in a managed repository, 2026-09-09/10"
origin: researcher-authored
production_mode: hand-written
deferred_after: "v0.8.0"
deferral_reason: "Closing a spec ships its intent unconditionally, with no way to close one without the other and no way to split an intent whose criteria are only half met. Both are lifecycle changes: what it should mean to close a spec against a partially delivered intent is a question about the lifecycle's shape, and a session that met this stopped and asked rather than close, which was the right instinct and is the reason the record exists."
found_at: "internal (spec close, intent lifecycle)"
---

Closing a spec ships its intent unconditionally, and there is no way to do one without the other.

Observed in an autonomous run over a managed repository. A spec was complete and ready to close, while the intent it realised had roughly half its acceptance criteria met. `abcd spec close` moves the spec to `closed/` and, as its close-hook, moves the intent from `planned/` to `shipped/`. The worker stopped and asked rather than close, which was the right call, but the verb offered no third option: no way to close a spec without shipping its intent, and no way to split the intent so the delivered half ships and the rest stays planned.

The coupling is deliberate and mostly correct — an intent whose spec is closed has usually shipped — but it makes the shipped bucket a claim the tool will assert on the operator's behalf whether or not it is true. A shipped intent with half its criteria unmet is the false-green shape at the level of the record: the changelog derives from terminal folders, so the cut announces the whole intent, and the fidelity audit that would catch it is owed rather than performed.

Wanted: a way to close a spec while leaving its intent planned, with the reason recorded on the intent (the spec's work is done, the intent is not); or a split, minting a successor intent for the unmet criteria and shipping only what was delivered. Either makes the shipped bucket mean what it says. Failing both, `spec close` should at least refuse — or require an explicit acknowledgement — when the intent's criteria are visibly unmet, rather than moving it silently.
