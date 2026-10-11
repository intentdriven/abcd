---
schema_version: 1
id: "iss-2610111851312171"
slug: "the-readme-contents-part-lists-an-atx"
severity: "minor"
category: "bug"
source: "user-observation"
found_during: "review of README parts step 2, run-2610110908176735"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/ahoy/readme_derive.go"
remedy: "Track an open HTML block until the line that ends it, and read no heading inside one."
---

The README contents part lists an ATX heading that sits inside an HTML block no blank line has closed, though the forge renders that line as text: '<div align="center">' then '## Title' then '</div>' gives a contents entry linking an anchor the page does not have.
