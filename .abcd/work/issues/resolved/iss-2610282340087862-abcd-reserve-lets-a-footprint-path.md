---
schema_version: 1
id: "iss-2610282340087862"
slug: "abcd-reserve-lets-a-footprint-path"
severity: "minor"
category: "ux"
source: "agent-finding"
found_during: "lane A, coordinator request after a footprint-overlap report, 2026-10-28"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/claim/crowd.go"
remedy: "Have the overlap judgement collect each shared path it lets through as generated, with the other reservation's record, and have abcd reserve print one line per such path and carry them in --json"
resolution: "abcd reserve names each shared generated file and the record it is shared with"
impact: additive
---

abcd reserve lets a footprint path through when it is a file origin's default branch declares generated, but says nothing about it: a person reading the reservation cannot tell that it shares docs/reference/cli/commands.md with another live reservation, which the second lander must regenerate after merging main. The act names a shared path only when it backs off, warns of the person's own other copy, or cannot read the declaration.
