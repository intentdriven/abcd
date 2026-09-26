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
resolution: "The site chapter now says the seven gate names live once in the code (site.CheckNames) and the check's report prints each as it runs it, instead of claiming the help text names them."
impact: internal
resolved_by:
  commit: "254f366406f8c5f04b88632bff26c78ccc0edb2b"
---

The site chapter of the brief (04-surfaces/22-site.md, The gates) says the check's own help text names the same seven gates, kept beside the code that runs them; the check's help carries only its sentence and its --out flag and names no gate, so the chapter claims a surface that does not exist. The seven names do live in one place in the code (site.CheckNames), and the check's report prints each one as it runs it.

## Grounds

- pursued: we expect the chapter's claim about where the gate names are stated to match the binary; shown wrong if the lint site help or report is found not to name the gates as the chapter now says
