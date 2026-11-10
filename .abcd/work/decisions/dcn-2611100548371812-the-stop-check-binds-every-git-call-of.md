---
id: dcn-2611100548371812
slug: the-stop-check-binds-every-git-call-of
date: 2026-11-10
---
2026-11-10 — The stop check binds every git call of its own read to the hook's deadline except the reservations' view (claim.View and peers.RecordsAt), which the checklist pages name as the one read not yet bound. Binding the view means about 37 call sites shared with the reservation verbs, too large for a fix round of spc-2610230805429429; a later lane binds it (iss-2611100548341325). To reverse: bind the view and drop the named exception.
