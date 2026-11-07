---
schema_version: 1
id: "iss-2611071242184002"
slug: "a-refused-receipt-idles-a-drive-to-its"
severity: "critical"
category: "bug"
source: "user-observation"
found_during: "first live abcd drain --drive"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/implement/loop/runner_pause.go"
remedy: "Treat a receipt refused for the work as consumed: move it aside with its refusal and relaunch the role, so the failed-launch bound counts it and a third refusal hands the lane back; and have the drain's stuck check, which already sees the refusal repeat, reach a lane the drive is holding."
---

A drive whose implementer handed in a receipt with a failing definition of done never launches again and never hands the lane back. At each hourly outage wake it re-reads the same receipt, is refused for the same reason and waits another hour: the held role skips a launch while a receipt stands (internal/core/implement/loop/runner_pause.go heldRole), and the bound of three failed launches counts only launches, so it is never reached. Under --until the outage clock has no limit, so one bad receipt idles a drive to its end. The first live drive spent five hours this way on its first lane.
