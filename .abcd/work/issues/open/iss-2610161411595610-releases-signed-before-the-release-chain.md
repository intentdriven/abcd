---
schema_version: 1
id: "iss-2610161411595610"
slug: "releases-signed-before-the-release-chain"
severity: "minor"
category: "documentation"
source: "impl-review"
found_during: "security review of the release-chain fix"
origin: researcher-authored
production_mode: hand-written
found_at: "docs"
remedy: "Add a cutover note to the release runbook naming the first release the pinned checks accept and what to do for an older one."
---

Releases signed before the release-chain fix carry a tag-ref signing identity, so the new pinned provenance checks and the emergency redeploy refuse them; the release runbook does not say so.
