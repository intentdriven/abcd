---
schema_version: 1
id: "iss-2609251238184553"
slug: "the-launch-preview-s-retention-plan-reads-a-shallow-checkout"
severity: "minor"
category: "bug"
source: "user-observation"
found_during: "autonomous run A resumed 2026-09-25"
origin: researcher-authored
production_mode: hand-written
---

The launch preview's retention plan reads a shallow checkout's tag listing as the whole release set: in a shallow clone the listing succeeds but holds only the tags that were fetched, so the plan can preview nothing to prune (or keep the wrong release) for a repository whose tags were never all seen. The listing-error half was fixed under iss-194 (computeRetentionForReport in internal/core/launch/dryrun.go); a SUCCESSFUL but incomplete listing is still indistinguishable from a complete one. Remedy: refuse the plan with its reason when git reports the checkout is shallow.
