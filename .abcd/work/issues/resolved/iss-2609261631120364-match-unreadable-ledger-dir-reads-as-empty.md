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
resolution: "an unreadable status directory never reads as an empty one: scanStatusDir lists through readStatusDir, the one place an absent directory and an unreadable one part company (iss-2609261241121312), so the filing-time match reports an unreadable open/ or resolved/ as an unread record set, and capture list and capture status fault naming the directory."
impact: fix
resolved_by:
  commit: "b24ec86f"
---

The filing-time match reads an unreadable open/ or resolved/ ledger directory as an empty candidate set: capture/match.go matchCandidates discards scanLedger's result for the directory, and scanLedger treats any ReadDir error as an absent ledger, so a capture or intent create reports 'compared N' or 'no record to compare' instead of the unread outcome the intent store's own failure takes. The same absent-or-unreadable conflation makes capture list and capture status count an unreadable status directory as zero records with nothing in the skipped roster.

## Grounds

- pursued: a capture with resolved/ at mode 000 reports the match as unread and links nothing, and list and status carry one read-layer skip naming the directory; TestMatchReportsAnUnreadableStatusDirectoryAsUnread, TestListAndStatusReportAnUnreadableStatusDirectory and TestAScopedListNamesAnUnreadableOpenDirectory would fail if the directory read as empty again
