---
schema_version: 1
id: "iss-2610231127555243"
slug: "the-live-reservation-cap-still-counts"
severity: "minor"
category: "ux"
source: "agent-finding"
found_during: "the fix for iss-2610231004140527, 2026-10-23"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/claim/crowd.go"
remedy: "Ruled 2026-11-25 (dcn-2611251747513582): replace the single login-wide gate with two limits. A per-drain limit on lanes in work, set per repository, overridable per run, default 1, counting only the lane being implemented (a lane whose pull request is queued keeps its hold but leaves the count). A per-person ceiling in the repository's configuration, default 10, counting every live hold under the login across machines and accounts, queued pull requests included; a machine's reservation.max_live may lower it and never raise it. Test that a second drain under the same login opens its next lane while the first's queued pull request still holds its issue, and that a machine setting above the repository's ceiling is held to the repository's value. Built as a later step of the intent for drives that keep working across machines."
refines: [iss-2610231004140527]
related_intents: [itd-2611251809580662]
resolution: "A drain now holds its lanes in work to drain.in_work_max, the repository's figure at origin's default branch or --in-work, one by default; a lane whose pull request is queued keeps its hold and leaves the count, and the per-person ceiling is the repository's reservation.max_live, which a machine may only lower (step 1)."
impact: additive
resolved_by:
  intent: "itd-2611251809580662"
  spec: "spc-2611251810508418"
---

The live-reservation cap still counts one total per GitHub login across every account that pushes as it, while each machine reads its own reservation.max_live, so a drain on another account under the same login stays held at its own ceiling whenever this machine's sessions hold more. iss-2610231004140527 made that visible: abcd peers counts the reader's holds by copy and warns from four fifths of this machine's ceiling, and the cap refusal groups the held records by copy. Whether the count itself should change is a ruling the coordinator is taking to the product thinker: keep one ceiling per person (the ruling recorded in iss-2610170717515674) or count per account and machine.
