---
schema_version: 1
id: "iss-275"
slug: "guard-shim-tests-inherit-real-path"
severity: "minor"
category: "bug"
source: "agent-observation"
found_during: "v0.5.1 cut preflight"
found_at: "internal/surface/cli/guard_shim_test.go"
resolution: "runShim pins PATH (system dirs plus a caller-supplied directory) and TestGuardShimFallsBackToPATH pins the PATH rung with an empty plugin root; fixed in c4ee12f"
impact: internal
---

TestGuardShimFailsOpenLoud's runShim inherits the host PATH, so on a machine with a real abcd installed the guard hook's PATH-fallback rung (shipped with the hook resolution ladder) finds it: the binary-absent case genuinely guards the session via the PATH binary, no UNGUARDED warning prints, and the test fails — environmentally, exactly the machines that dogfood the install. CI is unaffected (no abcd on PATH). Same class as iss-249, in the cli shim harness; the hooks_selfprovision_test hookRun helper already pins PATH for this reason and is the pattern to copy. Detector: the test itself on an affected machine; acceptance: go test ./internal/surface/cli/ -run TestGuardShim passes unmodified with an abcd on PATH — including a new case asserting the PATH rung guards the session when the plugin root is empty.

## Correction, 2026-09-29

The resolution was true when it was written, and its second half is not true
at this tip. `c637a734` ("fix: the hook shims run only the abcd this machine
recorded") removed the plain PATH rung that `TestGuardShimFallsBackToPATH`
pinned: the shim runs a PATH binary only when `~/.abcd/path-entry` records it.
The same commit replaced the test with `TestGuardShimFallsBackToAnOwnedPathBinary`
and `TestGuardShimRefusesAnUnrecordedPathBinary` in
`internal/surface/cli/guard_shim_test.go`. The first half, runShim pinning
PATH, still holds.
