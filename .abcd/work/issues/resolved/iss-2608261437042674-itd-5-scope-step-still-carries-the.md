---
schema_version: 1
id: "iss-2608261437042674"
slug: "itd-5-scope-step-still-carries-the"
severity: "nitpick"
category: "observation"
source: "agent-observation"
found_during: "bughunt-b-round-9"
found_at: ".abcd/development/intents/disciplines/itd-5-prompt-quality-additions.md"
resolution: "Step 3 of itd-5's Add 2 still read 'shorter by >10%' after the itd-81 amendment struck that tiebreak; steps 2 and 3 now run the calibration corpus against both variants, accept the better score, keep the candidate on a tie, and say length is no tiebreak. No gate reads intent prose against its own amendments, so none caught it."
impact: internal
resolved_by:
  commit: "4dda0ae79"
---

itd-5 scope step still carries the tiebreak its own amendment struck

## Grounds

- pursued: the scope steps now agree with the amendment and with the Why section; shown wrong if any line of itd-5 still makes length decide the pre-flight
