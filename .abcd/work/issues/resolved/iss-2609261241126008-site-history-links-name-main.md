---
schema_version: 1
id: "iss-2609261241126008"
slug: "site-history-links-name-main"
severity: "minor"
category: "bug"
source: "review-followup"
found_during: "autonomous run A resumed 2026-09-25: review-drain1"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/site/explorer.go"
resolution: "the history, graph and relative-target forge links go through forgeView with forgeBlob and name commits/HEAD and tree/HEAD, so no rendered link names a branch called main, and iss-2609261206437257's resolution is amended to say so"
impact: fix
resolved_by:
  commit: "3daafa42"
---

The site's commit-history links still name the main branch: internal/core/site/explorer.go and graphpage.go build commits/main/ forge links, which 404 on a repository whose default branch is not main, while iss-2609261206437257's resolution says every forge file link names blob/HEAD. The history links should go through the same helper family and name commits/HEAD/, which the forge resolves to the default branch. Found while fixing it: the record page's link for a relative non-markdown target (explorer.go, the directory or file beside a record) builds <repository>/tree/main/ or /blob/main/ the same way, so it goes through the helper too.

## Grounds

- pursued: no rendered page of the fixture site carries blob/main, tree/main or commits/main while each HEAD kind appears; a rendered page carrying any of the three would show it wrong
