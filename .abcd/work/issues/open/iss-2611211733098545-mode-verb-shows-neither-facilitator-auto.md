---
schema_version: 1
id: "iss-2611211733098545"
slug: "mode-verb-shows-neither-facilitator-auto"
severity: "minor"
category: "ux"
source: "agent-finding"
found_during: "lab-261121170045-71e07e7"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/surface/cli/mode.go"
remedy: "Leave the first line of the plain form as the one word it is today, and add two things as further lines after it and as added JSON fields, so a reader that takes the first line or an existing field sees no change: the facilitator setting, read as the status line reads it, from the default branch as last fetched and with no network request, which the verb's help promises; and the away record's state (holding until when, or ended with something unread). Grounds: the probes of lab-261121170045-71e07e7; enforcement-claims-are-facts."
---

The mode verb shows neither the facilitator setting nor the away record. With the technical facilitator on auto in the committed configuration, and with the away record holding, the verb prints managed in both its plain and its JSON form, so a session or a person asking what mode holds learns nothing of either. Measured in lab-261121170045-71e07e7 at commit 71e07e7ea, probes mode-under-auto, mode-under-away-2 and mode-text-under-away.
