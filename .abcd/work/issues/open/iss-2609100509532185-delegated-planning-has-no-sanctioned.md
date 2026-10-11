---
schema_version: 1
id: "iss-2609100509532185"
slug: "delegated-planning-has-no-sanctioned"
severity: "minor"
category: "future-work-seed"
source: "user-observation"
found_during: "autonomous-run field experiment in a managed repository, 2026-09-09/10"
origin: researcher-authored
production_mode: hand-written
found_at: "internal (intent plan) / conventions"
deferred_after: "v0.11.1"
deferral_reason: "ruling owed to the product thinker (away; run A 2026-09-25; rulings-owed H): Add an intent plan --delegated-by stamp and a gate for delegated planning?"
remedy: "Waits on ruling (a --delegated-by stamp and its gate): if yes, add intent plan --delegated-by <who> writing delegated_by and the date onto the planned intent, and a gate refusing a plan that carries neither the stamp nor interview answers on the record (the answers half rides iss-2609200830002711), proven by a plan test for each case; if no, wontfix, keeping planning human-only with no delegated path."
---

The planning interview is human-only by rule, so an autonomous run has no sanctioned way to record that the human delegated it.

Observed in an autonomous run that took one intent from filing through planning to implementation without the human interview. The convention is right — planning is where the human's intent is fixed, and an agent inventing it is the failure the interview exists to prevent — but delegation is a real and legitimate case: a human who has decided to let the agent plan a specific intent on a specific date has made a decision, and the record has nowhere to put it. The only available shape was a free-text line in the decisions log, which is unauditable: nothing distinguishes it from an agent asserting delegation that never happened, and nothing links it to the intent it authorises.

The asymmetry is the point. Every other authority-bearing act in abcd is recorded as a field on the record it affects, checkable by a gate. Delegated planning is recorded, when it is recorded, as prose.

Wanted: a flag on the planning path — `abcd intent plan --delegated-by <who>` or similar — that stamps the intent with who delegated and when, so a reader of the planned intent can see that a human authorised the agent to plan it, and a gate can require the stamp before accepting a plan that no interview produced. That turns an unverifiable claim in a log into a field on the record it belongs to.

## Remedy grounds (2026-09-29)

Why: every other authority-bearing act is a field on the record it affects, so delegation becomes one too and a gate can read it. Rejected: a decisions-log line alone, which the record shows is indistinguishable from an agent asserting a delegation that never happened.
