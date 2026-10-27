---
schema_version: 1
id: "iss-2610252211498831"
slug: "the-reserve-sweep-help-says-a-hand-back"
severity: "minor"
category: "documentation"
source: "agent-finding"
found_during: "docs-currency review of docs/, 2026-10-24, reader D7"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/surface/cli/reserve.go"
remedy: "Reword the help so the unclassifiable-commit and unread-history conditions also apply only to a record still open, and say that on a terminal record only an unknown or unreadable tip keeps the branch. Regenerate docs/reference/cli/commands.md. If the wider rule is the intent instead, change judge and its test."
---

The reserve --sweep help says 'A hand-back on a record still open, a commit of a kind the sweep cannot classify, a history it cannot read to its root, and an unreadable tip keep the branch' (internal/surface/cli/reserve.go:174-176, reflected at docs/reference/cli/commands.md:4532-4534). Only the hand-back carries the 'record still open' limit. The judge marks a branch whose record is terminal on origin's default branch as spent once its tip is classifiable, before it walks the chain (internal/core/reservesweep/judge.go:94-97), and TestATerminalRecordIsSpentWhateverItsTipOnceOlderThanTheAge (judge_test.go:96) pins that. So on a resolved, won't-fixed, shipped or superseded record, an unclassifiable non-tip commit or a history unread to its root does not keep the branch, and --apply deletes it. The help states the fail-closed rule more widely than the code applies it.
