---
schema_version: 1
id: "iss-2610250007404944"
slug: "every-preflight-on-this-10-core-mac"
severity: "major"
category: "bug"
source: "agent-observation"
found_during: "autonomous run, landing-marks preflight 3"
origin: researcher-authored
production_mode: hand-written
found_at: "Makefile"
remedy: "Give preflight's unit stage the race stage's 40-minute package budget (Makefile and the AGENTS.md definition of done), the same step taken for the race lane on 2026-10-17 from measured CI times; the per-size budget derived from history stays with iss-2610171105560765."
---

Every preflight on this 10-core Mac fails its unit stage at the 20-minute per-package budget: on 2026-10-24/25 six preflights in a row, from three sessions and through the gate's command form at a load of 5 to 7, ended with internal/core/claim, internal/core/implement/loop and internal/surface/cli each killed at 1200s with no test hung (each dump shows one test a few seconds in). The same packages take 260s, 633s and 535s on CI's Linux runner and implement/loop took 1830s in macOS CI's race pass, so the budget set on 2026-10-17 from one ten-minute run no longer fits macOS, and no preflight here can mint a receipt.
