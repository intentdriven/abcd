---
schema_version: 1
id: "iss-2610032202256580"
slug: "setup-s-marker-detection-reports-a-conventions-file-it-can"
severity: "minor"
category: "bug"
source: "user-observation"
found_during: "autonomous run A resumed 2026-09-25 (lane agentsStep5, from step 3's reviews)"
origin: researcher-authored
production_mode: hand-written
remedy: "Classify a target that exists but cannot be read whole as its own non-resolvable state, as markerSymlink already is (classifyMarker keeps markerMissing for an absent file only), and have detectMarkerDrift raise a non-resolvable gap naming why the file cannot take the block; grounds: detection must never advertise a resolvable gap apply can never close (marker.go's own markerSymlink comment)."
---

Setup's marker detection reports a conventions file it can never write as a resolvable marker.missing gap: classifyMarker (internal/core/ahoy/marker.go) folds every read error into markerMissing, so an AGENTS.md (or the chosen target file) that is a directory, a FIFO or unreadable raises a required, resolvable gap, install's write then refuses it ('it could not be read'), and the repository stays partial on every run with a gap install promises to close and never can.
