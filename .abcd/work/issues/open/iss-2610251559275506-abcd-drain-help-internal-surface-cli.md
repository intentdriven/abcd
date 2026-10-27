---
schema_version: 1
id: "iss-2610251559275506"
slug: "abcd-drain-help-internal-surface-cli"
severity: "minor"
category: "drift"
source: "agent-finding"
found_during: "docs-currency review of docs/, 2026-10-24, reader D5"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/surface/cli/drain.go"
remedy: "Add the release hold to the list in internal/surface/cli/drain.go:141-142 (for example 'the wait for a preflight receipt, an outage's wait, a land fence that cannot read origin and one held by another person's release cut are never counted') and to the header comment at internal/core/implement/loop/stuck.go:15-17, then run go generate ./internal/surface/cli."
---

abcd drain --help (internal/surface/cli/drain.go:141-142; commands.md:1281-1283) lists the waits a lane's refusal count never includes as 'the wait for a preflight receipt, an outage's wait, and a land fence that cannot read origin'. Since b15e2e48a a land fence held by another person's release cut is a wait by design too and is never counted toward stuck (internal/core/implement/loop/stuck.go:118-120; refusal.go:57-60; reservation.go:331-339). The commit did not touch the help, so a person reading the list expects a lane held by a release cut to be handed back as stuck after three refusals or sixty minutes, and it never is. The header comment of stuck.go:15-17 lists the same three.
