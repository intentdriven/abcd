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
resolution: "Project sites no longer show git-history details: no authors, no declared models, no commit counts and no commit dates, and their build no longer reads git history. This follows the product thinker's ruling."
impact: additive
resolved_by:
  commit: "924be8072"
---

A project site still shows authorship read from git history, which abcd does not write: the health page's same-author panel names git authors, and record.json carries far more than tallies. On a project build its authorship block lists every author by name, people under "humans" and bots under "bots", each with a profile link wherever their commit address is a GitHub noreply one, and lists every model string the commits declare under "by_model". That is all the data the dropped contributors page showed, so dropping the page hid nothing: anyone can still read it from record.json. The contributors page was dropped from project sites for the same reason, pending the product thinker's confirmation

The same question covers the record's git-derived dates and counts: record.json carries the repository's history (first and last commit dates, commit count) and each record's created, entered and touched dates, all read from git, and the about page orders its newest shipped intents by the date each entered shipped. None of it carries a data-src, so the gate cannot see it.

## Grounds

- pursued: the ruling settled the question, so the fix left every git-history detail off project sites and kept abcd's own site unchanged
