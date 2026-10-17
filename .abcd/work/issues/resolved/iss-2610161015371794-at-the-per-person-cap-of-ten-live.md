---
schema_version: 1
id: "iss-2610161015371794"
slug: "at-the-per-person-cap-of-ten-live"
severity: "major"
category: "bug"
source: "review-followup"
found_during: "multi-account drain investigation, 2026-10-16"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/claim/crowd.go"
remedy: "At the cap the drain waits (next_eligible_at) and keeps its judgements, and the advice says to wait for a pull request to merge (grounds: the drain already treats its pace window as a wait)"
related_intents: [itd-2610161016394967]
resolution: "At the per-person cap a drain move no longer refuses: it counts the holder's live reservations in the move's view, reserves and asks nothing, writes waiting_until fifteen minutes on apart from the pace window, keeps its judgements, and names the pull requests waiting to merge; the cap's refusal is typed and advises waiting for a merge rather than releasing a reservation"
impact: fix
resolved_by:
  commit: "7cbf37c26"
---

At the per-person cap of ten live reservations a drain move refuses with exit 2 (the cap is not treated as contention), returns before writing its state so the judgement just answered is lost, and advises releasing a reservation, which would free an issue whose pull request is only waiting for a merge.
