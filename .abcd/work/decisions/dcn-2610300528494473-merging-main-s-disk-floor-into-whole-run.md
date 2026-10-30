---
id: dcn-2610300528494473
slug: merging-main-s-disk-floor-into-whole-run
date: 2026-10-30
---
2026-10-30 — Merging main's disk floor into whole-run admission (itd-2610262127138979): the load gate reads the disk where main reads it (before the wait, which main's one-slot race shortcut followed, and at the open look after the jitter) and also before a one-slot race shortcut a later look reaches, not at every look, and judges each read against the floor sized by the slot count read at that same look. A race shortcut reached at a later look, after the count fell to one, refuses with the at-open wording, since the waiter has waited by then.
