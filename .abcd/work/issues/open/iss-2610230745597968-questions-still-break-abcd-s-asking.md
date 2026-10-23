---
schema_version: 1
id: "iss-2610230745597968"
slug: "questions-still-break-abcd-s-asking"
severity: "minor"
category: "bug"
source: "user-observation"
found_during: "user report with two screenshots from two accounts, 2026-10-23"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/question"
remedy: "Add a read-only abcd verb that checks a draft question before it is asked and returns every fault, the row count of each tab and the ready-to-ask input, have the question drafter and agents call it instead of counting rows themselves, and make the hook's refusal list every fault grouped by rule without truncation and name that verb; the open choice of whether the status line alone makes a question abcd's goes to the product thinker."
related_intents: [itd-2610230756259215]
---

Questions still break abcd's asking limits after several rounds of fixes, on two accounts on 2026-10-23. (1) In a repository abcd manages, on another account, an agent bundled four separate issues into one question without going through the abcd:question-drafter agent, and the question check refused 21 parts at once: headers that were not role chips, no Decide later option, no Now: or Change later: lines, and question text that did not quote the thing being decided; the status line meanwhile read 'waiting on the product thinker'. (2) In abcd's own checkout, a question was refused because its text did not end with the question, and it also ran 25 rows against the 24-row limit (header and frame 8, question text 7, options 10), which the check reports but does not refuse, so it would have been shown and a narrow window might cut it; at the same time an abcd:question-drafter run spent 9 minutes 32 seconds and 174.2k tokens recounting over-limit tabs with a hand-written rows script. Earlier rounds include iss-2610071538055431, iss-2610100626327722, iss-2610161734179773 and the open iss-2610090816296964.

## Grounds

- pursued: we expect a command that checks a draft question and returns every fault and the exact row count to end the repeated refusals, because the drafter and agents would stop counting rows by hand and see every fault in one pass; it is shown wrong if a question drafted through the command is still refused for layout or rows, or if a drafter run still writes its own counting script
