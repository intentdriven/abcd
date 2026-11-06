---
schema_version: 1
id: "iss-2611061847520694"
slug: "drain-dry-run-reads-as-a-dump"
severity: "minor"
category: "ux"
source: "user-observation"
found_during: "first real drain dry run ahead of a long run"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/surface/cli"
remedy: "Open the plain render with the counts as a coloured summary, a progress bar of the cap against what is eligible, and the first few lanes it would take; the cap is the drive's default of eight unless --max is given, which means admitting --max beside --dry-run, as that pair is refused today; cap each outcome's list with the rest behind a flag; give the running drive the same bar. Colour and bar only on a terminal that takes them, honouring NO_COLOR, and --json unchanged."
---

The plain render of abcd drain --dry-run reads as a dump, not a report: 1856 lines on this ledger, every record on a line of its own, the counts only at the very end, no colour and no sense of how far anything has got. The eligible issues do come first, in the order a drain takes them, but on a ledger this size they push everything else off the screen, so someone deciding whether to start a drain has to scroll to the bottom for the counts and still cannot see how much of the ledger the cap covers, or what needs them. It is too much for a technical facilitator, let alone a product thinker. The drive's own running output has the same want: nothing shows how far a drain has come through its cap.
