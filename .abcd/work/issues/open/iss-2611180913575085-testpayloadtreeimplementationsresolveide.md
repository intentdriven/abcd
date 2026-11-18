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
remedy: "Have bundleTreeFor fail with the rejected reasons when ResolveBundle includes nothing, so a fail-closed resolution is named rather than read as a disagreement; then establish from those reasons which path fired (gitignore_check_failed from the ignore probe, or the untagged fs_error an uncertain inode map gives) and fix that one."
---

TestPayloadTreeImplementationsResolveIdentically in internal/core/launch failed once in the merge queue's race lane on 2026-11-18. The run was #1615's queue entry, and the test took 336s. The bundle side resolved to no entries while the rendered directory held the full surface, so the error read as bundle and directory resolution disagreeing. #1615 changes internal/core/identity and internal/core/provenance, which launch's test binary links, but nothing on the failing test's call path (ResolveBundle, ResolveInstallSurface, the rendering) reaches them, and #1611's entry just before it passed. Two fail-closed paths in the resolver produce exactly this symptom, and the log cannot tell them apart because the test prints only the entries. One is the ignore probe at internal/core/launch/bundle.go:483: when git check-ignore errors, every survivor is rejected with gitignore_check_failed. The other is the inode map: buildInodeMap walks the repository root, .git/ included, and any ReadDir or Lstat error (a lock file a concurrent git command creates and removes mid-walk would do) marks the map uncertain, after which classifyRegular at bundle.go:308 rejects every regular file with an untagged fs_error. Either way the bundle comes back empty with no error, and the test compares an empty list instead of naming the failure.
