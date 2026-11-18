---
schema_version: 1
id: "iss-2611180913575085"
slug: "testpayloadtreeimplementationsresolveide"
severity: "minor"
category: "bug"
source: "user-observation"
found_during: "merge-queue failure of #1615"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/launch/installsurface_test.go"
remedy: "Have bundleTreeFor fail with the rejected reasons when ResolveBundle includes nothing, so a failed gitignore probe is named rather than read as a disagreement; then establish whether check-ignore fails under race-lane load and retry or bound it."
---

TestPayloadTreeImplementationsResolveIdentically in internal/core/launch failed once in the merge queue's race lane on 2026-11-18. The run was #1615's queue entry, and the test took 336s. The bundle side resolved to no entries while the rendered directory held the full surface, so the error read as bundle and directory resolution disagreeing. #1615 changes nothing launch depends on, and #1611's entry just before it passed. The likely cause is resolveBundle's fail-closed path at internal/core/launch/bundle.go:483: when the git check-ignore probe errors (here plausibly under race-lane load), every survivor is rejected with gitignore_check_failed. The bundle then comes back empty with no error, and the test compares an empty list instead of naming the probe failure.
