---
id: dcn-2611280725158656
slug: a-drive-s-state-reader-says-nothing-when
date: 2026-11-28
---
2026-11-28 — A drive's state reader says nothing when the state file is absent, and names why on stderr only when a file is there but cannot be read, is of a newer version, or carries control codes. Grounds: a checkout that never ran a drive is the ordinary case, and a line on every bare board there is noise that also breaks every test reading the board's two streams together; an absent file hides nothing, since no drive wrote one. Reverse by printing the absent case too, in board_drive.go's reason switch.
