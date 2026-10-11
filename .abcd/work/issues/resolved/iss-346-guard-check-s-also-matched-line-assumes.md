---
schema_version: 1
id: "iss-346"
slug: "guard-check-s-also-matched-line-assumes"
severity: "minor"
category: "bug"
source: "agent-finding"
found_during: "bughunt-round-2"
found_at: "internal/surface/cli/guard.go"
resolution: "also-matched line selects non-winners by id instead of Matches[1:]"
impact: fix
---

guard check's also-matched line assumes Matches[0] is the winner, so a synthetic block over registry warns silently drops the matched warn entry and echoes the winner twice
## Evidence

- `internal/surface/cli/guard.go:335` renders `dec.Matches[1:]` as "also matched", but `Matches` is ordered blockers, warns, synthetics (`internal/core/guard/guard.go:441`, contract documented at `:155-158`) — winner-first holds except when a synthetic block wins over registry warns.
- Reproduced: `guard check --command 'git clean -fd && env -S "a$b"'` blocks on `execute-string-uninspectable`, prints "also matched: execute-string-uninspectable" (the winner, twice) and omits `git-clean`; the JSON plane shows `matches: ["git-clean","execute-string-uninspectable"]`.
- Refuter verdict: CONFIRMED substantive (low end — report integrity in a safety tool; exit code and JSON unaffected). Fix is display-only: filter by `id != dec.EntryID`; do not reorder core `Matches` (the corpus warn-rate gate reads it).
