---
schema_version: 1
id: "iss-2610161457311626"
slug: "two-live-processes-can-run-one-session"
severity: "minor"
category: "bug"
source: "user-observation"
found_during: "crash recovery on 2026-10-16 (abcd-22/abcd-a2)"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/sessionname"
remedy: "At session start, register the host session id and refuse to name a second live process on the same id. Instead, print a warning that names the other process and tells the person to exit one copy before either does work. Remove the entry on session end."
---

Two live processes can run one session transcript at once, and nothing warns either. On 2026-10-16 the person resumed sessions in new tabs while the old processes were still alive, so seven pairs, one per session, briefly shared a transcript. Each copy could have dispatched agents, committed or pushed as the other, and only a cross-session message caught it.
