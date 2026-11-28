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

Seen again in the coordinating session itself, 2026-11-26 between 01:10Z and 02:30Z, on the installed release: with every owed item already with a running builder, reviewer or preflight, the hook returned the same "set has not moved over two wakes" text after every reply, well past the host's nine, because it returns additional context rather than a block and the host does not count that toward its cap. It also kept naming a preflight as owed while that preflight was running. This tree's build (run from the night branch) named two items and not the running preflight, so the count and the running-run reading are fixed here; the repeat itself is not.

Seen again 2026-11-28 between about 11:30Z and 11:50Z, in the coordinating session on the installed release: the hook named as owed a branch whose own `make preflight` was then waiting at the load gate's capacity queue behind other preflights, with a wake scheduled within the look interval, and returned the "has not moved over two wakes" text after each reply for about a dozen turns until the preflight began its stages. Which item lacked a counted whole-run marker, and why, was not diagnosed.
