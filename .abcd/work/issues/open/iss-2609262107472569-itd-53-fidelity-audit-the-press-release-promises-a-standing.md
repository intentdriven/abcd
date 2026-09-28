---
schema_version: 1
id: "iss-2609262107472569"
slug: "itd-53-fidelity-audit-the-press-release-promises-a-standing"
severity: "minor"
category: "drift"
source: "review-followup"
found_during: "autonomous run A resumed 2026-09-25: fidelity audit itd-53"
origin: researcher-authored
production_mode: hand-written
---

itd-53 fidelity audit: the press release promises a standing list of shipped intents whose latest fidelity review is missing OR not-met, and decision 2 names the owed listing as that report; the delivered listing (bare intent audit / itd-2609150819445595) reads the review marker state only, so an intent whose ingested verdict was NOT_MET reads INGESTED and leaves the list, and the unmet half survives only as the page-instructed capture per verdict (receipt rcp-d2372b1cb47f)
