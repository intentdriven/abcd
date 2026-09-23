---
schema_version: 1
id: "iss-2609231526392449"
slug: "guard-hook-reads-abcd-guard-json-from-a-refused-foreign-uid"
severity: "minor"
category: "security"
source: "user-observation"
found_during: "autonomous run 2026-09-23"
origin: researcher-authored
production_mode: hand-written
---

guard hook reads .abcd/guard.json from a REFUSED foreign-uid root when the workdir (or session cwd) is that root, while the refusal note says it was not read (internal/surface/cli/guard.go:430, internal/core/rules/root.go:255); it can only add hazards (Strictest), but the foreign file's why/successor text reaches the agent as the block message. Fixed on feat/ruled-security-forks (da3efea6/ebc6290d), not on main.
