---
schema_version: 1
id: "iss-2610170926295770"
slug: "a-reservation-made-by-hand-lapses-while"
severity: "minor"
category: "bug"
source: "agent-observation"
found_during: "PR #1068 preflight"
origin: researcher-authored
production_mode: hand-written
found_at: "abcd implement load --wait, and hand-made reservations (abcd reserve)"
remedy: "Have implement load --wait renew the caller's own live reservations from this copy once per half lease while it waits"
resolution: "implement load --wait now renews the caller's own live reservations from this copy once each is half through its lease, while it waits, beside the gate and never in it"
impact: fix
---

A reservation made by hand lapses while its holder waits in abcd implement load --wait: nothing renews it, so a preflight queued behind the gate for longer than the lease leaves the record unreserved while its work is still in progress. On 2026-10-17 a cloud-scripts fix waited about two hours for its preflight, and both of its reservations lapsed for about an hour before the holder renewed them by hand; no one else took them, but any peer could have.
