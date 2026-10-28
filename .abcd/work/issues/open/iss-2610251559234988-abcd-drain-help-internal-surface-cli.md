---
schema_version: 1
id: "iss-2610251559234988"
slug: "abcd-drain-help-internal-surface-cli"
severity: "minor"
category: "drift"
source: "agent-finding"
found_during: "docs-currency review of docs/, 2026-10-24, reader D5"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/surface/cli/drain.go"
remedy: "In internal/surface/cli/drain.go:54-55 change 'and a person's own abcd build <iss-N> still takes it' to 'and a person's own abcd build <iss-N> still takes it when no other rule would hand it back', then run go generate ./internal/surface/cli so docs/reference/cli/commands.md follows."
---

abcd drain --help (internal/surface/cli/drain.go:54-55, generated into docs/reference/cli/commands.md:1190-1191) says of a promoted issue that 'a person's own abcd build <iss-N> still takes it'. Since 6d4307058 a person's build takes a promoted issue only when no other rule would hand it back; one that another rule holds (security, category, severity, 'Waits on', a deferral, no remedy) is refused with that rule's reason (internal/core/implement/loop/check.go:219-230, 251-286). That commit updated commands/drain.md:155 but not the help string, so the reference tells a person that build takes a promoted security or major issue it refuses.
