---
schema_version: 1
id: "iss-2611201429249531"
slug: "a-reservation-renews-for-as-long-as-its"
severity: "minor"
category: "process"
source: "user-observation"
found_during: "person's interview on 20 Nov 2026 during the autonomous run of 15 Nov"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/implement/loop/reservation.go"
remedy: "Stop renewing a reservation once its run's landing has made no merge progress for a configured time (no new head, not armed, not in the merge queue, not merged, and no forge outage or stuck queue recorded by the run, which adr-2610222222326104 and dcn-2611151650309222 keep renewing through), reservation.stall_hours with a default of six, and let it lapse at its current expiry; status and abcd peers name a reservation that has stopped renewing for that reason, and the run says so when it resumes. A test renews a reservation whose landing has not moved past the bound and finds it left to lapse, one whose landing moved and finds it renewed, and one whose pull request has sat armed in the queue past the bound and finds it renewed."
---

A reservation renews for as long as its run lives, even when its landing has stopped moving: the other account's drain lane for iss-2610230851422913 opened #1622 on 18 Nov, the pull request passed its checks but was never armed or merged, and the reservation renewed past 20 Nov 13:22Z, holding a test file in internal/surface/cli, which overlaps every footprint naming that package, for over two days. Nothing tells the holder or the copies it blocks.
