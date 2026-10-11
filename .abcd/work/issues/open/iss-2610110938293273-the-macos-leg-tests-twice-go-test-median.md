---
schema_version: 1
id: "iss-2610110938293273"
slug: "the-macos-leg-tests-twice-go-test-median"
severity: "minor"
category: "tech-debt"
source: "agent-finding"
found_during: "CI-cycles research, 2026-10-11"
origin: researcher-authored
production_mode: hand-written
found_at: ".github/workflows/ci.yml"
remedy: "run one race-enabled pass over the whole module on macOS instead of a plain go test ./... followed by go test -race ./internal/..., keeping the slowest-first package order and the timeout budget"
---

The macOS leg tests twice: go test ./... (median 13.6 min) and then go test -race ./internal/... (median 23.1 min), of a 39.9-min median job over 8 queue runs on 2026-10-11. The race pass already runs every internal test, so the plain pass adds only the packages outside internal/ at a third of the job's time.
