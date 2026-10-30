---
id: dcn-2610301309039861
slug: a-find-that-deletes-each-match-with-rm
date: 2026-10-30
---
2026-10-30 — A find that deletes each match with rm and no recursive flag (find . -name x -exec rm {} \;) is not a guard hazard: it removes only the files the find names, one at a time, the same everyday delete dcn-2609280000000003 allows as rm -rf $(find ...), and a blocker there would teach sessions to route around the guard (adr-42, a mistake filter). Decided by the technical facilitator on delegation while judging iss-2609300536290307.
