---
schema_version: 1
id: "iss-2610252214288494"
slug: "commands-launch-md-533-535-says-the"
severity: "minor"
category: "drift"
source: "agent-finding"
found_during: "brief single-source study, 2026-10-25, reader S03"
origin: researcher-authored
production_mode: hand-written
found_at: "commands/launch.md"
remedy: "Replace 'so the branch is exactly two commits' with the roll, the receipts and, when a finding was deferred, the captures, as protocol.go composes them"
---

commands/launch.md:533-535 says the receipts commit makes 'the branch ... exactly two commits', but internal/core/release/protocol.go:69-71 composes a captures commit after the receipts, as commands/launch.md:943-949 and 04-launch.md:151-153 also say
