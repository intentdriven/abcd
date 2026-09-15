---
schema_version: 1
id: "iss-2608300108376943"
slug: "bundle-member-has-no-shared-spec-path"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "cold-reading workstream Phase 2 planning, 2026-08-30"
found_at: "internal/core/intent/lifecycle.go (Link), internal/core/spec/spec.go (Intent field)"
---

A bundle of intents cannot share one spec through the ceremony: a spec's intent back-link is a single itd-N, intent link refuses a spec that names a different intent, and spec close refuses a disagreeing link, so the bundle-member kind (itd-114 vocabulary) has no shared-spec path. Ruling (13) of the cold-reading workstream binds the instrument trio (itd-183, itd-184, itd-185) as one bundle with a shared spec; the planning session stamps bundle-member and plans three specs written as one cross-linked design instead, logged as an improvisation. Either the spec back-link becomes a list, or the bundle kind's contract states that members carry sibling specs.
