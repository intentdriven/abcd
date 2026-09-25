---
schema_version: 1
id: "iss-2609250937494009"
slug: "the-sub-verb-lint-internal-core-lint-subverbs-go"
severity: "minor"
category: "bug"
source: "user-observation"
found_during: "autonomous run A resumed 2026-09-25"
origin: researcher-authored
production_mode: hand-written
---

The sub-verb lint (internal/core/lint/subverbs.go parseSubVerbTable) treats finding the ## Sub-verbs heading as finding the table: a surface file with the heading followed by prose and no header row passes with no finding, while the brief surfaces README promises that a file without the table is a finding whatever its verb registers. Fix: require the table's header row under the heading; test a heading-only file.
