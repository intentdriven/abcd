---
id: dcn-2607270000000012
slug: itd-103-guard-wiring-the-two-front-doors
date: 2026-07-27
---
2026-07-27: itd-103 guard wiring — the two front doors disagree on ONE case by design. A guard that cannot be evaluated (unparsable command, registry that will not load) is a FAULT for `guard check` (exit 2: a script must never read silence as clearance) and fail-OPEN for `guard hook` (exit 0 + loud stderr: a broken guard must never stop a session). Rejected: one shared behaviour, which would either brick sessions or teach scripts that silence means safe.
