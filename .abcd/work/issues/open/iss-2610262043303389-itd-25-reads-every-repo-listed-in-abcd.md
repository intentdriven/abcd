---
schema_version: 1
id: "iss-2610262043303389"
slug: "itd-25-reads-every-repo-listed-in-abcd"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 1142 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/drafts/itd-25-dredge-cross-corpus-synthesist.md:18"
remedy: "none (filed automatically)"
related_intents: [itd-25, itd-40]
finding_ref: cfk-d39451cfc56c4c0e
---

itd-25 reads every repo listed in .abcd/corpus.json; itd-40 says abcd does not group repos and keeps one registry

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/drafts/itd-25-dredge-cross-corpus-synthesist.md:18` — "clusters open issues across every repo in `.abcd/corpus.json`"
- End B: `.abcd/development/intents/shipped/itd-40-folder-classification.md:41` — "abcd is one repository (adr-28); it does not group repos"

itd-25's synthesist clusters open issues across every repo in .abcd/corpus.json, a list of repositories grouped for synthesis. itd-40 rules that abcd is one repository, does not group repos, and that the history index.json is the single registry of managed repos. The draft's corpus file is the grouping layer and second registry the shipped intent rules out.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 1142.
