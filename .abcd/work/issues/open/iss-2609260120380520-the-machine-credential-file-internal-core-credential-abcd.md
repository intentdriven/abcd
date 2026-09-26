---
schema_version: 1
id: "iss-2609260120380520"
slug: "the-machine-credential-file-internal-core-credential-abcd"
severity: "minor"
category: "security"
source: "impl-review"
found_during: "autonomous run A resumed 2026-09-25 (fix round, review of lane sitesetup)"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/credential/credential.go"
deferred_after: "v0.10.0"
deferral_reason: "deferred to the integration step (run A, 2026-09-26): the strict duplicate-key decoder jsonstrict lives on the unmerged lintB lane and copying it here would fork it; once lintB lands, credential.go reroutes its decode through jsonstrict and this record is resolved there"
---

The machine credential file (internal/core/credential, ~/.abcd/credentials.json) is decoded with plain json.Unmarshal, so a duplicate key silently takes the last value instead of being refused; the strict duplicate-key decoder (jsonstrict) lives on an unmerged lane, not on this base.
