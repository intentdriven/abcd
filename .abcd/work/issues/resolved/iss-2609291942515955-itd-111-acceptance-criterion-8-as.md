---
schema_version: 1
id: "iss-2609291942515955"
slug: "itd-111-acceptance-criterion-8-as"
severity: "minor"
category: "inconsistency"
source: "review-followup"
found_during: "autonomous run A resumed 2026-09-25: fidelity audit itd-111"
origin: researcher-authored
production_mode: hand-written
resolution: "The ruling itd-111 criterion 8's amendment cites is recorded: the 2026-09-29 entry in .abcd/work/DECISIONS.md where the product thinker answered twenty-six owed rulings says itd-111's last criterion is met by an agent's timing check on macOS plus the Linux CI run, so the citation resolves as written and the criterion text stands."
impact: internal
resolved_by:
  commit: "ec768aac3"
---

itd-111 acceptance criterion 8, as amended at the close, says the product thinker's 2026-09-29 ruling (an agent's check on macOS plus the Linux CI run stand in for the per-platform check by a person) is recorded in .abcd/work/DECISIONS.md under that date, but no line of DECISIONS.md carries it at 8ffb6d069 or on main: the three 2026-09-29 entries concern the pre-commit sources refresh, a guidance principle and the five-agent ceiling. The shipped intent cites a ruling the ledger does not hold. Either the ruling is appended (the staleClose pull-request body says a separate rulings lane appends it) or the amendment cites where the ruling actually lives.

## Grounds

- pursued: the amended criterion 8 names a 2026-09-29 ruling in .abcd/work/DECISIONS.md, and the twenty-six-ruling entry of that date carries the itd-111 clause; a DECISIONS.md with no 2026-09-29 line naming itd-111's macOS and Linux runs would show it wrong
