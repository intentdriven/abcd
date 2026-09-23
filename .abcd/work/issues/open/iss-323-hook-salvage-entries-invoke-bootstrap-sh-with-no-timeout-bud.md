---
schema_version: 1
id: "iss-323"
slug: "hook-salvage-entries-invoke-bootstrap-sh-with-no-timeout-bud"
severity: "minor"
category: "bug"
source: "agent-finding"
found_during: "bughunt-round-1"
found_at: "hooks/hooks.json"
deferred_after: "v0.9.0"
deferral_reason: "Routed to the product thinker by the 2026-09-23 run (ruling owed on the salvage-hook timeout value (60s default vs 90-120s vs prompt-stall cost), then pin every per-event timeout in a test). The 2026-09-23 interview gave routed minor and nitpick captures the default: deferred past v0.9.0, returning at the next anchor."
---

hook salvage entries invoke bootstrap.sh with no timeout budget below its 180s curl worst case so a slow-link provision is killed and suppressed for ten minutes
## Evidence
`hooks/hooks.json` — SessionStart (`:18`) declares `"timeout":240`; the four salvage entries (UserPromptSubmit `:3`, PreToolUse `:24`, PreCompact `:35`, SessionEnd `:45`) invoke the same bootstrap.sh with NO timeout, inheriting the harness 60s default (DECISIONS.md:898 records the 60s default and the 180s worst case as the reason SessionStart got 240). On a slow link the salvage hook is killed at 60s after `.bootstrap.attempt` is stamped, suppressing retry for 10 min, output to /dev/null.

## Adversarial verdict: CONFIRMED (minor) — RECORD-ONLY
Default-timeout premise verified (60s, DECISIONS.md:898 + the hooks reference). State is kill-safe (trap cleanup; verified-then-mv install; no corrupt binary). Not fixed this round because the correct value is a design judgment: setting 240 on UserPromptSubmit would quadruple the user's synchronous prompt stall to 4 minutes. Recommended direction (for a human): a modest explicit timeout (90-120s) on the salvage entries, and — separately valuable — a test asserting the per-event timeout values (none exists today; even SessionStart's 240 is unpinned). No test currently pins any timeout field.
