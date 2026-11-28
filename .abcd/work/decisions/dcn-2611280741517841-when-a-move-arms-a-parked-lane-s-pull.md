---
id: dcn-2611280741517841
slug: when-a-move-arms-a-parked-lane-s-pull
date: 2026-11-28
---
2026-11-28 — When a move arms a parked lane's pull request again and a later step of the same move refuses (origin unread, for one), the drive's terminal does not say the 'armed again' line, because a refused move returns no result for sayMove. The run's record still holds the arming. Grounds: saying it would mean carrying a partial move's notes through every refusal path in drain.go for a line that only informs, and the next move reads the pull request as queued and acts correctly. Reverse by returning the parked notes from refuseMoved and saying them before the refusal.
