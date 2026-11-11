---
schema_version: 1
id: "iss-2611101318271021"
slug: "a-record-that-waits-on-a-person-s-ruling"
severity: "minor"
category: "process"
source: "agent-observation"
found_during: "autonomous run of 7 Nov 2026"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/capture/eligible.go"
remedy: "Waits on a ruling first: then carry one hold on a record that build and reserve refuse with its reason, and that the status board and abcd <id> show."
---

A record that waits on a person's ruling carries no mark that build, reserve and the status board all show, so a coordinator or a lane can start it by mistake. The drain hands back an issue whose remedy opens with Waits on, but reserve takes such an issue by hand, the board does not show it, and a record whose ruling is noted only in its prose carries nothing at all. On 8 Nov at 06:05 a coordinator sent a lane to iss-2611051719381143, which waits on a ruling about what the board may read, and the lane reserved it by hand and started an implementer before the mistake was caught. A person must choose where the hold lives: the existing remedy prefix, a field of its own, or both.
