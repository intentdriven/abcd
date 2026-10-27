---
schema_version: 1
id: "iss-2610252214410618"
slug: "abcd-work-issues-readme-md-70-140-lists"
severity: "minor"
category: "drift"
source: "agent-finding"
found_during: "brief single-source study, 2026-10-25, reader S07"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/work/issues/README.md"
remedy: "Add the seven keys, or point at internal/core/issueschema as the schema's single source of truth, as 06-capture.md:396 already does."
---

.abcd/work/issues/README.md:70-140 lists the schema as validated strictly with unknown keys rejected (:72), but leaves out remedy, origin, production_mode, duplicates, refines, deferred_after and deferral_reason, all of which internal/core/issueschema/issueschema.go:59-129 accepts.
