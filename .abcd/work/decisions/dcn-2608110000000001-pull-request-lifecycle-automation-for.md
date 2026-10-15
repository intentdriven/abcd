---
id: dcn-2608110000000001
slug: pull-request-lifecycle-automation-for
date: 2026-08-11
---
2026-08-11 — Pull-request lifecycle automation for autonomous rounds settled
five points, recorded against the existing records rather than a new intent
(one-canonical-primitive: iss-172, iss-178 and itd-107 already hold this
ground; `capture promote` would have minted a sixth). (1) AUTO-MERGE
ELIGIBILITY splits on additive-versus-editing, not documentation-versus-code:
a new record file plus an append to this log, insertions only, record gate
green, single commit. The evidence is this repo's own correction history —
the post-merge correction commits in the recent window each fix a FALSE CLAIM
about the system ("describe the real fix", "state what is true", the
packaging and exclusion claims), which no deterministic gate catches, and all
of them sit in changes that REWRITE existing prose; the purely additive
captures needed none. (2) The eligible set is an ALLOWLIST of inert paths,
never a denylist of source extensions: the plugin command pages, the rules
file, the lint configuration and the agent-instruction router are markdown or
JSON that change behaviour, so a "no source touched" test fails open on all
of them. (3) Intent DRAFTS count as additive — the lifecycle already gates at
promotion (adoption is a maintainer decision; implementing an unready intent
is a STOP), so a second gate at merge is redundant. (4) The strict
status-check policy is NOT relaxed to smooth the stalled-behind-base
condition: it is the only thing that gates a concurrently-minted duplicate
record id, because the mint lock is a filesystem lock that does not span
checkouts and each branch passes in isolation; the stall is cleared by
updating the branch instead. (5) Delivery is rung 1 first — protocol in the
rendered routine, host credentials — per script-first-mvp, with the binary
acquiring no platform CLI dependency. Deferred to the itd-107 grill: the
rung-2 credential question, a declared merge style, peer-session concurrency
policy, and attribution-footer defence depth.
