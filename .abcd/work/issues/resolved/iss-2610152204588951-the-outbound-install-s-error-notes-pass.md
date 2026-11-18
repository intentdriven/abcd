---
schema_version: 1
id: "iss-2610152204588951"
slug: "the-outbound-install-s-error-notes-pass"
severity: "minor"
category: "bug"
source: "user-observation"
found_during: "outbound one-switch final review, 2026-10-15"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/ahoy/outbound_shim.go"
remedy: "Describe the failing location in words in every outbound shim error note, and keep the raw error out of the note"
resolution: "The outbound install's notes now say why a step failed in words through causeInWords: a file error by the operation and the system's reason, git's failure by its exit status with the command that shows git's reason, anything else by a fixed sentence, so neither a path outside the working tree nor git's message (which quotes the file it could not read) reaches a note; the arm step for the commit checks takes the same words; TestOutboundNotesKeepGitsReasonAndPathsOut and TestCauseInWordsNamesNoPath hold it."
impact: fix
---

The outbound install's error notes pass git and filesystem errors through fsutil.RedactHome only, so a failure to open a linked worktree's shared git directory prints a home-relative path outside the working tree, against the rule that notes describe such paths in words.

## Grounds

- pursued: a note from the outbound install names no path outside the working tree, such as the person's own git config, while still saying what failed and how to see git's reason
