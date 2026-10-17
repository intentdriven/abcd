---
id: dcn-2607290000000001
slug: privacy-leak-follow-up-illustrative
date: 2026-07-29
---
2026-07-29 — privacy-leak follow-up: illustrative machine identifiers in anything committed or published always come from reserved documentation ranges (RFC 5737 IPv4, RFC 3849 IPv6, RFC 2606 domains, RFC 7042 MACs; hostnames derived from the persona registry, e.g. alice-laptop) — the itd-79 persona rule applied to infrastructure. Recorded as principle `examples-use-reserved-identifiers`; enforcement is the iss-154 allowlist-inversion lint (flag any identifier OUTSIDE the reserved ranges), whose shipping promotes the principle to a discipline. No new intent: real private identifiers are itd-74's banlist territory — the incident evidence extends itd-74's scope to machine identifiers (iss-158) rather than founding a second primitive. Incident issue text deliberately names no repo and no values (shape only), so the ledger itself cannot re-leak.
