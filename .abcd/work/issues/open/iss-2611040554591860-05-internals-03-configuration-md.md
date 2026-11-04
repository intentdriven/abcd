---
schema_version: 1
id: "iss-2611040554591860"
slug: "05-internals-03-configuration-md"
severity: "minor"
category: "drift"
source: "user-observation"
found_during: "the v0.16.1 release gates"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/05-internals/03-configuration.md"
remedy: "Add fill and bracket to the table and example, and widen recalled to notes and pages."
---

05-internals/03-configuration.md:1491-1513 gives the prompt router's --json envelope as text, session_title, injected, recalled, active and error. abcd hook prompt-router --json also returns fill and bracket, neither in the table nor the example. The table calls recalled the ids of memory notes (:1481 similarly), while the verb's help says it names notes and pages, a page as page:<filename>.
