---
schema_version: 1
id: "iss-312"
slug: "guard-wrapper-probe-tests-write-flock"
severity: "minor"
category: "tech-debt"
source: "agent-finding"
found_during: "bughunt-round-1"
found_at: "internal/core/guard/wrappers_test.go"
resolution: "wrapper probes run in a TestMain temp dir; the four committed stray files are removed."
impact: internal
---

guard wrapper-probe tests write flock lock files into the tracked package source tree and four zero-byte artefacts are committed
## Evidence
`internal/core/guard/wrappers_test.go:258-260` — `runsCommand` does `exec.Command(name,args...)` with no `cmd.Dir`, so cwd is the package source dir. The flock probe (`:108-109`, `:223` `genericProbeValues={"1","0","/tmp","root","PATH"}`) probes boolean flags; real flock reads the value token as its lockfile operand and `O_CREAT`s it. Result: `internal/core/guard/{0,1,PATH,root}` are tracked zero-byte files (`git ls-files` confirms; added in 0f2d6ef). The commit hides it: tracked-empty + regenerated-empty keeps `git status` clean.

## Adversarial verdict: CONFIRMED (minor)
Reproduced: rm the four, run the test, they re-materialise and `git status` is clean. Platform-gated to Linux+util-linux (fires here). Fix: `git rm` the four files + isolate the probe cwd (a package-scoped `TestMain` temp dir, or thread `t.TempDir()` through `runsCommand`). Verified clean: classifications unchanged (121), package tests pass, zero artefacts, tree clean, build/vet/gofmt clean. Self-masking committed junk in a core source package, contrary to the repo tier discipline — above nitpick.
