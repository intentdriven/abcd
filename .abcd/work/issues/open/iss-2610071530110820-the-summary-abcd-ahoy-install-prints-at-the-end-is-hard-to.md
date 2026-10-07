---
schema_version: 1
id: "iss-2610071530110820"
slug: "the-summary-abcd-ahoy-install-prints-at-the-end-is-hard-to"
severity: "minor"
category: "ux"
source: "user-observation"
found_during: "abcd ahoy install in a downstream repository on v0.13.2, 2026-10-07"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/ahoy/install_summary.go"
remedy: "In a Terminal, show the summary in pages: one page for what changed, one for what to do next, and one for detail. Close with a short status block in the board's colours, each line a word and a symbol as well as a colour: done (✔ N changes made), to do (⚠ commit the removed file and the drain decision record, naming the command), and problems if any (✖). Rewrite each action as an instruction or 'nothing to do', name a removal as 'removed' in the detail list, and print plain text without colour or paging off a terminal and under --json."
---

The summary abcd ahoy install prints at the end is hard to read. Problems:
1. It runs past a screen as one block: the answered questions, 'abcd ahoy install — clean', 'abcd is set up in this repository.', six what/why/action paragraphs, then a 'detail:' list of written paths and a long 'note:' about visibility.
2. Each action line reads oddly ('Nothing. To change a setting, run…', 'Nothing, unless the kind is wrong; then…').
3. What the person must still do (commit the removed conventions file, commit the drain rule's decision record) is buried mid-paragraph.
4. The detail list says 'wrote: CLAUDE.md' and 'wrote: GEMINI.md' while the paragraph above says a tool's conventions file was removed.
5. Nothing uses colour or a status mark to say at a glance whether the install succeeded and what is left.
Seen by the product thinker in a downstream repository on v0.13.2, 2026-10-07.
