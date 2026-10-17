---
schema_version: 1
id: "iss-2610111729375108"
slug: "lfs-repos-get-pointer-files-in-lanes"
severity: "major"
category: "bug"
source: "agent-finding"
found_during: "security triage and fix run, 2026-10-11"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/implement/loop/lane.go"
remedy: "Run the git commands that write a lane's working tree (worktree add, restart's reset --hard and clean, and any other isolated call that checks files out) with the person's global config, as pickGit already does (ScrubbedEnv), so the repository's own Git LFS setup applies; prove with a test over a repository with a global-config LFS filter (a stand-in filter in a temp HOME is enough) that a new lane and a restarted lane hold the smudged bytes, watched failing first"
resolution: "Fixed by routing every git command that writes or judges a lane's working tree through gitutil.WorkingTreeCommand, which reads the person's global and system config (ScrubbedEnv) with the ExecPins pins, automatic maintenance off and GIT_NO_LAZY_FETCH kept: the lane's git worktree add, a restart's staging snapshot, reset --hard and clean, and the land/hold stage's git worktree remove, alongside pickGit's existing commit, merge and status calls, which now build on the same helper. Proved by TestANewLaneHoldsWhatTheGlobalFilterSmudges, TestARestartedLaneHoldsWhatTheGlobalFilterSmudges and TestALaneRemovalJudgesTheTreeAsThePersonsGitDoes (internal/core/implement/loop/globalconfig_test.go), each watched failing first over a stand-in content filter and ignore file defined only in a temporary global config."
impact: fix
---

abcd's lane worktrees and restarts in a repository that uses Git LFS hold LFS pointer text instead of the real files. Lane creation (`git worktree add`, internal/core/implement/loop/lane.go) and a restart's `reset --hard` (restart.go) run through gitutil's isolated environment, which sets GIT_CONFIG_GLOBAL=/dev/null (internal/gitutil/repo.go), so the filter that a normal, global `git lfs install` defines is never seen and nothing is smudged. Reproduced 2026-10-11 with git-lfs 3.7.1: in a repository whose image is tracked by LFS, the person's own `git worktree add` writes the image's bytes, while the same command with GIT_CONFIG_GLOBAL=/dev/null writes `version https://git-lfs.github…` pointer text. The implementer then builds and tests against placeholders. This is abcd changing how a repository works, which the security scope rule (principle abcd-fixes-what-it-adds) forbids.
