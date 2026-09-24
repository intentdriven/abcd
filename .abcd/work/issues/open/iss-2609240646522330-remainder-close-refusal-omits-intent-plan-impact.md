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
---

`abcd spec close <spc-N> --remainder <slug> --impact <value>` is refused by design, because a remainder close ships nothing, and the refusal says to supply the impact at the close that ships. It does not say that the judgement can be kept now: `abcd intent plan <itd-N> --impact <value>` stamps the impact on an intent that is already planned, and the later close then needs no flag. In autonomous run A a lane that knew the impact at a remainder close dropped it, which left it to be remembered by whichever session makes the final close. Wanted: both remainder refusals in internal/core/intent/lifecycle.go name `abcd intent plan <itd-N> --impact <value>` as the way to record the judgement at once.
