---
schema_version: 1
id: "iss-2610290750089812"
slug: "when-load-preflights-is-lowered-to-one"
severity: "minor"
category: "bug"
source: "user-observation"
found_during: "review of whole-run admission steps 1-2 (itd-2610262127138979), 2026-10-29"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/implement/load_wait.go"
remedy: "At one slot, open the race gate at once only while no other preflight's race run is live; a live one keeps it shut as above one, and the test asserts both the drop-to-one open and the wait behind a live race run."
---

When load.preflights is lowered to one while two whole runs are admitted, the race gate's at-one shortcut opens for a run that is waiting behind the other's live race run, so two race stages overlap. At two slots, runs A and B are both admitted. A is in its race stage, and B waits at its race gate because A's race run is live. The person lowers the count to 1, and B's race gate opens at its next look. The shortcut assumes a first gate that admitted this run alone, which no longer holds for runs admitted under the higher count. Lowering the count is meant to reduce load, and here it does the opposite. The spec of itd-2610262127138979 asks for this shortcut, and TestWaitRaceGateOpensOnceTheCountFallsToOne asserts it.
