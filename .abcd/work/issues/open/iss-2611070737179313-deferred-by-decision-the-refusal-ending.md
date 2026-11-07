---
schema_version: 1
id: "iss-2611070737179313"
slug: "deferred-by-decision-the-refusal-ending"
severity: "minor"
category: "bug"
source: "user-observation"
found_during: "follow-up list from a release cut"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/surface/cli/drain_drive.go"
remedy: "Make the refusal name a reservation that stands, or reword it so it never claims nothing was reserved when a lane still is."
---

Deferred by decision: the refusal ending '(nothing started, nothing reserved)' can be false on a resuming start. A lane held past the ceiling keeps its reservation when the release is taken, and a kept copy or retried start can then refuse with that line, so a person would not look for the reservation.
