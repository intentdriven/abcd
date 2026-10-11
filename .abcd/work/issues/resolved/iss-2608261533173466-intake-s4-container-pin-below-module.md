---
schema_version: 1
id: "iss-2608261533173466"
slug: "intake-s4-container-pin-below-module"
severity: "minor"
category: "documentation"
source: "agent-observation"
found_during: "bughunt-a round 9"
found_at: ".abcd/work/intake.md"
resolution: "S4 pins the image to the module toolchain, pre-fetches modules and mounts the module cache read-only before the network is cut"
impact: internal
resolved_by:
  commit: "7e274002"
---

The intake protocol's contained-review stage pins a container that cannot build the module. The S4 recipe pins golang:1.25 while go.mod declares a higher toolchain, and --network=none in the same line blocks the GOTOOLCHAIN auto rescue, so the one command the protocol hands a maintainer fails at the toolchain check and S5's build-or-INCONCLUSIVE tri-state misreads the stale pin as an inconclusive contribution. Drift, not born broken: the pin matched go.mod when written; the toolchain bump that advertised lockstep did not include this file, and the work tier sits outside every lint-configured tree. The recipe also lacks any module-cache provision, so even a corrected tag fails on dependency fetch under --network=none — the line has never worked as written. S4 is load-bearing: a recorded maintainer decision cites it as the mitigation that closed a review-agent exposure question. Acceptance: the pin tracks the module toolchain and the recipe provisions modules before the network is cut.