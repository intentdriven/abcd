---
schema_version: 1
id: "iss-2611031146255825"
slug: "no-abcd-verb-drives-an-intent-build-to"
severity: "minor"
category: "process"
source: "user-observation"
found_during: "coordinating the build lanes, 2026-11-03"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/implement/loop"
remedy: "Give abcd build a drive mode run from a plain terminal that steps the run in-process until it is done, waiting on the queue and the merge, refusing and stopping at anything that needs the person, under the same walls and consent as drain --drive."
related_intents: [itd-2611031342011261]
---

No abcd verb drives an intent build to done unattended: implement step takes one stage and exits, so a build of several steps needs something outside abcd to re-run it after each stage, wait out the merge queue, step again once the lane's pull request merges, and open the next lane. On 2026-11-03 the coordinator did this with a polling helper script (drive-run.sh) and timed wake-ups; drain --drive (itd-2610222215485265) covers drain lanes only, and itd-2610240933025076 starts a build but does not carry it through.
