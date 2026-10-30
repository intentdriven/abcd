---
schema_version: 1
id: "iss-2610282035340363"
slug: "the-macos-leg-of-the-check-job-in-ci-yml"
severity: "major"
category: "process"
source: "user-observation"
found_during: "manual-capture"
origin: researcher-authored
production_mode: hand-written
found_at: ".github/workflows/ci.yml"
remedy: "Raise the macOS leg's timeout-minutes from 70 to 85, below the merge queue's 90-minute check response timeout, and move its warning line to 75 so it still fires ten minutes ahead; keep the ubuntu leg at 45; leave the faster race pass to iss-2610181757540736, which stays open."
resolution: "The check job's macOS leg ceiling rises from 70 to 85 minutes, below the merge queue's 90, and its warning line moves to 75 so it still fires ten minutes ahead (dcn-2610282036219503). The faster race pass that would remove the need stays open as iss-2610181757540736."
impact: internal
---

The macOS leg of the check job in ci.yml cancels merge-group runs at its 70-minute ceiling: on 2026-10-28 run 40752008734 (PR #1196's merge group) had check (macos-latest) cancelled after about 71 minutes, from 19:11:30Z to 20:22:05Z, and the queue dropped the group with failed_checks. That day's passing merge-group runs took roughly 46 to 68 minutes, so the leg runs near its cap and each cancel rebuilds every group queued behind it. The lasting remedy is a faster suite (iss-2610181711260494, resolved by the raise to 70, which named a faster race pass, iss-2610181757540736, as the longer lever).
