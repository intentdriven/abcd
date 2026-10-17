---
id: dcn-2607240000000003
slug: itd-93-design-settled-in-maintainer
date: 2026-07-24
---
2026-07-24 — itd-93 design settled in maintainer grill: (a) surface is a
`launch` sub-verb (extends 04-launch; rejected: new top-level verb, ahoy
install step, embark-time family); (b) SELF-SCAFFOLD PARITY — abcd-cli's own
release workflows are regenerated from the shipped template and a test
asserts parity, so the proven pattern and the template are one artifact
(rejected: lockstep diff test between two artifacts; frozen verbatim copy);
(c) built-in workflow_dispatch REHEARSAL mode — arms the full gate against a
simulated roll, publishes nothing; a green rehearsal is the runbook
precondition for the first real release; (d) the changelog dated-heading
format IS the itd-73 seam — `launch ship` is one optional producer.
Promotes standalone / severity minor, PRD synthesised from the interview
(no grandfathering); the 5 seeded ACs stand, amended for rehearsal mode.
