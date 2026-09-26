---
schema_version: 1
id: "iss-2608300941548519"
slug: "itd-178-second-round-observations"
severity: "nitpick"
category: "inconsistency"
source: "impl-review"
found_during: "itd-178 second-round security review, 2026-08-30"
found_at: "internal/core/capture/workflow.go (restampField)"
resolution: "The restamp gate calls provenance.ParseOrigin and refuses a restamp beside an out-of-vocabulary origin; the lone-origin repair is kept as a deliberate write, pinned by a test and stated in spc-56 and commands/capture.md."
impact: fix
resolved_by:
  commit: "48e9b4f4"
---

itd-178 second-round observations: the restamp gate tests origin presence, not validity, so a record with an out-of-vocabulary origin accepts a restamp (the origin stays invalid and the lint still reports it — harmless; calling the origin parser would make the intent literal); a record carrying a lone origin, itself a blocker, is silently repaired into a clean pair by a restamp — a legal write inside the lint's declared residual, worth a sentence in spc-56.

## Grounds

- pursued: no restamp writes a disclosure pair outside the vocabulary; a resolve --production-mode succeeding over an invented origin would show it wrong
