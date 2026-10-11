---
schema_version: 1
id: "iss-2609251455354719"
slug: "the-cold-reading-detection-position-s"
severity: "minor"
category: "observation"
source: "agent-observation"
found_during: "autonomous run A resumed 2026-09-25"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/config/reading-presets.json"
remedy: "Waits on ruling O: if the passes read less, drop the test kind (and source if still over) from the detection and widening entries in `.abcd/config/reading-presets.json` in a commit that records why, recalibrating their windows under 1,000,000; if they keep growing, give each entry a reader-bound field and have assembly split an over-bound reading into declared parts under one manifest; either way extend the window eval so a window above the declared reader bound fails unless the entry says it is read in parts."
deferred_after: "v0.11.1"
deferral_reason: "ruling owed on a reading instrument: bring the detection and widening windows back under the million-token reader bound by narrowing a preset entry's kinds, or accept that a reading at those positions is not handed whole to a million-token reader? Recalibrations keep citing this record meanwhile."
---

The cold-reading detection position's declared window is now 1,010,000 estimated tokens (widening 1,000,000) after the recalibration at 1a19fe55, which puts the assembled detection input past the million-token reader window the preset entries' own comment names as the bound. Each merged corpus lane grows it further, so the next recalibration will push widening past a million too. The documented alternative remedy is narrowing an entry's kinds (the comment measures widening without test at about 530,000), which changes a reading instrument and needs a ruling. Until then a reading at detection cannot be handed to a million-token reader whole.

## Remedy grounds (2026-09-29)

- At the base both entries are past any whole reading: detection declares 1,410,000 and widening 1,400,000 (measured at c23b5f30).
- SOTA check: the host vendor's context-window guide (https://platform.claude.com/docs/en/build-with-claude/context-windows, read 2026-09-29) gives 1M tokens as the largest current window and says recall degrades as context grows; RULER (Hsieh et al., COLM 2024, https://arxiv.org/abs/2404.06654) finds effective context well below the claimed length. So a larger reader is not on offer, and a whole reading near 1M already sits in the degraded range; both favour reading less, but the pick is the ruling's.
- Rejected: raising the declared windows again with no bound, which the record exists to stop.
