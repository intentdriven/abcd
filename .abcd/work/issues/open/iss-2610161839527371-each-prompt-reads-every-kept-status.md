---
schema_version: 1
id: "iss-2610161839527371"
slug: "each-prompt-reads-every-kept-status"
severity: "minor"
category: "tech-debt"
source: "agent-finding"
found_during: "messaging step 3 fix round 3 (feat/announce-step3)"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/claim"
remedy: "Keep a small index of unshown announcements beside the kept ones, written when an announcement is taken in, so a prompt reads only the index and the unshown ones, and keep the full read for the status command."
---

Each prompt reads every kept status announcement to decide what to show, at about 0.17 ms per announcement, so at the 2,000-announcement keep cap the prompt hook spends about 340 ms of its one-second budget on this read alone (measured in the messaging step 3 fix round, 2026-10-16).

Above 2,000 kept files the read is also uncapped before the cap applies: `keptAnnouncementsCounted` (`internal/core/claim/status.go`) takes the details of every kept file and sorts them all by when each was written, before it keeps the newest 2,000, so that step grows with every file kept past the cap (found in the messaging step 3 fix round 4 review, 2026-10-16).
