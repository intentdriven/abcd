---
schema_version: 1
id: "iss-311"
slug: "cmd-record-lint-and-scaffold-sync"
severity: "nitpick"
category: "security"
source: "agent-finding"
found_during: "bughunt-round-1"
found_at: "cmd/record-lint/main.go"
resolution: "cmd/record-lint and cmd/scaffold-sync scrub the env via gitutil.IsolatedEnv before git rev-parse."
impact: internal
---

cmd record-lint and scaffold-sync resolve the repo root with an unscrubbed git rev-parse so an inherited GIT_WORK_TREE redirects the lint or release gate
## Evidence
`cmd/record-lint/main.go:100` and `cmd/scaffold-sync/main.go:64` — `exec.Command("git","rev-parse","--show-toplevel")` with no `cmd.Env`. An inherited `GIT_WORK_TREE`/`GIT_DIR` from a subdirectory redirects discovery; record-lint (and `--release-gate`) then lints the wrong tree. Identical shape to the fixed `capture.discoverRepoRoot` (`internal/core/capture/roots.go:67`, IsolatedEnv with an explanatory comment). scaffold-sync landed in v0.5.0 as a verbatim copy AFTER the principles doc declared the git-subprocess sweep "audited clean".

## Adversarial verdict: CONFIRMED (nitpick)
Reachability real (empirically `GIT_WORK_TREE=/x git rev-parse --show-toplevel` returns /x, and record-lint loads that tree's config). Dev-only tools, fail-closed if the redirected tree lacks the config — hence nitpick. Fix: `cmd.Env = gitutil.IsolatedEnv()` on both (both already fall back to os.Getwd, so no CI regression). Also corrects two overstated "sweep finished / audited clean" sentences in the principles note. Not prior art: iss-45 (surface-boundary duplication, no env angle), iss-264 (output sanitisation).
