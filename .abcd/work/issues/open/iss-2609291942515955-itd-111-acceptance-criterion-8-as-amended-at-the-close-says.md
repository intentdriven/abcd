---
schema_version: 1
id: "iss-2609291942515955"
slug: "itd-111-acceptance-criterion-8-as-amended-at-the-close-says"
severity: "minor"
category: "inconsistency"
source: "review-followup"
found_during: "autonomous run A resumed 2026-09-25: fidelity audit itd-111"
origin: researcher-authored
production_mode: hand-written
---

itd-111 acceptance criterion 8, as amended at the close, says the product thinker's 2026-09-29 ruling (an agent's check on macOS plus the Linux CI run stand in for the per-platform check by a person) is recorded in .abcd/work/DECISIONS.md under that date, but no line of DECISIONS.md carries it at 8ffb6d069 or on main: the three 2026-09-29 entries concern the pre-commit sources refresh, a guidance principle and the five-agent ceiling. The shipped intent cites a ruling the ledger does not hold. Either the ruling is appended (the staleClose pull-request body says a separate rulings lane appends it) or the amendment cites where the ruling actually lives.
