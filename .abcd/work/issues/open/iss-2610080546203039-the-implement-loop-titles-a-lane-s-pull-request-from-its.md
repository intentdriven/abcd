---
schema_version: 1
id: "iss-2610080546203039"
slug: "the-implement-loop-titles-a-lane-s-pull-request-from-its"
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
