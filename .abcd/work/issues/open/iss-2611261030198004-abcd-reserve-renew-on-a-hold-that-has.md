---
schema_version: 1
id: "iss-2611261030198004"
slug: "abcd-reserve-renew-on-a-hold-that-has"
severity: "minor"
category: "ux"
source: "user-observation"
found_during: "surge run hold renewal, 26/11"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/claim/crowd.go"
remedy: "Make a renew of a lapsed hold judge overlap the way a plain reserve does, so a path also held by the same holder warns rather than refuses, and say in the renew's output that the hold had lapsed and was taken again"
---

abcd reserve --renew on a hold that has lapsed refuses with exit 3 when a path in its footprint is also in another live hold of the same holder, saying take other work, while a plain reserve of the same id and footprint from the same worktree succeeds with a warning that both holds are yours. Three surge holds lapsed this way: the renew refused, nothing was renewed, and a plain reserve with the full footprint restored each.

Two sites make the refusal. The renew request is built at internal/surface/cli/reserve.go:465 with Renew and AnyWorktree only, while the plain reserve built just above it sets WarnOwnOverlap. The overlap is then judged at internal/core/claim/crowd.go:166, where a path shared with the same holder's other copy goes ahead only with WarnOwnOverlap, or for a renew that names OwnOverlapOn and touches none of it; the renew from the command line carries neither, so it backs off as an automated reserve would.
