---
schema_version: 1
id: "iss-2609282105242542"
slug: "the-guard-s-bundled-registry-internal-core-guard-defaults"
severity: "major"
category: "security"
source: "review-followup"
found_during: "v0.11.1 release gate crosscheck (autonomous run A)"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/guard/defaults/guard.json"
deferred_after: v0.11.0
deferral_reason: "Found by the v0.11.1 release gate after the content commit; the gap already shipped in v0.11.0 (registry unchanged), and a guard-registry change needs its own tests and a security review, so it is fixed in the first lane after the v0.11.1 tag rather than re-rolling this cut"
---

The guard's bundled registry (internal/core/guard/defaults/guard.json) has one rm entry, rm-rf-after-cd-chain, so a bare `rm -rf *` or `rm -rf /` checked with `abcd guard check` returns allow and exit 0, while the guard chapter's headline promises the catch. Found by the v0.11.1 crosscheck (x-047); registry unchanged since v0.11.0.

## Deferral 2026-09-28

Deferred past v0.11.0: Found by the v0.11.1 release gate after the content commit; the gap already shipped in v0.11.0 (registry unchanged), and a guard-registry change needs its own tests and a security review, so it is fixed in the first lane after the v0.11.1 tag rather than re-rolling this cut
