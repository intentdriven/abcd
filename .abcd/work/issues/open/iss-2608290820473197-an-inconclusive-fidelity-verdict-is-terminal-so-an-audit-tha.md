---
schema_version: 1
id: "iss-2608290820473197"
slug: "an-inconclusive-fidelity-verdict-is-terminal-so-an-audit-tha"
severity: "major"
category: "bug"
source: "impl-review"
found_during: "intent-implementation-run"
found_at: "internal/core/intent/audit.go"
---

An INCONCLUSIVE fidelity verdict is terminal, so an audit that could not decide anything is indistinguishable from one that passed. The ingest does not branch on the verdict value: it rolls the per-criterion verdicts into counts and replaces the parked OWED marker with INGESTED whatever they say, so a verdict of all-INCONCLUSIVE closes the receipt exactly as a verdict of all-MET does. The re-emit verb then refuses to reopen it, reporting already_ingested and leaving the Audit Notes untouched, which is correct for a decided audit and wrong for an undecided one. The consequence is that there is no way to ensure the re-run that an INCONCLUSIVE calls for. The only lever that produces a fresh receipt is editing the acceptance-criteria section, because the receipt digest is taken over that section alone, which conflates two unrelated acts: clarifying a promise, and retrying an audit that was merely under-fed. This is a loud-staging violation in the precise sense the principle names, since a stage that degraded presents as a completed one. The narrow fix is for the ingest to branch: an INCONCLUSIVE leaves the receipt OWED, or moves it to a distinct re-run state, so the outstanding work stays visible without minting a ledger issue for what is an input fault rather than a product defect.

Reframed 2026-08-29 under adr-55: an inconclusive verdict is a stop that needs a verdict, not a terminal state. It is answered by re-dispatching with better inputs, automatically and more than once, before anything escalates. It escalates to the facilitator and never to the product thinker, because whether the evidence was sufficient is precisely what the product thinker cannot judge.
