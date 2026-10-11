---
schema_version: 1
id: "iss-2609252117203691"
slug: "in-the-ship-to-tag-window-a-re-run-of"
severity: "minor"
category: "ux"
source: "user-observation"
found_during: "autonomous run A resumed 2026-09-25"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/surface/cli/launch_deep.go"
resolution: "The --baseline flag belongs to launch --dry-run; launch ship does not take it. runShipIngest now asks changelog.ReleaseInFlight, the in-flight rule extracted from Derive, before the pre-flight, and in the ship-to-tag window skips it so the cut's own release-in-flight refusal (exit 1, nothing written) is what the operator sees."
impact: fix
resolved_by:
  commit: "4a37e7e87"
---

In the ship-to-tag window a re-run of launch ship now refuses on parity first, and that refusal names --baseline <tag>, a flag launch ship does not have; the older 'release vX in flight, tag pending' refusal was the clearer message there (internal/surface/cli/launch_deep.go:136-139; review3-launchdiff note). Reword the clause for the ship path, or run the in-flight check ahead of the precheck in runShipIngest.

## Grounds

- pursued: we expect a rendering ship in the ship-to-tag window to refuse as a release in flight without naming --baseline; a ship refusal there naming --baseline would show it wrong
