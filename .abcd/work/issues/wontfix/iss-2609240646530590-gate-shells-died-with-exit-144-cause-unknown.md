---
schema_version: 1
id: "iss-2609240646530590"
slug: "gate-shells-died-with-exit-144-cause-unknown"
severity: "minor"
category: "observation"
source: "agent-observation"
found_during: "autonomous run 2026-09-23"
origin: researcher-authored
production_mode: hand-written
found_at: "the harness background shells that ran make preflight and git push under the HOME alias"
wontfix_reason: "exit 144 sits in the host's shell management, was not reproduced, and nothing in abcd caused it; the lesson about orphans is the 2026-09-23 load-trust rule (check what runs)"
---

On 2026-09-23 at about 16:20-16:22Z, during autonomous run A, four shells running gates in three agent sessions died at once: a lane's preflight, another lane's preflight in a separate worktree, a push whose pre-push preflight "died of signal 15", and the orchestrator's own push. The killed shells exited with status 144, and their `make` children survived, reparented to init. The pattern kill that explained the two SIGTERMs earlier that day (the guard-registry record filed with this one) was ruled out: the lanes alive at the time reported no kill beyond one agent's `kill -TERM` of its own orphaned process group, checked by its working directory first. Every killed command line carried the HOME alias the run used for gates, and every killed shell was a harness background shell, so the cause is placed provisionally in the host's shell management rather than in abcd; it was not reproduced. Recorded so that the next run seeing exit 144 on a gate has this instance: the gate's log ends with no `rc=` line, the lane loses one gate run and nothing committed, and re-running the gate was the remedy. The orchestrator then stopped three orphaned process groups by group id, and one of them was a peer's live preflight; that is the other half of the lesson: an orphaned process is not an unread one, so check its working directory, its log redirect and its owner before stopping it.

## Grounds

- declined: nothing in abcd sent the signal; this would be wrong if an abcd process were shown to signal sibling shells
