---
schema_version: 1
id: "iss-2609300711402515"
slug: "the-drain-eligibility-reader-admits-a"
severity: "major"
category: "security"
source: "review-followup"
found_during: "autonomous run 2026-09-23"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/drainrule/drainrule.go"
remedy: "Refuse, as malformed, any candidate record that states any top-level key twice (Duplicates already returns them) whatever its status reads as, and one whose frontmatter id disagrees with the id its file name gives it; read each record through fsutil.ReadGuardedInRoot with issueschema.RecordReadLimit so a link or an oversized record refuses; map every refusal of the rule load to exit 2 on the dry run."
resolution: "Resolved: any duplicated top-level key and a frontmatter id its file name contradicts refuse as malformed; each record is read through fsutil.ReadGuardedInRoot under the record cap; every rule refusal exits 2 on the dry run. Tests in drainrule and cli drain_surface_test."
impact: fix
resolved_by:
  commit: "788f0a550"
---

The drain eligibility reader admits a candidate record that states a non-drain key twice: drainrule.Load narrowed frontmatter.Duplicates to drain_ keys, so status: accepted followed by status: superseded loads and applies on the line scanner's first-wins reading, while any YAML reader (last wins) calls the record superseded. The same reader read every store record uncapped (root.ReadFile) and trusted a frontmatter id that disagrees with the file name.

## Grounds

- pursued: status accepted then superseded, a mismatched id, a linked record and a record past 1 MiB each refuse the load; any of them yielding a rule would show it wrong
