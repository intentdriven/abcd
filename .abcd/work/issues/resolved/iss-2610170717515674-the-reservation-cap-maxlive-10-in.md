---
schema_version: 1
id: "iss-2610170717515674"
slug: "the-reservation-cap-maxlive-10-in"
severity: "minor"
category: "ux"
source: "agent-finding"
found_during: "abcd-a2 overnight run, 2026-10-17"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/claim/crowd.go"
remedy: "Make the cap a machine-layer setting (reservation.max_live in the person's config, default 10), keeping one ceiling per person, so a person who runs many sessions can raise it; the drain already waits at the cap (spec step 6)."
resolution: "reservation.max_live in the person's own ~/.abcd.noindex/config.json now sets the per-person cap, 1 to 100 and ten when unset, read from the machine layer alone; the cap refusal names the setting"
impact: additive
resolved_by:
  commit: "428b9d98f"
---

The reservation cap (MaxLive, 10, in internal/core/claim/crowd.go) counts every live reservation one person holds across all of their sessions and copies, so one person running seven sessions on one account hit it all night: a session's fix waited for slots, two drain issues went unreserved, and sessions released and re-took holds to make room.
