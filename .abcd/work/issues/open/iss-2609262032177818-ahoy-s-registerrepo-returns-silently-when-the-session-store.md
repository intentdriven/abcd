---
schema_version: 1
id: "iss-2609262032177818"
slug: "ahoy-s-registerrepo-returns-silently-when-the-session-store"
severity: "minor"
category: "bug"
source: "review-followup"
found_during: "autonomous run A resumed 2026-09-25: review-integ5"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/ahoy/apply.go"
---

ahoy's registerRepo returns silently when the session store's index cannot be read: loadHistoryIndex's error (an unreadable, oversize or malformed ~/.abcd/history/index.json) skips the repository's registration with no note, unlike stepHistory's three store failures, which each leave a note naming the store and the reason
