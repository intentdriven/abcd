---
schema_version: 1
id: "iss-2609260057112822"
slug: "installresult-explain-labels-a-write-that-carries-no-kind-as"
severity: "minor"
category: "bug"
source: "review-followup"
found_during: "autonomous run A resumed 2026-09-25"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/ahoy/install_summary.go"
---

InstallResult.explain labels a write that carries no kind as the optional-scanner hint (its fallback is writeScannerHint), so an unkinded write would be explained as something it is not, and a kind with no entry in allWriteKinds drops out of the summary entirely. The fallback should be an honest unexplained-write item.
