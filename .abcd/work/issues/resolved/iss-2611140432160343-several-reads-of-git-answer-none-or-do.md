---
schema_version: 1
id: "iss-2611140432160343"
slug: "several-reads-of-git-answer-none-or-do"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "manual-capture"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/gitutil/contentprograms.go"
remedy: "Judge each starts-nothing subcommand by the form it runs in, read a git that was not run as a failed read before any exit status, give the spot-check store's git the lazy-fetch floor guard, and make a failed read of the hooks path decide on each caller's safe side, naming the read"
resolution: "Fixed in this branch: subcommands on the starts-nothing list are judged by form, a git not run is a failed read before any exit status, the spot-check store's git passes the lazy-fetch floor and says when it was stopped, and a failed read of core.hooksPath lands on each caller's safe side. The ahoy banlist report's hooks-path state is left for a later change."
impact: fix
---

Several reads of git answer 'none' or do not compare where the command's form or a failed read leaves that unknown: the isolated table's starts-nothing list is by name, a hooks path that cannot be read is taken as none by three callers, and the spot-check store's git builds its own command
