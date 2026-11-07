---
schema_version: 1
id: "iss-2611070737096049"
slug: "the-refusal-text-for-a-script-written"
severity: "minor"
category: "ux"
source: "user-observation"
found_during: "follow-up list from a release cut"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/guard/script.go"
remedy: "Add one clause to the refusal naming the accepted form: run the steps as separate commands."
---

The refusal text for a script written and then run in one line does not say that running the same steps on separate lines is fine, so a person or agent is left guessing how to proceed.
