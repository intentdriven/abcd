---
schema_version: 1
id: "iss-2610311539091723"
slug: "abcd-reserve-still-accepts-an-issue-or"
severity: "major"
category: "process"
source: "agent-observation"
found_during: "the product thinker's ruling on iss-2610311224444576"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/surface/cli/reserve.go"
remedy: "Once iss-2610311224444576's widening has shipped, refuse reserve --footprint for an iss-N or itd-N record (release reservations and --release stay), naming abcd build as the way to start the work, and rewrite the pages and rules that teach a hand reservation"
blocked_by: [iss-2610311224444576]
---

abcd reserve still accepts an issue or an intent with a hand-named footprint of any width, though dcn-2610311539024848 rules that only a build or a drain reserves such a record; one hand hold of about 35 paths on 2026-10-31, for an issue whose found_at was one file, stalled nine of thirteen chosen builds for hours. The command pages, AGENTS.md's concurrent-sessions rule and every lane brief still tell a session to reserve each path by hand.
