---
schema_version: 1
id: "iss-2610090824041378"
slug: "the-status-board-s-now-list-shows-a-run"
severity: "minor"
category: "bug"
source: "user-observation"
found_during: "2026-10-09 /abcd status board"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/statusblock/statusblock.go"
remedy: "Look an issue-keyed run up in the issue ledger and render its title (marking it as an issue), and leave a lane out of Now once its pull request has merged or its branch is gone (or have the landing record the run as done when the merge is seen); test both with an issue-keyed run fixture, one in flight and one merged, watched fail first."
resolution: "The status board now looks an issue-keyed run up in the issue ledger and titles its Now rows with the issue's one-line summary prefixed 'issue: ' (kind issue, bucket its ledger folder), and leaves a lane out of Now once its branch is gone or, at the land stage, its branch's tip is on the default branch."
impact: fix
resolved_by:
  commit: "9650221651a438bf5497681ad23ff69c7c19a0c9"
---

The status board's Now list shows a run's lanes with an empty title when the run was started for an issue rather than an intent: statusblock.go builds each started run's row by looking the id up in the intent corpus only, so an issue-keyed lane (every drain lane, every 'abcd build <iss-N>' run) renders as a bare 'building:' line. It also counts a lane as building after its pull request merged: on 2026-10-09 the product thinker's view showed 24 blank 'building:' rows, the overnight drain's lanes, most merged days earlier but still recorded at the land stage because nothing moves a run on once its pull request lands.

## Grounds

- pursued: the board's Now list shows only lanes still being built, each titled; shown wrong if a drain lane renders a blank 'building:' row, or a lane whose pull request merged by merge commit or whose branch was deleted is still listed
