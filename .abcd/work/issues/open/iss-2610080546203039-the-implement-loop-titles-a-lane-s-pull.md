---
schema_version: 1
id: "iss-2610080546203039"
slug: "the-implement-loop-titles-a-lane-s-pull"
severity: "minor"
category: "bug"
source: "agent-finding"
found_during: "landing PR #865 overnight, 2026-10-08"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/implement/loop/land.go"
remedy: "When the landing composes a lane's pull-request title and body from the issue, add a 'Refs: <id>' line for every other record id the title or body names (and is not already declared), so RS004 does not refuse the PR."
---

The implement loop titles a lane's pull request from its issue's title; when that title names another record id, RS004 refuses the PR because no Refs line declares it (PR #865 named iss-2609261536147903 from its issue's title; fixed by hand by editing the body).

## Evidence 2026-10-11

A second occurrence: PR #947 (run-2610111820413440, iss-2610110359273529) quoted its issue's first line, which names iss-2610092011131598, into both the title and the body. The attribution job's RS004 step refused both ("the pull-request title names iss-2610092011131598 without declaring its relation to it", and the same for the body). It was fixed by hand: a `Refs: iss-2610092011131598` line added to the body, which clears the title mention too. Every lane whose issue text names another record will hit this until the landing writes the declarations itself.
