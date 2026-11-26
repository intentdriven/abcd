---
schema_version: 1
id: "iss-2611251828563210"
slug: "shipped-code-on-proposed-decisions"
severity: "minor"
category: "drift"
source: "agent-finding"
found_during: "manual build of 25 November, logged as a test of what abcd relies on"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/decisions/adrs/README.md"
remedy: "Have a person rule on each, and add a check that a shipped intent rests on no proposed decision. Grounds: the index tells a reader which decisions hold, and a check is cheaper than a reader noticing."
---

Three decisions whose code ships are still marked proposed: adr-2611062149324491 (a drive replaces its own binary), adr-2610181003541997 (the away mode reads the forge at a stop), and adr-2610150729333337 (a status branch moves without a yes). A proposed decision is not in force, and the product does what each says. Checked on 2026-11-25 by the status line of each record.
