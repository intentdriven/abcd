---
schema_version: 1
id: "iss-2609261241117925"
slug: "forge-list-total-not-held-across-pages"
severity: "minor"
category: "security"
source: "review-followup"
found_during: "autonomous run A resumed 2026-09-25: review-drain1"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/site/forge.go"
---

The forge list read holds no total across pages: internal/core/site/forge.go listAll reads total_count afresh on every page and never compares it with page 1's, so a list that shrinks between two page reads (page 1 total 150 serving entries 0-99, one entry deleted, page 2 total 149 serving the old 101-149) returns 149 entries with no error and the old entry 100 is never seen. If that entry is an existing environment, setup reads it as absent and on confirm PUTs the endpoint that replaces its whole protection set. The read should hold page 1's total and fail closed on a later page reporting a different one.
