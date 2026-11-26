---
id: dcn-2611262236204687
slug: while-the-person-is-away-a-session
date: 2026-11-26
---
2026-11-26 — While the person is away, a session holding a scheduled wake whose prompt opens with a slash (a slash command) has no prompt read as the person's, so no prompt opens its conversation window until it holds no such wake, as with a session listing more wakes than its state keeps. Grounds: a wake is told from the person typing only by its prompt's text matching what the stop listed; a plain-text wake arrives as it was set (checked on 2026-11-26 against 47 firings of a session's own wakes, every one equal after trimming), but a slash command reaches the prompt hook expanded, so its firing would read as the person's and a frequent one would keep the session read as present for the whole away period. Failing closed costs the person only a window in such a session. Reverse by dropping the slash test from KeepWakes in internal/core/autonomy/conversation.go.
