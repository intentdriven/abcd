---
schema_version: 1
id: "iss-2611101317242315"
slug: "renewing-or-widening-a-reservation-by"
severity: "minor"
category: "ux"
source: "agent-observation"
found_during: "autonomous run of 7 Nov 2026"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/surface/cli/reserve.go"
remedy: "Waits on a ruling first for the other-copy half; the rest needs none: have a renewal keep the held footprint when none is given, add the paths given rather than replacing the held ones, and take several paths in one flag; a test renews with no footprint and with one new path and finds the held paths kept."
---

Renewing or widening a reservation by hand repeats what the hold already knows. abcd reserve takes the footprint again on every renewal and writes the paths given in place of the held ones (since iss-2610290507512835 it warns about the paths it drops), and a renewal from any copy but the one that took the reservation is refused with exit 3, even for the same person on the same machine (since iss-2610250013552001 the refusal names the sibling worktree). On 7 Nov at 10:47 and 12:46 a renewal was refused without its footprint; on 9 Nov at 06:45 six calls with one footprint each left one path reserved; on 9 Nov at 22:22 and 10 Nov at 09:14 a build started in one worktree could be renewed only from there, and a renewal had to name the whole spec footprint again plus two files the lane changed outside it. iss-2610231004012419 declined renewal from another copy under dcn-2610311539024848, which removes hand reservations of issues and intents, but until that removal is built (iss-2610311539091723) lanes still renew by hand. A person must choose whether a renewal from another copy of the same holder is allowed after all.
