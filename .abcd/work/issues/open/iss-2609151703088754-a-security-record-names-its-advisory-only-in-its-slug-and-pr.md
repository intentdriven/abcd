---
schema_version: 1
id: "iss-2609151703088754"
slug: "a-security-record-names-its-advisory-only-in-its-slug-and-pr"
severity: "minor"
category: "future-work-seed"
source: "user-observation"
found_during: "security-advisory triage 2026-09-15"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/issueschema"
deferred_after: "v0.11.1"
deferral_reason: "planning F owed to the product thinker: issueschema admits no advisory field and capture has no flag to write one, so a resolved security record still names its advisory only in slug and prose. A typed field carried to the release workflow has no planned intent. (re-checked at e792a2314 by lane drainDQ3, run A, 2026-09-29)"
remedy: "Waits on planning F (the advisory field): admit an optional advisory: list on issue records in internal/core/issueschema, each entry an OSV-style advisory id (GHSA-xxxx-xxxx-xxxx or CVE-YYYY-NNNN), written by a capture --advisory flag, carried unchanged through resolve and wontfix, and refused by record-lint when malformed; backfill iss-2609012039107700 with GHSA-gx3m-3224-qqcv. Prove it with schema and capture round-trip tests; the release workflow's read of the field is a later step of the publication intent."
---

A security record names its advisory only in its slug and prose (ghsa-gx3m-..., 'GHSA-gx3m-3224-qqcv (CWE-426...)'), so nothing can match a resolved record to the draft advisory it fixes without reading the text. The publication step the advisory-handling pilot note names as its target (the GHSAs publish after the cut) needs a typed field on the issue record, e.g. advisory: GHSA-xxxx-xxxx-xxxx, admitted by issueschema, written by capture through a flag, carried through resolve and wontfix unchanged, and read by the release workflow to publish or close the advisory. Today GHSA-gx3m-3224-qqcv is fixed in v0.8.0 (record iss-2609012039107700 resolved 2026-09-08) and still an unpublished draft, which is the gap a mechanical match would have closed. Split out of the publication intent at the 2026-09-15 routing so the field can land first on its own.

## Remedy grounds (2026-09-29)

- OSV gives advisory ids the form DB-ENTRYID (GHSA-..., CVE-...) and models other databases' ids as aliases of one entry: https://ossf.github.io/osv-schema/ (consulted 2026-09-29), so a list of such ids is the smallest field a publication step can match on.
- Rejected: parsing the id from the slug or prose, the text-matching the record exists to end; planning the field is owed (rulings-owed F), so this states the shape without scheduling it.
