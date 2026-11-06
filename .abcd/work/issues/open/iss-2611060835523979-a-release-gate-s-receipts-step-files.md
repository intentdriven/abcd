---
schema_version: 1
id: "iss-2611060835523979"
slug: "a-release-gate-s-receipts-step-files"
severity: "minor"
category: "process"
source: "agent-observation"
found_during: "v0.16.2 cut"
origin: researcher-authored
production_mode: hand-written
found_at: "a release gate's receipts step"
remedy: "Pass the source in the receipts step, take each entry's severity from its record, and let two findings share a record or link them."
---

A release gate's receipts step files agents' findings as `user-observation`, because its capture call passes no `--source`: all 85 captures of the v0.16.2 gate did. Six of them duplicate an open record with no link, because a deferred finding must have its own record, so closing one leaves the other stale; and two receipt entries grade minor where their record says major.
