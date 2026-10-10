---
schema_version: 1
id: "iss-2608271804189287"
slug: "abcd-namespace-documents-lag-the-tree-in"
severity: "minor"
category: "observation"
source: "agent-finding"
found_during: "structural consistency review of .abcd/ and docs/ (2026-08-27)"
found_at: ".abcd/README.md"
resolution: "namespace docs reconciled: .abcd/README root-file index and .work.local roster, AGENTS roster, config/ and corpus.json tree lines"
impact: internal
resolved_by:
  commit: "a032377d"
---

the .abcd namespace documents lag the tree in four places: .abcd/README.md describes only the three subdirectories and none of the nine tracked root files the binary reads; the config/ directory and its three members are documented nowhere; the brief's canonical layout tree lists corpus.json which does not exist and nothing reads (annotate as the itd-25 design target rather than delete); and both CLAUDE.md and .abcd/README.md misstate .work.local/'s contents (missing reviews/ — the intent-audit receipt outbox — and private-names.txt, the per-machine banlist layer). One documentation sweep, framed as an index pointing at the owning records, not duplicated schemas.