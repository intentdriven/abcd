---
schema_version: 1
id: "iss-2609251455354719"
slug: "the-cold-reading-detection-position-s-declared-window-is-now"
severity: "minor"
category: "observation"
source: "agent-observation"
found_during: "autonomous run A resumed 2026-09-25"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/config/reading-presets.json"
---

The cold-reading detection position's declared window is now 1,010,000 estimated tokens (widening 1,000,000) after the recalibration at 1a19fe55, which puts the assembled detection input past the million-token reader window the preset entries' own comment names as the bound. Each merged corpus lane grows it further, so the next recalibration will push widening past a million too. The documented alternative remedy is narrowing an entry's kinds (the comment measures widening without test at about 530,000), which changes a reading instrument and needs a ruling. Until then a reading at detection cannot be handed to a million-token reader whole.
