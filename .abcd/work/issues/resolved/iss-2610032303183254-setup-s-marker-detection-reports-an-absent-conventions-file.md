---
schema_version: 1
id: "iss-2610032303183254"
slug: "setup-s-marker-detection-reports-an-absent-conventions-file"
severity: "minor"
category: "bug"
source: "user-observation"
found_during: "autonomous run A resumed 2026-09-25 (lane agentsStep5fix, from the review of step 5)"
origin: researcher-authored
production_mode: hand-written
remedy: "In detectMarkerDrift, for a missing target ask markerFolderRefusal (access(2) with W_OK|X_OK on the folder, the check the write itself makes first) and raise a non-resolvable gap naming that the folder cannot take the file, the same shape as marker.unreadable, instead of a resolvable marker.missing; grounds: detection and the write then share one check, so a gap is resolvable only when the write can resolve it, and access(2) creates nothing."
resolution: "Detection asks the marker write's own first check, access(2) on the target's folder, before raising a gap for a block to plant or rewrite, and raises marker.unwritable, required and not resolvable, when the folder cannot take a new file; TestMarkerInAFolderThatCannotTakeItIsNotAResolvableGap covers an absent and an outdated AGENTS.md in a mode-0555 root."
impact: fix
resolved_by:
  commit: "67e472c7736e2a26a43319c22173c7bf37ab0e1c"
---

Setup's marker detection reports an absent conventions file as a resolvable marker.missing gap even when the folder it would be created in cannot take a new file: with the repository root at mode 0555 and no AGENTS.md, detectMarkerDrift raises marker.missing (required, resolvable), install promises to plant the block and its write refuses on every run (permission denied creating the lock or the temporary file beside it), so the gap never closes and each run repeats the promise. The defect is on main: detectMarkerDrift there classifies an absent file as missing without asking whether its folder can take one.

## Grounds

- pursued: an absent or outdated conventions file in a folder abcd cannot create files in raises a non-resolvable gap and install leaves no marker gap outstanding; a resolvable marker gap for such a folder, or a run that still reports marker.missing in Remaining, would show it wrong
