---
schema_version: 1
id: "iss-2610231057586567"
slug: "internal-core-afterupdate-afterupdate-go"
severity: "nitpick"
category: "tech-debt"
source: "user-observation"
found_during: "drain lane run-2610230715022943, 2026-10-23"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/afterupdate/afterupdate.go"
remedy: "Replace afterupdate's rootSHARe with gitutil.IsFullSHA, keeping the package's tests green, and add the same no-private-full-SHA-pattern detector that lane added to history and lifeboat."
---

internal/core/afterupdate/afterupdate.go:72 declares its own rootSHARe full-SHA pattern, a third private copy of what gitutil.IsFullSHA answers, beside the two the 2026-10-23 drain lane for iss-2610140813542607 removed from history/store.go and lifeboat/voyage.go.
