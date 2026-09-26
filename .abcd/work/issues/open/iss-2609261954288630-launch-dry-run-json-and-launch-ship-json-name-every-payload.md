---
schema_version: 1
id: "iss-2609261954288630"
slug: "launch-dry-run-json-and-launch-ship-json-name-every-payload"
severity: "minor"
category: "security"
source: "review-followup"
found_during: "autonomous run A resumed 2026-09-25: review-drainSite"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/launch/bundle.go"
---

launch --dry-run --json and launch ship --json name every payload file absolutely: IncludedFile.ResolvedPath (internal/core/launch/bundle.go) is the file's absolute on-disk path and is tagged resolved_path, so the bundle's files list in the dry-run report and in a ship's payload.bundle carries the checkout's absolute path once per file, naming the developer's home whenever the checkout sits under it, against the iss-81 rule. Found in the drainPaths sweep by running the read-only --json verbs from a checkout under the home.
