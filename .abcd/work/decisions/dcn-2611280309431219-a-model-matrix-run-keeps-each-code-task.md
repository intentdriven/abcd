---
id: dcn-2611280309431219
slug: a-model-matrix-run-keeps-each-code-task
date: 2026-11-28
---
2026-11-28 — A model-matrix run keeps each code task's hidden tests by content in a state file of its own beside the run's record (<run>/hidden/<task>.json), written once as the run's start places them, not in the run record's hidden_tests list (itd-2610161733316518, spc-2610161949559050 step 10). Grounds: consent writes the run record before the starter runs and never rewrites it, and that list names no task, so two code tasks' tests could not be told apart in it; the state file is written once in the same store, so it binds a copy's score to the tests its run placed as firmly. To reverse: read and place each folder in StartRun before the record is written, key the record's hidden_tests by task, and read the tests from there.
