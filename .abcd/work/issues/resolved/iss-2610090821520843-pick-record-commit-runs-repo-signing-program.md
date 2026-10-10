---
schema_version: 1
id: "iss-2610090821520843"
slug: "pick-record-commit-runs-repo-signing-program"
severity: "minor"
category: "security"
source: "agent-finding"
found_during: "private security advisory GHSA-4j38-5p8w-gf5r, filed 2026-10-05"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/implement/loop/pickcommit.go"
remedy: "Pass `-c commit.gpgsign=false` as a `pickGit` argument before the `commit` subcommand; prove it with a loop test (watched fail first) that the picked record commit leaves the gpg.program, gpg.ssh.program and defaultKeyCommand marks empty and still succeeds, the pre-commit hook still does not run, and a clean filter for a real LFS path still runs; sweep siblings (every commit, merge commit or tag abcd composes, where tag.gpgSign and the merge in GHSA-486x-8h9p-pq5x can sign too)."
resolution: "The shared pin list gitutil.ExecPins now carries commit.gpgsign=false, and pickGit prepends that list instead of its own copy, so the pick's record commit and a sync's merge commit no longer start gpg.program, gpg.ssh.program or gpg.ssh.defaultKeyCommand; the commit is still made and content filters still run. TestPickGitStartsNoRepoSigningProgram proves it."
impact: fix
---

The picked implement run's hooks-off record commit runs the repository's signing program (`gpg.program`, `gpg.ssh.program` or `gpg.ssh.defaultKeyCommand`) as the operator when the checkout sets `commit.gpgsign=true`.

Private security advisory GHSA-4j38-5p8w-gf5r (draft, severity medium). Full text, evidence and reproduction: the security-drain-2026-10-09 run directory in the main checkout's local tier. This record stays uncommitted until its fix lands; the fix commit adds it directly to resolved/.

Evidence (lines at main 7549ca2d5): `pickCommit` (internal/core/implement/loop/pickcommit.go:92) calls `pickGit(wt, "commit", "-q", "-m", ..., "--", rel)` (internal/core/implement/loop/pickcommit.go:149). `pickGit` prepends only the hooksPath, fsmonitor and quotePath pins and runs under ScrubbedEnv (internal/core/implement/loop/pickcommit.go:71), while the comment at internal/core/implement/loop/pickcommit.go:20 says a hook dispatcher is code the loop does not run. Repo-local `commit.gpgsign=true` overrides a global false. The commit runs only for a picked run whose `Pick.Lane` is this lane.

Reproduction: on git 2.39.5, set `commit.gpgsign=true` and `gpg.program` to a mode-0755 script in the checkout; start a picked run and let `abcd implement step` reach the record commit. The script runs with gpg's `--status-fd=2 -bsau` arguments, the pre-commit hook does not, and git exits 128 when the script writes no signature. `gpg.format=ssh` with `gpg.ssh.program`, and `gpg.ssh.defaultKeyCommand` with no signing key, each start too. `-c commit.gpgsign=false` before `commit` exits 0 and starts nothing.
