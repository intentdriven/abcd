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
---

The site internals chapter (brief 05-internals/10-site.md) lists /record/timeline/ as a page the explorer serves, but the build never emits it: the genealogy renders folded into the /record/ dashboard, and timelinePage in internal/core/site/timeline.go is unreachable code.
