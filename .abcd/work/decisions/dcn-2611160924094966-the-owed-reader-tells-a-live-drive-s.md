---
id: dcn-2611160924094966
slug: the-owed-reader-tells-a-live-drive-s
date: 2026-11-16
---
2026-11-16 — The owed reader tells a live drive's lane by the driver its run's control record names, not by probing the drive lock (iss-2611150851542274). A drive started in a plain terminal records its own process, with its start time and no session (the loop's selfDriver). While that process is alive by id and start time in the process table, and is neither the stopping check's own process nor one the walk up from it reaches, the run's lanes and the reservation only its runs are for are left out of the owed work, with a note naming the drive's process. Grounds: the record already says which process drives which run, so no lock is taken or probed, nothing is added beside the drive lock primitive, and drain_drive.go is untouched; a lock would only say some drive holds the checkout, not which runs are its. Whatever cannot be told stays owed: an ended drive, a reused pid, a walk cut short, an unreadable table. Reversal: remove driveScope from scope() in internal/core/autonomy/holders.go, and a terminal's lanes are owed again as before.
