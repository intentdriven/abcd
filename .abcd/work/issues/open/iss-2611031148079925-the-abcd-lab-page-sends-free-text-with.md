---
schema_version: 1
id: "iss-2611031148079925"
slug: "the-abcd-lab-page-sends-free-text-with"
severity: "minor"
category: "ux"
source: "user-observation"
found_during: "coordinating the build lanes, 2026-11-03"
origin: researcher-authored
production_mode: hand-written
found_at: "commands/lab.md"
remedy: "Have the page route free text that names no sub-verb to lab mint, with the text as the question, and keep the bare list for an empty invocation."
---

The /abcd:lab page sends free text with no sub-verb to the bare list: on 2026-11-03 the product thinker typed /abcd:lab with a change to try on the status line, meaning a temporary experiment to evaluate, and the page's 'with none, list the labs' rule read it as no sub-verb, so the session treated it as a request to change the product and drafted an intent instead of minting a lab.
