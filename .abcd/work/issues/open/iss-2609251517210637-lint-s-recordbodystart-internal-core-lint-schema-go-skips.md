---
schema_version: 1
id: "iss-2609251517210637"
slug: "lint-s-recordbodystart-internal-core-lint-schema-go-skips"
severity: "minor"
category: "bug"
source: "impl-review"
found_during: "autonomous run A resumed 2026-09-25"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/lint/schema.go"
---

lint's recordBodyStart (internal/core/lint/schema.go) skips only leading lines that START with <!--, so the second line of a multi-line leading comment is taken for the start of the body and recordTitle reads comment text as the record's title (a record opening with a two-line attribution comment is titled 'attribution line'). It keeps a private reading of HTML comments beside mdrecord's, the tree's one comment reader. One committed file has the shape today (internal/core/ahoy/defaults/claude-md-marker-block.md, not a record).
