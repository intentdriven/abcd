---
schema_version: 1
id: "iss-2610120729071657"
slug: "abcd-cannot-launch-or-manage-cloud"
severity: "minor"
category: "future-work-seed"
source: "user-observation"
found_during: "release preparation with cloud sessions, 2026-10-12"
origin: researcher-authored
production_mode: hand-written
found_at: "cloud session orchestration in abcd's run verbs"
remedy: "Promote to an intent: abcd starts a cloud session for a ledger item with a brief abcd composes (identity, toolchain, rules, stop-and-report), lists running and finished cloud sessions with their branch and status, and collects each one's result (a pushed branch or its stop report); the person's launch path (routines or the cloud session API) is the mechanism to research first"
related_intents: [itd-2610120738075547]
---

abcd has no way to launch or manage cloud agent sessions. On 2026-10-12 a session asked to move three drain-eligible issues to cloud sessions: its sub-agents could not reach the cloud and quietly started on the local machine instead, and the real cloud sessions had to be started by the person at claude.ai/code. Nothing in abcd starts a cloud session with the right brief, lists the running ones, or collects their results, so cloud work is launched and tracked by hand.
