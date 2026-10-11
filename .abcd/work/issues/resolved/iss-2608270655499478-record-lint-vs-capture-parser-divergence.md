---
schema_version: 1
id: "iss-2608270655499478"
slug: "record-lint-vs-capture-parser-divergence"
severity: "minor"
category: "tech-debt"
source: "agent-finding"
found_during: "security-cut-agent-flagged-siblings-2026-08-27"
found_at: "internal/core/frontmatter"
resolution: "Overtaken by 35600e968: record_schema's issue-store reader-parity leg asks capture's strict ledger reader itself (capture.ReadRefusal, registered by the front doors), so a block-sequence field in an issue record is refused by the gate exactly as the reader refuses it, while the intent and ADR stores, whose readers take block sequences, are untouched: the store-scoped fix the record asks for. Pinned by TestRecordSchemaRefusesAnIssueBlockSequenceAsTheReaderDoes in 8f0cf5f42."
impact: internal
resolved_by:
  commit: "35600e968"
---

record-lint vs capture parser divergence remainder: block-sequence frontmatter fields are legitimate and used in 21+ intent/adr records but only capture's strict ledger parser rejects them, so a correct fix is store-scoped (share capture's typed strict parser through the canonical frontmatter package) rather than a universal rejection. The duplicate-key and space-before-colon halves of #357 are fixed; this block-sequence remainder is the follow-up. Flagged by the lint-integrity fix agent.

## Grounds

- pursued: record-lint refuses an issue record with a block-sequence field because capture's reader does, and no other store; an issue record carrying a block sequence that lints green, or an intent or ADR block sequence the gate starts refusing, would show it wrong
