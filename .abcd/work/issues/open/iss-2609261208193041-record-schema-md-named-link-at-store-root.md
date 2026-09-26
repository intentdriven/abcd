---
schema_version: 1
id: "iss-2609261208193041"
slug: "record-schema-md-named-link-at-store-root"
severity: "minor"
category: "bug"
source: "review-followup"
found_during: "autonomous run A resumed 2026-09-25: fix4-lintA sweep"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/lint/schema.go"
---

record_schema still passes silently over one link shape at a record store root: a link whose name ends in .md but does not match the record filename pattern (for example notes.md pointing at a directory). A real directory with that name is reported; the link is not, because telling whether it points at a directory would mean following it, which the walk never does. Reporting every link at a store root other than README.md, without following it, closes the shape.
