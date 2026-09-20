---
schema_version: 1
id: "iss-2608210934566220"
slug: "bundled-opinions-pointers-dangle-in-managed-repos"
severity: "minor"
category: "tech-debt"
source: "impl-review"
found_during: "memory-graduation principle work"
---

The six bundled OPINIONS rules each end with a pointer to a file under .abcd/development/principles/ that exists only in the abcd repo itself — a managed repo inherits the injected lines verbatim, so every prompt whose recall matches the domain hands its agent six dangling references (the principles corpus is not part of adoption). Either the bundled lines need self-contained phrasings with the pointer marked as abcd-repo-only, or adoption (prepare-this-repo / ahoy) should ship a distilled principles set the pointers can resolve against. Found while adding the memory-graduation rule, whose line was written self-contained for exactly this reason
**Corroboration (2026-09-17, Gropius managed-repo session gropiusllm-56, relayed
to abcd-17).** The dangling pointers were met in practice: the bundled OPINIONS
domain injected its six `.abcd/development/principles/<name>.md` pointers on a
matching prompt, a reviewer in that session was asked to check the principles
the rules cite, and found no such directory in the managed repo. The session
proposed the three remedies this record already weighs plus one more: go dormant
when the directory is absent, lay the principles down at prepare time, or
resolve the pointers to the plugin's bundled copies. Second independent hit;
the first was the filing.

**Third hit (2026-09-19, Gropius session gropiusllm-66, relayed to abcd-17).**
The managed repository answered the dangling pointers itself: it wrote its own
`.abcd/development/principles/` directory (its PR 88) and a test holding the
six pointers resolvable (its PR 97). The binary still ships pointers a fresh
repository cannot follow, so every other adopter starts where this one did.
