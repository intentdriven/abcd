---
schema_version: 1
id: "iss-2611060809106010"
slug: "abcd-development-brief-04-surfaces"
severity: "minor"
category: "inconsistency"
source: "user-observation"
found_during: "the v0.16.2 release gates"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/34-build.md"
remedy: "Rewrite lines 532 and 594-597 to say the lane pauses and the fallback's route is `none`, keeping the host hand-off for a review route only."
---

`.abcd/development/brief/04-surfaces/34-build.md:594-597` says a runner that is absent, refuses, fails, runs past its time, answers unparsably or writes a receipt the verifier refuses records a fallback naming the route that runs the role and hands the host the await; line 532 likewise says the role falls back. The binary pauses the lane instead: `pauseAwait` in `internal/core/implement/loop/runner_pause.go:609` sets the fallback's route to `none`, the run record says the lane is paused rather than handed to the host, and `abcd implement step --help` agrees, as does the chapter's own line 552. Only a review route still falls back to the host (`internal/core/implement/loop/validate.go:1116`).
