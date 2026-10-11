---
schema_version: 1
id: "iss-292"
slug: "abcd-guard-s-wrapper-probe-test-wrappers"
severity: "minor"
category: "bug"
source: "agent-finding"
found_during: "bughunt-round-1"
found_at: "internal/core/guard/wrappers_test.go"
resolution: "runsCommand runs probes in a temp dir; removed the four committed junk files (0,1,PATH,root)."
impact: internal
---

abcd guard's wrapper-probe test (wrappers_test.go runsCommand) runs wrapper binaries with no cmd.Dir, inheriting the package source directory as cwd; flock opens its probe values (0 1 PATH root) as lock files, so go test writes four zero-byte files into the tracked tree — and they are already committed under internal/core/guard/
## Evidence

- `internal/core/guard/wrappers_test.go` — `runsCommand` calls `exec.Command(name, args...)`
  with no `cmd.Dir`, so the child inherits `go test`'s cwd (the package source dir).
  `consumesNextToken` feeds `genericProbeValues` (`1 0 /tmp root PATH`) into flag positions;
  for `flock` the value lands in the lock-file operand, which `flock` opens `O_CREAT`.
- `git ls-files internal/core/guard/` tracks four committed zero-byte artefacts: `0`, `1`,
  `PATH`, `root`; `go test ./...` recreates them in place.

## Adversarial review

CONFIRMED (self-verified: the four files are tracked and regenerate). Fix: set
`cmd.Dir = t.TempDir()` in `runsCommand` (and/or use temp-path probe values), then `git rm`
the four artefacts.
