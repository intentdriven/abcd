---
id: dcn-2611100347208820
slug: the-session-start-hook-gets-one-deadline
date: 2026-11-10
---
2026-11-10 — The session-start hook sets a deadline of a minute at its start for its checklist part, and these reads of that part run under what is left of it: the checkout root's lookup, the read's tracked check and the scanner's identity probe; the hook's other reads are not yet bound (iss-2611100548341325): the host gives the hook's shell 240 seconds with a first run's binary download inside them, the stop hook's 14 seconds is tuned to a 30-second host timeout and would cut a slow start's drain short, and a minute keeps the read well inside the host's limit while a start that spends it leaves the list out and says so on stderr (spc-2610230805429429).
