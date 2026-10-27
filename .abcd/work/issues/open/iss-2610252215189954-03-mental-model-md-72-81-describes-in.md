---
schema_version: 1
id: "iss-2610252215189954"
slug: "03-mental-model-md-72-81-describes-in"
severity: "minor"
category: "drift"
source: "agent-finding"
found_during: "brief single-source study, 2026-10-25, reader S12"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/01-product/03-mental-model.md"
remedy: "Under the truth rule, mark these passages staged, or cut them to one-line pointers at itd-44 and itd-62, matching 04-naming.md:184, which already marks the verdict (staged)."
---

03-mental-model.md:72-81 describes in the present tense an LLM suggested_kind classifier, an intent classify-capture-kind verdict (itd-44, a draft) and a discipline_kind field with a gate registry under .abcd/disciplines/. The create path writes suggested_kind: null (internal/core/intent/create.go:471), no code reads discipline_kind, and .abcd/disciplines/ does not exist.
