---
schema_version: 1
id: "iss-2609012037443635"
slug: "ghsa-35fj-9w6f-7h62-the-admission-alternative-not-taken-memo"
severity: "minor"
category: "observation"
source: "agent-observation"
found_during: "autonomous-run-2026-09-01"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/memory/ingest.go"
---

GHSA-35fj-9w6f-7h62, the admission alternative not taken: memory ingest now admits https sources only, with no escape — the posture of abcd update and of invariant 12 in the brief. The advisory proposed an additive --allow-http flag instead (https by default, explicit opt-in for a consumer that needs a plaintext source such as an internal mirror or a local test server), wired on the CLI, commands/memory.md and the reference page. Left open as the fork: a consumer that needs plaintext has no route today; adding one is additive, and needs a decision on whether the flag also relaxes the per-hop scheme pin or only the admission.
