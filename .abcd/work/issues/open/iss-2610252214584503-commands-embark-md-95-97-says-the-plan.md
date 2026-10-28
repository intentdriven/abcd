---
schema_version: 1
id: "iss-2610252214584503"
slug: "commands-embark-md-95-97-says-the-plan"
severity: "minor"
category: "drift"
source: "agent-finding"
found_during: "brief single-source study, 2026-10-25, reader S10"
origin: researcher-authored
production_mode: hand-written
found_at: "commands/embark.md"
remedy: "Name the spec-store lock in commands/embark.md before 03-embark.md § 2 is reduced to a pointer."
---

commands/embark.md:95-97 says the plan is judged again 'under the target's ledger and intent locks', but the write also takes the spec store's lock (internal/core/lifeboat/embark.go:102-114; 03-embark.md:86-89), so the counterpart a pointer would lean on is the less complete copy.
