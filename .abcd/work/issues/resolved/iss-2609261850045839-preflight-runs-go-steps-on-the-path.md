---
schema_version: 1
id: "iss-2609261850045839"
slug: "preflight-runs-go-steps-on-the-path"
severity: "minor"
category: "bug"
source: "review-followup"
found_during: "autonomous run A resumed 2026-09-25: #728 CI failure"
origin: researcher-authored
production_mode: hand-written
found_at: "Makefile"
resolution: "make preflight exports GOTOOLCHAIN from go.mod's go directive to every Go step it makes, and runs fmt-check second, whose resolver scripts/pinned-toolchain.sh (the one resolver the format gate uses) refuses naming the skew when the declared toolchain cannot be fetched. TestPreflightRunsTheDeclaredToolchain holds the export, the single resolver and fmt-check's place."
impact: internal
resolved_by:
  commit: "67a2adf31"
---

make preflight runs its Go steps (go build, go vet, go test, the race lane, and every go run and go test its prerequisites make) on the go found on PATH, which is go1.27.1 on this machine, while CI builds and tests with the toolchain go.mod declares (go 1.26.7) through setup-go's go-version-file. iss-2609081953452204 closed exactly this skew for gofmt alone, by resolving the format gate through the declared toolchain; the build and test half was left on PATH. On 2026-09-26 pull request 728 failed CI on a test whose assertion depended on the go 1.27 encoding/json error wording, after a clean local preflight: the push gate judged the tree with a toolchain CI does not use, so a green preflight vouched for nothing on that point. Fix: run preflight's Go steps under the declared toolchain, resolved by the same resolver the format gate uses rather than a second one, and refuse loudly, naming the skew, when it cannot be fetched, exactly as the format gate refuses. Detector: a test holding every Go step under preflight to the declared toolchain.

## Grounds

- pursued: a test whose assertion depends on standard-library wording that differs between the PATH go and go.mod's release fails preflight as it fails CI; a green preflight on such a test followed by a red CI would show it wrong
