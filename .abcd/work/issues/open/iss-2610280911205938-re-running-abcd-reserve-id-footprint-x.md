---
schema_version: 1
id: "iss-2610280911205938"
slug: "re-running-abcd-reserve-id-footprint-x"
severity: "minor"
category: "bug"
source: "agent-observation"
found_during: "autonomous-run-2026-10-28"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/claim"
remedy: "Make a repeated --footprint on a held id add to the footprint (union) and print the resulting list; narrowing becomes an explicit flag."
---

Re-running abcd reserve <id> --footprint X on an id already held replaces the held footprint with X instead of adding X to it, so extending a reservation path by path silently drops the earlier paths and leaves them open to peers. Two sessions met it on 27-28 Oct and had to re-reserve with the full list in one call.
