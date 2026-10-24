---
id: itd-2610240919030303
slug: when-the-folder-abcd-installs-into-is
spec_id: spc-2610241008191355
kind: standalone
suggested_kind: null
reclassification_history: []
builds_on: [itd-2610240823055889]
severity: minor
origin: researcher-authored
production_mode: hand-written
impact: additive
---

# Setup offers to put abcd's folder on the PATH

## Press Release

> When the folder abcd installs into is not on the person's PATH, setup asks once whether to add it to their shell settings, changes them only on a yes, never adds the same line twice, and undoes the change when abcd is removed; declined, the line to add is on the checklist at the end.

## Why This Matters

_Split out of itd-2610240823055889 by the product thinker on 2026-10-24; drafted by the agent, not yet confirmed._

On 2026-10-24 the product thinker had to put abcd on the PATH by pasting the README's long install line before setup could finish. The research found that tools disagree on whether setup should edit shell settings; the product thinker chose that setup asks.

## Mechanism

- We expect the edit to be safe to make and to undo because setup edits only a file it can read whole, adds one marked line, and can find and remove that line later.


## Scope Conditions

- The zsh and bash shells. <!-- cond: cond-2610241008190652 -->
- macOS and Linux. <!-- cond: cond-2610241008197828 -->


## Acceptance Criteria

_Each criterion confirmed by the product thinker, one at a time, on 2026-10-24._

- Given abcd's folder is not on the PATH, when setup runs at a terminal, then it asks once whether to add it, naming the file and the line; on a yes it keeps a copy of the file, adds one line, and says to open a new terminal.
- Given the person's settings already add the folder, or the settings file is a link kept elsewhere (shared dotfiles), when setup runs, then it does not edit the file: it does nothing in the first case, and puts the line on the checklist in the second.
- Given the person answers no, when setup ends, then the line is on the checklist; and given abcd is removed, when the removal runs, then the line abcd added is removed and nothing else in the file changes.


## Open Questions

- resolved: Decision 2026-10-24 on shell settings reverses a written rule (the brief's setup page: "abcd prints a one-line export fix and never edits a shell profile"), and the status line does not need it, since it runs the recorded absolute path. Confirm the reversal, with its costs (which file per shell, linked dotfiles, undoing it on uninstall, a new terminal before it takes effect), or keep the export line on the checklist. (raised 2026-10-24 by the design review) (answered 2026-10-24; see Decisions)

## Decisions

- 2026-10-24, the product thinker: when `~/.local/bin` is not on the PATH, setup asks whether to edit the person's shell settings to add it, and edits them only on a yes; declined, the line to add goes on the end-of-run checklist.
- 2026-10-24, the product thinker, confirming the reversal: setup asks, then edits the person's shell settings to add abcd's folder to the PATH, reversing the written rule that abcd never edits a shell profile; the costs named by the design review (which file for each shell, linked dotfiles, undoing the edit when abcd is removed, a new terminal before it takes effect) are this plan's to meet.

## Audit Notes

_Empty. Populated by intent-auditor when intent moves to shipped/._
