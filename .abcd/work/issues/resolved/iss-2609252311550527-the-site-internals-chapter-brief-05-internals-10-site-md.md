---
schema_version: 1
id: "iss-2609252311550527"
slug: "the-site-internals-chapter-brief-05-internals-10-site-md"
severity: "minor"
category: "drift"
source: "user-observation"
found_during: "autonomous run A resumed 2026-09-25"
origin: researcher-authored
production_mode: hand-written
resolution: "The internals chapter now places the genealogy in the dashboard, and the dead timeline page code is removed."
impact: internal
resolved_by:
  commit: "501e0896b04eb1233fe2e3eafbeb803b6f1e9fc8"
---

The site internals chapter (brief 05-internals/10-site.md) lists /record/timeline/ as a page the explorer serves, but the build never emits it: the genealogy renders folded into the /record/ dashboard, and timelinePage in internal/core/site/timeline.go is unreachable code.

## Grounds

- pursued: the chapter names only pages the build writes; shown wrong if a build emits a page the chapter omits or the chapter names one no build emits
