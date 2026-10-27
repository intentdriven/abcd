---
schema_version: 1
id: "iss-2610251559345278"
slug: "minor-imprecisions-in-abcd-drain-help"
severity: "nitpick"
category: "documentation"
source: "agent-finding"
found_during: "docs-currency review of docs/, 2026-10-24, reader D5"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/surface/cli/drain.go"
remedy: "Make the seven wording changes in the long help of internal/surface/cli/drain.go, each the smallest that makes the clause true (for example 'ten unless reservation.max_live in your machine's config.json sets another'; 'abcd implement step --run <run-id> --discard <lane-id> removes ... and closes its pull request'), then run go generate ./internal/surface/cli."
---

Minor imprecisions in abcd drain --help (internal/surface/cli/drain.go): (a) 60-61, the dispositions omit unreadable (internal/core/capture/eligible.go:61-62, 500-510); (b) 78-79, the dry run 'plans from origin's default branch as last fetched', but when that cannot be read it plans from the checkout's own ledger and says so (internal/core/implement/loop/drain_origin.go:151-154), which 62-63 'never from the checkout' does not allow for; (c) 122-123, a lapsed judgement request is also not asked again before a five-minute cool-down, the take-over margin (drain_hold.go:185-202; internal/core/claim/claim.go:61; internal/core/lease/lease.go:33); (d) 143-144, abcd implement step --discard needs its lane id (the hand-back names abcd implement step --run <run-id> --discard <lane-id>, stuck.go:251-252), and the discard also closes the pull request (internal/surface/cli/build.go:1072); (e) 144-146, 'Every move renews ... waiting or not', but a move that waits on origin or stops on a given-up outage returns before the renewal (loop/drain.go:545-551, 660-669, ahead of 676-678); (f) 176-177, 'ten unless reservation.max_live ... raises it', but the key takes any whole number from 1 to 100 (internal/core/claim/maxlive.go:31-63), so it can lower the cap too; (g) 194-195, exit 3 is reliable only for the drain's own lock: a run lock held while a lane starts passes the issue over as held (internal/core/implement/loop/loop.go:1107-1121; drain.go:1079-1115).
