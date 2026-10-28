---
schema_version: 1
id: "iss-2610251559309480"
slug: "abcd-drain-help-internal-surface-cli"
severity: "minor"
category: "documentation"
source: "agent-finding"
found_during: "docs-currency review of docs/, 2026-10-24, reader D5"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/surface/cli/drain.go"
remedy: "Add one sentence for each to the long help in internal/surface/cli/drain.go: beside the reservation step at lines 89-92 (an issue whose found_at names no path is handed back before its judgement, to be given a found_at), and beside the parked-lane sentence at lines 152-156 (a merged lane is finished, its worktree and branch removed; one merged by squash or rebase is named for the person to confirm and clean up). Then run go generate ./internal/surface/cli. Naming the cleanup in the verb's sentence belongs with iss-2610160608444348."
---

abcd drain --help (internal/surface/cli/drain.go:42-195) never mentions two things every move does. (1) On origin, an eligible issue whose found_at names no path is handed back before any judgement is asked, flagged from 'reservation' as no-found-at with the home 'give it a found_at naming the file the fix changes' (internal/core/implement/loop/drain.go:911-921; reservation.go:963-981). (2) Before its window clock, every move takes the landing's last step for each issue lane in the checkout parked at land whose pushed head is on the default branch, removing that lane's worktree and branch, and names a pull request merged by squash or rebase so the person can confirm it and clean up (internal/core/implement/loop/drain_finish.go:3-11, 154-157, 184-339; drain.go:671-686). The help's account of parked lanes (drain.go:152-156) stops at closed, conflicting and open.
