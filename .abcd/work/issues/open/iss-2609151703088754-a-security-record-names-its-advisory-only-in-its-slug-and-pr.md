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
deferred_after: "v0.9.0"
deferral_reason: "Routed to the product thinker by the 2026-09-23 run (planning owed: a typed advisory field on issue records carried to the release workflow's advisory publication; no planned intent). The 2026-09-23 interview gave routed minor and nitpick captures the default: deferred past v0.9.0, returning at the next anchor."
---

A security record names its advisory only in its slug and prose (ghsa-gx3m-..., 'GHSA-gx3m-3224-qqcv (CWE-426...)'), so nothing can match a resolved record to the draft advisory it fixes without reading the text. The publication step the advisory-handling pilot note names as its target (the GHSAs publish after the cut) needs a typed field on the issue record, e.g. advisory: GHSA-xxxx-xxxx-xxxx, admitted by issueschema, written by capture through a flag, carried through resolve and wontfix unchanged, and read by the release workflow to publish or close the advisory. Today GHSA-gx3m-3224-qqcv is fixed in v0.8.0 (record iss-2609012039107700 resolved 2026-09-08) and still an unpublished draft, which is the gap a mechanical match would have closed. Split out of the publication intent at the 2026-09-15 routing so the field can land first on its own.
