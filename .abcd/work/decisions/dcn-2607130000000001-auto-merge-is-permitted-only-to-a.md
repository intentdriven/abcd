---
id: dcn-2607130000000001
slug: auto-merge-is-permitted-only-to-a
date: 2026-07-13
---
2026-07-13 — Auto-merge is permitted ONLY to a non-protected trunk, gated on a SHIP
review *verdict* (not merely green CI) + lint/smoke + an audit entry; never to `main`
(explicit human `abcd spec ship` promotes). A bounded, opt-in reversal of the standing
"a human merges" default — safe because the merge target is staging, not the protected
branch, and the gate is a verdict, not a checkmark (green CI shipped a real leak during
the 2026-07-12 drain; a security review then HELD it). Record homes: experience → itd-29
(already scoped, deferred v2); enforceable form → a brief invariant + an ADR *when built*,
not now (capture-now-build-later). SOTA is itd-29's (GitHub-native auto-merge, host-owns-
git, no new dep); the ADR inherits it. Surfaced the `facilitator-default-thinker-optional`
principle.
