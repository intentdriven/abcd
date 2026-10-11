---
schema_version: 1
id: "iss-2609262032177818"
slug: "ahoy-s-registerrepo-returns-silently"
severity: "minor"
category: "bug"
source: "review-followup"
found_during: "autonomous run A resumed 2026-09-25: review-integ5"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/ahoy/apply.go"
resolution: "registerRepo leaves a note naming the index and the reason when the session store's index cannot be read, as stepHistory does for its three store failures."
impact: fix
resolved_by:
  commit: "5572760de"
---

ahoy's registerRepo returns silently when the session store's index cannot be read: loadHistoryIndex's error (an unreadable, oversize or malformed ~/.abcd/history/index.json) skips the repository's registration with no note, unlike stepHistory's three store failures, which each leave a note naming the store and the reason

## Grounds

- pursued: an install over a malformed index.json carries a note that the repository was not registered and why; TestUnreadableHistoryIndexIsNoted finding no such note would show it wrong
