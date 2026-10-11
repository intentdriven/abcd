---
schema_version: 1
id: "iss-2608270908342889"
slug: "capture-s-validatestrict-refuses-enum"
severity: "major"
category: "bug"
source: "agent-finding"
found_during: "issue-sweep-2026-08-27"
found_at: "internal/core/capture/validate.go"
resolution: "checkIssueRecordShape mirrors capture's enum-membership, kebab-slug and unknown-key checks from the shared issueschema home; folder-invariant/type mirroring noted as follow-up"
impact: fix
resolved_by:
  commit: "b55c5c38"
---

capture's validateStrict refuses enum membership, kebab slugs, unknown fields and folder-to-field invariants that no lint rule mirrors, so a record capture skips is lint-green and invisible to every capture verb — fail-soft, but a lost-record class with no armed detector