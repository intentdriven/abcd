---
id: dcn-2611101309224327
slug: this-supersedes-the-reasoning-of
date: 2026-11-10
---
2026-11-10 — This supersedes the reasoning of dcn-2611101235207798 and leaves the pass unexported for a different reason. The coordination pass in internal/core/coord has no production caller yet: only its own tests call it. Unexporting it takes it out of the exported-reach audit's view and does not satisfy that audit, whose unexport remedy fits a function its own package uses, and no production code in the package uses the pass. It is staged code, said so in its comment and on the spec's steps: step 2 of spc-2610230758581403 must export the pass and wire its first caller, or remove it (facilitator-on-delegation; grounds: two reviews of step 1, internal/reachaudit).
