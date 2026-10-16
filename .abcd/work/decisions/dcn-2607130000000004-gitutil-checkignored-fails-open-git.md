---
id: dcn-2607130000000004
slug: gitutil-checkignored-fails-open-git
date: 2026-07-13
---
2026-07-13 (itd-85, carry to M3): `gitutil.CheckIgnored` fails OPEN — git
absent or not-a-repo returns "nothing ignored". The `three-tier-layout` rule
MUST treat an empty result as "cannot tell", never as "compliant", or a repo
with git unavailable silently passes the "is `.abcd/.work.local/` gitignored"
assertion. Security review flagged this as the one consumer-side spec note.
