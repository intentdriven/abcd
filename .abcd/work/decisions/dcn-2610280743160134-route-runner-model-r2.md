---
id: dcn-2610280743160134
slug: route-runner-model-r2
date: 2026-10-28
---
2026-10-28 — Route-runner-model r2 (iss-2610161801358637), rebased onto local-pin-r2: a route record's override joins run-state version 16 with a route's pins, and drain-state version 2 with a judge route's pins, rather than taking version 17. Grounds: r2 already joined the override to the pins' version while neither was released, version 16 is still unreleased (local-pin-r2 lands just before r2), and a version holds the unreleased additions that land together; a file of version 15 or lower carrying override is refused, as before. If a release writes version 16 before r2 lands, r2 moves the override to version 17 before it lands.
