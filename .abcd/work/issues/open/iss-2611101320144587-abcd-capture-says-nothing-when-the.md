---
schema_version: 1
id: "iss-2611101320144587"
slug: "abcd-capture-says-nothing-when-the"
severity: "minor"
category: "ux"
source: "agent-observation"
found_during: "autonomous run of 7 Nov 2026"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/capture/workflow.go"
remedy: "Have capture print one line when it writes a major or critical record, saying it will hold the next cut until it is fixed, wontfixed or deferred; a test captures a major issue and finds the line."
---

abcd capture says nothing when the severity it writes will hold the next release cut, though the cut refuses any major or critical record that entered the ledger since the anchor tag and is still open. On 7 Nov a filing agent rated two follow-ups critical and six major from a list of follow-ups, and the coordinator lowered five to minor and two to major before they landed.
