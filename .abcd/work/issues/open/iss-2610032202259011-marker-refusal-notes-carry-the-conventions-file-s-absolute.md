---
schema_version: 1
id: "iss-2610032202259011"
slug: "marker-refusal-notes-carry-the-conventions-file-s-absolute"
severity: "minor"
category: "security"
source: "user-observation"
found_during: "autonomous run A resumed 2026-09-25 (lane agentsStep5, from step 3's reviews)"
origin: researcher-authored
production_mode: hand-written
remedy: "Drop the path from the wrapped error at its source: wrap the OS error's cause (the *PathError's Err, or fsutil's sentinel) rather than the error that names the path, in every 'could not be read/written/examined: %w' note of the marker code, so ahoy and embark both print only the file's base name they already name; grounds: AGENTS.md's no-absolute-local-paths rule and fsutil.RedactHome's purpose, and a test asserting no note contains the target's directory."
---

Marker refusal notes carry the conventions file's absolute path through the wrapped OS error: installMarkerFileLocked and removeMarkerFileLocked (internal/core/ahoy/marker.go) wrap the guarded read's error with 'it could not be read: %w', and embark's dry-run refusal (internal/core/ahoy/embark_marker.go) does the same, so the embark result's note, rendered and in --json, prints the target's full path, the account name included, wherever the file cannot be read.
