---
schema_version: 1
id: "iss-2608271804178239"
slug: "filename-slug-agreement-unenforced-two"
severity: "minor"
category: "observation"
source: "agent-finding"
found_during: "structural consistency review of .abcd/ and docs/ (2026-08-27)"
found_at: "internal/core/capture/validate.go"
resolution: "filename<->frontmatter slug agreement enforced across all four stores via the shared recordid splitter and the record_schema blocker; both drifts fixed"
impact: fix
resolved_by:
  commit: "bf734d96"
---

filename-slug agreement is unenforced across the record stores and two records have drifted: superseded itd-47's frontmatter slug embeds the foreign handle spc-12 (now a live unrelated spec) and its H1 carries the same stale prefix, and iss-2608231237300997 is the single ledger record whose filename slug disagrees with its frontmatter slug. Fix the two instances (frontmatter+H1 edit for itd-47; rename to the frontmatter slug for the iss record, id unchanged) and close the class: capture validateInvariants and the record-lint schema check filename id but never slug.