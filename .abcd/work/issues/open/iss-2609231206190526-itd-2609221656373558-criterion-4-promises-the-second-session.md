---
schema_version: 1
id: "iss-2609231206190526"
slug: "itd-2609221656373558-criterion-4-promises-the-second-session"
severity: "minor"
category: "drift"
source: "user-observation"
found_during: "autonomous run 2026-09-23 fidelity audit"
origin: researcher-authored
production_mode: hand-written
---

itd-2609221656373558 criterion 4 promises the second session refuses a lane that recalibrates the reading corpus. Delivered: the corpus bound is applied only to paths the session declares, and a claim or lane check that declares no paths asks no corpus question (internal/core/implement/bounds.go:94-98), so a second session that names no paths opens a corpus lane unrefused. The refusal rests on self-declaration; the record should say so, or the bound should refuse an undeclared lane for the second role. Found by the fidelity audit; not fixed here.
