---
schema_version: 1
id: "iss-2608210934566225"
slug: "ahoy-offered-statusline-verb"
severity: "minor"
category: "future-work-seed"
source: "user-observation"
found_during: "plugin-update post-mortem 2026-08-21"
resolution: "The verb this seed asked for exists: ahoy install offers the status line as its own consent category and points the harness at 'abcd statusline', which consumes the harness stdin JSON (itd-200, spc-70). The extra elements the seed listed (guard health, version skew, update-available) are each their own later decision, recorded as out of scope on itd-200."
impact: additive
resolved_by:
  intent: "itd-200"
  spec: "spc-70"
---

The harness statusline is a user-level setting no plugin can inject, but ahoy install could offer it as an owned transparent-confirm ConfigChange (same pattern as the PATH symlink): point statusLine at a new 'abcd statusline' verb consuming the harness stdin JSON and appending guard health, version skew, and update-available. Event-driven refresh (300ms debounce, min 1s interval) rules out live download progress; steady-state health only.

## Grounds

- pursued: the offered verb is the wiring every later status element hangs on; if the line is declined more often than taken at install, the offer's placement in onboarding was wrong
