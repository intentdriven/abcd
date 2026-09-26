---
schema_version: 1
id: "iss-2609260543090196"
slug: "abcd-guard-warns-instead-of-blocking-a-dash-glued"
severity: "minor"
category: "bug"
source: "drift-detection"
found_during: "v0.11.0 release gate (autonomous run A, abcd-a2)"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/guard"
---

abcd guard warns instead of blocking a dash-glued substitution word after timeout when no duration follows: `timeout --$(x) pkill -f node` and `timeout -$(x) pkill -f node` come back as unrecognised-launcher (warn), while the same word behind sudo, env, xargs, nice, exec, doas, stdbuf, su or git blocks, and `timeout 5 pkill -f node`, `timeout --$(x) 5 pkill -f node` and `timeout 5 --$(x) pkill -f node` all block. Found by the v0.11.0 docs-currency gate (dc-4) and confirmed by the changelog composer against bin/abcd-darwin-arm64 built from ae116575.
