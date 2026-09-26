---
schema_version: 1
id: "iss-2609261850045839"
slug: "preflight-runs-go-steps-on-the-path-toolchain-not-the-declared-one"
severity: "minor"
category: "bug"
source: "review-followup"
found_during: "autonomous run A resumed 2026-09-25: #728 CI failure"
origin: researcher-authored
production_mode: hand-written
found_at: "Makefile"
---

make preflight runs its Go steps (go build, go vet, go test, the race lane, and every go run and go test its prerequisites make) on the go found on PATH, which is go1.27.1 on this machine, while CI builds and tests with the toolchain go.mod declares (go 1.26.7) through setup-go's go-version-file. iss-2609081953452204 closed exactly this skew for gofmt alone, by resolving the format gate through the declared toolchain; the build and test half was left on PATH. On 2026-09-26 pull request 728 failed CI on a test whose assertion depended on the go 1.27 encoding/json error wording, after a clean local preflight: the push gate judged the tree with a toolchain CI does not use, so a green preflight vouched for nothing on that point. Fix: run preflight's Go steps under the declared toolchain, resolved by the same resolver the format gate uses rather than a second one, and refuse loudly, naming the skew, when it cannot be fetched, exactly as the format gate refuses. Detector: a test holding every Go step under preflight to the declared toolchain.
