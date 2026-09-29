---
schema_version: 1
id: "iss-2609290055092630"
slug: "fsutil-redactroot-and-the-scanner-s-home-sweep-still"
severity: "minor"
category: "inconsistency"
source: "impl-review"
found_during: "autonomous run A resumed 2026-09-25"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/fsutil/paths.go"
---

fsutil.RedactRoot and the scanner's home sweep still disagree on the LEADING boundary for a home of two or more segments under a longer root: SweepCallerHome sweeps '/Volumes/T7/srv/qzhome/x' to '/Volumes/T7~/x' (iss-324: a longer root does not make the name someone else's), while RedactRoot, and so RedactHome, the CLI error scrub and the install receipt, leave it whole because the byte before the root is a path-segment byte (iss-2609230641546141 chose that to stop '/private~/wt/a' mangles). The identity survives in CLI output wherever a message names a home under a backup volume or mount. Closing it needs a choice between the two rules for the '~' polarity (the '.' repo polarity has no counterpart); found while unifying the trailing half under iss-2608292037564347.
