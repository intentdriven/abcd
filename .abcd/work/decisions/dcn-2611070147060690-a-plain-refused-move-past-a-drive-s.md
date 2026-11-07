---
id: dcn-2611070147060690
slug: a-plain-refused-move-past-a-drive-s
date: 2026-11-07
---
2026-11-07 — A plain refused move past a drive's first still ends the run on any binary, as it did before self-update, and this change leaves that alone. Waiting on a refused move is deferred to the follow-up bugfix release, because what a long unattended run waits on after a refusal needs its own review, and changing it now would change what reaches the way back while the release is being cut. The follow-up: wait on a refused move under a bound, with a say-so line.
