---
schema_version: 1
id: "iss-2609252243290965"
slug: "cold-reading-exclusion-of-the-reframe-record-is-by-path-a"
severity: "minor"
category: "security"
source: "impl-review"
found_during: "autonomous run A resumed 2026-09-25"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/lint"
resolution: "record-lint's cross_store_id_claim names a reframe-shaped file (rfm-N name, rfm-N id, or occasioned_by beside a before fingerprint) outside the reframe store as a blocker"
impact: fix
resolved_by:
  commit: "cbb54351"
---

Cold-reading exclusion of the reframe record is by path: a reframe-shaped file (an rfm-N name or reframe frontmatter) planted outside .abcd/work/issues/reframes/, e.g. as a brief chapter, reaches every cold reading with its grounds and body, and no gate names it, where the open/ look-alike is flagged as a mis-named issue

## Grounds

- pursued: a reframe planted as a brief chapter in a tracked tree makes record-lint exit 1 naming it, while the store, a nested fixture store and a prose mention stay clean; a planted reframe the gate passes would show it wrong
