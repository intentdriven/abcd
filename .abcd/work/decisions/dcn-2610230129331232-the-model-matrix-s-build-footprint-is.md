---
id: dcn-2610230129331232
slug: the-model-matrix-s-build-footprint-is
date: 2026-10-23
---
2026-10-23 — The model matrix's build footprint is step-wise and by file (abcd-ae's ruling as coordinator of 2026-10-23 for spc-2610161949559050, itd-2610161733316518, following dcn-2610222227336378 and dcn-2610222356532638): the spec's packages line named whole packages (internal/core/implement/loop, internal/core/runner, internal/core/oracle, internal/surface/cli and others) that the merge gate's, the worktree store's and abcd-f1's pin and route builds also change, so a directory reservation would hold them all for the days the matrix takes. The build reserves at start only step 1's sixteen files (the new internal/core/matrix package, internal/core/lab/procedure.go and its test, internal/core/lab/preflight.go, internal/README.md and ACKNOWLEDGEMENTS.md); each later step's files join the packages line when that step begins, and abcd build then reserves the widened footprint, judged for overlap. Generated output is reserved by no one. <!-- record-lint: forward-looking -->
