---
id: dcn-2611201429109439
slug: a-reservation-stops-renewing-once-its
date: 2026-11-20
---
2026-11-20 — A reservation stops renewing once its landing has made no merge progress for a set time, so a stalled landing no longer holds its footprint for days; the time is a setting with a default, and iss-2611201429249531 records the remedy. A pull request armed or in the merge queue, and a wait the run records as a forge outage or a stuck queue, count as progress, so the waits adr-2610222222326104 (amendment of 2026-11-05) and dcn-2611151650309222 keep renewing through stay as they are: the rule reaches a landing that is neither armed, queued nor waiting on a recorded outage, such as a pull request whose arming was withdrawn. Decided by the person, holding both roles, in an interview on 20 Nov 2026.
