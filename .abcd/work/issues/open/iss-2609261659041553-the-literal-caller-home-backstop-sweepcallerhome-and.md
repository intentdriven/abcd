---
schema_version: 1
id: "iss-2609261659041553"
slug: "the-literal-caller-home-backstop-sweepcallerhome-and"
severity: "minor"
category: "security"
source: "agent-finding"
found_during: "autonomous run A resumed 2026-09-25"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/adapter/scanner/residual.go"
---

The literal caller-home backstop (SweepCallerHome and SurvivingCallerHome, internal/adapter/scanner/residual.go) reads the home only as written, so an escaped spelling of it is invisible to the one stage the store-before-commit redactors keep independent of the detector: the solidus escape (\/ between segments), its \u002f form, and a separator written as an escape run. Stage one and the stage-two rescan read those spellings through the scanner's JSON-escape views since fix/drain-scanner-identity, so a leak needs both detector passes to miss before the backstop matters; but the backstop exists for exactly that case, and against an escaped home it is not a backstop. Detector: with HOME under the /Users root, SweepCallerHome collapses the solidus-escaped and the \u002f spelling of the home to the tilde, as it does the literal one.
