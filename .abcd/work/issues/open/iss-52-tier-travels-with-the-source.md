---
schema_version: 1
id: "iss-52"
slug: "tier-travels-with-the-source"
severity: "minor"
category: "process"
source: "agent-finding"
found_during: "2026-07-09 practice/MVP/tool extraction"
found_at: ".abcd"
deferred_after: "v0.11.1"
deferral_reason: "ruling owed to the product thinker (lapsed-deferral triage, run A 2026-09-29): Should every corpus source carry an epistemic tier (peer-reviewed, preprint, practitioner, ai-generated) recorded at ingest, shown by consult and required by the ledger lint? The shipped corpus (itd-76) records a class and a permission, and ai-generated-never-cite covers only the AI-generated half."
remedy: "Waits on the epistemic-tier ruling: if adopted, fold it into the sources-tools intent ruled for planning in J5: abcd source add requires --tier from a closed list (peer-reviewed, preprint, practitioner, ai-generated) stored beside permission_status, consult shows it, and source ledger refuses an influence line whose source has no tier, existing entries backfilled by the J6 pass; if declined, move the record to wontfix naming ai-generated-never-cite as the half kept. Prove it with source add refusing an entry without a tier and a ledger test refusing an untiered source."
---

Every corpus source carries an epistemic tier — peer-reviewed, preprint, practitioner, or ai-generated — recorded at ingest time and surfaced by consult, so that the trust level of a source travels with the source rather than living in the reader's head. The convention serves evidence discipline: AI-generated sources support nothing on their own and may only corroborate claims grounded elsewhere, and a consult result that hides tier invites laundering weak evidence into strong claims. Detector: the ledger lint refuses influence edges whose source lacks a tier; acceptance is an ingest path that cannot register a source without one and a consult output that displays it.

## Remedy grounds (2026-09-29)

- Why: the owed question's premise that ingest lives outside the binary is stale; source add and source ledger are verbs (internal/core/source/add.go), so the tier has a home at registration.
- Sources (consulted 2026-09-29): the GRADE handbook rates the certainty of a body of evidence per outcome, not a single source (https://gdt.gradepro.org/app/handbook/handbook.html), so a per-source field records publication status rather than a GRADE level.
- Rejected: GRADE's four certainty levels, and checklist scores that are a reader's judgement per use rather than a property of the source.
