---
schema_version: 1
id: "iss-2610262045139907"
slug: "itd-94-has-autonomous-drain-runs"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 1260 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/shipped/itd-94-an-intent-that-is-not-planned-cannot-be.md:25"
remedy: "none (filed automatically)"
related_intents: [itd-94, itd-82]
finding_ref: cfk-8371490bb6381f69
---

itd-94 has autonomous drain runs implementing intents; itd-82 defines drain as working issues only and leaves picking intents to `abcd build next`

A terminology drift (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/shipped/itd-94-an-intent-that-is-not-planned-cannot-be.md:25` — "Autonomous drain runs get the same gate"
- End B: `.abcd/development/intents/planned/itd-82-drain-ledger-triage.md:57` — "Picking among intents: `abcd build next` (`itd-2609211116005482`), the sibling run."

itd-94 says autonomous drain runs get the intent-readiness gate, so a drain there is a run that implements intents. itd-82 defines `abcd drain` over the open issue ledger and lists picking among intents as out of scope, the job of the sibling run `abcd build next`. 'Drain' means two different runs, and itd-82's drain never reaches the intent gate that itd-94 credits it with.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 1260.
