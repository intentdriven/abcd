---
id: dcn-2611280309536132
slug: a-model-matrix-hidden-test-run
date: 2026-11-28
---
2026-11-28 — A model-matrix hidden-test run (spc-2610161949559050 step 10) has a fixed deadline of 30 minutes, the copy's export included, past which the copy is unexamined, never failed. Grounds: the declaration names no test deadline yet, and a run needs a bound; 30 minutes covers a cold build and test of a repository of abcd's size while keeping a host's hand-back, which waits on the run, from holding for hours. To reverse: add an optional declared test deadline, held to at most maxHiddenDeadline (6 hours), shown on the consent screen and covered by the plan's digest, and pass it in place of hiddenRunDeadline.
