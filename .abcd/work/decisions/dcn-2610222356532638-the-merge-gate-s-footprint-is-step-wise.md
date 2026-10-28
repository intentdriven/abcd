---
id: dcn-2610222356532638
slug: the-merge-gate-s-footprint-is-step-wise
date: 2026-10-22
---
2026-10-22 — The merge gate's footprint is step-wise (abcd-ae's ruling as coordinator of 2026-10-23, spc-2610161609350616, itd-2610150602547271, following the worktree store's dcn-2610222227336378): the build reserves at start only the files steps 1-3, 5 and 6 change, leaving out internal/core/implement/loop/land.go and its test, which only step 4 changes and which the drain's reservation of iss-2610120138028442 holds while that lane waits for its preflight, and internal/surface/cli/cli.go, which only step 2 changes and which iss-2610171110154902 and iss-2610171200535090 hold until 2026-10-23T08:40Z; when the step that changes a left-out file begins, the spec's packages line widens to name it and a re-run of abcd build reserves the widened footprint (the verb reserves again when the spec has widened, per commands/build.md), judged for overlap like any reservation. Steps 4 and 6 still land as one change. <!-- record-lint: forward-looking -->
