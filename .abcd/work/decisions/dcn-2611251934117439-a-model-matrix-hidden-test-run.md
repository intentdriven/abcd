---
id: dcn-2611251934117439
slug: a-model-matrix-hidden-test-run
date: 2026-11-25
---
2026-11-25 — A model-matrix hidden-test run (itd-2610161733316518, step 10) is bounded by its environment, its working directory and its process lifetime only, and is no sandbox. The copy's code runs as the person's account with HOME set and the network reachable, as the builder's own session already let it run; a HOME override would break the toolchain's caches without making a boundary. Three limits are known and stated in the code: the group kill reaches only processes that stay in the run's process group, so one that leaves it by setsid is out of its reach; the score is the test command's exit status alone, so code whose init or TestMain exits 0 before any test runs passes; and step 11's consent text must tell the person, before they consent, that the run is not sandboxed.
