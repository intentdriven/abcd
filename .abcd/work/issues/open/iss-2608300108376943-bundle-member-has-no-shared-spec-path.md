---
schema_version: 1
id: "iss-2608300108376943"
slug: "bundle-member-has-no-shared-spec-path"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "cold-reading workstream Phase 2 planning, 2026-08-30"
found_at: "internal/core/intent/lifecycle.go (Link), internal/core/spec/spec.go (Intent field)"
deferred_after: "v0.11.1"
deferral_reason: "ruling owed to the product thinker (renewed by run A 2026-09-29 after the v0.10.0 grant lapsed at the v0.11.0 anchor): Does a spec's intent back-link become a list, or does the bundle-member contract say members carry sibling specs?"
remedy: "Waits on the spec back-link ruling (rulings-owed): if a list: a spec's intent field becomes a list and intent link and spec close accept any listed member, proven by a bundle test that closes one shared spec and ships every member; if sibling specs: glossary core/bundle.md states that bundle members carry their own cross-linked specs, matching how itd-183, itd-184 and itd-185 were planned."
---

A bundle of intents cannot share one spec through the ceremony: a spec's intent back-link is a single itd-N, intent link refuses a spec that names a different intent, and spec close refuses a disagreeing link, so the bundle-member kind (itd-114 vocabulary) has no shared-spec path. Ruling (13) of the cold-reading workstream binds the instrument trio (itd-183, itd-184, itd-185) as one bundle with a shared spec; the planning session stamps bundle-member and plans three specs written as one cross-linked design instead, logged as an improvisation. Either the spec back-link becomes a list, or the bundle kind's contract states that members carry sibling specs.

## Remedy grounds (2026-09-29)

The sibling-spec form is already practised, so the second answer costs one glossary sentence; the first changes a schema field every spec reader parses. Rejected: a bundle-level spec with no back-link, which spec close could not tie to any intent.
