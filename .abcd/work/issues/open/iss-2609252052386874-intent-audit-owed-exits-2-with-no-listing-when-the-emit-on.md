---
schema_version: 1
id: "iss-2609252052386874"
slug: "intent-audit-owed-exits-2-with-no-listing-when-the-emit-on"
severity: "minor"
category: "bug"
source: "review-followup"
found_during: "autonomous run A resumed 2026-09-25"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/intent/drain.go"
---

intent audit --owed exits 2 with no listing when the emit on the queue head fails (a malformed spec_id, an unreadable file), and --max cannot skip it, so one bad oldest record blocks every drain run until it is fixed by hand
