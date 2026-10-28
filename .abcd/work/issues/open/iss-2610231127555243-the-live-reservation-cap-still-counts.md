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
remedy: "Waits on ruling (one ceiling per person, or a count per account and machine): if per account and machine, count against this machine's reservation.max_live only the live holds whose copy is this machine's (this copy and its sibling worktrees, as claim.CeilingFor already tells them apart), keep the login-wide total shown beside it, and test that a hold from another clone under the same login no longer refuses this machine's next reservation"
refines: [iss-2610231004140527]
---

The live-reservation cap still counts one total per GitHub login across every account that pushes as it, while each machine reads its own reservation.max_live, so a drain on another account under the same login stays held at its own ceiling whenever this machine's sessions hold more. iss-2610231004140527 made that visible: abcd peers counts the reader's holds by copy and warns from four fifths of this machine's ceiling, and the cap refusal groups the held records by copy. Whether the count itself should change is a ruling the coordinator is taking to the product thinker: keep one ceiling per person (the ruling recorded in iss-2610170717515674) or count per account and machine.
