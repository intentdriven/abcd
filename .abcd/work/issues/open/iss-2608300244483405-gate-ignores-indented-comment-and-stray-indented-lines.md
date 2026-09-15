---
schema_version: 1
id: "iss-2608300244483405"
slug: "gate-ignores-indented-comment-and-stray-indented-lines"
severity: "minor"
category: "bug"
source: "impl-review"
found_during: "itd-182 final security review, 2026-08-30"
found_at: "internal/core/lint/schema.go, internal/core/capture/parse.go"
---

Two pre-existing gate-versus-reader divergences reproduced on main with found_at, affecting every key and not covered by the single-quote or escape captures: a key whose only continuation is an indented comment line is refused by the reader (nested line is not key: value) while the gate is green; and an indented line following a valued key (or a null one) is refused by the reader (unexpected indented line) while the record-lint gate has no check for stray indented lines anywhere.
