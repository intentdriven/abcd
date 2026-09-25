---
schema_version: 1
id: "iss-2609251751298408"
slug: "commands-launch-md-describes-the-release-job-s-semantic"
severity: "minor"
category: "documentation"
source: "agent-finding"
found_during: "autonomous run A resumed 2026-09-25"
origin: researcher-authored
production_mode: hand-written
found_at: "commands/launch.md"
---

commands/launch.md describes the release job's semantic receipt gate at a position it no longer holds. Four places — the Ship intro ('the tag is already created by then'), the release-day failure list ('The tag exists by then'), Semantic receipts ('release.yml derives the content commit as <merge>^2^') and 'Prove the gate before you merge' ('receipt_gate runs inside the release job, which is after the tag is created') — predate adr-52 and iss-355: release.yml runs the gate in its verify job, which the tag job needs, so on the auto-release path a refusal leaves no tag and the version free, and the content commit is derived from the receipts directory of the released tree (record-lint --derive-content-sha), not from merge ancestry. Only a hand-pushed tag exists before the gate. An operator reading the page believes a refusal consumes the version and reaches for a tag deletion the machinery no longer needs.
