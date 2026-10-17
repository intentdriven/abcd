---
schema_version: 1
id: "iss-2610171626336875"
slug: "the-v0-15-1-release-review-found-the"
severity: "minor"
category: "documentation"
source: "review-followup"
found_during: "v0.15.1 release review"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/16-lint.md"
remedy: "Rewrite the passage at .abcd/development/brief/04-surfaces/16-lint.md:37-42 to match the shipped behaviour the review names, and check its sibling chapters for the same claim."
resolution: "The page now states what the shipped code does, re-verified against the code and the verb's --help at v0.15.1 before the edit."
impact: internal
resolved_by:
  commit: "a394eb10a"
---

The v0.15.1 release review found the brief at .abcd/development/brief/04-surfaces/16-lint.md:37-42 out of step with the shipped code. It says: The verbs that write stay under the docs, site and identity verbs (the citation baseline, the site render, the recording of the identity block), and 'No check runs under those verbs any more'. In the code: `abcd docs fidelity` is a shipped check under the docs verb ('Judge the brief against every shipped surface and the saved docs review'; 10-docs.md lists it as bucket gate), and `abcd identity render` prints every drifted surface's correction and writes nothing. The list of what stays under those verbs also leaves out …
