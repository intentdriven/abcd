---
id: spc-2611051628562376
slug: drain-drive-live-proof-and-key-route
intent: itd-2610222215485265
origin: researcher-authored
production_mode: hand-written
---
# drain-drive-live-proof-and-key-route

## Summary

What spc-2610230813348813 did not deliver for itd-2610222215485265: the live proof of `abcd drain --drive` on the real harness, which needs a person's login and their own terminal, then the key route and the decision on a lane paused past the outage. The drive itself, its walls, its stop and resume, and its docs landed under that spec.

## Footprint

- packages: internal/core/runner, internal/core/implement/loop, internal/surface/cli
- tests: internal/core/runner/profile_test.go, internal/surface/cli/drain_drive_test.go, internal/core/implement/loop/terminal_outage_test.go

## Steps

1. **Live proof, on the one walled login** (MUST, live)
   - priority: MUST, Friday
   - login: the person, once, into the one walled directory on the account the drive runs in
   - needs: none
   - packages: internal/core/runner/wall.go
   - tests: the run's log and the three pull requests on the scratch remote; a dated receipt in the local tier
   A real `abcd drain --drive` on a scratch repository with three seeded minor issues, the real harness on this account's subscription, the cap at 3. The evidence is the run's log, the three pull requests, and the absence of any input after the yes, which is the mechanism's falsifier; one Ctrl-C mid-lane and a resuming drive are part of the run. A wall that does not hold in the real launch is fixed in `wall.go` here or stops the route and goes back to the product thinker. Live; it asks its own yes, since it spends the person's usage and opens real pull requests on a scratch remote. The first proof's lab (dcn-2611031720003633) is the method.
2. **The key route, on a key profile** (AFTER)
   - priority: AFTER
   - login: none
   - needs: none
   - packages: internal/core/runner/profile.go, internal/surface/cli/drain_drive.go
   - tests: internal/core/runner/profile_test.go, internal/surface/cli/drain_drive_test.go
   Not before itd-2611011230226453 and itd-2611040344170206 ship: a drive whose configuration names a key runs on a key profile with abcd's hooks on, the key reaching the harness by a channel the agent's shell cannot read, under the budget's caps and its composing rule (dcn-2611041550126639, dcn-2611041550129278). Until then the refusal of spc-2610230813348813's step 3 stands.
3. **A lane paused past the outage** (AFTER)
   - priority: AFTER
   - login: none
   - needs: none
   - packages: internal/core/implement/loop/terminal.go
   - tests: internal/core/implement/loop/terminal_outage_test.go
   What the drive does with a lane still paused when its outage runs out: hand it back as stuck with the runner's reason and release it, as the intent's decision 7 was written, or leave it paused on its branch for the next drive, as a stop leaves a lane. Both are on the record; the product thinker settles it before this step is built, and the intent's decision 7 is amended to the answer.

