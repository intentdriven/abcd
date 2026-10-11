---
schema_version: 1
id: "iss-2610101900387053"
slug: "a-drain-opens-a-full-lane-on-an-issue"
severity: "minor"
category: "ux"
source: "agent-observation"
found_during: "abcd-60 drain run 2026-10-10, lane for iss-2610080546210831"
origin: researcher-authored
production_mode: hand-written
found_at: "commands/drain.md"
remedy: "Before a drain opens a lane, list the commits on the default branch since the issue's capture that touch its found_at path; when any exists, open the lane with a first stage that checks whether the remedy already holds and, if so, resolves the issue against that commit instead of implementing."
---

A drain opens a full lane on an issue whose fix has already landed under another issue's id. On 2026-10-10 the drain took iss-2610080546210831 (prepare-this-repo's unconditional core.hooksPath step); commit afa6e7b16 had already made that step conditional while citing iss-2610081312364884, so the record stayed open. The lane spent about 35 minutes and 92k tokens to find nothing to fix, then added a guard test so the landing could resolve it. Neither the dry run nor abcd capture mentions sees this, because the fixing commit names a different issue.
