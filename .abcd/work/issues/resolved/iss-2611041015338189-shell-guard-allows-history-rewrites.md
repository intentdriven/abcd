---
schema_version: 1
id: "iss-2611041015338189"
slug: "shell-guard-allows-history-rewrites"
severity: "major"
category: "security"
source: "user-observation"
found_during: "the v0.16.1 cut, 2026-11-04"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/guard"
remedy: "Add blocker entries to the bundled registry for each command, each asking for a new commit instead: a revert, or a fix on top."
resolution: "The bundled registry refuses each of these commands, every one asking for a new commit instead. Two reading marks keep the refusals from reaching ordinary lines: subcommand_named, so git \"$@\" is not read as filter-repo, and a forge ref path with a by-lead field value, so a gh api field a substitution fills is read as force=true only where it can be. A rebase or an amend stays allowed, since a line cannot show whether its commits were pushed."
impact: fix
---

The shell guard's bundled registry refused a force push but not the other commands that rewrite published history or remove the means of recovering from a rewrite. git filter-repo, git filter-branch, git update-ref, git replace, git reflog expire, git gc --prune=now, git push --mirror, git send-pack --force, git fast-import --force and a gh api PATCH of a ref with force=true all ran unchallenged.
