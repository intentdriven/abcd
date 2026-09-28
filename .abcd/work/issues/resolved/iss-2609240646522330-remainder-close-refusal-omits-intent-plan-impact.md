---
schema_version: 1
id: "iss-2609240646522330"
slug: "remainder-close-refusal-omits-intent-plan-impact"
severity: "nitpick"
category: "ux"
source: "agent-observation"
found_during: "autonomous run 2026-09-23"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/intent/lifecycle.go"
resolution: "Both early-close impact refusals (another spec still open; a remainder minted) name abcd intent plan <itd-N> --impact <value> as the way to record the judgement now, after which the close that ships needs no flag."
impact: fix
resolved_by:
  commit: "fe9c705c2d9f239b5d8b5afee46877072e949fd0"
---

`abcd spec close <spc-N> --remainder <slug> --impact <value>` is refused by design, because a remainder close ships nothing, and the refusal says to supply the impact at the close that ships. It does not say that the judgement can be kept now: `abcd intent plan <itd-N> --impact <value>` stamps the impact on an intent that is already planned, and the later close then needs no flag. In autonomous run A a lane that knew the impact at a remainder close dropped it, which left it to be remembered by whichever session makes the final close. Wanted: both remainder refusals in internal/core/intent/lifecycle.go name `abcd intent plan <itd-N> --impact <value>` as the way to record the judgement at once.

## Grounds

- pursued: we expect naming the plan route in the refusal to keep an impact a lane already knows from being dropped at a remainder close; it is shown wrong if a lane refused there still re-runs without recording the impact and the judgement is lost to the final close
