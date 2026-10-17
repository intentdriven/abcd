# Release 0.15.1 (2026-10-17)

A person runs three sessions on one machine, and each one names itself: Alice, Bob, Carol. From the first prompt on, the name appears in the session's status line and prompt border, and in the session list other sessions use to message it, so the person can tell the sessions apart at a glance and say "ask Bob" rather than "ask abcd-a5". A session the person has already named keeps that name, a resumed or compacted session keeps the name it had, and no two live sessions under one account on the machine share a name, even when they work in different repositories, so "Bob" always means one session. Naming works on macOS and Linux, through abcd's own session-start and prompt hooks, with nothing for the person to do. (itd-2610111707085098)

Several people, each on their own GitHub account and machine, run abcd drain on one repository at the same time, and the drains share the work rather than repeat it. Each drain plans from what origin's default branch holds now, never from a checkout that has fallen behind, so it never reserves an issue another drain already fixed. It skips an issue another account holds before spending a judgement on it, and each account starts at a different point in the queue. An issue one drain hands back stays handed back for every drain, recorded on origin, and a drain stops as empty only when nothing is left for anyone. (itd-2610161016394967)

Also in this release:

- A stuck drain lane hands itself back (itd-2610161016391855)
- Drains run unattended and visibly (itd-2610161016401115)

The line-by-line record of this release is its section in CHANGELOG.md.
