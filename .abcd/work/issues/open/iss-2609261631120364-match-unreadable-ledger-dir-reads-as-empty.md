---
schema_version: 1
id: "iss-2609261631120364"
slug: "match-unreadable-ledger-dir-reads-as-empty"
severity: "minor"
category: "bug"
source: "review-followup"
found_during: "autonomous run A resumed 2026-09-25: review-match"
origin: researcher-authored
production_mode: hand-written
---

The filing-time match reads an unreadable open/ or resolved/ ledger directory as an empty candidate set: capture/match.go matchCandidates discards scanLedger's result for the directory, and scanLedger treats any ReadDir error as an absent ledger, so a capture or intent create reports 'compared N' or 'no record to compare' instead of the unread outcome the intent store's own failure takes. The same absent-or-unreadable conflation makes capture list and capture status count an unreadable status directory as zero records with nothing in the skipped roster.
