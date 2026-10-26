---
schema_version: 1
id: "iss-2610241412028278"
slug: "a-session-holding-several-push-ready"
severity: "minor"
category: "process"
source: "agent-observation"
found_during: "abcd build itd-82 session, 2026-10-24, chaining four branches' preflights one at a time"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/landing"
remedy: "add a session-held landing queue, 'abcd land --queue <branch>...' (and a status of it), that runs each branch's preflight in its own worktree one at a time per session through the load gate, re-queues a run the gate timed out rather than failing it, lands each that passes with the land fence and any live release hold respected, and records each step so a resumed session carries on from the queue"
---

A session holding several push-ready branches drives each through preflight and landing by hand: on 2026-10-24 this session had four (docs/itd-82-landed-steps, fix/landing-marks-step-landed, fix/lapsed-release-hold, fix/name-ungated-tests), and with the rule relayed that day of one preflight per session at a time it started each preflight itself, watched for the 2-hour gate timeout (three runs timed out and had to be restarted by hand), ran 'abcd land' after each that passed, and kept the order in its own notes and wake-ups. No verb holds a session's queue of branches, so the chaining, the re-queue after a timeout and the landing all depend on the session staying alive and remembering.
