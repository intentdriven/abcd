---
schema_version: 1
id: "iss-2608290820473197"
slug: "an-inconclusive-fidelity-verdict-is-terminal-so-an-audit-tha"
severity: "major"
category: "bug"
source: "impl-review"
found_during: "intent-implementation-run"
found_at: "internal/core/intent/audit.go"
remedy: "Waits on itd-165's planning interview: then branch the ingest in internal/core/intent/audit.go so a verdict whose criteria are all INCONCLUSIVE (or any INCONCLUSIVE, as the interview settles) leaves the receipt OWED in a distinct re-run state rather than INGESTED, re-dispatched with better inputs and escalated to the facilitator, never the product thinker. Prove it with an ingest test that an all-INCONCLUSIVE verdict keeps the OWED marker and that a later valid verdict replaces it through the existing re-ingest path."
deferred_after: v0.11.1
deferral_reason: "planning owed (re-deferred at v0.11.1 by run A's major-triage lane): the narrow fix is written down in itd-165, still in drafts with no spec, and the automatic re-dispatch the adr-55 reframe asks for is not built, so branching the ingest now would harden an unplanned draft. Owed: itd-165's planning interview, then a lane."
---

An INCONCLUSIVE fidelity verdict is terminal, so an audit that could not decide anything is indistinguishable from one that passed. The ingest does not branch on the verdict value: it rolls the per-criterion verdicts into counts and replaces the parked OWED marker with INGESTED whatever they say, so a verdict of all-INCONCLUSIVE closes the receipt exactly as a verdict of all-MET does. The re-emit verb then refuses to reopen it, reporting already_ingested and leaving the Audit Notes untouched, which is correct for a decided audit and wrong for an undecided one. The consequence is that there is no way to ensure the re-run that an INCONCLUSIVE calls for. The only lever that produces a fresh receipt is editing the acceptance-criteria section, because the receipt digest is taken over that section alone, which conflates two unrelated acts: clarifying a promise, and retrying an audit that was merely under-fed. This is a loud-staging violation in the precise sense the principle names, since a stage that degraded presents as a completed one. The narrow fix is for the ingest to branch: an INCONCLUSIVE leaves the receipt OWED, or moves it to a distinct re-run state, so the outstanding work stays visible without minting a ledger issue for what is an input fault rather than a product defect.

Reframed 2026-08-29 under adr-55: an inconclusive verdict is a stop that needs a verdict, not a terminal state. It is answered by re-dispatching with better inputs, automatically and more than once, before anything escalates. It escalates to the facilitator and never to the product thinker, because whether the evidence was sufficient is precisely what the product thinker cannot judge.

## Deferral 2026-09-29

Deferred past v0.11.1: planning owed (re-deferred at v0.11.1 by run A's major-triage lane): the narrow fix is written down in itd-165, still in drafts with no spec, and the automatic re-dispatch the adr-55 reframe asks for is not built, so branching the ingest now would harden an unplanned draft. Owed: itd-165's planning interview, then a lane.

## Remedy grounds (2026-09-29)

- The record's narrow fix is the smallest change that makes a degraded stage loud, and itd-165 already states that an inconclusive verdict mints no record yet must stay visibly outstanding.
- The re-ingest path (reingestVerdict in internal/core/intent/audit.go) lets a valid verdict replace an ingested block, which gives the re-run a way in; the ingest still writes INGESTED whatever the rollup says, so the defect stands.
- Rejected: minting a ledger issue per inconclusive verdict, which itd-165 refuses as noise.
