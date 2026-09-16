---
schema_version: 1
id: "iss-2609091155497399"
slug: "reconstruct-loads-every-record-s-full-body-even-in-the-mode"
severity: "minor"
category: "tech-debt"
source: "agent-finding"
found_during: "sub-agent transcript capture branch review"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/history/reconstruct.go"
---

Reconstruct loads every record's full body even in the mode whose whole purpose is to not render them. The thread loader reads each record body into memory for every thread in a session before the renderer consults the mode, so the spine mode, which reduces each delegate to its opening instruction and closing turn, still pays the full memory cost of every delegate it is about to discard. It is bounded per file by the record read cap rather than unbounded, and the largest main thread observed is well under that cap, so this is a ceiling rather than a leak: a session with a few dozen verbose delegates near the cap could still hold most of a gigabyte resident before the first byte is elided. The ingest path in the same package takes the opposite approach deliberately, processing one transcript at a time and discarding the bytes after probing, because the corpus it walks is far larger than memory. The fix is to let the mode reach the loader, so a spine run decodes only the head and tail turns of a thread it will summarise. Not urgent while sessions stay at the observed fan-out, and worth doing before a session with wide delegation is reconstructed on a small machine.
