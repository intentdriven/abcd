---
schema_version: 1
id: "iss-2611040555347290"
slug: "abcd-development-brief-04-surfaces"
severity: "minor"
category: "bug"
source: "user-observation"
found_during: "the v0.16.1 release gates"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/26-mode.md"
remedy: "Treat a home that cannot be opened as the no-home refusal, exit 2, on all three verbs."
---

.abcd/development/brief/04-surfaces/26-mode.md:97-103 says every refusal of these verbs exits 2 with nothing written and that ending the mode refuses only with no checkout, no home or no root commit. With HOME set to a directory that does not exist, `abcd mode autonomous` and `abcd mode queue <step>` exit 1 with an open error, and `abcd mode back` exits 0, printing both that the record was closed without the store's lock and that there is nothing to end, though no record exists.
