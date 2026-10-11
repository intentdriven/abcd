---
schema_version: 1
id: "iss-2610090701491123"
slug: "the-product-thinker-ruled-on-2026-10-09"
severity: "minor"
category: "ux"
source: "review-followup"
found_during: "2026-10-09 product thinker interview on the run's hand-backs"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/oracle/dispatch.go"
remedy: "Report a connect failure after a successful size count as unreachable-with-no-answer so Route.FellBack moves the step to the harness, and say in the fallback line that the brief had already reached the server for counting; test the count-succeeds-then-connect-fails case falls back, watched fail first."
---

The product thinker ruled on 2026-10-09 that once abcd has sent a brief to the person's own model server for the size count, a later failure before any answer still moves the step to the agent harness: 'no answer received' is the trigger, not 'nothing was sent'. Today the adapter stops with an error instead, because the fix for iss-2610030956156354 stopped reporting such a failure as unreachable, and only an unreachable error falls back (internal/core/oracle/dispatch.go FellBack).
