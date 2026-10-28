---
schema_version: 1
id: "iss-2610121541515477"
slug: "abcd-s-setup-questions-in-a-terminal-are"
severity: "minor"
category: "ux"
source: "user-observation"
found_during: "the person ran abcd ahoy remote apply on Dessau, 2026-10-12"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/ahoy/setup_question.go"
remedy: "Render setup questions as a short plain-language question with a 'Now:' line stating the current state and a 'Change:' line stating what the yes does, in words, never API keys; give every option a distinct, non-circular meaning that fits the verb asking; drop lines whose value is 'not applicable'; select with a clear TUI cursor (arrow keys, a visible marker and highlight on the focused option, Enter to confirm, the chosen option echoed back); and end with a plain result ('Done: secret scanning and push protection are now on for <repo>'); relate to itd-2610030810370060 (interviews in a plain Terminal) and the GRILL asking rules in internal/core/question"
---

abcd's setup questions in a terminal are unclear to a product thinker and to a technical facilitator. Seen on 2026-10-12 with abcd ahoy remote apply on a managed repository: (1) a meaningless 'Setup Q1' heading; (2) the question names raw API keys ('secret_scanning: disabled -> enabled; secret_scanning_push_protection: disabled -> enabled') where plain words are needed; (3) the option meanings are circular ('Makes the change the question names') or wrong for the verb ('so the next install asks again' on a remote apply), and 'No, leave it' and 'Decide later' read the same; (4) 'Now: not applicable' and 'Change later: not applicable' lines are printed as noise; (5) the cursor highlight is not recognisable as the selection, so the person could not tell which option they had picked, and the result line 'abcd ahoy remote apply — clean' after a change does not say plainly that it succeeded.
