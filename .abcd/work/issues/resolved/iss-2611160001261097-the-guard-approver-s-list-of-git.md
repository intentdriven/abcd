---
schema_version: 1
id: "iss-2611160001261097"
slug: "the-guard-approver-s-list-of-git"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "manual-capture"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/guard/approvepaths.go"
remedy: "judge each such setting against git's documentation and list it, with one list serving the approver and the hook"
resolution: "The approver's list of git settings that start a program now holds each setting git 2.52's git-config(1) and git-archive(1), and Git LFS 3.7.1's git-lfs-config(5), document whose job is to name a command, a program path or a helper, to pick the tool, viewer or signing program git starts, or to switch on a program, refused as a local git config write and as a git command's own setting; the approver also judges a section rename onto a section that holds such a setting, the clone, init, fetch, pull, ls-remote, archive, push, fetch-pack and send-pack options git documents as the command-line form of one, git's own --exec-path given a directory, and the environment variables git documents as naming a program, set on a git command or by an earlier command of its line; the editor and askpass variables and settings pass only for a value that is exactly true or a colon; a git config write with GIT_CONFIG set is judged as a --file write; a GIT_CONFIG_GLOBAL or GIT_CONFIG_SYSTEM an earlier command sets is refused; a program-naming or GIT_CONFIG name an earlier command assigns without a value the line spells (read, printf -v, a for loop, an assigning expansion) is carried as a value the line does not fix; a line holding a loop, a function, an eval or an assigning expansion is carried whole before any git command in it is judged, and a command line a git setting or option runs is judged with the carry; a command line rebase -x, submodule foreach, bisect run, difftool -x, filter-branch's --setup and command filters, or send-email's --to-cmd, --cc-cmd, --header-cmd and --sendmail-cmd hand git is approved only when spelled out and cleared as a line of its own, and grep -O's value as a pager's is; send-email --smtp-server given a path, instaweb's --httpd, --browser and --module-path, and the --tool of difftool and mergetool are refused as their settings are; submodule.<name>.update is refused only for a value beginning with an exclamation mark; the guard's hook never read this list and is unchanged; the drive's held set keeps its own list, and a test fails when a program class it holds is missing from the approver's."
impact: fix
---

The guard approver's list of git settings that name a program is not yet judged one by one against git's documentation
