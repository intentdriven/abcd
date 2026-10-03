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
---

Setup's marker detection reports an absent conventions file as a resolvable marker.missing gap even when the folder it would be created in cannot take a new file: with the repository root at mode 0555 and no AGENTS.md, detectMarkerDrift raises marker.missing (required, resolvable), install promises to plant the block and its write refuses on every run (permission denied creating the lock or the temporary file beside it), so the gap never closes and each run repeats the promise. The defect is on main: detectMarkerDrift there classifies an absent file as missing without asking whether its folder can take one.
