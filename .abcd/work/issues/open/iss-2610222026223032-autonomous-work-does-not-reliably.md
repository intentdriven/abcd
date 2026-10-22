---
schema_version: 1
id: "iss-2610222026223032"
slug: "autonomous-work-does-not-reliably"
severity: "minor"
category: "future-work-seed"
source: "user-observation"
found_during: "PT asked for it during the 2026-10-22 crash recovery"
origin: researcher-authored
production_mode: hand-written
found_at: "hooks"
remedy: "each abcd session keeps its to-do list as a file in the local tier (.abcd/.work.local/, one per session), written as it works; abcd's PreCompact hook makes the session bring it up to date before a compaction and the SessionStart and post-compaction hooks inject it back, so the session resumes from the list; deliver it as the first slice of the session checklist intent itd-2610170638184486"
---

Autonomous work does not reliably survive the host's automatic compaction: after a compaction a long-running session keeps only a summary, and the work it was in the middle of (which lanes it dispatched, which PRs it is watching, what it promised a peer, the next step of a multi-step plan) survives only if the session happened to write it down. The person wants every autonomous session's work to survive auto-compaction, for example through a to-do list kept in the checkout's local tier (.abcd/.work.local) that the session reads back after a compaction. Related: the draft intent every abcd session keeps a checklist (itd-2610170638184486), being planned on 2026-10-22.
