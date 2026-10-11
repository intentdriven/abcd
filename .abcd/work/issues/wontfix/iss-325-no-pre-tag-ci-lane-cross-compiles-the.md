---
schema_version: 1
id: "iss-325"
slug: "no-pre-tag-ci-lane-cross-compiles-the"
severity: "nitpick"
category: "process"
source: "agent-finding"
found_during: "bughunt-round-1"
found_at: ".github/workflows/ci.yml"
wontfix_reason: "adversarial review refuted materiality: the degradation is trusted-worktree-only and self-diagnosing; keeping it as a finding implies a fix that was declined"
---

no pre-tag CI lane cross-compiles the full release matrix so a darwin-amd64 or linux-arm64 interaction break first surfaces post-tag behind the release approval gate
## Evidence
`Makefile:4` releases four targets; `ci.yml` compiles only ubuntu-latest (linux/amd64) + macos-latest (darwin/arm64) via plain `go build ./...`; `release.yml`'s pre-approval `verify` job does not run `make build`; the first cross-compile is in the `release` job (environment: release, required reviewer), post-tag.

## Adversarial verdict: CONFIRMED facts, materiality refuted (trivial) — RECORD-ONLY
CI's two legs are a strength-1 covering array: every GOOS and every GOARCH is compiled by some lane; only the darwin/amd64 and linux/arm64 COMBINATIONS are uncompiled. The module has almost no interaction surface (zero arch-conditional files, no cgo, no x/sys; the only Stat_t reads are written portably). Empirically `go vet ./...` is clean for all four targets. Failure is fail-closed pre-publish; blast radius is one burned tag. Optional one-line fix (a pre-tag `make build` step) only if CI is touched anyway. Adjacent: nothing pre-approval asserts the `-ldflags -X ...Version` stamp matches (a `version | grep $TAG` assertion would). Not prior art: iss-48 is behavioural e2e, not arch coverage.
