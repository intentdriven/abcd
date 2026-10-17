---
schema_version: 1
id: "iss-2610161821254457"
slug: "inside-a-bash-c-or-sh-c-script-a-delete"
severity: "nitpick"
category: "bug"
source: "agent-finding"
found_during: "re-review of fix/checkable-shell-forms 99cea9c43 (abcd-05 Sonnet, 2026-10-16)"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/guard"
remedy: "Count a stdin redirect from a file that an earlier variable-built segment of the same -c script wrote as feeding the delete, the way a pipe already counts."
---

Inside a bash -c or sh -c script, a delete that reads its names from a file redirected into it (xargs with a delete command and stdin from a file an earlier variable-built command wrote) is not refused by shell-c-variable-delete, because only a pipe or a here-document counts as feeding the delete; the host still stops to ask before such a script runs.
