---
schema_version: 1
id: "iss-2610161535233301"
slug: "abcd-lint-reports-privacy-hygiene-errors"
severity: "minor"
category: "observation"
source: "user-observation"
found_during: "manual-capture"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/surface/session/server_test.go"
remedy: "mark those deliberately illustrative tailnet addresses with abcd-lint:allow, or teach the privacy-hygiene rule that 100.64.0.0/10 test addresses are reserved for Tailscale's tailnet"
---

abcd lint reports privacy-hygiene errors for the tailnet test addresses (100.64.0.0/10, shared address space, not a documentation range) in internal/surface/session/server_test.go and internal/surface/cli/session_surface_test.go, as it does in the tailscale adapter's tests
