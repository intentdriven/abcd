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
---

itd-111 acceptance criterion 6 (after provisioning fetched a new pinned binary, the next session reports the version transition performed) is delivered as a per-repo comparison: ahoy.VersionTransition (internal/core/ahoy/vintage.go:220-234) compares core.Version against meta.setup_version in the repo's .abcd/config.json, written by ahoy install, and reports no transition when either side is a dev build. spc-22 stated the reference would be recorded beside the plugin-cache metadata. So a repo never set up by ahoy install, or a dogfood dev build, sees no transition report after provisioning fetched a new binary, which is the scenario the criterion describes. Either the plugin-cache record the spec named is the reference to consult, or the narrowing is recorded on the criterion.
