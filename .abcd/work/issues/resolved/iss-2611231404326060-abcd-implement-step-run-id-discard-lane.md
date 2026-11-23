---
schema_version: 1
id: "iss-2611231404326060"
slug: "abcd-implement-step-run-id-discard-lane"
severity: "minor"
category: "bug"
source: "user-observation"
found_during: "manual-capture"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/implement/loop/stop.go"
remedy: "write the acceptance only when the step it accompanies succeeds, or apply it in the same atomic write as the step; test that a refused discard leaves state.json byte-identical"
resolution: "the acceptance given with --discard, --release or --restart is written in the move's own write, so a move that is refused leaves the run's state and fingerprint byte for byte as they were"
impact: fix
---

abcd implement step --run <id> --discard <lane> --accept-config-change writes the acceptance into the run's state (rewriting state.json at the current schema with a record line that the person accepted the config change) even when the step then refuses the discard and reports that nothing was written.
