---
schema_version: 1
id: "iss-2611060835302148"
slug: "the-int-trap-in-scripts-preflight-scans"
severity: "minor"
category: "tech-debt"
source: "agent-observation"
found_during: "v0.16.2 cut"
origin: researcher-authored
production_mode: hand-written
found_at: "scripts/preflight-scans.sh"
remedy: "Run the scan in the background and kill it on the interrupt, or say in the script why the delay is accepted."
---

The INT trap in `scripts/preflight-scans.sh:90` is deferred until the command substitution it interrupts ends, so a descendant that outlives the interrupt holds the script until that descendant exits. At worst a person's Ctrl-C is delayed by that long; it has not been seen with the real scan.
