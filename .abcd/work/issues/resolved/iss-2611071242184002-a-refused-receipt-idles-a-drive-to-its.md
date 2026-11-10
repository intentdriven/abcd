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
resolution: "A drive sets aside an implementer's receipt refused for the work it reports, only while the file holds the bytes judged and no runner has claimed the role since, kept in a new directory under the lane's aside directory with only the report and the definition of done's output at the names the loop assigned them directly in the lane's directory, never a path the receipt chooses, the refusal and the digest in the run record, and below the stuck bound starts the role again; each consumed receipt counts as a refused step under one check whatever it lacked, so the third in a row hands the lane back as stuck at the drain's stuck bound, with or without a bound on the drive and across a fresh drive on the same state, with no further launch after the receipt that reaches the bound whether the runner handed it in or a step found it standing; the count is kept on the drain's issue runs, and a refusal of another kind restarts it; the drive also gives a stuck lane back to the next drain move, so a receipt it cannot consume no longer holds the lane to the outage clock."
impact: fix
resolved_by:
  commit: "254ee93de591cc2d9d5eb2d8c85482871326f3b9"
---

A drive whose implementer handed in a receipt with a failing definition of done never launches again and never hands the lane back. At each hourly outage wake it re-reads the same receipt, is refused for the same reason and waits another hour: the held role skips a launch while a receipt stands (internal/core/implement/loop/runner_pause.go heldRole), and the bound of three failed launches counts only launches, so it is never reached. Under --until the outage clock has no limit, so one bad receipt idles a drive to its end. The first live drive spent five hours this way on its first lane.
