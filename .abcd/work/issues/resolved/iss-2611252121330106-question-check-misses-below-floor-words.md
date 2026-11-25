---
schema_version: 1
id: "iss-2611252121330106"
slug: "question-check-misses-below-floor-words"
severity: "minor"
category: "bug"
source: "user-observation"
found_during: "an interview on 25 November"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/question/check.go"
remedy: "Hold the product thinker's tabs to the knowledge floor in the check: a list of its below-floor words, read from the floor's own entry so the two cannot differ, each refused unless the question introduces it in one sentence; and have the question writer read the floor and refuse a brief whose material is in those words, returning the plain words it would use."
resolution: "The question check now holds a tab for the product thinker to the knowledge floor: each word the floor page's table lists, which a test holds equal to the check's own list, is a refusing finding of the register rule naming the plain words to use, unless the tab's text introduces it in one sentence before its first use."
impact: fix
---

A question to the product thinker passes the question check with no finding while it uses words the knowledge floor says they are not assumed to know. The floor names version control (a checkout, a branch, a merge, a commit, a worktree), continuous integration, a merge queue, a checksum and more, and says a question that fails it is rewritten; the check judges the register by record handles and commands alone. Met on 2026-11-25: a question on who may let abcd merge on its own, written by the question writer from a brief worded in internals, passed the check, and the person answered that it was unclear and full of jargon. Shown the same day: the same question naming a worktree, a checkout, the merge queue, a commit, CI and a checksum, with the mode set to the product thinker, has 0 findings.
