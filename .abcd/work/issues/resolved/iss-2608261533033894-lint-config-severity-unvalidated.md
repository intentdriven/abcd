---
schema_version: 1
id: "iss-2608261533033894"
slug: "lint-config-severity-unvalidated"
severity: "minor"
category: "bug"
source: "agent-observation"
found_during: "bughunt-a round 9"
found_at: "internal/core/lint/config.go"
resolution: "LoadConfig decodes strictly (DisallowUnknownFields) and refuses an enabled rule or token with an off-enum severity; both live configs load unchanged"
impact: fix
resolved_by:
  commit: "ac06c817"
---

A record-lint or docs-lint rule whose severity is missing, misspelt, or off-enum emits findings that print but count toward no exit code, so the gate exits 0 beside a non-empty findings list. internal/core/lint/config.go decodes with a plain json.Unmarshal (a misspelt key silently zero-values the field) and validates only banned-token successors; the exit paths in cmd/record-lint and abcd docs lint count Severity == blocker verbatim. The sibling engines fail closed on exactly this vocabulary (repolint Evaluate refuses a finding severity outside error/warn as a rule bug; guard Validate rejects an unknown tier on the committed guard.json; banlist AddPublic refuses to write a severity lint will happily read). The config is a documented trust boundary in this very file. The same underlying cause silently disarms a rule via a misspelt enabled key. All 27 live rules spell both correctly today — latent, one character from silent. Detector: watched-fail test loading a config with an off-enum severity; acceptance: LoadConfig refuses an enabled rule or token whose severity is outside the engine's enum, and unknown config keys are refused.