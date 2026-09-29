---
schema_version: 1
id: "iss-2609291942520919"
slug: "itd-111-acceptance-criterion-6-after-provisioning-fetched-a"
severity: "minor"
category: "drift"
source: "review-followup"
found_during: "autonomous run A resumed 2026-09-25: fidelity audit itd-111"
origin: researcher-authored
production_mode: hand-written
deferred_after: "v0.11.1"
deferral_reason: "Honouring the plugin-cache reference spc-22 names needs a design choice the record does not settle, so nothing is built yet. Three questions are open: which process records the last-reported version (the session-start hook, which itd-111 design decision 1 says never writes, or the bootstrap that fetches the binary); where the record lives when the plugin data directory comes from the environment, which the cache attestation (GHSA-4q78-ccfv-f374) treats as untrusted; and whether it replaces or sits beside the per-repo setup_version comparison that ships today. Owed: a ruling on who writes the record and where."
---

itd-111 acceptance criterion 6 (after provisioning fetched a new pinned binary, the next session reports the version transition performed) is delivered as a per-repo comparison: ahoy.VersionTransition (internal/core/ahoy/vintage.go:220-234) compares core.Version against meta.setup_version in the repo's .abcd/config.json, written by ahoy install, and reports no transition when either side is a dev build. spc-22 stated the reference would be recorded beside the plugin-cache metadata. So a repo never set up by ahoy install, or a dogfood dev build, sees no transition report after provisioning fetched a new binary, which is the scenario the criterion describes. Either the plugin-cache record the spec named is the reference to consult, or the narrowing is recorded on the criterion.

## Deferral 2026-09-29

Deferred past v0.11.1: Honouring the plugin-cache reference spc-22 names needs a design choice the record does not settle, so nothing is built yet. Three questions are open: which process records the last-reported version (the session-start hook, which itd-111 design decision 1 says never writes, or the bootstrap that fetches the binary); where the record lives when the plugin data directory comes from the environment, which the cache attestation (GHSA-4q78-ccfv-f374) treats as untrusted; and whether it replaces or sits beside the per-repo setup_version comparison that ships today. Owed: a ruling on who writes the record and where.
