---
id: dcn-2610170116582725
slug: a-land-fence-that-cannot-read-the
date: 2026-10-17
---
2026-10-17 — A land fence that cannot read the reservation from origin is a wait by design, never a refusal a drain counts toward a stuck lane (facilitator-on-delegation, drain-many-accounts; refines dcn-2610170005357588 and dcn-2610170028405029): at the push and at the arming alike, any fetch the fence makes that fails is never logged as the lane's refused, whatever git says, since the drain's own read of origin at every move already waits on an origin it cannot read and gives up on one that stays unread through the outage schedule; a failure that reads as a lost network (implement.IsNetworkFailure) also waits on the shared outage probe, as the push's own failure does, and any other failure does not, because the network probe proves only the network back (grounds: one bound for an unreadable origin, the drain's, rather than a second one at the lane; the push's netFailure as the existing shape; review of spc-2610161044461448 steps 5 and 7)
