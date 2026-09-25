---
schema_version: 1
id: "iss-2609251600023777"
slug: "site-renderer-a-top-level-three-backtick-fence-indented-one"
severity: "minor"
category: "bug"
source: "impl-review"
found_during: "autonomous run A resumed 2026-09-25"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/site/markdown.go"
---

site renderer: a top-level three-backtick fence indented one to three spaces still renders silently as a paragraph with inline code (site/markdown.go unrenderedFenceRe covers tildes and four or more backticks only). No page in docs/ or site-src/ has the shape today. Refuse it like the other unsupported fence forms, or render it.
