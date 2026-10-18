---
schema_version: 1
id: "iss-2610170711334760"
slug: "checklists-for-autonomous-sessions"
severity: "minor"
category: "process"
source: "user-observation"
found_during: "conductor-intent design session, 2026-10-17"
origin: researcher-authored
production_mode: hand-written
found_at: "the per-session checklist of the draft intent Every session keeps a checklist on disk (not yet on main)"
remedy: "Run a state-of-the-art pass on crash-safe local state files (write to a temporary file, flush, rename over the old one, flush the directory; a journal or a last-good copy; JSON validation on read), then mint an abcd lab that kills a session mid-write and reboots the host, and confirm the checklist survives intact and is re-injected, before the checklist draft's crash claim is marked confirmed."
---

The per-session checklist that autonomous sessions keep on disk (the draft intent "Every session keeps a checklist on disk", on branch chore/capture-agent-builder and not yet on main) must survive a crash: a session killed mid-write, a host or terminal restart, or a machine reboot must leave the last complete checklist readable and re-injected on the next start, never a truncated or half-written file and never an empty one. The design states this; nothing yet shows it holds. The product thinker requires a state-of-the-art pass and a lab before the claim is confirmed.
