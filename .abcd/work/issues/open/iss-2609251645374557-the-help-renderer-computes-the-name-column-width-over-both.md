---
schema_version: 1
id: "iss-2609251645374557"
slug: "the-help-renderer-computes-the-name-column-width-over-both"
severity: "minor"
category: "ux"
source: "user-observation"
found_during: "autonomous run A resumed 2026-09-25"
origin: researcher-authored
production_mode: hand-written
---

The help renderer computes the name column width over both blocks, so the person's block renders at width 12 under --help and 21 under --help --agent: the people block is not byte-stable across the two forms (internal/surface/cli/helpgroups.go:266-274; review-helpgroups 2).
