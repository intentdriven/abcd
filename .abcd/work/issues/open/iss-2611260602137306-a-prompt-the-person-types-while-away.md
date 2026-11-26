---
schema_version: 1
id: "iss-2611260602137306"
slug: "a-prompt-the-person-types-while-away"
severity: "major"
category: "bug"
source: "user-observation"
found_during: "examining a stop-hook loop, 2026-11-26"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/autonomy/judge.go"
remedy: "Have the prompt hook record in the session's state when a prompt the person typed arrived, never one a session cron fired, and have the stop check read that session as present for fifteen minutes after it: its stops are allowed with the present line, the away record still holds for every other session, and the checks resume on their own when the window ends. Prove it with a judge test (blocked without a prompt, allowed inside the window, blocked once it ends) and a hook test that a cron's prompt opens no window."
---

With the person away, a prompt the person types does not count as their presence, so the stop check blocks every end of the reply to it until the host's continuation cap ends the turn: on 2026-11-26 a session answered the person's 'status?' and was then blocked nine times in a row, and a later session the person was talking to was blocked on each reply the same way. The person cannot add to a conversation while sessions work on without them, short of ending the away mode, which is the friction the mode exists to avoid.
