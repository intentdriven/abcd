---
schema_version: 1
id: "iss-2610262033340952"
slug: "the-intent-chapter-s-issue-drift-form"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 420 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/05-intent.md:329"
remedy: "Narrow the 'write no report at all' list at 04-universal-patterns.md line 90: intent audit with the issue-drift flag writes a report under the log tier (internal/core/capture/drift.go), and the audit forms stage requests under the reviews tier."
finding_ref: cfk-b6677e5186d5f034
---

The intent chapter's issue-drift form writes a report.json in the log tier, while universal patterns says the intent verb writes no report at all.

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/04-surfaces/05-intent.md:329` — "Findings land in `.abcd/.work.local/logs/audit/issue-drift-<ts>/report.json`."
- End B: `.abcd/development/brief/05-internals/04-universal-patterns.md:90` — "Most verbs write no report at all: `lint` and its targets, `capture`, `intent`, `guard`"

05-intent states that the issue-drift check's findings land in a report.json under the log tier's audit directory. 04-universal-patterns, describing what ships today, lists intent among the verbs that write no report at all. One of the two statements about what the shipped intent verb writes is false.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 420.
