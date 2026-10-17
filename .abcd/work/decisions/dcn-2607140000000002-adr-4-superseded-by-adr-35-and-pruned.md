---
id: dcn-2607140000000002
slug: adr-4-superseded-by-adr-35-and-pruned
date: 2026-07-14
---
2026-07-14 — adr-4 SUPERSEDED by adr-35 and pruned per the ADR convention
(superseded ADRs are pruned; git preserves the text; the successor carries the
transition rationale). What survives is restated in adr-35: the lifeboat is
regenerable output, and the `lifeboat`(noun)/`voyage`(verb) distinction is
load-bearing. What changes: disembark is READ-ONLY and OUT-OF-TREE (a test hashes
the source tree before and after), and `voyage/` moves to the OPERATOR level
(`~/.abcd/voyage/<source-root-sha>/`, keyed like the history store). The voyage
move is not cosmetic — voyage records absolute source paths, and the
`privacy-hygiene` audit rule (itd-85) flags those in committed files, so abcd
would have failed its OWN audit. adr-4's overwrite-with-`.bak` model is replaced
by a destination safety gate (never overwrite a directory abcd did not produce);
its `shared_with` field is dropped (nothing produces it, and an empty field is a
lie in a schema); and its hash chain — asserted but never defined — is pinned.
Nine inbound references repointed by hand (2 links, 7 prose/frontmatter).
