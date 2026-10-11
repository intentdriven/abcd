---
schema_version: 1
id: "iss-354"
slug: "ci-yml-s-three-go-version-pins-are"
severity: "minor"
category: "tech-debt"
source: "agent-finding"
found_during: "bughunt-round-2"
found_at: ".github/workflows/ci.yml"
resolution: "TestWorkflowGoVersionsMatchSubstitutions couples every workflow go-version pin to the scaffold substitutions; verified by mutation"
impact: internal
---

ci.yml's three go-version pins are coupled to nothing: TestSelfScaffoldParity gates release.yml and auto-release.yml against scaffold substitutions but no test reads ci.yml, so the d594511 drift where CI scanned green on a newer toolchain than the release build recurs silently on the next bump
## Evidence

- `TestSelfScaffoldParity` builds its case table from `ReleaseYMLPath`/`AutoReleaseYMLPath` only (`internal/core/launch/scaffold/scaffold_test.go:31-54`); nothing reads `.github/workflows/ci.yml`, whose three `go-version` pins (`ci.yml:217,355,390`) float free of `AbcdSubstitutions().GoVersion` (`substitutions.go:27`).
- The drift is not hypothetical: between d594511 and 8b70d70 (2026-08-19) CI scanned green on 1.25.13 while release binaries built on 1.25.12 with four unpatched stdlib CVEs. iss-289 (resolved) records that incident and its substitutions fix; this residual — no coupling, so the identical divergence recurs silently — survived it. iss-329 (open) wants the 1.26 move and assumes lockstep exists.
- Refuter verdict: CONFIRMED substantive. Fix: a test in the scaffold package asserting every `go-version:` in `.github/workflows/*.yml` equals `AbcdSubstitutions().GoVersion`.

## Correction, 2026-09-29

The resolution was true when it was written, and the test it names is not in
the tree. `df377ee6` ("fix: the format gate is pinned everywhere it runs, not
only in ci.yml") removed `TestWorkflowGoVersionsMatchSubstitutions` together
with the literal `go-version:` pins it coupled: every setup-go step reads
`go-version-file: go.mod`, which `TestEverySetupGoResolvesTheToolchainFromGoMod`
in `internal/core/launch/scaffold/goversion_lockstep_test.go` pins
(iss-2609090951291799). The drift this record names cannot recur in that form,
since no workflow carries a Go version of its own.
