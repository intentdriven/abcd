---
schema_version: 1
id: "iss-2610160608369772"
slug: "the-brief-at-abcd-development-brief"
severity: "minor"
category: "documentation"
source: "review-followup"
found_during: "v0.15.0 release review"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/06-capture.md"
remedy: "Name the private-report intake as the third automatic filer in the exception at 06-capture.md:73-77."
resolution: "The capture chapter named two automatic filers that write the machine remedy; it now also names the embargo's intake, which files each privately reported security advisory into its private store with that same value."
impact: internal
---

The brief at .abcd/development/brief/04-surfaces/06-capture.md:73-77 says the one exception to the required person-written remedy is abcd's own automatic filers, the consistency pass and every promoted inbox report, which write 'none (filed automatically)'. A third automatic filer writes the same value (issueschema.MachineRemedy): the intake that files a privately sent report into abcd's private issue store.
