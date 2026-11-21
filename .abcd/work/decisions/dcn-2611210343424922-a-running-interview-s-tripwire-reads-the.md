---
id: dcn-2611210343424922
slug: a-running-interview-s-tripwire-reads-the
date: 2026-11-21
---
2026-11-21 — A running interview's tripwire reads the push receipts in its own checkout's local tier again, by mode, size and content hash, and where that receipts directory leads when it is a link, held to the followed-link budget (iss-2611012144422634). The pre-push gate lets a commit through on a receipt there, so a role that writes one during a turn would let a push skip the local preflight; that write is the role's to answer for and stops the interview. A sibling worktree's receipts stay unread: a peer's preflight mints one in its own worktree at any time, and that is the peer's own work. This narrows dcn-2611210325292707, which left receipts out in this checkout's tier as well as every sibling's.
