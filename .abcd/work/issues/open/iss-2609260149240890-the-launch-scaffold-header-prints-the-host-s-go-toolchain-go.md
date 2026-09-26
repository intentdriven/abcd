---
schema_version: 1
id: "iss-2609260149240890"
slug: "the-launch-scaffold-header-prints-the-host-s-go-toolchain-go"
severity: "minor"
category: "ux"
source: "impl-review"
found_during: "autonomous run A resumed 2026-09-25 (fix round, review of lane launchkind)"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/surface/cli/scaffold.go"
---

The launch scaffold header prints the host's Go toolchain (go 1.25) for a repository that declares a non-Go application and carries no go.mod; the Go version is only meaningful when go.mod exists.
