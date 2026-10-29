---
schema_version: 1
id: "iss-2610241550026188"
slug: "record-schema-resolves-a-cross-reference"
severity: "minor"
category: "bug"
source: "agent-finding"
found_during: "hand-back triage of 2026-10-24, batch b06 reading iss-2609221820487644"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/lint/schema.go"
remedy: "Make record_schema in internal/core/lint/schema.go refuse a related_adrs entry that is not an adr-N and a related_intents entry that is not an itd-N, move iss-2609221820487644 from itd-2609221842494980's related_adrs into the promote join, and prove it with a schema test that a related_adrs naming an iss-N is a blocker finding."
---

record_schema resolves a cross-reference field's target in any record family (internal/core/lint/schema.go:764 checks only that the handle is in the corpus), so itd-2609221842494980 carries related_adrs: [iss-2609221820487644], an issue, and passes record-lint, although the ADR README documents related_adrs as adr-N only; because the link sits in the wrong field, the promote join between that shipped intent and its still-open originating issue is invisible to intent audit --issue-drift.
