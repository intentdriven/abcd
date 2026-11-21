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
found_at: "internal/core/mode/mode.go"
remedy: "Print the facilitator setting and the away record's state (holding until when, or ended with steps unread) beside the badge, in the plain form and as added JSON fields, so nothing a reader parses today changes. Grounds: the probes of lab-261121170045-71e07e7; enforcement-claims-are-facts."
---

The mode verb shows neither the facilitator setting nor the away record. With the technical facilitator on auto in the committed configuration, and with the away record holding, the verb prints managed in both its plain and its JSON form, so a session or a person asking what mode holds learns nothing of either. Measured in lab-261121170045-71e07e7 at commit 71e07e7ea, probes mode-under-auto, mode-under-away-2 and mode-text-under-away.
