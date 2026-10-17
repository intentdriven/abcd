---
schema_version: 1
id: "iss-2610161457307727"
slug: "after-the-2026-10-16-crash-the-person"
severity: "minor"
category: "ux"
source: "user-observation"
found_during: "crash recovery on 2026-10-16 (abcd-22/abcd-a2)"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/sessionname"
remedy: "Show in the status line and the tab title the name peers use to reach the session, next to abcd's own session name. Keep a register of host session id to peer name, so a resumed session can say which name it was before."
---

After the 2026-10-16 crash the person could not tell which terminal tab held which session. Sessions message each other by the host's peer names (abcd-22, abcd-2c), but those names show nowhere a person can see. Every resume and every reboot gave each session a new peer name (abcd-22 became abcd-b4, then abcd-a2). Matching each tab to its work took a round of messages to every session, and closing the old copies had to be done by process id.
