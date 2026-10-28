---
schema_version: 1
id: "iss-2610252214429155"
slug: "06-capture-md-519-527-4-keeps-the-legacy"
severity: "minor"
category: "drift"
source: "agent-finding"
found_during: "brief single-source study, 2026-10-25, reader S07"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/06-capture.md"
remedy: "Delete §4 (or reduce it to one line citing dcn-2607170000000002), and drop the 'migrator-only' framing of ForceID at internal/core/capture/capture.go:191, whose only caller is a test."
---

06-capture.md:519-527 (§4) keeps the legacy scratch migration as a staged design ('Not built yet ... on first run after install, the command parses a free-form scratch buffer'), while dcn-2607170000000002:14-15 ruled it 'satisfied-by-history ... no dead migration code built' and 05-internals/03-configuration.md:1198-1200 says it 'has nothing left to migrate'.
