---
schema_version: 1
id: "iss-2610281754577313"
slug: "abcd-site-setup-creates-the-site-render"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "manual-capture"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/site/setup.go"
remedy: "Make desiredPolicies the default branch alone, word every setup step and message to match, report an existing tag rule as one to remove without rewriting it, and have the written workflow relay a published release to a run on the default branch, so every render and deploy starts from it."
---

abcd site setup creates the site-render and site environments admitting the default branch and v* tags, and the site workflow it writes also runs on a published release. The intended contract is that both environments admit only the default branch, and that every run of the site workflow starts from it.
