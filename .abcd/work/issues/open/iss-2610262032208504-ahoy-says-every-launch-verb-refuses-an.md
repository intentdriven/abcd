---
schema_version: 1
id: "iss-2610262032208504"
slug: "ahoy-says-every-launch-verb-refuses-an"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 255 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/01-ahoy.md:826"
remedy: "In 01-ahoy.md, the sentence ending 'would leave every launch verb refusing the repository' (now near line 925, not 826): say only the launch preview and the scaffold refuse a repository with no declared kind, while the cut, the archive render and the receipts check read an absent declaration as the plugin shape."
finding_ref: cfk-bb02fca88570f898
---

Ahoy says every launch verb refuses an undeclared artefact kind; launch says the cut, the archive render and the receipts check assume the plugin shape

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/04-surfaces/01-ahoy.md:826` — "withholding the declaration would leave every launch verb refusing the repository"
- End B: `.abcd/development/brief/04-surfaces/04-launch.md:46` — "the cut, the archive render and the receipts check read an absent declaration as the plugin shape they have always assumed"

Ahoy justifies declaring application on an unattended install because withholding the declaration would leave every launch verb refusing the repository. Launch says only the preview and the scaffold refuse a repository that declared no kind, while the cut, the archive render and the receipts check read an absent declaration as the plugin shape. Ahoy's premise that every launch verb refuses is false by launch's own account of its verbs.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 255.
