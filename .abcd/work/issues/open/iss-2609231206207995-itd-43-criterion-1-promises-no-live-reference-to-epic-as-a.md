---
schema_version: 1
id: "iss-2609231206207995"
slug: "itd-43-criterion-1-promises-no-live-reference-to-epic-as-a"
severity: "minor"
category: "drift"
source: "user-observation"
found_during: "autonomous run 2026-09-23 fidelity audit"
origin: researcher-authored
production_mode: hand-written
---

itd-43 criterion 1 promises no live reference to epic as a standalone noun in abcd-owned files. One survives on the brief's intent surface page, .abcd/development/brief/04-surfaces/05-intent.md:244 ('no epic currently owns it'), inside the fenced surface diagram opened at line 176, which GL002 skips by design as fenced text and a grep finds. Reword it to spec, or record that fenced diagrams are out of the sweep's scope. Found by the fidelity audit; not fixed here.
