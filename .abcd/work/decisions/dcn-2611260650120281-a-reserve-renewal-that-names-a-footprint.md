---
id: dcn-2611260650120281
slug: a-reserve-renewal-that-names-a-footprint
date: 2026-11-26
---
2026-11-26 — A reserve renewal that names a footprint keeps replacing the held footprint, as commands/reserve.md documents, rather than gaining an add-to-footprint mode; iss-2611260633438543's second half is closed by the warning that already names each dropped path (dropped_footprint in --json) and by --renew, which keeps the footprint recorded. Grounds: replacing is what lets a lane narrow its footprint, a union mode would make narrowing need a second flag, and the warning reaches the caller in the same command, before any other lane can act on the gap. Reverse by adding --add-footprint to reserve.
