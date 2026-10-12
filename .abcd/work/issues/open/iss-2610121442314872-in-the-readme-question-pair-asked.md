---
schema_version: 1
id: "iss-2610121442314872"
slug: "in-the-readme-question-pair-asked"
severity: "minor"
category: "ux"
source: "user-observation"
found_during: "product thinker interview on missing decisions, 2026-10-12"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/ahoy/readme_ask.go"
remedy: "Order each README part's options with its usual answer first (abcd-badge and contents: off, on, Decide later; the others: on, off, Decide later) in readme_tool and the drawn and stdin doors alike, and change TestReadmeOfferView and any order test to pin it."
---

In the README question pair asked through the host's question tool, every part's tab lists on, off, Decide later in one fixed order, so the cursor rests on on and a quick Enter turns on the two parts that are off unless chosen: the managed-with-abcd badge and the table of contents. The product thinker ruled on 2026-10-12 that those two tabs list off first, so Enter takes each part's usual answer; the order may differ between tabs. The drawn and stdin terminal doors already start on the default.
