---
id: dcn-2610311539024848
slug: the-product-thinker-ruled-on-2026-10-31
date: 2026-10-31
---
2026-10-31 — The product thinker ruled on 2026-10-31, answering the question dcn-2610311251002713 sent back on iss-2610311224444576, that work on an issue or an intent is reserved only by a build or a drain: a fix that would be done by hand starts through abcd build instead, which holds an issue's found_at (an intent's, its spec's footprint) and, once iss-2610311224444576 builds it, widens an issue lane's hold file by file before each further edit, backing off a live hold another copy has there. The hand reservation of an issue or an intent is removed, but only after that widening ships, so a fix that edits beyond its found_at is never left with no way to hold the file (iss-2610311539091723); a release's reservation, and releasing a hold, stay. Accepted costs: an issue needs a found_at and an intent a READY spec before any work starts, a small fix runs the whole loop, and someone working outside abcd has no way left to mark work as taken. Rejected: keeping hand reservations, since one wide hold stalled nine of thirteen chosen builds on 2026-10-31.
