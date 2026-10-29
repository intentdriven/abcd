---
schema_version: 1
id: "iss-2610270935000288"
slug: "abcd-drain-unskip-given-alone-on-a-drain"
severity: "minor"
category: "bug"
source: "agent-finding"
found_during: "itd-82 lane-1 round-2 review, 2026-10-27"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/implement/loop"
remedy: "Refuse --unskip on a drain that has ended, with nothing written, naming that there is no hold to lift because the list cleared when the drain ended; --skip alone may still begin a drain, as it does today."
---

abcd drain --unskip given alone, on a drain that has ended, begins a new drain: the skip list is applied after a finished drain's state is reset, so a command meant only to lift a hold starts a drain run the person did not ask for. Found by the round-2 ruthless review of itd-82 lane-1 (build/run-2610260500113538-lane-1).
