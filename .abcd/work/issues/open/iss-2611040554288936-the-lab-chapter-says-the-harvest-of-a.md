---
schema_version: 1
id: "iss-2611040554288936"
slug: "the-lab-chapter-says-the-harvest-of-a"
severity: "minor"
category: "documentation"
source: "user-observation"
found_during: "the v0.16.1 release gates"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/31-lab.md"
remedy: "Render a halted lab's gate findings ahead of section 3 in the harvest, or say in the chapter and help that they lead the open section only."
---

The lab chapter says the harvest of a halted lab leads with its gate finding (.abcd/development/brief/04-surfaces/31-lab.md:165), and `abcd lab harvest --help` says its gate findings come first (internal/surface/cli/lab.go:216). Gate findings sort first only within a section (internal/core/lab/harvest.go:285-286). A gate finding is open, so it sits in section 4, and any worked finding in section 3 precedes it; only a one-line halted note appears earlier, in section 2.
