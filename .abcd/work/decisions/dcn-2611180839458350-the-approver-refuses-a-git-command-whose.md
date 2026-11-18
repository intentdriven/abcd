---
id: dcn-2611180839458350
slug: the-approver-refuses-a-git-command-whose
date: 2026-11-18
---
2026-11-18 — The approver refuses a git command whose git directory, found from -C, a fixed cd or a wrapper's change of directory (env -C or --chdir, sudo -D or --chdir), or named by --git-dir, GIT_DIR, GIT_COMMON_DIR, --bare or git init --separate-git-dir, stands inside the working tree and is not the working tree's own; one whose directory, or the directory git finds its repository from, the line does not fix; and, after an earlier command of the line writes, one that runs in a directory inside the working tree other than its root or its own git directories, or names a git directory inside the working tree other than its own, because git reads that directory's configuration, which may name a program, the approver judges only the working tree's own, and a write earlier in the line may make a repository where none stood when the line was read.
