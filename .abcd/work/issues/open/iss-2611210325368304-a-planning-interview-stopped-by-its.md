---
schema_version: 1
id: "iss-2611210325368304"
slug: "a-planning-interview-stopped-by-its"
severity: "minor"
category: "ux"
source: "user-observation"
found_during: "fix of iss-2611012144422634"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/surface/cli/written_interview.go"
remedy: "When a running interview's tripwire fires, the terminal front door tells the person in plain words what changed and offers to continue the interview from the next question or to stop, recording their choice."
---

A planning interview stopped by its tripwire ends the run with a refusal: it names what changed, but never offers the person to continue past a change they recognise as harmless, so they must start the interview again from its first question.
