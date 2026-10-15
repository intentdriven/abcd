---
schema_version: 1
id: "iss-2610152020234609"
slug: "fence-holds-a-day-without-a-mirror"
severity: "minor"
category: "process"
source: "user-observation"
found_during: "product-thinker ruling, 2026-10-15"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/implement/loop/reservation.go"
remedy: "The land fence renews for lease.Max whenever the land step leaves a pull request waiting on a person, mirror or not, and decision 29 and C44 of itd-2609091034175565 are amended to match."
---

The product thinker ruled on 2026-10-15, at about 20:15 BST, answering "Hold the sign a day": a pull request the land step leaves waiting on a person holds its reservation on the shared remote for a day, mirror or not. Today the land fence (fenceLease in internal/core/implement/loop/reservation.go) renews for lease.Max only where the checkout has a ruleset mirror (.abcd/work/rulesets/), and for the two-hour lease.Default otherwise, per decision 29 and criterion C44 of itd-2609091034175565 (spc-2610031753359849). Without a mirror, a pull request that is only waiting to merge loses its reservation two hours after the fence unless something renews it, and a colleague's drain can then take the record and redo the work.
