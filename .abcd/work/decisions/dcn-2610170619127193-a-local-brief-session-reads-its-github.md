---
id: dcn-2610170619127193
slug: a-local-brief-session-reads-its-github
date: 2026-10-17
---
2026-10-17 — A local brief session reads its GitHub App's public client id from `.abcd/relay/github-app.json` (`{"client_id": "…"}`), which repository setup writes beside `relays.json`; the file holds nothing secret, and `abcd session start` refuses to start without a client id rather than offering no sign-in (facilitator, on delegation, on 2026-10-16; adr-2610161400542916, itd-2610091016262335).
