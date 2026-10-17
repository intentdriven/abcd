---
schema_version: 1
id: "iss-2610152204590878"
slug: "the-outbound-runs-the-check-reading-of-a"
severity: "nitpick"
category: "bug"
source: "user-observation"
found_during: "outbound one-switch final review, 2026-10-15"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/ahoy/outbound_hook.go"
remedy: "Accept braced parameter expansions as the command word in outboundHookRunsCheckRe, with a test"
---

The outbound 'runs the check' reading of a hook's text does not recognise an unquoted ${bin} lint outbound, so a hook that runs the check still draws the not-wired note and gets the copy of the check saved beside it.
