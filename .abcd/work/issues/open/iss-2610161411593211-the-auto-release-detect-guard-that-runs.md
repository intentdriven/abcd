---
schema_version: 1
id: "iss-2610161411593211"
slug: "the-auto-release-detect-guard-that-runs"
severity: "minor"
category: "tech-debt"
source: "impl-review"
found_during: "security review of the release-chain fix"
origin: researcher-authored
production_mode: hand-written
found_at: ".github/workflows"
remedy: "Remove the dead guard or re-point it at the event that now carries verify, and rewrite its header comment to match."
---

The auto-release detect guard that runs gh run list --event push can no longer fire, because verify is always skipped on tag runs; its header comment describes the old behaviour.
