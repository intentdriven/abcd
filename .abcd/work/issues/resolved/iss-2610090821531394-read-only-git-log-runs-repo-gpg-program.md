---
schema_version: 1
id: "iss-2610090821531394"
slug: "read-only-git-log-runs-repo-gpg-program"
severity: "major"
category: "security"
source: "agent-finding"
found_during: "private security advisory GHSA-9fp7-gmjr-v28j, filed 2026-10-05"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/gitutil/repo.go"
remedy: "Pin `-c log.showSignature=false` in `isolatedArgs` beside the hooks and fsmonitor pins; prove it with a gitutil test (watched fail first) that a checkout with log.showSignature=true, a gpg.program script and one gpgsig commit leaves the mark empty on `abcd disembark probe`, `LoadHistory`, the mention walk and the decisions-append subject read, each still returning the subject; sweep siblings (log and show calls through pickGit, ScrubbedEnv or any other runner)."
resolution: "Every isolated git command now carries log.showSignature=false through one shared pin list (gitutil.ExecPins), so a read-only log or show over a signed commit no longer starts the repository's gpg.program; the probe, site history, mention walk and decisions-append subject reads all route through it. TestReadOnlyLogDoesNotRunARepoSigningProgram proves it."
impact: fix
---

abcd's read-only `git log` and `git show` calls do not pin `log.showSignature=false`, so a checkout with `log.showSignature=true`, a repo `gpg.program` and one `gpgsig` commit runs that program as the operator during probes such as `abcd disembark probe`.

Private security advisory GHSA-9fp7-gmjr-v28j (draft, severity high). Full text, evidence and reproduction: the security-drain-2026-10-09 run directory in the main checkout's local tier. This record stays uncommitted until its fix lands; the fix commit adds it directly to resolved/.

Evidence (lines at main 7549ca2d5): the comment at internal/gitutil/repo.go:22 names log as a command the hooks and fsmonitor pins make safe; `isolatedArgs` (internal/gitutil/repo.go:52) does not set log.showSignature. Sinks: `gitReverts` asks for `log --format=%s` (internal/core/lifeboat/sources_git.go:35) under `abcd disembark probe`; `LoadHistory` runs `git log --reverse --name-status` (internal/core/site/dates.go:63); `walkMentionCommits` (internal/core/capture/mentions.go:360); the decisions-append subject read (internal/core/lint/decisionsappend.go:308). A script that exits 0 (and in a replay one that exits 1) leaves the call returning the subject and a nil error. The directory must be a copy or zip including `.git`; a clone does not carry it.

Reproduction: on git 2.39.5, put a mode-0755 script in the copied checkout, set `gpg.program` to it and `log.showSignature=true`, and point HEAD at a commit object carrying a `gpgsig` header. From another cwd run `abcd disembark probe` on it: the script runs and probe returns a report and nil. With unsigned history the script does not run. `rev-list`, `status`, `diff --name-only` and `cat-file -p` do not run it; `show --format=%s --no-patch` of the signed commit does.
