---
schema_version: 1
id: "iss-2611260001393033"
slug: "while-the-person-is-away-the-stop-hook"
severity: "minor"
category: "bug"
source: "user-observation"
found_during: "live check of the sub-agent start hook, 2026-11-25"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/surface/cli"
remedy: "Hold only a session the away run owns (the one the person left running, or one abcd's drive started); let a print-mode session that abcd did not start end after its task, and stop holding any session once the set of owed items has not moved over two blocks in a row, before the host's cap does it"
---

While the person is away, the stop hook held a one-off print-mode session nine times running, until the host overrode it: an agent-started check (claude -p in a scratch clone of this repository, one fixed task, done in its first turn) was told work was owed and kept from ending, spending eight turns that answered nothing, and the host ended it with its own cap warning (check stop_hook_active). The away hold reaches every session that shares the repository's root commit, including ones the person did not start and abcd did not drive.
