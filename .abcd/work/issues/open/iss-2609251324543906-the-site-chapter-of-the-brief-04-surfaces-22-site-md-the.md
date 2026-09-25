---
schema_version: 1
id: "iss-2609251324543906"
slug: "the-site-chapter-of-the-brief-04-surfaces-22-site-md-the"
severity: "minor"
category: "documentation"
source: "user-observation"
found_during: "autonomous run A resumed 2026-09-25"
origin: researcher-authored
production_mode: hand-written
---

The site chapter of the brief (04-surfaces/22-site.md, The gates) says the check's own help text names the same seven gates, kept beside the code that runs them; the check's help carries only its sentence and its --out flag and names no gate, so the chapter claims a surface that does not exist. The seven names do live in one place in the code (site.CheckNames), and the check's report prints each one as it runs it.
