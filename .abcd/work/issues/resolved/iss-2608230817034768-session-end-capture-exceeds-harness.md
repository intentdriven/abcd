---
schema_version: 1
id: "iss-2608230817034768"
slug: "session-end-capture-exceeds-harness"
severity: "major"
category: "bug"
source: "user-observation"
found_during: "session-end 'Hook cancelled' post-mortem 2026-08-23"
found_at: "internal/surface/cli/cli.go"
resolution: "Capture is split across the two hooks: SessionEnd stages the raw transcript in one write, the next SessionStart redacts and stores it through the same fail-closed path. Exit went from scaling with size (3.13s at 4.1MB, 8.89s at 11.8MB) to flat (0.15s-0.41s across the whole range). The store invariant is untouched because staging is not the store, and a staged file is the outcome record the store never had."
impact: fix
resolved_by:
  commit: "bc3e439"
---

`hook session-end` redacts the whole transcript in-line before it writes, at
roughly 0.7 s per MB, and the host cancels a SessionEnd hook during shutdown
rather than wait for it. Every session whose transcript is big enough to push
that work past the cancellation budget is therefore dropped, silently. Field hit
2026-08-23: session `7d884491` exited with `SessionEnd hook [...] failed: Hook
cancelled` and never reached the store.

Dropped, but not yet destroyed, and the earlier wording here overstated it. Only
abcd's redacted copy is lost; the host's own raw transcript stays on disk, and
`abcd history capture` ingests retroactively, so a missed session is recoverable
by hand for as long as that file survives. The loss becomes irreversible only
when the host prunes its transcript, which it does on its own schedule and
without telling anyone. So the real exposure is a race nobody is watching:
a bounded, unannounced recovery window opened by a failure that reports nothing.
That is worse than a clean permanent loss in one respect — a permanent loss is
at least a fact, whereas this is a fact with an unknown expiry, and every day it
goes unnoticed is a day of the window spent.

This is **not** iss-2608210934566223 recurring. That was the blocking bootstrap
download at exit, fixed in `baf0551`; on the field-hit machine the plugin-root
binary was present, so the bootstrap branch was never taken. The hook reached
`exec "$r/abcd" hook session-end` and was cancelled inside abcd's own work. The
two failures share a symptom and a cancellation mechanism, and nothing else —
`baf0551` removed one source of slowness at exit but left the budget itself
unguarded.

Measured over every one of one repo's harness transcripts — all 24, not a
sample — replaying each through `hook session-end` and checking it against
`abcd history list`:

| transcript | size | redaction | in store | note |
| --- | --- | --- | --- | --- |
| `f6c93513` | 0.00 MB | 0.13 s | yes | |
| `81e39450` | 0.01 MB | 0.15 s | yes | |
| `8db3dbd6` | 0.01 MB | 0.16 s | yes | |
| `d4c9d65e` | 0.25 MB | 0.31 s | yes | |
| `75f8641e` | 0.23 MB | 0.32 s | yes | |
| `e5c69120` | 0.40 MB | 0.40 s | no | pre-bootstrap |
| `b7fa33b3` | 0.71 MB | 0.65 s | no | still running |
| `6802420c` | 0.69 MB | 0.66 s | yes | |
| `5263dd84` | 1.18 MB | 0.98 s | no | still running |
| `38ab27e8` | 1.25 MB | 1.03 s | yes | |
| `e7c58f16` | 1.94 MB | 1.43 s | yes | |
| `f1a5692a` | 1.98 MB | 1.47 s | yes | |
| `18c908e9` | 1.88 MB | 1.51 s | yes | |
| `484ea221` | 1.85 MB | 1.52 s | no | **counterexample** |
| `c43de2e1` | 2.54 MB | 1.95 s | no | still running |
| `f050596a` | 3.78 MB | 2.77 s | no | pre-bootstrap |
| `134e647f` | 3.33 MB | 2.78 s | no | |
| `51fc2a94` | 3.96 MB | 3.01 s | no | |
| `7d884491` | 4.11 MB | 3.13 s | no | the field hit |
| `cc5a634b` | 5.10 MB | 3.60 s | no | |
| `6ceee26b` | 6.11 MB | 4.57 s | no | |
| `093cd456` | 6.35 MB | 5.80 s | no | |
| `4fbbb0a3` | 8.10 MB | 6.05 s | no | |
| `9c89b576` | 11.80 MB | 8.89 s | no | |

Read this table carefully, because it says less than it first appears to. Cost
scales cleanly with size at roughly 0.7 s per MB, and every ended transcript
costing over 1.6 s is absent from the store while every ended transcript under
1.5 s is present. But the boundary is **not** pinned: `484ea221` at 1.52 s was
dropped and `18c908e9` at 1.51 s was kept, so two samples of near-identical cost
fall on opposite sides. Three of the absences are sessions still running, and
two more (`e5c69120`, `f050596a`) precede the first record the store ever holds,
so they are plausibly pre-bootstrap rather than lost — plausibly, not provably.

So the honest reading is that a budget somewhere near 1.5 s is the best current
estimate from observational data with real confounders, not a measured constant.
Pinning it needs a controlled run that ends actual sessions at chosen transcript
sizes, which is the recovery sweep's job. Cost also tracks content as well as
bytes (secret and home-path hit density), so size is a proxy for the real
variable rather than the variable itself. What the data does establish firmly is
the direction and the steepness: past a few MB, loss is not a risk but the norm.

Those five confounded rows are the sharper finding, and they deserve to be read
before the timing data rather than as a caveat on it. Five of twenty-four rows
could not be classified from the store at all, in the one analysis whose entire
purpose was to count losses. The store records that a transcript is present; it
has no way to say that a session ended, so "absent" silently spans at least four
distinct states: never ended, ended before the store existed, ended and captured,
ended and lost. The timing curve is evidence about a budget. This is evidence
about the instrument, and it is the stronger of the two, because an instrument
that cannot distinguish the thing it counts from three other states will
misreport every future measurement taken with it, including the one that checks
whether a fix worked.

The consequence for capture is that it works precisely where it matters least. A
short session is cheap to redact and gets stored; a long, dense, expensive
session — the one actually worth keeping — is the one guaranteed to be dropped.
Nine of this repo's ended transcripts are absent from its store, and until the
outcome gap above is closed, that nine is itself a figure arrived at by hand.

The silence is designed, not accidental, which is what makes this an ADR-shaped
question rather than a bug report. `session-end`'s own comment block in
`internal/surface/cli/cli.go` reasons about precisely this failure and accepts
it: "It is the only irreversible thing abcd does. A session that ends without
being captured is gone... a missed capture is permanent, a failed capture is
merely a lost session — is why every path here degrades to log and exit 0 rather
than surfacing an error to the host." That trade is defensible for a hook that
fails occasionally. It is not defensible once the failure is systematic and
correlated with value, because the degradation mode chosen cannot report itself
— which is exactly why a week passed before anyone noticed.

Directions, none of them taken here, and none to be taken before a detector is
armed: move the write ahead of the redaction and redact in place afterwards, so
a cancellation costs cleanliness rather than the record; or capture the raw
transcript at exit and redact on the next SessionStart; or hand the work to a
detached process the harness's shutdown does not own. Each trades a different
thing away, and choosing between them wants an ADR, not a patch.
`iss-2608210934566224` (the recovery sweep) is the detector this needs and is
now the blocking dependency, not a nice-to-have: without it there is no way to
watch a fix fail.

Adjacent, and deliberately not merged into this one: `iss-2608230752354928`
records that `source_kind` conflates ingest route with source harness. The two
look like one issue and are not. That one is a field-vocabulary change on
records that exist; this one concerns records that do not exist, which no value
added to a field can describe, because `Capture` is fail-closed and the
cancellation kills the hook before anything is written. Recording a capture that
produced nothing needs an artefact written before the redaction runs, with its
own lifecycle. Merging them would produce a record no single change could close.
The natural moment to ask whether an outcome axis is owed is a redesign of that
kind vocabulary, which is why the cross-reference is worth carrying.
