---
schema_version: 1
id: "iss-2610090747102654"
slug: "abcd-has-a-place-for-a-short-note-of"
severity: "minor"
category: "ux"
source: "agent-finding"
found_during: "2026-10-09 ideate research leg"
origin: researcher-authored
production_mode: hand-written
found_at: "AGENTS.md"
remedy: "Give the handover note a writer and a freshness check: a verb that writes a dated where-we-are entry (appending, never replacing unread), offered at Stop and PreCompact as draft itd-2610050548126044 plans, with the session-start greeting naming the note's age; keep it in the local tier or a home store keyed on the root commit so removing a worktree does not lose it, never committed."
---

abcd has a place for a short note of where the work stands now, the local-tier handover file .abcd/.work.local/NEXT.md, but nothing keeps it: no command writes it, nothing says when it is stale, the PreCompact and Stop hooks write nothing durable to it, a session can overwrite it unread, and it is lost when the worktree holding it is removed. A short note of where the work stands, rewritten every session, is the one thing an outside agent-memory pattern adds over abcd's brief; abcd keeps it out of the repository on purpose, but leaves it unmaintained.
