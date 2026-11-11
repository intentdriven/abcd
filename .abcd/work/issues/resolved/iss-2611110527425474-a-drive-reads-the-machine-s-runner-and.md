---
schema_version: 1
id: "iss-2611110527425474"
slug: "a-drive-reads-the-machine-s-runner-and"
severity: "minor"
category: "bug"
source: "user-observation"
found_during: "autonomous run, live drive"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/surface/cli/drain_drive.go"
remedy: "Say it where the person looks: one line in the drive's start text and one passage in the drain command page and the drain brief page, naming what is read once and that the class map and the review set are read again per run. Re-reading provider settings mid-drive is not proposed: it would move a trust decision the start's refusal now makes before anything is reserved."
resolution: "The drive's start text now says which settings are read once and that the class map and the review set are read again as each run starts; the drain command page and the drain brief page say the same."
impact: fix
---

A drive reads the machine's runner and provider settings once, at its start, and nothing says so. A provider block or a denylist entry changed in the machine's configuration while a drive runs takes effect only at the next start; the class map and the review set are the parts read again as each run starts. A person who changes the file to unblock a running drive sees no effect and no reason: on 10 Nov 2026 a live drive kept refusing its reviewers after the configuration had been corrected, until it was started again. Neither the start's text nor the drain page states which settings are read once.
