---
schema_version: 1
id: "iss-2610090821531570"
slug: "record-lint-agent-diff-runs-repo-diff"
severity: "minor"
category: "security"
source: "agent-finding"
found_during: "private security advisory GHSA-q333-3p8r-xc2f, filed 2026-10-05"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/lint/agentcontract.go"
remedy: "Pass `--no-ext-diff --no-textconv` on the unified diff in `promptVersionChanged`, as decisionsappend.go already does; prove it with a lint test (watched fail first) that with the range armed diff.external, diff.<driver>.textconv and diff.<driver>.command leave the mark empty and an unbumped prompt edit is still reported, while the name-only path list and default unarmed lint are unchanged; sweep siblings (every git diff abcd parses)."
resolution: "The armed agent-contract bump check now asks git for its own diff: the unified diff passes --no-ext-diff and --no-textconv, and both of its diffs blank the repository's content filters first (gitutil.FilterOverrides), so no repository diff driver, textconv or clean filter runs or writes the text the check parses. TestAgentContractDiffRunsNoRepoDiffDriver proves it for diff.external, diff.<driver>.command, diff.<driver>.textconv and filter.<name>.clean."
impact: fix
---

record-lint's armed agent-diff check runs `git diff --unified=0` without `--no-ext-diff --no-textconv`, so a repository `diff.external`, `diff.<driver>.textconv` or `diff.<driver>.command` runs as the operator and its stdout can forge the `prompt_version` bump the check looks for.

Private security advisory GHSA-q333-3p8r-xc2f (draft, severity medium). Full text, evidence and reproduction: the security-drain-2026-10-09 run directory in the main checkout's local tier. This record stays uncommitted until its fix lands; the fix commit adds it directly to resolved/.

Evidence (lines at main 7549ca2d5): `promptVersionChanged` runs `gitutil.Run(repoRoot, "diff", "--unified=0", rangeSpec, "--", rel)` (internal/core/lint/agentcontract.go:388-389) and returns true on any added line prefixed `+prompt_version:`. The range is armed only by `record-lint -agent-diff` (cmd/record-lint/main.go:124, `lint.ArmAgentDiff`) or the Makefile record-lint target (`origin/main...HEAD`); default `abcd lint` does not arm it. The neighbouring decisions-append diff already passes `--no-ext-diff` and `--no-textconv` (internal/core/lint/decisionsappend.go:396, rationale at :378). A CI checkout of a pull request does not install the author's `.git/config`; a local run does.

Reproduction: on git 2.39.5, in a local checkout, plant `refs/remotes/origin/main`, change a prompt path, set `diff.external` to a mode-0755 script, and run `go run ./cmd/record-lint -agent-diff origin/main...HEAD`. The script runs; if it prints `+prompt_version: 9.9.9` the unbumped-prompt finding is absent and `lint.Lint` returns nil. `* diff=evil` with `diff.evil.textconv` or `diff.evil.command` runs too.
