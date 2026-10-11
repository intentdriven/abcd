---
schema_version: 1
id: "iss-329"
slug: "go-1-25-has-left-the-support-window-go"
severity: "minor"
category: "tech-debt"
source: "user-observation"
found_during: "manual-capture"
found_at: "internal/core/launch/scaffold/substitutions.go"
resolution: "the toolchain moved to the Go 1.26 line (1.26.7) in lockstep across go.mod, the ci pins, and the release workflow's scaffold substitution; govulncheck scans clean on the new line"
impact: internal
resolved_by:
  commit: "25d580d"
---

Go 1.25 has left the support window: go.dev lists 1.27.0 and 1.26.7 as the stable lines, so 1.25 receives no further security patches and the pinned 1.25.13 is the line's end. The toolchain (go.mod go directive, ci and release pins, scaffold substitutions) should move to the older supported line, 1.26.x, as its own change with the parity test in lockstep