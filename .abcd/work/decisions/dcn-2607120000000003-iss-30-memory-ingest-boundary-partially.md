---
id: dcn-2607120000000003
slug: iss-30-memory-ingest-boundary-partially
date: 2026-07-12
---
2026-07-12 — iss-30 (memory ingest boundary) partially resolved: the fetch/read
subset — C12 (HTTP status), P11 (SSRF NAT64/6to4), C13 (local size cap), the
~user tilde mangle — landed in PR #38. iss-30 stays OPEN for its remaining
instances (the larger "ingest test-suite" effort): the --keep-original
partial-failure reporting, CRLF parser-parity (parseFrontmatter vs
splitFileFrontmatter), and broader URL-ingest/content-type/PDF path coverage.
