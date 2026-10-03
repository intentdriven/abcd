---
schema_version: 1
id: "iss-2610032048280232"
slug: "embark-probe-mispredicts-the-marker-action-for-a-chosen"
severity: "minor"
category: "bug"
source: "user-observation"
found_during: "itd-2610030814013772 step 3 review 2026-10-03"
origin: researcher-authored
production_mode: hand-written
remedy: "In EnsureMarker's dry run, a file classified missing is read once more through fsutil.ReadGuarded, and an error that is not os.IsNotExist returns the same refusal the write gives (cannot write marker to X: it could not be read: ...), so probe and from agree on action and note. classifyMarker itself keeps folding a read error into missing, which ahoy detection and TestClassifyMarkerRejectsFifoPromptly rely on. Grounds: installMarkerFileLocked refuses exactly that read error; a test running EmbarkProbe and EmbarkFrom over a directory, a FIFO and a mode-000 AGENTS.md asserts the two results are equal."
resolution: "EnsureMarker's dry run re-reads a file classified missing and returns the write's own refusal for any read error but an absent file, so embark probe and embark from agree on a directory, FIFO or unreadable AGENTS.md"
impact: fix
resolved_by:
  commit: "df18541fb"
---

embark probe mispredicts the marker action for a chosen AGENTS.md that is a directory, a FIFO or an unreadable regular file: ahoy.EnsureMarker's dry run trusts classifyMarker, which folds every guarded-read error except a symlink into markerMissing, so the probe reports install (changed) while embark from, through installMarkerFileLocked, refuses the same file ('it could not be read') and reports skip. The person approves a plan that says AGENTS.md will be installed and the run then skips it; nothing wrong is written and the records land, but the plan was wrong. The embark marker tests ran the not-a-regular-file case through EmbarkFrom only.

## Grounds

- pursued: EmbarkProbe and EmbarkFrom give equal MarkerResults (skip, unchanged, same note) for a directory, a FIFO and a mode-000 AGENTS.md; TestEmbarkMarkerNeverWritesThroughALink failed on all three before the fix and passes after. A probe reporting install, or a note differing from from's, for any of them would show it wrong.
