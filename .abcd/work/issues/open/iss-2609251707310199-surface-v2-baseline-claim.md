---
schema_version: 1
id: "iss-2609251707310199"
slug: "surface-v2-baseline-claim"
severity: "nitpick"
category: "documentation"
source: "impl-review"
found_during: "autonomous run A resumed 2026-09-25"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/surface/sentence_test.go"
---

The sentence lane's claim that the release guardrail reads a version-2 surface baseline from the last tag is wrong: v0.10.0 carries a schema-1 surface.json, and versions 1 to 3 are readable; the claim sits in the doc comment of TestDecodeReadsTheVersionTwoShape
