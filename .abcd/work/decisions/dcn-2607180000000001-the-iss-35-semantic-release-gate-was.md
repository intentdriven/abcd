---
id: dcn-2607180000000001
slug: the-iss-35-semantic-release-gate-was
date: 2026-07-18
---
2026-07-18 — The iss-35 semantic release gate was self-referential
(armed against the tagged commit, read receipts from that commit's own tree)
and fail-closed the first public release; fixed in PR #99 by arming with the
reviewed content commit (HEAD^2^ / HEAD^) from a full-history checkout, plus a
check-reviews.sh RD001 exemption for sha-keyed receipt dirs. The gate is
abcd-cli's OWN CI: it is NOT shipped or scaffolded to managed repos
(launch-payload.json excludes .github/; ahoy/launch write no CI; lifeboat only
reads .github/workflows as a grounding signal), so the flaw had no managed-repo
reach — the iss-108 capture's "systemic" framing was corrected on resolve. Any
future release-scaffolding intent should scaffold the fixed two-commit
(roll -> receipts) pattern, not the original self-referential one.
