---
schema_version: 1
id: "iss-2610150256285152"
slug: "v11-answered-refusal-points-at-deletion"
severity: "nitpick"
category: "bug"
source: "review-followup"
found_during: "PR 976 schema-12 review"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/implement/loop/state.go"
remedy: "Read a version-11 file that carries answered but no reserved_at as the branch's own version 11 and drop answered (crediting the review to the host), or give that refusal the do-not-remove-the-run wording."
---

A run state written by unreleased builds of the reviewer-route branch, which stamped schema 11 and stored answered on an await, is refused after the move to schema 12 with the remedy to restore or remove the run directory, which would throw away an intact run; its stated reason (version 11 never wrote answered) is also untrue for that file. Only branch builds wrote it, in the window between the route placing its answer and the hand-back.
