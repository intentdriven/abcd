---
schema_version: 1
id: "iss-2610150706489874"
slug: "a-project-site-still-shows-authorship"
severity: "minor"
category: "inconsistency"
source: "review-followup"
found_during: "single-source project site fix round, 2026-10-15"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/site/health.go"
remedy: "Follow the product thinker's ruling on the contributors page: if git authorship counts as not written by abcd, leave the same-author panel and the authorship tallies out of project sites too; if it counts as part of the record, restore the contributors page"
---

A project site still shows authorship read from git history, which abcd does not write: the health page's same-author panel names git authors, and record.json carries authorship tallies. The contributors page was dropped from project sites for the same reason, pending the product thinker's confirmation
