---
id: dcn-2608300000000017
slug: itd-179-round-5-a-relocation-is-not-a
date: 2026-08-30
---
2026-08-30 — itd-179 round-5: a RELOCATION is not a BACKFILL, and the forward-only rule forbids only the second. Moving a pre-tooling `## Grounds (pursued)` section from a spec onto its intent (commit 09f7d91a, thirteen specs) put text authored at the moment of pursuit where it belongs; nothing was reconstructed, which is the whole of what "never backfilled" refuses. Three shipped intents therefore carry a `## Grounds` section — itd-177, itd-182, itd-188 — and the surface claim now says so instead of asserting the corpus carries none (iss-2608301657357989). The enforcement is unchanged and stays maximally strict: `RecordGrounds` refuses a shipped or superseded record whatever the provenance of the text, because nothing in it can tell relocated text from invented text, so the relocated state is deliberately not reachable through `abcd intent ready --grounds`. Rejected: weakening the refusal to admit a `--relocate` escape, which would make the distinction a caller's assertion rather than a reviewable fact about a commit.
