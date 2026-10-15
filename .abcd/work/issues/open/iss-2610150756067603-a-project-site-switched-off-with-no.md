---
schema_version: 1
id: "iss-2610150756067603"
slug: "a-project-site-switched-off-with-no"
severity: "nitpick"
category: "bug"
source: "review-followup"
found_during: "single-source project site verification, 2026-10-15"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/repolint/rule_site.go"
remedy: "Skip the site gates in repolint for a project site switched off, and let lint site --project report a switched-off empty site as a note rather than a refusal"
---

A project site switched off with no abcd-written content left is flagged by bare abcd lint as an error and refused by lint site --project, although setup exempts a switched-off site from the empty check; it needs the record emptied after the site was switched on, so it is rare
