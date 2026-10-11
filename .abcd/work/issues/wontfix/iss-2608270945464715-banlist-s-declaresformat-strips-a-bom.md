---
schema_version: 1
id: "iss-2608270945464715"
slug: "banlist-s-declaresformat-strips-a-bom"
severity: "nitpick"
category: "inconsistency"
source: "agent-finding"
found_during: "issue-sweep-review-2026-08-27"
found_at: "internal/core/banlist/private.go"
wontfix_reason: "premise incorrect: the authoritative shell pre-commit hook (normalise_decl) strips a leading BOM on EVERY line, not only line 0, so declaresFormat stripping on any line is correct Go/shell parity. Restricting the strip to line 0 (the proposed fix) breaks parity in the direction the file forbids (one file, two verdicts) — the Go status reader would call a keyed store healthy that the hook blocks. Current behaviour stands; found and reverted during the adversarial review of the rules-and-correctness batch."
---

banlist's declaresFormat strips a BOM from any line with no index guard while every other reader trims it at line 0 only — its misplaced-declaration caller fails closed, so an inconsistency in the BOM-position family rather than a defect