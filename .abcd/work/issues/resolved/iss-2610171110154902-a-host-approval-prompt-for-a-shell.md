---
schema_version: 1
id: "iss-2610171110154902"
slug: "a-host-approval-prompt-for-a-shell"
severity: "major"
category: "bug"
source: "user-observation"
found_during: "autonomous runs, week of 2026-10-17"
origin: researcher-authored
production_mode: hand-written
found_at: "hooks/hooks.json"
remedy: "Wire the host's PermissionRequest event for Bash to abcd: while a build or drain is running in the checkout (unfinished, active within the reservation lease) or the session is one of its lanes, deny every approval prompt with a message telling the agent to rewrite the command into plain steps the guard can read, or skip the step and record why; never grant; leave interactive sessions' prompts as they are; build and drain say once at start that refusals are on."
resolution: "abcd now refuses the host's approval prompt for a shell command while a build or drain runs for the checkout or one of its lanes, telling the agent to rewrite the command into plain checkable steps or skip it; it never grants one, and build and drain say so once as they start"
impact: additive
---

A host approval prompt for a shell command stalls an autonomous run. The host asks the person to approve a command it cannot check (an unread script, a compound line), and with nobody at the terminal the run waits until someone returns. Auto-approving a repeat was considered and rejected: it lets any agent past every check by retrying, so the refusal is repeated every time instead and the agent is told how to rewrite the command.
