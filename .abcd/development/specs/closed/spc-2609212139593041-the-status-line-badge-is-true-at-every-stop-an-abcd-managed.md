---
id: spc-2609212139593041
slug: the-status-line-badge-is-true-at-every-stop-an-abcd-managed
intent: itd-2609212130146198
origin: researcher-authored
production_mode: hand-written
---
# the-status-line-badge-is-true-at-every-stop-an-abcd-managed

## Summary

The design record for itd-2609212130146198: the badge's three states, the guard on the question tool, the setter and the reset.

## Scope

1. **States**: the statusline renderer maps mode file values to the three labels and refuses to print a bare tag (criterion 1).
2. **The guard**: `abcd guard hook` on the host's question tool (PreToolUse on the question tool's name) reads the mode file; managed → exit 2 with the two settings named (criterion 2).
3. **The setter**: unchanged `mode` verb (criterion 3).
4. **The reset**: the UserPromptSubmit hook (the rules loader's) reads a `question_open` marker the guard writes when it admits a question, resets the mode to managed, clears the marker, prints one stderr line (criterion 4).
5. **The paint**: the ANSI reset after the badge (criterion 5).

## Out of scope

- Deriving the addressee; other states.

## Approach

Two hooks that already run gain one read each; the marker is a file in the local tier; both captures resolve in the lane.

## Footprint

- packages: internal/core/mode, internal/core/guard, internal/core/statusline, hooks/
- tests: the three renders; the guard's refusal and admission; the reset on the next prompt with the marker; the paint

## How the criteria are satisfied

| Criterion | Where |
| --- | --- |
| 1 three states | scope 1 |
| 2 guard refuses | scope 2 |
| 3 setter | scope 3 |
| 4 reset | scope 4 |
| 5 paint | scope 5 |
