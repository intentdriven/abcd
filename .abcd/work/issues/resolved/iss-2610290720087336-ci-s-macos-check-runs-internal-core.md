---
schema_version: 1
id: "iss-2610290720087336"
slug: "ci-s-macos-check-runs-internal-core"
severity: "major"
category: "process"
source: "agent-finding"
found_during: "merge queue timeouts on 29 Oct"
origin: researcher-authored
production_mode: hand-written
found_at: ".github/workflows/ci.yml"
remedy: "Export ABCD_TEST_SHARDS=3 on CI's race passes (the macOS whole-module pass and the Linux internal race lane), the same internal/testshard mechanism preflight uses, keeping -timeout 40m; pin it with a test that the CI race steps and the preflight recipe set the same shard count; show the loop package's per-shard time well under 40 minutes in the pull request's own macOS queue run."
resolution: "CI's race passes (the macOS whole-module pass and the Linux internal race lane) run the longest packages as three test processes each through internal/testshard, the count preflight uses, under the same 40-minute -timeout per process; a test holds the steps to the preflight recipe's count."
impact: internal
---

CI's macOS check runs internal/core/implement/loop as one test process under -race, and it now takes most of the step's 40-minute budget: 2244s on a passing run (Actions run 40821104565), and on 29 Oct a queue entry for pull request 1201 timed out on it (run 40825987527) with no test hung, dropping the change from the merge queue. Preflight already runs the three longest packages as three processes each through internal/testshard (ABCD_TEST_SHARDS=3); CI sets no shards, so every queued change risks the same timeout.

## Grounds

- pursued: the loop package's slowest shard on the macOS queue run finishes well under the 40-minute -timeout; shown wrong if a shard still nears or crosses it
