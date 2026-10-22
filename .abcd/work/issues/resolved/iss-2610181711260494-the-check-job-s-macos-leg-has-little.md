---
schema_version: 1
id: "iss-2610181711260494"
slug: "the-check-job-s-macos-leg-has-little"
severity: "minor"
category: "process"
source: "agent-finding"
found_during: "manual-capture"
origin: researcher-authored
production_mode: hand-written
found_at: ".github/workflows/ci.yml"
remedy: "Shorten the macOS race pass below about 40 minutes on a slow runner, starting from its slowest packages (internal/core/implement/loop, internal/surface/cli, internal/core/claim), or ask the technical facilitator to raise the macOS ceiling and warning with the spread as the evidence"
resolution: "The macOS leg's ceiling is 70 minutes and its warning 60 (dcn-2610181756328249), with ten runs of 35.1 to 49.3 minutes as the evidence; a faster race pass is the longer lever, iss-2610181757540736"
impact: internal
---

The check job's macOS leg has little headroom before its 50-minute warning: across nine merge-group runs on 2026-10-18 it took 35.2, 36.5, 38.9, 42.3, 42.4, 43.0, 43.9, 49.4 and 44.8 minutes, and the run of #1099 came within 0.6 minutes of the warning. Nearly all of it is the one race-enabled pass over the whole module, 41.9 to 47.4 minutes in the last three; the plain-only guard step after it takes under a minute. The warning fires at 50 and the job ceiling is 60.

## Grounds

- pursued: a merge-group macOS leg stays below 60 minutes, so the warning fires only on a run slower than any seen; one past 60 would show the leg needs a faster pass, not another raise
