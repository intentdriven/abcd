---
schema_version: 1
id: "iss-2611251739436752"
slug: "rules-prompt-prints-no-rules"
severity: "minor"
category: "ux"
source: "user-observation"
found_during: "manual build of 25 November, logged as a test of what abcd relies on"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/rules"
remedy: "Give `abcd rules` a form that prints the rules a task recalls, whole, for a reader with no hook, and make the preview header say the form each row is really in. Grounds: the sub-agent of 2026-11-25 that followed the instruction literally received no rule."
---

`abcd rules --prompt` does not print the rules a prompt recalls, and its header can contradict its rows: it says "Rules in full" and lists the one matched domain as its heading and first rule. A sub-agent told to run it on its task got domain names and had to render each domain by a second command. No one command prints the rules a task recalls.
