---
schema_version: 1
id: "iss-2610241550154898"
slug: "commands-ahoy-md-lists-the-kinds-of-id"
severity: "nitpick"
category: "documentation"
source: "agent-finding"
found_during: "hand-back triage of 2026-10-24, batch b11 reading iss-221"
origin: researcher-authored
production_mode: hand-written
found_at: "commands/ahoy.md"
remedy: "Name history.lineage and its yes, no and later values in commands/ahoy.md's answers-file paragraph and among the questions the run itself decides to ask, proven by a test that every id setupConfirmIDs in internal/core/ahoy/setup_question.go declares and a file may answer appears on that page."
resolution: "commands/ahoy.md now names history.lineage, its yes, no and later values, and its place among the questions checked only when asked, with adopt and an offer's gap id named too. TestAhoyPageNamesEveryApprovalAFileMayAnswer and TestAhoyPageSaysHowAFileAnswersTheLineageQuestion fail before the change and pass after, with the whole ahoy package."
impact: fix
resolved_by:
  commit: "c7afad57f27a37154b750688298d6cdd3808e423"
---

commands/ahoy.md lists the kinds of id an answers file may answer (a value question's key, adopt, approve.<category>, an offer's gap id) but not history.lineage, the id the re-founding lineage question is answered by, and does not name the lineage question among the questions checked only when they are asked.
