---
schema_version: 1
id: "iss-2611040555506702"
slug: "interview-redrafts-is-named-in-the-v0-16"
severity: "minor"
category: "documentation"
source: "user-observation"
found_during: "the v0.16.1 release gates"
origin: researcher-authored
production_mode: hand-written
found_at: "commands/intent.md"
remedy: "Document the key beside the planning interview's other settings."
resolution: "commands/intent.md documents interview.redrafts in its plain-Terminal section, beside the planning interview's runner route: the repository's .abcd/config.json, 0 to 10, 3 by default, what happens at the bound, and the refusal of the key in the machine's file. A page test holds the figures to the reader's constants."
impact: fix
resolved_by:
  commit: "490504756e13275082fc23f3f4bdb58afba8a02d"
---

interview.redrafts is named in the v0.16.1 changelog but documented on no page under docs/ or commands/; only agents/planning-interviewer.md mentions it.

## Grounds

- pursued: a person reading the intent page finds the key, its file, range and default; the page test TestIntentPageDocumentsTheRedraftsBound failing, or the page's sentences disagreeing with internal/core/layered/interview.go or internal/core/interview/redraft.go, would show it wrong
