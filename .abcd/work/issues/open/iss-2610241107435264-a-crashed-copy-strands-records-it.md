---
schema_version: 1
id: "iss-2610241107435264"
slug: "a-crashed-copy-strands-records-it"
severity: "minor"
category: "bug"
source: "agent-observation"
found_during: "drain session abcd-0d, 2026-10-24, reading a crashed copy's reservations"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/claim"
remedy: "abcd peers and the drain name an expired reservation whose footprint names a record file absent from origin's default branch and from every branch on origin as stranded work, naming the copy and the files; and an autonomous run pushes each record it creates to its own branch as it creates it, so a crash loses no record."
---

A crashed copy strands records it reserved but never pushed, and nothing says so. After the 2026-10-24 crash, origin holds expired reservations whose footprints name records that exist nowhere on origin: itd-2610222214476887 (the draft intent for a sweep verb) and four decision records (dcn-2610222216264421 and its three siblings), under reservation iss-2610222001116066. The work was reserved, then lost with the computer, and no other copy can see that it is gone: abcd peers lists the reservation as an ordinary expired one, and only a person comparing the footprint with origin's tree finds the gap.
