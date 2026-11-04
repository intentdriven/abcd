---
schema_version: 1
id: "iss-2611040554295339"
slug: "the-lab-chapter-and-abcd-lab-harvest"
severity: "minor"
category: "documentation"
source: "user-observation"
found_during: "the v0.16.1 release gates"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/31-lab.md"
remedy: "Have the harvest print the line with a --remedy placeholder, or say in the chapter and help that the line needs --remedy added before it files anything."
refines: [iss-2609091956001547]
---

The lab chapter and `abcd lab harvest --help` say every product finding is listed with the capture line that files it (.abcd/development/brief/04-surfaces/31-lab.md:124). The line built at internal/core/lab/harvest.go:135 has no --remedy, and `abcd capture` refuses it with exit 2. harvest.md says the flag must be added before the command runs; the chapter, the help and the verb's terminal listing do not.
