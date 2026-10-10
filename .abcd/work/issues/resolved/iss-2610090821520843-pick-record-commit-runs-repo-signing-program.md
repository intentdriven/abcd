---
schema_version: 1
id: "iss-2610090821520843"
slug: "pick-record-commit-runs-repo-signing-program"
severity: "minor"
category: "security"
source: "agent-finding"
found_during: "private security report, filed 2026-10-05"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/implement/loop/pickcommit.go"
remedy: "Pass `-c commit.gpgsign=false` as a `pickGit` argument before the `commit` subcommand; prove it with a loop test (watched fail first) that the picked record commit leaves the gpg.program, gpg.ssh.program and defaultKeyCommand marks empty and still succeeds, the pre-commit hook still does not run, and a clean filter for a real LFS path still runs; sweep siblings (every commit, merge commit or tag abcd composes, where tag.gpgSign and the merge in a private security report can sign too)."
resolution: "The shared pin list gitutil.ExecPins now carries commit.gpgsign=false, and pickGit prepends that list instead of its own copy, so the pick's record commit and a sync's merge commit no longer start gpg.program, gpg.ssh.program or gpg.ssh.defaultKeyCommand; the commit is still made and content filters still run. TestPickGitStartsNoRepoSigningProgram proves it."
impact: fix
---

The picked implement run's hooks-off record commit runs the repository's signing program (`gpg.program`, `gpg.ssh.program` or `gpg.ssh.defaultKeyCommand`) as the operator when the checkout sets `commit.gpgsign=true`.

A private security report, fixed in this release; its advisory, with the full text and reproduction, is published with the release.

Evidence (lines at main 7549ca2d5): `pickCommit` (internal/core/implement/loop/pickcommit.go:92) calls `pickGit(wt, "commit", "-q", "-m", ..., "--", rel)` (internal/core/implement/loop/pickcommit.go:149). `pickGit` prepends only the hooksPath, fsmonitor and quotePath pins and runs under ScrubbedEnv (internal/core/implement/loop/pickcommit.go:71), while the comment at internal/core/implement/loop/pickcommit.go:20 says a hook dispatcher is code the loop does not run. Repo-local `commit.gpgsign=true` overrides a global false. The commit runs only for a picked run whose `Pick.Lane` is this lane.