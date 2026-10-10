---
schema_version: 1
id: "iss-2610090821531394"
slug: "read-only-git-log-runs-repo-gpg-program"
severity: "major"
category: "security"
source: "agent-finding"
found_during: "private security report, filed 2026-10-05"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/gitutil/repo.go"
remedy: "Pin `-c log.showSignature=false` in `isolatedArgs` beside the hooks and fsmonitor pins; prove it with a gitutil test (watched fail first) that a checkout with log.showSignature=true, a gpg.program script and one gpgsig commit leaves the mark empty on `abcd disembark probe`, `LoadHistory`, the mention walk and the decisions-append subject read, each still returning the subject; sweep siblings (log and show calls through pickGit, ScrubbedEnv or any other runner)."
resolution: "Every isolated git command now carries log.showSignature=false through one shared pin list (gitutil.ExecPins), so a read-only log or show over a signed commit no longer starts the repository's gpg.program; the probe, site history, mention walk and decisions-append subject reads all route through it. TestReadOnlyLogDoesNotRunARepoSigningProgram proves it."
impact: fix
---

abcd's read-only `git log` and `git show` calls do not pin `log.showSignature=false`, so a checkout with `log.showSignature=true`, a repo `gpg.program` and one `gpgsig` commit runs that program as the operator during probes such as `abcd disembark probe`.

A private security report, fixed in this release; its advisory, with the full text and reproduction, is published with the release.

Evidence (lines at main 7549ca2d5): the comment at internal/gitutil/repo.go:22 names log as a command the hooks and fsmonitor pins make safe; `isolatedArgs` (internal/gitutil/repo.go:52) does not set log.showSignature. Sinks: `gitReverts` asks for `log --format=%s` (internal/core/lifeboat/sources_git.go:35) under `abcd disembark probe`; `LoadHistory` runs `git log --reverse --name-status` (internal/core/site/dates.go:63); `walkMentionCommits` (internal/core/capture/mentions.go:360); the decisions-append subject read (internal/core/lint/decisionsappend.go:308). A script that exits 0 (and in a replay one that exits 1) leaves the call returning the subject and a nil error. The directory must be a copy or zip including `.git`; a clone does not carry it.