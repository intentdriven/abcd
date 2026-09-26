---
schema_version: 1
id: "iss-2608300244483405"
slug: "gate-ignores-indented-comment-and-stray-indented-lines"
severity: "minor"
category: "bug"
source: "impl-review"
found_during: "itd-182 final security review, 2026-08-30"
found_at: "internal/core/lint/schema.go, internal/core/capture/parse.go"
resolution: "record_schema's issue-store legs decode values as capture's reader does (schemaRecord.scalar), and a reader-parity leg calls capture.ReadRefusal on any issue record the other legs pass, registered by cmd/record-lint and the CLI. TestRecordSchemaAgreesWithTheLedgerReader asserts each case against the reader itself; TestRecordLintRegistersTheLedgerReader and TestCLIRegistersTheLedgerReader pin the wiring."
impact: fix
resolved_by:
  commit: "35600e96"
---

Two pre-existing gate-versus-reader divergences reproduced on main with found_at, affecting every key and not covered by the single-quote or escape captures: a key whose only continuation is an indented comment line is refused by the reader (nested line is not key: value) while the gate is green; and an indented line following a valued key (or a null one) is refused by the reader (unexpected indented line) while the record-lint gate has no check for stray indented lines anywhere.

## Grounds

- pursued: record_schema and capture's ledger reader give one verdict on every committed issue record; a record capture list skips that record-lint passes, or the reverse, would show it wrong
