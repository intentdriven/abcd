---
schema_version: 1
id: "iss-2611040337158228"
slug: "implement-receipt-path-is-refused-when"
severity: "minor"
category: "ux"
source: "user-observation"
found_during: "the v0.17.0 cut and the overnight lanes, 2026-11-04"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/surface/cli/build.go"
remedy: "Infer the run from the path, or print --run in next:; a test follows the printed line with two runs open."
refines: [iss-2610101659347329]
wontfix_reason: "Duplicate of the open record with the same claim and remedy."
duplicates: [iss-2610101659347329]
---

implement receipt <path> is refused when several runs are in progress, though the path names the run, and the driver's own next: line omits --run, so following it fails.

## Grounds

- declined: Duplicate of the open record with the same claim and remedy.
