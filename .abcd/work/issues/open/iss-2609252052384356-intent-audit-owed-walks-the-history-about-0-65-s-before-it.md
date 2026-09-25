---
schema_version: 1
id: "iss-2609252052384356"
slug: "intent-audit-owed-walks-the-history-about-0-65-s-before-it"
severity: "nitpick"
category: "ux"
source: "review-followup"
found_during: "autonomous run A resumed 2026-09-25"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/surface/cli/intent_drain.go"
---

intent audit --owed walks the history (about 0.65 s) before it refuses a negative --max, and its flag help does not say it writes (it parks an OWED stub on a markerless head, a diff a peer sees)
