---
schema_version: 1
id: "iss-323"
slug: "hook-salvage-entries-invoke-bootstrap-sh-with-no-timeout-bud"
severity: "minor"
category: "bug"
source: "agent-finding"
found_during: "bughunt-round-1"
found_at: "hooks/hooks.json"
deferred_after: "v0.11.1"
deferral_reason: "ruling owed to the product thinker (renewed by run A 2026-09-29 after the v0.10.0 grant lapsed at the v0.11.0 anchor): Give the three salvage hooks that still run bootstrap.sh (UserPromptSubmit, PreToolUse, PreCompact; SessionEnd no longer does) an explicit 90-120s timeout, at the cost of a first prompt stalling that long on a slow link, or keep the harness's 60s default?"
---

hook salvage entries invoke bootstrap.sh with no timeout budget below its 180s curl worst case so a slow-link provision is killed and suppressed for ten minutes
## Evidence
`hooks/hooks.json` — SessionStart (`:18`) declares `"timeout":240`; the four salvage entries (UserPromptSubmit `:3`, PreToolUse `:24`, PreCompact `:35`, SessionEnd `:45`) invoke the same bootstrap.sh with NO timeout, inheriting the harness 60s default (DECISIONS.md:898 records the 60s default and the 180s worst case as the reason SessionStart got 240). On a slow link the salvage hook is killed at 60s after `.bootstrap.attempt` is stamped, suppressing retry for 10 min, output to /dev/null.

## Adversarial verdict: CONFIRMED (minor) — RECORD-ONLY
Default-timeout premise verified (60s, DECISIONS.md:898 + the hooks reference). State is kill-safe (trap cleanup; verified-then-mv install; no corrupt binary). Not fixed this round because the correct value is a design judgment: setting 240 on UserPromptSubmit would quadruple the user's synchronous prompt stall to 4 minutes. Recommended direction (for a human): a modest explicit timeout (90-120s) on the salvage entries, and — separately valuable — a test asserting the per-event timeout values (none exists today; even SessionStart's 240 is unpinned). No test currently pins any timeout field.
