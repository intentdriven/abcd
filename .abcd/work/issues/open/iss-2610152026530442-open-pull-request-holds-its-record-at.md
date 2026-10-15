---
schema_version: 1
id: "iss-2610152026530442"
slug: "open-pull-request-holds-its-record-at"
severity: "minor"
category: "future-work-seed"
source: "user-observation"
found_during: "product-thinker ruling, 2026-10-15"
origin: researcher-authored
production_mode: hand-written
found_at: "the pick's reservation check (abcd build next, abcd build <id>, abcd drain)"
remedy: "If the one-day hold still leads to duplicate work, also treat an open pull request for a record as holding it at pick time; the cost is a forge call at pick time, which breaks today's offline pick (invariant 7)."
---

The product thinker, ruling on 2026-10-15 at about 20:15 BST that a pull request waiting on a person holds its reservation for a day (iss-2610152020234609), added: "capture to also record open PRs if the 1 day default still results in conflicts". Today the pick judges a record held only by its reservation on the shared remote, as last fetched; an open pull request for the record does not hold it. A pull request still waiting after its day, with nothing renewing it, leaves the record open to a colleague's drain. Reading open pull requests at pick time would close that gap, at the cost of a forge call when the pick runs, which breaks the offline pick invariant 7 keeps today. This stays open until the one-day hold is seen to fall short.
