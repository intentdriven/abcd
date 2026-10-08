---
schema_version: 1
id: "iss-2610071528375981"
slug: "during-abcd-ahoy-install-each-category-approval-setup-q1-to"
severity: "major"
category: "ux"
source: "user-observation"
found_during: "abcd ahoy install in a downstream repository on v0.13.2, 2026-10-07"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/ahoy/apply.go"
remedy: "Compose each category approval from the gaps it would apply: one material line per gap naming what changes and where (the gap's title and the file or setting it writes), with the category named in plain words in the ask (\"Update the conventions file AGENTS.md?\"), and make the Yes option name what it writes (\"Writes the 2 changes listed above\"); refuse, in the question check, a confirm whose Yes meaning points at text above when the question carries no material."
resolution: "each category approval now lists its changes, asks in plain words, and yes counts what it writes; the question check refuses a meaning pointing at text above with no material"
impact: fix
---

During abcd ahoy install, each category approval (Setup Q1 to Q5 in a managed repository, e.g. 'Apply conventions-file changes?') is asked with nothing but that one line: apply.go composes the question as "Apply " + category + " changes?" and the setup question carries no material, while its Yes option says it 'Writes what the text above describes.' There is no text above, so the person approves file writes without seeing which files change or how; the category name itself (conventions-file, git-hooks, ...) is an internal word. Seen 2026-10-07 by the product thinker in a downstream repository on v0.13.2 (screenshot: five approvals answered 'Yes, make the change' with no list shown).
