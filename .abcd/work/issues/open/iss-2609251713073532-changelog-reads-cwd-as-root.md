---
schema_version: 1
id: "iss-2609251713073532"
slug: "changelog-reads-cwd-as-root"
severity: "major"
category: "bug"
source: "agent-finding"
found_during: "autonomous run A resumed 2026-09-25"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/surface/cli/ship.go"
---

abcd changelog and abcd launch ship hand the working directory to the cut as the repository root: run from a subdirectory of a checkout, changelog reports an empty cut and a missing surface baseline with exit 0, a plausible wrong answer; outside a checkout both refuse with git's bare 'exit status 128' instead of naming the missing checkout
