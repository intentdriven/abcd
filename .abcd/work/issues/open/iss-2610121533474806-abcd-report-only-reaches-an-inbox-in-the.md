---
schema_version: 1
id: "iss-2610121533474806"
slug: "abcd-report-only-reaches-an-inbox-in-the"
severity: "minor"
category: "future-work-seed"
source: "user-observation"
found_during: "a peer agent outside this account had issues to report, 2026-10-12"
origin: researcher-authored
production_mode: hand-written
found_at: "commands/report.md"
remedy: "Add a cross-account delivery route to abcd report and an intake that processes what arrives: (1) delivery: when the sender cannot write the receiving inbox, file the report as a GitHub issue on abcd's repository from the same template, redacted the same way, the sender named only as 'a managed repository' unless they opt in, security kinds kept off the public tracker (private vulnerability reporting); (2) intake: abcd inbox lists those issues beside local reports, and promote files each as a ledger record and closes or labels the GitHub issue with the record id, so nothing is processed twice; (3) drain: the promoted records enter the ordinary abcd drain by their remedy, and an unpromoted forge report is counted on the board so it cannot sit unread; relate to the security-advisory drain (iss-2610090703514921) and choose the route after a state-of-the-art check of cross-account bug intake"
---

abcd report only reaches an inbox in the same user account (~/.abcd.noindex/inbox), so a managed repository worked from another account, another machine or a cloud session cannot report a defect or proposal to abcd at all; on 2026-10-12 an agent building a game engine with abcd had two issues to report and no route to deliver them. Reports need a delivery path that works across accounts, for example a GitHub issue on abcd's repository.
