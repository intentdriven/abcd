---
schema_version: 1
id: "iss-2610240824047504"
slug: "in-a-source-checkout-of-abcd-go-run-cmd"
severity: "minor"
category: "bug"
source: "user-observation"
found_during: "ahoy install from a terminal, 2026-10-24"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/ahoy"
remedy: "treat a go run build whose module root is the checkout being installed as current (vintage: source tip), and keep the stale refusal for real binaries"
resolution: "a go run build of the checkout being installed is treated as current by the stale-binary refusal"
impact: fix
resolved_by:
  spec: "spc-2610241007434505"
---

in a source checkout of abcd, go run ./cmd/abcd ahoy install is always refused as a stale binary, because go run never stamps VCS build info, although AGENTS.md makes go run the only sanctioned way to run abcd there; every run needs --allow-stale-binary.
